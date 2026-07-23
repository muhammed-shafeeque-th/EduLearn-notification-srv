package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	log "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
)

type Producer struct {
	producer sarama.SyncProducer
	logger   ports.LoggerService
}

func NewProducer(brokers []string, logger ports.LoggerService) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Return.Successes = true

	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		logger.Error("failed to create kafka producer", log.Error(err))
		return nil, err
	}
	return &Producer{producer: prod, logger: logger}, nil
}

func (p *Producer) Produce(ctx context.Context, topic string, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(value),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, _, err := p.producer.SendMessage(msg)
		if err != nil {
			p.logger.Error("failed to send kafka message", log.String("topic", topic), log.Error(err))
			return err
		}
		return nil
	}
}
func (p *Producer) ProduceWithKey(ctx context.Context, topic string, key string, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, _, err := p.producer.SendMessage(msg)
		if err != nil {
			p.logger.Error("failed to send kafka message", log.String("topic", topic), log.Error(err))
			return err
		}
		return nil
	}
}

func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		p.logger.Error("failed to close kafka producer", log.Error(err))
		return err
	}
	return nil
}
