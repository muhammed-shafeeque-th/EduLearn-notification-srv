package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_events "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/events"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"go.uber.org/zap"
)

// InAppNotificationChannelHandler handles in-app notifications, including persistence and broadcasting.
type InAppNotificationChannelHandler struct {
	notificationRepo repository.NotificationRepository
	hub              ports.WsHubAdaptor
	logger           *zap.Logger
}

// NewInAppNotificationChannelHandler initializes a new InAppNotificationChannelHandler.
func NewInAppNotificationChannelHandler(
	notificationRepo repository.NotificationRepository,
	hub ports.WsHubAdaptor,
	logger *zap.Logger,
) *InAppNotificationChannelHandler {
	return &InAppNotificationChannelHandler{
		notificationRepo: notificationRepo,
		hub:              hub,
		logger:           logger,
	}
}

// Handle processes an InAppNotificationEvent, saves to database, and broadcasts via WebSocket.
func (h *InAppNotificationChannelHandler) Handle(ctx context.Context, message []byte) error {
	var event domain_events.InAppNotificationEvent
	if err := json.Unmarshal(message, &event); err != nil {
		h.logger.Error("Failed to unmarshal InAppNotificationEvent", zap.Error(err))
		return fmt.Errorf("invalid InAppNotificationEvent JSON: %w", err)
	}

	// Validate incoming event
	if err := event.Validate(); err != nil {
		h.logger.Error("Invalid InAppNotificationEvent", zap.Error(err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Idempotency check
	if h.notificationRepo != nil && event.EventID != "" {
		alreadyProcessed, err := h.notificationRepo.CheckIfProcessed(ctx, event.EventID)
		if err != nil {
			h.logger.Error("Failed idempotency check for InApp notification", zap.Error(err))
			return fmt.Errorf("idempotency check failed: %w", err)
		}
		if alreadyProcessed {
			h.logger.Info("In-App notification event already processed, skipping.",
				zap.String("notification_id", event.EventID),
			)
			return nil
		}
	}

	// Compose the notification entity from the event
	notification := &entity.Notification{
		ID:        event.EventID, // Prefer explicit IDs for idempotency
		UserId:    event.UserID,
		Type:      entity.InAppNotification,
		Subject:   event.Title,
		Body:      event.Message,
		Recipient: event.UserID, // In-app recipient is typically the user ID
		IsRead:    false,
		CreatedAt: time.Now(), // Use server time for consistency
		Priority:  event.Priority,
		ActionURL: event.ActionUrl,
		Category:  entity.MapToCategory(event.Category),
		// Metadata is filled when building WebSocket message
	}

	// Save the notification to the database
	if h.notificationRepo != nil {
		if err := h.notificationRepo.SaveNotification(ctx, notification); err != nil {
			h.logger.Error("Failed to save in-app notification",
				zap.String("notification_id", notification.ID),
				zap.String("user_id", notification.UserId),
				zap.Error(err))
			return fmt.Errorf("failed to save notification: %w", err)
		}
	}

	// Build the WebSocket message and broadcast
	wsMessage := buildWebSocketMessage(notification)
	if err := h.hub.NotifyInAppMessage(wsMessage); err != nil {
		h.logger.Warn("Failed to broadcast WebSocket message",
			zap.String("notification_id", notification.ID),
			zap.String("user_id", notification.UserId),
			zap.Error(err))
		// Notification already saved; do not return error
	}

	h.logger.Info("In-app notification sent and broadcast",
		zap.String("notification_id", notification.ID),
		zap.String("user_id", notification.UserId),
		zap.String("subject", notification.Subject))

	return nil
}

// buildWebSocketMessage converts a notification entity into a WebSocket message format.
func buildWebSocketMessage(n *entity.Notification) *entity.InAppWSMessage {
	return &entity.InAppWSMessage{
		Type:             "notification",
		ID:               n.ID,
		UserID:           n.UserId,
		Subject:          n.Subject,
		Body:             n.Body,
		Recipient:        n.Recipient,
		IsRead:           n.IsRead,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
		Priority:         determinePriority(n),
		NotificationType: string(n.Type),
		Metadata:         buildMetadata(n),
	}
}

// determinePriority infers notification priority, falling back to "normal".
func determinePriority(n *entity.Notification) string {
	if n.Priority != "" {
		return strings.ToLower(n.Priority)
	}
	// Default inference based on high-priority keywords
	highPriorityKeywords := []string{"urgent", "important", "critical", "action required"}
	subjectLower := strings.ToLower(n.Subject)
	bodyLower := strings.ToLower(n.Body)
	for _, kw := range highPriorityKeywords {
		if strings.Contains(subjectLower, kw) || strings.Contains(bodyLower, kw) {
			return "high"
		}
	}
	return "normal"
}



// buildMetadata constructs the metadata map for a WebSocket message.
func buildMetadata(n *entity.Notification) map[string]string {
	metadata := make(map[string]string)
	metadata["created_timestamp"] = n.CreatedAt.Local().String()
	metadata["time_ago"] = getTimeAgo(n.CreatedAt)
	metadata["category"] = categorizeNotification(n)

	if actionURL := extractActionURL(n); actionURL != "" {
		metadata["action_url"] = actionURL
	}
	return metadata
}

// categorizeNotification finds a display category for a notification.
func categorizeNotification(n *entity.Notification) string {
	subject := strings.ToLower(n.Subject)
	body := strings.ToLower(n.Body)
	categories := map[string][]string{
		"system":  {"system", "update", "maintenance"},
		"account": {"account", "profile", "password", "security"},
		"course":  {"course", "lesson", "assignment", "grade"},
		"social":  {"comment", "like", "mention", "follow"},
		"payment": {"payment", "subscription", "invoice", "billing"},
	}
	for category, keywords := range categories {
		for _, kw := range keywords {
			if strings.Contains(subject, kw) || strings.Contains(body, kw) {
				return category
			}
		}
	}
	return "general"
}

// extractActionURL returns the action URL from either a direct field or by extracting from text.
func extractActionURL(n *entity.Notification) string {
	if n.ActionURL != "" {
		return n.ActionURL
	}
	// [OPTIONAL] expand with regex extraction from n.Body if needed for your use case.
	return ""
}

// getTimeAgo computes a concise human-readable time difference.
func getTimeAgo(created time.Time) string {
	duration := time.Since(created)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
}
