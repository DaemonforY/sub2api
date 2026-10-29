package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 内容引擎 (docs/CONTENT_ENGINE_PLAN.md, P0): turns topics into a master article and one version per
// platform — 公众号, 知乎, CSDN, 今日头条, 小红书 — and stops at 待审核.
//
//   - 选题池: topics come from the support assistant's questions, the GEO questions, the learning
//     site and the web (one model call suggests and scores them), or are typed in by the owner.
//   - Pipeline (content_engine_pipeline.go): outline → draft → review → humanize (去 AI 味, with the
//     owner's style prompt and 写作规则) → one version per platform → automatic checks
//     (content_engine_check.go). Every call goes through this site's own gateway with one of the
//     owner's own API keys, in the background; a failed run resumes from the step that failed.
//   - Review: the owner edits, approves or skips each version, or rejects the whole article with a
//     reason that can become a 写作规则. Approving a version creates its channel link (and a promo
//     code on 头条 / 小红书, which allow no links), so every post is tied to visitors, sign-ups and
//     payments. P0 publishes by hand: the owner copies the version out and records the post URL.
//   - Daily run: at the owner's time the best pending topics start; a WeChat push says when an
//     article is waiting for review (content_engine_notify.go).
//
// Everything is keyed by owner so the same code serves users later; today the routes are admin-only.

const (
	ContentPlatformWechat  = "wechat"
	ContentPlatformZhihu   = "zhihu"
	ContentPlatformCSDN    = "csdn"
	ContentPlatformToutiao = "toutiao"
	ContentPlatformXHS     = "xiaohongshu"

	ContentOutlining  = "outlining"
	ContentDrafting   = "drafting"
	ContentReviewing  = "reviewing"
	ContentHumanizing = "humanizing"
	ContentAdapting   = "adapting"
	ContentPending    = "pending_review"
	ContentDone       = "done"
	ContentFailed     = "failed"
	ContentRejected   = "rejected"
	ContentCanceled   = "canceled"

	ContentVariantPending    = "pending_review"
	ContentVariantGenerating = "generating"
	ContentVariantApproved   = "approved"
	ContentVariantPublished  = "published"
	ContentVariantSkipped    = "skipped"
	ContentVariantFailed     = "failed"

	ContentTopicPending   = "pending"
	ContentTopicUsed      = "used"
	ContentTopicDismissed = "dismissed"

	contentDefaultModel      = "gpt-6.1-sol"
	contentDefaultDailyTime  = "09:00"
	contentDefaultPromoBonus = 1.0
	contentMaxDailyCount     = 3
	contentMaxActivePerOwner = 3
	contentMaxRunsSiteWide   = 3
	contentRunTimeout        = 25 * time.Minute
	contentListLimit         = 50
	contentTopicListLimit    = 100
	contentMaxTopic          = 200
	contentMaxAngle          = 500
	contentMaxNotes          = 6000
	contentMaxStyle          = 2000
	contentMaxBrief          = 1000
	contentMaxRule           = 300
	contentMaxFeedback       = 500
	contentMaxBody           = 20000
	contentPromoMaxUses      = 500
	contentPromoDays         = 90
	contentTickInterval      = 5 * time.Minute
	contentSaveEvery         = 3 * time.Second
)

// ContentPlatforms in display order.
var ContentPlatforms = []string{ContentPlatformWechat, ContentPlatformZhihu, ContentPlatformCSDN, ContentPlatformToutiao, ContentPlatformXHS}

var (
	ErrContentNotFound  = infraerrors.NotFound("CONTENT_NOT_FOUND", "没有找到这篇稿子或这个选题（Not found）")
	ErrContentState     = infraerrors.Conflict("CONTENT_STATE", "稿子正在生成，或者当前状态不能这样操作，刷新后再试（Not allowed in the current state）")
	ErrContentBusy      = infraerrors.TooManyRequests("CONTENT_BUSY", fmt.Sprintf("已经有 %d 篇稿子在生成，等它们完成后再开始新的（Too many running articles）", contentMaxActivePerOwner))
	ErrContentSiteBusy  = infraerrors.TooManyRequests("CONTENT_SITE_BUSY", "现在生成的稿子太多，稍后再试（Server busy）")
	ErrContentKey       = infraerrors.BadRequest("CONTENT_KEY", "请先在「设置」里选一个你自己的、可用的 GPT 分组 Key（Choose one of your active GPT keys）")
	ErrContentSettings  = infraerrors.BadRequest("CONTENT_SETTINGS", "设置不正确：时间要写成 09:00 这样，每天篇数 1–3，至少选一个平台（Invalid settings）")
	ErrContentTopic     = infraerrors.BadRequest("CONTENT_TOPIC", fmt.Sprintf("请填写选题，最多 %d 字；角度最多 %d 字，素材最多 %d 字（Topic required）", contentMaxTopic, contentMaxAngle, contentMaxNotes))
	ErrContentRule      = infraerrors.BadRequest("CONTENT_RULE", fmt.Sprintf("规则不能为空，最多 %d 字（Rule required）", contentMaxRule))
	ErrContentVariant   = infraerrors.BadRequest("CONTENT_VARIANT", "标题不能为空，正文不能为空且不超过 2 万字（Title and body required）")
	ErrContentPostURL   = infraerrors.BadRequest("CONTENT_POST_URL", "请粘贴发布后的帖子链接，以 http:// 或 https:// 开头（Post URL required）")
	ErrContentNotify    = infraerrors.BadRequest("CONTENT_NOTIFY", "微信推送没有配置好：先选推送方式并填好 SendKey / Token / Webhook 地址（Push not configured）")
	ErrContentNotifyURL = infraerrors.BadRequest("CONTENT_NOTIFY_URL", "企业微信机器人要填完整的 Webhook 地址，以 https://qyapi.weixin.qq.com/ 开头（Invalid webhook URL）")
	ErrContentEncryptor = infraerrors.ServiceUnavailable("CONTENT_ENCRYPTOR", "服务器没有配置加密密钥，不能保存推送密钥（Encryption unavailable）")
)

