package notification

import (
	"context"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

type NotificationSender struct {
	strategies map[entity.NotificationType][]ports.NotificationSender
}

func NewNotificationSender(strategies map[entity.NotificationType][]ports.NotificationSender) *NotificationSender {
	return &NotificationSender{strategies: strategies}
}

func (s *NotificationSender) Send(ctx context.Context, n *entity.Notification) error {
	strategies, ok := s.strategies[n.Type]
	if !ok || len(strategies) == 0 {
		return fmt.Errorf("no strategy for notification type: %s", n.Type)
	}
	for _, st := range strategies {
		if err := st.Send(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
