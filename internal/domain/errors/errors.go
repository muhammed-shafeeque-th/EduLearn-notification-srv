package domain_errors

import "errors"

var (
	ErrOTPNotFound      = errors.New("otp not found")
	ErrOTPExpired       = errors.New("otp has expired")
	ErrRateLimit        = errors.New("rate limit exceeded")
	ErrDatabase         = errors.New("database error")
	ErrKafkaProduce     = errors.New("failed to produce Kafka message")
	ErrEmailSend        = errors.New("failed to send email")
	ErrAlreadyProcessed = errors.New("notification already processed")
	ErrNotFound         = errors.New("resource not found")
	ErrUnauthorized     = errors.New("unauthorized access to resource")
	ErrInvalidOTP        = errors.New("invalid OTP")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidRecipient  = errors.New("invalid recipient")
	ErrInvalidNotificationType = errors.New("invalid notification type")
)
