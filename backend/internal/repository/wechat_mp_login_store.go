package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// wechatMPLoginStore backs service.WeChatMPLoginService (keys "wechat:qrlogin:…",
// "wechat:logincode:…", "wechat:mp:…").
type wechatMPLoginStore struct {
	rdb *redis.Client
}

func NewWeChatMPLoginStore(rdb *redis.Client) service.WeChatMPLoginStore {
	return &wechatMPLoginStore{rdb: rdb}
}

// wechatMPClaimHashFieldScript: -1 hash gone, 1 field set now (TTL raised to ARGV[3] ms) or already
// equal, 0 held by another value.
var wechatMPClaimHashFieldScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[1]) == 0 then return -1 end
if redis.call('HSETNX', KEYS[1], ARGV[1], ARGV[2]) == 1 then
  local ttl = redis.call('PTTL', KEYS[1])
  if ttl < tonumber(ARGV[3]) then redis.call('PEXPIRE', KEYS[1], ARGV[3]) end
  return 1
end
if redis.call('HGET', KEYS[1], ARGV[1]) == ARGV[2] then return 1 end
return 0
`)

// wechatMPTakeHashScript deletes KEYS[1] and returns its fields when field ARGV[1] is set, writing the
// marker hash KEYS[2] (fields ARGV[3..], TTL ARGV[2] ms) in the same step; {} otherwise.
var wechatMPTakeHashScript = redis.NewScript(`
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 then return {} end
local fields = redis.call('HGETALL', KEYS[1])
redis.call('DEL', KEYS[1])
if #ARGV > 2 then
  redis.call('DEL', KEYS[2])
  for i = 3, #ARGV, 2 do
    redis.call('HSET', KEYS[2], ARGV[i], ARGV[i + 1])
  end
  redis.call('PEXPIRE', KEYS[2], ARGV[2])
end
return fields
`)

func (s *wechatMPLoginStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := loginGuardIncrScript.Run(ctx, s.rdb, []string{key}, ttl.Milliseconds()).Int64()
	if err != nil {
		return 0, fmt.Errorf("wechat mp login increment: %w", err)
	}
	return n, nil
}

func (s *wechatMPLoginStore) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (s *wechatMPLoginStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, value, ttl).Err()
}

func (s *wechatMPLoginStore) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (s *wechatMPLoginStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *wechatMPLoginStore) HSetWithTTL(ctx context.Context, key string, fields map[string]string, ttl time.Duration) error {
	values := make(map[string]any, len(fields))
	for k, v := range fields {
		values[k] = v
	}
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, values)
	pipe.PExpire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *wechatMPLoginStore) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return s.rdb.HGetAll(ctx, key).Result()
}

func (s *wechatMPLoginStore) ClaimHashField(ctx context.Context, key, field, value string, minTTL time.Duration) (int, error) {
	return wechatMPClaimHashFieldScript.Run(ctx, s.rdb, []string{key}, field, value, minTTL.Milliseconds()).Int()
}

func (s *wechatMPLoginStore) TakeHash(ctx context.Context, key, requiredField, markerKey string, marker map[string]string, markerTTL time.Duration) (map[string]string, error) {
	args := make([]any, 0, 2+2*len(marker))
	args = append(args, requiredField, markerTTL.Milliseconds())
	for k, v := range marker {
		args = append(args, k, v)
	}
	raw, err := wechatMPTakeHashScript.Run(ctx, s.rdb, []string{key, markerKey}, args...).StringSlice()
	if err != nil {
		return nil, fmt.Errorf("wechat mp take hash: %w", err)
	}
	out := make(map[string]string, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		out[raw[i]] = raw[i+1]
	}
	return out, nil
}
