package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AI 做贴图 (公众号排版 → AI 做贴图): an agent that turns a poem into a 公众号贴图 (newspic).
//
//  1. Suggest (only when the input is a theme): three poems that carry many details, checked against
//     the poem library. The user picks one.
//  2. Plan: the text comes from the poem library and 维基文库; where they differ the best-known reading
//     is chosen. The model decides on a story series or close-ups of one scene and writes concepts,
//     a period-correct detail list, the panels with one 「找一找」 detail each, titles and a question.
//     The user edits, asks for changes or confirms.
//  3. Covers: three candidates (title and author lettered by the model, no body text). The user picks.
//  4. Panels: drawn with the chosen cover as the reference picture, so characters and style hold.
//  5. Check: a vision model looks at every picture (wrong title characters, stray text, anachronisms,
//     story order, the 「找一找」 detail, readability at feed-card size); the post text is composed.
//
// All runs call this site's gateway with one of the user's own keys (billed as usual) and keep going
// when the tab is closed. Framing to 3:4 and pushing to the draft box happen in the editor (the
// AppSecret lives only in the user's browser). Rules come from the operator's own tests (2026-11).

const (
	TietuSuggesting = "suggesting"
	TietuSuggested  = "suggest_ready"
	TietuPlanning   = "planning"
	TietuPlanned    = "plan_ready"
	TietuCovering   = "covering"
	TietuCovered    = "cover_ready"
	TietuDrawing    = "drawing"
	TietuChecking   = "checking"
	TietuDone       = "done"
	TietuFailed     = "failed"
	TietuCanceled   = "canceled"

	tietuMaxActivePerUser = 1
	tietuMaxRunsSiteWide  = 3
	tietuRunTimeout       = 30 * time.Minute
	tietuRetention        = 30 * 24 * time.Hour
	tietuListLimit        = 30
	tietuMaxInput         = 100
	tietuMaxNotes         = 500
	tietuMaxFeedback      = 500
	tietuMaxStyle         = 2000
	tietuMinPanels        = 2
	tietuMaxPanels        = 8
	tietuDefaultPanels    = 6
	tietuCovers           = 3
	tietuImageParallel    = 3
	tietuImageSize        = "1024x1536"
	tietuImageQuality     = "high"
	tietuPlanTokens       = 8000
	tietuSuggestTokens    = 2500
	tietuCheckTokens      = 1500
	tietuVariantTokens    = 1500
)

var (
	ErrTietuNotFound = infraerrors.NotFound("TIETU_NOT_FOUND", "没有找到这条贴图（Not found）")
	ErrTietuInput    = infraerrors.BadRequest("TIETU_INPUT", fmt.Sprintf("请填写篇名（可带作者）或一个主题，最多 %d 字（Input required）", tietuMaxInput))
	ErrTietuBusy     = infraerrors.TooManyRequests("TIETU_BUSY", "你已经有一条贴图在生成，等它完成或停止后再开始新的（One run at a time）")
	ErrTietuState    = infraerrors.Conflict("TIETU_STATE", "贴图正在生成，或者当前状态不能这样操作，刷新后再试（Not allowed in the current state）")
	ErrTietuPlan     = infraerrors.BadRequest("TIETU_PLAN", "方案至少要有封面画面和两幕分镜（Plan needs a cover and panels）")
	ErrTietuSiteBusy = infraerrors.TooManyRequests("TIETU_SITE_BUSY", "现在做贴图的人太多，稍后再试（Server busy）")
	ErrTietuAdmin    = infraerrors.Forbidden("TIETU_ADMIN", "AI 做贴图暂时只对管理员开放（Admins only for now）")
	ErrTietuPoem     = infraerrors.BadRequest("TIETU_POEM", "诗词库和维基文库里都没找到这首诗，换个写法试试，例如「篇名 作者」（Poem not found）")
)

// TietuBrief is what the user asks for.
type TietuBrief struct {
	Input     string `json:"input"`                // 篇名（可带作者）或主题
	Theme     bool   `json:"theme"`                // true: Input is a theme, suggest poems first
	Mode      string `json:"mode"`                 // auto | single | series
	Panels    int    `json:"panels"`               // 分镜 / 特写张数（不含封面）
	Style     string `json:"style"`                // 风格预设 id，custom 用 StyleText
	StyleText string `json:"style_text,omitempty"` // 自定义风格提示词
	Notes     string `json:"notes,omitempty"`      // 额外要求
}

