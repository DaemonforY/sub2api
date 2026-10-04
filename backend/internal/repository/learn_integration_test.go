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

func TestLearnRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	u1 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln-1-" + suffix + "@test.local", Username: "ln1" + suffix[len(suffix)-6:]})
	u2 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln-2-" + suffix + "@test.local", Username: "ln2" + suffix[len(suffix)-6:]})
	repo := NewLearnRepository(integrationDB)
	since := time.Now().Add(-time.Minute)

	require.NoError(t, repo.MarkDone(ctx, u1.ID, []string{"a1", "a2"}))
	require.NoError(t, repo.MarkDone(ctx, u1.ID, []string{"a2", "b1"})) // repeats are ignored
	done, err := repo.Progress(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, done, 3)

	before, err := repo.CountRuns(ctx, 0, since)
	require.NoError(t, err)
	ok, err := repo.StartRun(ctx, u1.ID, "a3")
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, ok, "ok", 1200, 30, 90))
	failed, err := repo.StartRun(ctx, u1.ID, "a3")
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, failed, "failed", 300, 0, 0))
	_, err = repo.StartRun(ctx, u2.ID, "a4") // still pending: counts
	require.NoError(t, err)

	n, err := repo.CountRuns(ctx, u1.ID, since)
	require.NoError(t, err)
	require.Equal(t, 1, n) // the failed run does not count
	all, err := repo.CountRuns(ctx, 0, since)
	require.NoError(t, err)
	require.Equal(t, before+2, all)

	stats, err := repo.Stats(ctx, since)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.Learners, 1)
	require.GreaterOrEqual(t, stats.RunsToday, 2)
	require.GreaterOrEqual(t, stats.FailedRuns7d, 1)
	require.GreaterOrEqual(t, stats.Tokens7d, 120)
	var a3 *service.LearnLessonStat
	for i := range stats.Lessons {
		if stats.Lessons[i].LessonID == "a3" {
			a3 = &stats.Lessons[i]
		}
	}
	require.NotNil(t, a3)
	require.GreaterOrEqual(t, a3.Runs, 1)
}
