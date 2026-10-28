package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 课本诗画: the admin turns textbook poems and short classical prose into illustrated posters.
//
//	works (text checked twice) → a text model drafts the scene ("theme") → batches of drawings through
//	this site's own gateway (gpt-image-2, billed to the admin's key, retried on 429 / 5xx) → the admin
//	picks one per work → super-resolution on the imagetools service → the admin page sets the body
//	text, seal and AI label in the browser and uploads one poster per aspect ratio → zip export.
//
// The body text is never left to the image model: tested on 2026-10-25 it got 106 characters of
// 《小石潭记》 wrong in four places, and a vision model asked to proofread reported no errors. By
// default the model draws no text at all and the layout sets the title, the author and the body in a
// calligraphy font the admin picks; the older mode (settings title_by_model) lets the model write
// the title and the author's name.

const (
	SettingKeyShihua = "shihua_settings"

	ShihuaTextDraft     = "draft"
	ShihuaTextVerified1 = "verified_1"
	ShihuaTextVerified2 = "verified_2"

	ShihuaImageQueued    = "queued"
	ShihuaImageRunning   = "running"
	ShihuaImageSucceeded = "succeeded"
	ShihuaImageFailed    = "failed"
	ShihuaImageCanceled  = "canceled"

	ShihuaUpscaleQueued  = "queued"
	ShihuaUpscaleRunning = "running"
	ShihuaUpscaleDone    = "done"
	ShihuaUpscaleFailed  = "failed"

	shihuaImageModel        = "gpt-image-2"
	shihuaDefaultTextModel  = "gpt-5.6-sol"
	shihuaDefaultSize       = "1024x1536"
	shihuaDefaultQuality    = "high"
	shihuaDefaultConcurrent = 2
	shihuaDefaultDailyLimit = 50
	shihuaMaxConcurrent     = 4
	shihuaMaxDailyLimit     = 500
	shihuaMaxCandidates     = 4
	shihuaMaxJobWorks       = 200
	shihuaMaxAttempts       = 3
	shihuaMaxTextRunes      = 4000
	shihuaMaxPromptRunes    = 3800 // the canvas' ?prompt= takes 4000
	shihuaMaxImportWorks    = 1000
)

var (
	ErrShihuaWorkNotFound  = infraerrors.NotFound("SHIHUA_WORK_NOT_FOUND", "找不到这篇篇目（Work not found）")
	ErrShihuaImageNotFound = infraerrors.NotFound("SHIHUA_IMAGE_NOT_FOUND", "找不到这张图（Image not found）")
	ErrShihuaJobNotFound   = infraerrors.NotFound("SHIHUA_JOB_NOT_FOUND", "找不到这个出图任务（Job not found）")
	ErrShihuaInvalid       = infraerrors.BadRequest("SHIHUA_INVALID", "填写的内容不完整或格式不对（Invalid input）")
	ErrShihuaSlugTaken     = infraerrors.Conflict("SHIHUA_SLUG_TAKEN", "这个 slug 已被其它篇目使用（Slug already used）")
	ErrShihuaKey           = infraerrors.BadRequest("SHIHUA_KEY", "请先在「设置」里选一把自己的、OpenAI 分组的 API Key（Pick an OpenAI API key in settings）")
	ErrShihuaNoTheme       = infraerrors.BadRequest("SHIHUA_NO_THEME", "有篇目还没有画面描述，请先生成或填写（Some works have no scene description）")
	ErrShihuaVerifyOrder   = infraerrors.BadRequest("SHIHUA_VERIFY_ORDER", "请按顺序核对：先一校再二校（Verify level 1 before level 2）")
	ErrShihuaVerifySame    = infraerrors.BadRequest("SHIHUA_VERIFY_SAME", "二校需要换一位管理员（Level 2 needs a different admin）")
	ErrShihuaNotVerified   = infraerrors.BadRequest("SHIHUA_NOT_VERIFIED", "正文还没通过二校，不能排版导出（Text not verified twice yet）")
	ErrShihuaNotPicked     = infraerrors.BadRequest("SHIHUA_NOT_PICKED", "只能给选中的图排版（Only the picked image can be laid out）")
	ErrShihuaThemeFailed   = infraerrors.BadRequest("SHIHUA_THEME_FAILED", "文本模型没有返回可用的画面描述，请重试（The text model returned no usable scene）")
)

// ShihuaTextbook is one place a work appears in a textbook.
type ShihuaTextbook struct {
	Edition string `json:"edition"`        // 统编 / 人教 / 苏教 …
	Era     string `json:"era,omitempty"`  // 2017- / 2001-2016 …
	Grade   string `json:"grade"`          // 八上 / 必修上 …
	Unit    int    `json:"unit,omitempty"` // 单元
	Lesson  string `json:"lesson,omitempty"`
	Kind    string `json:"kind,omitempty"` // 课文 / 课外古诗词诵读 / 日积月累
}

// ShihuaTheme is the scene of a work: drafted by a text model, edited by the admin. PromptTheme is
// what goes into the image prompt; the other fields are notes that helped write it.
type ShihuaTheme struct {
	Subject     string     `json:"subject,omitempty"`
	Scene       string     `json:"scene,omitempty"`
	Season      string     `json:"season,omitempty"`
	Time        string     `json:"time,omitempty"`
	Figures     string     `json:"figures,omitempty"`
	KeyImagery  []string   `json:"key_imagery,omitempty"`
	Palette     string     `json:"palette,omitempty"`
	Avoid       []string   `json:"avoid,omitempty"`
	PromptTheme string     `json:"prompt_theme"`
	Model       string     `json:"model,omitempty"`
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
}

