package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"go.uber.org/zap"
)

type ForgotPasswordEventHandler struct {
	renderer         ports.TemplateRenderer
	notificationRepo repository.NotificationRepository
	messageBroker    ports.MessageBroker
	emailSender      ports.EmailSender[ports.NotificationLike]
	logger           *zap.Logger
}

func NewForgotPasswordEventHandler(
	renderer ports.TemplateRenderer,
	notificationRepo repository.NotificationRepository,
	messageBroker ports.MessageBroker,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger *zap.Logger,
) *ForgotPasswordEventHandler {
	return &ForgotPasswordEventHandler{
		renderer:         renderer,
		notificationRepo: notificationRepo,
		messageBroker:    messageBroker,
		emailSender:      emailSender,
		logger:           logger,
	}
}

// Handle processes a forgot password event message by validating, rendering, and emailing reset instructions.
func (s *ForgotPasswordEventHandler) Handle(ctx context.Context, message []byte) error {
	var event domain_events.ForgotPasswordRequestEvent
	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Error("Failed to unmarshal ForgotPasswordRequestEvent", zap.Error(err))
		return fmt.Errorf("invalid forgot-password-request event json: %w", err)
	}

	// Validate event fields
	if err := event.Payload.Validate(); err != nil {
		s.logger.Error("Invalid forgot-password-request event", zap.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Idempotency check if repo is provided
	if s.notificationRepo != nil && event.EventID != "" {
		alreadyProcessed, err := s.notificationRepo.CheckIfProcessed(ctx, event.EventID)
		if err != nil {
			s.logger.Error("Failed to check idempotency for forgot password notification", zap.Error(err))
			return fmt.Errorf("idempotency check failed: %w", err)
		}
		if alreadyProcessed {
			s.logger.Info("Forgot password event already processed, skipping.",
				zap.String("notification_id", event.EventID),
			)
			return nil
		}
	}

	payload := event.Payload

	// Render email template
	body, err := s.renderer.Render("forgot-mail.html", map[string]string{
		"RESET_LINK":  payload.ResetLink,
		"USER_NAME":   payload.Username,
		"EXPIRY_TIME": fmt.Sprintf("%d minutes", payload.Expiry),
	})
	if err != nil {
		s.logger.Error("Failed to render forgot-password template", zap.String("username", payload.Username), zap.Error(err))
		return fmt.Errorf("failed to render forgot password email template: %w", err)
	}

	notification := &entity.Notification{
		ID:        event.EventID, // Use payload.EventID for deduplication/idempotency
		UserId:    payload.UserID,
		Type:      entity.EmailNotification,
		Subject:   "Password Reset Request",
		Body:      body,
		Recipient: payload.Email,
	}

	// Send email
	if err := s.emailSender.Send(ctx, notification); err != nil {
		s.logger.Error("Failed to send forgot password email",
			zap.String("email", payload.Email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send forgot password email: %w", err)
	}

	// Mark processed if needed
	if s.notificationRepo != nil && event.EventID != "" {
		if err := s.notificationRepo.MarkAsProcessed(ctx, event.EventID); err != nil {
			s.logger.Error("Failed to mark notification as processed", zap.Error(err))
			// Not a terminal error; log and continue
		}
	}

	s.logger.Info("Forgot password email sent successfully",
		zap.String("userId", payload.UserID),
		zap.String("email", payload.Email),
		zap.String("notification_id", event.EventID),
	)

	return nil
}
