package repository

import (
	"context"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
)

type OTPRepository interface {
	SaveOTP(ctx context.Context, otp *entity.OTP) error
	GetOTP(ctx context.Context, email string) (*entity.OTP, error)
	DeleteOTP(ctx context.Context, email string) error
}
