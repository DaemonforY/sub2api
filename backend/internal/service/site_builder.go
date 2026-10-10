package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// AI 建站 (我的网站 → AI 生成网站): the user describes a site, the model writes a single-file HTML
// page, gpt-image-2 draws the pictures the page asks for (<img src="img/N.jpg" data-ai-prompt=...>),
// and the user keeps changing it by chat (each change rewrites the whole page; the last few versions
// are kept for undo). Publishing packs index.html and the pictures into a zip and goes through site
// hosting, so its subscription rule, quotas and content review apply unchanged.
//
// Runs call this site's own gateway with one of the user's own keys (billed as usual), run in the
// background and keep going when the tab is closed; the partial page is saved every few seconds so
// the preview grows while the model writes.

const (
	SiteDraftGenerating = "generating"
	SiteDraftDrawing    = "drawing"
	SiteDraftReady      = "ready"
	SiteDraftFailed     = "failed"

	siteBuilderMaxActivePerUser = 2
	siteBuilderMaxRunsSiteWide  = 4
	siteBuilderRunTimeout       = 20 * time.Minute
	siteBuilderRetention        = 90 * 24 * time.Hour
	siteBuilderListLimit        = 30
	siteBuilderMaxDescription   = 2000
	siteBuilderMaxStyle         = 200
	siteBuilderMaxInstruction   = 1000
	siteBuilderMaxImages        = 4
	siteBuilderDefaultImages    = 1
	siteBuilderMaxTokens        = 16000
	siteBuilderMaxHTML          = 200 << 10
	siteBuilderHistory          = 5
	siteBuilderMaxTurns         = 50
	siteBuilderImageParallel    = 2
	siteBuilderSaveEvery        = 3 * time.Second
	// Estimates: the upstream adds ~4000 input tokens to every request; a page is ~8000 tokens.
	siteBuilderEstPromptTokens = 4000 + 1500
	siteBuilderEstPageTokens   = 8000
)

var (
	ErrSiteDraftNotFound    = infraerrors.NotFound("SITE_DRAFT_NOT_FOUND", "没有找到这个网页草稿（Draft not found）")
	ErrSiteDraftDescription = infraerrors.BadRequest("SITE_DRAFT_DESCRIPTION", fmt.Sprintf("请描述你想要的网站，最多 %d 字（Description required）", siteBuilderMaxDescription))
	ErrSiteDraftInstruction = infraerrors.BadRequest("SITE_DRAFT_INSTRUCTION", fmt.Sprintf("请写下要怎么改，最多 %d 字（Instruction required）", siteBuilderMaxInstruction))
	ErrSiteDraftBusy        = infraerrors.TooManyRequests("SITE_DRAFT_BUSY", fmt.Sprintf("你已经有 %d 个网页在生成，等它们完成后再开始新的（Too many running drafts）", siteBuilderMaxActivePerUser))
	ErrSiteDraftState       = infraerrors.Conflict("SITE_DRAFT_STATE", "网页正在生成，或者当前状态不能这样操作，刷新后再试（Not allowed in the current state）")
	ErrSiteDraftSiteBusy    = infraerrors.TooManyRequests("SITE_DRAFT_SITE_BUSY", "现在生成网页的人太多，稍后再试（Server busy）")
	ErrSiteDraftNoUndo      = infraerrors.Conflict("SITE_DRAFT_NO_UNDO", "没有可以撤销的修改了（Nothing to undo）")
	ErrSiteDraftEmpty       = infraerrors.Conflict("SITE_DRAFT_EMPTY", "网页还没生成好，不能发布（The page is not ready）")
	ErrSiteDraftHosting     = infraerrors.ServiceUnavailable("SITE_DRAFT_HOSTING", "网站发布暂未开放（Site hosting is unavailable）")
)

// SiteDraftBrief is what the user asks for.
type SiteDraftBrief struct {
	Description string `json:"description"`
	Style       string `json:"style"`
	Images      int    `json:"images"`
}

// SiteDraftTurn is one request: the first description or a change.
type SiteDraftTurn struct {
	At          time.Time `json:"at"`
	Instruction string    `json:"instruction"`
	Status      string    `json:"status"` // running, ok, failed
	Error       string    `json:"error,omitempty"`
}

