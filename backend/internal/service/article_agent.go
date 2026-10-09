package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // gpt-image returns PNG
	"log/slog"
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
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"golang.org/x/sync/errgroup"
)

// AI 写文章 (公众号排版 → AI 写文章): an agent that writes a 公众号 article for the user in two runs.
//
//  1. Outline: optionally searching the web (the gateway's 联网搜索 providers, Tavily / Brave), the
//     model drafts three titles, a digest, the sections with their points and where pictures go.
//     The user edits or regenerates it, then confirms.
//  2. Article: the model writes the Markdown (pictures as ![alt](img:N)), then gpt-image-2 draws
//     the cover and the pictures, saved as JPEG files under <dir>/<id>/.
//
// Both runs call this site's own gateway with one of the user's own keys (billed as usual), run in
// the background and keep going when the tab is closed. Pushing to the 公众号 draft box stays in
// the editor: the AppSecret lives only in the user's browser, and the user confirms the push.

const (
	ArticleOutlining = "outlining"
	ArticleOutlined  = "outline_ready"
	ArticleWriting   = "writing"
	ArticleDrawing   = "drawing"
	ArticleDone      = "done"
	ArticleFailed    = "failed"
	ArticleCanceled  = "canceled"

	articleMaxActivePerUser = 2
	articleMaxRunsSiteWide  = 6
	articleRunTimeout       = 15 * time.Minute
	articleRetention        = 30 * 24 * time.Hour
	articleListLimit        = 30
	articleMaxTopic         = 200
	articleMaxMaterials     = 8000
	articleMaxFeedback      = 500
	articleMaxImages        = 4
	articleMaxSearches      = 4
	articleOutlineTokens    = 3000
	articleWriteTokens      = 8000
	articleImageParallel    = 2
	articleImageSize        = "1536x1024"
	articleSaveEvery        = 3 * time.Second
	articleJPEGMaxBytes     = 900 << 10 // WeChat body pictures must stay under 1 MB
)

var (
	ErrArticleNotFound = infraerrors.NotFound("ARTICLE_NOT_FOUND", "没有找到这篇文章（Article not found）")
	ErrArticleTopic    = infraerrors.BadRequest("ARTICLE_TOPIC", fmt.Sprintf("请填写文章主题，最多 %d 字；参考资料最多 %d 字（Topic required）", articleMaxTopic, articleMaxMaterials))
	ErrArticleBusy     = infraerrors.TooManyRequests("ARTICLE_BUSY", fmt.Sprintf("你已经有 %d 篇文章在生成，等它们完成后再开始新的（Too many running articles）", articleMaxActivePerUser))
	ErrArticleState    = infraerrors.Conflict("ARTICLE_STATE", "文章正在生成，或者当前状态不能这样操作，刷新后再试（Not allowed in the current state）")
	ErrArticleOutline  = infraerrors.BadRequest("ARTICLE_OUTLINE", "大纲至少要有一个标题和一个小节（Outline needs a title and a section）")
	ErrArticleSiteBusy = infraerrors.TooManyRequests("ARTICLE_SITE_BUSY", "现在写文章的人太多，稍后再试（Server busy）")
)

// ArticleBrief is what the user asks for.
type ArticleBrief struct {
	Topic     string `json:"topic"`
	Materials string `json:"materials,omitempty"`
	Audience  string `json:"audience,omitempty"`
	Tone      string `json:"tone,omitempty"`
	Length    string `json:"length"` // short | standard | long
	Images    int    `json:"images"` // body pictures, 0–4; a cover is always drawn
	Search    bool   `json:"search"`
}

type ArticleSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

type ArticleOutlineImage struct {
	Prompt string `json:"prompt"`
	Alt    string `json:"alt"`
}

type ArticleSection struct {
	Heading string               `json:"heading"`
	Points  []string             `json:"points"`
	Image   *ArticleOutlineImage `json:"image,omitempty"`
}

type ArticleOutline struct {
	Titles      []string         `json:"titles"`
	Title       string           `json:"title"`
	Digest      string           `json:"digest"`
	CoverPrompt string           `json:"cover_prompt"`
	Sections    []ArticleSection `json:"sections"`
}

