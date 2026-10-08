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

func TestLoginGuardCacheFixedWindowAndTake(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	c := NewLoginGuardCache(redis.NewClient(&redis.Options{Addr: mr.Addr()}))

	n, err := c.Incr(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	mr.FastForward(40 * time.Second)
	n, err = c.Incr(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	ttl, err := c.TTL(ctx, "k")
	require.NoError(t, err)
	require.LessOrEqual(t, ttl, 20*time.Second, "later increments keep the window's original expiry")
	mr.FastForward(21 * time.Second)
	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	require.Zero(t, got)

	ttl, err = c.TTL(ctx, "missing")
	require.NoError(t, err)
	require.Zero(t, ttl)

	require.NoError(t, c.Set(ctx, "cap", "AB7K", time.Minute))
	v, err := c.Take(ctx, "cap")
	require.NoError(t, err)
	require.Equal(t, "AB7K", v)
	v, err = c.Take(ctx, "cap")
	require.NoError(t, err)
	require.Empty(t, v)
}
