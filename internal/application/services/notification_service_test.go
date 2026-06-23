package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
)

type mockNotificationRepository struct {
	saveNotificationFn   func(ctx context.Context, notification *entity.Notification) error
	getNotificationFn    func(ctx context.Context, id, userID string) (*entity.Notification, error)
	getNotificationsFn   func(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error)
	markAsReadFn         func(ctx context.Context, id, userID string) error
	deleteNotificationFn func(ctx context.Context, id, userID string) error
	clearNotificationsFn func(ctx context.Context, userID string) error
	markAllAsReadFn      func(ctx context.Context, userID string) error
}

func (m *mockNotificationRepository) SaveNotification(ctx context.Context, notification *entity.Notification) error {
	if m.saveNotificationFn != nil {
		return m.saveNotificationFn(ctx, notification)
	}
	return nil
}

func (m *mockNotificationRepository) GetNotification(ctx context.Context, id, userID string) (*entity.Notification, error) {
	if m.getNotificationFn != nil {
		return m.getNotificationFn(ctx, id, userID)
	}
	return nil, nil
}

func (m *mockNotificationRepository) GetNotifications(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error) {
	if m.getNotificationsFn != nil {
		return m.getNotificationsFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockNotificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationRepository) DeleteNotification(ctx context.Context, id, userID string) error {
	if m.deleteNotificationFn != nil {
		return m.deleteNotificationFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationRepository) ClearNotifications(ctx context.Context, userID string) error {
	if m.clearNotificationsFn != nil {
		return m.clearNotificationsFn(ctx, userID)
	}
	return nil
}

func (m *mockNotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	if m.markAllAsReadFn != nil {
		return m.markAllAsReadFn(ctx, userID)
	}
	return nil
}

func (m *mockNotificationRepository) CheckIfProcessed(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (m *mockNotificationRepository) MarkAsProcessed(ctx context.Context, id string) error {
	return nil
}

type mockNotificationSender struct {
	sendFn func(ctx context.Context, notification *entity.Notification) error
}

func (m *mockNotificationSender) Send(ctx context.Context, notification *entity.Notification) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, notification)
	}
	return nil
}

func TestNotificationService_CreateAndQueue(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("success", func(t *testing.T) {
		calledSave := false
		calledSend := false
		repo := &mockNotificationRepository{
			saveNotificationFn: func(ctx context.Context, notification *entity.Notification) error {
				calledSave = true
				return nil
			},
		}
		sender := &mockNotificationSender{
			sendFn: func(ctx context.Context, notification *entity.Notification) error {
				calledSend = true
				return nil
			},
		}

		svc := NewNotificationService(repo, sender, logger)
		n, err := svc.CreateAndQueue(ctx, "user-1", "recipient@test.com", "hello", "world", entity.EmailNotification)

		require.NoError(t, err)
		require.NotNil(t, n)
		assert.Equal(t, "user-1", n.UserId)
		assert.True(t, calledSave)
		assert.True(t, calledSend)
	})

	t.Run("db failure returns domain database error", func(t *testing.T) {
		repo := &mockNotificationRepository{
			saveNotificationFn: func(ctx context.Context, notification *entity.Notification) error {
				return errors.New("db down")
			},
		}
		sender := &mockNotificationSender{
			sendFn: func(ctx context.Context, notification *entity.Notification) error {
				t.Errorf("sender should not be called when save fails")
				return nil
			},
		}

		svc := NewNotificationService(repo, sender, logger)
		n, err := svc.CreateAndQueue(ctx, "user-2", "recipient@test.com", "hello", "world", entity.EmailNotification)

		assert.Nil(t, n)
		assert.ErrorIs(t, err, domain_errors.ErrDatabase)
	})

	t.Run("sender failure propagates error", func(t *testing.T) {
		repo := &mockNotificationRepository{
			saveNotificationFn: func(ctx context.Context, notification *entity.Notification) error {
				return nil
			},
		}
		sender := &mockNotificationSender{
			sendFn: func(ctx context.Context, notification *entity.Notification) error {
				return errors.New("send failed")
			},
		}

		svc := NewNotificationService(repo, sender, logger)
		n, err := svc.CreateAndQueue(ctx, "user-3", "recipient@test.com", "hello", "world", entity.EmailNotification)

		require.Error(t, err)
		assert.NotNil(t, n)
		assert.Equal(t, "send failed", err.Error())
	})
}

func TestNotificationService_GetNotification(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	repo := &mockNotificationRepository{
		getNotificationFn: func(ctx context.Context, id, userID string) (*entity.Notification, error) {
			return &entity.Notification{ID: id, UserId: userID, Recipient: "to@test.com"}, nil
		},
	}

	svc := NewNotificationService(repo, &mockNotificationSender{}, logger)
	n, err := svc.GetNotification(ctx, "notif-1", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "notif-1", n.ID)
	assert.Equal(t, "user-1", n.UserId)
}

func TestNotificationService_ListNotifications(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	var receivedFilter *domain.NotificationFilter
	repo := &mockNotificationRepository{
		getNotificationsFn: func(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error) {
			receivedFilter = filter
			return []entity.Notification{{ID: "x", UserId: filter.UserID}}, 2, nil
		},
	}

	svc := NewNotificationService(repo, &mockNotificationSender{}, logger)
	ns, total, err := svc.ListNotifications(ctx, "user-1", 1, 20, nil, nil)

	require.NoError(t, err)
	assert.Len(t, ns, 1)
	assert.Equal(t, int64(2), total)
	assert.Equal(t, "user-1", receivedFilter.UserID)
}

func TestNotificationService_Mutations(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	repo := &mockNotificationRepository{
		markAsReadFn: func(ctx context.Context, id, userID string) error {
			assert.Equal(t, "notify-1", id)
			assert.Equal(t, "user-1", userID)
			return nil
		},
		deleteNotificationFn: func(ctx context.Context, id, userID string) error {
			assert.Equal(t, "notify-2", id)
			assert.Equal(t, "user-1", userID)
			return nil
		},
		clearNotificationsFn: func(ctx context.Context, userID string) error {
			assert.Equal(t, "user-1", userID)
			return nil
		},
		markAllAsReadFn: func(ctx context.Context, userID string) error {
			assert.Equal(t, "user-1", userID)
			return nil
		},
	}

	svc := NewNotificationService(repo, &mockNotificationSender{}, logger)

	require.NoError(t, svc.MarkAsRead(ctx, "notify-1", "user-1"))
	require.NoError(t, svc.DeleteNotification(ctx, "notify-2", "user-1"))
	require.NoError(t, svc.ClearNotifications(ctx, "user-1"))
	require.NoError(t, svc.MarkAllAsRead(ctx, "user-1"))
}