// ArticleImage is a picture of the article: N 0 is the cover, then the body pictures in order.
type ArticleImage struct {
	N      int    `json:"n"`
	Kind   string `json:"kind"` // cover | body
	Prompt string `json:"prompt"`
	Alt    string `json:"alt"`
	Status string `json:"status"` // pending | ok | failed
	Error  string `json:"error,omitempty"`
	Bytes  int    `json:"bytes,omitempty"`
}

type ArticleEvent struct {
	At   time.Time `json:"at"`
	Kind string    `json:"kind"` // step | done | error | info
	Text string    `json:"text"`
}

// ArticleData is everything about an article besides its row columns (stored as JSON).
type ArticleData struct {
	Brief            ArticleBrief    `json:"brief"`
	Sources          []ArticleSource `json:"sources"`
	Outline          *ArticleOutline `json:"outline,omitempty"`
	Feedback         string          `json:"feedback,omitempty"`
	Markdown         string          `json:"markdown"`
	Images           []ArticleImage  `json:"images"`
	Events           []ArticleEvent  `json:"events"`
	Model            string          `json:"model,omitempty"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	ImagesDrawn      int             `json:"images_drawn"`
	Searches         int             `json:"searches"`
	PushedAt         *time.Time      `json:"pushed_at,omitempty"`
}

type ArticleProject struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	KeyID     int64     `json:"key_id"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ArticleData
}

func articleActive(status string) bool {
	return status == ArticleOutlining || status == ArticleWriting || status == ArticleDrawing
}

type ArticleProjectRepository interface {
	Create(ctx context.Context, p *ArticleProject) error
	Get(ctx context.Context, userID, id int64) (*ArticleProject, error) // nil when not found
	List(ctx context.Context, userID int64, limit int) ([]ArticleProject, error)
	Save(ctx context.Context, p *ArticleProject) error
	CountActive(ctx context.Context, userID int64) (int, error)
	FailActive(ctx context.Context, message string) (int64, error)
	Delete(ctx context.Context, userID, id int64) error
	DeleteBefore(ctx context.Context, before time.Time) ([]int64, error)
}

// ArticleSearcher searches the web (the gateway's 联网搜索 providers).
type ArticleSearcher interface {
	Available(ctx context.Context) bool
	Search(ctx context.Context, query string, max int) ([]websearch.SearchResult, error)
}

type articleChatFunc func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error)
type articleAgentFunc func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error)
type articleImageFunc func(ctx context.Context, key, prompt, size string) ([]byte, error)

type ArticleAgentService struct {
	repo     ArticleProjectRepository
	learn    *LearnService
	search   ArticleSearcher
	dir      string
	chat     articleChatFunc
	agent    articleAgentFunc
	draw     articleImageFunc
	now      func() time.Time
	slots    chan struct{}
	gateway  string
	runsMu   sync.Mutex
	runs     map[int64]context.CancelFunc
	wg       sync.WaitGroup
	savingMu sync.Mutex // one Save at a time per process (pictures finish in parallel)
}

func NewArticleAgentService(repo ArticleProjectRepository, learn *LearnService, search ArticleSearcher, dir string) *ArticleAgentService {
	s := &ArticleAgentService{repo: repo, learn: learn, search: search, dir: dir, now: time.Now,
		slots: make(chan struct{}, articleMaxRunsSiteWide), runs: map[int64]context.CancelFunc{}}
	if learn != nil {
		s.gateway = learn.gatewayURL
	}
	client := &http.Client{Timeout: 6 * time.Minute}
	s.chat = func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
		return streamChatCompletion(ctx, client, s.gateway, "hivegpt-article/1", key, model, messages, maxTokens, onDelta)
	}
	s.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		in.GatewayURL, in.UserAgent = s.gateway, "hivegpt-article/1"
		return RunAgent(ctx, client, in)
	}
	s.draw = func(ctx context.Context, key, prompt, size string) ([]byte, error) {
		return gatewayImage(ctx, client, s.gateway, "hivegpt-article/1", key, prompt, size)
	}
	return s
}

// RecoverInterrupted fails the runs a previous process left behind; the user can retry them.
func (s *ArticleAgentService) RecoverInterrupted(ctx context.Context) {
	if n, err := s.repo.FailActive(ctx, "服务器重启，生成中断了，点「重试」接着做（Interrupted by a server restart）"); err != nil {
		slog.Warn("article: failing interrupted runs", "err", err)
	} else if n > 0 {
		slog.Info("article: interrupted runs marked failed", "count", n)
	}
}

