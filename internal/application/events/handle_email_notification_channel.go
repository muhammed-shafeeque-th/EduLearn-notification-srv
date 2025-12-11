package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/utils"
	"go.uber.org/zap"
)

// HandleEmailNotificationChannel is responsible for handling arbitrary email notifications.
type HandleEmailNotificationChannel struct {
	notificationRepo repository.NotificationRepository
	emailSender      ports.EmailSender[ports.NotificationLike]
	logger           *zap.Logger
}

// NewHandleEmailNotificationChannel constructs a new HandleEmailNotificationChannel with its dependencies.
func NewHandleEmailNotificationChannel(
	notificationRepo repository.NotificationRepository,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger *zap.Logger,
) *HandleEmailNotificationChannel {
	return &HandleEmailNotificationChannel{
		notificationRepo: notificationRepo,
		emailSender:      emailSender,
		logger:           logger,
	}
}

// Handle processes an EmailNotificationEvent: validates it, checks idempotency, sends the email, and marks the event as processed.
func (h *HandleEmailNotificationChannel) Handle(ctx context.Context, message []byte) error {
	h.logger.Info("Received request on email notification channel", zap.ByteString("raw_message", message))

	var event domain_events.EmailNotificationEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error("Failed to unmarshal EmailNotificationEvent", zap.Error(err))
		return fmt.Errorf("invalid email notification event json: %w", err)
	}

	h.logger.Info("Event unmarshaled", zap.String("to", event.To), zap.String("event_id", event.EventID))

	// Validate event
	if err := event.Validate(); err != nil {
		h.logger.Error("Email notification event validation failed", zap.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Idempotency check
	if h.notificationRepo != nil && event.EventID != "" {
		alreadyProcessed, err := h.notificationRepo.CheckIfProcessed(ctx, event.EventID)
		if err != nil {
			h.logger.Error("Failed to check idempotency", zap.Error(err))
			return fmt.Errorf("idempotency check failed: %w", err)
		}
		if alreadyProcessed {
			h.logger.Info(
				"Email notification event already processed; skipping.",
				zap.String("event_id", event.EventID),
			)
			return nil
		}
	}

	notification := buildEmailNotificationFromEvent(&event)

	if notification == nil {
		h.logger.Error("Failed to build email notification from event (incomplete fields)", zap.Any("event", event))
		return errors.New("failed to build email notification from event")
	}

	if err := h.emailSender.Send(ctx, notification); err != nil {
		h.logger.Error(
			"Failed to send email notification",
			zap.String("email", event.To),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	// Optionally persist event as processed
	if h.notificationRepo != nil && event.EventID != "" {
		if err := h.notificationRepo.MarkAsProcessed(ctx, event.EventID); err != nil {
			h.logger.Error("Failed to mark notification as processed", zap.Error(err))
		}
	}

	h.logger.Info(
		"Email notification sent successfully",
		zap.String("user_id", event.UserID),
		zap.String("to", event.To),
		zap.String("event_id", event.EventID),
	)

	return nil
}

// buildEmailNotificationFromEvent constructs a Notification entity based on properties from the event.
// Returns nil if required fields are missing.
func buildEmailNotificationFromEvent(event *domain_events.EmailNotificationEvent) *entity.Notification {
	if event == nil || event.To == "" || event.Subject == "" || event.Body == "" {
		return nil
	}
	return &entity.Notification{
		ID:        utils.GenerateID(),
		UserId:    event.UserID,
		Type:      entity.EmailNotification,
		Subject:   event.Subject,
		Body:      event.Body,
		Recipient: event.To,
		Priority:  event.Priority,
		// Optionally use event.Template, event.CC, event.BCC, event.Attachments in future
	}
}

