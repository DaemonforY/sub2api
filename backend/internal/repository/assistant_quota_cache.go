package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// assistantQuotaCache counts the support assistant's questions per day (keys "assistant:<day>:…").
type assistantQuotaCache struct {
	rdb *redis.Client
}

func NewAssistantQuotaCache(rdb *redis.Client) service.AssistantQuotaCache {
	return &assistantQuotaCache{rdb: rdb}
}

func (c *assistantQuotaCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("assistant quota increment: %w", err)
	}
	return incr.Val(), nil
}

func (c *assistantQuotaCache) Decr(ctx context.Context, key string) error {
	return c.rdb.Decr(ctx, key).Err()
}

func (c *assistantQuotaCache) Get(ctx context.Context, key string) (int64, error) {
	n, err := c.rdb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return n, err
}
