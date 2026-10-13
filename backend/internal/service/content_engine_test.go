//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// --- in-memory repository ----------------------------------------------------------------------

type contentRepoStub struct {
	mu       sync.Mutex
	next     int64
	settings map[int64]ContentOwnerRow
	topics   map[int64]ContentTopic
	items    map[int64]ContentItem
	variants map[int64]ContentVariant
	rules    map[int64]ContentRule
	personas map[int64]ContentPersona
	versions []ContentVersion
}

func newContentRepoStub() *contentRepoStub {
	return &contentRepoStub{settings: map[int64]ContentOwnerRow{}, topics: map[int64]ContentTopic{}, items: map[int64]ContentItem{},
		variants: map[int64]ContentVariant{}, rules: map[int64]ContentRule{}, personas: map[int64]ContentPersona{}}
}

func (r *contentRepoStub) id() int64 { r.next++; return r.next }

// deep copies through JSON, like a database round trip
func contentCloneT[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}

func (r *contentRepoStub) GetSettings(_ context.Context, owner int64) (*ContentOwnerRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.settings[owner]
	if !ok {
		return nil, nil
	}
	row.Settings = contentCloneT(row.Settings)
	return &row, nil
}
func (r *contentRepoStub) SaveSettings(_ context.Context, row *ContentOwnerRow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.settings[row.OwnerUserID]
	cp := *row
	cp.Settings = contentCloneT(row.Settings)
	cp.LastRunDay = old.LastRunDay
	r.settings[row.OwnerUserID] = cp
	return nil
}
func (r *contentRepoStub) ListEnabledOwners(context.Context) ([]ContentOwnerRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentOwnerRow{}
	for _, row := range r.settings {
		if row.Enabled {
			out = append(out, row)
		}
	}
	return out, nil
}
func (r *contentRepoStub) MarkRun(_ context.Context, owner int64, day string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.settings[owner]
	if row.LastRunDay == day {
		return false, nil
	}
	row.LastRunDay = day
	r.settings[owner] = row
	return true, nil
}
func (r *contentRepoStub) ListTopics(_ context.Context, owner int64, status string, limit int) ([]ContentTopic, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentTopic{}
	for _, t := range r.topics {
		if t.OwnerUserID == owner && (status == "" || t.Status == status) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (r *contentRepoStub) GetTopic(_ context.Context, owner, id int64) (*ContentTopic, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.topics[id]
	if !ok || t.OwnerUserID != owner {
		return nil, nil
	}
	return &t, nil
}
func (r *contentRepoStub) CreateTopic(_ context.Context, t *ContentTopic) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t.ID, t.CreatedAt = r.id(), time.Now()
	r.topics[t.ID] = *t
	return nil
}
func (r *contentRepoStub) UpdateTopic(_ context.Context, t *ContentTopic) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.topics[t.ID] = *t
	return nil
}
func (r *contentRepoStub) DeleteTopic(_ context.Context, owner, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.topics[id].OwnerUserID == owner {
		delete(r.topics, id)
	}
	return nil
}
func (r *contentRepoStub) RecentTitles(_ context.Context, owner int64, _ int) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []string{}
	for _, t := range r.topics {
		if t.OwnerUserID == owner {
			out = append(out, t.Title)
		}
	}
	return out, nil
}
func (r *contentRepoStub) CreateItem(_ context.Context, it *ContentItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	it.ID, it.CreatedAt = r.id(), time.Now()
	cp := contentCloneT(*it)
	cp.OwnerUserID = it.OwnerUserID
	r.items[it.ID] = cp
	return nil
}
func (r *contentRepoStub) GetItem(_ context.Context, owner, id int64) (*ContentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	it, ok := r.items[id]
	if !ok || it.OwnerUserID != owner {
		return nil, nil
	}
	it = contentCloneT(it)
	it.OwnerUserID = owner
	return &it, nil
}
func (r *contentRepoStub) ListItems(_ context.Context, owner int64, status string, _ int) ([]ContentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentItem{}
	for _, it := range r.items {
		if it.OwnerUserID == owner && (status == "" || it.Status == status) {
			out = append(out, it)
		}
	}
	return out, nil
}
func (r *contentRepoStub) SaveItem(_ context.Context, it *ContentItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := contentCloneT(*it)
	cp.OwnerUserID = it.OwnerUserID
	r.items[it.ID] = cp
	return nil
}
func (r *contentRepoStub) DeleteItem(_ context.Context, owner, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, id)
	return nil
}
func (r *contentRepoStub) CountActive(_ context.Context, owner int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, it := range r.items {
		switch it.Status {
		case ContentOutlining, ContentDrafting, ContentReviewing, ContentHumanizing, ContentAdapting:
			if it.OwnerUserID == owner {
				n++
			}
		}
	}
	return n, nil
}
func (r *contentRepoStub) FailActive(context.Context, string) (int64, error) { return 0, nil }
func (r *contentRepoStub) CountByStatus(_ context.Context, owner int64) (map[string]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]int{}
	for _, it := range r.items {
		if it.OwnerUserID == owner {
			out[it.Status]++
		}
	}
	return out, nil
}
func (r *contentRepoStub) ListVariants(_ context.Context, owner, itemID int64) ([]ContentVariant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentVariant{}
	for _, v := range r.variants {
		if v.OwnerUserID == owner && v.ItemID == itemID {
			cp := contentCloneT(v)
			cp.OwnerUserID = owner
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
func (r *contentRepoStub) GetVariant(_ context.Context, owner, id int64) (*ContentVariant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.variants[id]
	if !ok || v.OwnerUserID != owner {
		return nil, nil
	}
	cp := contentCloneT(v)
	cp.OwnerUserID = owner
	return &cp, nil
}
func (r *contentRepoStub) UpsertVariant(_ context.Context, v *ContentVariant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, old := range r.variants {
		if old.ItemID == v.ItemID && old.Platform == v.Platform {
			v.ID = id
		}
	}
	if v.ID == 0 {
		v.ID = r.id()
	}
	cp := contentCloneT(*v)
	cp.OwnerUserID = v.OwnerUserID
	r.variants[v.ID] = cp
	return nil
}
func (r *contentRepoStub) SaveVariant(_ context.Context, v *ContentVariant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := contentCloneT(*v)
	cp.OwnerUserID = v.OwnerUserID
	r.variants[v.ID] = cp
	return nil
}
func (r *contentRepoStub) ListPublished(_ context.Context, owner int64, _ time.Time) ([]ContentVariant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentVariant{}
	for _, v := range r.variants {
		if v.OwnerUserID == owner && v.PublishedAt != nil {
			out = append(out, contentCloneT(v))
		}
	}
	return out, nil
}
func (r *contentRepoStub) FailGenerating(context.Context, string) (int64, error) { return 0, nil }
func (r *contentRepoStub) FailDeskJobs(context.Context, string) (int64, error)   { return 0, nil }
func (r *contentRepoStub) GetPersona(_ context.Context, owner int64) (*ContentPersona, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.personas[owner]
	if !ok {
		return nil, nil
	}
	p = contentCloneT(p)
	return &p, nil
}
func (r *contentRepoStub) SavePersona(_ context.Context, owner int64, p *ContentPersona) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.personas[owner] = contentCloneT(*p)
	return nil
}
func (r *contentRepoStub) CreateVersion(_ context.Context, v *ContentVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v.ID, v.CreatedAt = r.id(), time.Now()
	r.versions = append(r.versions, *v)
	return nil
}
func (r *contentRepoStub) ListVersions(_ context.Context, owner, itemID int64) ([]ContentVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentVersion{}
	for i := len(r.versions) - 1; i >= 0; i-- {
		if v := r.versions[i]; v.OwnerUserID == owner && v.ItemID == itemID {
			v.Markdown = ""
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *contentRepoStub) GetVersion(_ context.Context, owner, id int64) (*ContentVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range r.versions {
		if v.ID == id && v.OwnerUserID == owner {
			return &v, nil
		}
	}
	return nil, nil
}
func (r *contentRepoStub) ListRules(_ context.Context, owner int64) ([]ContentRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ContentRule{}
	for _, x := range r.rules {
		if x.OwnerUserID == owner {
			out = append(out, x)
		}
	}
	return out, nil
}
func (r *contentRepoStub) CreateRule(_ context.Context, x *ContentRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	x.ID = r.id()
	r.rules[x.ID] = *x
	return nil
}
func (r *contentRepoStub) SetRuleEnabled(_ context.Context, owner, id int64, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	x := r.rules[id]
	x.Enabled = enabled
	r.rules[id] = x
	return nil
}
func (r *contentRepoStub) DeleteRule(_ context.Context, _, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rules, id)
	return nil
}

// --- other fakes -------------------------------------------------------------------------------

type contentLinkerStub struct {
	created []ChannelLinkInput
	stats   map[string]ChannelLink
}

func (l *contentLinkerStub) Create(_ context.Context, in ChannelLinkInput) (*ChannelLink, error) {
	l.created = append(l.created, in)
	return &ChannelLink{Code: in.Code, Source: in.Source, TargetPath: in.TargetPath}, nil
}
func (l *contentLinkerStub) Report(context.Context, int, float64) (*ChannelLinkReport, error) {
	out := &ChannelLinkReport{}
	for _, v := range l.stats {
		out.Links = append(out.Links, v)
	}
	return out, nil
}

type contentPromoStub struct {
	codes map[int64]*PromoCode
}

func (p *contentPromoStub) Create(_ context.Context, in *CreatePromoCodeInput) (*PromoCode, error) {
	id := int64(len(p.codes) + 100)
	pc := &PromoCode{ID: id, Code: in.Code, BonusAmount: in.BonusAmount, MaxUses: in.MaxUses, Notes: in.Notes}
	p.codes[id] = pc
	return pc, nil
}
func (p *contentPromoStub) GetByID(_ context.Context, id int64) (*PromoCode, error) {
	return p.codes[id], nil
}

type contentNotifierStub struct {
	mu   sync.Mutex
	sent []contentNotifyMessage
	keys []string
}

func (n *contentNotifierStub) Send(_ context.Context, kind, key string, msg contentNotifyMessage) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, msg)
	n.keys = append(n.keys, kind+":"+key)
	return nil
}

type contentEncryptorStub struct{}

func (contentEncryptorStub) Encrypt(s string) (string, error) { return "enc:" + s, nil }
func (contentEncryptorStub) Decrypt(s string) (string, error) {
	if !strings.HasPrefix(s, "enc:") {
		return "", errors.New("bad")
	}
	return strings.TrimPrefix(s, "enc:"), nil
}

const contentTestOwner = int64(5)

func newContentForTest(t *testing.T) (*ContentEngineService, *contentRepoStub, *contentLinkerStub, *contentPromoStub, *contentNotifierStub) {
	t.Helper()
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: contentTestOwner, Name: "内容", Key: "sk-owner", Status: StatusActive},
		{ID: 8, UserID: 6, Name: "别人的", Key: "sk-other", Status: StatusActive},
	}}
	learn, _ := newL2Service(t, "http://127.0.0.1:1", LearnSources{Keys: keys})
	repo := newContentRepoStub()
	svc := NewContentEngineService(repo, learn, nil, contentEncryptorStub{})
	links := &contentLinkerStub{stats: map[string]ChannelLink{}}
	promo := &contentPromoStub{codes: map[int64]*PromoCode{}}
	notifier := &contentNotifierStub{}
	svc.notifier = notifier
	svc.SetDeps(nil, links, promo, ContentSources{
		Search: func(string, int) []assistantHit {
			return []assistantHit{{URL: "/learn/connect/base-url", Title: "Base URL 怎么填", Text: "工具最终请求的地址必须是 /v1/chat/completions"}}
		},
		Catalog: func() []AssistantSource {
			return []AssistantSource{{Title: "Base URL 怎么填", URL: "/learn/connect/base-url"}}
		},
	})
	return svc, repo, links, promo, notifier
}

