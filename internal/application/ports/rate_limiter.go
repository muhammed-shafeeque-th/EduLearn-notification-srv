package ports

import "context"

// RateLimiter interface for dependency injection
type RateLimiter interface {
	Allow(ctx context.Context, key string) error
}