// Wait blocks until running articles stop (shutdown).
func (s *ArticleAgentService) Wait() { s.wg.Wait() }

// ArticleConfig is what the editor shows before starting.
type ArticleConfig struct {
	Search    bool `json:"search"`
	MaxImages int  `json:"max_images"`
}

func (s *ArticleAgentService) Config(ctx context.Context) ArticleConfig {
	return ArticleConfig{Search: s.search != nil && s.search.Available(ctx), MaxImages: articleMaxImages}
}

// ArticleCreateInput starts an article.
type ArticleCreateInput struct {
	KeyID int64 `json:"key_id"`
	ArticleBrief
}

func normalizeArticleBrief(b *ArticleBrief) error {
	b.Topic = strings.TrimSpace(b.Topic)
	b.Materials = strings.TrimSpace(b.Materials)
	b.Audience = articleClip(strings.TrimSpace(b.Audience), 100)
	b.Tone = articleClip(strings.TrimSpace(b.Tone), 100)
	if b.Topic == "" || utf8.RuneCountInString(b.Topic) > articleMaxTopic || utf8.RuneCountInString(b.Materials) > articleMaxMaterials {
		return ErrArticleTopic
	}
	if b.Length != "short" && b.Length != "long" {
		b.Length = "standard"
	}
	b.Images = max(0, min(b.Images, articleMaxImages))
	return nil
}

func articleClip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// Create saves the article and starts the outline run.
func (s *ArticleAgentService) Create(ctx context.Context, userID int64, in ArticleCreateInput) (*ArticleProject, error) {
	if err := normalizeArticleBrief(&in.ArticleBrief); err != nil {
		return nil, err
	}
	key, err := s.key(ctx, userID, in.KeyID)
	if err != nil {
		return nil, err
	}
	if n, err := s.repo.CountActive(ctx, userID); err != nil {
		return nil, err
	} else if n >= articleMaxActivePerUser {
		return nil, ErrArticleBusy
	}
	if in.Search && (s.search == nil || !s.search.Available(ctx)) {
		in.Search = false
	}
	p := &ArticleProject{UserID: userID, KeyID: in.KeyID, Status: ArticleOutlining, Title: articleClip(in.Topic, 200),
		ArticleData: ArticleData{Brief: in.ArticleBrief, Sources: []ArticleSource{}, Images: []ArticleImage{}, Events: []ArticleEvent{}}}
	s.event(p, "info", "开始：「"+articleClip(in.Topic, 60)+"」")
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.cleanup()
	runCtx, _ := s.claim(p.ID)
	s.start(runCtx, p, key, s.runOutline)
	return p, nil
}

func (s *ArticleAgentService) key(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if keyID <= 0 {
		return nil, ErrLearnOwnKeyRequired
	}
	if s.learn == nil {
		return nil, ErrLearnKeyInvalid
	}
	return s.learn.ownKey(ctx, userID, keyID)
}

func (s *ArticleAgentService) model(ctx context.Context) string {
	if s.learn != nil {
		if st, _ := s.learn.loadSettings(ctx); st.Model != "" {
			return st.Model
		}
	}
	return "gpt-5.5"
}

func (s *ArticleAgentService) Get(ctx context.Context, userID, id int64) (*ArticleProject, error) {
	p, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrArticleNotFound
	}
	return p, nil
}

func (s *ArticleAgentService) List(ctx context.Context, userID int64) ([]ArticleProject, error) {
	return s.repo.List(ctx, userID, articleListLimit)
}

// ArticleOutlineInput: either feedback to redo the outline, or the (edited) outline to confirm.
type ArticleOutlineInput struct {
	Outline  *ArticleOutline `json:"outline"`
	Feedback string          `json:"feedback"`
	Confirm  bool            `json:"confirm"`
}

