package database

import (
	"context"
	"fmt"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/database/gorm/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

// NotificationRepository implements domain.NotificationRepository
type NotificationRepository struct {
	db     *gorm.DB
	cache  ports.Cache
	logger *zap.Logger
}

func NewNotificationRepository(db *gorm.DB, cache ports.Cache, logger *zap.Logger) repository.NotificationRepository {
	return &NotificationRepository{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

func (r *NotificationRepository) SaveNotification(ctx context.Context, notification *entity.Notification) error {
	if err := notification.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Set timestamps
	now := time.Now().UTC()
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = now
	}
	notification.UpdatedAt = now

	model := r.mapToEntityModel(notification)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		r.logger.Error("Failed to save notification",
			zap.String("notification_id", notification.ID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save notification: %w", err)
	}

	if err := r.invalidateUserNotificationCaches(ctx, notification.UserId); err != nil {
		r.logger.Warn("Failed to invalidate caches after SaveNotification",
			zap.String("user_id", notification.UserId),
			zap.Error(err),
		)
	}
	return nil
}

func (r *NotificationRepository) GetNotification(ctx context.Context, id, userID string) (*entity.Notification, error) {
	cacheKey := fmt.Sprintf("notification:%s", id)
	var cached entity.Notification

	if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
		if cached.UserId == userID {
			return &cached, nil
		}
	}

	var model models.NotificationModel
	if err := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain_errors.ErrNotificationNotFound
		}
		r.logger.Error("Failed to get notification",
			zap.String("id", id),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	entityNotification := r.mapToDomainEntity(&model)

	// Cache entity
	if err := r.cache.Set(ctx, cacheKey, *entityNotification, 5*time.Minute); err != nil {
		r.logger.Warn("Failed to cache notification in GetNotification",
			zap.String("cache_key", cacheKey),
			zap.Error(err),
		)
	}

	return entityNotification, nil
}

func (r *NotificationRepository) GetNotifications(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	isUnfilteredFirstPage := filter.Page == 1 && filter.Category == nil && filter.IsRead == nil
	cacheKey := ""
	if isUnfilteredFirstPage {
		cacheKey = fmt.Sprintf("user_notifications:%s:page1", filter.UserID)
		var cached struct {
			Notifications []entity.Notification
			Total         int64
		}
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached.Notifications, cached.Total, nil
		}
	}

	query := r.db.WithContext(ctx).Model(&models.NotificationModel{}).
		Where("user_id = ?", filter.UserID)

	if filter.Category != nil {
		query = query.Where("category = ?", *filter.Category)
	}
	if filter.IsRead != nil {
		query = query.Where("is_read = ?", *filter.IsRead)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		r.logger.Error("Failed counting notifications",
			zap.String("user_id", filter.UserID),
			zap.Error(err),
		)
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	offset := (filter.Page - 1) * filter.PageSize
	var modelsSlice []models.NotificationModel
	if err := query.
		Order("created_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&modelsSlice).Error; err != nil {
		r.logger.Error("Failed to fetch notifications",
			zap.String("user_id", filter.UserID),
			zap.Error(err),
		)
		return nil, 0, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	notifications := make([]entity.Notification, 0, len(modelsSlice))
	for _, m := range modelsSlice {
		n := r.mapToDomainEntity(&m)
		if n != nil {
			notifications = append(notifications, *n)
		}
	}

	if isUnfilteredFirstPage {
		cached := struct {
			Notifications []entity.Notification
			Total         int64
		}{
			Notifications: notifications,
			Total:         total,
		}
		if err := r.cache.Set(ctx, cacheKey, cached, 2*time.Minute); err != nil {
			r.logger.Warn("Failed to cache notifications in GetNotifications",
				zap.String("cache_key", cacheKey),
				zap.Error(err),
			)
		}
	}

	return notifications, total, nil
}

// invalidateUserNotificationCaches tries to remove all user notification caches for the user.
// This includes both user_notifications:<userID> and (if present) user_notifications:<userID>:page1.
func (r *NotificationRepository) invalidateUserNotificationCaches(ctx context.Context, userID string) error {
	var allErrs []string
	// Invalidate both possible keys.
	rawKeys := []string{
		fmt.Sprintf("user_notifications:%s", userID),
		fmt.Sprintf("user_notifications:%s:page1", userID),
	}
	for _, key := range rawKeys {
		if err := r.cache.Delete(ctx, key); err != nil {
			allErrs = append(allErrs, fmt.Sprintf("failed to delete cache key %s: %v", key, err))
			r.logger.Warn("Failed to invalidate user notification cache",
				zap.String("cache_key", key),
				zap.Error(err),
			)
		}
	}
	if len(allErrs) > 0 {
		return fmt.Errorf(strings.Join(allErrs, "; "))
	}
	return nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark as read: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain_errors.ErrNotificationNotFound
	}

	if err := r.cache.Delete(ctx, fmt.Sprintf("notification:%s", id)); err != nil {
		r.logger.Warn("Failed to invalidate cache after MarkAsRead",
			zap.String("cache_key", fmt.Sprintf("notification:%s", id)),
			zap.Error(err),
		)
	}

	if err := r.invalidateUserNotificationCaches(ctx, userID); err != nil {
		r.logger.Warn("Failed to invalidate user notifications caches after MarkAsRead",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.NotificationModel{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read":    true,
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to mark all as read: %w", result.Error)
	}

	if err := r.invalidateUserNotificationCaches(ctx, userID); err != nil {
		r.logger.Warn("Failed to invalidate user_notifications caches after MarkAllAsRead",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	r.logger.Info("Marked all notifications as read",
		zap.String("user_id", userID),
		zap.Int64("count", result.RowsAffected))

	return nil
}

// Implementation for deleting a single notification for a user
func (r *NotificationRepository) DeleteNotification(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.NotificationModel{})

	if result.Error != nil {
		r.logger.Error("Failed to delete notification",
			zap.String("notification_id", id),
			zap.String("user_id", userID),
			zap.Error(result.Error),
		)
		return fmt.Errorf("failed to delete notification: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain_errors.ErrNotificationNotFound
	}

	// Invalidate relevant caches
	if err := r.cache.Delete(ctx, fmt.Sprintf("notification:%s", id)); err != nil {
		r.logger.Warn("Failed to invalidate notification cache after DeleteNotification",
			zap.String("cache_key", fmt.Sprintf("notification:%s", id)),
			zap.Error(err),
		)
	}
	if err := r.invalidateUserNotificationCaches(ctx, userID); err != nil {
		r.logger.Warn("Failed to invalidate user notifications caches after DeleteNotification",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}
	return nil
}

// Implementation for clearing all notifications for a user
func (r *NotificationRepository) ClearNotifications(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.NotificationModel{})
	if result.Error != nil {
		r.logger.Error("Failed to clear notifications for user",
			zap.String("user_id", userID),
			zap.Error(result.Error),
		)
		return fmt.Errorf("failed to clear notifications: %w", result.Error)
	}

	// Invalidate relevant caches
	if err := r.invalidateUserNotificationCaches(ctx, userID); err != nil {
		r.logger.Warn("Failed to invalidate user notifications caches after ClearNotifications",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}
	// It is impractical to invalidate all per-notification caches, so only user-level.
	return nil
}

func (r *NotificationRepository) CheckIfProcessed(ctx context.Context, id string) (bool, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("processed:%s", id)
	exists, err := r.cache.Exists(ctx, cacheKey)
	if err == nil && exists {
		return true, nil
	}

	// Check database
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&entity.ProcessedNotification{}).
		Where("notification_id = ?", id).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check if processed: %w", err)
	}

	if count > 0 {
		if err := r.cache.Set(ctx, cacheKey, true, time.Hour); err != nil {
			r.logger.Warn("Failed to cache 'processed' result",
				zap.String("cache_key", cacheKey),
				zap.Error(err),
			)
		}
		return true, nil
	}

	return false, nil
}

func (r *NotificationRepository) MarkAsProcessed(ctx context.Context, id string) error {
	processed := entity.ProcessedNotification{
		NotificationID: id,
		ProcessedAt:    time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(&processed).Error; err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	// Cache for 1 hour
	cacheKey := fmt.Sprintf("processed:%s", id)
	if err := r.cache.Set(ctx, cacheKey, true, time.Hour); err != nil {
		r.logger.Warn("Failed to cache result in MarkAsProcessed",
			zap.String("cache_key", cacheKey),
			zap.Error(err),
		)
	}

	return nil
}

func (r *NotificationRepository) mapToEntityModel(notification *entity.Notification) *models.NotificationModel {
	if notification == nil {
		return nil
	}
	return &models.NotificationModel{
		ID:               notification.ID,
		UserId:           notification.UserId,
		Type:             notification.Type,
		ActionURL:        notification.ActionURL,
		Subject:          notification.Subject,
		Body:             notification.Body,
		Recipient:        notification.Recipient,
		IsRead:           notification.IsRead,
		CreatedAt:        notification.CreatedAt,
		UpdatedAt:        notification.UpdatedAt,
		Priority:         notification.Priority,
		Category:         notification.Category,
		Metadata:         notification.Metadata,
	}
}

func (r *NotificationRepository) mapToDomainEntity(notification *models.NotificationModel) *entity.Notification {
	if notification == nil {
		return nil
	}
	return &entity.Notification{
		ID:               notification.ID,
		UserId:           notification.UserId,
		Type:             notification.Type,
		ActionURL:        notification.ActionURL,
		Subject:          notification.Subject,
		Body:             notification.Body,
		Recipient:        notification.Recipient,
		IsRead:           notification.IsRead,
		CreatedAt:        notification.CreatedAt,
		UpdatedAt:        notification.UpdatedAt,
		Priority:         notification.Priority,
		Category:         notification.Category,
		Metadata:         notification.Metadata,
	}
}
