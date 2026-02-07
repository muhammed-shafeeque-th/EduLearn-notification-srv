package entity

import (
	"errors"
	"time"

	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
)

type NotificationType string
type NotificationCategory string

const (
	EmailNotification NotificationType = "email"
	InAppNotification NotificationType = "inapp"
	SMSNotification   NotificationType = "sms"
	PushNotification  NotificationType = "push"
)

const (
	CourseCategory      NotificationCategory = "course"
	AssignmentCategory  NotificationCategory = "assignment"
	AchievementCategory NotificationCategory = "achievement"
	MessageCategory     NotificationCategory = "message"
	SystemCategory      NotificationCategory = "system"
)

// MapToCategory validates and maps a string to a known NotificationCategory constant.
func  MapToCategory(category string) NotificationCategory {
	switch category {
	case string(CourseCategory),
		string(AssignmentCategory),
		string(AchievementCategory),
		string(MessageCategory),
		string(SystemCategory):
		return NotificationCategory(category)
	default:
		return MessageCategory
	}
}
func (nt NotificationType) IsValid() bool {
	switch nt {
	case EmailNotification, SMSNotification, PushNotification, InAppNotification:
		return true
	}
	return false
}

type Notification struct {
	ID        string               `json:"id"`
	UserId    string               `json:"userId"`
	Type      NotificationType     `json:"type"`
	Subject   string               `json:"subject"`
	Body      string               `json:"body"`
	Recipient string               `json:"recipient"`
	IsRead    bool                 `json:"isRead"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
	Priority  string               `json:"priority,omitempty"`
	ActionURL string               `json:"actionUrl,omitempty"`
	Category  NotificationCategory `json:"category,omitempty"`
	Metadata  map[string]string    `json:"metadata,omitempty"`
}

// Getters

func (n *Notification) GetID() string {
	return n.ID
}

func (n *Notification) GetUserId() string {
	return n.UserId
}

func (n *Notification) GetType() NotificationType {
	return n.Type
}

func (n *Notification) GetSubject() string {
	return n.Subject
}

func (n *Notification) GetBody() string {
	return n.Body
}

func (n *Notification) GetRecipient() string {
	return n.Recipient
}

func (n *Notification) GetIsRead() bool {
	return n.IsRead
}

func (n *Notification) GetCreatedAt() time.Time {
	return n.CreatedAt
}

func (n *Notification) GetUpdatedAt() time.Time {
	return n.UpdatedAt
}

func (n *Notification) GetPriority() string {
	return n.Priority
}

func (n *Notification) GetActionURL() string {
	return n.ActionURL
}

func (n *Notification) GetCategory() NotificationCategory {
	return n.Category
}

// Setters

func (n *Notification) SetID(id string) {
	n.ID = id
}

func (n *Notification) SetUserId(userId string) {
	n.UserId = userId
}

func (n *Notification) SetNotificationType(nt NotificationType) {
	n.Type = nt
}

func (n *Notification) SetSubject(subject string) {
	n.Subject = subject
}

func (n *Notification) SetBody(body string) {
	n.Body = body
}

func (n *Notification) SetRecipient(recipient string) {
	n.Recipient = recipient
}

func (n *Notification) SetIsRead(read bool) {
	n.IsRead = read
}

func (n *Notification) SetCreatedAt(createdAt time.Time) {
	n.CreatedAt = createdAt
}

func (n *Notification) SetUpdatedAt(updatedAt time.Time) {
	n.UpdatedAt = updatedAt
}

func (n *Notification) SetPriority(priority string) {
	n.Priority = priority
}

func (n *Notification) SetActionURL(actionURL string) {
	n.ActionURL = actionURL
}

func (n *Notification) SetCategory(category string) {
	n.Category = MapToCategory(category)
}

func (n *Notification) Validate() error {
	if n.UserId == "" {
		return errors.New("userId is required")
	}
	if n.Recipient == "" {
		return domain_errors.ErrInvalidRecipient
	}
	if !n.Type.IsValid() {
		return domain_errors.ErrInvalidNotificationType
	}
	if n.Subject == "" && n.Body == "" {
		return errors.New("either subject or body is required")
	}
	return nil
}

// MarkRead marks notification as read
func (n *Notification) MarkRead() {
	n.IsRead = true
	n.UpdatedAt = time.Now().UTC()
}

type ProcessedNotification struct {
	NotificationID string
	ProcessedAt    time.Time
}
