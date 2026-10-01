//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func promptTestItem(id, title, prompt string, scenes []string, quality int) service.PromptItem {
	item := service.PromptItem{
		ExternalID: id, Kind: "image", Title: title, Prompt: prompt, CoverURL: "https://cdn.test/" + id + ".png",
		ReferenceImageURLs: []string{}, SourceTags: []string{"t"}, Scenes: scenes, Model: "gpt-image-2", Lang: "zh",
		AutoFlags: []string{}, Visibility: service.PromptVisibilityPublic, Status: service.PromptStatusActive,
		QualityScore: quality, DedupeKey: service.PromptDedupeKey(prompt),
	}
	item.SyncHash = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%v", id, title, prompt, scenes))))
	return item
}

func TestPromptLibraryRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	alice := mustCreateUser(t, integrationEntClient, &service.User{Email: "prompt-a-" + suffix + "@prompt.test", Username: "pa" + suffix[len(suffix)-6:]})
	bob := mustCreateUser(t, integrationEntClient, &service.User{Email: "prompt-b-" + suffix + "@prompt.test", Username: "pb" + suffix[len(suffix)-6:]})
	repo := NewPromptLibraryRepository(integrationDB)
	src := "itest-" + suffix
	other := "itest2-" + suffix
	for _, id := range []string{src, other} {
		_, err := integrationDB.ExecContext(ctx, `INSERT INTO prompt_sources (id, name, url) VALUES ($1, $1, 'https://x.test/a.json')`, id)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM prompt_items WHERE source_id IN ($1, $2)`, src, other)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM prompt_sources WHERE id IN ($1, $2)`, src, other)
	})

	marker := "itest" + suffix
	items := []service.PromptItem{
		promptTestItem("a", "海报 A", marker+" 海报提示词 A", []string{"poster"}, 10),
		promptTestItem("b", "电商 B", marker+" 电商提示词 B", []string{"ecommerce", "photo"}, 8),
		promptTestItem("c", "海报 C", marker+" 海报提示词 C", []string{"poster", "brand"}, 4),
	}
	require.NoError(t, repo.UpsertSourceItems(ctx, src, items))
	// The same prompt from a second source (without a cover) is kept as a duplicate.
	dup := promptTestItem("a-copy", "海报 A copy", marker+" 海报提示词 A", []string{"poster"}, 6)
	dup.CoverURL = ""
	require.NoError(t, repo.UpsertSourceItems(ctx, other, []service.PromptItem{dup}))
	require.NoError(t, repo.RefreshDuplicates(ctx))

	public := service.PromptListQuery{Keyword: marker, Sort: service.PromptSortPopular, Page: 1, PageSize: 10}
	list, total, err := repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(3), total, "the duplicate from the other source is not listed")
	require.Equal(t, []string{"a", "b", "c"}, externalIDs(list), "no usage yet: quality decides")
	counts, err := repo.SceneCounts(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(2), counts["poster"])
	require.Equal(t, int64(1), counts["photo"])
	withScene := public
	withScene.Scene = "brand"
	list, _, err = repo.List(ctx, withScene)
	require.NoError(t, err)
	require.Equal(t, []string{"c"}, externalIDs(list))

	byExternal := map[string]int64{}
	all, _, err := repo.List(ctx, service.PromptListQuery{Admin: true, Keyword: marker, Sort: service.PromptSortLatest, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, all, 4)
	for _, item := range all {
		byExternal[item.ExternalID] = item.ID
		if item.ExternalID == "a-copy" {
			require.Equal(t, service.PromptStatusDuplicate, item.Status)
		}
	}

	// Uses: a repeat within the window only counts for the user, not the global order.
	n, err := repo.RecordUse(ctx, alice.ID, byExternal["c"], time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	n, err = repo.RecordUse(ctx, alice.ID, byExternal["c"], time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	n, err = repo.RecordUse(ctx, bob.ID, byExternal["c"], time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	list, _, err = repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, "c", list[0].ExternalID, "most used first")

	fav, err := repo.SetFavorite(ctx, alice.ID, byExternal["b"], true)
	require.NoError(t, err)
	require.Equal(t, int64(1), fav)
	fav, err = repo.SetFavorite(ctx, alice.ID, byExternal["b"], true)
	require.NoError(t, err)
	require.Equal(t, int64(1), fav, "favoriting twice does not double count")
	fav, err = repo.SetFavorite(ctx, alice.ID, byExternal["b"], false)
	require.NoError(t, err)
	require.Equal(t, int64(0), fav)

	history, err := repo.UserHistory(ctx, alice.ID, 10)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, byExternal["b"], history[0].ItemID)
	require.Equal(t, 2, history[1].Uses)

	cands, err := repo.Candidates(ctx, alice.ID, "image", []string{"poster"}, 500)
	require.NoError(t, err)
	ids := externalIDs(cands)
	require.Contains(t, ids, "a")
	require.NotContains(t, ids, "c", "already used")
	require.NotContains(t, ids, "b", "not in the scene and already touched")

	// Admin edits stick across syncs; uncurated items follow the source.
	curatedItem, err := repo.Get(ctx, byExternal["a"])
	require.NoError(t, err)
	curatedItem.Title = "人工标题"
	curatedItem.Scenes = []string{"brand"}
	curatedItem.Curated = true
	require.NoError(t, repo.Update(ctx, curatedItem))
	resynced := []service.PromptItem{
		promptTestItem("a", "源标题", marker+" 海报提示词 A", []string{"poster"}, 10),
		promptTestItem("b", "电商 B 新", marker+" 电商提示词 B", []string{"ecommerce"}, 8),
		promptTestItem("c", "海报 C", marker+" 海报提示词 C", []string{"poster", "brand"}, 4),
	}
	resynced[0].SyncHash = "changed"
	require.NoError(t, repo.UpsertSourceItems(ctx, src, resynced))
	got, err := repo.Get(ctx, byExternal["a"])
	require.NoError(t, err)
	require.Equal(t, "人工标题", got.Title)
	require.Equal(t, []string{"brand"}, got.Scenes)
	got, err = repo.Get(ctx, byExternal["b"])
	require.NoError(t, err)
	require.Equal(t, "电商 B 新", got.Title)
	require.Equal(t, int64(0), got.FavoriteCount)

	// Translations: shown instead of the English title, searchable, dropped when the source renames it.
	enTitle := promptTestItem("en", "Cat poster "+suffix, marker+" english prompt", []string{"poster"}, 3)
	require.NoError(t, repo.UpsertSourceItems(ctx, src, []service.PromptItem{enTitle}))
	applied, err := repo.ApplyTitleTranslations(ctx, map[string]string{"Cat poster " + suffix: "猫咪海报" + suffix})
	require.NoError(t, err)
	require.Equal(t, int64(1), applied)
	list, _, err = repo.List(ctx, service.PromptListQuery{Keyword: "猫咪海报" + suffix, Page: 1, PageSize: 5, Sort: service.PromptSortLatest})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "猫咪海报"+suffix, list[0].Title)
	require.Equal(t, "Cat poster "+suffix, list[0].OriginalTitle)
	renamed := promptTestItem("en", "Dog poster "+suffix, marker+" english prompt", []string{"poster"}, 3)
	require.NoError(t, repo.UpsertSourceItems(ctx, src, []service.PromptItem{renamed}))
	got, err = repo.Get(ctx, list[0].ID)
	require.NoError(t, err)
	require.Equal(t, "Dog poster "+suffix, got.Title)
	require.Empty(t, got.OriginalTitle)
	require.NoError(t, repo.Delete(ctx, got.ID))

	// Bundled scene corrections apply to items no admin has edited, and only when they differ.
	changed, err := repo.ApplySceneOverrides(ctx, map[string][]string{src + ":b": {"photo", "ecommerce"}, src + ":a": {"video"}})
	require.NoError(t, err)
	require.Equal(t, int64(1), changed, "a is curated")
	got, _ = repo.Get(ctx, byExternal["b"])
	require.Equal(t, []string{"photo", "ecommerce"}, got.Scenes)
	changed, err = repo.ApplySceneOverrides(ctx, map[string][]string{src + ":b": {"photo", "ecommerce"}})
	require.NoError(t, err)
	require.Equal(t, int64(0), changed)

	// Model-checked scenes survive syncs of the same prompt and are checked again when the prompt changes.
	fresh := promptTestItem("m", "新条目 "+suffix, marker+" fresh prompt", []string{"social"}, 3)
	require.NoError(t, repo.UpsertSourceItems(ctx, src, []service.PromptItem{fresh}))
	unchecked, err := repo.UncheckedScenePrompts(ctx, 100000)
	require.NoError(t, err)
	var freshID int64
	for _, c := range unchecked {
		require.NotEqual(t, byExternal["a"], c.ID, "curated items are never re-labelled")
		require.NotEqual(t, byExternal["b"], c.ID, "bundled corrections count as checked")
		if c.Title == "新条目 "+suffix {
			freshID = c.ID
			require.Equal(t, []string{"social"}, c.Scenes)
		}
	}
	require.NotZero(t, freshID)
	changed, err = repo.ApplyCheckedScenes(ctx, map[int64][]string{freshID: {"3d", "portrait"}, byExternal["a"]: {"other"}})
	require.NoError(t, err)
	require.Equal(t, int64(1), changed)
	fresh.SyncHash = fmt.Sprintf("%x", sha256.Sum256([]byte(fresh.SyncHash+"again")))
	require.NoError(t, repo.UpsertSourceItems(ctx, src, []service.PromptItem{fresh}))
	got, _ = repo.Get(ctx, freshID)
	require.Equal(t, []string{"3d", "portrait"}, got.Scenes)
	fresh.Prompt = marker + " changed prompt"
	fresh.SyncHash = fmt.Sprintf("%x", sha256.Sum256([]byte(fresh.SyncHash+"changed")))
	require.NoError(t, repo.UpsertSourceItems(ctx, src, []service.PromptItem{fresh}))
	got, _ = repo.Get(ctx, freshID)
	require.Equal(t, []string{"social"}, got.Scenes)
	require.NoError(t, repo.Delete(ctx, freshID))

	// Batch edits keep scene order and fall back to {other}.
	updated, err := repo.Batch(ctx, []int64{byExternal["b"], byExternal["c"]}, service.PromptBatchOp{Action: "add_scenes", Scenes: []string{"poster", "3d"}})
	require.NoError(t, err)
	require.Equal(t, int64(2), updated)
	got, _ = repo.Get(ctx, byExternal["c"])
	require.Equal(t, []string{"poster", "brand", "3d"}, got.Scenes)
	require.True(t, got.Curated)
	_, err = repo.Batch(ctx, []int64{byExternal["c"]}, service.PromptBatchOp{Action: "remove_scenes", Scenes: []string{"poster", "brand", "3d"}})
	require.NoError(t, err)
	got, _ = repo.Get(ctx, byExternal["c"])
	require.Equal(t, []string{"other"}, got.Scenes)
	_, err = repo.Batch(ctx, []int64{byExternal["c"]}, service.PromptBatchOp{Action: "add_tags", Tags: []string{"国风", "手绘", "国风"}})
	require.NoError(t, err)
	got, _ = repo.Get(ctx, byExternal["c"])
	require.Equal(t, []string{"国风", "手绘"}, got.Tags)
	_, err = repo.Batch(ctx, []int64{byExternal["c"]}, service.PromptBatchOp{Action: "set_status", Status: service.PromptStatusHidden, Note: "质量差"})
	require.NoError(t, err)
	_, total, err = repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	_, total, err = repo.List(ctx, service.PromptListQuery{Admin: true, Tag: "国风", Keyword: marker, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)

	// Disabling a source hides its items; enabling restores them.
	require.NoError(t, repo.SetSourceEnabled(ctx, src, false))
	_, total, err = repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.NoError(t, repo.SetSourceEnabled(ctx, src, true))
	_, total, err = repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	// User prompts: private until shared and approved.
	owner := alice.ID
	mine := &service.PromptItem{SourceID: service.PromptSourceUser, ExternalID: "u-" + suffix, OwnerUserID: &owner, Kind: "image", Title: "我的", Prompt: marker + " 我的提示词",
		ReferenceImageURLs: []string{}, SourceTags: []string{}, Scenes: []string{"poster"}, Tags: []string{}, Model: "unknown", Lang: "zh", AutoFlags: []string{},
		Visibility: service.PromptVisibilityPrivate, Status: service.PromptStatusActive}
	require.NoError(t, repo.Insert(ctx, mine))
	t.Cleanup(func() { _ = repo.Delete(context.Background(), mine.ID) })
	_, total, err = repo.List(ctx, public)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	ownList, ownTotal, err := repo.List(ctx, service.PromptListQuery{OwnerUserID: alice.ID, Page: 1, PageSize: 10, Sort: service.PromptSortLatest})
	require.NoError(t, err)
	require.Equal(t, int64(1), ownTotal)
	require.Equal(t, "社区分享", ownList[0].SourceName)
	count, err := repo.CountOwned(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	stats, err := repo.Stats(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.StatusCounts[service.PromptStatusActive], int64(3))

	// Covers are owned per user and counted per day.
	file := fmt.Sprintf("%032x.png", time.Now().UnixNano())
	require.NoError(t, repo.InsertCover(ctx, file, alice.ID, 100))
	ok, err := repo.CoverOwnedBy(ctx, file, alice.ID)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.CoverOwnedBy(ctx, file, bob.ID)
	require.NoError(t, err)
	require.False(t, ok)
	recent, err := repo.CountCoversSince(ctx, alice.ID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, recent)
}

func externalIDs(items []service.PromptItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ExternalID)
	}
	return out
}
