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

func TestContentEngineRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "content-" + suffix + "@content.test", Username: "ce" + suffix[len(suffix)-4:]})
	t.Cleanup(func() {
		// content_* rows go with the user (ON DELETE CASCADE).
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, user.ID)
	})
	owner := user.ID
	repo := NewContentEngineRepository(integrationDB)

	// Settings: none yet, then upsert, then the daily mark once per day.
	row, err := repo.GetSettings(ctx, owner)
	require.NoError(t, err)
	require.Nil(t, row)
	st := service.ContentSettings{KeyID: 3, DailyTime: "09:00", DailyCount: 1, Platforms: []string{service.ContentPlatformZhihu}, NotifyType: "serverchan"}
	require.NoError(t, repo.SaveSettings(ctx, &service.ContentOwnerRow{OwnerUserID: owner, Settings: st, NotifyKeyEnc: "enc", Enabled: true}))
	row, err = repo.GetSettings(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, "enc", row.NotifyKeyEnc)
	require.Equal(t, []string{service.ContentPlatformZhihu}, row.Settings.Platforms)
	owners, err := repo.ListEnabledOwners(ctx)
	require.NoError(t, err)
	found := false
	for _, o := range owners {
		found = found || o.OwnerUserID == owner
	}
	require.True(t, found)
	ok, err := repo.MarkRun(ctx, owner, "2026-10-14")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.MarkRun(ctx, owner, "2026-10-14")
	require.NoError(t, err)
	require.False(t, ok, "only once a day")

	// Topics.
	t1 := &service.ContentTopic{OwnerUserID: owner, Title: "Base URL 要不要加 v1", Source: "geo", Score: 60, Status: service.ContentTopicPending}
	t2 := &service.ContentTopic{OwnerUserID: owner, Title: "Cherry Studio 配置", Source: "support", Score: 90, Status: service.ContentTopicPending}
	require.NoError(t, repo.CreateTopic(ctx, t1))
	require.NoError(t, repo.CreateTopic(ctx, t2))
	topics, err := repo.ListTopics(ctx, owner, service.ContentTopicPending, 10)
	require.NoError(t, err)
	require.Len(t, topics, 2)
	require.Equal(t, t2.ID, topics[0].ID, "best score first")
	got, err := repo.GetTopic(ctx, owner+100000, t1.ID)
	require.NoError(t, err)
	require.Nil(t, got, "another owner's topic is invisible")

	// An article from topic 1.
	topicID := t1.ID
	it := &service.ContentItem{OwnerUserID: owner, TopicID: &topicID, KeyID: 3, Status: service.ContentOutlining, Title: t1.Title,
		ContentItemData: service.ContentItemData{Brief: service.ContentBrief{Topic: t1.Title, Platforms: []string{service.ContentPlatformZhihu}},
			Sources: []service.ArticleSource{}, Events: []service.ArticleEvent{}}}
	require.NoError(t, repo.CreateItem(ctx, it))
	itemID := it.ID
	t1.Status, t1.ItemID = service.ContentTopicUsed, &itemID
	require.NoError(t, repo.UpdateTopic(ctx, t1))
	n, err := repo.CountActive(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	titles, err := repo.RecentTitles(ctx, owner, 10)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"Base URL 要不要加 v1", "Cherry Studio 配置"}, titles, "topic and article titles, de-duplicated")

	it.Status, it.Markdown, it.PromptTokens, it.LandingPath = service.ContentPending, "# 标题\n\n正文", 1200, "/learn/connect/base-url"
	it.Checks = []service.ContentCheck{{Level: "warn", Code: "repeat", Text: "x"}}
	require.NoError(t, repo.SaveItem(ctx, it))

	// Versions: upsert by (item, platform), save, published list.
	v := &service.ContentVariant{ItemID: it.ID, OwnerUserID: owner, Platform: service.ContentPlatformZhihu, Kind: "answer", Status: service.ContentVariantGenerating}
	require.NoError(t, repo.UpsertVariant(ctx, v))
	firstID := v.ID
	v2 := &service.ContentVariant{ItemID: it.ID, OwnerUserID: owner, Platform: service.ContentPlatformZhihu, Kind: "answer", Status: service.ContentVariantPending,
		Title: "标题", Body: "正文 {{link}}", ContentVariantData: service.ContentVariantData{Question: "问题？", Tags: []string{"API"}}}
	require.NoError(t, repo.UpsertVariant(ctx, v2))
	require.Equal(t, firstID, v2.ID, "same row")

	list, err := repo.ListItems(ctx, owner, service.ContentPending, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 1200, list[0].PromptTokens)
	require.Equal(t, "/learn/connect/base-url", list[0].LandingPath)
	require.Len(t, list[0].Checks, 1)
	require.Equal(t, []service.ContentVariantSummary{{Platform: service.ContentPlatformZhihu, Status: service.ContentVariantPending}}, list[0].Variants)
	require.Empty(t, list[0].Markdown, "lists leave out the article")

	published := time.Now().UTC().Truncate(time.Millisecond)
	v2.Status, v2.LinkCode, v2.PostURL, v2.PublishedAt = service.ContentVariantPublished, "ce1-zhihu", "https://www.zhihu.com/question/1", &published
	v2.PromoCodeID = 7
	require.NoError(t, repo.SaveVariant(ctx, v2))
	gv, err := repo.GetVariant(ctx, owner, v2.ID)
	require.NoError(t, err)
	require.Equal(t, "问题？", gv.Question)
	require.Equal(t, int64(7), gv.PromoCodeID)
	require.True(t, gv.PublishedAt.Equal(published))
	pubs, err := repo.ListPublished(ctx, owner, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, pubs, 1)
	require.Equal(t, "Base URL 要不要加 v1", pubs[0].ItemTitle)

	counts, err := repo.CountByStatus(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 1, counts[service.ContentPending])
	require.Equal(t, 1, counts["topics"])
	require.Equal(t, 1, counts["published"])

	// Rules.
	rule := &service.ContentRule{OwnerUserID: owner, Rule: "不要用宝子们", Origin: "manual", Enabled: true}
	require.NoError(t, repo.CreateRule(ctx, rule))
	require.NoError(t, repo.SetRuleEnabled(ctx, owner, rule.ID, false))
	rules, err := repo.ListRules(ctx, owner)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.False(t, rules[0].Enabled)
	require.NoError(t, repo.DeleteRule(ctx, owner, rule.ID))

	// A generating version and a running article left by a restart are failed.
	v3 := &service.ContentVariant{ItemID: it.ID, OwnerUserID: owner, Platform: service.ContentPlatformCSDN, Kind: "article", Status: service.ContentVariantGenerating}
	require.NoError(t, repo.UpsertVariant(ctx, v3))
	_, err = repo.FailGenerating(ctx, "重启了")
	require.NoError(t, err)
	gv, _ = repo.GetVariant(ctx, owner, v3.ID)
	require.Equal(t, service.ContentVariantFailed, gv.Status)
	require.Equal(t, "重启了", gv.Error)

	// Deleting the article puts its topic back in the pool.
	require.NoError(t, repo.DeleteItem(ctx, owner, it.ID))
	got, err = repo.GetTopic(ctx, owner, t1.ID)
	require.NoError(t, err)
	require.Equal(t, service.ContentTopicPending, got.Status)
	require.Nil(t, got.ItemID)
	gv, err = repo.GetVariant(ctx, owner, v2.ID)
	require.NoError(t, err)
	require.Nil(t, gv, "versions go with the article")
}
