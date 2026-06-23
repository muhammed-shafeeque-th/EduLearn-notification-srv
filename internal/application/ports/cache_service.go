package ports

// Package interfaces provides contracts for caching functionality in the application.
import (
	"context"
	"time"
)

type Cache interface {
	// Set stores a value with a given key and optional expiration.
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Get retrieves a value by key, returning ErrCacheMiss if not found.
	Get(ctx context.Context, key string, dest interface{}) error

	// Delete removes a value from the cache.
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)
	
	// Expire updates the TTL for a key.
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// Invalidate clears all cached values (use with care).
	Invalidate(ctx context.Context) error
	
	// Ping checks if the cache backend is reachable.
	Ping(ctx context.Context) error
	
	// Del removes arbitrary number of values from the cache.
	Del(ctx context.Context, keys ...string) (int64, error)

	Incr(ctx context.Context, key string) (int64, error)
}