// contentPlatformSpec is what each platform takes; the prompts, the checks and the publish package
// all read from it.
type contentPlatformSpec struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`      // channel link utm_source
	TitleMax   int    `json:"title_max"`   // characters
	BodyMin    int    `json:"body_min"`    // characters, a warning below
	BodyMax    int    `json:"body_max"`    // characters, an error above
	Links      bool   `json:"links"`       // the body may carry {{link}}
	ReadMore   bool   `json:"read_more"`   // the link goes in 阅读原文 instead (公众号)
	Promo      bool   `json:"promo"`       // no links: the body carries {{promo}}
	TagsMin    int    `json:"tags_min"`    // 0 = no tags
	TagsMax    int    `json:"tags_max"`    //
	Cards      bool   `json:"cards"`       // 小红书 image cards
	MicroMax   int    `json:"micro_max"`   // 头条 微头条, 0 = none
	AILabel    string `json:"ai_label"`    // the AI-content declaration to tick when publishing
	PublishURL string `json:"publish_url"` // where to paste it
}

var contentPlatformSpecs = map[string]contentPlatformSpec{
	ContentPlatformWechat: {ID: ContentPlatformWechat, Name: "公众号", Source: "wechat", TitleMax: 64, BodyMin: 1200, BodyMax: 5000, ReadMore: true,
		AILabel: "发表前在「创作来源 / 原创声明」里如实选择「AI 生成」或「AI 辅助创作」", PublishURL: "/editor/"},
	ContentPlatformZhihu: {ID: ContentPlatformZhihu, Name: "知乎", Source: "zhihu", TitleMax: 50, BodyMin: 500, BodyMax: 4000, Links: true, TagsMin: 1, TagsMax: 5,
		AILabel: "发布设置里勾选「包含 AI 辅助创作」", PublishURL: "https://zhuanlan.zhihu.com/write"},
	ContentPlatformCSDN: {ID: ContentPlatformCSDN, Name: "CSDN", Source: "csdn", TitleMax: 80, BodyMin: 1000, BodyMax: 8000, Links: true, TagsMin: 3, TagsMax: 5,
		AILabel: "文末保留「本文由 AI 辅助创作」，发布时按页面要求选择创作声明", PublishURL: "https://mp.csdn.net/mp_blog/creation/editor"},
	ContentPlatformToutiao: {ID: ContentPlatformToutiao, Name: "今日头条", Source: "toutiao", TitleMax: 30, BodyMin: 800, BodyMax: 3000, Promo: true, MicroMax: 300,
		AILabel: "发布时在「作品声明」里选择「内容由 AI 生成」", PublishURL: "https://mp.toutiao.com/profile_v4/graphic/publish"},
	ContentPlatformXHS: {ID: ContentPlatformXHS, Name: "小红书", Source: "xhs", TitleMax: 20, BodyMin: 200, BodyMax: 1000, Promo: true, TagsMin: 5, TagsMax: 8, Cards: true,
		AILabel: "发布时勾选「笔记含 AI 合成内容」", PublishURL: "https://creator.xiaohongshu.com/publish/publish"},
}

// ContentPlatformSpecs lists the specs in display order (the admin page shows them).
func ContentPlatformSpecs() []contentPlatformSpec {
	out := make([]contentPlatformSpec, 0, len(ContentPlatforms))
	for _, p := range ContentPlatforms {
		out = append(out, contentPlatformSpecs[p])
	}
	return out
}

// --- types -------------------------------------------------------------------------------------

// ContentSettings is one owner's setup. NotifyKey is write-only (reads report NotifyKeySet); an
// empty NotifyKey on save keeps the stored one.
type ContentSettings struct {
	Enabled     bool     `json:"enabled"`
	KeyID       int64    `json:"key_id"`
	Model       string   `json:"model"`
	DailyTime   string   `json:"daily_time"`
	DailyCount  int      `json:"daily_count"`
	Platforms   []string `json:"platforms"`
	Search      bool     `json:"search"`
	StylePrompt string   `json:"style_prompt"`
	Product     string   `json:"product"` // what the posts lead readers to
	PromoBonus  float64  `json:"promo_bonus"`
	NotifyType  string   `json:"notify_type"` // "" | serverchan | pushplus | wecom
	NotifyKey   string   `json:"notify_key,omitempty"`

	// Read-only.
	NotifyKeySet    bool   `json:"notify_key_set"`
	KeyName         string `json:"key_name,omitempty"`
	KeyProblem      string `json:"key_problem,omitempty"`
	SearchAvailable bool   `json:"search_available"`
	LastRunDay      string `json:"last_run_day,omitempty"`
}

// ContentOwnerRow is how settings are stored.
type ContentOwnerRow struct {
	OwnerUserID  int64
	Settings     ContentSettings
	NotifyKeyEnc string
	Enabled      bool
	LastRunDay   string
}

type ContentTopic struct {
	ID          int64     `json:"id"`
	OwnerUserID int64     `json:"-"`
	Title       string    `json:"title"`
	Angle       string    `json:"angle"`
	Notes       string    `json:"notes"`
	Source      string    `json:"source"` // manual | support | geo | learn | hot | idea | comment
	Reason      string    `json:"reason"`
	Score       int       `json:"score"`
	Status      string    `json:"status"`
	ItemID      *int64    `json:"item_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ContentTopicInput struct {
	Title string `json:"title"`
	Angle string `json:"angle"`
	Notes string `json:"notes"`
	Score *int   `json:"score,omitempty"`
}

// ContentBrief is what a run was asked to write.
type ContentBrief struct {
	Topic     string   `json:"topic"`
	Angle     string   `json:"angle,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	Source    string   `json:"source,omitempty"`
	Platforms []string `json:"platforms"`
	Search    bool     `json:"search"`
}

type ContentSection struct {
	Heading string   `json:"heading"`
	Points  []string `json:"points"`
}

type ContentOutline struct {
	Title           string           `json:"title"`
	Angle           string           `json:"angle"`
	Reader          string           `json:"reader"`
	Hook            string           `json:"hook"`
	Sections        []ContentSection `json:"sections"`
	MaterialsNeeded []string         `json:"materials_needed"`
	LandingPath     string           `json:"landing_path"`
}

// ContentLanding is a learning-site page a post could send readers to.
type ContentLanding struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type ContentCheck struct {
	Level string `json:"level"` // error | warn | info
	Code  string `json:"code"`
	Text  string `json:"text"`
}

type ContentItemData struct {
	Brief            ContentBrief     `json:"brief"`
	Materials        []ContentLanding `json:"landing,omitempty"`
	LandingPath      string           `json:"landing_path,omitempty"`
	Sources          []ArticleSource  `json:"sources"`
	Outline          *ContentOutline  `json:"outline,omitempty"`
	Draft            string           `json:"draft,omitempty"`
	Reviewed         bool             `json:"reviewed,omitempty"`
	ReviewNotes      []string         `json:"review_notes,omitempty"`
	AIScore          int              `json:"ai_score,omitempty"` // the reviewer's 0–100 "reads like AI"
	Markdown         string           `json:"markdown,omitempty"`
	Checks           []ContentCheck   `json:"checks,omitempty"`
	RejectReason     string           `json:"reject_reason,omitempty"`
	Events           []ArticleEvent   `json:"events"`
	Searches         int              `json:"searches,omitempty"`
	Model            string           `json:"model,omitempty"`
	PromptTokens     int              `json:"prompt_tokens"`
	CompletionTokens int              `json:"completion_tokens"`
	Desk             *ContentDesk     `json:"desk,omitempty"` // 写作台 articles only
}

// ContentVariantSummary is a version's platform and status, for lists.
type ContentVariantSummary struct {
	Platform string `json:"platform"`
	Status   string `json:"status"`
}

type ContentItem struct {
	ID          int64     `json:"id"`
	OwnerUserID int64     `json:"-"`
	TopicID     *int64    `json:"topic_id,omitempty"`
	KeyID       int64     `json:"key_id"`
	Status      string    `json:"status"`
	Title       string    `json:"title"`
	Error       string    `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ContentItemData
	Variants []ContentVariantSummary `json:"variants,omitempty"` // lists only
	Running  bool                    `json:"running"`
}

type ContentCard struct {
	Title  string   `json:"title"`
	Points []string `json:"points"`
}

type ContentVariantData struct {
	Question    string         `json:"question,omitempty"` // 知乎回答: the question to answer
	Digest      string         `json:"digest,omitempty"`   // 公众号摘要
	Micro       string         `json:"micro,omitempty"`    // 头条微头条
	Category    string         `json:"category,omitempty"` // CSDN
	Tags        []string       `json:"tags,omitempty"`
	Cards       []ContentCard  `json:"cards,omitempty"` // 小红书
	Checks      []ContentCheck `json:"checks,omitempty"`
	Edited      bool           `json:"edited,omitempty"`
	Feedback    string         `json:"feedback,omitempty"`
	PromoCodeID int64          `json:"promo_code_id,omitempty"`
	Error       string         `json:"error,omitempty"`
}

