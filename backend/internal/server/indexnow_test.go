//go:build unit

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fakeIndexNowPages map[string]string

func (f fakeIndexNowPages) IndexNowPages(context.Context) map[string]string { return f }

func TestRunIndexNowPushesOnlyChangedPages(t *testing.T) {
	var got []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		got = append(got, body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := config.IndexNowConfig{Key: "abcdef0123456789", Site: "https://hivegpt.cn/", Endpoint: srv.URL}
	ctx := context.Background()

	pages := fakeIndexNowPages{"/": "static", "/learn/connect/python.html": "aa11", "/courses/x": "2026-10-08T00:00:00Z"}
	n, err := runIndexNow(ctx, cfg, pages, rdb, srv.Client())
	require.NoError(t, err)
	require.Equal(t, 3, n, "first run pushes everything")
	require.Equal(t, "hivegpt.cn", got[0]["host"])
	require.Equal(t, "https://hivegpt.cn/abcdef0123456789.txt", got[0]["keyLocation"])
	require.ElementsMatch(t, []any{"https://hivegpt.cn/", "https://hivegpt.cn/courses/x", "https://hivegpt.cn/learn/connect/python.html"}, got[0]["urlList"])

	n, err = runIndexNow(ctx, cfg, pages, rdb, srv.Client())
	require.NoError(t, err)
	require.Zero(t, n, "nothing changed")
	require.Len(t, got, 1)

	pages["/learn/connect/python.html"] = "bb22"
	pages["/learn/connect/new.html"] = "cc33"
	n, err = runIndexNow(ctx, cfg, pages, rdb, srv.Client())
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.ElementsMatch(t, []any{"https://hivegpt.cn/learn/connect/new.html", "https://hivegpt.cn/learn/connect/python.html"}, got[1]["urlList"])
}

func TestRunIndexNowKeepsPagesUnsentWhenRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer srv.Close()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := config.IndexNowConfig{Key: "abcdef0123456789", Site: "https://hivegpt.cn", Endpoint: srv.URL}
	_, err := runIndexNow(context.Background(), cfg, fakeIndexNowPages{"/": "static"}, rdb, srv.Client())
	require.Error(t, err)
	keys, _ := rdb.HKeys(context.Background(), indexNowRedisKey).Result()
	require.Empty(t, keys, "a rejected push is retried next time")
}

func TestIndexNowEnabled(t *testing.T) {
	require.False(t, indexNowEnabled(config.IndexNowConfig{}))
	require.False(t, indexNowEnabled(config.IndexNowConfig{Key: "short", Site: "https://hivegpt.cn"}))
	require.False(t, indexNowEnabled(config.IndexNowConfig{Key: "abcdef0123456789", Site: "http://hivegpt.cn"}))
	require.True(t, indexNowEnabled(config.IndexNowConfig{Key: "abcdef0123456789", Site: "https://hivegpt.cn"}))
}