// SiteDraftImage is a picture the page asks for.
type SiteDraftImage struct {
	N      int    `json:"n"`
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
	Status string `json:"status"` // pending, ok, failed
	Error  string `json:"error,omitempty"`
	Bytes  int    `json:"bytes,omitempty"`
}

type SiteDraftData struct {
	Brief SiteDraftBrief `json:"brief"`
	// HTML is the current page; DraftHTML the one being written (shown while generating).
	HTML      string           `json:"html"`
	DraftHTML string           `json:"draft_html,omitempty"`
	History   []string         `json:"history,omitempty"`
	Turns     []SiteDraftTurn  `json:"turns"`
	Images    []SiteDraftImage `json:"images"`
	Events    []ArticleEvent   `json:"events"`
	// Published site (the last one this draft went to).
	SiteID      int64      `json:"site_id,omitempty"`
	SiteName    string     `json:"site_name,omitempty"`
	SiteURL     string     `json:"site_url,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`

	Model            string `json:"model,omitempty"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ImagesDrawn      int    `json:"images_drawn"`
}

type SiteDraft struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	KeyID     int64     `json:"key_id"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	Error     string    `json:"error"`
	CanUndo   bool      `json:"can_undo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	SiteDraftData
}

func siteDraftActive(status string) bool {
	return status == SiteDraftGenerating || status == SiteDraftDrawing
}

type SiteDraftRepository interface {
	Create(ctx context.Context, d *SiteDraft) error
	Get(ctx context.Context, userID, id int64) (*SiteDraft, error) // nil when not found
	List(ctx context.Context, userID int64, limit int) ([]SiteDraft, error)
	Save(ctx context.Context, d *SiteDraft) error
	CountActive(ctx context.Context, userID int64) (int, error)
	FailActive(ctx context.Context, message string) (int64, error)
	Delete(ctx context.Context, userID, id int64) error
	DeleteBefore(ctx context.Context, before time.Time) ([]int64, error)
}

// SiteBuilderPricer estimates calls on a key with the gateway's own pricing.
type SiteBuilderPricer interface {
	EstimateImageCost(ctx context.Context, apiKey *APIKey, model, size string) (float64, bool)
	EstimateTextCost(ctx context.Context, apiKey *APIKey, model string, tokens UsageTokens) (float64, bool)
}

// SiteBuilderPublisher is site hosting.
type SiteBuilderPublisher interface {
	Create(ctx context.Context, userID int64, up SiteUpload) (*Site, error)
	Update(ctx context.Context, userID, siteID int64, up SiteUpload) (*Site, error)
}

type SiteBuilderService struct {
	repo     SiteDraftRepository
	learn    *LearnService
	pricer   SiteBuilderPricer
	hosting  SiteBuilderPublisher
	dir      string
	chat     articleChatFunc
	draw     articleImageFunc
	now      func() time.Time
	slots    chan struct{}
	runsMu   sync.Mutex
	runs     map[int64]context.CancelFunc
	wg       sync.WaitGroup
	savingMu sync.Mutex
}

func NewSiteBuilderService(repo SiteDraftRepository, learn *LearnService, hosting SiteBuilderPublisher, dir string) *SiteBuilderService {
	s := &SiteBuilderService{repo: repo, learn: learn, hosting: hosting, dir: dir, now: time.Now,
		slots: make(chan struct{}, siteBuilderMaxRunsSiteWide), runs: map[int64]context.CancelFunc{}}
	gateway := ""
	if learn != nil {
		gateway = learn.gatewayURL
	}
	client := &http.Client{Timeout: 12 * time.Minute}
	s.chat = func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
		return streamChatCompletion(ctx, client, gateway, "hivegpt-sitebuilder/1", key, model, messages, maxTokens, onDelta)
	}
	s.draw = func(ctx context.Context, key, prompt, size string) ([]byte, error) {
		return gatewayImage(ctx, client, gateway, "hivegpt-sitebuilder/1", key, prompt, size)
	}
	return s
}

// SetPricer lets Config quote prices on the user's key.
func (s *SiteBuilderService) SetPricer(p SiteBuilderPricer) { s.pricer = p }

// RecoverInterrupted settles the runs a previous process left behind.
func (s *SiteBuilderService) RecoverInterrupted(ctx context.Context) {
	if n, err := s.repo.FailActive(ctx, "服务器重启，生成中断了，点「重试」接着做（Interrupted by a server restart）"); err != nil {
		slog.Warn("sitebuilder: failing interrupted runs", "err", err)
	} else if n > 0 {
		slog.Info("sitebuilder: interrupted runs settled", "count", n)
	}
}

// Wait blocks until running drafts stop (shutdown).
func (s *SiteBuilderService) Wait() { s.wg.Wait() }

// SiteBuilderConfig is what the page shows before starting (USD, as billed on the chosen key).
type SiteBuilderConfig struct {
	MaxImages     int      `json:"max_images"`
	DefaultImages int      `json:"default_images"`
	ImagePrice    *float64 `json:"image_price,omitempty"`
	PagePrice     *float64 `json:"page_price,omitempty"`
	RevisePrice   *float64 `json:"revise_price,omitempty"`
}

func roundPrice(v float64) *float64 {
	v = math.Round(v*10000) / 10000
	return &v
}

func (s *SiteBuilderService) Config(ctx context.Context, userID, keyID int64) SiteBuilderConfig {
	out := SiteBuilderConfig{MaxImages: siteBuilderMaxImages, DefaultImages: siteBuilderDefaultImages}
	if s.pricer == nil || keyID <= 0 {
		return out
	}
	key, err := s.key(ctx, userID, keyID)
	if err != nil {
		return out
	}
	if price, ok := s.pricer.EstimateImageCost(ctx, key, editorImageModel, "1536x1024"); ok {
		out.ImagePrice = roundPrice(price)
	}
	model := s.model(ctx)
	if price, ok := s.pricer.EstimateTextCost(ctx, key, model, UsageTokens{InputTokens: siteBuilderEstPromptTokens, OutputTokens: siteBuilderEstPageTokens}); ok {
		out.PagePrice = roundPrice(price)
	}
	if price, ok := s.pricer.EstimateTextCost(ctx, key, model, UsageTokens{InputTokens: siteBuilderEstPromptTokens + siteBuilderEstPageTokens, OutputTokens: siteBuilderEstPageTokens}); ok {
		out.RevisePrice = roundPrice(price)
	}
	return out
}

// SiteDraftCreateInput starts a page.
type SiteDraftCreateInput struct {
	KeyID int64 `json:"key_id"`
	SiteDraftBrief
}

func (s *SiteBuilderService) key(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if keyID <= 0 {
		return nil, ErrLearnOwnKeyRequired
	}
	if s.learn == nil {
		return nil, ErrLearnKeyInvalid
	}
	return s.learn.ownKey(ctx, userID, keyID)
}

func (s *SiteBuilderService) model(ctx context.Context) string {
	if s.learn != nil {
		if st, _ := s.learn.loadSettings(ctx); st.Model != "" {
			return st.Model
		}
	}
	return "gpt-5.5"
}

// Create saves the draft and starts writing the page.
func (s *SiteBuilderService) Create(ctx context.Context, userID int64, in SiteDraftCreateInput) (*SiteDraft, error) {
	in.Description = strings.TrimSpace(in.Description)
	in.Style = articleClip(strings.TrimSpace(in.Style), siteBuilderMaxStyle)
	if in.Description == "" || utf8.RuneCountInString(in.Description) > siteBuilderMaxDescription {
		return nil, ErrSiteDraftDescription
	}
	in.Images = max(0, min(in.Images, siteBuilderMaxImages))
	key, err := s.key(ctx, userID, in.KeyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkActive(ctx, userID); err != nil {
		return nil, err
	}
	d := &SiteDraft{UserID: userID, KeyID: in.KeyID, Status: SiteDraftGenerating, Title: articleClip(firstLine(in.Description), 60),
		SiteDraftData: SiteDraftData{Brief: in.SiteDraftBrief, Turns: []SiteDraftTurn{}, Images: []SiteDraftImage{}, Events: []ArticleEvent{}}}
	d.Turns = append(d.Turns, SiteDraftTurn{At: s.now(), Instruction: in.Description, Status: "running"})
	s.event(d, "info", "开始生成网页")
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	s.cleanup()
	runCtx, _ := s.claim(d.ID)
	s.start(runCtx, d, key, s.runCreate)
	return s.view(d), nil
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func (s *SiteBuilderService) checkActive(ctx context.Context, userID int64) error {
	n, err := s.repo.CountActive(ctx, userID)
	if err != nil {
		return err
	}
	if n >= siteBuilderMaxActivePerUser {
		return ErrSiteDraftBusy
	}
	return nil
}

func (s *SiteBuilderService) get(ctx context.Context, userID, id int64) (*SiteDraft, error) {
	d, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, ErrSiteDraftNotFound
	}
	return d, nil
}

// view is what the user sees: the undo stack stays on the server.
func (s *SiteBuilderService) view(d *SiteDraft) *SiteDraft {
	out := *d
	out.CanUndo = len(d.History) > 0 && !siteDraftActive(d.Status)
	out.History = nil
	return &out
}

func (s *SiteBuilderService) Get(ctx context.Context, userID, id int64) (*SiteDraft, error) {
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return s.view(d), nil
}

func (s *SiteBuilderService) List(ctx context.Context, userID int64) ([]SiteDraft, error) {
	return s.repo.List(ctx, userID, siteBuilderListLimit)
}

// Revise changes the page as asked.
func (s *SiteBuilderService) Revise(ctx context.Context, userID, id int64, instruction string) (*SiteDraft, error) {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" || utf8.RuneCountInString(instruction) > siteBuilderMaxInstruction {
		return nil, ErrSiteDraftInstruction
	}
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if d.HTML == "" || siteDraftActive(d.Status) {
		return nil, ErrSiteDraftState
	}
	key, err := s.key(ctx, userID, d.KeyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkActive(ctx, userID); err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(d.ID)
	if !ok {
		return nil, ErrSiteDraftState
	}
	d.Status, d.Error = SiteDraftGenerating, ""
	d.Turns = append(d.Turns, SiteDraftTurn{At: s.now(), Instruction: instruction, Status: "running"})
	if len(d.Turns) > siteBuilderMaxTurns {
		d.Turns = d.Turns[len(d.Turns)-siteBuilderMaxTurns:]
	}
	s.event(d, "info", "修改："+articleClip(instruction, 60))
	if err := s.repo.Save(ctx, d); err != nil {
		s.release(d.ID)
		return nil, err
	}
	s.start(runCtx, d, key, s.runRevise)
	return s.view(d), nil
}

// Retry writes the page again when the first one failed, or draws the pictures that failed.
func (s *SiteBuilderService) Retry(ctx context.Context, userID, id int64) (*SiteDraft, error) {
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if siteDraftActive(d.Status) {
		return nil, ErrSiteDraftState
	}
	failedImages := false
	for _, im := range d.Images {
		failedImages = failedImages || im.Status != "ok"
	}
	if d.HTML != "" && !failedImages {
		return nil, ErrSiteDraftState
	}
	key, err := s.key(ctx, userID, d.KeyID)
	if err != nil {
		return nil, err
	}
	if err := s.checkActive(ctx, userID); err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(d.ID)
	if !ok {
		return nil, ErrSiteDraftState
	}
	fn := s.runDraw
	d.Status, d.Error = SiteDraftDrawing, ""
	if d.HTML == "" {
		fn = s.runCreate
		d.Status = SiteDraftGenerating
		d.Turns = append(d.Turns, SiteDraftTurn{At: s.now(), Instruction: d.Brief.Description, Status: "running"})
		s.event(d, "info", "重新生成网页")
	}
	if err := s.repo.Save(ctx, d); err != nil {
		s.release(d.ID)
		return nil, err
	}
	s.start(runCtx, d, key, fn)
	return s.view(d), nil
}

// Undo goes back to the page before the last change.
func (s *SiteBuilderService) Undo(ctx context.Context, userID, id int64) (*SiteDraft, error) {
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if siteDraftActive(d.Status) || s.running(id) {
		return nil, ErrSiteDraftState
	}
	if len(d.History) == 0 {
		return nil, ErrSiteDraftNoUndo
	}
	d.HTML = d.History[len(d.History)-1]
	d.History = d.History[:len(d.History)-1]
	s.syncImages(d, d.HTML)
	s.event(d, "info", "已撤销上一次修改")
	if err := s.repo.Save(ctx, d); err != nil {
		return nil, err
	}
	return s.view(d), nil
}

func (s *SiteBuilderService) Cancel(ctx context.Context, userID, id int64) error {
	if _, err := s.get(ctx, userID, id); err != nil {
		return err
	}
	s.runsMu.Lock()
	cancel := s.runs[id]
	s.runsMu.Unlock()
	if cancel == nil {
		return ErrSiteDraftState
	}
	cancel()
	return nil
}

func (s *SiteBuilderService) Delete(ctx context.Context, userID, id int64) error {
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return err
	}
	if s.running(id) || siteDraftActive(d.Status) {
		return ErrSiteDraftState
	}
	if err := s.repo.Delete(ctx, userID, id); err != nil {
		return err
	}
	_ = os.RemoveAll(s.draftDir(id))
	return nil
}

// ImageFile is the path of one of the draft's pictures (after checking it is the user's).
func (s *SiteBuilderService) ImageFile(ctx context.Context, userID, id int64, n int) (string, error) {
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return "", err
	}
	for _, im := range d.Images {
		if im.N == n && im.Status == "ok" {
			return s.imagePath(id, im), nil
		}
	}
	return "", ErrSiteDraftNotFound
}

// SiteDraftPublishInput: SiteID > 0 publishes a new version of that site, otherwise a new site.
type SiteDraftPublishInput struct {
	SiteID int64  `json:"site_id"`
	Name   string `json:"name"`
	Title  string `json:"title"`
}

// Publish sends the page and its pictures to site hosting.
func (s *SiteBuilderService) Publish(ctx context.Context, userID, id int64, in SiteDraftPublishInput) (*Site, error) {
	if s.hosting == nil {
		return nil, ErrSiteDraftHosting
	}
	d, err := s.get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if siteDraftActive(d.Status) || s.running(id) {
		return nil, ErrSiteDraftState
	}
	if d.HTML == "" {
		return nil, ErrSiteDraftEmpty
	}
	data, err := s.pack(d)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = siteDraftPageTitle(d.HTML)
	}
	up := SiteUpload{Title: title, Name: strings.TrimSpace(in.Name), FileName: "site.zip", Data: data}
	var site *Site
	if in.SiteID > 0 {
		site, err = s.hosting.Update(ctx, userID, in.SiteID, up)
	} else {
		site, err = s.hosting.Create(ctx, userID, up)
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	d.SiteID, d.SiteName, d.SiteURL, d.PublishedAt = site.ID, site.Name, site.URL, &now
	s.event(d, "done", "已发布到 "+site.URL)
	if err := s.repo.Save(ctx, d); err != nil {
		slog.Warn("sitebuilder: save after publish", "id", d.ID, "err", err)
	}
	return site, nil
}

var siteDraftTitleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func siteDraftPageTitle(page string) string {
	if m := siteDraftTitleRe.FindStringSubmatch(page); m != nil {
		return articleClip(strings.TrimSpace(html.UnescapeString(m[1])), 60)
	}
	return ""
}

// pack builds the zip: index.html without the drawing hints, plus the pictures that were drawn
// (an <img> whose picture failed is left out).
func (s *SiteBuilderService) pack(d *SiteDraft) ([]byte, error) {
	ok := map[int]bool{}
	for _, im := range d.Images {
		ok[im.N] = im.Status == "ok"
	}
	page := siteDraftImgRe.ReplaceAllStringFunc(d.HTML, func(tag string) string {
		n, _, _ := siteDraftImgInfo(tag)
		if n == 0 {
			return tag
		}
		if !ok[n] {
			return ""
		}
		return siteDraftHintRe.ReplaceAllString(tag, "")
	})
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("index.html")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte(page)); err != nil {
		return nil, err
	}
	for _, im := range d.Images {
		if im.Status != "ok" {
			continue
		}
		raw, err := os.ReadFile(s.imagePath(d.ID, im))
		if err != nil {
			return nil, errLearnRunFailed(fmt.Sprintf("第 %d 张图片找不到了，请重新生成后再发布", im.N))
		}
		w, err := zw.Create("img/" + strconv.Itoa(im.N) + ".jpg")
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(raw); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- pictures in the page ----------------------------------------------------------------------

var (
	siteDraftImgRe    = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	siteDraftSrcRe    = regexp.MustCompile(`(?is)\bsrc\s*=\s*["']img/(\d+)\.jpg["']`)
	siteDraftPromptRe = regexp.MustCompile(`(?is)\bdata-ai-prompt\s*=\s*"([^"]*)"`)
	siteDraftSizeRe   = regexp.MustCompile(`(?is)\bdata-ai-size\s*=\s*"([^"]*)"`)
	siteDraftHintRe   = regexp.MustCompile(`(?is)\s+data-ai-(?:prompt|size)\s*=\s*"[^"]*"`)
)

// siteDraftImgInfo reads an <img> tag: its picture number (0: not one of ours), prompt and size.
func siteDraftImgInfo(tag string) (int, string, string) {
	m := siteDraftSrcRe.FindStringSubmatch(tag)
	if m == nil {
		return 0, "", ""
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 || n > siteBuilderMaxImages {
		return 0, "", ""
	}
	prompt := ""
	if p := siteDraftPromptRe.FindStringSubmatch(tag); p != nil {
		prompt = strings.TrimSpace(html.UnescapeString(p[1]))
	}
	size := "1536x1024"
	if z := siteDraftSizeRe.FindStringSubmatch(tag); z != nil {
		switch strings.ToLower(strings.TrimSpace(z[1])) {
		case "square":
			size = "1024x1024"
		case "portrait":
			size = "1024x1536"
		}
	}
	return n, articleClip(prompt, 800), size
}

// pageImages lists the pictures a page asks for (first use of each number wins) and drops <img>
// tags of numbers it may not use (over the limit or with no description).
func pageImages(page string, limit int) (string, []SiteDraftImage) {
	seen := map[int]bool{}
	var out []SiteDraftImage
	page = siteDraftImgRe.ReplaceAllStringFunc(page, func(tag string) string {
		if siteDraftSrcRe.FindStringSubmatch(tag) == nil {
			return tag
		}
		n, prompt, size := siteDraftImgInfo(tag)
		if n == 0 || n > limit {
			return ""
		}
		if seen[n] {
			return tag
		}
		if prompt == "" {
			return ""
		}
		seen[n] = true
		out = append(out, SiteDraftImage{N: n, Prompt: prompt, Size: size, Status: "pending"})
		return tag
	})
	sort.Slice(out, func(i, j int) bool { return out[i].N < out[j].N })
	return page, out
}

// syncImages makes d.Images match the page: pictures already drawn (now or before an undo) are
// ok, new or changed ones are pending.
func (s *SiteBuilderService) syncImages(d *SiteDraft, page string) {
	_, want := pageImages(page, siteBuilderMaxImages)
	old := map[int]SiteDraftImage{}
	for _, im := range d.Images {
		old[im.N] = im
	}
	for i, im := range want {
		if prev, ok := old[im.N]; ok && prev.Prompt == im.Prompt && prev.Size == im.Size && prev.Status == "ok" {
			want[i] = prev
		} else if st, err := os.Stat(s.imagePath(d.ID, im)); err == nil && st.Size() > 0 {
			want[i].Status, want[i].Bytes = "ok", int(st.Size())
		}
	}
	if want == nil {
		want = []SiteDraftImage{}
	}
	d.Images = want
}

// --- runs --------------------------------------------------------------------------------------

func (s *SiteBuilderService) draftDir(id int64) string {
	return filepath.Join(s.dir, strconv.FormatInt(id, 10))
}

// imagePath names a picture by its number and what it shows, so undoing a change finds the
// earlier picture still on disk.
func (s *SiteBuilderService) imagePath(id int64, im SiteDraftImage) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(im.Size + "\n" + im.Prompt))
	return filepath.Join(s.draftDir(id), fmt.Sprintf("%d-%08x.jpg", im.N, h.Sum32()))
}