// Outline regenerates the outline with feedback, or confirms it and starts writing.
func (s *ArticleAgentService) Outline(ctx context.Context, userID, id int64, in ArticleOutlineInput) (*ArticleProject, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != ArticleOutlined {
		return nil, ErrArticleState
	}
	key, err := s.key(ctx, userID, p.KeyID)
	if err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(id)
	if !ok {
		return nil, ErrArticleState
	}
	started := false
	defer func() {
		if !started {
			s.release(id)
		}
	}()
	if in.Outline != nil {
		o, err := cleanArticleOutline(*in.Outline, p.Brief.Images)
		if err != nil {
			return nil, err
		}
		p.Outline = o
	}
	if !in.Confirm {
		fb := strings.TrimSpace(in.Feedback)
		if fb == "" || utf8.RuneCountInString(fb) > articleMaxFeedback {
			return nil, infraerrors.BadRequest("ARTICLE_FEEDBACK", fmt.Sprintf("写下想怎么改，最多 %d 字（Feedback 1–%d characters）", articleMaxFeedback, articleMaxFeedback))
		}
		p.Feedback = fb
		p.Status = ArticleOutlining
		s.event(p, "info", "按你的意见重写大纲："+articleClip(fb, 80))
		if err := s.repo.Save(ctx, p); err != nil {
			return nil, err
		}
		started = true
		s.start(runCtx, p, key, s.runOutline)
		return p, nil
	}
	if p.Outline == nil || len(p.Outline.Sections) == 0 {
		return nil, ErrArticleOutline
	}
	p.Status = ArticleWriting
	p.Title = articleClip(p.Outline.Title, 200)
	s.event(p, "info", "大纲已确认，开始写全文")
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	started = true
	s.start(runCtx, p, key, s.runArticle)
	return p, nil
}

// Retry restarts a failed or canceled article from where it stopped.
func (s *ArticleAgentService) Retry(ctx context.Context, userID, id int64) (*ArticleProject, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	retryImages := p.Status == ArticleDone && articleImagesFailed(p.Images)
	if p.Status != ArticleFailed && p.Status != ArticleCanceled && !retryImages {
		return nil, ErrArticleState
	}
	key, err := s.key(ctx, userID, p.KeyID)
	if err != nil {
		return nil, err
	}
	if n, err := s.repo.CountActive(ctx, userID); err != nil {
		return nil, err
	} else if n >= articleMaxActivePerUser {
		return nil, ErrArticleBusy
	}
	runCtx, ok := s.claim(id)
	if !ok {
		return nil, ErrArticleState
	}
	p.Error = ""
	run := s.runArticle
	switch {
	case p.Outline == nil:
		p.Status = ArticleOutlining
		run = s.runOutline
	case strings.TrimSpace(p.Markdown) == "" && p.Status != ArticleDone:
		p.Status = ArticleWriting
	default:
		p.Status = ArticleDrawing
	}
	s.event(p, "info", "重试")
	if err := s.repo.Save(ctx, p); err != nil {
		s.release(id)
		return nil, err
	}
	s.start(runCtx, p, key, run)
	return p, nil
}

func articleImagesFailed(images []ArticleImage) bool {
	for _, im := range images {
		if im.Status != "ok" {
			return true
		}
	}
	return false
}

// Cancel stops a running article.
func (s *ArticleAgentService) Cancel(ctx context.Context, userID, id int64) error {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return err
	}
	s.runsMu.Lock()
	cancel := s.runs[id]
	s.runsMu.Unlock()
	if cancel == nil {
		return ErrArticleState
	}
	cancel()
	return nil
}

// Pushed records that the editor pushed the article to the draft box.
func (s *ArticleAgentService) Pushed(ctx context.Context, userID, id int64) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if p.Status != ArticleDone {
		return ErrArticleState
	}
	now := s.now()
	p.PushedAt = &now
	s.event(p, "done", "已推送到公众号草稿箱")
	return s.repo.Save(ctx, p)
}

func (s *ArticleAgentService) Delete(ctx context.Context, userID, id int64) error {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return err
	}
	if s.running(id) || articleActive(p.Status) {
		return ErrArticleState
	}
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return err
	}
	_ = os.RemoveAll(s.projectDir(id))
	return nil
}

// ImageFile is the path of one of the article's pictures (after checking it is the user's).
func (s *ArticleAgentService) ImageFile(ctx context.Context, userID, id int64, n int) (string, error) {
	p, err := s.Get(ctx, userID, id)
	if err != nil {
		return "", err
	}
	for _, im := range p.Images {
		if im.N == n && im.Status == "ok" {
			return s.imagePath(id, n), nil
		}
	}
	return "", ErrArticleNotFound
}

