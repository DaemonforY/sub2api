//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyPromptMatchesCanvasTaxonomy(t *testing.T) {
	poster := ClassifyPrompt(PromptTraitInput{Title: "足球主题电影海报", Prompt: "生成一张电影海报，主角是一只猫", Tags: []string{"海报", "gpt-image-2"}})
	require.Equal(t, "poster", poster.Scenes[0])
	require.Equal(t, "gpt-image-2", poster.Model)
	require.Equal(t, "zh", poster.Lang)
	require.False(t, poster.Sensitive)

	// Weak tags only count when nothing specific matched; the title decides here.
	weak := ClassifyPrompt(PromptTraitInput{Title: "Isometric diorama of a tiny cafe", Prompt: "An isometric 3D diorama", Tags: []string{"commerce"}})
	require.Equal(t, "3d", weak.Scenes[0])

	// "Article" must not count as art, and a quote collection is not a celebrity.
	article := ClassifyPrompt(PromptTraitInput{Title: "Article layout", Prompt: "名人名言卡片", Tags: []string{"publishing"}})
	require.Equal(t, []string{"infographic"}, article.Scenes)
	require.False(t, article.Sensitive)

	require.True(t, ClassifyPrompt(PromptTraitInput{Title: "President portrait", Prompt: "portrait of the president"}).Sensitive)
	require.True(t, ClassifyPrompt(PromptTraitInput{Title: "x", Prompt: "y", Tags: []string{"NSFW"}}).NSFW)
	require.True(t, ClassifyPrompt(PromptTraitInput{Title: "换装", Prompt: "请上传一张你的照片，然后换成汉服"}).NeedsReference)
	require.Equal(t, []string{"other"}, ClassifyPrompt(PromptTraitInput{Title: "zzz", Prompt: "zzz"}).Scenes)

	// Source scene hints: specific hints lead, catch-all hints follow rule matches.
	hinted := ClassifyPrompt(PromptTraitInput{Title: "Cinematic photo of a street", Prompt: "p", SceneHints: []string{"social"}})
	require.Equal(t, []string{"photo", "social"}, hinted.Scenes)
	strong := ClassifyPrompt(PromptTraitInput{Title: "Cinematic photo of a street", Prompt: "p", SceneHints: []string{"ecommerce"}})
	require.Equal(t, []string{"ecommerce", "photo"}, strong.Scenes)
	require.Equal(t, "nano-banana", ClassifyPrompt(PromptTraitInput{Title: "t", Prompt: "p", ModelHint: "A detailed prompt for Nano Banana Pro"}).Model)
}

func TestPromptDisplayTitle(t *testing.T) {
	require.Equal(t, "Keep me", PromptDisplayTitle("Keep me", "whatever", nil))
	require.Equal(t, "足球主题电影海报", PromptDisplayTitle("UI与界面", "请生成「足球主题电影海报」，要求……", []string{"UI与界面"}))
	require.Equal(t, "赛博朋克城市夜景", PromptDisplayTitle("其他", "生成一张赛博朋克城市夜景，霓虹灯", nil))
	require.Equal(t, "一只非常非常非常非常非常可爱的小猫咪在草…", PromptDisplayTitle("", "一只非常非常非常非常非常可爱的小猫咪在草地上玩耍", nil))
}

func TestPromptQualityPrefersUsableChinesePrompts(t *testing.T) {
	zh := PromptQualityScore("https://x/a.png", "中文标题", PromptTraits{Scenes: []string{"poster"}, Model: "gpt-image-2", Lang: "zh"})
	en := PromptQualityScore("https://x/a.png", "English", PromptTraits{Scenes: []string{"poster"}, Model: "nano-banana", Lang: "en"})
	bare := PromptQualityScore("", "English", PromptTraits{Scenes: []string{"other"}, Model: "unknown", Lang: "en"})
	require.Greater(t, zh, en)
	require.Greater(t, en, bare)
}

