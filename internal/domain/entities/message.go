package entity

type InAppWSMessage struct {
	Type             string            `json:"type"` 
	ID               string            `json:"id"`
	UserID           string            `json:"userId"`
	Subject          string            `json:"subject,omitempty"`
	Body             string            `json:"body,omitempty"`
	Recipient        string            `json:"recipient,omitempty"`
	IsRead           bool              `json:"isRead"`
	CreatedAt        string            `json:"createdAt"`
	Priority         string            `json:"priority,omitempty"` // low, medium, high
	ActionURL        string            `json:"actionUrl,omitempty"`
	NotificationType string            `json:"notificationType,omitempty"` // course, assignment, etc.
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type PingMessage struct {
	Type      string `json:"type"` // "ping"
	Timestamp string `json:"timestamp"`
}

type PongMessage struct {
	Type      string `json:"type"` // "pong"
	Timestamp string `json:"timestamp"`
}

type ErrorMessage struct {
	Type    string `json:"type"` // "error"
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type ConnectionMessage struct {
	Type      string `json:"type"` // "connected"
	Message   string `json:"message"`
	UserID    string `json:"userId"`
	Timestamp string `json:"timestamp"`
}