func waitContent(t *testing.T, svc *ContentEngineService, id int64) {
	t.Helper()
	require.Eventually(t, func() bool { return !svc.running(id) }, 5*time.Second, 5*time.Millisecond)
}

func contentLong(s string, n int) string { return strings.Repeat(s, n) }

// contentFakeAgent answers each step by what its system prompt asks for.
func contentFakeAgent(t *testing.T, calls *[]string, mu *sync.Mutex) articleAgentFunc {
	return func(_ context.Context, in AgentRunInput) (*AgentRunResult, error) {
		system := *in.Messages[0].Content
		res := &AgentRunResult{Model: "gpt-test", PromptTokens: 100, CompletionTokens: 50}
		step := ""
		switch {
		case strings.Contains(system, "策划"):
			step = "topics"
			res.Text = `{"topics":[{"title":"Cherry Studio 怎么接 GPT API","angle":"一步步配置","source":"support","reason":"用户常问","score":88},
{"title":"Base URL 要不要加 v1","angle":"对照表","source":"geo","reason":"高频","score":70}]}`
		case strings.Contains(system, "定角度、写大纲"):
			step = "outline"
			res.Text = "```json\n" + `{"title":"Base URL 要不要加 /v1","angle":"一张表说清","reader":"新手","hook":"看最终地址",
"sections":[{"heading":"判断方法","points":["拼起来等于 /v1/chat/completions"]},{"heading":"常见工具","points":["SDK 加 v1"]}],
"materials_needed":[],"landing_path":"/learn/connect/base-url"}` + "\n```"
		case strings.Contains(system, "完整的母稿"):
			step = "draft"
			res.Text = "# Base URL 要不要加 /v1\n\n" + contentLong("判断方法只有一条，看最终请求地址。", 40)
		case strings.Contains(system, "严格的技术编辑"):
			step = "review"
			res.Text = `{"notes":["第二节缺 NextChat 的例子"],"ai_score":40}`
		case strings.Contains(system, "去掉 AI 味"):
			step = "humanize"
			res.Text = "# Base URL 末尾的 /v1 到底加不加\n\n" + contentLong("先看工具最后拼出来的地址，再决定。", 40)
		case strings.Contains(system, "改写成下面这个平台的版本"):
			step = "adapt"
			switch {
			case strings.Contains(system, "平台：小红书"):
				res.Text = `{"kind":"note","title":"Base URL 填错？一招搞定","body":"` + contentLong("先看最终地址。", 40) + `详见 https://hivegpt.cn/learn","tags":["AI","GPT","API","编程","教程"],
"cards":[{"title":"封面","points":["一招"]},{"title":"判断","points":["看地址"]},{"title":"总结","points":["记住"]}]}`
			case strings.Contains(system, "平台：今日头条"):
				res.Text = `{"kind":"article","title":"配置 AI 工具总报错？多半是这个地址","body":"` + contentLong("头条正文，通俗地讲地址怎么拼。", 60) + `","micro":"一个小技巧：看最终地址。你踩过这个坑吗？"}`
			case strings.Contains(system, "平台：知乎"):
				res.Text = `{"kind":"answer","title":"Base URL 加不加 v1","question":"OpenAI 兼容接口的 Base URL 要不要加 /v1？","body":"` + contentLong("知乎回答：先给结论。", 50) + `完整对照表：{{link}}","tags":["OpenAI","API"]}`
			case strings.Contains(system, "平台：CSDN"):
				res.Text = `{"kind":"article","title":"Base URL 末尾要不要加 /v1：10 个工具填法对照","body":"` + contentLong("CSDN 教程正文，步骤和代码。", 80) + `{{link}}","tags":["OpenAI","Python","API"],"category":"人工智能"}`
			default:
				res.Text = `{"kind":"article","title":"Base URL 的 /v1，到底加不加","digest":"一张表说清","body":"` + contentLong("公众号长文，像和老读者聊天。", 90) + `"}`
			}
		default:
			t.Fatalf("unexpected prompt: %s", system[:80])
		}
		mu.Lock()
		*calls = append(*calls, step)
		mu.Unlock()
		return res, nil
	}
}