type ContentVariant struct {
	ID          int64      `json:"id"`
	ItemID      int64      `json:"item_id"`
	OwnerUserID int64      `json:"-"`
	Platform    string     `json:"platform"`
	Kind        string     `json:"kind"` // article | answer | note
	Status      string     `json:"status"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	LinkCode    string     `json:"link_code"`
	PromoCode   string     `json:"promo_code"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	PostURL     string     `json:"post_url"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ContentVariantData

	// Filled in for the page: the body and micro post with the link and promo code in place, and
	// the link itself.
	Rendered      string `json:"rendered,omitempty"`
	RenderedMicro string `json:"rendered_micro,omitempty"`
	LinkURL       string `json:"link_url,omitempty"`
	ItemTitle     string `json:"item_title,omitempty"` // stats only
}

// ContentItemDetail is one article with its versions.
type ContentItemDetail struct {
	*ContentItem
	Versions []ContentVariant `json:"versions"`
}

type ContentRule struct {
	ID          int64     `json:"id"`
	OwnerUserID int64     `json:"-"`
	Rule        string    `json:"rule"`
	Origin      string    `json:"origin"` // manual | reject
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// ContentVariantInput is the owner's edit of a version.
type ContentVariantInput struct {
	Title    string        `json:"title"`
	Body     string        `json:"body"`
	Question string        `json:"question"`
	Digest   string        `json:"digest"`
	Micro    string        `json:"micro"`
	Category string        `json:"category"`
	Tags     []string      `json:"tags"`
	Cards    []ContentCard `json:"cards"`
}

// ContentStatsRow is a published version with what it brought in.
type ContentStatsRow struct {
	ContentVariant
	Clicks    int64   `json:"clicks"`
	Visitors  int64   `json:"visitors"`
	Signups   int64   `json:"signups"`
	Activated int64   `json:"activated"`
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`
	PromoUses int     `json:"promo_uses"`
}

