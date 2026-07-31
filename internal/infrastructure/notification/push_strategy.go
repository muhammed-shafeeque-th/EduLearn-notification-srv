package notification

import (
	"context"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
)

// PushStrategy handles push notifications (placeholder)
type PushStrategy struct {
	logger ports.LoggerService
}

// NewPushStrategy creates a new push notification strategy
func NewPushStrategy(logger ports.LoggerService) ports.NotificationSender {
	return &PushStrategy{
		logger: logger,
	}
}

// Send sends push notification
func (s *PushStrategy) Send(ctx context.Context, n *entity.Notification) error {
	// TODO: Implement push notification logic
	s.logger.Info("Push notification (not implemented)",
		logger.String("notification_id", n.ID),
		logger.String("user_id", n.UserId))

	return fmt.Errorf("push strategy not implemented yet")
}