func saveContentSettings(t *testing.T, svc *ContentEngineService, mutate func(*ContentSettings)) ContentSettings {
	t.Helper()
	st := contentDefaultSettings()
	st.KeyID = 7
	st.NotifyType, st.NotifyKey = contentNotifyServerChan, "SCT12345abcdef"
	if mutate != nil {
		mutate(&st)
	}
	out, err := svc.SaveSettings(context.Background(), contentTestOwner, st)
	require.NoError(t, err)
	return out
}

// --- tests -------------------------------------------------------------------------------------

func TestContentSettingsValidationAndSecret(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _ := newContentForTest(t)

	st := contentDefaultSettings()
	st.KeyID = 8 // someone else's key
	_, err := svc.SaveSettings(ctx, contentTestOwner, st)
	require.ErrorIs(t, err, ErrContentKey)

	st.KeyID, st.DailyTime = 7, "9点"
	_, err = svc.SaveSettings(ctx, contentTestOwner, st)
	require.ErrorIs(t, err, ErrContentSettings)

	st.DailyTime, st.NotifyType, st.NotifyKey = "08:30", contentNotifyWecom, "https://evil.example/hook"
	_, err = svc.SaveSettings(ctx, contentTestOwner, st)
	require.ErrorIs(t, err, ErrContentNotifyURL)

	out := saveContentSettings(t, svc, func(s *ContentSettings) { s.Platforms = []string{"xiaohongshu", "nope", "zhihu"} })
	require.Empty(t, out.NotifyKey, "the push key is never returned")
	require.True(t, out.NotifyKeySet)
	require.Equal(t, "内容", out.KeyName)
	require.Equal(t, []string{ContentPlatformZhihu, ContentPlatformXHS}, out.Platforms, "unknown platforms dropped, display order kept")
	require.Equal(t, "enc:SCT12345abcdef", repo.settings[contentTestOwner].NotifyKeyEnc)

	// Saving again without a key keeps the stored one.
	st = out
	st.NotifyKey = ""
	out, err = svc.SaveSettings(ctx, contentTestOwner, st)
	require.NoError(t, err)
	require.True(t, out.NotifyKeySet)
}

