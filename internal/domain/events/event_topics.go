package domain_events

// KafkaTopic represents a Kafka topic name used by the notification service.
type KafkaTopic string

const (
	// Notification Outbox/Events Topics (
	TopicNotificationEmailSent KafkaTopic = "notification.sent.email.v1"
	TopicNotificationSMSSent   KafkaTopic = "notification.sent.sms.v1"
	TopicNotificationInAppSent KafkaTopic = "notification.sent.in-app.v1"
	TopicNotificationPushSent  KafkaTopic = "notification.sent.push.v1"

	// Notification Request Topics
	TopicNotificationRequestEmail KafkaTopic = "notification.request.email.v1"
	TopicNotificationRequestSMS   KafkaTopic = "notification.request.sms.v1"
	TopicNotificationRequestInApp KafkaTopic = "notification.request.in-app.v1"
	TopicNotificationRequestPush  KafkaTopic = "notification.request.push.v1"

	// Auth Related Notification Request Topics
	TopicNotificationRequestOTP            KafkaTopic = "notification.request.auth.otp.v1"
	TopicNotificationRequestForgotPassword KafkaTopic = "notification.request.auth.forgot-password.v1"
	TopicNotificationRequestAccountCreated KafkaTopic = "notification.request.auth.account-created.v1"

	// OTP Event Topics
	TopicAuthOTPRequested          KafkaTopic = "auth.otp.requested.v1"
	TopicAuthOTPVerified           KafkaTopic = "auth.otp.verified.v1"
	TopicAuthOTPVerificationFailed KafkaTopic = "auth.otp.verification-failed.v1"

	// Password-Related Event Topics
	TopicAuthForgotPasswordRequested KafkaTopic = "auth.forgot-password.requested.v1"
	TopicAuthForgotPasswordSucceeded KafkaTopic = "auth.forgot-password.succeeded.v1"
	TopicAuthForgotPasswordFailed    KafkaTopic = "auth.forgot-password.failed.v1"

	// Account Lifecycle Event Topics
	TopicAuthAccountCreated   KafkaTopic = "auth.account.created.v1"
	TopicAuthAccountActivated KafkaTopic = "auth.account.activated.v1"
	TopicAuthAccountSuspended KafkaTopic = "auth.account.suspended.v1"

	// Generic Notification Event Streams
	TopicNotificationEmailChannel KafkaTopic = "notification.channel.email.v1"
	TopicNotificationSMSChannel   KafkaTopic = "notification.channel.sms.v1"
	TopicNotificationPushChannel  KafkaTopic = "notification.channel.push.v1"
	TopicNotificationInAppChannel KafkaTopic = "notification.channel.in-app.v1"

	// Miscellaneous or catch-all
	TopicNotificationRequestGeneral KafkaTopic = "notification.request.general.v1"
	TopicNotificationEventGeneral   KafkaTopic = "notification.event.general.v1"
)
