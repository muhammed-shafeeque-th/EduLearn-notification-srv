package kafka

import (
	"context"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/services"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	ws "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
	"go.uber.org/zap"
)

type NotificationProcessor struct {
	NotificationSender    ports.NotificationSender
	EmailOTPService       *services.EmailOTPService
	ForgotPasswordService *services.ForgotPasswordService
	WSHub                 *ws.Hub
	Logger                *zap.Logger
}

func NewNotificationProcessor(
	NotificationSender ports.NotificationSender,
	EmailOTPService *services.EmailOTPService,
	ForgotPasswordService *services.ForgotPasswordService,
	WSHub *ws.Hub,
	Logger *zap.Logger) ports.NotificationProcessor {

	return &NotificationProcessor{
		NotificationSender:    NotificationSender,
		EmailOTPService:       EmailOTPService,
		ForgotPasswordService: ForgotPasswordService,
		WSHub:                 WSHub,
		Logger:                Logger,
	}
}

func (np *NotificationProcessor) Process(ctx context.Context, notification *entity.Notification) error {
 return nil
}