func TestContentPipelineToReviewAndApprove(t *testing.T) {
	ctx := context.Background()
	svc, repo, links, promo, notifier := newContentForTest(t)
	var calls []string
	var mu sync.Mutex
	svc.agent = contentFakeAgent(t, &calls, &mu)
	saveContentSettings(t, svc, nil)

	topic, err := svc.CreateTopic(ctx, contentTestOwner, ContentTopicInput{Title: "Base URL 要不要加 v1", Notes: "实测：Cherry Studio 不用加"})
	require.NoError(t, err)
	it, err := svc.StartTopic(ctx, contentTestOwner, topic.ID)
	require.NoError(t, err)
	waitContent(t, svc, it.ID)

	detail, err := svc.Item(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	require.Equal(t, ContentPending, detail.Status, detail.Error)
	require.Equal(t, "Base URL 末尾的 /v1 到底加不加", detail.Title, "title from the final Markdown")
	require.Equal(t, "/learn/connect/base-url", detail.LandingPath)
	require.Equal(t, []string{"第二节缺 NextChat 的例子"}, detail.ReviewNotes)
	require.Len(t, detail.Versions, 5)
	require.Equal(t, ContentTopicUsed, repo.topics[topic.ID].Status)
	require.Equal(t, 1, strings.Count(strings.Join(calls, ","), "outline"))
	require.Equal(t, 5, strings.Count(strings.Join(calls, ","), "adapt"))
	require.Equal(t, 9*100, detail.PromptTokens, "every call counted")

	byPlatform := map[string]ContentVariant{}
	for _, v := range detail.Versions {
		require.Equal(t, ContentVariantPending, v.Status, v.Platform)
		byPlatform[v.Platform] = v
	}
	require.Equal(t, []string{ContentPlatformWechat, ContentPlatformZhihu, ContentPlatformCSDN, ContentPlatformToutiao, ContentPlatformXHS},
		[]string{detail.Versions[0].Platform, detail.Versions[1].Platform, detail.Versions[2].Platform, detail.Versions[3].Platform, detail.Versions[4].Platform})
	xhs := byPlatform[ContentPlatformXHS]
	require.NotContains(t, xhs.Body, "https://", "小红书 bodies lose their links")
	require.Contains(t, xhs.Body, contentPlaceholderPromo, "and get a promo line")
	require.Len(t, xhs.Cards, 3)
	require.Equal(t, "answer", byPlatform[ContentPlatformZhihu].Kind)
	require.NotEmpty(t, byPlatform[ContentPlatformZhihu].Question)

	// WeChat push for the review.
	require.Len(t, notifier.sent, 1)
	require.Contains(t, notifier.sent[0].Title, "待审核")
	require.Contains(t, notifier.sent[0].URL, "/admin/content?item=")
	require.Equal(t, "serverchan:SCT12345abcdef", notifier.keys[0])

	// Approve 知乎: a channel link to the landing page, no promo code.
	zh, err := svc.ApproveVariant(ctx, contentTestOwner, byPlatform[ContentPlatformZhihu].ID)
	require.NoError(t, err)
	require.Equal(t, ContentVariantApproved, zh.Status)
	require.Equal(t, "/learn/connect/base-url", links.created[0].TargetPath)
	require.Equal(t, "zhihu", links.created[0].Source)
	require.Equal(t, "answer", links.created[0].Medium)
	require.Empty(t, zh.PromoCode)
	require.Contains(t, zh.Rendered, "https://hivegpt.cn/go/"+zh.LinkCode)
	require.NotContains(t, zh.Rendered, contentPlaceholderLink)

	// Approve 小红书: a promo code too, shown in the body.
	x, err := svc.ApproveVariant(ctx, contentTestOwner, xhs.ID)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(x.PromoCode, "XHS"))
	require.Contains(t, x.Rendered, x.PromoCode)
	require.Equal(t, contentDefaultPromoBonus, promo.codes[x.PromoCodeID].BonusAmount)

	// Approving twice is refused.
	_, err = svc.ApproveVariant(ctx, contentTestOwner, xhs.ID)
	require.ErrorIs(t, err, ErrContentState)

	// Publish needs a URL; then the stats pick up the link's numbers and the promo uses.
	_, err = svc.PublishVariant(ctx, contentTestOwner, zh.ID, "zhihu.com/answer/1")
	require.ErrorIs(t, err, ErrContentPostURL)
	_, err = svc.PublishVariant(ctx, contentTestOwner, zh.ID, "https://www.zhihu.com/question/1/answer/2")
	require.NoError(t, err)
	_, err = svc.PublishVariant(ctx, contentTestOwner, x.ID, "https://www.xiaohongshu.com/explore/abc")
	require.NoError(t, err)
	links.stats[zh.LinkCode] = ChannelLink{Code: zh.LinkCode, Clicks: 40, Signups: 3, PaidUsers: 1, Revenue: 20}
	promo.codes[x.PromoCodeID].UsedCount = 2
	stats, err := svc.Stats(ctx, contentTestOwner, 30)
	require.NoError(t, err)
	require.Len(t, stats.Rows, 2)
	totals := map[string]ContentPlatformTotal{}
	for _, p := range stats.Platforms {
		totals[p.Platform] = p
	}
	require.EqualValues(t, 3, totals[ContentPlatformZhihu].Signups)
	require.Equal(t, 2, totals[ContentPlatformXHS].PromoUses)

	// Skip the rest: the article is done.
	for _, p := range []string{ContentPlatformWechat, ContentPlatformCSDN, ContentPlatformToutiao} {
		_, err := svc.SkipVariant(ctx, contentTestOwner, byPlatform[p].ID)
		require.NoError(t, err)
	}
	got, _ := svc.item(ctx, contentTestOwner, it.ID)
	require.Equal(t, ContentDone, got.Status)

	// Someone else sees nothing.
	_, err = svc.Item(ctx, 6, it.ID)
	require.ErrorIs(t, err, ErrContentNotFound)
	_, err = svc.ApproveVariant(ctx, 6, byPlatform[ContentPlatformCSDN].ID)
	require.ErrorIs(t, err, ErrContentNotFound)
}

