//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGatewayRequestLogRepo_InsertListFilterAndRetention(t *testing.T) {
	ctx := context.Background()
	_, err := integrationDB.ExecContext(ctx, "TRUNCATE gateway_request_logs")
	require.NoError(t, err)

	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "reqlog-" + time.Now().Format("150405.000000") + "@test.com"})
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: user.ID, Name: "canvas-key", Key: "sk-reqlog-" + time.Now().Format("150405.000000")})

	repo := NewGatewayRequestLogRepository(integrationDB)
	now := time.Now().UTC()
	uid, kid := user.ID, key.ID
	inserted, err := repo.BatchInsert(ctx, []*service.GatewayRequestLog{
		{CreatedAt: now.Add(-time.Minute), Method: "POST", URL: "https://hivegpt.cn/v1/images/generations", Path: "/v1/images/generations",
			StatusCode: 200, Success: true, DurationMs: 1234, APIKey: key.Key, APIKeyID: &kid, UserID: &uid, Model: "gpt-image-2", ClientIP: "1.2.3.4"},
		{CreatedAt: now, Method: "POST", URL: "https://hivegpt.cn/chat/completions", Path: "/chat/completions",
			StatusCode: 401, Success: false, ErrorCode: "invalid_api_key", APIKey: "sk-bogus", ClientIP: "5.6.7.8"},
		{CreatedAt: now.AddDate(0, 0, -40), Method: "GET", URL: "https://hivegpt.cn/v1/models", Path: "/v1/models", StatusCode: 200, Success: true},
	})
	require.NoError(t, err)
	require.EqualValues(t, 3, inserted)

	all, err := repo.List(ctx, &service.GatewayRequestLogFilter{PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, 3, all.Total)
	require.Equal(t, "sk-bogus", all.Logs[0].APIKey, "newest first")
	require.Equal(t, user.Email, all.Logs[1].UserEmail)
	require.Equal(t, "canvas-key", all.Logs[1].APIKeyName)
	require.Equal(t, kid, *all.Logs[1].APIKeyID)

	failed := false
	onlyFailed, err := repo.List(ctx, &service.GatewayRequestLogFilter{Success: &failed})
	require.NoError(t, err)
	require.Equal(t, 1, onlyFailed.Total)
	require.Equal(t, "invalid_api_key", onlyFailed.Logs[0].ErrorCode)

	byEmail, err := repo.List(ctx, &service.GatewayRequestLogFilter{Query: user.Email})
	require.NoError(t, err)
	require.Equal(t, 1, byEmail.Total)

	byKey, err := repo.List(ctx, &service.GatewayRequestLogFilter{Query: "bogus"})
	require.NoError(t, err)
	require.Equal(t, 1, byKey.Total)

	deleted, err := repo.DeleteBefore(ctx, now.AddDate(0, 0, -30), 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
}