func TestPromptSyncParsesRegistryAndYouMind(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dist/sources/demo.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id": "demo:1", "title": "电商主图", "prompt": "白底产品图", "coverUrl": "../img/1.png", "tags": ["电商"], "createdAt": "2026-05-01T00:00:00Z"},
			{"title": "no id", "prompt": "second", "referenceImageUrls": ["https://cdn/x.png"], "imageModel": "gpt-image-2"},
			{"id": "demo:1", "title": "dup id", "prompt": "ignored"},
			{"id": 7, "title": "", "prompt": "missing title is skipped"},
			{"id": "demo:nsfw", "title": "x", "prompt": "y", "tags": ["nsfw"]}
		]`))
	})
	mux.HandleFunc("/references/manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"categories":[{"slug":"poster-flyer","title":"Poster / Flyer","file":"poster-flyer.json"},{"slug":"social-media-post","title":"Social Media Post","file":"social-media-post.json"},{"slug":"evil","file":"../secret.json"}]}`))
	})
	mux.HandleFunc("/references/poster-flyer.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id": 4031, "content": "A poster", "title": "New Year poster", "description": "for Nano Banana Pro", "sourceMedia": ["https://cms/a.jpg"], "needReferenceImages": true}]`))
	})
	mux.HandleFunc("/references/social-media-post.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id": 4031, "content": "A poster", "title": "New Year poster", "sourceMedia": ["https://cms/a.jpg"]}, {"id": 5, "content": "A selfie photo", "title": "Street photo", "sourceMedia": []}]`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	sync := NewPromptLibrarySyncService(nil, srv.Client())
	ctx := context.Background()

	items, err := sync.fetchRegistry(ctx, &PromptSource{ID: "demo", URL: srv.URL + "/dist/sources/demo.json"})
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, "demo:1", items[0].ExternalID)
	require.Equal(t, srv.URL+"/dist/img/1.png", items[0].CoverURL)
	require.Equal(t, "ecommerce", items[0].Scenes[0])
	require.NotNil(t, items[0].PublishedAt)
	require.Equal(t, "demo-0002", items[1].ExternalID, "same fallback id as the canvas source runtime")
	require.Equal(t, "https://cdn/x.png", items[1].CoverURL)
	require.Equal(t, PromptStatusHidden, items[2].Status)
	require.Equal(t, []string{"nsfw"}, items[2].AutoFlags)
	require.NotEmpty(t, items[0].SyncHash)
	require.NotEqual(t, items[0].SyncHash, items[1].SyncHash)

	ym, err := sync.fetchYouMind(ctx, &PromptSource{ID: "ym", Format: "youmind", URL: srv.URL + "/references/manifest.json"})
	require.NoError(t, err)
	require.Len(t, ym, 2)
	require.Equal(t, "4031", ym[0].ExternalID)
	require.Equal(t, []string{"poster", "social"}, ym[0].Scenes, "a prompt listed in two use cases keeps both")
	require.Equal(t, []string{"Poster / Flyer", "Social Media Post"}, ym[0].SourceTags)
	require.True(t, ym[0].NeedsReference)
	require.Equal(t, "nano-banana", ym[0].Model)
	require.Equal(t, "photo", ym[1].Scenes[0])
	require.Equal(t, "", ym[1].CoverURL)
}

type recommendRepo struct {
	PromptLibraryRepository
	history    []PromptUseHistory
	candidates []PromptItem
	popular    []PromptItem
	gotScenes  []string
}

func (r *recommendRepo) UserHistory(context.Context, int64, int) ([]PromptUseHistory, error) {
	return r.history, nil
}

func (r *recommendRepo) Candidates(_ context.Context, _ int64, _ string, scenes []string, _ int) ([]PromptItem, error) {
	if len(scenes) == 0 {
		return r.popular, nil
	}
	r.gotScenes = scenes
	return r.candidates, nil
}

func TestRecommendFollowsTheUsersScenes(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	repo := &recommendRepo{
		history: []PromptUseHistory{
			{ItemID: 1, Scenes: []string{"poster"}, Lang: "zh", Uses: 5, LastUsedAt: now.Add(-time.Hour)},
			{ItemID: 2, Scenes: []string{"poster", "brand"}, Lang: "zh", Uses: 1, Favorited: true, LastUsedAt: now.Add(-24 * time.Hour)},
			// Old activity fades: a year ago counts for almost nothing.
			{ItemID: 3, Scenes: []string{"3d"}, Uses: 10, LastUsedAt: now.Add(-365 * 24 * time.Hour)},
		},
		candidates: []PromptItem{
			{ID: 10, Scenes: []string{"3d"}, QualityScore: 10},
			{ID: 11, Scenes: []string{"poster"}, QualityScore: 4, Lang: "zh"},
			{ID: 12, Scenes: []string{"brand"}, QualityScore: 4},
		},
		popular: []PromptItem{{ID: 11}, {ID: 20}},
	}
	svc := NewPromptLibraryService(repo, nil)
	svc.now = func() time.Time { return now }

	res, err := svc.Recommend(context.Background(), 42, "image", 4)
	require.NoError(t, err)
	require.Equal(t, "history", res.Basis)
	require.Equal(t, "poster", res.Scenes[0].Scene)
	require.Equal(t, "poster", repo.gotScenes[0])
	require.Equal(t, int64(11), res.Items[0].ID, "the user's main scene beats a higher quality item in a faded one")
	ids := []int64{}
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	require.ElementsMatch(t, []int64{10, 11, 12, 20}, ids, "filled up with popular items, without repeats")

	cold := &recommendRepo{popular: []PromptItem{{ID: 1}, {ID: 2}}}
	res, err = NewPromptLibraryService(cold, nil).Recommend(context.Background(), 7, "image", 10)
	require.NoError(t, err)
	require.Equal(t, "popular", res.Basis)
	require.Len(t, res.Items, 2)
}