func (s *SiteBuilderService) cleanup() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ids, err := s.repo.DeleteBefore(ctx, s.now().Add(-siteBuilderRetention))
		if err != nil {
			return
		}
		for _, id := range ids {
			_ = os.RemoveAll(s.draftDir(id))
		}
	}()
}

func (s *SiteBuilderService) running(id int64) bool {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	return s.runs[id] != nil
}

func (s *SiteBuilderService) event(d *SiteDraft, kind, text string) {
	d.Events = append(d.Events, ArticleEvent{At: s.now(), Kind: kind, Text: text})
	if len(d.Events) > 200 {
		d.Events = d.Events[len(d.Events)-200:]
	}
}

func (s *SiteBuilderService) save(d *SiteDraft) {
	s.savingMu.Lock()
	defer s.savingMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.repo.Save(ctx, d); err != nil {
		slog.Warn("sitebuilder: save", "id", d.ID, "err", err)
	}
}

func (s *SiteBuilderService) claim(id int64) (context.Context, bool) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if s.runs[id] != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), siteBuilderRunTimeout)
	s.runs[id] = cancel
	return ctx, true
}

func (s *SiteBuilderService) release(id int64) {
	s.runsMu.Lock()
	defer s.runsMu.Unlock()
	if cancel := s.runs[id]; cancel != nil {
		cancel()
		delete(s.runs, id)
	}
}

