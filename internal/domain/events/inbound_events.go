package domain_events

import (
	"errors"
	"net/mail"
	"strings"
)

// BaseEvent provides the shared event fields.
type BaseEvent struct {
	EventID       string `json:"eventId"`
	EventType     string `json:"eventType"`
	Timestamp     int64  `json:"timestamp"`
	Source        string `json:"source,omitempty"`       // system/user
	EventVersion  string `json:"eventVersion,omitempty"` // e.g. "0.0.1"
	CorrelationId string `json:"correlationId,omitempty"`
}

// Generic Event wrapper with payload.
// T is the payload type: all domain event payloads are carried via this wrapper.
type Event[T any] struct {
	BaseEvent
	Payload T `json:"payload"`
}

// OTPRequestPayload holds the OTP request information.
type OTPRequestPayload struct {
	UserID        string `json:"userId"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	PhoneNumber   string `json:"phoneNumber,omitempty"`
	OTPChannel    string `json:"otpChannel"` // email, sms, inapp
	AppID         string `json:"appId,omitempty"`
	RequestSource string `json:"requestSource"` // web, mobile, api
	IP            string `json:"ip,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
}

// Validate checks the OTPRequestPayload for correctness.
func (p *OTPRequestPayload) Validate() error {
	if p.UserID == "" || p.Email == "" || p.Username == "" {
		return errors.New("userId, username, and email must not be empty")
	}
	if _, err := mail.ParseAddress(p.Email); err != nil {
		return errors.New("invalid email format")
	}
	if p.OTPChannel == "" {
		return errors.New("otpChannel is required")
	}
	return nil
}

// ForgotPasswordRequestPayload holds forgot password request information.
type ForgotPasswordRequestPayload struct {
	UserID        string `json:"userId"`
	Email         string `json:"email"`
	Username      string `json:"username"`
	ResetLink     string `json:"resetLink"`
	Expiry        int    `json:"expiryMinutes"`           // e.g. 15
	RequestSource string `json:"requestSource,omitempty"` // web, mobile, api
	IP            string `json:"ip,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
}

// Validate checks the ForgotPasswordRequestPayload for correctness.
func (p *ForgotPasswordRequestPayload) Validate() error {
	if p.UserID == "" || p.Email == "" || p.ResetLink == "" {
		return errors.New("userId, email, and resetLink must not be empty")
	}
	if _, err := mail.ParseAddress(p.Email); err != nil {
		return errors.New("invalid email format")
	}
	if p.Expiry <= 0 {
		return errors.New("expiryMinutes must be positive")
	}
	if !strings.HasPrefix(p.ResetLink, "http") {
		return errors.New("resetLink must be a valid URL")
	}
	return nil
}

// InAppNotificationPayload holds in-app notification fields.
type InAppNotificationPayload struct {
	UserID    string `json:"userId"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	ActionUrl string `json:"actionUrl,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Priority  string `json:"priority,omitempty"` // low, normal, high
	AppID     string `json:"appId,omitempty"`
	Category  string `json:"category,omitempty"`
}

// Validate checks the InAppNotificationPayload for correctness.
func (p *InAppNotificationPayload) Validate() error {
	if p.UserID == "" || p.Title == "" || p.Message == "" {
		return errors.New("userId, title, and message must not be empty")
	}
	return nil
}

// EmailNotificationPayload holds arbitrary email notification fields.
type EmailNotificationPayload struct {
	To          string   `json:"to"`
	CC          []string `json:"cc,omitempty"`
	BCC         []string `json:"bcc,omitempty"`
	Subject     string   `json:"subject"`
	UserID      string   `json:"userId"`
	Body        string   `json:"body"`
	Template    string   `json:"template,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
	Priority    string   `json:"priority,omitempty"`
}

// Validate checks the EmailNotificationPayload for correctness.
func (p *EmailNotificationPayload) Validate() error {
	if p.To == "" || p.Subject == "" || p.Body == "" {
		return errors.New("to, subject, and body must not be empty")
	}
	if _, err := mail.ParseAddress(p.To); err != nil {
		return errors.New("invalid 'to' email format")
	}
	for _, c := range p.CC {
		if c != "" {
			if _, err := mail.ParseAddress(c); err != nil {
				return errors.New("invalid 'cc' email format")
			}
		}
	}
	for _, b := range p.BCC {
		if b != "" {
			if _, err := mail.ParseAddress(b); err != nil {
				return errors.New("invalid 'bcc' email format")
			}
		}
	}
	return nil
}

// PushNotificationPayload holds push notification fields for mobile/web/app.
type PushNotificationPayload struct {
	UserID      string            `json:"userId"`
	Title       string            `json:"title"`
	Message     string            `json:"message"`
	Data        map[string]string `json:"data,omitempty"`
	Priority    string            `json:"priority,omitempty"`
	DeviceToken string            `json:"deviceToken"`
	Platform    string            `json:"platform"` // ios, android, web, etc
}

// Validate checks the PushNotificationPayload for correctness.
func (p *PushNotificationPayload) Validate() error {
	if p.UserID == "" || p.Title == "" || p.Message == "" || p.DeviceToken == "" || p.Platform == "" {
		return errors.New("userId, title, message, deviceToken, and platform must not be empty")
	}
	return nil
}

// Type aliases for strongly typed event instances.
type OTPRequestEvent = Event[OTPRequestPayload]
type ForgotPasswordRequestEvent = Event[ForgotPasswordRequestPayload]
type InAppNotificationEvent = Event[InAppNotificationPayload]
type EmailNotificationEvent = Event[EmailNotificationPayload]
type PushNotificationEvent = Event[PushNotificationPayload]
