package domain_events

import (
	"time"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

type OTPVerifiedEventPayload struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Status    string `json:"status"`
}

type NotificationEventPayload struct {
	ID        string                  `json:"id"`
	UserID    string                  `json:"userId"`
	Type      entity.NotificationType `json:"type"`
	Subject   string                  `json:"subject"`
	Body      string                  `json:"body"`
	Recipient string                  `json:"recipient"`
	Timestamp time.Time               `json:"timestamp"`
}

type OTPVerifiedEvent = Event[OTPVerifiedEventPayload]
type NotificationEvent = Event[NotificationEventPayload]