package ports

import (
	"context"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
)

type EventPublisher interface {
	PublishOTPVerifiedEvent(ctx context.Context, event *domain_events.OTPVerifiedEvent) error
}

// Message broker abstraction
type MessageBroker interface {
	EventPublisher
	Publish(ctx context.Context, topic string, message []byte) error
	Subscribe(topics string, handler ...MessageHandler) error
	StartConsuming(ctx context.Context) error
	Close() error
}

type MessageHandler interface {
	Handle(ctx context.Context, message []byte) error
}