type ShihuaWork struct {
	ID            int64            `json:"id"`
	Slug          string           `json:"slug"`
	Title         string           `json:"title"`
	Author        string           `json:"author"`
	Dynasty       string           `json:"dynasty"`
	Genre         string           `json:"genre"` // shi / ci / qu / wen / fu
	Stage         string           `json:"stage"` // primary / junior / senior / legacy
	Grade         string           `json:"grade"`
	Textbooks     []ShihuaTextbook `json:"textbooks"`
	FullText      string           `json:"full_text"`
	DisplayText   string           `json:"display_text"` // what goes on the poster; "" = full_text
	ExcerptNote   string           `json:"excerpt_note"`
	FamousLines   string           `json:"famous_lines"`
	Sources       string           `json:"sources"`
	VariantNote   string           `json:"variant_note"`
	TextStatus    string           `json:"text_status"`
	Verify1By     *int64           `json:"verify1_by,omitempty"`
	Verify1At     *time.Time       `json:"verify1_at,omitempty"`
	Verify2By     *int64           `json:"verify2_by,omitempty"`
	Verify2At     *time.Time       `json:"verify2_at,omitempty"`
	Theme         ShihuaTheme      `json:"theme"`
	PickedImageID *int64           `json:"picked_image_id,omitempty"`
	Sort          int              `json:"sort"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`

	// Filled in by lists.
	CharCount        int               `json:"char_count"`
	DisplayCharCount int               `json:"display_char_count"`
	Images           ShihuaImageCounts `json:"images"`
	Renders          int               `json:"renders"`
}

// PosterText is the text set on the poster.
func (w *ShihuaWork) PosterText() string {
	if strings.TrimSpace(w.DisplayText) != "" {
		return w.DisplayText
	}
	return w.FullText
}

type ShihuaImageCounts struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type ShihuaWorkQuery struct {
	Stage  string
	Grade  string
	Status string // text status
	Q      string
	IDs    []int64
}

