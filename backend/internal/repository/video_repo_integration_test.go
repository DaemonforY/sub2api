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

func TestVideoRepositoryChargeTTS(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "video-tts-" + suffix + "@video.test", Username: "vtts" + suffix[len(suffix)-4:], Balance: 1})
	repo := NewVideoRepository(integrationDB).(service.VideoTTSBilling)

	balance, err := repo.Balance(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 1, balance, 1e-9)
	charge := service.VideoTTSCharge{UserID: user.ID, ProjectID: "p-" + suffix, SceneID: "s1", Voice: "zh_male_m191_uranus_bigtts", Chars: 300, Cost: 0.18}
	require.NoError(t, repo.ChargeTTS(ctx, charge))
	charge.SceneID, charge.Cost = "s2", 1 // the balance was checked before recording, so it may go below zero
	require.NoError(t, repo.ChargeTTS(ctx, charge))
	balance, err = repo.Balance(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, -0.18, balance, 1e-9)
	var rows int
	var total float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*), SUM(cost) FROM video_tts_charges WHERE user_id = $1`, user.ID).Scan(&rows, &total))
	require.Equal(t, 2, rows)
	require.InDelta(t, 1.18, total, 1e-9)
	require.ErrorIs(t, repo.ChargeTTS(ctx, service.VideoTTSCharge{UserID: -1, ProjectID: "x", SceneID: "s1", Voice: "v", Chars: 1, Cost: 0.1}), service.ErrUserNotFound)
}

func TestVideoRepositoryExports(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "video-export-" + suffix + "@video.test", Username: "vexp" + suffix[len(suffix)-4:]})
	repo := NewVideoRepository(integrationDB)
	exports := repo.(service.VideoExportRepository)
	p := &service.VideoProject{UserID: user.ID, APIKeyID: 5, Mode: service.VideoModeFilm, Title: "导出测试", Prompt: "p " + suffix,
		Options: service.VideoOptions{Ratio: "16:9"}, Status: service.VideoStatusReady, Width: 1920, Height: 1080, Visibility: service.VideoVisibilityPrivate}
	require.NoError(t, repo.Create(ctx, p))

	e := &service.VideoExport{ID: "e" + suffix, ProjectID: p.ID, UserID: user.ID, Layout: service.VideoLayoutPortrait, Status: service.VideoExportQueued, Signature: "sig1"}
	require.NoError(t, exports.CreateExport(ctx, e))
	require.False(t, e.CreatedAt.IsZero())
	ids, err := exports.UnfinishedExports(ctx)
	require.NoError(t, err)
	require.Contains(t, ids, e.ID)

	e.Status, e.File, e.Cover, e.Size, e.Duration = service.VideoExportReady, p.ID+"/exports/x.mp4", p.ID+"/exports/x.jpg", 1234, 61.5
	require.NoError(t, exports.UpdateExport(ctx, e))
	got, err := exports.GetExport(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, service.VideoExportReady, got.Status)
	require.Equal(t, int64(1234), got.Size)
	require.Equal(t, 61.5, got.Duration)
	list, err := exports.ListExports(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	n, err := exports.CountExportsSince(ctx, user.ID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, n)

	raw, err := exports.GetPublishCopy(ctx, p.ID)
	require.NoError(t, err)
	require.Nil(t, raw)
	require.NoError(t, exports.SetPublishCopy(ctx, p.ID, []byte(`{"douyin":{"title":"一"}}`)))
	require.NoError(t, exports.SetPublishCopy(ctx, p.ID, []byte(`{"douyin":{"title":"二"}}`)))
	raw, err = exports.GetPublishCopy(ctx, p.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"douyin":{"title":"二"}}`, string(raw))

	// Deleting the project removes its exports and copy.
	ok, err := repo.Delete(ctx, user.ID, p.ID)
	require.NoError(t, err)
	require.True(t, ok)
	got, err = exports.GetExport(ctx, e.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestVideoRepositoryClones(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	a := mustCreateUser(t, integrationEntClient, &service.User{Email: "video-clone-a-" + suffix + "@video.test", Username: "vca" + suffix[len(suffix)-4:], Balance: 50})
	b := mustCreateUser(t, integrationEntClient, &service.User{Email: "video-clone-b-" + suffix + "@video.test", Username: "vcb" + suffix[len(suffix)-4:], Balance: 50})
	repo := NewVideoRepository(integrationDB).(service.VideoCloneRepository)
	slots := []string{"S_t1" + suffix, "S_t2" + suffix}

	c1 := &service.VideoVoiceClone{UserID: a.ID, Name: "A 的声音", Status: service.VideoCloneTraining, Trainings: 1, Price: 30, ConsentIP: "1.2.3.4"}
	require.NoError(t, repo.ClaimClone(ctx, c1, slots, "同意"))
	require.Equal(t, slots[0], c1.SpeakerID)
	c2 := &service.VideoVoiceClone{UserID: b.ID, Name: "B", Status: service.VideoCloneTraining, Trainings: 1, Price: 30}
	require.NoError(t, repo.ClaimClone(ctx, c2, slots, "同意"))
	require.Equal(t, slots[1], c2.SpeakerID)
	require.ErrorIs(t, repo.ClaimClone(ctx, &service.VideoVoiceClone{UserID: b.ID, Name: "C", Status: service.VideoCloneTraining}, slots, "同意"), service.ErrVideoCloneFull)

	// Ready: charged once, recorded with the TTS charges.
	charged, err := repo.MarkCloneReady(ctx, c1.ID)
	require.NoError(t, err)
	require.InDelta(t, 30, charged, 1e-9)
	charged, err = repo.MarkCloneReady(ctx, c1.ID)
	require.NoError(t, err)
	require.Zero(t, charged)
	got, err := repo.GetClone(ctx, c1.ID)
	require.NoError(t, err)
	require.Equal(t, service.VideoCloneReady, got.Status)
	require.True(t, got.Charged)
	require.Equal(t, "1.2.3.4", got.ConsentIP)
	balance, _ := NewVideoRepository(integrationDB).(service.VideoTTSBilling).Balance(ctx, a.ID)
	require.InDelta(t, 20, balance, 1e-9)
	var cost float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT SUM(cost) FROM video_tts_charges WHERE user_id = $1 AND project_id = 'voice-clone'`, a.ID).Scan(&cost))
	require.InDelta(t, 30, cost, 1e-9)

	// Retrained after being ready: back to training, never charged again.
	got.Status, got.Trainings = service.VideoCloneTraining, 2
	require.NoError(t, repo.UpdateClone(ctx, got))
	charged, err = repo.MarkCloneReady(ctx, c1.ID)
	require.NoError(t, err)
	require.Zero(t, charged)

	// An admin's disable sticks over the owner's updates.
	require.NoError(t, repo.SetCloneStatus(ctx, c1.ID, service.VideoCloneDisabled, "停用"))
	got.Status, got.Name = service.VideoCloneReady, "改名"
	require.NoError(t, repo.UpdateClone(ctx, got))
	got, _ = repo.GetClone(ctx, c1.ID)
	require.Equal(t, service.VideoCloneDisabled, got.Status)
	require.Equal(t, "改名", got.Name)
	admin, err := repo.AdminListClones(ctx, 500)
	require.NoError(t, err)
	found := false
	for _, x := range admin {
		if x.ID == c1.ID {
			found = true
			require.Equal(t, slots[0], x.SpeakerID)
			require.Equal(t, a.Email, x.UserEmail)
		}
	}
	require.True(t, found)

	// Only the owner deletes; the slot is free again, and the less-trained slot goes first.
	ok, err := repo.DeleteClone(ctx, b.ID, c1.ID)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = repo.DeleteClone(ctx, a.ID, c1.ID)
	require.NoError(t, err)
	require.True(t, ok)
	_, err = repo.GetClone(ctx, c1.ID)
	require.ErrorIs(t, err, service.ErrVideoCloneNotFound)
	list, err := repo.ListClones(ctx, a.ID)
	require.NoError(t, err)
	require.Empty(t, list)
	c3 := &service.VideoVoiceClone{UserID: b.ID, Name: "D", Status: service.VideoCloneTraining, Trainings: 1}
	require.NoError(t, repo.ClaimClone(ctx, c3, slots, "同意"))
	require.Equal(t, slots[0], c3.SpeakerID)
}