// start runs fn in the background on a claimed draft. However it ends, a page that already exists
// stays usable: the draft goes back to ready (with the error shown) and only a draft with no page
// at all is failed.
func (s *SiteBuilderService) start(ctx context.Context, d *SiteDraft, key *APIKey, fn func(ctx context.Context, d *SiteDraft, key string) error) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.release(d.ID)
		var err error
		queued := false
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
			err = fn(ctx, d, key.Key)
		case <-ctx.Done():
			err, queued = ErrSiteDraftSiteBusy, true
		}
		msg := ""
		switch {
		case err == nil:
		case errors.Is(ctx.Err(), context.Canceled):
			msg = "已停止"
		case queued:
			msg = ErrSiteDraftSiteBusy.Message
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			msg = "生成超时（超过 20 分钟），可以重试"
		default:
			msg = siteBuilderErrorText(d.ID, err)
		}
		d.DraftHTML = ""
		if n := len(d.Turns); n > 0 && d.Turns[n-1].Status == "running" {
			if msg == "" {
				d.Turns[n-1].Status = "ok"
			} else {
				d.Turns[n-1].Status, d.Turns[n-1].Error = "failed", msg
			}
		}
		switch {
		case msg == "":
			d.Status, d.Error = SiteDraftReady, ""
		case d.HTML != "":
			d.Status, d.Error = SiteDraftReady, msg
			s.event(d, "error", msg)
		default:
			d.Status, d.Error = SiteDraftFailed, msg
			s.event(d, "error", msg)
		}
		s.save(d)
	}()
}

