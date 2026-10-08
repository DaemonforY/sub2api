package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// loginGuardIncrScript increments a fixed-window counter, setting the TTL only when
// the key is new (or lost its TTL).
var loginGuardIncrScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 or redis.call('PTTL', KEYS[1]) == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return n
`)

// loginGuardCache backs service.LoginGuardService (keys "login_guard:…", "login_captcha:…").
type loginGuardCache struct {
	rdb *redis.Client
}

func NewLoginGuardCache(rdb *redis.Client) service.LoginGuardCache {
	return &loginGuardCache{rdb: rdb}
}

func (c *loginGuardCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := loginGuardIncrScript.Run(ctx, c.rdb, []string{key}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("login guard increment: %w", err)
	}
	return n, nil
}

func (c *loginGuardCache) Get(ctx context.Context, key string) (int64, error) {
	n, err := c.rdb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return n, err
}

func (c *loginGuardCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	d, err := c.rdb.PTTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if d < 0 { // -2 missing, -1 no expiry (never written that way)
		return 0, nil
	}
	return d, nil
}

func (c *loginGuardCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *loginGuardCache) Take(ctx context.Context, key string) (string, error) {
	v, err := c.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}

func (c *loginGuardCache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}