type ContentPlatformTotal struct {
	Platform  string  `json:"platform"`
	Posts     int     `json:"posts"`
	Clicks    int64   `json:"clicks"`
	Signups   int64   `json:"signups"`
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`
	PromoUses int     `json:"promo_uses"`
}

type ContentStats struct {
	Days      int                    `json:"days"`
	Rows      []ContentStatsRow      `json:"rows"`
	Platforms []ContentPlatformTotal `json:"platforms"`
}

// ContentOverview is the page's header: counts and the specs.
type ContentOverview struct {
	Desk      int                   `json:"desk"` // articles on the 写作台
	Pending   int                   `json:"pending"`
	Running   int                   `json:"running"`
	Topics    int                   `json:"topics"`
	Published int                   `json:"published"`
	Platforms []contentPlatformSpec `json:"platforms"`
}

// --- dependencies ------------------------------------------------------------------------------

type ContentEngineRepository interface {
	GetSettings(ctx context.Context, owner int64) (*ContentOwnerRow, error) // nil when none
	SaveSettings(ctx context.Context, row *ContentOwnerRow) error
	ListEnabledOwners(ctx context.Context) ([]ContentOwnerRow, error)
	// MarkRun sets the owner's last run day; false when it already was that day (another tick won).
	MarkRun(ctx context.Context, owner int64, day string) (bool, error)

	ListTopics(ctx context.Context, owner int64, status string, limit int) ([]ContentTopic, error)
	GetTopic(ctx context.Context, owner, id int64) (*ContentTopic, error) // nil when unknown
	CreateTopic(ctx context.Context, t *ContentTopic) error
	UpdateTopic(ctx context.Context, t *ContentTopic) error
	DeleteTopic(ctx context.Context, owner, id int64) error
	RecentTitles(ctx context.Context, owner int64, limit int) ([]string, error)

	CreateItem(ctx context.Context, it *ContentItem) error
	GetItem(ctx context.Context, owner, id int64) (*ContentItem, error) // nil when unknown
	ListItems(ctx context.Context, owner int64, status string, limit int) ([]ContentItem, error)
	SaveItem(ctx context.Context, it *ContentItem) error
	DeleteItem(ctx context.Context, owner, id int64) error
	CountActive(ctx context.Context, owner int64) (int, error)
	FailActive(ctx context.Context, message string) (int64, error)
	CountByStatus(ctx context.Context, owner int64) (map[string]int, error)

	ListVariants(ctx context.Context, owner, itemID int64) ([]ContentVariant, error)
	GetVariant(ctx context.Context, owner, id int64) (*ContentVariant, error) // nil when unknown
	UpsertVariant(ctx context.Context, v *ContentVariant) error               // by (item, platform)
	SaveVariant(ctx context.Context, v *ContentVariant) error
	ListPublished(ctx context.Context, owner int64, since time.Time) ([]ContentVariant, error)
	FailGenerating(ctx context.Context, message string) (int64, error)
	FailDeskJobs(ctx context.Context, message string) (int64, error)

	GetPersona(ctx context.Context, owner int64) (*ContentPersona, error) // nil when none
	SavePersona(ctx context.Context, owner int64, p *ContentPersona) error
	CreateVersion(ctx context.Context, v *ContentVersion) error
	ListVersions(ctx context.Context, owner, itemID int64) ([]ContentVersion, error) // without the text
	GetVersion(ctx context.Context, owner, id int64) (*ContentVersion, error)        // nil when unknown

	ListRules(ctx context.Context, owner int64) ([]ContentRule, error) // oldest first (by id)
	CreateRule(ctx context.Context, r *ContentRule) error
	SetRuleEnabled(ctx context.Context, owner, id int64, enabled bool) error
	DeleteRule(ctx context.Context, owner, id int64) error
}

// contentLinker creates and reports channel links (ChannelLinkService).
type contentLinker interface {
	Create(ctx context.Context, in ChannelLinkInput) (*ChannelLink, error)
	Report(ctx context.Context, days int, costPerUSD float64) (*ChannelLinkReport, error)
}

// contentPromoter creates promo codes and reads their use (PromoService).
type contentPromoter interface {
	Create(ctx context.Context, in *CreatePromoCodeInput) (*PromoCode, error)
	GetByID(ctx context.Context, id int64) (*PromoCode, error)
}

// ContentSources feed the topic suggestions and the drafts; any may be nil.
type ContentSources struct {
	Runs AgentRunRepository   // the support assistant's questions
	Geo  GeoMonitorRepository // GEO questions
	// Learning-site passages for a query (material and landing pages) and the list of its pages.
	Search  func(query string, limit int) []assistantHit
	Catalog func() []AssistantSource
}

type ContentEngineService struct {
	repo      ContentEngineRepository
	learn     *LearnService
	settings  learnSettingStore
	encryptor SecretEncryptor
	search    ArticleSearcher
	links     contentLinker
	promo     contentPromoter
	sources   ContentSources
	notifier  contentNotifier
	fetcher   ContentArticleFetcher

	locksMu sync.Mutex
	locks   map[int64]*sync.Mutex

	agent articleAgentFunc
	now   func() time.Time
	slots chan struct{}

	runsMu   sync.Mutex
	runs     map[int64]context.CancelFunc
	wg       sync.WaitGroup
	savingMu sync.Mutex

	stop     chan struct{}
	stopOnce sync.Once
}

func NewContentEngineService(repo ContentEngineRepository, learn *LearnService, settings learnSettingStore, encryptor SecretEncryptor) *ContentEngineService {
	s := &ContentEngineService{repo: repo, learn: learn, settings: settings, encryptor: encryptor, now: time.Now,
		slots: make(chan struct{}, contentMaxRunsSiteWide), runs: map[int64]context.CancelFunc{}, stop: make(chan struct{})}
	gateway := ""
	if learn != nil {
		gateway = learn.gatewayURL
	}
	client := &http.Client{Timeout: 6 * time.Minute}
	s.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		in.GatewayURL, in.UserAgent = gateway, "hivegpt-content/1"
		return RunAgent(ctx, client, in)
	}
	s.notifier = newContentHTTPNotifier()
	return s
}

// SetDeps wires the optional collaborators (tests set fakes).
func (s *ContentEngineService) SetDeps(search ArticleSearcher, links contentLinker, promo contentPromoter, sources ContentSources) {
	s.search, s.links, s.promo, s.sources = search, links, promo, sources
}

// RecoverInterrupted fails the runs a previous process left behind; the owner can retry them.
func (s *ContentEngineService) RecoverInterrupted(ctx context.Context) {
	const msg = "服务器重启，生成中断了，点「重试」接着做（Interrupted by a server restart）"
	if n, err := s.repo.FailActive(ctx, msg); err != nil {
		slog.Warn("content: failing interrupted runs", "err", err)
	} else if n > 0 {
		slog.Info("content: interrupted runs marked failed", "count", n)
	}
	if _, err := s.repo.FailGenerating(ctx, msg); err != nil {
		slog.Warn("content: failing interrupted versions", "err", err)
	}
	if _, err := s.repo.FailDeskJobs(ctx, "服务器重启，这一步中断了，再点一次"); err != nil {
		slog.Warn("content: failing interrupted desk steps", "err", err)
	}
}

// Start runs the daily schedule until Stop.
func (s *ContentEngineService) Start() {
	go func() {
		t := time.NewTicker(contentTickInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				s.Tick(ctx)
				cancel()
			}
		}
	}()
}

// Stop ends the schedule and cancels running articles (they resume with 重试).
func (s *ContentEngineService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stop) })
	s.runsMu.Lock()
	for _, cancel := range s.runs {
		cancel()
	}
	s.runsMu.Unlock()
	s.wg.Wait()
}

// --- settings ----------------------------------------------------------------------------------

var contentTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

const contentDefaultProduct = "HiveGPT（hivegpt.cn）：国内可直接使用的 GPT / Codex API 服务和 AI 学习站。注册送体验额度，按量计费；学习站有免费的 AI 编程、API 接入、提示词和大数据教程。"

func contentDefaultSettings() ContentSettings {
	return ContentSettings{Model: "", DailyTime: contentDefaultDailyTime, DailyCount: 1, Platforms: append([]string(nil), ContentPlatforms...),
		Search: true, Product: contentDefaultProduct, PromoBonus: contentDefaultPromoBonus}
}

func (s *ContentEngineService) loadRow(ctx context.Context, owner int64) (*ContentOwnerRow, error) {
	row, err := s.repo.GetSettings(ctx, owner)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &ContentOwnerRow{OwnerUserID: owner, Settings: contentDefaultSettings()}, nil
	}
	st := &row.Settings
	if st.DailyTime == "" {
		st.DailyTime = contentDefaultDailyTime
	}
	if st.DailyCount <= 0 {
		st.DailyCount = 1
	}
	if len(st.Platforms) == 0 {
		st.Platforms = append([]string(nil), ContentPlatforms...)
	}
	if strings.TrimSpace(st.Product) == "" {
		st.Product = contentDefaultProduct
	}
	st.Enabled = row.Enabled
	return row, nil
}

func (s *ContentEngineService) model(ctx context.Context, st ContentSettings) string {
	if m := strings.TrimSpace(st.Model); m != "" {
		return m
	}
	if s.learn != nil {
		if ls, _ := s.learn.loadSettings(ctx); ls.Model != "" {
			return ls.Model
		}
	}
	return contentDefaultModel
}

func (s *ContentEngineService) key(ctx context.Context, owner, keyID int64) (*APIKey, error) {
	if keyID <= 0 || s.learn == nil {
		return nil, ErrContentKey
	}
	k, err := s.learn.ownKey(ctx, owner, keyID)
	if err != nil {
		return nil, ErrContentKey
	}
	return k, nil
}

func (s *ContentEngineService) searchAvailable(ctx context.Context) bool {
	return s.search != nil && s.search.Available(ctx)
}

// Settings for the page (the push key is never returned).
func (s *ContentEngineService) Settings(ctx context.Context, owner int64) (ContentSettings, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return ContentSettings{}, err
	}
	st := row.Settings
	st.NotifyKey = ""
	st.NotifyKeySet = row.NotifyKeyEnc != ""
	st.LastRunDay = row.LastRunDay
	st.SearchAvailable = s.searchAvailable(ctx)
	if st.KeyID > 0 {
		if k, err := s.key(ctx, owner, st.KeyID); err == nil {
			st.KeyName = k.Name
		} else {
			st.KeyProblem = infraerrors.Message(err)
		}
	}
	return st, nil
}

func normalizeContentPlatforms(in []string) []string {
	want := map[string]bool{}
	for _, p := range in {
		want[strings.TrimSpace(p)] = true
	}
	out := []string{}
	for _, p := range ContentPlatforms {
		if want[p] {
			out = append(out, p)
		}
	}
	return out
}

// SaveSettings stores the owner's settings; the key must be one of the owner's own GPT keys.
func (s *ContentEngineService) SaveSettings(ctx context.Context, owner int64, in ContentSettings) (ContentSettings, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return ContentSettings{}, err
	}
	in.Model = strings.TrimSpace(in.Model)
	in.DailyTime = strings.TrimSpace(in.DailyTime)
	in.Platforms = normalizeContentPlatforms(in.Platforms)
	in.StylePrompt = strings.TrimSpace(in.StylePrompt)
	in.Product = strings.TrimSpace(in.Product)
	in.NotifyType = strings.TrimSpace(in.NotifyType)
	if len(in.Model) > 100 || !contentTimePattern.MatchString(in.DailyTime) || in.DailyCount < 1 || in.DailyCount > contentMaxDailyCount ||
		len(in.Platforms) == 0 || utf8.RuneCountInString(in.StylePrompt) > contentMaxStyle || utf8.RuneCountInString(in.Product) > contentMaxBrief ||
		in.PromoBonus < 0 || in.PromoBonus > 20 || !contentNotifyTypeOK(in.NotifyType) {
		return ContentSettings{}, ErrContentSettings
	}
	if in.KeyID > 0 {
		if _, err := s.key(ctx, owner, in.KeyID); err != nil {
			return ContentSettings{}, err
		}
	} else if in.Enabled {
		return ContentSettings{}, ErrContentKey
	}
	if key := strings.TrimSpace(in.NotifyKey); key != "" {
		if in.NotifyType == contentNotifyWecom && !strings.HasPrefix(key, "https://qyapi.weixin.qq.com/") {
			return ContentSettings{}, ErrContentNotifyURL
		}
		if s.encryptor == nil {
			return ContentSettings{}, ErrContentEncryptor
		}
		enc, err := s.encryptor.Encrypt(key)
		if err != nil {
			return ContentSettings{}, ErrContentEncryptor
		}
		row.NotifyKeyEnc = enc
	}
	if in.NotifyType == "" {
		row.NotifyKeyEnc = ""
	}
	stored := ContentSettings{Enabled: in.Enabled, KeyID: in.KeyID, Model: in.Model, DailyTime: in.DailyTime, DailyCount: in.DailyCount,
		Platforms: in.Platforms, Search: in.Search, StylePrompt: in.StylePrompt, Product: in.Product, PromoBonus: in.PromoBonus, NotifyType: in.NotifyType}
	if stored.Product == "" {
		stored.Product = contentDefaultProduct
	}
	row.Settings, row.Enabled = stored, in.Enabled
	if err := s.repo.SaveSettings(ctx, row); err != nil {
		return ContentSettings{}, err
	}
	return s.Settings(ctx, owner)
}

// Keys: the owner's own active GPT keys, to choose from.
func (s *ContentEngineService) Keys(ctx context.Context, owner int64) ([]LearnKeyOption, error) {
	if s.learn == nil {
		return []LearnKeyOption{}, nil
	}
	return s.learn.OwnKeys(ctx, owner)
}

// TestNotify sends a test push with the stored settings.
func (s *ContentEngineService) TestNotify(ctx context.Context, owner int64) error {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return err
	}
	return s.push(ctx, row, "内容引擎推送测试", "收到这条消息，说明待审核提醒能发到你的微信了。", s.siteURL(ctx)+"/admin/content")
}

// push sends one message with the owner's channel.
func (s *ContentEngineService) push(ctx context.Context, row *ContentOwnerRow, title, body, link string) error {
	if row.Settings.NotifyType == "" || row.NotifyKeyEnc == "" || s.encryptor == nil || s.notifier == nil {
		return ErrContentNotify
	}
	key, err := s.encryptor.Decrypt(row.NotifyKeyEnc)
	if err != nil || key == "" {
		return ErrContentNotify
	}
	return s.notifier.Send(ctx, row.Settings.NotifyType, key, contentNotifyMessage{Title: title, Body: body, URL: link})
}

func (s *ContentEngineService) siteURL(ctx context.Context) string {
	if s.settings != nil {
		if v, err := s.settings.GetMultiple(ctx, []string{SettingKeyFrontendURL}); err == nil && strings.HasPrefix(v[SettingKeyFrontendURL], "http") {
			return strings.TrimRight(v[SettingKeyFrontendURL], "/")
		}
	}
	return "https://hivegpt.cn"
}

func (s *ContentEngineService) siteName(ctx context.Context) string {
	if s.settings != nil {
		if v, err := s.settings.GetMultiple(ctx, []string{SettingKeySiteName}); err == nil && strings.TrimSpace(v[SettingKeySiteName]) != "" {
			return strings.TrimSpace(v[SettingKeySiteName])
		}
	}
	return "HiveGPT"
}

// --- overview ----------------------------------------------------------------------------------

func (s *ContentEngineService) Overview(ctx context.Context, owner int64) (*ContentOverview, error) {
	counts, err := s.repo.CountByStatus(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := &ContentOverview{Platforms: ContentPlatformSpecs(), Desk: counts[ContentDeskStatus], Pending: counts[ContentPending], Topics: counts["topics"], Published: counts["published"]}
	for _, st := range []string{ContentOutlining, ContentDrafting, ContentReviewing, ContentHumanizing, ContentAdapting} {
		out.Running += counts[st]
	}
	return out, nil
}

// --- topics ------------------------------------------------------------------------------------

func normalizeContentTopic(in *ContentTopicInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Angle = strings.TrimSpace(in.Angle)
	in.Notes = strings.TrimSpace(in.Notes)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > contentMaxTopic || utf8.RuneCountInString(in.Angle) > contentMaxAngle ||
		utf8.RuneCountInString(in.Notes) > contentMaxNotes {
		return ErrContentTopic
	}
	return nil
}

func (s *ContentEngineService) Topics(ctx context.Context, owner int64, status string) ([]ContentTopic, error) {
	if status == "" {
		status = ContentTopicPending
	}
	if status == "all" {
		status = ""
	}
	return s.repo.ListTopics(ctx, owner, status, contentTopicListLimit)
}

func (s *ContentEngineService) CreateTopic(ctx context.Context, owner int64, in ContentTopicInput) (*ContentTopic, error) {
	if err := normalizeContentTopic(&in); err != nil {
		return nil, err
	}
	t := &ContentTopic{OwnerUserID: owner, Title: in.Title, Angle: in.Angle, Notes: in.Notes, Source: "manual", Score: 70, Status: ContentTopicPending}
	if in.Score != nil {
		t.Score = max(0, min(*in.Score, 100))
	}
	if err := s.repo.CreateTopic(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ContentEngineService) UpdateTopic(ctx context.Context, owner, id int64, in ContentTopicInput) (*ContentTopic, error) {
	if err := normalizeContentTopic(&in); err != nil {
		return nil, err
	}
	t, err := s.topic(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	t.Title, t.Angle, t.Notes = in.Title, in.Angle, in.Notes
	if in.Score != nil {
		t.Score = max(0, min(*in.Score, 100))
	}
	if err := s.repo.UpdateTopic(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetTopicStatus dismisses a topic or puts it back in the pool.
func (s *ContentEngineService) SetTopicStatus(ctx context.Context, owner, id int64, status string) (*ContentTopic, error) {
	if status != ContentTopicPending && status != ContentTopicDismissed {
		return nil, ErrContentState
	}
	t, err := s.topic(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	t.Status = status
	if err := s.repo.UpdateTopic(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *ContentEngineService) DeleteTopic(ctx context.Context, owner, id int64) error {
	return s.repo.DeleteTopic(ctx, owner, id)
}

func (s *ContentEngineService) topic(ctx context.Context, owner, id int64) (*ContentTopic, error) {
	t, err := s.repo.GetTopic(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrContentNotFound
	}
	return t, nil
}

// StartTopic writes an article on a topic now.
func (s *ContentEngineService) StartTopic(ctx context.Context, owner, id int64) (*ContentItem, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return nil, err
	}
	t, err := s.topic(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if t.Status == ContentTopicUsed {
		return nil, ErrContentState
	}
	return s.startItem(ctx, row, t)
}

func (s *ContentEngineService) startItem(ctx context.Context, row *ContentOwnerRow, t *ContentTopic) (*ContentItem, error) {
	owner, st := row.OwnerUserID, row.Settings
	key, err := s.key(ctx, owner, st.KeyID)
	if err != nil {
		return nil, err
	}
	if n, err := s.repo.CountActive(ctx, owner); err != nil {
		return nil, err
	} else if n >= contentMaxActivePerOwner {
		return nil, ErrContentBusy
	}
	topicID := t.ID
	it := &ContentItem{OwnerUserID: owner, TopicID: &topicID, KeyID: st.KeyID, Status: ContentOutlining, Title: truncateRunes(t.Title, 200),
		ContentItemData: ContentItemData{
			Brief:   ContentBrief{Topic: t.Title, Angle: t.Angle, Notes: t.Notes, Source: t.Source, Platforms: st.Platforms, Search: st.Search && s.searchAvailable(ctx)},
			Sources: []ArticleSource{}, Events: []ArticleEvent{},
		}}
	s.event(it, "info", "开始：「"+truncateRunes(t.Title, 60)+"」")
	if err := s.repo.CreateItem(ctx, it); err != nil {
		return nil, err
	}
	itemID := it.ID
	t.Status, t.ItemID = ContentTopicUsed, &itemID
	if err := s.repo.UpdateTopic(ctx, t); err != nil {
		slog.Warn("content: marking topic used", "topic", t.ID, "err", err)
	}
	runCtx, _ := s.claim(it.ID)
	s.start(runCtx, it, key, s.runPipeline)
	return it, nil
}

// --- items -------------------------------------------------------------------------------------

func (s *ContentEngineService) Items(ctx context.Context, owner int64, status string) ([]ContentItem, error) {
	items, err := s.repo.ListItems(ctx, owner, status, contentListLimit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Running = s.running(items[i].ID)
	}
	return items, nil
}

func (s *ContentEngineService) item(ctx context.Context, owner, id int64) (*ContentItem, error) {
	it, err := s.repo.GetItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it == nil {
		return nil, ErrContentNotFound
	}
	it.Running = s.running(it.ID)
	return it, nil
}

// Item returns an article with its versions, links and promo codes in place.
func (s *ContentEngineService) Item(ctx context.Context, owner, id int64) (*ContentItemDetail, error) {
	it, err := s.item(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	vs, err := s.repo.ListVariants(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	site := s.siteURL(ctx)
	for i := range vs {
		s.decorate(&vs[i], site)
	}
	sort.SliceStable(vs, func(i, j int) bool {
		return contentPlatformOrder(vs[i].Platform) < contentPlatformOrder(vs[j].Platform)
	})
	return &ContentItemDetail{ContentItem: it, Versions: vs}, nil
}

func contentPlatformOrder(p string) int {
	for i, x := range ContentPlatforms {
		if x == p {
			return i
		}
	}
	return len(ContentPlatforms)
}

func (s *ContentEngineService) decorate(v *ContentVariant, site string) {
	if v.LinkCode != "" {
		v.LinkURL = site + "/go/" + v.LinkCode
	}
	v.Rendered = renderContentBody(v.Body, v.LinkURL, v.PromoCode)
	if v.Micro != "" {
		v.RenderedMicro = renderContentBody(v.Micro, v.LinkURL, v.PromoCode)
	}
}

// Retry resumes a failed or stopped article from the step it stopped at; failed versions are
// written again.
func (s *ContentEngineService) Retry(ctx context.Context, owner, id int64) (*ContentItem, error) {
	it, err := s.item(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it.Status != ContentFailed && it.Status != ContentCanceled {
		return nil, ErrContentState
	}
	key, err := s.key(ctx, owner, it.KeyID)
	if err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(it.ID)
	if !ok {
		return nil, ErrContentState
	}
	it.Status, it.Error = contentResumeStatus(it), ""
	s.event(it, "info", "重试")
	s.save(it)
	s.start(runCtx, it, key, s.runPipeline)
	return it, nil
}

func contentResumeStatus(it *ContentItem) string {
	switch {
	case it.Outline == nil:
		return ContentOutlining
	case it.Draft == "":
		return ContentDrafting
	case !it.Reviewed:
		return ContentReviewing
	case it.Markdown == "":
		return ContentHumanizing
	default:
		return ContentAdapting
	}
}

// Cancel stops a running article.
func (s *ContentEngineService) Cancel(ctx context.Context, owner, id int64) error {
	if _, err := s.item(ctx, owner, id); err != nil {
		return err
	}
	s.runsMu.Lock()
	cancel := s.runs[id]
	s.runsMu.Unlock()
	if cancel == nil {
		return ErrContentState
	}
	cancel()
	return nil
}

func (s *ContentEngineService) DeleteItem(ctx context.Context, owner, id int64) error {
	if _, err := s.item(ctx, owner, id); err != nil {
		return err
	}
	if s.running(id) {
		return ErrContentState
	}
	return s.repo.DeleteItem(ctx, owner, id)
}

// Reject drops the whole article; the reason can become a 写作规则 for every later article.
func (s *ContentEngineService) Reject(ctx context.Context, owner, id int64, reason string, asRule bool) (*ContentItem, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > contentMaxRule {
		return nil, ErrContentRule
	}
	it, err := s.item(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it.Running {
		return nil, ErrContentState
	}
	it.Status, it.RejectReason = ContentRejected, reason
	s.event(it, "info", "已打回"+contentSuffix(reason))
	s.save(it)
	if asRule && reason != "" {
		if err := s.repo.CreateRule(ctx, &ContentRule{OwnerUserID: owner, Rule: reason, Origin: "reject", Enabled: true}); err != nil {
			return nil, err
		}
	}
	return it, nil
}

func contentSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return "：" + reason
}

// --- versions ----------------------------------------------------------------------------------

func (s *ContentEngineService) variant(ctx context.Context, owner, id int64) (*ContentVariant, error) {
	v, err := s.repo.GetVariant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrContentNotFound
	}
	return v, nil
}

func (s *ContentEngineService) variantOut(ctx context.Context, v *ContentVariant) *ContentVariant {
	s.decorate(v, s.siteURL(ctx))
	return v
}

// UpdateVariant saves the owner's edit and checks it again.
func (s *ContentEngineService) UpdateVariant(ctx context.Context, owner, id int64, in ContentVariantInput) (*ContentVariant, error) {
	v, err := s.variant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v.Status == ContentVariantGenerating {
		return nil, ErrContentState
	}
	in.Title, in.Body = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if in.Title == "" || in.Body == "" || utf8.RuneCountInString(in.Body) > contentMaxBody {
		return nil, ErrContentVariant
	}
	v.Title, v.Body = truncateRunes(in.Title, 200), in.Body
	v.Question, v.Digest, v.Micro, v.Category = truncateRunes(strings.TrimSpace(in.Question), 200), truncateRunes(strings.TrimSpace(in.Digest), 120),
		truncateRunes(strings.TrimSpace(in.Micro), 1000), truncateRunes(strings.TrimSpace(in.Category), 50)
	v.Tags, v.Cards = cleanContentTags(in.Tags), cleanContentCards(in.Cards)
	v.Edited = true
	v.Checks = checkContentVariant(v, nil, s.siteName(ctx))
	if err := s.repo.SaveVariant(ctx, v); err != nil {
		return nil, err
	}
	return s.variantOut(ctx, v), nil
}

// ApproveVariant passes a version: it gets its channel link (and promo code on platforms without
// links) and is ready to publish.
func (s *ContentEngineService) ApproveVariant(ctx context.Context, owner, id int64) (*ContentVariant, error) {
	v, err := s.variant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v.Status != ContentVariantPending && v.Status != ContentVariantSkipped {
		return nil, ErrContentState
	}
	it, err := s.item(ctx, owner, v.ItemID)
	if err != nil {
		return nil, err
	}
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return nil, err
	}
	spec := contentPlatformSpecs[v.Platform]
	if v.LinkCode == "" && s.links != nil {
		landing := it.LandingPath
		if landing == "" {
			landing = "/"
		}
		l, err := s.links.Create(ctx, ChannelLinkInput{Code: fmt.Sprintf("ce%d-%s", v.ID, spec.Source), Name: truncateRunes(spec.Name+"·"+v.Title, 100),
			Source: spec.Source, Medium: v.Kind, TargetPath: landing, Note: fmt.Sprintf("内容引擎 #%d 自动创建", it.ID)})
		if err != nil {
			return nil, err
		}
		v.LinkCode = l.Code
	}
	if spec.Promo && v.PromoCode == "" && s.promo != nil && row.Settings.PromoBonus > 0 {
		expires := s.now().AddDate(0, 0, contentPromoDays)
		code := strings.ToUpper(spec.Source) + randomContentCode()
		p, err := s.promo.Create(ctx, &CreatePromoCodeInput{Code: code, BonusAmount: row.Settings.PromoBonus, MaxUses: contentPromoMaxUses, ExpiresAt: &expires,
			Notes: truncateRunes(fmt.Sprintf("内容引擎 #%d %s：%s", it.ID, spec.Name, v.Title), 200)})
		if err != nil {
			return nil, err
		}
		v.PromoCode, v.PromoCodeID = p.Code, p.ID
	}
	v.Status = ContentVariantApproved
	if err := s.repo.SaveVariant(ctx, v); err != nil {
		return nil, err
	}
	s.settleItem(ctx, owner, v.ItemID)
	return s.variantOut(ctx, v), nil
}

// SkipVariant leaves a platform out of this article.
func (s *ContentEngineService) SkipVariant(ctx context.Context, owner, id int64) (*ContentVariant, error) {
	v, err := s.variant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v.Status == ContentVariantGenerating || v.Status == ContentVariantPublished {
		return nil, ErrContentState
	}
	v.Status = ContentVariantSkipped
	if err := s.repo.SaveVariant(ctx, v); err != nil {
		return nil, err
	}
	s.settleItem(ctx, owner, v.ItemID)
	return s.variantOut(ctx, v), nil
}

// PublishVariant records where an approved version was posted.
func (s *ContentEngineService) PublishVariant(ctx context.Context, owner, id int64, postURL string) (*ContentVariant, error) {
	postURL = strings.TrimSpace(postURL)
	u, err := url.Parse(postURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(postURL) > 500 {
		return nil, ErrContentPostURL
	}
	v, err := s.variant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v.Status != ContentVariantApproved && v.Status != ContentVariantPublished {
		return nil, ErrContentState
	}
	now := s.now()
	v.PostURL, v.Status = postURL, ContentVariantPublished
	if v.PublishedAt == nil {
		v.PublishedAt = &now
	}
	if err := s.repo.SaveVariant(ctx, v); err != nil {
		return nil, err
	}
	s.settleItem(ctx, owner, v.ItemID)
	return s.variantOut(ctx, v), nil
}

// RegenerateVariant writes one version again, with the owner's feedback.
func (s *ContentEngineService) RegenerateVariant(ctx context.Context, owner, id int64, feedback string) (*ContentVariant, error) {
	feedback = strings.TrimSpace(feedback)
	if utf8.RuneCountInString(feedback) > contentMaxFeedback {
		return nil, ErrContentVariant
	}
	v, err := s.variant(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if v.Status == ContentVariantGenerating || v.Status == ContentVariantPublished || v.Status == ContentVariantApproved {
		return nil, ErrContentState
	}
	it, err := s.item(ctx, owner, v.ItemID)
	if err != nil {
		return nil, err
	}
	if it.Markdown == "" {
		return nil, ErrContentState
	}
	key, err := s.key(ctx, owner, it.KeyID)
	if err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(it.ID)
	if !ok {
		return nil, ErrContentState
	}
	v.Status, v.Feedback = ContentVariantGenerating, feedback
	if err := s.repo.SaveVariant(ctx, v); err != nil {
		s.release(it.ID)
		return nil, err
	}
	platform := v.Platform
	s.start(runCtx, it, key, func(ctx context.Context, it *ContentItem, key string) error {
		return s.regenerate(ctx, it, key, platform, feedback)
	})
	return s.variantOut(ctx, v), nil
}

// settleItem marks the article done once no version is waiting.
func (s *ContentEngineService) settleItem(ctx context.Context, owner, itemID int64) {
	it, err := s.repo.GetItem(ctx, owner, itemID)
	if err != nil || it == nil || it.Status != ContentPending {
		return
	}
	vs, err := s.repo.ListVariants(ctx, owner, itemID)
	if err != nil {
		return
	}
	for _, v := range vs {
		if v.Status == ContentVariantPending || v.Status == ContentVariantGenerating {
			return
		}
	}
	it.Status = ContentDone
	s.save(it)
}

// --- rules -------------------------------------------------------------------------------------

func (s *ContentEngineService) Rules(ctx context.Context, owner int64) ([]ContentRule, error) {
	return s.repo.ListRules(ctx, owner)
}

func (s *ContentEngineService) CreateRule(ctx context.Context, owner int64, rule string) (*ContentRule, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" || utf8.RuneCountInString(rule) > contentMaxRule {
		return nil, ErrContentRule
	}
	r := &ContentRule{OwnerUserID: owner, Rule: rule, Origin: "manual", Enabled: true}
	if err := s.repo.CreateRule(ctx, r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *ContentEngineService) SetRuleEnabled(ctx context.Context, owner, id int64, enabled bool) error {
	return s.repo.SetRuleEnabled(ctx, owner, id, enabled)
}

func (s *ContentEngineService) DeleteRule(ctx context.Context, owner, id int64) error {
	return s.repo.DeleteRule(ctx, owner, id)
}

func (s *ContentEngineService) activeRules(ctx context.Context, owner int64) []string {
	rules, err := s.repo.ListRules(ctx, owner)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, r := range rules {
		if r.Enabled && len(out) < 30 {
			out = append(out, r.Rule)
		}
	}
	return out
}

// --- stats -------------------------------------------------------------------------------------

// Stats: every version published in the last days days (0 = all time) with the visitors, sign-ups
// and payments of its channel link and the uses of its promo code.
func (s *ContentEngineService) Stats(ctx context.Context, owner int64, days int) (*ContentStats, error) {
	since := time.Time{}
	if days != 0 {
		days = max(1, min(days, 365))
		since = s.now().AddDate(0, 0, -days)
	}
	vs, err := s.repo.ListPublished(ctx, owner, since)
	if err != nil {
		return nil, err
	}
	byCode := map[string]ChannelLink{}
	if s.links != nil && len(vs) > 0 {
		if rep, err := s.links.Report(ctx, 0, 0); err == nil {
			for _, l := range rep.Links {
				byCode[l.Code] = l
			}
		}
	}
	out := &ContentStats{Days: days, Rows: []ContentStatsRow{}, Platforms: []ContentPlatformTotal{}}
	totals := map[string]*ContentPlatformTotal{}
	site := s.siteURL(ctx)
	for _, v := range vs {
		v.Body = "" // the list doesn't need it
		v.Cards = nil
		if v.LinkCode != "" {
			v.LinkURL = site + "/go/" + v.LinkCode
		}
		row := ContentStatsRow{ContentVariant: v}
		if l, ok := byCode[v.LinkCode]; ok {
			row.Clicks, row.Visitors, row.Signups, row.Activated, row.PaidUsers, row.Revenue = l.Clicks, l.Visitors, l.Signups, l.Activated, l.PaidUsers, round2(l.Revenue)
		}
		if v.PromoCodeID > 0 && s.promo != nil {
			if p, err := s.promo.GetByID(ctx, v.PromoCodeID); err == nil && p != nil {
				row.PromoUses = p.UsedCount
			}
		}
		out.Rows = append(out.Rows, row)
		t := totals[v.Platform]
		if t == nil {
			t = &ContentPlatformTotal{Platform: v.Platform}
			totals[v.Platform] = t
		}
		t.Posts++
		t.Clicks += row.Clicks
		t.Signups += row.Signups
		t.PaidUsers += row.PaidUsers
		t.Revenue = round2(t.Revenue + row.Revenue)
		t.PromoUses += row.PromoUses
	}
	for _, p := range ContentPlatforms {
		if t := totals[p]; t != nil {
			out.Platforms = append(out.Platforms, *t)
		}
	}
	return out, nil
}

// --- daily run ---------------------------------------------------------------------------------

// Tick starts the day's articles for every owner whose time has come (once a day).
func (s *ContentEngineService) Tick(ctx context.Context) {
	owners, err := s.repo.ListEnabledOwners(ctx)
	if err != nil {
		slog.Warn("content: listing owners", "err", err)
		return
	}
	now := s.now().In(learnDayZone)
	day, clock := now.Format("2006-01-02"), now.Format("15:04")
	for i := range owners {
		row := &owners[i]
		if row.LastRunDay == day || clock < row.Settings.DailyTime {
			continue
		}
		if ok, err := s.repo.MarkRun(ctx, row.OwnerUserID, day); err != nil || !ok {
			continue
		}
		if err := s.RunDaily(ctx, row.OwnerUserID); err != nil {
			slog.Warn("content: daily run", "owner", row.OwnerUserID, "err", err)
			s.notifyFailure(ctx, row, err)
		}
	}
}

// RunDaily tops up the topic pool when it runs low and starts the day's articles on the best
// pending topics. The page's 「立即运行」 calls it too.
func (s *ContentEngineService) RunDaily(ctx context.Context, owner int64) error {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return err
	}
	if _, err := s.key(ctx, owner, row.Settings.KeyID); err != nil {
		return err
	}
	count := max(1, min(row.Settings.DailyCount, contentMaxDailyCount))
	topics, err := s.repo.ListTopics(ctx, owner, ContentTopicPending, contentTopicListLimit)
	if err != nil {
		return err
	}
	if len(topics) < count {
		if _, err := s.SuggestTopics(ctx, owner); err != nil {
			return err
		}
		if topics, err = s.repo.ListTopics(ctx, owner, ContentTopicPending, contentTopicListLimit); err != nil {
			return err
		}
	}
	sort.SliceStable(topics, func(i, j int) bool { return topics[i].Score > topics[j].Score })
	started := 0
	for i := range topics {
		if started >= count {
			break
		}
		if _, err := s.startItem(ctx, row, &topics[i]); err != nil {
			if errors.Is(err, ErrContentBusy) {
				break
			}
			return err
		}
		started++
	}
	return nil
}

func (s *ContentEngineService) notifyFailure(ctx context.Context, row *ContentOwnerRow, err error) {
	if row.Settings.NotifyType == "" || row.NotifyKeyEnc == "" {
		return
	}
	msg := infraerrors.Message(err)
	if msg == "" {
		msg = "内部错误，看服务器日志"
	}
	if perr := s.push(ctx, row, "内容引擎今天没能开始", "原因："+msg, s.siteURL(ctx)+"/admin/content"); perr != nil {
		slog.Warn("content: push", "owner", row.OwnerUserID, "err", perr)
	}
}

// --- run machinery -----------------------------------------------------------------------------

func (s *ContentEngineService) running(id int64) bool {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	return s.runs[id] != nil
}

func (s *ContentEngineService) claim(id int64) (context.Context, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if s.runs[id] != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), contentRunTimeout)
	s.runs[id] = cancel
	return ctx, true
}

func (s *ContentEngineService) release(id int64) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if cancel := s.runs[id]; cancel != nil {
		cancel()
		delete(s.runs, id)
	}
}

func (s *ContentEngineService) event(it *ContentItem, kind, text string) {
	it.Events = append(it.Events, ArticleEvent{At: s.now(), Kind: kind, Text: text})
	if len(it.Events) > 200 {
		it.Events = it.Events[len(it.Events)-200:]
	}
}

func (s *ContentEngineService) save(it *ContentItem) {
	s.savingMu.Lock()
	defer s.savingMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.repo.SaveItem(ctx, it); err != nil {
		slog.Warn("content: save", "id", it.ID, "err", err)
	}
}

func (s *ContentEngineService) saveVariant(v *ContentVariant) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.repo.SaveVariant(ctx, v)
}

// start runs fn in the background on a claimed article; the outcome is always saved.
func (s *ContentEngineService) start(ctx context.Context, it *ContentItem, key *APIKey, fn func(ctx context.Context, it *ContentItem, key string) error) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.release(it.ID)
		var err error
		queued := false
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
			err = fn(ctx, it, key.Key)
		case <-ctx.Done():
			err, queued = ErrContentSiteBusy, true
		}
		switch {
		case err == nil:
		case errors.Is(ctx.Err(), context.Canceled):
			it.Status, it.Error = ContentCanceled, ""
			s.event(it, "error", "已停止")
		case queued:
			it.Status, it.Error = ContentFailed, ErrContentSiteBusy.Message
			s.event(it, "error", it.Error)
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			it.Status, it.Error = ContentFailed, "生成超时，点「重试」接着做"
			s.event(it, "error", it.Error)
		default:
			msg := contentErrorText(it.ID, err)
			it.Status, it.Error = ContentFailed, msg
			s.event(it, "error", "出错了："+msg)
		}
		if err != nil {
			s.failGenerating(it)
		}
		s.save(it)
	}()
}

// failGenerating puts versions a stopped run was writing back to a state the owner can act on.
func (s *ContentEngineService) failGenerating(it *ContentItem) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	vs, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return
	}
	for i := range vs {
		if vs[i].Status == ContentVariantGenerating {
			vs[i].Status, vs[i].Error = ContentVariantFailed, "生成中断了，点「重新生成」再来一次"
			_ = s.repo.SaveVariant(ctx, &vs[i])
		}
	}
	// A regeneration that failed leaves an article already under review where it was.
	if it.Markdown != "" && len(vs) > 0 && it.Status == ContentFailed && contentHasReviewable(vs) {
		it.Status = ContentPending
	}
}

func contentHasReviewable(vs []ContentVariant) bool {
	for _, v := range vs {
		if v.Status != ContentVariantFailed && v.Status != ContentVariantGenerating {
			return true
		}
	}
	return false
}

func contentErrorText(id int64, err error) string {
	var app *infraerrors.ApplicationError
	if errors.As(err, &app) && app.Message != "" {
		return app.Message
	}
	slog.Warn("content: run failed", "id", id, "err", err)
	return "生成出错了，点「重试」再来一次"
}

func (s *ContentEngineService) usage(it *ContentItem, res *AgentRunResult) {
	if res == nil {
		return
	}
	it.PromptTokens += res.PromptTokens
	it.CompletionTokens += res.CompletionTokens
	if res.Model != "" {
		it.Model = res.Model
	}
}

func randomContentCode() string {
	return strings.ToUpper(randomChannelSuffix())
}
