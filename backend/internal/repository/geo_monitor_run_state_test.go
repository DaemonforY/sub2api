//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGeoRunStateOneAtATime(t *testing.T) {
	mr := miniredis.RunT(t)
	st := NewGeoRunState(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ctx := context.Background()
	start := time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)

	p, err := st.Progress(ctx)
	require.NoError(t, err)
	require.False(t, p.Running)

	ok, err := st.TryStart(ctx, "run-a", start, 3)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = st.TryStart(ctx, "run-b", start, 1)
	require.NoError(t, err)
	require.False(t, ok, "a second run waits")
	require.Greater(t, mr.TTL(geoRedisLockKey), time.Hour)

	st.IncDone(ctx, "run-a")
	st.IncDone(ctx, "run-b") // not the current run: ignored
	p, err = st.Progress(ctx)
	require.NoError(t, err)
	require.True(t, p.Running)
	require.Equal(t, "run-a", p.RunID)
	require.Equal(t, 3, p.Total)
	require.Equal(t, 1, p.Done)
	require.Equal(t, start, p.StartedAt.UTC())

	st.Finish(ctx, "run-b") // someone else's lock stays
	require.True(t, mr.Exists(geoRedisLockKey))
	st.Finish(ctx, "run-a")
	require.False(t, mr.Exists(geoRedisLockKey))
	require.False(t, mr.Exists(geoRedisProgressKey))
	ok, _ = st.TryStart(ctx, "run-c", start, 1)
	require.True(t, ok)

	require.Nil(t, NewGeoRunState(nil))
}