func (s *ArticleAgentService) projectDir(id int64) string {
	return filepath.Join(s.dir, strconv.FormatInt(id, 10))
}

func (s *ArticleAgentService) imagePath(id int64, n int) string {
	return filepath.Join(s.projectDir(id), strconv.Itoa(n)+".jpg")
}

func (s *ArticleAgentService) cleanup() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ids, err := s.repo.DeleteBefore(ctx, s.now().Add(-articleRetention))
		if err != nil {
			return
		}
		for _, id := range ids {
			_ = os.RemoveAll(s.projectDir(id))
		}
	}()
}

func (s *ArticleAgentService) running(id int64) bool {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	return s.runs[id] != nil
}

func (s *ArticleAgentService) event(p *ArticleProject, kind, text string) {
	p.Events = append(p.Events, ArticleEvent{At: s.now(), Kind: kind, Text: text})
	if len(p.Events) > 200 {
		p.Events = p.Events[len(p.Events)-200:]
	}
}

func (s *ArticleAgentService) save(p *ArticleProject) {
	s.savingMu.Lock()
	defer s.savingMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.repo.Save(ctx, p); err != nil {
		slog.Warn("article: save", "id", p.ID, "err", err)
	}
}

// claim reserves the article for one run (false when it already has one); release undoes it.
func (s *ArticleAgentService) claim(id int64) (context.Context, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if s.runs[id] != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), articleRunTimeout)
	s.runs[id] = cancel
	return ctx, true
}

func (s *ArticleAgentService) release(id int64) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if cancel := s.runs[id]; cancel != nil {
		cancel()
		delete(s.runs, id)
	}
}

// start runs fn in the background on a claimed article; the outcome is always saved.
func (s *ArticleAgentService) start(ctx context.Context, p *ArticleProject, key *APIKey, fn func(ctx context.Context, p *ArticleProject, key string) error) {
	cancel := func() { s.release(p.ID) }
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		case <-ctx.Done():
			p.Status, p.Error = ArticleFailed, ErrArticleSiteBusy.Error()
			s.save(p)
			return
		}
		err := fn(ctx, p, key.Key)
		switch {
		case err == nil:
		case errors.Is(ctx.Err(), context.Canceled):
			p.Status, p.Error = ArticleCanceled, ""
			s.event(p, "error", "已停止")
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			p.Status, p.Error = ArticleFailed, "生成超时（超过 15 分钟），点「重试」接着做"
			s.event(p, "error", p.Error)
		default:
			msg := articleErrorText(p.ID, err)
			p.Status, p.Error = ArticleFailed, msg
			s.event(p, "error", "出错了："+msg)
		}
		s.save(p)
	}()
}

// articleErrorText: our own errors carry a user-facing message; anything else is logged and shown
// as a generic one.
func articleErrorText(id int64, err error) string {
	var app *infraerrors.ApplicationError
	if errors.As(err, &app) && app.Message != "" {
		return app.Message
	}
	slog.Warn("article: run failed", "id", id, "err", err)
	return "生成出错了，点「重试」再来一次"
}

func (s *ArticleAgentService) usage(p *ArticleProject, model string, prompt, completion int) {
	p.PromptTokens += prompt
	p.CompletionTokens += completion
	if model != "" {
		p.Model = model
	}
}

// --- run 1: research + outline -----------------------------------------------------------------