type TietuSuggestion struct {
	Title   string `json:"title"`
	Author  string `json:"author"`
	Dynasty string `json:"dynasty,omitempty"`
	Reason  string `json:"reason"`
	LibID   int64  `json:"lib_id,omitempty"`
}

type TietuSource struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text"`
}

// TietuVariant is one place where the sources read differently, and the reading that was used.
type TietuVariant struct {
	Options []string `json:"options"`
	Chosen  string   `json:"chosen"`
	Context string   `json:"context,omitempty"`
	Why     string   `json:"why,omitempty"`
}

type TietuPoem struct {
	Title    string         `json:"title"`
	Author   string         `json:"author"`
	Dynasty  string         `json:"dynasty"`
	Text     string         `json:"text"`
	LibID    int64          `json:"lib_id,omitempty"`
	Sources  []TietuSource  `json:"sources"`
	Variants []TietuVariant `json:"variants"`
	Agreed   bool           `json:"agreed"` // two sources agree (after choosing the variants)
	Note     string         `json:"note,omitempty"`
}

type TietuConcept struct {
	Name string `json:"name"`
	Idea string `json:"idea"`
}

// TietuScene is the cover or one panel.
type TietuScene struct {
	Caption string   `json:"caption"` // 正文里这一幕的小标题
	Line    string   `json:"line"`    // 一句话说明
	Scene   string   `json:"scene"`   // 画面描述
	Details []string `json:"details"` // 可细看的小细节
	Find    string   `json:"find"`    // 「找一找」
}

type TietuPlan struct {
	Kind       string         `json:"kind"` // series | single
	KindReason string         `json:"kind_reason"`
	Concepts   []TietuConcept `json:"concepts"`
	Concept    int            `json:"concept"`
	Character  string         `json:"character"`
	Color      string         `json:"color"`
	Cover      TietuScene     `json:"cover"`
	Panels     []TietuScene   `json:"panels"`
	Titles     []string       `json:"titles"`
	Question   string         `json:"question"`
}

type TietuCheck struct {
	OK          bool     `json:"ok"`
	Issues      []string `json:"issues"`
	Minor       []string `json:"minor"`
	FindVisible bool     `json:"find_visible"`
	FindSeen    string   `json:"find_seen,omitempty"`
	Focal       string   `json:"focal,omitempty"` // clear | weak
	Skipped     bool     `json:"skipped,omitempty"`
}

// TietuImage is a cover candidate (c1..c3) or a panel (p1..p8).
type TietuImage struct {
	Name   string      `json:"name"`
	Kind   string      `json:"kind"` // candidate | panel
	N      int         `json:"n"`
	Note   string      `json:"note,omitempty"` // the user's redraw note
	Status string      `json:"status"`         // pending | ok | failed
	Error  string      `json:"error,omitempty"`
	Check  *TietuCheck `json:"check,omitempty"`
}

type TietuPost struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type TietuData struct {
	Brief            TietuBrief        `json:"brief"`
	Suggestions      []TietuSuggestion `json:"suggestions"`
	Poem             *TietuPoem        `json:"poem,omitempty"`
	Plan             *TietuPlan        `json:"plan,omitempty"`
	Feedback         string            `json:"feedback,omitempty"`
	CoverNote        string            `json:"cover_note,omitempty"`
	Covers           []TietuImage      `json:"covers"`
	CoverPick        int               `json:"cover_pick"` // 1..3, 0 = not picked
	Panels           []TietuImage      `json:"panels"`
	Post             *TietuPost        `json:"post,omitempty"`
	PostEdited       bool              `json:"post_edited,omitempty"` // the user saved changes: keep them on redraws
	Events           []ArticleEvent    `json:"events"`
	Model            string            `json:"model,omitempty"`
	PromptTokens     int               `json:"prompt_tokens"`
	CompletionTokens int               `json:"completion_tokens"`
	ImagesDrawn      int               `json:"images_drawn"`
	PushedAt         *time.Time        `json:"pushed_at,omitempty"`
}

