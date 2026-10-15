//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestContentLogRepository(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	admin := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "cl-admin-" + sfx + "@test.local", Username: "cla" + sfx[len(sfx)-6:]}))
	target := rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "cl-target-" + sfx + "@test.local", Username: "clt" + sfx[len(sfx)-6:]}))
	ids := pq.Array([]int64{admin.ID, target.ID})
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, `DELETE FROM request_content_logs WHERE user_id = ANY($1)`, ids)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `DELETE FROM content_log_access_logs WHERE target_user_id = ANY($1) OR admin_id = ANY($1)`, ids)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `DELETE FROM content_log_targets WHERE user_id = ANY($1)`, ids)
		require.NoError(t, err)
	})
	repo := NewContentLogRepository(integrationDB)

	ok, err := repo.UserExists(ctx, target.ID)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.UserExists(ctx, -1)
	require.NoError(t, err)
	require.False(t, ok)

	enable := service.ContentLogAccess{AdminID: admin.ID, Action: service.ContentLogActionEnable, TargetUserID: target.ID}
	require.NoError(t, repo.EnableTarget(ctx, target.ID, "first", enable))
	require.NoError(t, repo.EnableTarget(ctx, target.ID, "second", enable))
	// An audit row that can't be written rolls the change back.
	require.Error(t, repo.DisableTarget(ctx, target.ID, service.ContentLogAccess{AdminID: admin.ID, Action: "bogus", TargetUserID: target.ID}))
	ids2, err := repo.ListTargetIDs(ctx)
	require.NoError(t, err)
	require.Contains(t, ids2, target.ID)

	group := int64(2)
	key := int64(35)
	now := time.Now().UTC()
	logs := []*service.RequestContentLog{
		{UserID: target.ID, APIKeyID: &key, GroupID: &group, RequestID: "r1", SessionID: "s1", Model: "m", InboundEndpoint: "/v1/responses",
			UserText: "hello", ToolCalls: []service.ContentToolCall{{Name: "shell", Arguments: `{"a":1}`}}, ReplyHead: "hi", InputTokens: 1, OutputTokens: 2, CacheReadTokens: 3, CreatedAt: now},
		{UserID: target.ID, RequestID: "r2", SessionID: "s2", UserText: "nul\x00byte", CreatedAt: now},
		{UserID: target.ID, RequestID: "old", CreatedAt: now.AddDate(0, 0, -8)},
	}
	n, err := repo.BatchInsertLogs(ctx, logs)
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	all, err := repo.ListLogs(ctx, service.ContentLogFilter{UserID: target.ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "old", all[0].RequestID) // newest id first
	page, err := repo.ListLogs(ctx, service.ContentLogFilter{UserID: target.ID, SessionID: "s1", Limit: 10})
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, []service.ContentToolCall{{Name: "shell", Arguments: `{"a":1}`}}, page[0].ToolCalls)
	require.Equal(t, int64(35), *page[0].APIKeyID)
	require.Equal(t, 3, page[0].CacheReadTokens)
	before, err := repo.ListLogs(ctx, service.ContentLogFilter{UserID: target.ID, BeforeID: all[0].ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, before, 2)
	require.Equal(t, "nulbyte", before[0].UserText)
	require.Equal(t, []service.ContentToolCall{}, before[0].ToolCalls)

	targets, err := repo.ListTargets(ctx)
	require.NoError(t, err)
	var found *service.ContentLogTarget
	for i := range targets {
		if targets[i].UserID == target.ID {
			found = &targets[i]
		}
	}
	require.NotNil(t, found)
	require.Equal(t, "second", found.Note)
	require.Equal(t, target.Email, found.Email)
	require.Equal(t, admin.Email, found.CreatedByEmail)
	require.Equal(t, int64(3), found.LogCount)
	require.NotNil(t, found.LastLoggedAt)

	require.NoError(t, repo.InsertAccess(ctx, service.ContentLogAccess{AdminID: admin.ID, Action: service.ContentLogActionView, TargetUserID: target.ID, Detail: `{"user_id":1}`, IPAddress: "1.2.3.4", UserAgent: "ua"}))
	access, err := repo.ListAccess(ctx, target.ID, 10)
	require.NoError(t, err)
	require.Len(t, access, 3) // enable, enable, view
	require.Equal(t, admin.Email, access[0].AdminEmail)
	require.Equal(t, target.Email, access[0].TargetEmail)
	_, err = repo.ListAccess(ctx, 0, 10)
	require.NoError(t, err)

	deleted, err := repo.DeleteLogsBefore(ctx, now.AddDate(0, 0, -7), 1000)
	require.NoError(t, err)
	require.GreaterOrEqual(t, deleted, int64(1))
	all, err = repo.ListLogs(ctx, service.ContentLogFilter{UserID: target.ID, Limit: 10})
	require.NoError(t, err)
	require.Len(t, all, 2)
	_, err = repo.DeleteAccessBefore(ctx, now.AddDate(0, 0, -180), 1000)
	require.NoError(t, err)

	require.NoError(t, repo.DisableTarget(ctx, target.ID, service.ContentLogAccess{AdminID: admin.ID, Action: service.ContentLogActionDisable, TargetUserID: target.ID}))
	ids2, err = repo.ListTargetIDs(ctx)
	require.NoError(t, err)
	require.NotContains(t, ids2, target.ID)
}
