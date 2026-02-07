package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	"go.uber.org/zap"
)

// MessageBrokerAdapter implements ports.MessageBroker using Kafka
type MessageBrokerAdapter struct {
	producer       *Producer
	consumer       *Consumer
	logger         *zap.Logger
	publishMetrics *PublishMetrics
	mu             sync.RWMutex
}

// PublishMetrics tracks publishing metrics
type PublishMetrics struct {
	TotalPublished     int64
	FailedPublished    int64
	OTPEvents          int64
	PasswordEvents     int64
	NotificationEvents int64
}

// NewMessageBrokerAdapter creates a new Kafka message broker adapter
func NewMessageBrokerAdapter(producer *Producer, consumer *Consumer) ports.MessageBroker {
	logger := producer.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &MessageBrokerAdapter{
		consumer:       consumer,
		producer:       producer,
		logger:         logger,
		publishMetrics: &PublishMetrics{},
	}
}

// Publish sends a raw message to a topic
func (a *MessageBrokerAdapter) Publish(ctx context.Context, topic string, message []byte) error {
	if topic == "" {
		return errors.New("topic must not be empty")
	}
	if len(message) == 0 {
		return errors.New("message must not be empty")
	}

	a.logger.Debug("Publishing message", zap.String("topic", topic), zap.Int("size", len(message)))
	err := a.producer.Produce(ctx, topic, message)
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.publishMetrics.FailedPublished++
		a.logger.Error("Failed to publish message", zap.String("topic", topic), zap.Error(err))
		return fmt.Errorf("failed to publish message: %w", err)
	}
	a.publishMetrics.TotalPublished++
	a.logger.Debug("Message published successfully", zap.String("topic", topic))
	return nil
}

// Subscribe sets up a consumer for a topic and handler(s)
func (a *MessageBrokerAdapter) Subscribe(topic string, handlers ...ports.MessageHandler) error {
	if a.consumer == nil {
		return errors.New("consumer is not configured")
	}
	if topic == "" {
		return errors.New("topic must not be empty")
	}
	if len(handlers) == 0 {
		return errors.New("at least one handler must be provided")
	}
	return a.consumer.RegisterHandlers(topic, handlers...)
}

// StartConsuming starts the consumer loop
func (a *MessageBrokerAdapter) StartConsuming(ctx context.Context) error {
	if a.consumer == nil {
		return errors.New("consumer is not configured")
	}
	return a.consumer.StartConsuming(ctx)
}

func (a *MessageBrokerAdapter) PublishOTPVerifiedEvent(ctx context.Context, event *domain_events.OTPVerifiedEvent) error {
	a.logger.Info("Publishing OTP verified event",
		zap.String("user_id", event.Payload.UserID),
		zap.String("email", event.Payload.Email),
		zap.String("status", event.Payload.Status),
	)

	if err := a.validateOTPVerifiedEvent(&event.Payload); err != nil {
		a.logger.Error("Invalid OTP verified event", zap.Error(err))
		return fmt.Errorf("invalid event: %w", err)
	}

	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	data, err := json.Marshal(event)
	if err != nil {
		a.logger.Error("Failed to marshal OTP verified event", zap.String("user_id", event.Payload.UserID), zap.Error(err))
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = a.producer.ProduceWithKey(ctx, string(domain_events.TopicAuthOTPVerified), event.Payload.UserID, data)
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.publishMetrics.FailedPublished++
		a.logger.Error("Failed to publish OTP verified event", zap.String("user_id", event.Payload.UserID), zap.Error(err))
		return fmt.Errorf("failed to publish event: %w", err)
	}
	a.publishMetrics.TotalPublished++
	a.publishMetrics.OTPEvents++
	a.logger.Info("OTP verified event published successfully", zap.String("user_id", event.Payload.UserID), zap.String("email", event.Payload.Email))
	return nil
}

