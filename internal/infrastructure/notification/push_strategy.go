package notification

import (
	"context"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"go.uber.org/zap"
)

// PushStrategy handles push notifications (placeholder)
type PushStrategy struct {
	logger *zap.Logger
}

// NewPushStrategy creates a new push notification strategy
func NewPushStrategy(logger *zap.Logger) ports.NotificationSender {
	return &PushStrategy{
		logger: logger,
	}
}

// Send sends push notification
func (s *PushStrategy) Send(ctx context.Context, n *entity.Notification) error {
	// TODO: Implement push notification logic
	s.logger.Info("Push notification (not implemented)",
		zap.String("notification_id", n.ID),
		zap.String("user_id", n.UserId))
	
	return fmt.Errorf("push strategy not implemented yet")
}