type ShihuaJob struct {
	ID         int64             `json:"id"`
	Name       string            `json:"name"`
	Candidates int               `json:"candidates"`
	CreatedBy  *int64            `json:"created_by,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	Works      int               `json:"works"`
	Counts     ShihuaImageCounts `json:"counts"`
	Canceled   int               `json:"canceled"`
	CostUSD    float64           `json:"cost_usd"`
}

type ShihuaImage struct {
	ID            int64      `json:"id"`
	WorkID        int64      `json:"work_id"`
	JobID         *int64     `json:"job_id,omitempty"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	NotBefore     *time.Time `json:"not_before,omitempty"`
	Prompt        string     `json:"prompt"`
	Size          string     `json:"size"`
	Quality       string     `json:"quality"`
	Error         string     `json:"error,omitempty"`
	DurationMs    int        `json:"duration_ms"`
	CostUSD       float64    `json:"cost_usd"`
	File          string     `json:"-"`
	Width         int        `json:"width"`
	Height        int        `json:"height"`
	UpscaleStatus string     `json:"upscale_status"`
	UpscaleError  string     `json:"upscale_error,omitempty"`
	UpscaledFile  string     `json:"-"`
	TitleCheck    string     `json:"title_check"`
	// TitleDrawn: the picture carries the title the model wrote (its prompt asked for one), so the
	// layout sets only the body text.
	TitleDrawn bool       `json:"title_drawn"`
	Rejected   bool       `json:"rejected"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Signed links, filled in for the admin page.
	URL         string `json:"url,omitempty"`
	ThumbURL    string `json:"thumb_url,omitempty"`
	UpscaledURL string `json:"upscaled_url,omitempty"`
}

type ShihuaImageQuery struct {
	WorkIDs []int64
	JobID   int64
	Status  string
	Limit   int
}

type ShihuaRender struct {
	ID        int64           `json:"id"`
	ImageID   int64           `json:"image_id"`
	WorkID    int64           `json:"work_id"`
	Aspect    string          `json:"aspect"`
	File      string          `json:"-"`
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Layout    json.RawMessage `json:"layout"`
	CreatedAt time.Time       `json:"created_at"`
	URL       string          `json:"url,omitempty"`
}

type ShihuaQueueStats struct {
	Queued       int `json:"queued"`
	Running      int `json:"running"`
	StartedToday int `json:"started_today"`
	UpscaleQueue int `json:"upscale_queue"`
}

type ShihuaRepository interface {
	ListWorks(ctx context.Context, q ShihuaWorkQuery) ([]ShihuaWork, error)
	// GetWork / GetWorkBySlug return nil when the work does not exist.
	GetWork(ctx context.Context, id int64) (*ShihuaWork, error)
	GetWorkBySlug(ctx context.Context, slug string) (*ShihuaWork, error)
	CreateWork(ctx context.Context, w *ShihuaWork) error
	UpdateWork(ctx context.Context, w *ShihuaWork) error
	DeleteWork(ctx context.Context, id int64) error
	SetPicked(ctx context.Context, workID int64, imageID *int64) error

	CreateJob(ctx context.Context, name string, candidates int, createdBy int64, images []ShihuaImage) (*ShihuaJob, error)
	ListJobs(ctx context.Context, limit int) ([]ShihuaJob, error)
	EnqueueImages(ctx context.Context, images []ShihuaImage) error
	// CancelQueued cancels the queued drawings of a job (jobID > 0) or of every job (0).
	CancelQueued(ctx context.Context, jobID int64) (int64, error)
	RetryFailed(ctx context.Context, jobID int64) (int64, error)
	// ClaimNext marks the oldest due queued drawing running; nil when none is due.
	ClaimNext(ctx context.Context) (*ShihuaImage, error)
	FinishImage(ctx context.Context, img *ShihuaImage) error
	// RequeueRunning puts drawings a previous process left running back in the queue.
	RequeueRunning(ctx context.Context) (int64, error)
	QueueStats(ctx context.Context, dayStart time.Time) (ShihuaQueueStats, error)

	ListImages(ctx context.Context, q ShihuaImageQuery) ([]ShihuaImage, error)
	GetImage(ctx context.Context, id int64) (*ShihuaImage, error)
	ReviewImage(ctx context.Context, id int64, rejected bool, titleCheck string) error
	SetUpscale(ctx context.Context, id int64, status, errText, file string) error
	ClaimUpscale(ctx context.Context) (*ShihuaImage, error)
	RequeueUpscaleRunning(ctx context.Context) (int64, error)

	SaveRender(ctx context.Context, r *ShihuaRender) error
	ListRenders(ctx context.Context, workIDs []int64) ([]ShihuaRender, error)
}

// shihuaKeyReader looks up the admin's API key.
type shihuaKeyReader interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
}

// ShihuaSettings is the admin's configuration (settings key shihua_settings).
type ShihuaSettings struct {
	KeyID           int64  `json:"key_id"`
	TextModel       string `json:"text_model"`
	Size            string `json:"size"`
	Quality         string `json:"quality"`
	Concurrency     int    `json:"concurrency"`
	DailyLimit      int    `json:"daily_limit"`
	StylePrompt     string `json:"style_prompt"` // "" = DefaultShihuaStylePrompt
	AutoUpscale     bool   `json:"auto_upscale"`
	TwoPeopleVerify bool   `json:"two_people_verify"`
	// TitleByModel has the image model write the title and the author's name (the font can't be
	// changed afterwards); off, the picture has no text and the layout sets the title too.
	TitleByModel bool `json:"title_by_model"`
}

// ShihuaSettingsView is the settings page: the stored values plus what the page needs to show.
type ShihuaSettingsView struct {
	ShihuaSettings
	DefaultStylePrompt string           `json:"default_style_prompt"`
	UpscaleAvailable   bool             `json:"upscale_available"`
	Queue              ShihuaQueueStats `json:"queue"`
	Paused             string           `json:"paused,omitempty"` // why the queue is not drawing
}

var shihuaSizes = map[string]bool{"1024x1536": true, "1536x1024": true, "1024x1024": true}
var shihuaQualities = map[string]bool{"low": true, "medium": true, "high": true}

// DefaultShihuaStylePrompt is the series' look (the user's 新中式细线设色 template) without its text
// part: BuildShihuaPrompt adds the text rules of the title mode.
const DefaultShihuaStylePrompt = "画风： 【美学定位】 新中式细线设色装饰插画海报，融合东方散点透视与现代平面构成。画面兼具工笔线描的严谨结构与现代版式的呼吸感，整体干净、饱满、雅致。 " +
	"【构图与空间】 采用高位俯瞰或散点透视构建完整统一的场景，保证景物间尺度、遮挡与远近关系明确。遵循“大色面与连贯留白→中尺度结构→局部细节”的层级组织画面。利用道路、水流、山势、枝干或建筑边界形成自然的视觉引导线。地面、水面或云气采用连贯的暖米白留白参与空间组织，避免画面被零散小景切碎。 " +
	"【造型与细节控制】 线条纤细、连续、准确。大形与大色面承担主要视觉重量，核心主体区域结构清晰、细节丰富，辅助区域与远景适度概括，逐步降低对比度与细节密度。严禁全图铺满均匀碎点、杂乱线网、重复纹理或琐碎色斑，确保核心焦点突出。 " +
	"【色彩与质感】 饱满、干净、哑光的东方传统设色（协调运用石青、石绿、朱砂、赭石、藤黄等传统色系，主次分明，点睛色克制集中）。色块内部具细微层次与自然叠色，搭配少量克制的墨黑阴影。通过精细线描与哑光设色自然呈现温润纸本感，无人工噪点与做旧磨损。 " +
	"【风格禁忌】 拒绝摄影写实、3D渲染、厚重水墨、油画笔触、日系动漫风、空洞极简色块、全图毛边及伪汉字。"

// The text rules of the two title modes. shihuaTitleMarker starts the part that names the texts the
// model writes; an image whose prompt has it carries a drawn title.
const (
	shihuaTextNoText  = "【文字与版式】 画面中不要出现任何文字、书法、印章、落款、题跋或伪文字。在画面右上方保留一块约占画面宽 40%、高 45% 的连续暖米白天空或云气留白，留白内不要任何景物（后期会用书法排版标题和正文）。"
	shihuaTextTitle   = "【文字与版式】 画面中只允许出现主标题和作者名两处文字：主标题采用大幅暖白色中文书法，竖排置于画面左上的高对比连续色面区，边缘干净无立体发光特效；作者名为小号端正楷体，在主标题下方。除此之外不要任何其它文字、印章、落款、题跋或伪文字。在画面右上方保留一块约占画面宽 30%、高 40% 的连续暖米白天空或云气留白，留白内不要任何景物和文字（后期会排版正文）。"
	shihuaTitleMarker = "【画面文字"
)

// shihuaTextSectionRe matches a 【文字与版式】 part in a saved style (up to the next 【 section), which
// the title mode replaces.
var shihuaTextSectionRe = regexp.MustCompile(`【文字与版式】[^【]*`)

// BuildShihuaPrompt is the full image prompt of a work: the style, the scene and the text rules. With
// titleByModel the model writes the title and the author's name; otherwise no text at all.
func BuildShihuaPrompt(style string, w *ShihuaWork, titleByModel bool) string {
	if strings.TrimSpace(style) == "" {
		style = DefaultShihuaStylePrompt
	}
	text := shihuaTextNoText
	if titleByModel {
		text = shihuaTextTitle
	}
	style = strings.TrimSpace(shihuaTextSectionRe.ReplaceAllString(style, ""))
	var b strings.Builder
	_, _ = b.WriteString(style)
	_, _ = b.WriteString(" ")
	_, _ = b.WriteString(text)
	_, _ = b.WriteString("\n主题： ")
	_, _ = b.WriteString(strings.TrimSpace(w.Theme.PromptTheme))
	if titleByModel {
		_, _ = b.WriteString("\n" + shihuaTitleMarker + "（必须逐字准确，不得增字、漏字、改字）】 主标题「")
		_, _ = b.WriteString(w.Title)
		_, _ = b.WriteString("」大字书法竖排")
		if author := strings.TrimSpace(w.Author); author != "" {
			_, _ = b.WriteString("；作者名「")
			_, _ = b.WriteString(author)
			_, _ = b.WriteString("」小字")
		}
		_, _ = b.WriteString("。")
	}
	_, _ = b.WriteString(" 出图")
	return b.String()
}

// ShihuaCharCount counts the Chinese characters of a text (punctuation and spaces excluded).
func ShihuaCharCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			n++
		}
	}
	return n
}

type ShihuaService struct {
	repo     ShihuaRepository
	settings SettingRepository
	keys     shihuaKeyReader
	billing  *BillingService
	tools    *ImageToolsService
	files    *shihuaFiles
	now      func() time.Time

	draw shihuaDrawFunc
	chat shihuaChatFunc
	// wikisource is where 「AI 添加篇目」 fetches texts (tests point it at a fake).
	wikisource *shihuaWikisource
	upscale    shihuaUpscaleFunc

	mu      sync.Mutex
	running int
	wake    chan struct{}
	stop    chan struct{}
	wg      sync.WaitGroup
}

func (s *ShihuaService) Settings(ctx context.Context) ShihuaSettings {
	st := ShihuaSettings{TextModel: shihuaDefaultTextModel, Size: shihuaDefaultSize, Quality: shihuaDefaultQuality,
		Concurrency: shihuaDefaultConcurrent, DailyLimit: shihuaDefaultDailyLimit, AutoUpscale: true}
	raw, err := s.settings.GetValue(ctx, SettingKeyShihua)
	if err != nil || strings.TrimSpace(raw) == "" {
		return st
	}
	_ = json.Unmarshal([]byte(raw), &st)
	normalizeShihuaSettings(&st)
	return st
}

func normalizeShihuaSettings(st *ShihuaSettings) {
	st.TextModel = strings.TrimSpace(st.TextModel)
	if st.TextModel == "" {
		st.TextModel = shihuaDefaultTextModel
	}
	if !shihuaSizes[st.Size] {
		st.Size = shihuaDefaultSize
	}
	if !shihuaQualities[st.Quality] {
		st.Quality = shihuaDefaultQuality
	}
	st.Concurrency = min(max(st.Concurrency, 1), shihuaMaxConcurrent)
	st.DailyLimit = min(max(st.DailyLimit, 0), shihuaMaxDailyLimit)
	st.StylePrompt = strings.TrimSpace(st.StylePrompt)
	if st.StylePrompt == DefaultShihuaStylePrompt {
		st.StylePrompt = ""
	}
}

func (s *ShihuaService) SettingsView(ctx context.Context, adminID int64) (*ShihuaSettingsView, error) {
	st := s.Settings(ctx)
	stats, err := s.repo.QueueStats(ctx, shihuaDayStart(s.now()))
	if err != nil {
		return nil, err
	}
	stats.Running = max(stats.Running, s.runningCount())
	v := &ShihuaSettingsView{ShihuaSettings: st, DefaultStylePrompt: DefaultShihuaStylePrompt, UpscaleAvailable: s.upscale != nil, Queue: stats}
	v.Paused = s.pausedReason(ctx, st, stats)
	return v, nil
}

func (s *ShihuaService) pausedReason(ctx context.Context, st ShihuaSettings, stats ShihuaQueueStats) string {
	if _, err := s.adminKey(ctx, st.KeyID, 0); err != nil {
		return "没有可用的 API Key，请在设置里选一把（No API key）"
	}
	if stats.StartedToday >= st.DailyLimit {
		return fmt.Sprintf("今天已画 %d 张，达到每日上限，明天 0 点（北京时间）继续（Daily limit reached）", stats.StartedToday)
	}
	return ""
}

func (s *ShihuaService) SaveSettings(ctx context.Context, adminID int64, st ShihuaSettings) (*ShihuaSettingsView, error) {
	normalizeShihuaSettings(&st)
	if utf8.RuneCountInString(st.StylePrompt) > 3000 || len(st.TextModel) > 100 {
		return nil, ErrShihuaInvalid
	}
	if st.KeyID != 0 {
		if _, err := s.adminKey(ctx, st.KeyID, adminID); err != nil {
			return nil, err
		}
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, SettingKeyShihua, string(raw)); err != nil {
		return nil, err
	}
	s.kick()
	return s.SettingsView(ctx, adminID)
}

// adminKey returns the configured key; ownerID > 0 also checks who owns it (when it is being set).
func (s *ShihuaService) adminKey(ctx context.Context, keyID, ownerID int64) (*APIKey, error) {
	if keyID <= 0 || s.keys == nil {
		return nil, ErrShihuaKey
	}
	k, err := s.keys.GetByID(ctx, keyID)
	if err != nil || k == nil || !k.IsActive() || k.Key == "" || (ownerID > 0 && k.UserID != ownerID) {
		return nil, ErrShihuaKey
	}
	if k.Group != nil && k.Group.Platform != "" && k.Group.Platform != PlatformOpenAI {
		return nil, ErrShihuaKey
	}
	return k, nil
}

// ---- works ----

var shihuaSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var shihuaGenres = map[string]bool{"shi": true, "ci": true, "qu": true, "wen": true, "fu": true}
var shihuaStages = map[string]bool{"": true, "primary": true, "junior": true, "senior": true, "legacy": true}

// ShihuaWorkInput is what the admin edits (and what an import file holds per work).
type ShihuaWorkInput struct {
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Author      string           `json:"author"`
	Dynasty     string           `json:"dynasty"`
	Genre       string           `json:"genre"`
	Stage       string           `json:"stage"`
	Grade       string           `json:"grade"`
	Textbooks   []ShihuaTextbook `json:"textbooks"`
	FullText    string           `json:"full_text"`
	DisplayText string           `json:"display_text"`
	ExcerptNote string           `json:"excerpt_note"`
	FamousLines string           `json:"famous_lines"`
	Sources     string           `json:"sources"`
	VariantNote string           `json:"variant_note"`
	Sort        int              `json:"sort"`
	Theme       *ShihuaTheme     `json:"theme,omitempty"`
}

func validateShihuaWork(in *ShihuaWorkInput) error {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Title = strings.TrimSpace(in.Title)
	in.Author = strings.TrimSpace(in.Author)
	in.Dynasty = strings.TrimSpace(in.Dynasty)
	in.Genre = strings.TrimSpace(in.Genre)
	if in.Genre == "" {
		in.Genre = "shi"
	}
	in.Stage = strings.TrimSpace(in.Stage)
	in.Grade = strings.TrimSpace(in.Grade)
	in.FullText = normalizeShihuaText(in.FullText)
	in.DisplayText = normalizeShihuaText(in.DisplayText)
	if in.DisplayText == in.FullText {
		in.DisplayText = ""
	}
	if in.Textbooks == nil {
		in.Textbooks = []ShihuaTextbook{}
	}
	switch {
	case !shihuaSlugRe.MatchString(in.Slug), in.Title == "", utf8.RuneCountInString(in.Title) > 32,
		utf8.RuneCountInString(in.Author) > 32, utf8.RuneCountInString(in.Dynasty) > 8, utf8.RuneCountInString(in.Grade) > 16,
		!shihuaGenres[in.Genre], !shihuaStages[in.Stage], in.FullText == "",
		utf8.RuneCountInString(in.FullText) > shihuaMaxTextRunes, utf8.RuneCountInString(in.DisplayText) > shihuaMaxTextRunes,
		len(in.Textbooks) > 20:
		return ErrShihuaInvalid
	}
	if in.Theme != nil && utf8.RuneCountInString(in.Theme.PromptTheme) > 1500 {
		return ErrShihuaInvalid
	}
	return nil
}

// normalizeShihuaText trims each line and drops blank ones at the ends; paragraphs stay on their own
// lines.
func normalizeShihuaText(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

func (s *ShihuaService) ListWorks(ctx context.Context, q ShihuaWorkQuery) ([]ShihuaWork, error) {
	works, err := s.repo.ListWorks(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range works {
		fillShihuaCounts(&works[i])
	}
	return works, nil
}

func fillShihuaCounts(w *ShihuaWork) {
	w.CharCount = ShihuaCharCount(w.FullText)
	w.DisplayCharCount = ShihuaCharCount(w.PosterText())
}

func (s *ShihuaService) GetWork(ctx context.Context, id int64) (*ShihuaWork, error) {
	w, err := s.repo.GetWork(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrShihuaWorkNotFound
	}
	fillShihuaCounts(w)
	return w, nil
}

func (s *ShihuaService) CreateWork(ctx context.Context, in ShihuaWorkInput) (*ShihuaWork, error) {
	if err := validateShihuaWork(&in); err != nil {
		return nil, err
	}
	if other, err := s.repo.GetWorkBySlug(ctx, in.Slug); err != nil {
		return nil, err
	} else if other != nil {
		return nil, ErrShihuaSlugTaken
	}
	w := &ShihuaWork{TextStatus: ShihuaTextDraft}
	applyShihuaInput(w, in)
	if err := s.repo.CreateWork(ctx, w); err != nil {
		return nil, err
	}
	return s.GetWork(ctx, w.ID)
}

func applyShihuaInput(w *ShihuaWork, in ShihuaWorkInput) {
	// A changed poster text needs checking again.
	if w.FullText != in.FullText || w.PosterText() != firstNonEmptyString(in.DisplayText, in.FullText) {
		w.TextStatus, w.Verify1By, w.Verify1At, w.Verify2By, w.Verify2At = ShihuaTextDraft, nil, nil, nil, nil
	}
	w.Slug, w.Title, w.Author, w.Dynasty, w.Genre, w.Stage, w.Grade = in.Slug, in.Title, in.Author, in.Dynasty, in.Genre, in.Stage, in.Grade
	w.Textbooks, w.FullText, w.DisplayText, w.ExcerptNote = in.Textbooks, in.FullText, in.DisplayText, strings.TrimSpace(in.ExcerptNote)
	w.FamousLines, w.Sources, w.VariantNote, w.Sort = strings.TrimSpace(in.FamousLines), strings.TrimSpace(in.Sources), strings.TrimSpace(in.VariantNote), in.Sort
	if in.Theme != nil {
		w.Theme = *in.Theme
		w.Theme.PromptTheme = strings.TrimSpace(w.Theme.PromptTheme)
	}
}

func (s *ShihuaService) UpdateWork(ctx context.Context, id int64, in ShihuaWorkInput) (*ShihuaWork, error) {
	if err := validateShihuaWork(&in); err != nil {
		return nil, err
	}
	w, err := s.GetWork(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Slug != w.Slug {
		if other, err := s.repo.GetWorkBySlug(ctx, in.Slug); err != nil {
			return nil, err
		} else if other != nil && other.ID != id {
			return nil, ErrShihuaSlugTaken
		}
	}
	applyShihuaInput(w, in)
	if err := s.repo.UpdateWork(ctx, w); err != nil {
		return nil, err
	}
	return s.GetWork(ctx, id)
}

func (s *ShihuaService) DeleteWork(ctx context.Context, id int64) error {
	if _, err := s.GetWork(ctx, id); err != nil {
		return err
	}
	if err := s.repo.DeleteWork(ctx, id); err != nil {
		return err
	}
	s.files.removeWork(id)
	return nil
}

type ShihuaImportResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Unchanged int      `json:"unchanged"`
	Reset     []string `json:"reset"` // slugs whose text changed, so their checks were cleared
	Errors    []string `json:"errors"`
}

// Import adds or updates works by slug. dryRun reports what would happen without writing.
func (s *ShihuaService) Import(ctx context.Context, items []ShihuaWorkInput, dryRun bool) (*ShihuaImportResult, error) {
	if len(items) == 0 || len(items) > shihuaMaxImportWorks {
		return nil, ErrShihuaInvalid
	}
	res := &ShihuaImportResult{Reset: []string{}, Errors: []string{}}
	seen := map[string]bool{}
	for i, in := range items {
		if err := validateShihuaWork(&in); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("第 %d 条（%s）：%s", i+1, firstNonEmptyString(in.Slug, in.Title, "?"), infraerrors.Message(err)))
			continue
		}
		if seen[in.Slug] {
			res.Errors = append(res.Errors, fmt.Sprintf("第 %d 条：slug %s 重复", i+1, in.Slug))
			continue
		}
		seen[in.Slug] = true
		existing, err := s.repo.GetWorkBySlug(ctx, in.Slug)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			res.Created++
			if !dryRun {
				w := &ShihuaWork{TextStatus: ShihuaTextDraft}
				applyShihuaInput(w, in)
				if err := s.repo.CreateWork(ctx, w); err != nil {
					return nil, err
				}
			}
			continue
		}
		if in.Theme == nil { // an import without a scene keeps the one already written
			in.Theme = &existing.Theme
		}
		before, _ := json.Marshal(existing)
		wasStatus := existing.TextStatus
		applyShihuaInput(existing, in)
		after, _ := json.Marshal(existing)
		if string(before) == string(after) {
			res.Unchanged++
			continue
		}
		res.Updated++
		if wasStatus != ShihuaTextDraft && existing.TextStatus == ShihuaTextDraft {
			res.Reset = append(res.Reset, in.Slug)
		}
		if !dryRun {
			if err := s.repo.UpdateWork(ctx, existing); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// Verify records a proofreading pass: level 1, then level 2 (by another admin when the settings ask
// for it). level 0 clears both.
func (s *ShihuaService) Verify(ctx context.Context, adminID, id int64, level int) (*ShihuaWork, error) {
	w, err := s.GetWork(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.now()
	switch level {
	case 0:
		w.TextStatus, w.Verify1By, w.Verify1At, w.Verify2By, w.Verify2At = ShihuaTextDraft, nil, nil, nil, nil
	case 1:
		w.TextStatus, w.Verify1By, w.Verify1At, w.Verify2By, w.Verify2At = ShihuaTextVerified1, &adminID, &now, nil, nil
	case 2:
		if w.TextStatus != ShihuaTextVerified1 && w.TextStatus != ShihuaTextVerified2 {
			return nil, ErrShihuaVerifyOrder
		}
		if s.Settings(ctx).TwoPeopleVerify && w.Verify1By != nil && *w.Verify1By == adminID {
			return nil, ErrShihuaVerifySame
		}
		w.TextStatus, w.Verify2By, w.Verify2At = ShihuaTextVerified2, &adminID, &now
	default:
		return nil, ErrShihuaInvalid
	}
	if err := s.repo.UpdateWork(ctx, w); err != nil {
		return nil, err
	}
	return s.GetWork(ctx, id)
}

// ---- jobs ----

type ShihuaJobInput struct {
	Name       string  `json:"name"`
	WorkIDs    []int64 `json:"work_ids"`
	Candidates int     `json:"candidates"`
}

// CreateJob queues Candidates drawings for each work.
func (s *ShihuaService) CreateJob(ctx context.Context, adminID int64, in ShihuaJobInput) (*ShihuaJob, error) {
	in.Name = strings.TrimSpace(in.Name)
	if len(in.WorkIDs) == 0 || len(in.WorkIDs) > shihuaMaxJobWorks || in.Candidates < 1 || in.Candidates > shihuaMaxCandidates ||
		utf8.RuneCountInString(in.Name) > 64 {
		return nil, ErrShihuaInvalid
	}
	works, err := s.repo.ListWorks(ctx, ShihuaWorkQuery{IDs: in.WorkIDs})
	if err != nil {
		return nil, err
	}
	if len(works) != len(uniqueInt64s(in.WorkIDs)) {
		return nil, ErrShihuaWorkNotFound
	}
	st := s.Settings(ctx)
	var missing []string
	images := make([]ShihuaImage, 0, len(works)*in.Candidates)
	for i := range works {
		w := &works[i]
		if strings.TrimSpace(w.Theme.PromptTheme) == "" {
			missing = append(missing, w.Title)
			continue
		}
		prompt := BuildShihuaPrompt(st.StylePrompt, w, st.TitleByModel)
		for n := 0; n < in.Candidates; n++ {
			images = append(images, ShihuaImage{WorkID: w.ID, Status: ShihuaImageQueued, Prompt: prompt, Size: st.Size, Quality: st.Quality})
		}
	}
	if len(missing) > 0 {
		return nil, infraerrors.BadRequest("SHIHUA_NO_THEME", "这些篇目还没有画面描述："+strings.Join(missing, "、")+"（Some works have no scene description）")
	}
	if in.Name == "" {
		in.Name = fmt.Sprintf("%d 篇 × %d 张", len(works), in.Candidates)
	}
	job, err := s.repo.CreateJob(ctx, in.Name, in.Candidates, adminID, images)
	if err != nil {
		return nil, err
	}
	s.kick()
	return job, nil
}

func (s *ShihuaService) ListJobs(ctx context.Context) ([]ShihuaJob, error) {
	return s.repo.ListJobs(ctx, 50)
}

func (s *ShihuaService) CancelJob(ctx context.Context, jobID int64) (int64, error) {
	return s.repo.CancelQueued(ctx, jobID)
}

func (s *ShihuaService) RetryFailed(ctx context.Context, jobID int64) (int64, error) {
	n, err := s.repo.RetryFailed(ctx, jobID)
	if err == nil && n > 0 {
		s.kick()
	}
	return n, err
}

type ShihuaRedrawInput struct {
	Count       int    `json:"count"`
	PromptTheme string `json:"prompt_theme"` // optional new scene, saved on the work
}

// Redraw queues more drawings of one work, optionally with a revised scene.
func (s *ShihuaService) Redraw(ctx context.Context, workID int64, in ShihuaRedrawInput) ([]ShihuaImage, error) {
	if in.Count < 1 || in.Count > shihuaMaxCandidates || utf8.RuneCountInString(in.PromptTheme) > 1500 {
		return nil, ErrShihuaInvalid
	}
	w, err := s.GetWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	if theme := strings.TrimSpace(in.PromptTheme); theme != "" && theme != w.Theme.PromptTheme {
		w.Theme.PromptTheme = theme
		if err := s.repo.UpdateWork(ctx, w); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(w.Theme.PromptTheme) == "" {
		return nil, ErrShihuaNoTheme
	}
	st := s.Settings(ctx)
	prompt := BuildShihuaPrompt(st.StylePrompt, w, st.TitleByModel)
	images := make([]ShihuaImage, in.Count)
	for i := range images {
		images[i] = ShihuaImage{WorkID: w.ID, Status: ShihuaImageQueued, Prompt: prompt, Size: st.Size, Quality: st.Quality}
	}
	if err := s.repo.EnqueueImages(ctx, images); err != nil {
		return nil, err
	}
	s.kick()
	return images, nil
}

// ---- images ----

func (s *ShihuaService) ListImages(ctx context.Context, q ShihuaImageQuery) ([]ShihuaImage, error) {
	if q.Limit <= 0 || q.Limit > 2000 {
		q.Limit = 2000
	}
	images, err := s.repo.ListImages(ctx, q)
	if err != nil {
		return nil, err
	}
	for i := range images {
		s.files.signImage(&images[i])
	}
	return images, nil
}

func (s *ShihuaService) getImage(ctx context.Context, id int64) (*ShihuaImage, error) {
	img, err := s.repo.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, ErrShihuaImageNotFound
	}
	return img, nil
}

type ShihuaReviewInput struct {
	Action     string `json:"action"`      // pick / unpick / reject / restore / title
	TitleCheck string `json:"title_check"` // ok / wrong / "" (action title)
}

// Review picks (one per work), rejects or marks the title of a finished drawing.
func (s *ShihuaService) Review(ctx context.Context, id int64, in ShihuaReviewInput) (*ShihuaImage, error) {
	img, err := s.getImage(ctx, id)
	if err != nil {
		return nil, err
	}
	if img.Status != ShihuaImageSucceeded {
		return nil, ErrShihuaInvalid
	}
	switch in.Action {
	case "pick":
		if err := s.repo.ReviewImage(ctx, id, false, "ok"); err != nil {
			return nil, err
		}
		if err := s.repo.SetPicked(ctx, img.WorkID, &id); err != nil {
			return nil, err
		}
		if s.Settings(ctx).AutoUpscale && s.upscale != nil && img.UpscaleStatus == "" {
			if err := s.repo.SetUpscale(ctx, id, ShihuaUpscaleQueued, "", ""); err != nil {
				return nil, err
			}
			s.kick()
		}
	case "unpick":
		w, err := s.GetWork(ctx, img.WorkID)
		if err != nil {
			return nil, err
		}
		if w.PickedImageID != nil && *w.PickedImageID == id {
			if err := s.repo.SetPicked(ctx, img.WorkID, nil); err != nil {
				return nil, err
			}
		}
	case "reject", "restore":
		if in.Action == "reject" {
			if w, err := s.GetWork(ctx, img.WorkID); err == nil && w.PickedImageID != nil && *w.PickedImageID == id {
				if err := s.repo.SetPicked(ctx, img.WorkID, nil); err != nil {
					return nil, err
				}
			}
		}
		if err := s.repo.ReviewImage(ctx, id, in.Action == "reject", img.TitleCheck); err != nil {
			return nil, err
		}
	case "title":
		if in.TitleCheck != "" && in.TitleCheck != "ok" && in.TitleCheck != "wrong" {
			return nil, ErrShihuaInvalid
		}
		if err := s.repo.ReviewImage(ctx, id, img.Rejected, in.TitleCheck); err != nil {
			return nil, err
		}
	case "upscale":
		if s.upscale == nil {
			return nil, infraerrors.BadRequest("SHIHUA_NO_UPSCALE", "服务器没有配置超分服务（imagetools）（Upscaling is not available）")
		}
		if err := s.repo.SetUpscale(ctx, id, ShihuaUpscaleQueued, "", img.UpscaledFile); err != nil {
			return nil, err
		}
		s.kick()
	default:
		return nil, ErrShihuaInvalid
	}
	img, err = s.getImage(ctx, id)
	if err != nil {
		return nil, err
	}
	s.files.signImage(img)
	return img, nil
}

// ---- renders ----

var shihuaAspects = map[string]bool{"2x3": true, "3x4": true, "9x16": true, "1x1": true, "16x9": true, "235x100": true}

// SaveRender stores a finished poster the admin page rendered from the picked drawing.
func (s *ShihuaService) SaveRender(ctx context.Context, imageID int64, aspect string, layout json.RawMessage, data []byte) (*ShihuaRender, error) {
	if !shihuaAspects[aspect] {
		return nil, ErrShihuaInvalid
	}
	img, err := s.getImage(ctx, imageID)
	if err != nil {
		return nil, err
	}
	w, err := s.GetWork(ctx, img.WorkID)
	if err != nil {
		return nil, err
	}
	if w.PickedImageID == nil || *w.PickedImageID != imageID {
		return nil, ErrShihuaNotPicked
	}
	if w.TextStatus != ShihuaTextVerified2 {
		return nil, ErrShihuaNotVerified
	}
	if len(layout) == 0 || !json.Valid(layout) || len(layout) > 64<<10 {
		layout = json.RawMessage("{}")
	}
	file, width, height, err := s.files.saveRender(w.ID, imageID, aspect, data)
	if err != nil {
		return nil, err
	}
	r := &ShihuaRender{ImageID: imageID, WorkID: w.ID, Aspect: aspect, File: file, Width: width, Height: height, Layout: layout}
	if err := s.repo.SaveRender(ctx, r); err != nil {
		return nil, err
	}
	r.URL = s.files.sign(r.File)
	return r, nil
}

func (s *ShihuaService) ListRenders(ctx context.Context, workIDs []int64) ([]ShihuaRender, error) {
	renders, err := s.repo.ListRenders(ctx, workIDs)
	if err != nil {
		return nil, err
	}
	for i := range renders {
		renders[i].URL = s.files.sign(renders[i].File)
	}
	return renders, nil
}

// shihuaDayStart is midnight in Beijing time, when the daily limit resets.
func shihuaDayStart(now time.Time) time.Time {
	loc := time.FixedZone("CST", 8*3600)
	t := now.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
