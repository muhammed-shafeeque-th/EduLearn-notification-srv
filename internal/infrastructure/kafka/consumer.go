package kafka

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"go.uber.org/zap"
)

// ConsumerConfig defines the configuration for the consumer, including both
// concurrency (workers) and retry/queue model.
type ConsumerConfig struct {
	Workers    int
	Retries    int
	RetryDelay time.Duration
	BufferSize int
}

// DefaultConsumerConfig provides sane default config values.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Workers:    8, // increased for better parallelism by default
		Retries:    3,
		RetryDelay: 2 * time.Second,
		BufferSize: 256,
	}
}

// Consumer abstracts kafka consumption and concurrency controls.
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	logger        *zap.Logger
	config        ConsumerConfig

	// shutdown/synchronization
	closeOnce sync.Once
	closeCh   chan struct{}
	workersWg sync.WaitGroup

	errorsCh chan error

	// unified handler registry
	mu            sync.RWMutex
	topicHandlers map[string][]ports.MessageHandler // topic -> []handler
}

// NewConsumer creates and initializes a new Consumer instance.
func NewConsumer(
	brokers []string,
	groupID string,
	logger *zap.Logger,
	config ConsumerConfig,
) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Return.Errors = true
	saramaConfig.Version = sarama.V2_1_0_0

	cg, err := sarama.NewConsumerGroup(brokers, groupID, saramaConfig)
	if err != nil {
		logger.Error("Failed to create Kafka consumer group", zap.Error(err))
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return &Consumer{
		consumerGroup: cg,
		logger:        logger,
		config:        config,
		closeCh:       make(chan struct{}),
		errorsCh:      make(chan error, 4),
		topicHandlers: make(map[string][]ports.MessageHandler),
	}, nil
}

// RegisterHandlers registers a slice of handlers for a given topic.
// It is thread-safe and can be called multiple times before starting consumption.
func (c *Consumer) RegisterHandlers(topic string, handlers ...ports.MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if topic == "" {
		return errors.New("topic name cannot be empty")
	}
	if len(handlers) == 0 {
		return fmt.Errorf("at least one handler required for topic %q", topic)
	}
	for i, h := range handlers {
		if h == nil {
			return fmt.Errorf("handler at index %d for topic %q is nil", i, topic)
		}
	}
	// Append to any existing handlers for this topic
	c.topicHandlers[topic] = append(c.topicHandlers[topic], handlers...)
	return nil
}

// MustRegisterHandlers is a convenience variant that panics on error.
func (c *Consumer) MustRegisterHandlers(topic string, handlers ...ports.MessageHandler) {
	if err := c.RegisterHandlers(topic, handlers...); err != nil {
		panic(err)
	}
}

// StartConsuming subscribes to all topics that have registered handlers, with their respective handler slices.
func (c *Consumer) StartConsuming(ctx context.Context) error {
	c.mu.RLock()
	if len(c.topicHandlers) == 0 {
		c.mu.RUnlock()
		return errors.New("no handlers registered for consumption")
	}
	// Defensive copy
	topicHandlers := make(map[string][]ports.MessageHandler, len(c.topicHandlers))
	var topics []string
	for topic, handlers := range c.topicHandlers {
		if topic == "" || len(handlers) == 0 {
			c.mu.RUnlock()
			return fmt.Errorf("invalid topic or empty handlers for topic: %q", topic)
		}
		topics = append(topics, topic)
		dst := make([]ports.MessageHandler, len(handlers))
		copy(dst, handlers)
		topicHandlers[topic] = dst
	}
	c.mu.RUnlock()

	topicList := sanitizeTopics(topics)
	if len(topicList) == 0 {
		return fmt.Errorf("no valid topics found to subscribe")
	}

	adapter := newHandlerAdapter(topicHandlers, c.logger, c.config, ctx, c.errorsCh)
	return c.startSubscription(ctx, topicList, adapter)
}

// startSubscription spins up the main consumer/worker loop and concurrency handlers.
func (c *Consumer) startSubscription(ctx context.Context, topics []string, adapter *handlerAdapter) error {
	// Worker pool
	for i := 0; i < c.config.Workers; i++ {
		c.workersWg.Add(1)
		go func(workerID int) {
			defer c.workersWg.Done()
			adapter.worker()
		}(i)
	}

	consumeCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer adapter.closeJobs()
		for {
			select {
			case <-c.closeCh:
				c.logger.Info("Consumer shutdown triggered, exiting consume loop")
				cancel()
				return
			case <-consumeCtx.Done():
				c.logger.Info("Context cancelled in consume loop, exiting")
				return
			default:
			}
			err := c.consumerGroup.Consume(consumeCtx, topics, adapter)
			if err != nil && err != context.Canceled {
				c.logger.Error("Consumer group error", zap.Error(err))
				select {
				case c.errorsCh <- err:
				default:
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// Error logger goroutine
	go func() {
		for {
			select {
			case err := <-c.consumerGroup.Errors():
				c.logger.Error("Kafka consumer group error", zap.Error(err))
			case <-c.closeCh:
				return
			}
		}
	}()

	// App-level error channel reporting
	go func() {
		for {
			select {
			case err := <-c.errorsCh:
				c.logger.Error("Kafka consumer (application) error", zap.Error(err))
			case <-c.closeCh:
				return
			}
		}
	}()

	c.logger.Info("Kafka subscription started", zap.Strings("topics", topics))
	return nil
}

// Close gracefully shuts down the consumer group and worker pool.
func (c *Consumer) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closeCh)
		err = c.consumerGroup.Close()
		c.workersWg.Wait()
		close(c.errorsCh)
	})
	if err != nil {
		c.logger.Error("Failed to close consumer group", zap.Error(err))
		return err
	}
	c.logger.Info("Kafka consumer closed")
	return nil
}

