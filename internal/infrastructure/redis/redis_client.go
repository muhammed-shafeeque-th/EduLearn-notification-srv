package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const defaultExpiration = 10 * time.Minute

// ErrorCacheMiss is returned when a requested item is not found in the Redis cache.
var ErrorCacheMiss = errors.New("the requested item was not found in the Redis cache (cache miss)")


// RedisCache implements the interfaces.Cache interface.
type RedisCache struct {
	client     *redis.Client
	logger     *zap.Logger
	keyPrefix  string
}

func NewRedisCache(addr string, logger *zap.Logger, prefix string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     50,   // optimized for high throughput
		MinIdleConns: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis connection failed", zap.Error(err))
		return nil, err
	}

	logger.Info("Redis connected successfully")

	return &RedisCache{
		client:    client,
		logger:    logger,
		keyPrefix: prefix,
	}, nil
}

// buildKey adds namespace prefix to avoid collisions
func (r *RedisCache) buildKey(key string) string {
	return fmt.Sprintf("%s:%s", r.keyPrefix, key)
}


func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := r.buildKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Error("cache:Set marshal error", zap.Error(err))
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	// If expiration is zero, use a default (e.g., 10 minutes)
	exp := expiration
	if exp == 0 {
		exp = defaultExpiration
	}

	err = r.client.Set(ctx, fullKey, data, exp).Err()
	if err != nil {
		r.logger.Error("cache:Set redis error", zap.Error(err))
		return err
	}

	return nil
}


func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	fullKey := r.buildKey(key)
	result, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("cache:Incr redis error", zap.Error(err))
		return 0, err
	}
	return result, nil
}

func (r *RedisCache) Del(ctx context.Context, keys ...string) (int64, error) {
	var fullKeys []string
	for _, key := range keys {
		fullKeys = append(fullKeys, r.buildKey(key))
	}
	n, err := r.client.Del(ctx, fullKeys...).Result()
	if err != nil {
		r.logger.Error("cache:Del redis error", zap.Error(err))
		return 0, err
	}
	return n, nil
}

func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.buildKey(key)

	data, err := r.client.Get(ctx, fullKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrorCacheMiss
	}
	if err != nil {
		r.logger.Error("cache:Get redis error", zap.Error(err))
		return err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		r.logger.Error("cache:Get unmarshal error", zap.Error(err))
		return fmt.Errorf("failed to decode cached value: %w", err)
	}

	return nil
}


func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.buildKey(key)
	return r.client.Del(ctx, fullKey).Err()
}


func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.buildKey(key)

	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		r.logger.Error("cache:Exists redis error", zap.Error(err))
		return false, err
	}

	return count == 1, nil
}


func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := r.buildKey(key)

	err := r.client.Expire(ctx, fullKey, expiration).Err()
	if err != nil {
		r.logger.Error("cache:Expire redis error", zap.Error(err))
	}
	return err
}


func (r *RedisCache) Invalidate(ctx context.Context) error {
	// safer than "FLUSHALL" (dangerous)
	pattern := r.keyPrefix + ":*"

	iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			r.logger.Error("cache:Invalidate delete error", zap.Error(err))
			return err
		}
	}

	if err := iter.Err(); err != nil {
		r.logger.Error("cache:Invalidate iterator error", zap.Error(err))
		return err
	}

	return nil
}


func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
