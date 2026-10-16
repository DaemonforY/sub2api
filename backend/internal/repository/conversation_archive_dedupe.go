package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// redisArchiveDedupeStore keeps 对话归档's per-session dedupe state in Redis (JSON, TTL refreshed
// on every write).
type redisArchiveDedupeStore struct {
	rdb *redis.Client
}

// NewArchiveDedupeStore provides the Redis-backed dedupe state for 对话归档.
func NewArchiveDedupeStore(rdb *redis.Client) service.ArchiveDedupeStore {
	return &redisArchiveDedupeStore{rdb: rdb}
}

func (r *redisArchiveDedupeStore) Load(ctx context.Context, key string) (*service.ArchiveDedupeState, error) {
	if r == nil || r.rdb == nil {
		return nil, errors.New("redis unavailable")
	}
	raw, err := r.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil //nolint:nilnil // an absent state is a valid result
	}
	if err != nil {
		return nil, err
	}
	var st service.ArchiveDedupeState
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, nil //nolint:nilnil // a corrupt state is treated as absent (the next turn is a full one)
	}
	return &st, nil
}

func (r *redisArchiveDedupeStore) Save(ctx context.Context, key string, state *service.ArchiveDedupeState, ttl time.Duration) error {
	if r == nil || r.rdb == nil {
		return errors.New("redis unavailable")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, raw, ttl).Err()
}
