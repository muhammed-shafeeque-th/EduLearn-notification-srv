package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/utils"
	"go.uber.org/zap"
)

// OTPRequestEventHandler handles OTP request events such as sending OTP emails.
type OTPRequestEventHandler struct {
	renderer         ports.TemplateRenderer
	notificationRepo repository.NotificationRepository
	otpRepo          repository.OTPRepository
	messageBroker    ports.MessageBroker
	emailSender      ports.EmailSender[ports.NotificationLike]
	logger           *zap.Logger
}

func NewOTPRequestEventHandler(
	renderer ports.TemplateRenderer,
	notificationRepo repository.NotificationRepository,
	otpRepo repository.OTPRepository,
	messageBroker ports.MessageBroker,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger *zap.Logger,
) *OTPRequestEventHandler {
	return &OTPRequestEventHandler{
		renderer:         renderer,
		notificationRepo: notificationRepo,
		otpRepo:          otpRepo,
		messageBroker:    messageBroker,
		emailSender:      emailSender,
		logger:           logger,
	}
}

func (s *OTPRequestEventHandler) Handle(ctx context.Context, message []byte) error {
	s.logger.Info("Request received for OTP request")
	var event domain_events.OTPRequestEvent
	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Error("Failed to unmarshal OTPRequestEvent", zap.Error(err))
		return fmt.Errorf("invalid otp-request event json: %w", err)
	}
	s.logger.Info("Successfully Unmarshaled event message", zap.String("userId", event.Payload.UserID), zap.String("Email", event.Payload.Email))

	// Validate event fields
	if err := event.Payload.Validate(); err != nil {
		s.logger.Error("Invalid OTP request event", zap.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Idempotency check if repo is provided
	if s.notificationRepo != nil && event.EventID != "" {
		alreadyProcessed, err := s.notificationRepo.CheckIfProcessed(ctx, event.EventID)
		if err != nil {
			s.logger.Error("Failed to check idempotency for OTP notification", zap.Error(err))
			return fmt.Errorf("idempotency check failed: %w", err)
		}
		if alreadyProcessed {
			s.logger.Info("OTP request event already processed, skipping.",
				zap.String("notification_id", event.EventID),
			)
			return nil
		}
	}

	payload := event.Payload;

	// Generate OTP
	otp, err := entity.NewOTP(payload.UserID, payload.Email)
	if err != nil {
		s.logger.Error("Failed to generate OTP",
			zap.String("userId", payload.UserID),
			zap.String("email", payload.Email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	// Save to OTP repository
	if s.otpRepo != nil {
		if err := s.otpRepo.SaveOTP(ctx, otp); err != nil {
			s.logger.Error("Failed to save OTP", zap.String("email", payload.Email), zap.Error(err))
			return fmt.Errorf("failed to save OTP: %w", err)
		}
	}

	body, err := s.buildOTPEmailBody(payload.Username, otp.Code)
	if err != nil {
		s.logger.Error(
			"Failed to render OTP template",
			zap.String("username", payload.Username),
			zap.Error(err),
		)
		return fmt.Errorf("failed to render OTP template: %w", err)
	}

	// Create email notification
	notification := &entity.Notification{
		ID:        utils.GenerateID(),
		UserId:    payload.UserID,
		Type:      entity.EmailNotification,
		Subject:   "Your OTP Code",
		Body:      body,
		Recipient: payload.Email,
	}

	// Send via email sender
	if err := s.emailSender.Send(ctx, notification); err != nil {
		s.logger.Error(
			"Failed to send OTP email",
			zap.String("email", payload.Email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	// Mark event as processed for idempotency
	if s.notificationRepo != nil && event.EventID != "" {
		if err := s.notificationRepo.MarkAsProcessed(ctx, event.EventID); err != nil {
			s.logger.Error("Failed to mark notification as processed", zap.Error(err))
		}
	}
	s.logger.Debug("OTP for debug with email and code",
		zap.String("email", payload.Email),
		zap.String("otp", otp.Code))

	s.logger.Info("OTP sent successfully",
		zap.String("otp", otp.Code),
		zap.String("userId", payload.UserID),
		zap.String("email", payload.Email),
		zap.String("notification_id", event.EventID),
	)

	return nil
}

// buildOTPEmailBody creates HTML email body
func (s *OTPRequestEventHandler) buildOTPEmailBody(username, code string, expiryInMinutes ...int) (string, error) {
	expiry := 10 // default value
	if len(expiryInMinutes) > 0 && expiryInMinutes[0] > 0 {
		expiry = expiryInMinutes[0]
	}

	body, err := s.renderer.Render("activation-mail.html", map[string]string{
		"USER_NAME":   username,
		"OTP_CODE":    code,
		"EXPIRY_TIME": fmt.Sprintf("%d minutes", expiry),
	})
	if err != nil {
		return "", err
	}

	return string(body), nil
}