func siteBuilderErrorText(id int64, err error) string {
	var app *infraerrors.ApplicationError
	if errors.As(err, &app) && app.Message != "" {
		return app.Message
	}
	slog.Warn("sitebuilder: run failed", "id", id, "err", err)
	return "生成出错了，可以重试"
}

func (s *SiteBuilderService) runCreate(ctx context.Context, d *SiteDraft, key string) error {
	msgs := []LearnMessage{
		{Role: "system", Content: siteBuilderCreatePrompt(d.Brief.Images)},
		{Role: "user", Content: siteBuilderCreateRequest(d.Brief)},
	}
	page, err := s.write(ctx, d, key, msgs, d.Brief.Images)
	if err != nil {
		return err
	}
	d.HTML = page
	s.syncImages(d, page)
	if t := siteDraftPageTitle(page); t != "" {
		d.Title = t
	}
	s.event(d, "step", fmt.Sprintf("网页写好了（%s）", formatSiteBytes(int64(len(page)))))
	return s.runDraw(ctx, d, key)
}

func (s *SiteBuilderService) runRevise(ctx context.Context, d *SiteDraft, key string) error {
	used := map[int]string{}
	for _, im := range d.Images {
		used[im.N] = im.Prompt
	}
	var allowed []int
	for n := 1; n <= siteBuilderMaxImages; n++ {
		if _, ok := used[n]; !ok {
			allowed = append(allowed, n)
		}
	}
	instruction := d.Turns[len(d.Turns)-1].Instruction
	msgs := []LearnMessage{
		{Role: "system", Content: siteBuilderRevisePrompt(allowed, used)},
		{Role: "user", Content: siteBuilderReviseRequest(d.HTML, instruction)},
	}
	page, err := s.write(ctx, d, key, msgs, siteBuilderMaxImages)
	if err != nil {
		return err
	}
	d.History = append(d.History, d.HTML)
	if len(d.History) > siteBuilderHistory {
		d.History = d.History[len(d.History)-siteBuilderHistory:]
	}
	d.HTML = page
	s.syncImages(d, page)
	if t := siteDraftPageTitle(page); t != "" {
		d.Title = t
	}
	s.event(d, "step", "改好了")
	return s.runDraw(ctx, d, key)
}