type TietuProject struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	KeyID     int64     `json:"key_id"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	TietuData
}

func tietuActive(status string) bool {
	switch status {
	case TietuSuggesting, TietuPlanning, TietuCovering, TietuDrawing, TietuChecking:
		return true
	}
	return false
}

type TietuProjectRepository interface {
	Create(ctx context.Context, p *TietuProject) error
	Get(ctx context.Context, userID, id int64) (*TietuProject, error) // nil when not found
	List(ctx context.Context, userID int64, limit int) ([]TietuProject, error)
	Save(ctx context.Context, p *TietuProject) error
	CountActive(ctx context.Context, userID int64) (int, error)
	FailActive(ctx context.Context, message string) (int64, error)
	Delete(ctx context.Context, userID, id int64) error
	DeleteBefore(ctx context.Context, before time.Time) ([]int64, error)
}

// TietuPoemSource looks poems up in the poem library (灵感库).
type TietuPoemSource interface {
	SearchPoems(ctx context.Context, q LibPoemQuery) ([]LibPoem, bool, error)
}

// tietuWiki reads 维基文库 (shihuaWikisource).
type tietuWiki interface {
	pageText(ctx context.Context, title string) (string, bool, error)
	search(ctx context.Context, q string) ([]string, error)
	pageURL(title string) string
}

type tietuChatFunc func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int) (string, *LearnTutorResult, error)
type tietuDrawFunc func(ctx context.Context, key, prompt string) ([]byte, error)
type tietuEditFunc func(ctx context.Context, key, prompt string, ref []byte) ([]byte, error)

type TietuService struct {
	repo     TietuProjectRepository
	learn    *LearnService
	poems    TietuPoemSource
	wiki     tietuWiki
	pricer   ArticleImagePricer
	dir      string
	chat     tietuChatFunc
	draw     tietuDrawFunc
	edit     tietuEditFunc
	now      func() time.Time
	slots    chan struct{}
	gateway  string
	runsMu   sync.Mutex
	runs     map[int64]context.CancelFunc
	wg       sync.WaitGroup
	savingMu sync.Mutex
}

func NewTietuService(repo TietuProjectRepository, learn *LearnService, poems TietuPoemSource, dir string) *TietuService {
	s := &TietuService{repo: repo, learn: learn, poems: poems, dir: dir, now: time.Now,
		wiki:  newShihuaWikisource(shihuaWikisourceBase),
		slots: make(chan struct{}, tietuMaxRunsSiteWide), runs: map[int64]context.CancelFunc{}}
	if learn != nil {
		s.gateway = learn.gatewayURL
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	s.chat = func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int) (string, *LearnTutorResult, error) {
		var b strings.Builder
		res, err := streamChatCompletion(ctx, client, s.gateway, "hivegpt-tietu/1", key, model, messages, maxTokens, func(d string) error {
			_, _ = b.WriteString(d)
			return nil
		})
		return b.String(), res, err
	}
	s.draw = func(ctx context.Context, key, prompt string) ([]byte, error) {
		return gatewayImageQuality(ctx, client, s.gateway, "hivegpt-tietu/1", key, prompt, tietuImageSize, tietuImageQuality)
	}
	s.edit = func(ctx context.Context, key, prompt string, ref []byte) ([]byte, error) {
		return gatewayImageEdit(ctx, client, s.gateway, "hivegpt-tietu/1", key, prompt, tietuImageSize, tietuImageQuality, ref)
	}
	return s
}

func (s *TietuService) SetPricer(p ArticleImagePricer) { s.pricer = p }

func (s *TietuService) RecoverInterrupted(ctx context.Context) {
	if n, err := s.repo.FailActive(ctx, "服务器重启，生成中断了，点「重试」接着做（Interrupted by a server restart）"); err != nil {
		slog.Warn("tietu: failing interrupted runs", "err", err)
	} else if n > 0 {
		slog.Info("tietu: interrupted runs marked failed", "count", n)
	}
}

func (s *TietuService) Wait() { s.wg.Wait() }

// TietuConfig is what the editor shows before starting.
type TietuConfig struct {
	Styles        []TietuStyleOption `json:"styles"`
	DefaultPanels int                `json:"default_panels"`
	MaxPanels     int                `json:"max_panels"`
	Covers        int                `json:"covers"`
	ImagePrice    *float64           `json:"image_price,omitempty"`
}

func (s *TietuService) Config(ctx context.Context, userID, keyID int64) TietuConfig {
	out := TietuConfig{Styles: tietuStyleOptions(), DefaultPanels: tietuDefaultPanels, MaxPanels: tietuMaxPanels, Covers: tietuCovers}
	if s.pricer == nil || keyID <= 0 {
		return out
	}
	if key, err := s.key(ctx, userID, keyID); err == nil {
		if price, ok := s.pricer.EstimateImageCost(ctx, key, editorImageModel, tietuImageSize); ok {
			price = math.Round(price*10000) / 10000
			out.ImagePrice = &price
		}
	}
	return out
}

type TietuCreateInput struct {
	KeyID int64 `json:"key_id"`
	TietuBrief
}

func normalizeTietuBrief(b *TietuBrief) error {
	b.Input = strings.TrimSpace(b.Input)
	b.Notes = articleClip(strings.TrimSpace(b.Notes), tietuMaxNotes)
	b.StyleText = articleClip(strings.TrimSpace(b.StyleText), tietuMaxStyle)
	if b.Input == "" || utf8.RuneCountInString(b.Input) > tietuMaxInput {
		return ErrTietuInput
	}
	if b.Mode != "single" && b.Mode != "series" {
		b.Mode = "auto"
	}
	if b.Panels == 0 {
		b.Panels = tietuDefaultPanels
	}
	b.Panels = max(tietuMinPanels, min(b.Panels, tietuMaxPanels))
	if b.Style == "custom" && b.StyleText == "" {
		b.Style = ""
	}
	if b.Style != "custom" && tietuStyleByID(b.Style) == nil {
		b.Style = tietuDefaultStyle
	}
	return nil
}

// Create saves the project and starts suggesting (theme) or planning (a poem).
func (s *TietuService) Create(ctx context.Context, userID int64, in TietuCreateInput) (*TietuProject, error) {
	if err := normalizeTietuBrief(&in.TietuBrief); err != nil {
		return nil, err
	}
	key, err := s.key(ctx, userID, in.KeyID)
	if err != nil {
		return nil, err
	}
	if n, err := s.repo.CountActive(ctx, userID); err != nil {
		return nil, err
	} else if n >= tietuMaxActivePerUser {
		return nil, ErrTietuBusy
	}
	p := &TietuProject{UserID: userID, KeyID: in.KeyID, Title: articleClip(in.Input, 200),
		TietuData: TietuData{Brief: in.TietuBrief, Suggestions: []TietuSuggestion{}, Covers: []TietuImage{}, Panels: []TietuImage{}, Events: []ArticleEvent{}}}
	run := s.runPlan
	if in.Theme {
		p.Status = TietuSuggesting
		run = s.runSuggest
		s.event(p, "info", "开始：按主题「"+articleClip(in.Input, 40)+"」挑诗")
	} else {
		p.Status = TietuPlanning
		s.event(p, "info", "开始：「"+articleClip(in.Input, 40)+"」")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.cleanup()
	runCtx, _ := s.claim(p.ID)
	s.start(runCtx, p, key, run)
	return p, nil
}

func (s *TietuService) key(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if keyID <= 0 {
		return nil, ErrLearnOwnKeyRequired
	}
	if s.learn == nil {
		return nil, ErrLearnKeyInvalid
	}
	return s.learn.ownKey(ctx, userID, keyID)
}

func (s *TietuService) model(ctx context.Context) string {
	if s.learn != nil {
		if st, _ := s.learn.loadSettings(ctx); st.Model != "" {
			return st.Model
		}
	}
	return "gpt-6.1-sol"
}

func (s *TietuService) Get(ctx context.Context, userID, id int64) (*TietuProject, error) {
	p, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrTietuNotFound
	}
	return p, nil
}

func (s *TietuService) List(ctx context.Context, userID int64) ([]TietuProject, error) {
	return s.repo.List(ctx, userID, tietuListLimit)
}

// resume claims a project for a new run after checking the state; the caller saves and starts it.
func (s *TietuService) resume(ctx context.Context, userID, id int64, states ...string) (*TietuProject, *APIKey, context.Context, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, nil, nil, err
	}
	allowed := false
	for _, st := range states {
		if p.Status == st {
			allowed = true
		}
	}
	if !allowed {
		return nil, nil, nil, ErrTietuState
	}
	key, err := s.key(ctx, userID, p.KeyID)
	if err != nil {
		return nil, nil, nil, err
	}
	runCtx, ok := s.claim(id)
	if !ok {
		return nil, nil, nil, ErrTietuState
	}
	return p, key, runCtx, nil
}

func (s *TietuService) launch(ctx context.Context, p *TietuProject, key *APIKey, runCtx context.Context, run func(context.Context, *TietuProject, string) error) (*TietuProject, error) {
	if err := s.repo.Save(ctx, p); err != nil {
		s.release(p.ID)
		return nil, err
	}
	s.start(runCtx, p, key, run)
	return p, nil
}

// Pick chooses one of the suggested poems (index) and starts planning.
func (s *TietuService) Pick(ctx context.Context, userID, id int64, index int) (*TietuProject, error) {
	p, key, runCtx, err := s.resume(ctx, userID, id, TietuSuggested)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= len(p.Suggestions) {
		s.release(id)
		return nil, infraerrors.BadRequest("TIETU_PICK", "请选择一首推荐的诗（Pick a suggestion）")
	}
	sg := p.Suggestions[index]
	p.Brief.Input = strings.TrimSpace(sg.Title + " " + sg.Author)
	p.Title = articleClip(sg.Title, 200)
	p.Status = TietuPlanning
	s.event(p, "info", "选了《"+sg.Title+"》")
	return s.launch(ctx, p, key, runCtx, s.runPlan)
}

// TietuPlanInput: feedback to redo the plan, or the (edited) plan to confirm.
type TietuPlanInput struct {
	Plan     *TietuPlan `json:"plan"`
	Feedback string     `json:"feedback"`
	Confirm  bool       `json:"confirm"`
}

func (s *TietuService) Plan(ctx context.Context, userID, id int64, in TietuPlanInput) (*TietuProject, error) {
	p, key, runCtx, err := s.resume(ctx, userID, id, TietuPlanned)
	if err != nil {
		return nil, err
	}
	if in.Plan != nil {
		plan, err := cleanTietuPlan(*in.Plan, p.Brief)
		if err != nil {
			s.release(id)
			return nil, err
		}
		p.Plan = plan
	}
	if !in.Confirm {
		fb := strings.TrimSpace(in.Feedback)
		if fb == "" || utf8.RuneCountInString(fb) > tietuMaxFeedback {
			s.release(id)
			return nil, infraerrors.BadRequest("TIETU_FEEDBACK", fmt.Sprintf("写下想怎么改，最多 %d 字（Feedback 1–%d characters）", tietuMaxFeedback, tietuMaxFeedback))
		}
		p.Feedback = fb
		p.Status = TietuPlanning
		s.event(p, "info", "按你的意见重新构思："+articleClip(fb, 80))
		return s.launch(ctx, p, key, runCtx, s.runPlan)
	}
	if p.Plan == nil || p.Poem == nil {
		s.release(id)
		return nil, ErrTietuPlan
	}
	p.Status = TietuCovering
	p.Covers, p.CoverPick, p.Panels, p.Post = []TietuImage{}, 0, []TietuImage{}, nil
	s.event(p, "info", "方案已确认，开始画封面")
	return s.launch(ctx, p, key, runCtx, s.runCovers)
}

// TietuCoverInput: pick a candidate (1..3) to go on, or redraw all three with a note.
type TietuCoverInput struct {
	Pick   int    `json:"pick"`
	Redraw bool   `json:"redraw"`
	Note   string `json:"note"`
}

func (s *TietuService) Cover(ctx context.Context, userID, id int64, in TietuCoverInput) (*TietuProject, error) {
	p, key, runCtx, err := s.resume(ctx, userID, id, TietuCovered)
	if err != nil {
		return nil, err
	}
	if in.Redraw {
		p.CoverNote = articleClip(strings.TrimSpace(in.Note), tietuMaxFeedback)
		p.Covers = []TietuImage{}
		p.Status = TietuCovering
		s.event(p, "info", "重画封面"+tietuNoteSuffix(p.CoverNote))
		return s.launch(ctx, p, key, runCtx, s.runCovers)
	}
	if in.Pick < 1 || in.Pick > len(p.Covers) || p.Covers[in.Pick-1].Status != "ok" {
		s.release(id)
		return nil, infraerrors.BadRequest("TIETU_COVER", "请选一张画好的封面（Pick a finished cover）")
	}
	p.CoverPick = in.Pick
	p.Panels = tietuPanelSlots(p.Plan)
	p.Post, p.PostEdited = nil, false
	p.Status = TietuDrawing
	s.event(p, "info", fmt.Sprintf("选了第 %d 张封面，开始画 %d 张分镜", in.Pick, len(p.Panels)))
	return s.launch(ctx, p, key, runCtx, s.runPanels)
}

// TietuRedrawInput redraws one panel (p1..p8) with an optional note, after the run is done.
type TietuRedrawInput struct {
	Name string `json:"name"`
	Note string `json:"note"`
}

func (s *TietuService) Redraw(ctx context.Context, userID, id int64, in TietuRedrawInput) (*TietuProject, error) {
	p, key, runCtx, err := s.resume(ctx, userID, id, TietuDone)
	if err != nil {
		return nil, err
	}
	found := false
	for i := range p.Panels {
		if p.Panels[i].Name == in.Name {
			found = true
			p.Panels[i].Status, p.Panels[i].Error, p.Panels[i].Check = "pending", "", nil
			p.Panels[i].Note = articleClip(strings.TrimSpace(in.Note), tietuMaxFeedback)
			s.event(p, "info", fmt.Sprintf("重画第 %d 张%s", p.Panels[i].N, tietuNoteSuffix(p.Panels[i].Note)))
		}
	}
	if !found {
		s.release(id)
		return nil, infraerrors.BadRequest("TIETU_REDRAW", "请选择要重画的分镜（Pick a panel）")
	}
	p.Status = TietuDrawing
	return s.launch(ctx, p, key, runCtx, s.runPanels)
}

// Retry restarts a failed or canceled run from where it stopped.
func (s *TietuService) Retry(ctx context.Context, userID, id int64) (*TietuProject, error) {
	p, key, runCtx, err := s.resume(ctx, userID, id, TietuFailed, TietuCanceled)
	if err != nil {
		return nil, err
	}
	if n, err := s.repo.CountActive(ctx, userID); err != nil || n >= tietuMaxActivePerUser {
		s.release(id)
		if err != nil {
			return nil, err
		}
		return nil, ErrTietuBusy
	}
	p.Error = ""
	var run func(context.Context, *TietuProject, string) error
	switch {
	case p.Brief.Theme && p.Poem == nil && len(p.Suggestions) == 0:
		p.Status, run = TietuSuggesting, s.runSuggest
	case p.Plan == nil:
		p.Status, run = TietuPlanning, s.runPlan
	case p.CoverPick == 0:
		p.Status, run = TietuCovering, s.runCovers
	default:
		p.Status, run = TietuDrawing, s.runPanels
	}
	s.event(p, "info", "重试")
	return s.launch(ctx, p, key, runCtx, run)
}

func (s *TietuService) Cancel(ctx context.Context, userID, id int64) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	s.runsMu.Lock()
	cancel := s.runs[id]
	s.runsMu.Unlock()
	if cancel == nil {
		return ErrTietuState
	}
	cancel()
	return nil
}

// TietuPostInput saves the user's edits to the title and body.
type TietuPostInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *TietuService) SavePost(ctx context.Context, userID, id int64, in TietuPostInput) (*TietuProject, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != TietuDone || s.running(id) {
		return nil, ErrTietuState
	}
	title, body := strings.TrimSpace(in.Title), strings.TrimSpace(in.Body)
	if title == "" || utf8.RuneCountInString(title) > 32 || body == "" || utf8.RuneCountInString(body) > 10000 {
		return nil, infraerrors.BadRequest("TIETU_POST", "标题 1–32 字，正文不能为空、最多 10000 字（Title 1–32, body required）")
	}
	p.Post = &TietuPost{Title: title, Body: body}
	p.PostEdited = true
	return p, s.repo.Save(ctx, p)
}

func (s *TietuService) Pushed(ctx context.Context, userID, id int64) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if p.Status != TietuDone {
		return ErrTietuState
	}
	now := s.now()
	p.PushedAt = &now
	s.event(p, "done", "已推送到公众号草稿箱")
	return s.repo.Save(ctx, p)
}

func (s *TietuService) Delete(ctx context.Context, userID, id int64) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if s.running(id) || tietuActive(p.Status) {
		return ErrTietuState
	}
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return err
	}
	_ = os.RemoveAll(s.projectDir(id))
	return nil
}

var tietuImageName = regexp.MustCompile(`^[cp][1-9]$`)

// ImageFile is the path of one of the project's pictures (c1..c3 covers, p1..p8 panels).
func (s *TietuService) ImageFile(ctx context.Context, userID, id int64, name string) (string, error) {
	if !tietuImageName.MatchString(name) {
		return "", ErrTietuNotFound
	}
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return "", err
	}
	for _, list := range [][]TietuImage{p.Covers, p.Panels} {
		for _, im := range list {
			if im.Name == name && im.Status == "ok" {
				return s.imagePath(id, name), nil
			}
		}
	}
	return "", ErrTietuNotFound
}

func (s *TietuService) projectDir(id int64) string {
	return filepath.Join(s.dir, strconv.FormatInt(id, 10))
}

func (s *TietuService) imagePath(id int64, name string) string {
	return filepath.Join(s.projectDir(id), name+".jpg")
}

func (s *TietuService) cleanup() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ids, err := s.repo.DeleteBefore(ctx, s.now().Add(-tietuRetention))
		if err != nil {
			return
		}
		for _, id := range ids {
			_ = os.RemoveAll(s.projectDir(id))
		}
	}()
}

func (s *TietuService) running(id int64) bool {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	return s.runs[id] != nil
}

func (s *TietuService) event(p *TietuProject, kind, text string) {
	p.Events = append(p.Events, ArticleEvent{At: s.now(), Kind: kind, Text: text})
	if len(p.Events) > 200 {
		p.Events = p.Events[len(p.Events)-200:]
	}
}

func (s *TietuService) save(p *TietuProject) {
	s.savingMu.Lock()
	defer s.savingMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.repo.Save(ctx, p); err != nil {
		slog.Warn("tietu: save", "id", p.ID, "err", err)
	}
}

func (s *TietuService) claim(id int64) (context.Context, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if s.runs[id] != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), tietuRunTimeout)
	s.runs[id] = cancel
	return ctx, true
}

func (s *TietuService) release(id int64) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if cancel := s.runs[id]; cancel != nil {
		cancel()
		delete(s.runs, id)
	}
}

func (s *TietuService) start(ctx context.Context, p *TietuProject, key *APIKey, fn func(ctx context.Context, p *TietuProject, key string) error) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.release(p.ID)
		var err error
		queued := false
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
			err = fn(ctx, p, key.Key)
		case <-ctx.Done():
			err, queued = ErrTietuSiteBusy, true
		}
		switch {
		case err == nil:
		case errors.Is(ctx.Err(), context.Canceled):
			p.Status, p.Error = TietuCanceled, ""
			s.event(p, "error", "已停止")
		case queued:
			p.Status, p.Error = TietuFailed, ErrTietuSiteBusy.Message
			s.event(p, "error", p.Error)
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			p.Status, p.Error = TietuFailed, "生成超时（超过 30 分钟），点「重试」接着做"
			s.event(p, "error", p.Error)
		default:
			msg := articleErrorText(p.ID, err)
			p.Status, p.Error = TietuFailed, msg
			s.event(p, "error", "出错了："+msg)
		}
		s.save(p)
	}()
}

func (s *TietuService) usage(p *TietuProject, res *LearnTutorResult) {
	if res == nil {
		return
	}
	p.PromptTokens += res.PromptTokens
	p.CompletionTokens += res.CompletionTokens
	if res.Model != "" {
		p.Model = res.Model
	}
}

func tietuNoteSuffix(note string) string {
	if note == "" {
		return ""
	}
	return "：" + articleClip(note, 60)
}