func (s *ArticleAgentService) runOutline(ctx context.Context, p *ArticleProject, key string) error {
	var tools []AgentTool
	calls := 2 // one retry when the outline isn't valid JSON
	if p.Brief.Search && s.search != nil && s.search.Available(ctx) {
		s.event(p, "step", "搜索资料…")
		s.save(p)
		tools = []AgentTool{s.searchTool(p)}
		calls = articleMaxSearches + 1
	} else {
		s.event(p, "step", "构思大纲…")
		s.save(p)
	}
	msgs := []AgentMessage{agentText("system", articleOutlinePrompt(p.Brief, tools != nil)), agentText("user", articleOutlineRequest(p))}
	model := s.model(ctx)
	res, err := s.agent(ctx, AgentRunInput{Key: key, Model: model, Messages: msgs, Tools: tools, MaxModelCalls: calls,
		MaxOutputTokens: articleOutlineTokens,
		OnTool: func(_, _ string) error {
			return ctx.Err()
		}})
	if res != nil {
		s.usage(p, res.Model, res.PromptTokens, res.CompletionTokens)
	}
	if err != nil {
		return err
	}
	outline, perr := parseArticleOutline(res.Text, p.Brief.Images)
	if perr != nil {
		// One more try without tools, showing the model what went wrong.
		msgs = append(msgs, agentText("assistant", articleClip(res.Text, 6000)), agentText("user", "上面的输出不是要求的 JSON（"+perr.Error()+"）。只输出一个合法的 JSON 对象，不要任何其他文字。"))
		res2, err := s.agent(ctx, AgentRunInput{Key: key, Model: model, Messages: msgs, MaxModelCalls: 1, MaxOutputTokens: articleOutlineTokens})
		if res2 != nil {
			s.usage(p, res2.Model, res2.PromptTokens, res2.CompletionTokens)
		}
		if err != nil {
			return err
		}
		if outline, perr = parseArticleOutline(res2.Text, p.Brief.Images); perr != nil {
			return errLearnRunFailed("模型没有给出可用的大纲，点「重试」再来一次")
		}
	}
	p.Outline = outline
	p.Feedback = ""
	p.Title = articleClip(outline.Title, 200)
	p.Status = ArticleOutlined
	s.event(p, "done", fmt.Sprintf("大纲好了：%d 个小节。看看是否满意，可以直接修改，或写下意见让 AI 重写", len(outline.Sections)))
	return nil
}

// searchTool lets the outline run search the web; results become the article's sources.
func (s *ArticleAgentService) searchTool(p *ArticleProject) AgentTool {
	seen := map[string]bool{}
	for _, src := range p.Sources {
		seen[src.URL] = true
	}
	return AgentTool{
		Name:        "web_search",
		Description: fmt.Sprintf("联网搜索最新资料。每次一个查询，最多 %d 次；用和主题相关的具体关键词，中文主题优先用中文。", articleMaxSearches),
		Label:       "正在搜索",
		Parameters: map[string]any{"type": "object", "required": []string{"query"}, "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "搜索关键词"}}},
		Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var args struct {
				Query string `json:"query"`
			}
			_ = json.Unmarshal(raw, &args)
			q := articleClip(strings.TrimSpace(args.Query), 200)
			if q == "" {
				return nil, errors.New("query 不能为空")
			}
			if p.Searches >= articleMaxSearches*2 {
				return nil, errors.New("搜索次数已用完，请根据已有资料出大纲")
			}
			p.Searches++
			s.event(p, "step", "搜索：「"+q+"」")
			s.save(p)
			results, err := s.search.Search(ctx, q, 6)
			if err != nil {
				return nil, errors.New("搜索服务暂时不可用，请根据已有资料出大纲")
			}
			out := []map[string]string{}
			for _, r := range results {
				out = append(out, map[string]string{"title": r.Title, "url": r.URL, "snippet": articleClip(r.Snippet, 400), "age": r.PageAge})
				if r.URL != "" && !seen[r.URL] && len(p.Sources) < 20 {
					seen[r.URL] = true
					p.Sources = append(p.Sources, ArticleSource{Title: articleClip(r.Title, 120), URL: r.URL, Snippet: articleClip(r.Snippet, 400)})
				}
			}
			return map[string]any{"query": q, "results": out}, nil
		},
	}
}

var articleJSONFence = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

func parseArticleOutline(text string, images int) (*ArticleOutline, error) {
	text = strings.TrimSpace(text)
	if m := articleJSONFence.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	if i, j := strings.Index(text, "{"), strings.LastIndex(text, "}"); i >= 0 && j > i {
		text = text[i : j+1]
	}
	var o ArticleOutline
	if err := json.Unmarshal([]byte(text), &o); err != nil {
		return nil, errors.New("JSON 解析失败")
	}
	return cleanArticleOutline(o, images)
}