func TestContentRetryResumesAtFailedStep(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _ := newContentForTest(t)
	var calls []string
	var mu sync.Mutex
	good := contentFakeAgent(t, &calls, &mu)
	failReview := true
	svc.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		if failReview && strings.Contains(*in.Messages[0].Content, "严格的技术编辑") {
			return nil, errLearnRunFailed("上游超时")
		}
		return good(ctx, in)
	}
	saveContentSettings(t, svc, func(s *ContentSettings) { s.Platforms = []string{ContentPlatformCSDN} })
	topic, _ := svc.CreateTopic(ctx, contentTestOwner, ContentTopicInput{Title: "Base URL"})
	it, err := svc.StartTopic(ctx, contentTestOwner, topic.ID)
	require.NoError(t, err)
	waitContent(t, svc, it.ID)
	got, _ := svc.item(ctx, contentTestOwner, it.ID)
	require.Equal(t, ContentFailed, got.Status)
	require.Contains(t, got.Error, "上游超时")
	require.NotEmpty(t, got.Draft, "the draft is kept")

	failReview = false
	calls = nil
	_, err = svc.Retry(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	waitContent(t, svc, it.ID)
	got, _ = svc.item(ctx, contentTestOwner, it.ID)
	require.Equal(t, ContentPending, got.Status, got.Error)
	require.Equal(t, []string{"review", "humanize", "adapt"}, calls, "outline and draft are not paid for again")
}

