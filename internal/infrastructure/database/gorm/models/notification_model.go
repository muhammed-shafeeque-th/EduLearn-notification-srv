package models

import (
	"time"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

type NotificationModel struct {
	ID        string                      `gorm:"primaryKey;type:varchar(64);column:id"`                                       // id
	UserId    string                      `gorm:"type:varchar(64);index:idx_user_created,priority:1;not null;column:user_id"`  // user_id
	Type      entity.NotificationType     `gorm:"type:varchar(16);index:idx_type;not null;column:type"`                        // type
	Subject   string                      `gorm:"type:text;not null;column:subject"`                                           // subject
	Body      string                      `gorm:"type:text;not null;column:body"`                                              // body
	Recipient string                      `gorm:"type:varchar(128);index:idx_recipient;not null;column:recipient"`             // recipient
	IsRead    bool                        `gorm:"type:boolean;default:false;index:idx_user_read;column:is_read"`               // is_read
	CreatedAt time.Time                   `gorm:"type:timestamp;index:idx_user_created,priority:2;not null;column:created_at"` // created_at
	UpdatedAt time.Time                   `gorm:"type:timestamp;autoUpdateTime;not null;column:updated_at"`                    // updated_at
	Priority  string                      `gorm:"type:varchar(32);default:null;column:priority"`                               // priority
	ActionURL string                      `gorm:"type:text;default:null;column:action_url"`                                    // action_url
	Category  entity.NotificationCategory `gorm:"type:varchar(32);default:null;column:category"`                               // notification_type
	Metadata  map[string]string           `gorm:"-;"`                                                                          // metadata (not stored in DB)
}

func (NotificationModel) TableName() string {
	return "notifications"
}
