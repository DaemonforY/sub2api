//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestMaterialRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMaterialRepository(integrationDB)
	tag := fmt.Sprintf("it-mat-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM mat_texts WHERE slug LIKE $1`, tag+"%")
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM mat_authors WHERE name LIKE $1`, tag+"%")
	})

	// Authors are found by (name, dynasty).
	authorID, err := repo.UpsertAuthor(ctx, &service.MaterialAuthor{Name: tag + "王维", Dynasty: "唐"})
	require.NoError(t, err)
	again, err := repo.UpsertAuthor(ctx, &service.MaterialAuthor{Name: tag + "王维", Dynasty: "唐"})
	require.NoError(t, err)
	require.Equal(t, authorID, again)

	body := "单车欲问边，属国过居延。\n征蓬出汉塞，归雁入胡天。\n大漠孤烟直，长河落日圆。\n萧关逢候骑，都护在燕然。"
	segments := service.SplitMaterialSegments("shi", body, "大漠孤烟直，长河落日圆")
	require.Len(t, segments, 4)
	workID := time.Now().UnixNano()
	text := &service.MaterialText{Slug: tag + "-saishang", Kind: "shi", Title: "使至塞上", AuthorID: &authorID, Body: body,
		LicenseStatus: service.MaterialLicensePublicDomain, Status: service.MaterialStatusPublished, ShihuaWorkID: &workID}
	created, err := repo.ImportText(ctx, text, segments)
	require.NoError(t, err)
	require.True(t, created)
	require.NotZero(t, text.ID)

	// Importing the same 课本诗画 work again writes nothing.
	dup := *text
	dup.Slug = tag + "-other"
	created, err = repo.ImportText(ctx, &dup, segments)
	require.NoError(t, err)
	require.False(t, created)

	got, err := repo.GetText(ctx, text.ID)
	require.NoError(t, err)
	require.Equal(t, tag+"王维", got.AuthorName)
	require.Equal(t, "唐", got.Dynasty)
	require.Len(t, got.Segments, 4)
	require.Equal(t, "第3联", got.Segments[2].Label)
	require.True(t, got.Segments[2].IsFamous)
	require.False(t, got.Segments[0].IsFamous)
	require.JSONEq(t, `[]`, string(got.Textbook))

	list, total, err := repo.ListTexts(ctx, service.MaterialTextQuery{Q: tag, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, 4, list[0].SegmentCount)

	missing, err := repo.GetText(ctx, -1)
	require.NoError(t, err)
	require.Nil(t, missing)

	// reference_only texts keep no words: the CHECK refuses a body, the trigger refuses segment content.
	song := &service.MaterialText{Slug: tag + "-song", Kind: "song", Title: "测试歌曲", Body: "原文",
		LicenseStatus: service.MaterialLicenseReference, Status: service.MaterialStatusDraft}
	_, err = repo.ImportText(ctx, song, nil)
	require.Error(t, err)
	song.Body = ""
	_, err = repo.ImportText(ctx, song, []service.MaterialSegment{{Seq: 1, Label: "副歌", Content: "原文"}})
	require.Error(t, err)
	created, err = repo.ImportText(ctx, song, []service.MaterialSegment{{Seq: 1, Label: "副歌", SceneDesc: "雨夜街灯下的背影"}})
	require.NoError(t, err)
	require.True(t, created)

	// Songs must be reference_only.
	_, err = repo.ImportText(ctx, &service.MaterialText{Slug: tag + "-song2", Kind: "song", Title: "测试歌曲二",
		LicenseStatus: service.MaterialLicensePublicDomain, Status: service.MaterialStatusDraft}, nil)
	require.Error(t, err)

	// A text with segment words cannot be switched to reference_only.
	_, err = integrationDB.ExecContext(ctx, `UPDATE mat_texts SET license_status = 'reference_only', body = '' WHERE id = $1`, text.ID)
	require.Error(t, err)

	// Seeds.
	styles, err := repo.ListStyles(ctx, false)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(styles), 9)
	require.Equal(t, "工笔青花", styles[0].Name)
	terms, err := repo.ListBlocklist(ctx, true)
	require.NoError(t, err)
	res := service.CheckMaterialBlocklist("a quiet village, Ghibli style", "prompt", terms)
	require.True(t, res.Blocked)
}

func TestMaterialImageryRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewMaterialRepository(integrationDB)
	tag := fmt.Sprintf("im%d", time.Now().UnixNano()%1_000_000)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM mat_texts WHERE slug LIKE $1`, tag+"%")
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM mat_imagery WHERE name LIKE $1`, tag+"%")
	})

	by := int64(3)
	smoke := &service.MaterialImagery{Name: tag + "烟", Aliases: []string{"孤烟"}, Category: "气候", VisualEn: "a column of smoke",
		Emotions: []string{"壮阔"}, Status: service.MaterialStatusDraft, CreatedBy: &by}
	created, err := repo.CreateImagery(ctx, smoke)
	require.NoError(t, err)
	require.True(t, created)
	require.NotZero(t, smoke.ID)
	dup := &service.MaterialImagery{Name: smoke.Name, Status: service.MaterialStatusDraft}
	created, err = repo.CreateImagery(ctx, dup)
	require.NoError(t, err)
	require.False(t, created)
	require.Zero(t, dup.ID)
	sun := &service.MaterialImagery{Name: tag + "日", Category: "天象", Status: service.MaterialStatusDraft}
	_, err = repo.CreateImagery(ctx, sun)
	require.NoError(t, err)

	smoke.Meaning, smoke.Seasons, smoke.StyleIDs = "边塞的烽烟", []string{"秋"}, []int64{2, 5}
	ok, err := repo.UpdateImagery(ctx, smoke)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.UpdateImagery(ctx, &service.MaterialImagery{ID: -1, Name: "x", Status: "draft"})
	require.NoError(t, err)
	require.False(t, ok)

	n, err := repo.SetImageryStatus(ctx, []int64{smoke.ID, sun.ID}, service.MaterialStatusPublished)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	list, total, err := repo.ListImagery(ctx, service.MaterialImageryQuery{Q: tag, Status: service.MaterialStatusPublished, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Equal(t, "边塞的烽烟", list[0].Meaning)
	require.Equal(t, []string{"孤烟"}, list[0].Aliases)
	require.Equal(t, []string{"秋"}, list[0].Seasons)
	require.Equal(t, []int64{2, 5}, list[0].StyleIDs)
	require.Equal(t, []int64{}, list[1].StyleIDs)
	byAlias, total, err := repo.ListImagery(ctx, service.MaterialImageryQuery{Q: "孤烟", Limit: 500})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 1)
	require.NotEmpty(t, byAlias)
	all, err := repo.AllImagery(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(all), 2)

	// A segment to tag.
	text := &service.MaterialText{Slug: tag + "-t", Kind: "shi", Title: "测试", Body: "大漠孤烟直，长河落日圆。",
		LicenseStatus: service.MaterialLicensePublicDomain, Status: service.MaterialStatusDraft}
	_, err = repo.ImportText(ctx, text, []service.MaterialSegment{{Seq: 1, Label: "第1联", Content: "大漠孤烟直，长河落日圆。"}})
	require.NoError(t, err)
	got, err := repo.GetText(ctx, text.ID)
	require.NoError(t, err)
	segID := got.Segments[0].ID

	seg, err := repo.GetSegment(ctx, segID)
	require.NoError(t, err)
	require.Equal(t, "测试", seg.Title)
	require.Equal(t, service.MaterialLicensePublicDomain, seg.LicenseStatus)
	require.Equal(t, "大漠孤烟直，长河落日圆。", seg.Body)
	missing, err := repo.GetSegment(ctx, -1)
	require.NoError(t, err)
	require.Nil(t, missing)
	require.NoError(t, repo.UpdateSegmentScene(ctx, segID, "黄昏的大漠"))

	// A manual confirmed tag survives ReplaceAISegmentImagery; unconfirmed AI tags are replaced.
	require.NoError(t, repo.SetSegmentImagery(ctx, segID, []service.MaterialSegmentImagery{
		{ImageryID: sun.ID, Weight: 1, Source: "manual", Confirmed: true},
		{ImageryID: smoke.ID, Weight: 1, Source: "ai"},
	}))
	require.NoError(t, repo.ReplaceAISegmentImagery(ctx, segID, []service.MaterialSegmentImagery{
		{ImageryID: smoke.ID, Weight: 3, Source: "ai"},
		{ImageryID: sun.ID, Weight: 2, Source: "ai"},
	}))
	links, err := repo.ListSegmentImagery(ctx, segID)
	require.NoError(t, err)
	require.Len(t, links, 2)
	require.Equal(t, smoke.ID, links[0].ImageryID)
	require.Equal(t, 3, links[0].Weight)
	require.Equal(t, tag+"烟", links[0].Name)
	require.Equal(t, "manual", links[1].Source)
	require.True(t, links[1].Confirmed)

	list, _, err = repo.ListImagery(ctx, service.MaterialImageryQuery{Q: tag, Limit: 10})
	require.NoError(t, err)
	for _, im := range list {
		require.Equal(t, 1, im.SegmentCount)
	}

	require.NoError(t, repo.SetSegmentImagery(ctx, segID, nil))
	links, err = repo.ListSegmentImagery(ctx, segID)
	require.NoError(t, err)
	require.Empty(t, links)
}

func TestMaterialArtworkQueue(t *testing.T) {
	ctx := context.Background()
	repo := NewMaterialRepository(integrationDB)
	tag := fmt.Sprintf("aq%d", time.Now().UnixNano()%1_000_000)
	var artIDs []int64
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM mat_artworks WHERE id = ANY($1)`, pq.Array(artIDs))
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM mat_texts WHERE slug LIKE $1`, tag+"%")
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM mat_imagery WHERE name LIKE $1`, tag+"%")
	})
	// Other tests' drawings must not be claimed here: this test only runs on its own rows.
	var before int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mat_artworks WHERE status IN ('queued','running')`).Scan(&before))
	require.Zero(t, before, "the queue must start empty")

	text := &service.MaterialText{Slug: tag + "-t", Kind: "shi", Title: "使至塞上", Body: "大漠孤烟直，长河落日圆。",
		LicenseStatus: service.MaterialLicensePublicDomain, Status: service.MaterialStatusDraft}
	_, err := repo.ImportText(ctx, text, []service.MaterialSegment{{Seq: 1, Label: "第3联", Content: "大漠孤烟直，长河落日圆。"}})
	require.NoError(t, err)
	got, err := repo.GetText(ctx, text.ID)
	require.NoError(t, err)
	segID := got.Segments[0].ID
	im := &service.MaterialImagery{Name: tag + "漠", Status: service.MaterialStatusPublished}
	_, err = repo.CreateImagery(ctx, im)
	require.NoError(t, err)
	styles, err := repo.ListStyles(ctx, false)
	require.NoError(t, err)
	styleID := styles[1].ID

	by := int64(5)
	mk := func() *service.MaterialArtwork {
		return &service.MaterialArtwork{SegmentID: &segID, StyleID: &styleID, Options: json.RawMessage(`{"mood":"冷白月光"}`),
			PromptFinal: "prompt", PromptParts: json.RawMessage(`{"frame":"f"}`), Model: "gpt-image-2", Size: "1024x1536",
			Quality: "high", Status: service.MaterialArtworkQueued, Source: "library", ReviewStatus: service.MaterialReviewPending,
			TextOverlayAllowed: true, CreatedBy: &by, ImageryIDs: []int64{im.ID, -42}}
	}
	arts := []*service.MaterialArtwork{mk(), mk(), mk()}
	require.NoError(t, repo.EnqueueArtworks(ctx, arts))
	for _, a := range arts {
		require.NotZero(t, a.ID)
		artIDs = append(artIDs, a.ID)
	}

	// Two claims at once never get the same drawing.
	var wg sync.WaitGroup
	claimed := make([]*service.MaterialArtwork, 2)
	for i := range claimed {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a, err := repo.ClaimNextArtwork(ctx)
			require.NoError(t, err)
			claimed[i] = a
		}(i)
	}
	wg.Wait()
	require.NotNil(t, claimed[0])
	require.NotNil(t, claimed[1])
	require.NotEqual(t, claimed[0].ID, claimed[1].ID)
	for _, c := range claimed {
		require.Equal(t, service.MaterialArtworkRunning, c.Status)
		require.Equal(t, 1, c.Attempts)
		require.NotNil(t, c.StartedAt)
		require.Equal(t, []int64{im.ID}, c.ImageryIDs, "unknown imagery is skipped")
		require.JSONEq(t, `{"mood":"冷白月光"}`, string(c.Options))
	}

	stats, err := repo.ArtworkQueueStats(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, service.MaterialQueueStats{Queued: 1, Running: 2, StartedToday: 2}, stats)

	// One succeeds, one is put back with a not_before in the future: it is not claimed yet.
	now := time.Now()
	ok := claimed[0]
	ok.Status, ok.File, ok.Width, ok.Height, ok.CostUSD, ok.DurationMs, ok.FinishedAt = service.MaterialArtworkSucceeded, "artworks/0/1.png", 1024, 1536, 0.25, 30000, &now
	require.NoError(t, repo.FinishArtwork(ctx, ok))
	later := now.Add(time.Hour)
	retry := claimed[1]
	retry.Status, retry.NotBefore, retry.Error = service.MaterialArtworkQueued, &later, "HTTP 503"
	require.NoError(t, repo.FinishArtwork(ctx, retry))
	third, err := repo.ClaimNextArtwork(ctx)
	require.NoError(t, err)
	require.NotNil(t, third)
	require.NotEqual(t, retry.ID, third.ID)
	none, err := repo.ClaimNextArtwork(ctx)
	require.NoError(t, err)
	require.Nil(t, none, "the retried one is not due")

	// A restart puts the running one back.
	n, err := repo.RequeueRunningArtworks(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	// A finish that arrives after a cancel does not resurrect the drawing.
	n, err = repo.CancelQueuedArtworks(ctx, []int64{third.ID, ok.ID})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	third.Status = service.MaterialArtworkSucceeded
	require.NoError(t, repo.FinishArtwork(ctx, third))
	c, err := repo.GetArtwork(ctx, third.ID)
	require.NoError(t, err)
	require.Equal(t, service.MaterialArtworkCanceled, c.Status)

	// Retry works on canceled / failed only.
	done, err := repo.RetryArtwork(ctx, third.ID)
	require.NoError(t, err)
	require.True(t, done)
	done, err = repo.RetryArtwork(ctx, ok.ID)
	require.NoError(t, err)
	require.False(t, done)

	// Review and the joined list.
	reviewed, err := repo.ReviewArtwork(ctx, ok.ID, service.MaterialReviewApproved, 5, "")
	require.NoError(t, err)
	require.True(t, reviewed)
	reviewed, err = repo.ReviewArtwork(ctx, -1, service.MaterialReviewApproved, 5, "")
	require.NoError(t, err)
	require.False(t, reviewed)
	items, total, err := repo.ListArtworks(ctx, service.MaterialArtworkQuery{TextID: text.ID, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Equal(t, third.ID, items[0].ID, "newest first")
	g, err := repo.GetArtwork(ctx, ok.ID)
	require.NoError(t, err)
	require.Equal(t, service.MaterialArtworkSucceeded, g.Status)
	require.Equal(t, service.MaterialReviewApproved, g.ReviewStatus)
	require.Equal(t, 5, g.Rating)
	require.Equal(t, "使至塞上", g.TextTitle)
	require.Equal(t, "第3联", g.SegmentLabel)
	require.Equal(t, styles[1].Name, g.StyleName)
	require.Equal(t, text.ID, *g.TextID)
	require.InDelta(t, 0.25, g.CostUSD, 1e-9)
	approved, total, err := repo.ListArtworks(ctx, service.MaterialArtworkQuery{SegmentID: segID, ReviewStatus: service.MaterialReviewApproved, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, ok.ID, approved[0].ID)
	missing, err := repo.GetArtwork(ctx, -1)
	require.NoError(t, err)
	require.Nil(t, missing)

	// Deleting the text keeps the drawings, without their segment.
	_, err = integrationDB.ExecContext(ctx, `DELETE FROM mat_texts WHERE id = $1`, text.ID)
	require.NoError(t, err)
	g, err = repo.GetArtwork(ctx, ok.ID)
	require.NoError(t, err)
	require.Nil(t, g.SegmentID)
	require.Empty(t, g.TextTitle)
}

func TestMaterialArtworkRenders(t *testing.T) {
	ctx := context.Background()
	repo := NewMaterialRepository(integrationDB)
	var artID int64
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM mat_artworks WHERE id = $1`, artID)
	})
	arts := []*service.MaterialArtwork{{PromptFinal: "p", Model: "gpt-image-2", Size: "1024x1536", Quality: "high",
		Status: service.MaterialArtworkSucceeded, Source: "library", ReviewStatus: service.MaterialReviewApproved}}
	require.NoError(t, repo.EnqueueArtworks(ctx, arts))
	artID = arts[0].ID

	r := &service.MaterialRender{ArtworkID: artID, Aspect: "2x3", File: "artworks/0/1-2x3.png", Width: 40, Height: 60,
		Layout: json.RawMessage(`{"text":"大漠孤烟直，长河落日圆。"}`)}
	require.NoError(t, repo.SaveArtworkRender(ctx, r))
	require.NotZero(t, r.ID)
	again := &service.MaterialRender{ArtworkID: artID, Aspect: "2x3", File: "artworks/0/1-2x3.jpg", Width: 44, Height: 66, Layout: nil}
	require.NoError(t, repo.SaveArtworkRender(ctx, again))
	require.Equal(t, r.ID, again.ID, "the same aspect is replaced")
	require.NoError(t, repo.SaveArtworkRender(ctx, &service.MaterialRender{ArtworkID: artID, Aspect: "16x9", File: "artworks/0/1-16x9.png"}))

	list, err := repo.ListArtworkRenders(ctx, []int64{artID, -1})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "16x9", list[0].Aspect)
	require.Equal(t, "artworks/0/1-2x3.jpg", list[1].File)
	require.Equal(t, 44, list[1].Width)
	require.JSONEq(t, `{}`, string(list[1].Layout))

	// Deleting the drawing deletes its posters.
	_, err = integrationDB.ExecContext(ctx, `DELETE FROM mat_artworks WHERE id = $1`, artID)
	require.NoError(t, err)
	list, err = repo.ListArtworkRenders(ctx, []int64{artID})
	require.NoError(t, err)
	require.Empty(t, list)
}