func TestContentRegenerateAndReject(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _ := newContentForTest(t)
	var calls []string
	var mu sync.Mutex
	inner := contentFakeAgent(t, &calls, &mu)
	var lastAdapt string
	svc.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		if strings.Contains(*in.Messages[0].Content, "改写成下面这个平台的版本") {
			mu.Lock()
			lastAdapt = *in.Messages[1].Content
			mu.Unlock()
		}
		return inner(ctx, in)
	}
	saveContentSettings(t, svc, func(s *ContentSettings) { s.Platforms = []string{ContentPlatformCSDN, ContentPlatformToutiao} })
	_, err := svc.CreateRule(ctx, contentTestOwner, "不要用「宝子们」")
	require.NoError(t, err)
	topic, _ := svc.CreateTopic(ctx, contentTestOwner, ContentTopicInput{Title: "Base URL"})
	it, _ := svc.StartTopic(ctx, contentTestOwner, topic.ID)
	waitContent(t, svc, it.ID)
	detail, _ := svc.Item(ctx, contentTestOwner, it.ID)
	require.Len(t, detail.Versions, 2)
	csdn := detail.Versions[0]

	_, err = svc.RegenerateVariant(ctx, contentTestOwner, csdn.ID, "标题要带版本号")
	require.NoError(t, err)
	waitContent(t, svc, it.ID)
	require.Contains(t, lastAdapt, "标题要带版本号")
	v, _ := svc.variant(ctx, contentTestOwner, csdn.ID)
	require.Equal(t, ContentVariantPending, v.Status)
	require.Equal(t, "标题要带版本号", v.Feedback)

	got, err := svc.Reject(ctx, contentTestOwner, it.ID, "选题太泛", true)
	require.NoError(t, err)
	require.Equal(t, ContentRejected, got.Status)
	rules, _ := svc.Rules(ctx, contentTestOwner)
	require.Len(t, rules, 2)
	require.Equal(t, "reject", rules[1].Origin)
	_ = repo
}

