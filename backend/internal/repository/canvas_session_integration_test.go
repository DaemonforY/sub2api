//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type canvasSubsStub struct{}

func (canvasSubsStub) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return nil, nil
}

func TestCanvasSessionsLifecycle(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "canvas-" + suffix + "@test.local", Username: "cv" + suffix[len(suffix)-6:]})
	repo := NewCanvasSessionRepository(integrationDB)
	users := NewUserRepository(integrationEntClient, integrationDB)
	svc := service.NewCanvasSessionService(repo, users, canvasSubsStub{})

	token, session, err := svc.Create(ctx, user.ID, "Mozilla/5.0 Test", "1.2.3.4")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	got, gotUser, err := svc.Authenticate(ctx, token)
	require.NoError(t, err)
	require.Equal(t, session.ID, got.ID)
	require.Equal(t, user.ID, gotUser.ID)
	_, _, err = svc.Authenticate(ctx, token+"x")
	require.ErrorIs(t, err, service.ErrCanvasSessionInvalid)

	// An idle session slides forward on use.
	_, err = integrationDB.ExecContext(ctx, `UPDATE canvas_sessions SET last_used_at = NOW() - INTERVAL '1 hour', expires_at = NOW() + INTERVAL '1 day' WHERE id = $1`, session.ID)
	require.NoError(t, err)
	got, _, err = svc.Authenticate(ctx, token)
	require.NoError(t, err)
	require.True(t, got.ExpiresAt.After(time.Now().Add(29*24*time.Hour)))

	list, err := svc.List(ctx, user.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "1.2.3.4", list[0].IP)

	ok, err := svc.Revoke(ctx, user.ID+1, session.ID)
	require.NoError(t, err)
	require.False(t, ok, "only the owner can sign a session out")
	ok, err = svc.Revoke(ctx, user.ID, session.ID)
	require.NoError(t, err)
	require.True(t, ok)
	_, _, err = svc.Authenticate(ctx, token)
	require.ErrorIs(t, err, service.ErrCanvasSessionInvalid)

	// Expired sessions are refused.
	token2, s2, err := svc.Create(ctx, user.ID, "", "")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE canvas_sessions SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, s2.ID)
	require.NoError(t, err)
	_, _, err = svc.Authenticate(ctx, token2)
	require.ErrorIs(t, err, service.ErrCanvasSessionInvalid)

	// At most 20 live sessions per user: the oldest are signed out.
	for i := 0; i < 22; i++ {
		_, _, err := svc.Create(ctx, user.ID, "", "")
		require.NoError(t, err)
	}
	var live int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM canvas_sessions WHERE user_id = $1 AND revoked_at IS NULL`, user.ID).Scan(&live))
	require.Equal(t, 20, live)
}
