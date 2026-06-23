package ports

import (
	"context"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

// Any struct implementing these methods will satisfy the NotificationLike interface.
type NotificationLike interface {
	GetUserId() string
	GetType() entity.NotificationType
	GetSubject() string
	GetBody() string
	GetRecipient() string
	GetIsRead() bool
}


// This provides encapsulation (fields are private) and
// allows any struct with those methods to be used as NotificationLike.
// type NumberLike interface {
//     ~int | ~int8 | ~int16 | ~int32 | ~int64 |
//     ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
//     ~float32 | ~float64
// }

// A generic function that accepts only types satisfying the NotificationLike constraint


// EmailSender interface for sending emails
type EmailSender[T NotificationLike] interface {
	Send(ctx context.Context, args T ) error
}	