func TestContentEditRechecks(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _ := newContentForTest(t)
	repo.variants[1] = ContentVariant{ID: 1, ItemID: 9, OwnerUserID: contentTestOwner, Platform: ContentPlatformToutiao, Status: ContentVariantPending, Title: "t", Body: "b"}
	repo.next = 10
	v, err := svc.UpdateVariant(ctx, contentTestOwner, 1, ContentVariantInput{Title: "这个标题非常非常非常非常非常非常非常非常非常非常长超过三十个字了吧", Body: "看这里 www.example.com 注册",
		Micro: contentLong("微", 301)})
	require.NoError(t, err)
	require.True(t, v.Edited)
	require.NotContains(t, v.Body, "www.example.com")
	codes := map[string]string{}
	for _, c := range v.Checks {
		codes[c.Code] = c.Level
	}
	require.Equal(t, "error", codes["title_long"])
	require.Equal(t, "error", codes["micro_long"])
	require.Equal(t, "info", codes["links_removed"])
	require.Equal(t, "info", codes["promo_added"])
	require.Equal(t, "warn", codes["body_short"])

	_, err = svc.UpdateVariant(ctx, contentTestOwner, 1, ContentVariantInput{Title: "", Body: "x"})
	require.ErrorIs(t, err, ErrContentVariant)
}

func TestContentChecks(t *testing.T) {
	v := &ContentVariant{ID: 1, Platform: ContentPlatformZhihu, Kind: "answer", Title: "标题",
		Body: "值得注意的是，这是最稳定的方案。总而言之，赋能开发者。综上所述就这样。" + contentLong("内容", 300) + "【素材卡：实测截图】"}
	other := ContentVariant{ID: 2, Platform: ContentPlatformCSDN, Body: v.Body}
	checks := checkContentVariant(v, []ContentVariant{*v, other}, "HiveGPT")
	codes := map[string]bool{}
	for _, c := range checks {
		codes[c.Code] = true
	}
	for _, want := range []string{"ai_phrases", "risky_words", "material", "link_missing", "question_empty", "similar", "tags"} {
		require.True(t, codes[want], want)
	}
	// 知乎 keeps links.
	v2 := &ContentVariant{Platform: ContentPlatformZhihu, Title: "t", Body: "看 https://hivegpt.cn 和 {{link}}"}
	checkContentVariant(v2, nil, "HiveGPT")
	require.Contains(t, v2.Body, "https://hivegpt.cn")

	// Markdown links keep their text on platforms without links.
	body, n := stripContentLinks("看[完整教程](https://x.cn/a)和 {{link}}，或 www.x.cn/b")
	require.Equal(t, 3, n)
	require.Equal(t, "看完整教程和 ，或 ", body)

	require.Equal(t, "去 https://a/go/x 用 HG1", renderContentBody("去 {{link}} 用 {{promo}}", "https://a/go/x", "HG1"))
	require.Equal(t, "去 {{link}}", renderContentBody("去 {{link}}", "", ""))
	require.Greater(t, contentJaccard("Base URL 要不要加 v1", "Base URL 要不要加 /v1？"), 0.6)
	require.Less(t, contentJaccard("Base URL 要不要加 v1", "小红书封面怎么做"), 0.2)
}

func TestContentParseJSON(t *testing.T) {
	var v struct {
		A int `json:"a"`
	}
	require.NoError(t, contentParseJSON("好的：\n```json\n{\"a\": 3}\n```", &v))
	require.Equal(t, 3, v.A)
	require.Error(t, contentParseJSON("没有 JSON", &v))
	require.Equal(t, "# T\n\nx", contentCleanMarkdown("```markdown\n# T\n\nx\n```"))
}

