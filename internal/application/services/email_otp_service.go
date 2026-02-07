package services

import (
	"context"
	"fmt"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/utils"
	"go.uber.org/zap"
)

type EmailOTPService struct {
	otpRepo       repository.OTPRepository
	renderer      ports.TemplateRenderer
	messageBroker ports.MessageBroker
	emailSender   ports.EmailSender[ports.NotificationLike]
	logger        *zap.Logger
}

func NewEmailOTPService(
	otpRepo repository.OTPRepository,
	renderer ports.TemplateRenderer,
	broker ports.MessageBroker,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger *zap.Logger,
) *EmailOTPService {
	return &EmailOTPService{
		otpRepo:       otpRepo,
		renderer:      renderer,
		messageBroker: broker,
		emailSender:   emailSender,
		logger:        logger,
	}
}

func (s *EmailOTPService) SendOTP(ctx context.Context, userID, email, username string) error {
	// Generate OTP
	otp, err := entity.NewOTP(userID, email)
	if err != nil {
		s.logger.Error("Failed to generate OTP", zap.Error(err))
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	if err := s.otpRepo.SaveOTP(ctx, otp); err != nil {
		s.logger.Error("Failed to save OTP", zap.String("email", email), zap.Error(err))
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	body, err := s.buildOTPEmailBody(username, otp.Code)

	if err != nil {
		s.logger.Error("failed to render OTP template", zap.Error(err))
		return fmt.Errorf("failed to render OTP template: %w", err)
	}

	notification := entity.Notification{
		ID:        utils.GenerateID(),
		UserId:    userID,
		Type:      entity.EmailNotification,
		Subject:   "Your OTP Code",
		Body:      body,
		Recipient: email,
	}

	// Send via email sender
	if err := s.emailSender.Send(ctx, &notification); err != nil {
		s.logger.Error("Failed to send OTP email",
			zap.String("email", email),
			zap.Error(err))
		return fmt.Errorf("failed to send OTP email: %w", err)
	}
	s.logger.Debug("OTP for debug with email and code",
		zap.String("email", email),
		zap.String("otp", otp.Code))

	s.logger.Info("OTP sent successfully",
		zap.String("otp", otp.Code),
		zap.String("userId", userID),
		zap.String("email", email))

	return nil
}

// VerifyOTP validates the provided OTP
func (s *EmailOTPService) VerifyOTP(ctx context.Context, email, code string) (bool, error) {
	s.logger.Warn("OTP verification Request received",
		zap.String("email", email))

	otp, err := s.otpRepo.GetOTP(ctx, email)
	if err != nil {
		if err == domain_errors.ErrOTPNotFound {
			return false, nil
		}
		s.logger.Error("Failed to retrieve OTP", zap.String("email", email), zap.Error(err))
		return false, err
	}

	s.logger.Warn("OTP retrieved ",
		zap.String("email", email), zap.String("code", otp.Code))
	// Verify OTP
	if err := otp.Verify(code); err != nil {
		s.logger.Warn("OTP verification failed",
			zap.String("email", email),
			zap.Int("attempts", otp.Attempts),
			zap.Error(err))

		// Update attempts in storage
		_ = s.otpRepo.SaveOTP(ctx, otp)

		s.logger.Warn("OTP Verification failed ",
			zap.String("email", email), zap.String("code", otp.Code))

		return false, nil
	}

	// Delete OTP after successful verification
	if err := s.otpRepo.DeleteOTP(ctx, email); err != nil {
		s.logger.Error("Failed to delete OTP after verification",
			zap.String("email", email),
			zap.Error(err))
	}

	s.logger.Warn("OTP Verified event publishing ",
		zap.String("email", email), zap.String("code", otp.Code))
	// Publish OTP verified event to Kafka
	if err := s.publishOTPVerifiedEvent(ctx, otp.UserId, email); err != nil {
		s.logger.Error("Failed to publish OTP verified event",
			zap.String("email", email),
			zap.Error(err))
	}

	s.logger.Info("OTP verified successfully",
		zap.String("email", email),
		zap.String("userId", otp.UserId))

	return true, nil
}

func (s *EmailOTPService) publishOTPVerifiedEvent(ctx context.Context, userID, email string) error {
	event := &domain_events.OTPVerifiedEvent{
		Payload: domain_events.OTPVerifiedEventPayload{UserID: userID,
			Email:  email,
			Status: "verified",
		},
		BaseEvent: domain_events.BaseEvent{
			EventID:   utils.GenerateID(),
			EventType: "OTPVerifiedEvent",
			Timestamp: time.Now().Unix(),
			Source:    "notification-service",
		},
	}

	// data, err := json.Marshal(event)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal event: %w", err)
	// }

	return s.messageBroker.PublishOTPVerifiedEvent(ctx, event)
}

// creates HTML email body
func (s *EmailOTPService) buildOTPEmailBody(username, code string, expiryInMinutes ...int) (string, error) {
	expiry := 10 // default 
	if len(expiryInMinutes) > 0 && expiryInMinutes[0] > 0 {
		expiry = expiryInMinutes[0]
	}

	body, err := s.renderer.Render("activation-mail.html", map[string]string{
		"USER_NAME":   username,
		"OTP_CODE":    code,
		"EXPIRY_TIME": fmt.Sprintf("%d minutes", expiry),
	})
	if err != nil {
		s.logger.Error("Failed to render OTP template", zap.String("username", username), zap.String("code", code), zap.Error(err))
		return "", err
	}
	return string(body), nil
}
