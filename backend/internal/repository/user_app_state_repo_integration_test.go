//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserAppStateCompareAndSet(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "appstate-" + suffix + "@appstate.test", Username: "appstate" + suffix[len(suffix)-4:]})
	svc := service.NewUserAppStateService(NewUserAppStateRepository(integrationDB))

	empty, err := svc.Get(ctx, user.ID, "canvas.favorites")
	require.NoError(t, err)
	require.Equal(t, int64(0), empty.Version)
	require.JSONEq(t, `{}`, string(empty.Value))

	first, applied, err := svc.Put(ctx, user.ID, "canvas.favorites", json.RawMessage(`{"items":{"a":1}}`), 0)
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, int64(1), first.Version)

	// A second device that has not seen version 1 must not overwrite it.
	stale, applied, err := svc.Put(ctx, user.ID, "canvas.favorites", json.RawMessage(`{"items":{"b":1}}`), 0)
	require.NoError(t, err)
	require.False(t, applied)
	require.Equal(t, int64(1), stale.Version)
	require.JSONEq(t, `{"items":{"a":1}}`, string(stale.Value))

	second, applied, err := svc.Put(ctx, user.ID, "canvas.favorites", json.RawMessage(`{"items":{"a":1,"b":1}}`), 1)
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, int64(2), second.Version)

	got, err := svc.Get(ctx, user.ID, "canvas.favorites")
	require.NoError(t, err)
	require.Equal(t, int64(2), got.Version)
	require.JSONEq(t, `{"items":{"a":1,"b":1}}`, string(got.Value))

	// Namespaces are independent.
	drafts, err := svc.Get(ctx, user.ID, "canvas.drafts")
	require.NoError(t, err)
	require.Equal(t, int64(0), drafts.Version)
}

func TestUserAppBlobQuotaEviction(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "appblob-" + suffix + "@appstate.test", Username: "appblob" + suffix[len(suffix)-4:]})
	repo := NewUserAppBlobRepository(integrationDB)

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.InsertBlob(ctx, &service.UserAppBlob{ID: fmt.Sprintf("%032x", time.Now().UnixNano()+int64(i)), UserID: user.ID, SHA256: fmt.Sprintf("%064d", i), MimeType: "image/png", SizeBytes: 100}))
		time.Sleep(5 * time.Millisecond)
	}
	// Keep the 3 most recently used, or 250 bytes: the 2 oldest (and by size a third) go.
	overCount, err := repo.BlobsOverQuota(ctx, user.ID, 3, 1<<20)
	require.NoError(t, err)
	require.Len(t, overCount, 2)
	require.Equal(t, fmt.Sprintf("%064d", 0), overCount[0].SHA256)

	overSize, err := repo.BlobsOverQuota(ctx, user.ID, 10, 250)
	require.NoError(t, err)
	require.Len(t, overSize, 3)

	found, err := repo.FindBlobBySHA(ctx, user.ID, fmt.Sprintf("%064d", 4))
	require.NoError(t, err)
	require.NotNil(t, found)
	other, err := repo.GetBlob(ctx, user.ID+999999, found.ID)
	require.NoError(t, err)
	require.Nil(t, other, "blobs are owner-scoped")

	require.NoError(t, repo.DeleteBlobs(ctx, user.ID, []string{found.ID}))
	gone, err := repo.GetBlob(ctx, user.ID, found.ID)
	require.NoError(t, err)
	require.Nil(t, gone)
}
