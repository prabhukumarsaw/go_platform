package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"newsplatform/api/pkg/config"
)

// NewRedisClient creates a configured Redis client.
func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 20,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
