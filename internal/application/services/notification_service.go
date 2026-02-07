package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"go.uber.org/zap"
)

type NotificationService struct {
	repo   repository.NotificationRepository
	sender ports.NotificationSender
	logger *zap.Logger
}

func NewNotificationService(repo repository.NotificationRepository, sender ports.NotificationSender, logger *zap.Logger) *NotificationService {
	return &NotificationService{repo: repo, sender: sender, logger: logger}
}

func (s *NotificationService) CreateAndQueue(
	ctx context.Context,
	userID, recipient, subject, body string,
	notifyType entity.NotificationType,
) (*entity.Notification, error) {

	n := &entity.Notification{
		ID:        uuid.NewString(),
		UserId:    userID,
		Type:      notifyType,
		Subject:   subject,
		Body:      body,
		Recipient: recipient,
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.SaveNotification(ctx, n); err != nil {
		s.logger.Error("failed to save notification", zap.Error(err))
		return nil, domain_errors.ErrDatabase
	}

	if err := s.sender.Send(ctx, n); err != nil {
		s.logger.Error("failed to send notification via sender strategy",
			zap.Error(err),
			zap.String("notification_id", n.ID),
		)
		return n, err
	}

	return n, nil
}

func (s *NotificationService) GetNotification(ctx context.Context, notificationID, userID string) (*entity.Notification, error) {
	n, err := s.repo.GetNotification(ctx, notificationID, userID)
	if err != nil {
		s.logger.Error("failed to get notification",
			zap.String("notification_id", notificationID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}
	return n, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, userID string, page, pageSize int, isRead *bool, category *entity.NotificationCategory) ([]entity.Notification, int64, error) {

	filters := &domain.NotificationFilter{
		UserID:   userID,
		Category: category,
		IsRead:   isRead,
		Page:     page,
		PageSize: pageSize,
	}

	s.logger.Info("failed to list notifications",
		zap.String("user_id", userID),
		zap.Any("filters", filters),
	)
	ns, total, err := s.repo.GetNotifications(ctx, filters)
	if err != nil {
		s.logger.Error("failed to list notifications",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, 0, err
	}
	return ns, total, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID string) error {
	if err := s.repo.MarkAsRead(ctx, notificationID, userID); err != nil {
		s.logger.Error("failed to mark notification as read",
			zap.String("notification_id", notificationID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}
	return nil
}
func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID, userID string) error {
	if err := s.repo.DeleteNotification(ctx, notificationID, userID); err != nil {
		s.logger.Error("failed to delete notification",
			zap.String("notification_id", notificationID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}
	return nil
}
func (s *NotificationService) ClearNotifications(ctx context.Context,  userID string) error {
	if err := s.repo.ClearNotifications(ctx,  userID); err != nil {
		s.logger.Error("failed to clear user notifications",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	if err := s.repo.MarkAllAsRead(ctx, userID); err != nil {
		s.logger.Error("failed to mark all notifications as read", zap.String("user_id", userID), zap.Error(err))
		return err
	}
	return nil
}