func (a *MessageBrokerAdapter) PublishNotificationEvent(ctx context.Context, event domain_events.NotificationEvent) error {
	a.logger.Info("Publishing notification event",
		zap.String("notification_id", event.Payload.ID),
		zap.String("user_id", event.Payload.UserID),
		zap.String("type", string(event.Payload.Type)))

	if err := a.validateNotificationEvent(event.Payload); err != nil {
		a.logger.Error("Invalid notification event", zap.Error(err))
		return fmt.Errorf("invalid event: %w", err)
	}

	if event.Payload.Timestamp.IsZero() {
		event.Payload.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(event)
	if err != nil {
		a.logger.Error("Failed to marshal notification event",
			zap.String("notification_id", event.Payload.ID), zap.Error(err))
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	topic := a.notificationTypeToTopic(event.Payload.Type)

	err = a.producer.ProduceWithKey(ctx, topic, event.Payload.UserID, data)
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		a.publishMetrics.FailedPublished++
		a.logger.Error("Failed to publish notification event",
			zap.String("notification_id", event.Payload.ID),
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("failed to publish event: %w", err)
	}
	a.publishMetrics.TotalPublished++
	a.publishMetrics.NotificationEvents++
	a.logger.Info("Notification event published successfully",
		zap.String("notification_id", event.Payload.ID),
		zap.String("user_id", event.Payload.UserID),
		zap.String("topic", topic),
	)
	return nil
}

// Close closes the message broker and underlying producer
func (a *MessageBrokerAdapter) Close() error {
	a.logger.Info("Closing message broker",
		zap.Int64("total_published", a.publishMetrics.TotalPublished),
		zap.Int64("failed", a.publishMetrics.FailedPublished))
	if a.producer != nil {
		return a.producer.Close()
	}
	return nil
}

// GetMetrics returns a snapshot of publishing metrics
func (a *MessageBrokerAdapter) GetMetrics() PublishMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return *a.publishMetrics
}

// Validation helpers

func (a *MessageBrokerAdapter) validateOTPVerifiedEvent(event *domain_events.OTPVerifiedEventPayload) error {
	switch {
	case event.UserID == "":
		return errors.New("user_id is required")
	case event.Email == "":
		return errors.New("email is required")
	case event.Status == "":
		return errors.New("status is required")
	default:
		return nil
	}
}

func (a *MessageBrokerAdapter) validateNotificationEvent(event domain_events.NotificationEventPayload) error {
	switch {
	case event.ID == "":
		return errors.New("notification_id is required")
	case event.UserID == "":
		return errors.New("user_id is required")
	case event.Type == "":
		return errors.New("type is required")
	case !event.Type.IsValid():
		return fmt.Errorf("invalid notification type: %s", event.Type)
	default:
		return nil
	}
}

// notificationTypeToTopic maps notification type to Kafka topic
func (a *MessageBrokerAdapter) notificationTypeToTopic(notificationType entity.NotificationType) string {
	switch notificationType {
	case entity.EmailNotification:
		return string(domain_events.TopicNotificationEmailChannel)
	case entity.SMSNotification:
		return string(domain_events.TopicNotificationSMSChannel)
	case entity.PushNotification:
		return string(domain_events.TopicNotificationPushChannel)
	case entity.InAppNotification:
		return string(domain_events.TopicNotificationInAppChannel)
	default:
		a.logger.Warn("Unknown notification type, defaulting to email",
			zap.String("type", string(notificationType)))
		return string(domain_events.TopicNotificationEmailChannel)
	}
}

// PublishNotification is a convenience method to publish notification directly
func (a *MessageBrokerAdapter) PublishNotification(ctx context.Context, notification entity.Notification) error {
	a.logger.Debug("Publishing notification",
		zap.String("notification_id", notification.ID),
		zap.String("type", string(notification.Type)))

	if err := notification.Validate(); err != nil {
		return fmt.Errorf("invalid notification: %w", err)
	}
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	topic := a.notificationTypeToTopic(notification.Type)
	return a.producer.ProduceWithKey(ctx, topic, notification.UserId, data)
}

// PublishBatch publishes multiple messages in a batch.
// If any individual publish operation fails, it continues and logs the error. It returns an error at the end if any failed.
func (a *MessageBrokerAdapter) PublishBatch(ctx context.Context, messages map[string][]byte) error {
	if len(messages) == 0 {
		return nil
	}

	a.logger.Info("Publishing batch of messages", zap.Int("count", len(messages)))

	var failedCount int
	for topic, message := range messages {
		if err := a.Publish(ctx, topic, message); err != nil {
			a.logger.Error("Failed to publish message in batch", zap.String("topic", topic), zap.Error(err))
			failedCount++
		}
	}

	if failedCount > 0 {
		return fmt.Errorf("batch publish completed with %d failures out of %d", failedCount, len(messages))
	}

	a.logger.Info("Batch messages published successfully", zap.Int("count", len(messages)))
	return nil
}

// Producer enhancement methods

// ProduceAsync sends a message asynchronously
func (p *Producer) ProduceAsync(ctx context.Context, topic string, message []byte, callback func(error)) {
	go func() {
		err := p.Produce(ctx, topic, message)
		if callback != nil {
			callback(err)
		}
	}()
}

// ProduceWithRetry sends a message with retry logic and exponential backoff
func (p *Producer) ProduceWithRetry(ctx context.Context, topic string, message []byte, maxRetries int) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := p.Produce(ctx, topic, message)
		if err == nil {
			return nil
		}
		lastErr = err
		p.logger.Warn("Failed to produce message, retrying",
			zap.String("topic", topic),
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries),
			zap.Error(err),
		)
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// retry
			}
		}
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// Health returns error if broker is not healthy
func (a *MessageBrokerAdapter) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testMessage := []byte(fmt.Sprintf(`{"type":"health_check","timestamp":"%s"}`, time.Now().Format(time.RFC3339)))
	err := a.producer.Produce(ctx, "health-check", testMessage)
	if err != nil {
		a.logger.Error("Message broker health check failed", zap.Error(err))
		return fmt.Errorf("message broker unhealthy: %w", err)
	}
	return nil
}