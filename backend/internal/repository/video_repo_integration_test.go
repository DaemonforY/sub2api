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

func TestVideoRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "video-" + suffix + "@video.test", Username: "video" + suffix[len(suffix)-4:]})
	repo := NewVideoRepository(integrationDB)

	p := &service.VideoProject{UserID: user.ID, APIKeyID: 5, Mode: service.VideoModeFilm, Title: "测试", Prompt: "讲讲区块链 " + suffix,
		Options: service.VideoOptions{Ratio: "16:9", Style: "tech-blogger"}, Status: service.VideoStatusRunning, Width: 1920, Height: 1080,
		Visibility: service.VideoVisibilityPrivate, Category: "film"}
	require.NoError(t, repo.Create(ctx, p))
	require.Len(t, p.ID, 36)

	running, err := repo.CountRunning(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, running)

	p.Spec = &service.VideoSpec{Title: "区块链为什么改不了", Width: 1920, Height: 1080, Scenes: []service.VideoScene{{ID: "s1", Title: "开场", Narration: "旁白", Duration: 4.5, Code: "return function(){}",
		Audio: &service.VideoSceneAudio{File: "s1-abc.mp3", Duration: 3.9, Words: []service.VideoWord{{Text: "旁白", Start: 0.1, End: 0.6}}}}}}
	p.Status, p.Title, p.Duration = service.VideoStatusReady, p.Spec.Title, 4.5
	p.Usage = service.VideoUsage{PromptTokens: 100, CompletionTokens: 50, Calls: 2}
	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, service.VideoStatusReady, got.Status)
	require.Equal(t, "区块链为什么改不了", got.Title)
	require.Equal(t, "tech-blogger", got.Options.Style)
	require.Equal(t, 100, got.Usage.PromptTokens)
	require.Equal(t, 0.6, got.Spec.Scenes[0].Audio.Words[0].End)
	require.Equal(t, user.ID, got.UserID)
	missing, err := repo.Get(ctx, "not-a-uuid")
	require.NoError(t, err)
	require.Nil(t, missing)

	// Events and versions.
	require.NoError(t, repo.AddEvent(ctx, p.ID, &service.VideoEvent{Kind: "user", Text: "做个视频"}))
	second := &service.VideoEvent{Kind: "step", Text: "写脚本", Data: []byte(`{"state":"done"}`)}
	require.NoError(t, repo.AddEvent(ctx, p.ID, second))
	events, err := repo.ListEvents(ctx, p.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.JSONEq(t, `{"state":"done"}`, string(events[1].Data))
	after, err := repo.ListEvents(ctx, p.ID, events[0].ID, 10)
	require.NoError(t, err)
	require.Len(t, after, 1)

	require.NoError(t, repo.AddVersion(ctx, p.ID, "初版", p.Spec))
	versions, err := repo.ListVersions(ctx, p.ID, 10)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	v, err := repo.GetVersion(ctx, p.ID, versions[0].ID)
	require.NoError(t, err)
	require.Equal(t, "开场", v.Spec.Scenes[0].Title)

	// Gallery: only public works, by category, search and featured.
	cards, total, err := repo.Gallery(ctx, service.VideoGalleryQuery{Category: "all", Page: 1, PageSize: 10, Search: suffix})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, cards)

	require.NoError(t, repo.SetVisibility(ctx, p.ID, service.VideoVisibilityPending, "film", "区块链为什么改不了", nil))
	pending, err := repo.Pending(ctx, 100)
	require.NoError(t, err)
	require.True(t, containsCard(pending, p.ID))

	featured := true
	require.NoError(t, repo.SetVisibility(ctx, p.ID, service.VideoVisibilityPublic, "science", "区块链为什么改不了", &featured))
	got, _ = repo.Get(ctx, p.ID)
	require.NotNil(t, got.PublishedAt)
	require.True(t, got.Featured)
	for _, q := range []service.VideoGalleryQuery{
		{Category: "all", Search: suffix},
		{Category: "science", Search: suffix},
		{Category: "featured", Search: suffix, Sort: "hot"},
		{Category: "all", Search: suffix, Mode: service.VideoModeFilm, Sort: "new"},
	} {
		q.Page, q.PageSize = 1, 10
		cards, total, err = repo.Gallery(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, total, "%+v", q)
		require.Equal(t, "tech-blogger", cards[0].Style)
	}
	cards, total, err = repo.Gallery(ctx, service.VideoGalleryQuery{Category: "logo", Search: suffix, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Zero(t, total)

	require.NoError(t, repo.AddView(ctx, p.ID))
	require.NoError(t, repo.AddRemix(ctx, p.ID))
	got, _ = repo.Get(ctx, p.ID)
	require.Equal(t, 1, got.Views)
	require.Equal(t, 1, got.Remixes)

	// Going private clears the publish time.
	require.NoError(t, repo.SetVisibility(ctx, p.ID, service.VideoVisibilityPrivate, "science", "x", nil))
	got, _ = repo.Get(ctx, p.ID)
	require.Nil(t, got.PublishedAt)

	// Interrupted runs.
	other := &service.VideoProject{UserID: user.ID, APIKeyID: 5, Mode: service.VideoModeMotion, Title: "m", Prompt: "m", Status: service.VideoStatusRunning, Width: 1080, Height: 1080, Visibility: service.VideoVisibilityPrivate}
	require.NoError(t, repo.Create(ctx, other))
	ids, err := repo.FailRunning(ctx, "interrupted")
	require.NoError(t, err)
	require.Contains(t, ids, other.ID)

	list, err := repo.ListByUser(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)

	ok, err := repo.Delete(ctx, user.ID+100000, p.ID)
	require.NoError(t, err)
	require.False(t, ok, "only the owner deletes")
	ok, err = repo.Delete(ctx, user.ID, p.ID)
	require.NoError(t, err)
	require.True(t, ok)
	events, err = repo.ListEvents(ctx, p.ID, 0, 10)
	require.NoError(t, err)
	require.Empty(t, events, "events go with the project")
}

func containsCard(cards []service.VideoCard, id string) bool {
	for _, c := range cards {
		if c.ID == id {
			return true
		}
	}
	return false
}
