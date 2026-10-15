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

func newWeChatMPLoginStoreForTest(t *testing.T) (*wechatMPLoginStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewWeChatMPLoginStore(rdb).(*wechatMPLoginStore), mr
}

func TestWeChatMPLoginStoreClaimHashField(t *testing.T) {
	store, mr := newWeChatMPLoginStoreForTest(t)
	ctx := context.Background()

	res, err := store.ClaimHashField(ctx, "wechat:qrlogin:x", "openid", "o1", 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, -1, res, "a missing ticket is never created")
	require.False(t, mr.Exists("wechat:qrlogin:x"))

	require.NoError(t, store.HSetWithTTL(ctx, "wechat:qrlogin:x", map[string]string{"browser": "b"}, 30*time.Second))
	res, err = store.ClaimHashField(ctx, "wechat:qrlogin:x", "openid", "o1", 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, res)
	require.Equal(t, 2*time.Minute, mr.TTL("wechat:qrlogin:x"), "a scanned ticket lives at least minTTL")

	res, err = store.ClaimHashField(ctx, "wechat:qrlogin:x", "openid", "o1", 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, res, "the same scanner again is fine")
	res, err = store.ClaimHashField(ctx, "wechat:qrlogin:x", "openid", "o2", 2*time.Minute)
	require.NoError(t, err)
	require.Equal(t, 0, res, "a second scanner does not take it over")

	values, err := store.HGetAll(ctx, "wechat:qrlogin:x")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"browser": "b", "openid": "o1"}, values)
}

func TestWeChatMPLoginStoreCountersAndOneTimeValues(t *testing.T) {
	store, mr := newWeChatMPLoginStoreForTest(t)
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		n, err := store.Incr(ctx, "wechat:logincode:verify_ip:1.2.3.4", 10*time.Minute)
		require.NoError(t, err)
		require.Equal(t, i, n)
	}
	require.Equal(t, 10*time.Minute, mr.TTL("wechat:logincode:verify_ip:1.2.3.4"))

	ok, err := store.SetNX(ctx, "wechat:logincode:123456", "openid", 5*time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = store.SetNX(ctx, "wechat:logincode:123456", "other", 5*time.Minute)
	require.NoError(t, err)
	require.False(t, ok)

	_, found, err := store.Get(ctx, "wechat:mp:access_token:wx")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, store.Set(ctx, "wechat:mp:access_token:wx", "tok", time.Hour))
	v, found, err := store.Get(ctx, "wechat:mp:access_token:wx")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "tok", v)
	require.NoError(t, store.Del(ctx, "wechat:mp:access_token:wx"))
	require.False(t, mr.Exists("wechat:mp:access_token:wx"))
}

func TestWeChatMPLoginStoreTakeHashIsOneShot(t *testing.T) {
	store, mr := newWeChatMPLoginStoreForTest(t)
	ctx := context.Background()
	marker := map[string]string{"status": "scanned", "browser": "b"}

	got, err := store.TakeHash(ctx, "wechat:qrlogin:t", "openid", "wechat:qrlogin:t:result", marker, time.Minute)
	require.NoError(t, err)
	require.Empty(t, got, "missing ticket")

	require.NoError(t, store.HSetWithTTL(ctx, "wechat:qrlogin:t", map[string]string{"browser": "b"}, 5*time.Minute))
	got, err = store.TakeHash(ctx, "wechat:qrlogin:t", "openid", "wechat:qrlogin:t:result", marker, time.Minute)
	require.NoError(t, err)
	require.Empty(t, got, "not scanned yet: left alone")
	require.True(t, mr.Exists("wechat:qrlogin:t"))
	require.False(t, mr.Exists("wechat:qrlogin:t:result"))

	res, err := store.ClaimHashField(ctx, "wechat:qrlogin:t", "openid", "o1", time.Minute)
	require.NoError(t, err)
	require.Equal(t, 1, res)
	got, err = store.TakeHash(ctx, "wechat:qrlogin:t", "openid", "wechat:qrlogin:t:result", marker, time.Minute)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"browser": "b", "openid": "o1"}, got)
	require.False(t, mr.Exists("wechat:qrlogin:t"))
	result, err := store.HGetAll(ctx, "wechat:qrlogin:t:result")
	require.NoError(t, err)
	require.Equal(t, marker, result)
	require.Equal(t, time.Minute, mr.TTL("wechat:qrlogin:t:result"))

	got, err = store.TakeHash(ctx, "wechat:qrlogin:t", "openid", "wechat:qrlogin:t:result", marker, time.Minute)
	require.NoError(t, err)
	require.Empty(t, got, "a second poll gets nothing")
}