var siteDraftFenceRe = regexp.MustCompile("(?s)^```(?:html)?\\s*(.*?)\\s*```$")

// write streams the page from the model (saving the partial page for the live preview) and checks
// it is a whole HTML document.
func (s *SiteBuilderService) write(ctx context.Context, d *SiteDraft, key string, msgs []LearnMessage, imageLimit int) (string, error) {
	d.Status = SiteDraftGenerating
	d.DraftHTML = ""
	s.event(d, "step", "写网页…")
	s.save(d)
	var b strings.Builder
	last := s.now()
	res, err := s.chat(ctx, key, s.model(ctx), msgs, siteBuilderMaxTokens, func(delta string) error {
		_, _ = b.WriteString(delta)
		if b.Len() > siteBuilderMaxHTML+(16<<10) {
			return errLearnRunFailed("网页写得太长了，换个简单点的要求再试")
		}
		if s.now().Sub(last) >= siteBuilderSaveEvery {
			last = s.now()
			d.DraftHTML = b.String()
			s.save(d)
		}
		return nil
	})
	if res != nil {
		d.PromptTokens += res.PromptTokens
		d.CompletionTokens += res.CompletionTokens
		if res.Model != "" {
			d.Model = res.Model
		}
	}
	d.DraftHTML = ""
	if err != nil {
		return "", err
	}
	page, err := cleanSitePage(b.String())
	if err != nil {
		return "", err
	}
	page, _ = pageImages(page, imageLimit)
	return page, nil
}