// ---------------------------------------------------------------------
// Handler Adapter: Efficient worker pool, robust concurrency patterns
// ---------------------------------------------------------------------

type handlerAdapter struct {
	topicHandlers map[string][]ports.MessageHandler // topic -> handler slice
	logger        *zap.Logger
	config        ConsumerConfig
	jobs          chan *sarama.ConsumerMessage
	ctx           context.Context
	stopOnce      sync.Once
	stopped       chan struct{}
	errorsCh      chan error
}

func newHandlerAdapter(
	topicHandlers map[string][]ports.MessageHandler,
	logger *zap.Logger,
	config ConsumerConfig,
	ctx context.Context,
	errorsCh chan error,
) *handlerAdapter {
	var handlersCopy map[string][]ports.MessageHandler
	if len(topicHandlers) > 0 {
		handlersCopy = cloneTopicHandlers(topicHandlers)
	}
	return &handlerAdapter{
		topicHandlers: handlersCopy,
		logger:        logger,
		config:        config,
		jobs:          make(chan *sarama.ConsumerMessage, config.BufferSize),
		ctx:           ctx,
		stopped:       make(chan struct{}),
		errorsCh:      errorsCh,
	}
}

func (h *handlerAdapter) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group handler setup")
	return nil
}

// Cleanup runs after the claims are finished (no-op for now)
func (h *handlerAdapter) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group handler cleanup")
	return nil
}

// ConsumeClaim runs for each partition/claim in its own goroutine per rebalance cycle
func (h *handlerAdapter) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
	// 	h.logger.Info("MESSAGE RECEIVED",
	// 	zap.String("topic", msg.Topic),
	// 	zap.String("value", string(msg.Value)),
	// 	zap.Int32("partition", msg.Partition),
	// 	zap.Int64("offset", msg.Offset),
	// )
		select {
		case h.jobs <- msg:
		case <-h.ctx.Done():
			h.closeJobs()
			return nil
		case <-h.stopped:
			return nil
		}
		// ACK message after enqueue
		session.MarkMessage(msg, "")
	}
	return nil
}

// worker processes messages in a pool with robust error handling & retry.
func (h *handlerAdapter) worker() {
	for {
		select {
		case <-h.stopped:
			return
		case msg, ok := <-h.jobs:
			if !ok {
				return
			}
			handlers := h.topicHandlers[msg.Topic]
			if len(handlers) == 0 {
				h.logger.Warn("No handlers registered for topic",
					zap.String("topic", msg.Topic),
					zap.Int32("partition", msg.Partition),
					zap.Int64("offset", msg.Offset))
				continue
			}
			var wg sync.WaitGroup
			for idx, handler := range handlers {
				wg.Add(1)
				go func(idx int, handler ports.MessageHandler) {
					defer wg.Done()
					err := h.executeHandler(handler, msg, idx)
					if err != nil {
						h.reportHandlerError(err, msg, idx)
					}
				}(idx, handler)
			}
			wg.Wait()
		}
	}
}

// More robust: return error for central reporting
func (h *handlerAdapter) executeHandler(handler ports.MessageHandler, msg *sarama.ConsumerMessage, handlerIndex int) error {
	var lastErr error
	for attempt := 1; attempt <= h.config.Retries; attempt++ {
		ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
		err := handler.Handle(ctx, msg.Value)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		h.logger.Warn("Handler processing failed, will retry if allowed",
			zap.Int("attempt", attempt),
			zap.Int("handler_index", handlerIndex),
			zap.String("topic", msg.Topic),
			zap.Int32("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
			zap.Error(err),
		)
		if attempt < h.config.Retries {
			time.Sleep(h.config.RetryDelay * time.Duration(attempt))
		}
	}
	return lastErr
}

func (h *handlerAdapter) reportHandlerError(err error, msg *sarama.ConsumerMessage, handlerIndex int) {
	if err == nil {
		return
	}
	h.logger.Error("Message handling failed after retries",
		zap.Int("handler_index", handlerIndex),
		zap.String("topic", msg.Topic),
		zap.Int32("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
		zap.Error(err),
	)
	select {
	case h.errorsCh <- err:
	default:
	}
}

func (h *handlerAdapter) closeJobs() {
	h.stopOnce.Do(func() {
		close(h.stopped)
		close(h.jobs)
	})
}

// sanitizeTopics deduplicates, sorts & cleans topic list.
func sanitizeTopics(topics []string) []string {
	set := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		if t == "" {
			continue
		}
		set[t] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for t := range set {
		result = append(result, t)
	}
	sort.Strings(result)
	return result
}

// cloneTopicHandlers makes a deep copy of the map for handler safety.
func cloneTopicHandlers(source map[string][]ports.MessageHandler) map[string][]ports.MessageHandler {
	dest := make(map[string][]ports.MessageHandler, len(source))
	for topic, handlers := range source {
		dstHandlers := make([]ports.MessageHandler, len(handlers))
		copy(dstHandlers, handlers)
		dest[topic] = dstHandlers
	}
	return dest
}