// cleanArticleOutline trims an outline (from the model or edited by the user) to sane sizes and
// keeps at most `images` body pictures.
func cleanArticleOutline(o ArticleOutline, images int) (*ArticleOutline, error) {
	out := ArticleOutline{Digest: articleClip(strings.TrimSpace(o.Digest), 120), CoverPrompt: articleClip(strings.TrimSpace(o.CoverPrompt), 600)}
	for _, t := range o.Titles {
		if t = articleClip(strings.TrimSpace(t), 64); t != "" && len(out.Titles) < 5 {
			out.Titles = append(out.Titles, t)
		}
	}
	out.Title = articleClip(strings.TrimSpace(o.Title), 64)
	if out.Title == "" && len(out.Titles) > 0 {
		out.Title = out.Titles[0]
	}
	pictures := 0
	for _, sec := range o.Sections {
		h := articleClip(strings.TrimSpace(sec.Heading), 60)
		if h == "" || len(out.Sections) >= 12 {
			continue
		}
		clean := ArticleSection{Heading: h, Points: []string{}}
		for _, pt := range sec.Points {
			if pt = articleClip(strings.TrimSpace(pt), 200); pt != "" && len(clean.Points) < 8 {
				clean.Points = append(clean.Points, pt)
			}
		}
		if sec.Image != nil && strings.TrimSpace(sec.Image.Prompt) != "" && pictures < images {
			pictures++
			clean.Image = &ArticleOutlineImage{Prompt: articleClip(strings.TrimSpace(sec.Image.Prompt), 600), Alt: articleClip(strings.TrimSpace(sec.Image.Alt), 60)}
		}
		out.Sections = append(out.Sections, clean)
	}
	if out.Title == "" || len(out.Sections) == 0 {
		return nil, ErrArticleOutline
	}
	if out.CoverPrompt == "" {
		out.CoverPrompt = "公众号文章封面插画，主题：" + out.Title
	}
	return &out, nil
}

// --- run 2: write + draw ---------------------------------------------------------------------------

func (s *ArticleAgentService) runArticle(ctx context.Context, p *ArticleProject, key string) error {
	if p.Status == ArticleWriting || strings.TrimSpace(p.Markdown) == "" {
		if err := s.write(ctx, p, key); err != nil {
			return err
		}
	}
	p.Status = ArticleDrawing
	if err := s.drawAll(ctx, p, key); err != nil {
		return err
	}
	p.Status = ArticleDone
	ok := 0
	for _, im := range p.Images {
		if im.Status == "ok" {
			ok++
		}
	}
	if failed := len(p.Images) - ok; failed > 0 {
		s.event(p, "done", fmt.Sprintf("文章写好了，%d 张图画好、%d 张失败（可以点「重试」补画，或在编辑器里换图）", ok, failed))
	} else {
		s.event(p, "done", fmt.Sprintf("文章和 %d 张配图都好了", ok))
	}
	return nil
}

func (s *ArticleAgentService) write(ctx context.Context, p *ArticleProject, key string) error {
	p.Status = ArticleWriting
	p.Markdown = ""
	s.event(p, "step", "写全文…")
	s.save(p)
	var b strings.Builder
	last := s.now()
	msgs := []LearnMessage{{Role: "system", Content: articleWritePrompt(p.Brief)}, {Role: "user", Content: articleWriteRequest(p)}}
	res, err := s.chat(ctx, key, s.model(ctx), msgs, articleWriteTokens, func(d string) error {
		_, _ = b.WriteString(d)
		if s.now().Sub(last) >= articleSaveEvery {
			last = s.now()
			p.Markdown = b.String()
			s.save(p)
		}
		return nil
	})
	if res != nil {
		s.usage(p, res.Model, res.PromptTokens, res.CompletionTokens)
	}
	if err != nil {
		p.Markdown = ""
		return err
	}
	md := articleFixImages(articleStripFence(b.String()), p.Outline)
	if utf8.RuneCountInString(md) < 200 {
		return errLearnRunFailed("模型写出来的内容太短，点「重试」再来一次")
	}
	p.Markdown = md
	p.Images = articleImageSlots(p.Outline)
	s.event(p, "step", fmt.Sprintf("全文写好了，约 %d 字", utf8.RuneCountInString(md)))
	return nil
}

func articleStripFence(md string) string {
	md = strings.TrimSpace(md)
	if strings.HasPrefix(md, "```") {
		if i := strings.Index(md, "\n"); i > 0 {
			md = md[i+1:]
		}
		md = strings.TrimSuffix(strings.TrimSpace(md), "```")
	}
	return strings.TrimSpace(md)
}

var articleImgRef = regexp.MustCompile(`!\[([^\]]*)\]\(img:(\d+)\)`)

