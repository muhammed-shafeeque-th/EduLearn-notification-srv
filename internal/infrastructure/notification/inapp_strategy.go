package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	ws "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
	"go.uber.org/zap"
)

// InAppStrategy handles in-app notifications with WebSocket integration
type InAppStrategy struct {
	repo   repository.NotificationRepository
	hub    *ws.Hub
	logger *zap.Logger
}

// NewInAppStrategy creates a new in-app notification strategy
func NewInAppStrategy(
	repo repository.NotificationRepository,
	hub *ws.Hub,
	logger *zap.Logger,
) ports.NotificationSender {
	return &InAppStrategy{
		repo:   repo,
		hub:    hub,
		logger: logger,
	}
}

// Send processes in-app notification and broadcasts via WebSocket
func (s *InAppStrategy) Send(ctx context.Context, n *entity.Notification) error {
	// Ensure required fields
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	if n.Type == "" {
		n.Type = entity.NotificationType(entity.InAppNotification)
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	n.UpdatedAt = time.Now().UTC()

	// Validate notification
	if err := n.Validate(); err != nil {
		s.logger.Error("Invalid in-app notification",
			zap.Error(err))
		return fmt.Errorf("invalid notification: %w", err)
	}

	// Save to database
	if err := s.repo.SaveNotification(ctx, n); err != nil {
		s.logger.Error("Failed to save in-app notification",
			zap.String("notification_id", n.ID),
			zap.String("user_id", n.UserId),
			zap.Error(err))
		return fmt.Errorf("failed to save notification: %w", err)
	}

	// Broadcast via WebSocket
	wsMessage := s.buildWebSocketMessage(*n)
	if err := s.hub.NotifyInAppMessage(wsMessage); err != nil {
		s.logger.Warn("Failed to broadcast WebSocket message",
			zap.String("notification_id", n.ID),
			zap.String("user_id", n.UserId),
			zap.Error(err))
		// Don't fail - notification is already saved
	}

	s.logger.Info("In-app notification sent and broadcast",
		zap.String("notification_id", n.ID),
		zap.String("user_id", n.UserId),
		zap.String("subject", n.Subject))

	return nil
}

// buildWebSocketMessage converts entity notification to WebSocket message
func (s *InAppStrategy) buildWebSocketMessage(n entity.Notification) *entity.InAppWSMessage {
	return &entity.InAppWSMessage{
		Type:             "notification",
		ID:               n.ID,
		UserID:           n.UserId,
		Subject:          n.Subject,
		Body:             n.Body,
		Recipient:        n.Recipient,
		IsRead:           n.IsRead,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
		Priority:         s.determinePriority(n),
		NotificationType: string(n.Type),
		// Metadata:         s.buildMetadata(n),
	}
}

// determinePriority determines notification priority based on content
func (s *InAppStrategy) determinePriority(n entity.Notification) string {
	
	// High priority keywords
	highPriorityKeywords := []string{"urgent", "important", "critical", "action required"}
	subjectLower := toLower(n.Subject)
	bodyLower := toLower(n.Body)
	
	for _, keyword := range highPriorityKeywords {
		if contains(subjectLower, keyword) || contains(bodyLower, keyword) {
			return "high"
		}
	}
	
	return "normal"
}

// buildMetadata creates metadata for WebSocket message
func (s *InAppStrategy) buildMetadata(n *entity.Notification) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	// Add timestamp info
	metadata["created_timestamp"] = n.CreatedAt.Unix()
	metadata["time_ago"] = s.getTimeAgo(n.CreatedAt)
	
	// Add notification type
	metadata["category"] = s.categorizeNotification(n)
	
	// Add action hints
	if actionURL := s.extractActionURL(n); actionURL != "" {
		metadata["action_url"] = actionURL
	}
	
	return metadata
}

// categorizeNotification categorizes notification for UI display
func (s *InAppStrategy) categorizeNotification(n *entity.Notification) string {
	subject := toLower(n.Subject)
	body := toLower(n.Body)
	
	categories := map[string][]string{
		"system":  {"system", "update", "maintenance"},
		"account": {"account", "profile", "password", "security"},
		"course":  {"course", "lesson", "assignment", "grade"},
		"social":  {"comment", "like", "mention", "follow"},
		"payment": {"payment", "subscription", "invoice", "billing"},
	}
	
	for category, keywords := range categories {
		for _, keyword := range keywords {
			if contains(subject, keyword) || contains(body, keyword) {
				return category
			}
		}
	}
	
	return "general"
}

// extractActionURL extracts action URL from notification body
func (s *InAppStrategy) extractActionURL(n *entity.Notification) string {
	
	return ""
}

// getTimeAgo returns human-readable time difference
func (s *InAppStrategy) getTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
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

// Helper functions
func toLower(s string) string {
	return strings.ToLower(s)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}