func TestPromptSharingGoesThroughReview(t *testing.T) {
	item := &PromptItem{Status: PromptStatusActive, Visibility: PromptVisibilityPrivate}
	setPromptSharing(item, true, false)
	require.Equal(t, PromptStatusPending, item.Status)
	require.Equal(t, PromptVisibilityPublic, item.Visibility)

	// Approved and unchanged: stays public; edited: back to review.
	item.Status = PromptStatusActive
	setPromptSharing(item, true, false)
	require.Equal(t, PromptStatusActive, item.Status)
	setPromptSharing(item, true, true)
	require.Equal(t, PromptStatusPending, item.Status)

	// Rejected, then made private: usable again by the owner.
	item.Status = PromptStatusRejected
	item.ReviewNote = "含真人照片"
	setPromptSharing(item, false, false)
	require.Equal(t, PromptStatusActive, item.Status)
	require.Equal(t, PromptVisibilityPrivate, item.Visibility)
	require.Empty(t, item.ReviewNote)

	// An admin-hidden item stays hidden when the owner flips it private.
	item.Status = PromptStatusHidden
	setPromptSharing(item, false, false)
	require.Equal(t, PromptStatusHidden, item.Status)
}

func TestPromptPublicViewHidesModeration(t *testing.T) {
	owner := int64(3)
	item := PromptItem{SourceID: PromptSourceUser, OwnerUserID: &owner, OwnerEmail: "a@b.c", AutoFlags: []string{"nsfw"}, ReviewNote: "x", Author: "a", Curated: true, QualityScore: 9}
	view := item.PublicView()
	require.Nil(t, view.OwnerUserID)
	require.Empty(t, view.OwnerEmail)
	require.Empty(t, view.AutoFlags)
	require.Empty(t, view.ReviewNote)
	require.Empty(t, view.Author)
	require.Zero(t, view.QualityScore)
	item.Mine = true
	require.Equal(t, "x", item.PublicView().ReviewNote, "owners see why their share was rejected")
}

func TestPromptCoverURLValidation(t *testing.T) {
	file := "0123456789abcdef0123456789abcdef.png"
	for _, url := range []string{PromptCoverPublicPrefix + file, "https://hivegpt.cn" + PromptCoverPublicPrefix + file} {
		got, ok := PromptCoverFileFromURL(url)
		require.True(t, ok, url)
		require.Equal(t, file, got)
	}
	for _, url := range []string{"", "https://evil/x.png", PromptCoverPublicPrefix + "../../etc/passwd", "javascript:" + PromptCoverPublicPrefix + file} {
		_, ok := PromptCoverFileFromURL(url)
		require.False(t, ok, url)
	}
}

type countingListRepo struct {
	PromptLibraryRepository
	lists int
}

func (r *countingListRepo) List(context.Context, PromptListQuery) ([]PromptItem, int64, error) {
	r.lists++
	return []PromptItem{{ID: 1, Scenes: []string{"poster"}}}, 1, nil
}
func (r *countingListRepo) SceneCounts(context.Context, PromptListQuery) (map[string]int64, error) {
	return map[string]int64{"poster": 1}, nil
}
func (r *countingListRepo) ListSources(context.Context) ([]PromptSource, error) { return nil, nil }

func TestPublicListingIsCachedBriefly(t *testing.T) {
	repo := &countingListRepo{}
	svc := NewPromptLibraryService(repo, nil)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	ctx := context.Background()
	_, err := svc.ListPublic(ctx, PromptListQuery{Keyword: "海报"})
	require.NoError(t, err)
	_, _ = svc.ListPublic(ctx, PromptListQuery{Keyword: " 海报 ", Page: 1})
	require.Equal(t, 1, repo.lists, "same normalized query is served from the cache")
	_, _ = svc.ListPublic(ctx, PromptListQuery{Keyword: "头像"})
	require.Equal(t, 2, repo.lists)
	now = now.Add(2 * time.Minute)
	_, _ = svc.ListPublic(ctx, PromptListQuery{Keyword: "海报"})
	require.Equal(t, 3, repo.lists, "entries expire")
}