func TestContentSuggestAndDailyRun(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _ := newContentForTest(t)
	var calls []string
	var mu sync.Mutex
	svc.agent = contentFakeAgent(t, &calls, &mu)
	saveContentSettings(t, svc, func(s *ContentSettings) {
		s.Enabled, s.DailyTime, s.DailyCount, s.Platforms = true, "09:00", 1, []string{ContentPlatformCSDN}
	})
	// An existing topic nearly the same as a suggestion keeps the suggestion out.
	_, _ = svc.CreateTopic(ctx, contentTestOwner, ContentTopicInput{Title: "Base URL 要不要加 /v1？", Score: ptrInt(10)})
	_ = repo.topics[1]
	_, _ = svc.SetTopicStatus(ctx, contentTestOwner, 1, ContentTopicDismissed)

	before := time.Date(2026, 10, 14, 0, 30, 0, 0, time.UTC) // 08:30 CST
	svc.now = func() time.Time { return before }
	svc.Tick(ctx)
	require.Empty(t, calls, "not yet time")

	at := before.Add(time.Hour) // 09:30 CST
	svc.now = func() time.Time { return at }
	svc.Tick(ctx)
	require.Equal(t, "2026-10-14", repo.settings[contentTestOwner].LastRunDay)
	topics, _ := svc.Topics(ctx, contentTestOwner, "all")
	var titles []string
	for _, tp := range topics {
		titles = append(titles, tp.Title)
	}
	require.Contains(t, titles, "Cherry Studio 怎么接 GPT API")
	require.Equal(t, 1, strings.Count(strings.Join(titles, "|"), "Base URL"), "the near-duplicate suggestion was dropped")

	items, _ := svc.Items(ctx, contentTestOwner, "")
	require.Len(t, items, 1, "one article a day")
	require.Equal(t, "Cherry Studio 怎么接 GPT API", items[0].Brief.Topic, "the best-scored pending topic")
	waitContent(t, svc, items[0].ID)

	n := len(calls)
	svc.Tick(ctx)
	require.Len(t, calls, n, "once a day")
}

func TestContentNotifiers(t *testing.T) {
	var got []struct {
		path, ctype, body string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = append(got, struct{ path, ctype, body string }{r.URL.Path, r.Header.Get("Content-Type"), string(b)})
		switch {
		case strings.HasPrefix(r.URL.Path, "/sc/"):
			_, _ = w.Write([]byte(`{"code":0}`))
		case r.URL.Path == "/pp":
			_, _ = w.Write([]byte(`{"code":200,"msg":"请求成功"}`))
		case r.URL.Path == "/bad":
			_, _ = w.Write([]byte(`{"code":40001,"message":"SendKey 无效"}`))
		default:
			_, _ = w.Write([]byte(`{"errcode":0}`))
		}
	}))
	defer srv.Close()
	n := newContentHTTPNotifier()
	n.serverChan, n.pushPlus, n.wecomPrefix = srv.URL+"/sc/%s", srv.URL+"/pp", srv.URL+"/"
	msg := contentNotifyMessage{Title: "【待审核】标题", Body: "正文", URL: "https://hivegpt.cn/admin/content?item=1"}
	ctx := context.Background()

	require.NoError(t, n.Send(ctx, contentNotifyServerChan, "SCT12345abc", msg))
	form, _ := url.ParseQuery(got[0].body)
	require.Equal(t, "/sc/SCT12345abc", got[0].path)
	require.Equal(t, "【待审核】标题", form.Get("title"))
	require.Contains(t, form.Get("desp"), "(https://hivegpt.cn/admin/content?item=1)")

	require.NoError(t, n.Send(ctx, contentNotifyPushPlus, "token12345", msg))
	require.Contains(t, got[1].body, `"template":"markdown"`)

	require.NoError(t, n.Send(ctx, contentNotifyWecom, srv.URL+"/hook?key=1", msg))
	require.Contains(t, got[2].body, `"msgtype":"markdown"`)

	require.ErrorIs(t, n.Send(ctx, contentNotifyWecom, "https://evil.example/x", msg), ErrContentNotifyURL)
	require.Error(t, n.Send(ctx, contentNotifyServerChan, "bad key!", msg))

	n.serverChan = srv.URL + "/bad?%s"
	err := n.Send(ctx, contentNotifyServerChan, "SCT12345abc", msg)
	require.ErrorContains(t, err, "SendKey 无效")
}
