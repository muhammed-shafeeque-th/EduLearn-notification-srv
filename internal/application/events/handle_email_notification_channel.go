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
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/utils"
)

type HandleEmailNotificationChannel struct {
	notificationRepo repository.NotificationRepository
	emailSender      ports.EmailSender[ports.NotificationLike]
	logger           ports.LoggerService
}

func NewHandleEmailNotificationChannel(
	notificationRepo repository.NotificationRepository,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger ports.LoggerService,
) *HandleEmailNotificationChannel {
	return &HandleEmailNotificationChannel{
		notificationRepo: notificationRepo,
		emailSender:      emailSender,
		logger:           logger,
	}
}

func (h *HandleEmailNotificationChannel) Handle(ctx context.Context, message []byte) error {
	h.logger.Info("Received request on email notification channel", logger.ByteString("raw_message", message))

	var event domain_events.EmailNotificationEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error("Failed to unmarshal EmailNotificationEvent", logger.Error(err))
		return fmt.Errorf("invalid email notification event json: %w", err)
	}

	payload := event.Payload

	h.logger.Info("Event unmarshaled", logger.String("to", payload.To), logger.String("event_id", event.EventID))

	// Validate payload
	if err := payload.Validate(); err != nil {
		h.logger.Error("Email notification event validation failed", logger.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Idempotency check
	if h.notificationRepo != nil && event.EventID != "" {
		alreadyProcessed, err := h.notificationRepo.CheckIfProcessed(ctx, event.EventID)
		if err != nil {
			h.logger.Error("Failed to check idempotency", logger.Error(err))
			return fmt.Errorf("idempotency check failed: %w", err)
		}
		if alreadyProcessed {
			h.logger.Info(
				"Email notification event already processed; skipping.",
				logger.String("event_id", event.EventID),
			)
			return nil
		}
	}

	notification := buildEmailNotificationFromEvent(&event.Payload)

	if notification == nil {
		h.logger.Error("Failed to build email notification from event (incomplete fields)", logger.Any("event", event))
		return errors.New("failed to build email notification from event")
	}

	if err := h.emailSender.Send(ctx, notification); err != nil {
		h.logger.Error(
			"Failed to send email notification",
			logger.String("email", payload.To),
			logger.Error(err),
		)
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	if h.notificationRepo != nil && event.EventID != "" {
		if err := h.notificationRepo.MarkAsProcessed(ctx, event.EventID); err != nil {
			h.logger.Error("Failed to mark notification as processed", logger.Error(err))
		}
	}

	h.logger.Info(
		"Email notification sent successfully",
		logger.String("user_id", payload.UserID),
		logger.String("to", payload.To),
		logger.String("event_id", event.EventID),
	)

	return nil
}

func buildEmailNotificationFromEvent(payload *domain_events.EmailNotificationPayload) *entity.Notification {
	if payload == nil || payload.To == "" || payload.Subject == "" || payload.Body == "" {
		return nil
	}
	return &entity.Notification{
		ID:        utils.GenerateID(),
		UserId:    payload.UserID,
		Type:      entity.EmailNotification,
		Subject:   payload.Subject,
		Body:      payload.Body,
		Recipient: payload.To,
		Priority:  payload.Priority,
	}
}
