package events

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	domain "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	ws "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
)

type mockRenderer struct {
	renderFn func(templateName string, data map[string]string) (string, error)
}

func (m *mockRenderer) Render(templateName string, data map[string]string) (string, error) {
	if m.renderFn != nil {
		return m.renderFn(templateName, data)
	}
	return "", nil
}

type mockNotificationRepo struct {
	checkIfProcessedFn func(ctx context.Context, id string) (bool, error)
	markAsProcessedFn  func(ctx context.Context, id string) error
	saveNotificationFn func(ctx context.Context, notification *entity.Notification) error
}

func (m *mockNotificationRepo) SaveNotification(ctx context.Context, notification *entity.Notification) error {
	if m.saveNotificationFn != nil {
		return m.saveNotificationFn(ctx, notification)
	}
	return nil
}

func (m *mockNotificationRepo) GetNotification(ctx context.Context, id, userID string) (*entity.Notification, error) {
	return nil, nil
}

func (m *mockNotificationRepo) GetNotifications(ctx context.Context, filter *domain.NotificationFilter) ([]entity.Notification, int64, error) {
	return nil, 0, nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, id, userID string) error { return nil }
func (m *mockNotificationRepo) DeleteNotification(ctx context.Context, id, userID string) error {
	return nil
}
func (m *mockNotificationRepo) ClearNotifications(ctx context.Context, userID string) error {
	return nil
}
func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, userID string) error { return nil }
func (m *mockNotificationRepo) CheckIfProcessed(ctx context.Context, id string) (bool, error) {
	if m.checkIfProcessedFn != nil {
		return m.checkIfProcessedFn(ctx, id)
	}
	return false, nil
}
func (m *mockNotificationRepo) MarkAsProcessed(ctx context.Context, id string) error {
	if m.markAsProcessedFn != nil {
		return m.markAsProcessedFn(ctx, id)
	}
	return nil
}

type mockOTPRepo struct {
	saveOTPFn func(ctx context.Context, otp *entity.OTP) error
}

func (m *mockOTPRepo) SaveOTP(ctx context.Context, otp *entity.OTP) error {
	if m.saveOTPFn != nil {
		return m.saveOTPFn(ctx, otp)
	}
	return nil
}

func (m *mockOTPRepo) GetOTP(ctx context.Context, email string) (*entity.OTP, error) { return nil, nil }
func (m *mockOTPRepo) DeleteOTP(ctx context.Context, email string) error             { return nil }

type mockEmailSender struct {
	sendFn func(ctx context.Context, args ports.NotificationLike) error
}

func (m *mockEmailSender) Send(ctx context.Context, args ports.NotificationLike) error {
	if m.sendFn != nil {
		return m.sendFn(ctx, args)
	}
	return nil
}

type mockMessageBroker struct{}

func (m *mockMessageBroker) PublishOTPVerifiedEvent(ctx context.Context, event *domain_events.OTPVerifiedEvent) error {
	return nil
}
func (m *mockMessageBroker) Publish(ctx context.Context, topic string, message []byte) error {
	return nil
}
func (m *mockMessageBroker) Subscribe(topics string, handler ...ports.MessageHandler) error {
	return nil
}
func (m *mockMessageBroker) StartConsuming(ctx context.Context) error { return nil }
func (m *mockMessageBroker) Close() error                             { return nil }

type mockWSHubAdaptor struct {
	notifyFn func(msg *entity.InAppWSMessage) error
}

func (m *mockWSHubAdaptor) Broadcast(userID string, payload any) {}
func (m *mockWSHubAdaptor) GetMetrics() map[string]any           { return nil }
func (m *mockWSHubAdaptor) NotifyInAppMessage(msg *entity.InAppWSMessage) error {
	if m.notifyFn != nil {
		return m.notifyFn(msg)
	}
	return nil
}
func (m *mockWSHubAdaptor) ServeWS(auth ws.AuthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
func (m *mockWSHubAdaptor) Shutdown(ctx context.Context) error { return nil }

func TestOTPRequestEventHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	marked := false
	saved := false
	sent := false

	repo := &mockNotificationRepo{
		checkIfProcessedFn: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
		markAsProcessedFn: func(ctx context.Context, id string) error {
			marked = true
			return nil
		},
	}

	otpRepo := &mockOTPRepo{
		saveOTPFn: func(ctx context.Context, otp *entity.OTP) error {
			saved = true
			return nil
		},
	}
	renderer := &mockRenderer{
		renderFn: func(templateName string, data map[string]string) (string, error) {
			return "body", nil
		},
	}
	emailer := &mockEmailSender{
		sendFn: func(ctx context.Context, args ports.NotificationLike) error {
			sent = true
			return nil
		},
	}
	svc := NewOTPRequestEventHandler(renderer, repo, otpRepo, &mockMessageBroker{}, emailer, logger)

	event := domain_events.OTPRequestEvent{
		BaseEvent: domain_events.BaseEvent{EventID: "evt-1", EventType: "OTPRequestEvent", Timestamp: time.Now().Unix()},
		Payload:   domain_events.OTPRequestPayload{UserID: "user-1", Email: "test@example.com", Username: "alice", OTPChannel: "email", RequestSource: "api"},
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)

	require.NoError(t, svc.Handle(ctx, payload))
	assert.True(t, saved)
	assert.True(t, sent)
	assert.True(t, marked)
}

