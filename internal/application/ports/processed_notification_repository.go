package ports

import (
	"context"

)

type ProcessedNotificationRepository interface {
	IsProcessed(ctx context.Context, notificationID string) (bool, error)
	MarkProcessed(ctx context.Context, notificationID string) error
}
