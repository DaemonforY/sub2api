//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestArchiveDedupeStore_Redis(t *testing.T) {
	mr := miniredis.RunT(t)
	store := NewArchiveDedupeStore(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ctx := context.Background()
	key := "conv_archive:sess:1:abc"

	st, err := store.Load(ctx, key)
	require.NoError(t, err)
	require.Nil(t, st)

	require.NoError(t, store.Save(ctx, key, &service.ArchiveDedupeState{InstrHash: "h", ItemHashes: []string{"a", "b"}}, 48*time.Hour))
	st, err = store.Load(ctx, key)
	require.NoError(t, err)
	require.Equal(t, "h", st.InstrHash)
	require.Equal(t, []string{"a", "b"}, st.ItemHashes)
	require.Greater(t, mr.TTL(key), 47*time.Hour)

	require.NoError(t, mr.Set(key, "not json"))
	st, err = store.Load(ctx, key)
	require.NoError(t, err)
	require.Nil(t, st, "a corrupt state is treated as absent")

	mr.Close()
	_, err = store.Load(ctx, key)
	require.Error(t, err, "Redis down is reported so the turn is archived in full")
}