func TestOTPRequestEventHandler_Handle_InvalidJSON(t *testing.T) {
	logger := zap.NewNop()
	svc := NewOTPRequestEventHandler(nil, nil, nil, nil, nil, logger)
	assert.Error(t, svc.Handle(context.Background(), []byte("not json")))
}

func TestForgotPasswordEventHandler_Handle_Success(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	marked := false
	renderer := &mockRenderer{
		renderFn: func(templateName string, data map[string]string) (string, error) {
			return "body", nil
		},
	}
	repo := &mockNotificationRepo{
		checkIfProcessedFn: func(ctx context.Context, id string) (bool, error) { return false, nil },
		markAsProcessedFn:  func(ctx context.Context, id string) error { marked = true; return nil },
	}
	emailer := &mockEmailSender{
		sendFn: func(ctx context.Context, args ports.NotificationLike) error { return nil },
	}
	svc := NewForgotPasswordEventHandler(renderer, repo, nil, emailer, logger)

	event := domain_events.ForgotPasswordRequestEvent{
		BaseEvent: domain_events.BaseEvent{EventID: "evt-2", EventType: "ForgotPasswordRequestEvent", Timestamp: time.Now().Unix()},
		Payload:   domain_events.ForgotPasswordRequestPayload{UserID: "user-2", Email: "test@example.com", Username: "alice", ResetLink: "https://reset", Expiry: 15},
	}

	payload, err := json.Marshal(event)
	require.NoError(t, err)
	require.NoError(t, svc.Handle(ctx, payload))
	assert.True(t, marked)
}

func TestForgotPasswordEventHandler_Handle_InvalidPayload(t *testing.T) {
	logger := zap.NewNop()
	svc := NewForgotPasswordEventHandler(nil, nil, nil, nil, logger)

	event := domain_events.ForgotPasswordRequestEvent{
		Payload: domain_events.ForgotPasswordRequestPayload{UserID: "", Email: "not-an-email", ResetLink: "bad", Expiry: 0},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	assert.Error(t, svc.Handle(context.Background(), payload))
}

func TestHandleEmailNotificationChannel_Handle_Success(t *testing.T) {
	logger := zap.NewNop()
	marked := false
	repo := &mockNotificationRepo{
		checkIfProcessedFn: func(ctx context.Context, id string) (bool, error) { return false, nil },
		markAsProcessedFn:  func(ctx context.Context, id string) error { marked = true; return nil },
	}
	emailer := &mockEmailSender{
		sendFn: func(ctx context.Context, args ports.NotificationLike) error { return nil },
	}
	svc := NewHandleEmailNotificationChannel(repo, emailer, logger)

	event := domain_events.EmailNotificationEvent{
		BaseEvent: domain_events.BaseEvent{EventID: "evt-3", EventType: "EmailNotificationEvent", Timestamp: time.Now().Unix()},
		Payload:   domain_events.EmailNotificationPayload{To: "test@example.com", Subject: "Hi", Body: "hello", UserID: "user-3"},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	require.NoError(t, svc.Handle(context.Background(), payload))
	assert.True(t, marked)
}

func TestHandleEmailNotificationChannel_Handle_InvalidPayload(t *testing.T) {
	logger := zap.NewNop()
	svc := NewHandleEmailNotificationChannel(nil, nil, logger)

	event := domain_events.EmailNotificationEvent{
		Payload: domain_events.EmailNotificationPayload{To: "", Subject: "", Body: ""},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	assert.Error(t, svc.Handle(context.Background(), payload))
}

func TestInAppNotificationChannelHandler_Handle_HubWarnsButSucceeds(t *testing.T) {
	logger := zap.NewNop()
	marked := false

	repo := &mockNotificationRepo{
		checkIfProcessedFn: func(ctx context.Context, id string) (bool, error) { return false, nil },
		saveNotificationFn: func(ctx context.Context, notification *entity.Notification) error {
			marked = true
			return nil
		},
		markAsProcessedFn: func(ctx context.Context, id string) error { return nil },
	}
	hub := &mockWSHubAdaptor{
		notifyFn: func(msg *entity.InAppWSMessage) error {
			return errors.New("ws down")
		},
	}
	svc := NewInAppNotificationChannelHandler(repo, hub, logger)

	event := domain_events.InAppNotificationEvent{
		BaseEvent: domain_events.BaseEvent{EventID: "evt-4", EventType: "InAppNotificationEvent", Timestamp: time.Now().Unix()},
		Payload:   domain_events.InAppNotificationPayload{UserID: "user-4", Title: "Update", Message: "Important account update", Priority: "high", Category: "account"},
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	require.NoError(t, svc.Handle(context.Background(), payload))
	assert.True(t, marked)
}

func TestBuildEmailNotificationFromEvent(t *testing.T) {
	result := buildEmailNotificationFromEvent(&domain_events.EmailNotificationPayload{To: "test@example.com", Subject: "hello", Body: "body", UserID: "user-1"})
	require.NotNil(t, result)
	assert.Equal(t, "hello", result.Subject)

	assert.Nil(t, buildEmailNotificationFromEvent(&domain_events.EmailNotificationPayload{To: "", Subject: "", Body: ""}))
}

func TestDeterminePriorityAndCategorizeNotification(t *testing.T) {
	notification := &entity.Notification{Subject: "Urgent payment request", Body: "Your invoice is due"}
	assert.Equal(t, "high", determinePriority(notification))
	assert.Equal(t, "payment", categorizeNotification(notification))
}
