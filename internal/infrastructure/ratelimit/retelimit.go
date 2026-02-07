package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	domain_errors "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/errors"
)

// Default configuration values for RateLimiter
const (
	DefaultRateLimit      = 10                 // Default number of allowed requests per window
	DefaultRateLimitWindow = time.Minute       // Default window duration for rate limiting
	DefaultLimiterPrefix   = "notification"    // Default prefix for redis keys
)

// RateLimiter does best-practice fixed-window distributed rate limiting using Redis' INCR/EXPIRE.
type RateLimiter struct {
	cache  ports.Cache
	rate   int
	window time.Duration
	prefix string
}

// NewRateLimiter constructs a new limiter instance.
func NewRateLimiter(cache ports.Cache, rate int, window time.Duration, prefix string) *RateLimiter {
	return &RateLimiter{
		cache:  cache,
		rate:   rate,
		window: window,
		prefix: prefix,
	}
}

// DefaultRateLimiter constructs a new RateLimiter instance with default configuration values.
func DefaultRateLimiter(cache ports.Cache) *RateLimiter {
	return &RateLimiter{
		cache:  cache,
		rate:   DefaultRateLimit,
		window: DefaultRateLimitWindow,
		prefix: DefaultLimiterPrefix,
	}
}

// Allow checks if the provided key is within the rate limit, otherwise returns domain.ErrRateLimit.
func (l *RateLimiter) Allow(ctx context.Context, key string) error {
	now := time.Now().UTC()
	windowKey := fmt.Sprintf("%s:rate_limit:%s:%d", l.prefix, key, now.Unix()/int64(l.window.Seconds()))

	// Atomically increment the counter.
	count, err := l.cache.Incr(ctx, windowKey)
	if err != nil {
		return domain_errors.ErrRateLimit // or return custom ErrRedisUnavailable
	}

	if count == 1 {
		// Set key to expire at end of window (best practice).
		_ = l.cache.Expire(ctx, windowKey, l.window+time.Second)
	}
	if count > int64(l.rate) {
		return domain_errors.ErrRateLimit
	}
	return nil
}

// Reset resets the rate limiter state for the key (useful in tests and for user management).
func (l *RateLimiter) Reset(ctx context.Context, key string) error {
	windowPattern := fmt.Sprintf("%s:rate_limit:%s:*", l.prefix, key)
	_, err := l.cache.Del(ctx, windowPattern)
	return err
}

