package models

import "time"

type ProcessedNotificationModel struct {
	NotificationID string    `gorm:"type:uuid;primaryKey"`
	ProcessedAt    time.Time `gorm:"autoCreateTime"`
}

func (ProcessedNotificationModel) TableName() string {
	return "processed_notifications"
}