// cleanSitePage cuts the HTML document out of the model's reply.
func cleanSitePage(text string) (string, error) {
	text = strings.TrimSpace(text)
	if m := siteDraftFenceRe.FindStringSubmatch(text); m != nil {
		text = m[1]
	}
	lower := strings.ToLower(text)
	start := strings.Index(lower, "<!doctype")
	if start < 0 {
		start = strings.Index(lower, "<html")
	}
	end := strings.LastIndex(lower, "</html>")
	if start < 0 || end < start {
		if start >= 0 {
			return "", errLearnRunFailed("网页没写完就断了（可能太长），可以重试，或把要求写简单一点")
		}
		return "", errLearnRunFailed("模型没有返回网页，可以重试")
	}
	page := text[start : end+len("</html>")]
	if len(page) > siteBuilderMaxHTML {
		return "", errLearnRunFailed("网页写得太长了，换个简单点的要求再试")
	}
	if !strings.Contains(strings.ToLower(page), "<body") {
		return "", errLearnRunFailed("模型返回的网页不完整，可以重试")
	}
	return page, nil
}

// runDraw draws the pictures the page still needs (a failed one does not fail the run).
func (s *SiteBuilderService) runDraw(ctx context.Context, d *SiteDraft, key string) error {
	todo := 0
	for _, im := range d.Images {
		if im.Status != "ok" {
			todo++
		}
	}
	if todo == 0 {
		return nil
	}
	d.Status = SiteDraftDrawing
	s.event(d, "step", fmt.Sprintf("画配图（%d 张）…", todo))
	s.save(d)
	if err := os.MkdirAll(s.draftDir(d.ID), 0o755); err != nil {
		return errLearnRunFailed("保存图片失败")
	}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(siteBuilderImageParallel)
	for i := range d.Images {
		if d.Images[i].Status == "ok" {
			continue
		}
		g.Go(func() error {
			mu.Lock()
			im := d.Images[i]
			mu.Unlock()
			data, err := s.drawOne(gctx, key, im)
			if err == nil {
				err = os.WriteFile(s.imagePath(d.ID, im), data, 0o644)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				msg := siteBuilderErrorText(d.ID, err)
				d.Images[i].Status, d.Images[i].Error = "failed", msg
				s.event(d, "error", fmt.Sprintf("第 %d 张图没画成：%s", im.N, msg))
			} else {
				d.Images[i].Status, d.Images[i].Error, d.Images[i].Bytes = "ok", "", len(data)
				d.ImagesDrawn++
				s.event(d, "step", fmt.Sprintf("第 %d 张图画好了", im.N))
			}
			s.save(d)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	failed := 0
	for _, im := range d.Images {
		if im.Status != "ok" {
			failed++
		}
	}
	if failed > 0 {
		s.event(d, "done", fmt.Sprintf("%d 张图没画成，可以点「补画」", failed))
	} else {
		s.event(d, "done", "配图都画好了")
	}
	return nil
}

func (s *SiteBuilderService) drawOne(ctx context.Context, key string, im SiteDraftImage) ([]byte, error) {
	raw, err := s.draw(ctx, key, im.Prompt+"\n这是网站上的配图。画面里不要出现任何文字、字母、Logo 和水印。", im.Size)
	if err != nil {
		return nil, err
	}
	return articleJPEG(raw)
}
