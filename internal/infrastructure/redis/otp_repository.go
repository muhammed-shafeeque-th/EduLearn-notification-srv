package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
	repository "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/repositories"
	"go.uber.org/zap"
)

// OTPRepository implementation
type OTPRepository struct {
	cache  ports.Cache
	logger *zap.Logger
}

func NewOTPRepository(cache ports.Cache, logger *zap.Logger) repository.OTPRepository {
	return &OTPRepository{
		cache:  cache,
		logger: logger,
	}
}

func (r *OTPRepository) SaveOTP(ctx context.Context, otp *entity.OTP) error {
	key := fmt.Sprintf("otp:%s", otp.Email)
	ttl := time.Until(otp.ExpiresAt)
	
	if ttl <= 0 {
		return fmt.Errorf("OTP already expired")
	}

	if err := r.cache.Set(ctx, key, otp, ttl); err != nil {
		r.logger.Error("Failed to save OTP",
			zap.String("email", otp.Email),
			zap.Error(err))
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	return nil
}

func (r *OTPRepository) GetOTP(ctx context.Context, email string) (*entity.OTP, error) {
	key := fmt.Sprintf("otp:%s", email)
	var otp entity.OTP

	if err := r.cache.Get(ctx, key, &otp); err != nil {
		return nil, domain_errors.ErrOTPNotFound
	}

	return &otp, nil
}

func (r *OTPRepository) DeleteOTP(ctx context.Context, email string) error {
	key := fmt.Sprintf("otp:%s", email)
	return r.cache.Delete(ctx, key)
}