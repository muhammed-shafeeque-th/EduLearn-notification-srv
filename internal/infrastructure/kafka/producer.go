package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type Producer struct {
	producer sarama.SyncProducer
	logger   *zap.Logger
}

func NewProducer(brokers []string, logger *zap.Logger) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Return.Successes = true

	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		logger.Error("failed to create kafka producer", zap.Error(err))
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
			p.logger.Error("failed to send kafka message", zap.String("topic", topic), zap.Error(err))
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
			p.logger.Error("failed to send kafka message", zap.String("topic", topic), zap.Error(err))
			return err
		}
		return nil
	}
}

func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		p.logger.Error("failed to close kafka producer", zap.Error(err))
		return err
	}
	return nil
}
