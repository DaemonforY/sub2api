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

func TestShihuaRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewShihuaRepository(integrationDB)
	slug := fmt.Sprintf("it-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM shihua_works WHERE slug = $1`, slug)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM shihua_jobs WHERE name = $1`, slug)
	})

	now := time.Now()
	by := int64(1)
	w := &service.ShihuaWork{Slug: slug, Title: "静夜思", Author: "李白", Genre: "shi", Stage: "primary", Grade: "一下",
		Textbooks: []service.ShihuaTextbook{{Edition: "统编", Grade: "一下", Lesson: "8"}}, FullText: "床前明月光，疑是地上霜。",
		TextStatus: service.ShihuaTextVerified1, Verify1By: &by, Verify1At: &now, Theme: service.ShihuaTheme{PromptTheme: "深秋夜"}}
	require.NoError(t, repo.CreateWork(ctx, w))
	got, err := repo.GetWorkBySlug(ctx, slug)
	require.NoError(t, err)
	require.Equal(t, "深秋夜", got.Theme.PromptTheme)
	require.Equal(t, "统编", got.Textbooks[0].Edition)
	require.Equal(t, by, *got.Verify1By)
	require.Nil(t, got.Verify2At)

	got.Grade = "一年级下册"
	require.NoError(t, repo.UpdateWork(ctx, got))

	images := []service.ShihuaImage{{WorkID: w.ID, Prompt: "p1", Size: "1024x1536", Quality: "high"}, {WorkID: w.ID, Prompt: "p2", Size: "1024x1536", Quality: "high"}}
	job, err := repo.CreateJob(ctx, slug, 2, by, images)
	require.NoError(t, err)
	require.Equal(t, 2, job.Counts.Queued)

	// Claim and finish one drawing; push the other back with a retry time in the future.
	first, err := repo.ClaimNext(ctx)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Equal(t, 1, first.Attempts)
	first.Status, first.File, first.Width, first.Height, first.CostUSD = service.ShihuaImageSucceeded, "works/1/1.png", 1024, 1536, 0.201
	first.FinishedAt = &now
	require.NoError(t, repo.FinishImage(ctx, first))

	second, err := repo.ClaimNext(ctx)
	require.NoError(t, err)
	later := time.Now().Add(time.Hour)
	second.Status, second.NotBefore, second.Error = service.ShihuaImageQueued, &later, "HTTP 502"
	require.NoError(t, repo.FinishImage(ctx, second))
	if next, err := repo.ClaimNext(ctx); err == nil && next != nil {
		require.NotEqual(t, second.ID, next.ID, "a drawing waiting to retry is not due yet")
		next.Status = service.ShihuaImageQueued
		require.NoError(t, repo.FinishImage(ctx, next))
	}

	stats, err := repo.QueueStats(ctx, now.Add(-time.Minute))
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.StartedToday, 2)

	list, err := repo.ListWorks(ctx, service.ShihuaWorkQuery{IDs: []int64{w.ID}})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, service.ShihuaImageCounts{Total: 2, Queued: 1, Succeeded: 1}, list[0].Images)
	require.Equal(t, "一年级下册", list[0].Grade)

	jobs, err := repo.ListJobs(ctx, 50)
	require.NoError(t, err)
	var mine *service.ShihuaJob
	for i := range jobs {
		if jobs[i].ID == job.ID {
			mine = &jobs[i]
		}
	}
	require.NotNil(t, mine)
	require.Equal(t, 1, mine.Works)
	require.InDelta(t, 0.201, mine.CostUSD, 1e-9)

	n, err := repo.CancelQueued(ctx, job.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	n, err = repo.RetryFailed(ctx, job.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	// Pick, upscale queue, renders.
	require.NoError(t, repo.ReviewImage(ctx, first.ID, false, "ok"))
	require.NoError(t, repo.SetPicked(ctx, w.ID, &first.ID))
	require.NoError(t, repo.SetUpscale(ctx, first.ID, service.ShihuaUpscaleQueued, "", ""))
	claimed, err := repo.ClaimUpscale(ctx)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.NoError(t, repo.SetUpscale(ctx, claimed.ID, service.ShihuaUpscaleDone, "", "works/1/1-2x.jpg"))

	r := &service.ShihuaRender{ImageID: first.ID, WorkID: w.ID, Aspect: "2x3", File: "works/1/1-2x3.jpg", Width: 2048, Height: 3072, Layout: json.RawMessage(`{"cell":60}`)}
	require.NoError(t, repo.SaveRender(ctx, r))
	r.File = "works/1/1-2x3.png"
	require.NoError(t, repo.SaveRender(ctx, r), "rendering the same aspect again replaces the file")
	renders, err := repo.ListRenders(ctx, []int64{w.ID})
	require.NoError(t, err)
	require.Len(t, renders, 1)
	require.Equal(t, "works/1/1-2x3.png", renders[0].File)

	list, err = repo.ListWorks(ctx, service.ShihuaWorkQuery{IDs: []int64{w.ID}})
	require.NoError(t, err)
	require.Equal(t, 1, list[0].Renders)
	require.Equal(t, first.ID, *list[0].PickedImageID)

	img, err := repo.GetImage(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, "works/1/1-2x.jpg", img.UpscaledFile)
	require.Equal(t, "ok", img.TitleCheck)

	require.NoError(t, repo.DeleteWork(ctx, w.ID))
	gone, err := repo.GetWork(ctx, w.ID)
	require.NoError(t, err)
	require.Nil(t, gone)
	left, err := repo.ListImages(ctx, service.ShihuaImageQuery{WorkIDs: []int64{w.ID}, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, left, "deleting a work deletes its drawings")
}
