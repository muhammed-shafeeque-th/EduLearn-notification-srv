package ports

import (
	"context"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

// NotificationSender is an interface that defines a contract
// for sending a notification of any type T.
// type NotificationSender[T any] interface {
// 	// Send sends the provided notification of type T using the given context.
// 	// Returns an error if sending fails.
// 	Send(ctx context.Context, notification T) error
// }
type NotificationSender interface {
	// Send sends the provided notification of type T using the given context.
	// Returns an error if sending fails.
	Send(ctx context.Context, notification *entity.Notification) error
}
