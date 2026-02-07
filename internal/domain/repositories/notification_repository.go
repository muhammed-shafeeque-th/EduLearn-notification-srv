package repository

import (
	"context"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)



type NotificationRepository interface {
	SaveNotification(ctx context.Context, notification *entity.Notification) error
	GetNotification(ctx context.Context, id, userID string) (*entity.Notification, error)
	GetNotifications(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error)
	MarkAsRead(ctx context.Context, id, userID string) error
	DeleteNotification(ctx context.Context, id, userID string) error
	ClearNotifications(ctx context.Context, userID string) error
	MarkAllAsRead(ctx context.Context, userID string) error
	
	// Idempotency
	CheckIfProcessed(ctx context.Context, id string) (bool, error)
	MarkAsProcessed(ctx context.Context, id string) error
}