// articleFixImages keeps only the picture placeholders the outline has, and adds missing ones at
// the end of their section (or the end of the article).
func articleFixImages(md string, o *ArticleOutline) string {
	want := map[int]ArticleOutlineImage{}
	n := 0
	if o != nil {
		for _, sec := range o.Sections {
			if sec.Image != nil {
				n++
				want[n] = *sec.Image
			}
		}
	}
	have := map[int]bool{}
	md = articleImgRef.ReplaceAllStringFunc(md, func(m string) string {
		k, _ := strconv.Atoi(articleImgRef.FindStringSubmatch(m)[2])
		if _, ok := want[k]; !ok || have[k] {
			return ""
		}
		have[k] = true
		return m
	})
	for k := 1; k <= n; k++ {
		if !have[k] {
			md += fmt.Sprintf("\n\n![%s](img:%d)", want[k].Alt, k)
		}
	}
	return strings.TrimSpace(md)
}

func articleImageSlots(o *ArticleOutline) []ArticleImage {
	out := []ArticleImage{}
	if o == nil {
		return out
	}
	out = append(out, ArticleImage{N: 0, Kind: "cover", Prompt: o.CoverPrompt, Alt: o.Title, Status: "pending"})
	n := 0
	for _, sec := range o.Sections {
		if sec.Image != nil {
			n++
			out = append(out, ArticleImage{N: n, Kind: "body", Prompt: sec.Image.Prompt, Alt: sec.Image.Alt, Status: "pending"})
		}
	}
	return out
}

func (s *ArticleAgentService) drawAll(ctx context.Context, p *ArticleProject, key string) error {
	if len(p.Images) == 0 {
		p.Images = articleImageSlots(p.Outline)
	}
	todo := 0
	for _, im := range p.Images {
		if im.Status != "ok" {
			todo++
		}
	}
	if todo == 0 {
		return nil
	}
	s.event(p, "step", fmt.Sprintf("画封面和配图（%d 张）…", todo))
	s.save(p)
	if err := os.MkdirAll(s.projectDir(p.ID), 0o755); err != nil {
		return errLearnRunFailed("保存图片失败")
	}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(articleImageParallel)
	for i := range p.Images {
		if p.Images[i].Status == "ok" {
			continue
		}
		g.Go(func() error {
			mu.Lock()
			im := p.Images[i]
			mu.Unlock()
			data, err := s.drawOne(gctx, key, im)
			if err == nil {
				err = os.WriteFile(s.imagePath(p.ID, im.N), data, 0o644)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				msg := articleErrorText(p.ID, err)
				p.Images[i].Status, p.Images[i].Error = "failed", msg
				s.event(p, "error", articleImageName(im)+"没画成："+msg)
			} else {
				p.Images[i].Status, p.Images[i].Error, p.Images[i].Bytes = "ok", "", len(data)
				p.ImagesDrawn++
				s.event(p, "step", articleImageName(im)+"画好了")
			}
			s.save(p)
			return nil
		})
	}
	return g.Wait()
}

func articleImageName(im ArticleImage) string {
	if im.Kind == "cover" {
		return "封面"
	}
	return fmt.Sprintf("配图 %d ", im.N)
}

// drawOne draws a picture and turns it into a JPEG small enough for WeChat.
func (s *ArticleAgentService) drawOne(ctx context.Context, key string, im ArticleImage) ([]byte, error) {
	prompt := im.Prompt + "\n画面里不要出现任何文字、字母、Logo 和水印。"
	if im.Kind == "cover" {
		prompt = im.Prompt + "\n这是公众号文章封面，横版构图，主体放在画面中间（两侧会被裁掉一部分）。画面里不要出现任何文字、字母、Logo 和水印。"
	}
	raw, err := s.draw(ctx, key, prompt, articleImageSize)
	if err != nil {
		return nil, err
	}
	return articleJPEG(raw)
}

func articleJPEG(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, errLearnRunFailed("画图服务返回的图片无法识别")
	}
	for _, q := range []int{88, 80, 70, 60} {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, err
		}
		if buf.Len() <= articleJPEGMaxBytes || q == 60 {
			return buf.Bytes(), nil
		}
	}
	return nil, errLearnRunFailed("图片压缩失败")
}
