//go:build integration

package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type cloudQuotaSettings struct{ values map[string]string }

func (s cloudQuotaSettings) GetMultiple(context.Context, []string) (map[string]string, error) {
	return s.values, nil
}
func (s cloudQuotaSettings) SetMultiple(context.Context, map[string]string) error { return nil }

func TestCanvasCloudFiles(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	alice := mustCreateUser(t, integrationEntClient, &service.User{Email: "cloud-a-" + suffix + "@test.local"})
	bob := mustCreateUser(t, integrationEntClient, &service.User{Email: "cloud-b-" + suffix + "@test.local"})
	dir := t.TempDir()
	// 1MB quota for everyone (no subscriptions in this test).
	svc := service.NewCanvasCloudService(NewCanvasCloudRepository(integrationDB), dir, cloudQuotaSettings{values: map[string]string{"canvas_cloud_quota_mb": "1"}}, nil)

	read := func(userID int64, path string) string {
		t.Helper()
		f, file, err := svc.Open(ctx, userID, path)
		require.NoError(t, err)
		defer func() { _ = file.Close() }()
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, int64(len(data)), f.Size)
		return string(data)
	}

	f, err := svc.Put(ctx, alice.ID, "canvas/manifest.json", "application/json; charset=utf-8", strings.NewReader(`{"v":1}`))
	require.NoError(t, err)
	require.Equal(t, "application/json", f.Mime)
	require.Equal(t, `{"v":1}`, read(alice.ID, "canvas/manifest.json"))

	// Overwriting keeps one file on disk.
	_, err = svc.Put(ctx, alice.ID, "canvas/manifest.json", "application/json", strings.NewReader(`{"v":2}`))
	require.NoError(t, err)
	require.Equal(t, `{"v":2}`, read(alice.ID, "canvas/manifest.json"))
	entries, err := os.ReadDir(filepath.Join(dir, fmt.Sprint(alice.ID)))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// Media types are kept; HTML, SVG and the rest become downloads of octet-stream.
	f, err = svc.Put(ctx, alice.ID, "assets/files/image_abc.png", "image/png", bytes.NewReader(make([]byte, 300<<10)))
	require.NoError(t, err)
	require.Equal(t, "image/png", f.Mime)
	f, err = svc.Put(ctx, alice.ID, "assets/files/page.html", "text/html", strings.NewReader("<script>alert(1)</script>"))
	require.NoError(t, err)
	require.Equal(t, "application/octet-stream", f.Mime)
	require.Equal(t, "application/octet-stream", service.CanvasCloudMime("image/svg+xml"))

	// Paths outside the sync layout are refused.
	for _, bad := range []string{"../etc/passwd", "canvas/../x", "other/manifest.json", "canvas/files/a/b.png", "canvas/files/", "/canvas/manifest.json"} {
		_, err = svc.Put(ctx, alice.ID, bad, "image/png", strings.NewReader("x"))
		require.ErrorIs(t, err, service.ErrCanvasCloudPath, bad)
	}
	_, err = svc.Put(ctx, alice.ID, "canvas/files/empty.png", "image/png", strings.NewReader(""))
	require.ErrorIs(t, err, service.ErrCanvasCloudEmpty)

	// Quota: 1MB in total; replacing a file only counts the difference.
	_, err = svc.Put(ctx, alice.ID, "canvas/files/big.png", "image/png", bytes.NewReader(make([]byte, 800<<10)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "云同步空间已满")
	_, err = svc.Put(ctx, alice.ID, "assets/files/image_abc.png", "image/png", bytes.NewReader(make([]byte, 600<<10)))
	require.NoError(t, err, "replacing a 300KB file with 600KB fits in 1MB")
	usage, err := svc.Usage(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, 3, usage.Files)
	require.Equal(t, int64(1<<20), usage.QuotaBytes)
	require.False(t, usage.Subscribed)

	// Each user only sees their own files.
	_, _, err = svc.Open(ctx, bob.ID, "canvas/manifest.json")
	require.ErrorIs(t, err, service.ErrCanvasCloudNotFound)
	files, err := svc.List(ctx, bob.ID)
	require.NoError(t, err)
	require.Empty(t, files)

	files, err = svc.List(ctx, alice.ID)
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.NoError(t, svc.Delete(ctx, alice.ID, "assets/files/page.html"))
	require.NoError(t, svc.Delete(ctx, alice.ID, "assets/files/page.html"), "deleting twice is fine")
	files, err = svc.List(ctx, alice.ID)
	require.NoError(t, err)
	require.Len(t, files, 2)
	entries, err = os.ReadDir(filepath.Join(dir, fmt.Sprint(alice.ID)))
	require.NoError(t, err)
	require.Len(t, entries, 2, "the deleted file is gone from disk too")
}
