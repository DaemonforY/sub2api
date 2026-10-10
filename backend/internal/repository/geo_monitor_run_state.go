package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// GEO 监测 run lock and progress in Redis: SET NX with a TTL, so only one run happens at a time
// across instances and a crashed run unlocks itself.

const (
	geoRedisLockKey     = "geo_monitor:run_lock"
	geoRedisProgressKey = "geo_monitor:run_progress"
)

type geoRedisRunState struct{ rdb *redis.Client }

// NewGeoRunState keeps the GEO 监测 run state in Redis (in-process when there is no Redis client).
func NewGeoRunState(rdb *redis.Client) service.GeoRunState {
	if rdb == nil {
		return nil
	}
	return &geoRedisRunState{rdb: rdb}
}

func (r *geoRedisRunState) TryStart(ctx context.Context, runID string, startedAt time.Time, total int) (bool, error) {
	ok, err := r.rdb.SetNX(ctx, geoRedisLockKey, runID, service.GeoRunLockTTL).Result()
	if err != nil || !ok {
		return false, err
	}
	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, geoRedisProgressKey)
	pipe.HSet(ctx, geoRedisProgressKey, "run_id", runID, "started_at", startedAt.UTC().Format(time.RFC3339), "total", total, "done", 0)
	pipe.Expire(ctx, geoRedisProgressKey, service.GeoRunLockTTL)
	_, err = pipe.Exec(ctx)
	return true, err
}

func (r *geoRedisRunState) Progress(ctx context.Context) (service.GeoRunStatus, error) {
	var st service.GeoRunStatus
	runID, err := r.rdb.Get(ctx, geoRedisLockKey).Result()
	if errors.Is(err, redis.Nil) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	st.Running, st.RunID = true, runID
	h, err := r.rdb.HGetAll(ctx, geoRedisProgressKey).Result()
	if err != nil {
		return st, err
	}
	if h["run_id"] == runID {
		if t, err := time.Parse(time.RFC3339, h["started_at"]); err == nil {
			st.StartedAt = &t
		}
		st.Total, _ = strconv.Atoi(h["total"])
		st.Done, _ = strconv.Atoi(h["done"])
	}
	return st, nil
}

var geoRedisIncDone = redis.NewScript(`if redis.call("HGET", KEYS[1], "run_id") == ARGV[1] then return redis.call("HINCRBY", KEYS[1], "done", 1) end return 0`)

func (r *geoRedisRunState) IncDone(ctx context.Context, runID string) {
	_ = geoRedisIncDone.Run(ctx, r.rdb, []string{geoRedisProgressKey}, runID).Err()
}

var geoRedisUnlock = redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then redis.call("DEL", KEYS[2]) return redis.call("DEL", KEYS[1]) end return 0`)

func (r *geoRedisRunState) Finish(ctx context.Context, runID string) {
	_ = geoRedisUnlock.Run(ctx, r.rdb, []string{geoRedisLockKey, geoRedisProgressKey}, runID).Err()
}
