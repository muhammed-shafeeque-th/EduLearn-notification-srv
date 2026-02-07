package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"go.uber.org/zap"
)

// EmailStrategy handles email notifications
type EmailStrategy struct {
	emailSender ports.EmailSender[ports.NotificationLike]
	logger      *zap.Logger
}

// NewEmailStrategy creates a new email notification strategy
func NewEmailStrategy(
	emailSender ports.EmailSender[ports.NotificationLike],
	logger *zap.Logger,
) ports.NotificationSender {
	return &EmailStrategy{
		emailSender: emailSender,
		logger:      logger,
	}
}

// Send sends email notification
func (s *EmailStrategy) Send(ctx context.Context, n *entity.Notification) error {
	// Ensure required fields
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	if n.Type == "" {
		n.Type = entity.NotificationType(entity.EmailNotification)
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	n.UpdatedAt = time.Now().UTC()

	// Validate notification
	if err := n.Validate(); err != nil {
		s.logger.Error("Invalid email notification",
			zap.Error(err))
		return fmt.Errorf("invalid notification: %w", err)
	}

	// Send email
	if err := s.emailSender.Send(ctx, n); err != nil {
		s.logger.Error("Failed to send email notification",
			zap.String("notification_id", n.ID),
			zap.String("recipient", n.Recipient),
			zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("Email notification sent",
		zap.String("notification_id", n.ID),
		zap.String("recipient", n.Recipient),
		zap.String("subject", n.Subject))

	return nil
}
