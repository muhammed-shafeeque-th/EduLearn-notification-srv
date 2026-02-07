package domain

import entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"

type NotificationFilter struct {
	UserID   string
	Category     *entity.NotificationCategory
	IsRead   *bool
	Page     int
	PageSize int
}