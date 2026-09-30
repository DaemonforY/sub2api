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
