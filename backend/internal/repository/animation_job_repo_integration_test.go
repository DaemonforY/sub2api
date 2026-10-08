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

func TestAnimationJobRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "anim-" + suffix + "@anim.test", Username: "anim" + suffix[len(suffix)-4:]})
	repo := NewAnimationJobRepository(integrationDB)

	job, err := repo.Create(ctx, user.ID, 99, "gpt-5.5", json.RawMessage(`{"logId":"x"}`))
	require.NoError(t, err)
	require.Equal(t, service.AnimationJobPending, job.Status)
	require.JSONEq(t, `{"logId":"x"}`, string(job.Meta))

	active, err := repo.CountActive(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, active)

	other, err := repo.Get(ctx, user.ID+100000, job.ID)
	require.NoError(t, err)
	require.Nil(t, other, "another user's job is invisible")

	require.NoError(t, repo.MarkRunning(ctx, job.ID))
	applied, err := repo.Finish(ctx, job.ID, service.AnimationJobResult{Status: service.AnimationJobSucceeded, SVG: "<svg></svg>", Chars: 11, PromptTokens: 5, CompletionTokens: 6})
	require.NoError(t, err)
	require.True(t, applied)
	applied, err = repo.Finish(ctx, job.ID, service.AnimationJobResult{Status: service.AnimationJobCanceled})
	require.NoError(t, err)
	require.False(t, applied, "an ended job keeps its outcome")

	got, err := repo.Get(ctx, user.ID, job.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnimationJobSucceeded, got.Status)
	require.Equal(t, "<svg></svg>", got.SVG)
	require.NotNil(t, got.StartedAt)
	require.NotNil(t, got.FinishedAt)

	list, err := repo.List(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Empty(t, list[0].SVG, "lists leave out the SVG")

	stuck, err := repo.Create(ctx, user.ID, 99, "gpt-5.5", json.RawMessage(`{}`))
	require.NoError(t, err)
	_, err = repo.FailActive(ctx, "interrupted")
	require.NoError(t, err)
	stuckNow, err := repo.Get(ctx, user.ID, stuck.ID)
	require.NoError(t, err)
	require.Equal(t, service.AnimationJobFailed, stuckNow.Status)
	require.Equal(t, "interrupted", stuckNow.Error)

	_, err = repo.DeleteBefore(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	gone, err := repo.Get(ctx, user.ID, job.ID)
	require.NoError(t, err)
	require.Nil(t, gone)
}
