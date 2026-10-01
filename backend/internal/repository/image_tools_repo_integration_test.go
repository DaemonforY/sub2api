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

func TestImageToolsRepositoryCharge(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "imgtools-" + suffix + "@test.local", Username: "it" + suffix[len(suffix)-6:], Balance: 0.06})
	repo := NewImageToolsRepository(integrationDB)
	today := time.Now().Add(-time.Hour)
	use := service.ImageToolUse{UserID: user.ID, Tool: service.ImageToolRemoveBg, Price: 0.02, Subscribed: true, InputBytes: 10, OutputBytes: 20}

	for i := 0; i < 2; i++ {
		free, err := repo.Charge(ctx, use, 2, today)
		require.NoError(t, err)
		require.True(t, free, "run %d is free", i)
	}
	free, err := repo.Charge(ctx, use, 2, today)
	require.NoError(t, err)
	require.False(t, free, "free runs are used up")
	n, err := repo.CountFreeSince(ctx, user.ID, today)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	n, err = repo.CountFreeSince(ctx, user.ID, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, n, "free runs reset the next day")
	balance, err := repo.Balance(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.04, balance, 1e-9)

	use.Subscribed = false
	use.Price = 0.05
	_, err = repo.Charge(ctx, use, 2, today)
	require.ErrorIs(t, err, service.ErrInsufficientBalance)
	balance, _ = repo.Balance(ctx, user.ID)
	require.InDelta(t, 0.04, balance, 1e-9, "an unaffordable run takes nothing")
	var rows int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_tool_uses WHERE user_id = $1`, user.ID).Scan(&rows))
	require.Equal(t, 3, rows)

	list, total, err := repo.ListUses(ctx, service.ImageToolUseQuery{UserID: user.ID, Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, list, 2)
	require.Equal(t, user.Email, list[0].UserEmail)
	require.False(t, list[0].Free, "newest first")
	require.InDelta(t, 0.02, list[0].Cost, 1e-9)
	list, total, err = repo.ListUses(ctx, service.ImageToolUseQuery{Keyword: "imgtools-" + suffix, Tool: service.ImageToolUpscale, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, list)
	stats, err := repo.Stats(ctx, user.ID, today)
	require.NoError(t, err)
	require.Equal(t, []service.ImageToolStat{{Tool: service.ImageToolRemoveBg, Runs: 3, FreeRuns: 2, Cost: 0.02, Users: 1}}, stats)
	all, err := repo.Stats(ctx, 0, today)
	require.NoError(t, err)
	require.NotEmpty(t, all)
}
