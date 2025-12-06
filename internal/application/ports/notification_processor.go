package ports

import (
	"context"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

// NotificationProcessor processes notification messages
type NotificationProcessor interface {
	Process(ctx context.Context, notification *entity.Notification) error
}