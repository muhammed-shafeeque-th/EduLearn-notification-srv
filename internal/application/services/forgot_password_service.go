package services

import (
	"context"
	"fmt"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/utils"
)

type ForgotPasswordService struct {
	renderer      ports.TemplateRenderer
	messageBroker ports.MessageBroker
	emailSender   ports.EmailSender[ports.NotificationLike]
	logger        ports.LoggerService
}

func NewForgotPasswordService(
	renderer ports.TemplateRenderer,
	messageBroker ports.MessageBroker,
	emailSender ports.EmailSender[ports.NotificationLike],
	logger ports.LoggerService,
) *ForgotPasswordService {
	return &ForgotPasswordService{
		renderer:      renderer,
		messageBroker: messageBroker,
		emailSender:   emailSender,
		logger:        logger,
	}
}

// Composes and sends a password reset email to the user.
func (s *ForgotPasswordService) Send(ctx context.Context, userID, username, email, resetLink string) error {
	const expiryMinutes = 10

	body, err := s.renderer.Render("forgot-mail.html", map[string]string{
		"RESET_LINK":  resetLink,
		"USER_NAME":   username,
		"EXPIRY_TIME": fmt.Sprintf("%d minutes", expiryMinutes),
	})
	if err != nil {
		s.logger.Error("Failed to render forgot-password template", logger.String("username", username), logger.Error(err))
		return fmt.Errorf("failed to render forgot password email template: %w", err)
	}

	notification := entity.Notification{
		ID:        utils.GenerateID(),
		UserId:    userID,
		Type:      entity.EmailNotification,
		Subject:   "Password Reset Request",
		Body:      body,
		Recipient: email,
	}

	if err := s.emailSender.Send(ctx, &notification); err != nil {
		s.logger.Error("Failed to send forgot password email",
			logger.String("email", email),
			logger.Error(err),
		)
		return fmt.Errorf("failed to send forgot password email: %w", err)
	}

	s.logger.Info("Forgot password email sent successfully",
		logger.String("userId", userID),
		logger.String("email", email),
	)

	return nil
}
