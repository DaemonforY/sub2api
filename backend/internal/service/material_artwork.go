package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// The 素材库 drawing queue works like the 课本诗画 one and draws on the same key: an admin queues 1–4
// drawings of a segment, one loop hands them to at most the configured concurrency, failed requests
// are retried on 429 / 5xx / network errors (30 s, then 90 s), and the day stops at the daily limit.
// Pictures are stored next to the 课本诗画 ones, under materials/artworks/<id / 1000>/.

const (
	MaterialArtworkQueued    = "queued"
	MaterialArtworkRunning   = "running"
	MaterialArtworkSucceeded = "succeeded"
	MaterialArtworkFailed    = "failed"
	MaterialArtworkCanceled  = "canceled"

	MaterialReviewPending  = "pending"
	MaterialReviewApproved = "approved"
	MaterialReviewRejected = "rejected"

	materialFileRoute       = "/api/v1/materials/files/"
	materialMaxDrawCount    = 4
	materialMaxArtworksPage = 200
	materialMaxRejectReason = 200
	materialMaxAttempts     = 3
	materialTick            = 5 * time.Second
)

var materialRelRe = regexp.MustCompile(`^artworks/[0-9]+/[0-9]+(-thumb|-[0-9]+x[0-9]+)?\.(png|jpg|webp)$`)

var (
	ErrMaterialArtworkNotFound = infraerrors.NotFound("MATERIAL_ARTWORK_NOT_FOUND", "找不到这张作品（Artwork not found）")
	ErrMaterialNoKey           = infraerrors.BadRequest("MATERIAL_NO_KEY", "没有可用的 API Key，请先在「课本诗画 → 设置」里选一把（No API key）")
)

// MaterialArtwork is one drawing (mat_artworks).
type MaterialArtwork struct {
	ID                 int64           `json:"id"`
	SegmentID          *int64          `json:"segment_id,omitempty"`
	StyleID            *int64          `json:"style_id,omitempty"`
	Options            json.RawMessage `json:"options"`
	PromptFinal        string          `json:"prompt_final"`
	PromptParts        json.RawMessage `json:"prompt_parts"`
	Model              string          `json:"model"`
	Size               string          `json:"size"`
	Quality            string          `json:"quality"`
	Status             string          `json:"status"`
	Attempts           int             `json:"attempts"`
	NotBefore          *time.Time      `json:"not_before,omitempty"`
	Error              string          `json:"error"`
	File               string          `json:"file"`
	Width              int             `json:"width"`
	Height             int             `json:"height"`
	CostUSD            float64         `json:"cost_usd"`
	DurationMs         int             `json:"duration_ms"`
	ParentID           *int64          `json:"parent_id,omitempty"`
	Source             string          `json:"source"`
	Rating             int             `json:"rating"`
	ReviewStatus       string          `json:"review_status"`
	RejectReason       string          `json:"reject_reason"`
	TextOverlayAllowed bool            `json:"text_overlay_allowed"`
	CreatedBy          *int64          `json:"created_by,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	StartedAt          *time.Time      `json:"started_at,omitempty"`
	FinishedAt         *time.Time      `json:"finished_at,omitempty"`
	ImageryIDs         []int64         `json:"imagery_ids"`

	// Joined for lists.
	StyleName    string `json:"style_name"`
	TextID       *int64 `json:"text_id,omitempty"`
	TextTitle    string `json:"text_title"`
	SegmentLabel string `json:"segment_label"`
	// Signed links, filled by the service.
	URL      string `json:"url,omitempty"`
	ThumbURL string `json:"thumb_url,omitempty"`
}

type MaterialArtworkQuery struct {
	SegmentID    int64
	TextID       int64
	Status       string
	ReviewStatus string
	Limit        int
	Offset       int
}

type MaterialQueueStats struct {
	Queued       int `json:"queued"`
	Running      int `json:"running"`
	StartedToday int `json:"started_today"`
}

// MaterialQueueView is the queue panel: the counts plus the limits and why drawing is paused.
type MaterialQueueView struct {
	MaterialQueueStats
	Concurrency int    `json:"concurrency"`
	DailyLimit  int    `json:"daily_limit"`
	Paused      string `json:"paused"`
}

// materialArtworkRepo is the queue part of MaterialRepository.
type materialArtworkRepo interface {
	// EnqueueArtworks inserts the drawings (with their imagery) and sets their IDs.
	EnqueueArtworks(ctx context.Context, arts []*MaterialArtwork) error
	// ClaimNextArtwork marks the oldest due queued drawing running; nil when none is due.
	ClaimNextArtwork(ctx context.Context) (*MaterialArtwork, error)
	FinishArtwork(ctx context.Context, a *MaterialArtwork) error
	// RequeueRunningArtworks puts drawings a previous process left running back in the queue.
	RequeueRunningArtworks(ctx context.Context) (int64, error)
	ArtworkQueueStats(ctx context.Context, dayStart time.Time) (MaterialQueueStats, error)
	ListArtworks(ctx context.Context, q MaterialArtworkQuery) ([]MaterialArtwork, int, error)
	// GetArtwork returns nil when the drawing does not exist.
	GetArtwork(ctx context.Context, id int64) (*MaterialArtwork, error)
	// ReviewArtwork returns false when the drawing does not exist.
	ReviewArtwork(ctx context.Context, id int64, status string, rating int, reason string) (bool, error)
	// CancelQueuedArtworks cancels the queued ones among ids.
	CancelQueuedArtworks(ctx context.Context, ids []int64) (int64, error)
	// RetryArtwork puts a failed or canceled drawing back in the queue; false when it is neither.
	RetryArtwork(ctx context.Context, id int64) (bool, error)
}

// materialDrawer draws pictures; ShihuaService is the real one.
type materialDrawer interface {
	AdminDraw(ctx context.Context, prompt, size, quality string) (*shihuaDrawn, error)
	DrawLimits(ctx context.Context) (concurrency, dailyLimit int, ready bool)
	ImageCost(w, h int) float64
}

// materialQueue is the drawing loop's state.
type materialQueue struct {
	drawer  materialDrawer
	files   *shihuaFiles
	mu      sync.Mutex
	running int
	wake    chan struct{}
	stop    chan struct{}
	wg      sync.WaitGroup
	started bool
}

func materialArtworkRel(id int64, ext string) string {
	return fmt.Sprintf("artworks/%d/%d.%s", id/1000, id, ext)
}

func materialArtworkThumbRel(id int64) string {
	return fmt.Sprintf("artworks/%d/%d-thumb.jpg", id/1000, id)
}

// ProvideMaterialService wires the 素材库: text and pictures go through the 课本诗画 key, pictures
// are stored next to the 课本诗画 ones, and the drawing queue starts right away.
func ProvideMaterialService(repo MaterialRepository, shihuaRepo ShihuaRepository, shihuaSvc *ShihuaService) *MaterialService {
	s := NewMaterialService(repo, shihuaRepo, shihuaSvc)
	if shihuaSvc != nil {
		s.EnableDrawing(shihuaSvc, filepath.Join(filepath.Dir(filepath.Clean(shihuaSvc.DataDir())), "materials"))
		s.Start()
	}
	return s
}

// EnableDrawing turns the queue on with a drawer and a picture directory (tests use fakes).
func (s *MaterialService) EnableDrawing(drawer materialDrawer, dir string) {
	s.queue = &materialQueue{drawer: drawer, files: newSignedFiles(dir, materialFileRoute, materialRelRe),
		wake: make(chan struct{}, 1), stop: make(chan struct{})}
}

// Start puts drawings a previous process left running back in the queue and starts the loop.
func (s *MaterialService) Start() {
	q := s.queue
	if q == nil || q.started {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if n, err := s.repo.RequeueRunningArtworks(ctx); err != nil {
		slog.Warn("materials: requeue interrupted drawings", "err", err)
	} else if n > 0 {
		slog.Info("materials: requeued interrupted drawings", "count", n)
	}
	q.started = true
	q.wg.Add(1)
	go s.loop()
}

// Stop ends the loop and waits for running drawings (tests).
func (s *MaterialService) Stop() {
	q := s.queue
	if q == nil || !q.started {
		return
	}
	close(q.stop)
	q.wg.Wait()
	q.started = false
}

func (s *MaterialService) kick() {
	if s.queue == nil {
		return
	}
	select {
	case s.queue.wake <- struct{}{}:
	default:
	}
}

func (q *materialQueue) runningCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}

func (s *MaterialService) loop() {
	q := s.queue
	defer q.wg.Done()
	ticker := time.NewTicker(materialTick)
	defer ticker.Stop()
	for {
		select {
		case <-q.stop:
			return
		case <-q.wake:
		case <-ticker.C:
		}
		s.dispatch()
	}
}

// dispatch starts queued drawings up to the concurrency, unless the day's limit is reached or no
// key is configured.
func (s *MaterialService) dispatch() {
	q := s.queue
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	concurrency, limit, ready := q.drawer.DrawLimits(ctx)
	if !ready {
		return
	}
	for q.runningCount() < concurrency {
		stats, err := s.repo.ArtworkQueueStats(ctx, shihuaDayStart(s.now()))
		if err != nil || stats.Queued == 0 || stats.StartedToday >= limit {
			return
		}
		art, err := s.repo.ClaimNextArtwork(ctx)
		if err != nil {
			slog.Warn("materials: claim", "err", err)
			return
		}
		if art == nil {
			return
		}
		q.mu.Lock()
		q.running++
		q.mu.Unlock()
		q.wg.Add(1)
		go s.runDraw(art)
	}
}

func (s *MaterialService) runDraw(art *MaterialArtwork) {
	q := s.queue
	defer q.wg.Done()
	defer func() {
		q.mu.Lock()
		q.running--
		q.mu.Unlock()
		s.kick()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), shihuaDrawTimeout)
	defer cancel()
	started := s.now()
	out, err := q.drawer.AdminDraw(ctx, art.PromptFinal, art.Size, art.Quality)
	art.DurationMs = int(s.now().Sub(started) / time.Millisecond)
	now := s.now()
	if err == nil {
		file, w, h, saveErr := q.files.saveWithThumb(materialArtworkRel(art.ID, shihuaExt(out.Data)), materialArtworkThumbRel(art.ID), out.Data)
		if saveErr != nil {
			err = &shihuaDrawError{Status: -1, Msg: "保存图片失败：" + saveErr.Error()}
		} else {
			art.Status, art.File, art.Width, art.Height, art.Error = MaterialArtworkSucceeded, file, w, h, ""
			art.CostUSD = q.drawer.ImageCost(w, h)
			art.FinishedAt, art.NotBefore = &now, nil
		}
	}
	if err != nil {
		var de *shihuaDrawError
		if !errors.As(err, &de) {
			de = &shihuaDrawError{Msg: err.Error()}
		}
		art.Error = de.Msg
		if de.retryable() && art.Attempts < materialMaxAttempts {
			next := now.Add(shihuaBackoff(art.Attempts))
			art.Status, art.NotBefore = MaterialArtworkQueued, &next
		} else {
			art.Status, art.FinishedAt = MaterialArtworkFailed, &now
		}
	}
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelWrite()
	if err := s.repo.FinishArtwork(writeCtx, art); err != nil {
		slog.Warn("materials: finish drawing", "id", art.ID, "err", err)
	}
}

// ---- service ----

func (s *MaterialService) drawingOrErr(ctx context.Context) (*materialQueue, error) {
	if s.queue == nil {
		return nil, ErrMaterialNoKey
	}
	if _, _, ready := s.queue.drawer.DrawLimits(ctx); !ready {
		return nil, ErrMaterialNoKey
	}
	return s.queue, nil
}

func (s *MaterialService) signArtwork(a *MaterialArtwork) {
	if s.queue == nil || a.File == "" {
		return
	}
	a.URL = s.queue.files.sign(a.File)
	a.ThumbURL = s.queue.files.sign(materialArtworkThumbRel(a.ID))
}

// materialBlockedError lists what a blocked prompt hit.
func materialBlockedError(hits []MaterialBlockHit) error {
	var terms []string
	for _, h := range hits {
		if h.Action == MaterialBlockActionBlock {
			terms = append(terms, h.Match)
		}
	}
	return infraerrors.BadRequest("MATERIAL_PROMPT_BLOCKED",
		"提示词里有禁止出现的内容："+strings.Join(terms, "、")+"，请修改后再画（Prompt hits the blocklist）")
}

// DrawSegment queues count drawings of a segment with the admin's picks.
func (s *MaterialService) DrawSegment(ctx context.Context, adminID, segmentID int64, in MaterialPromptInput, count int) ([]MaterialArtwork, error) {
	if count <= 0 {
		count = 1
	}
	if count > materialMaxDrawCount {
		return nil, ErrMaterialInvalid
	}
	if _, err := s.drawingOrErr(ctx); err != nil {
		return nil, err
	}
	p, err := s.BuildPrompt(ctx, segmentID, in)
	if err != nil {
		return nil, err
	}
	if p.Blocked {
		return nil, materialBlockedError(p.Hits)
	}
	if err := s.lyrics.Refuse(ctx, p.Prompt); err != nil {
		return nil, err
	}
	options, _ := json.Marshal(in)
	parts, _ := json.Marshal(p.Parts)
	seg, style, by := segmentID, p.StyleID, adminID
	arts := make([]*MaterialArtwork, count)
	for i := range arts {
		arts[i] = &MaterialArtwork{SegmentID: &seg, StyleID: &style, Options: options, PromptFinal: p.Prompt, PromptParts: parts,
			Model: shihuaImageModel, Size: p.Size, Quality: p.Quality, Status: MaterialArtworkQueued, Source: "library",
			ReviewStatus: MaterialReviewPending, TextOverlayAllowed: p.TextOverlayAllowed, CreatedBy: &by, ImageryIDs: p.ImageryIDs}
	}
	if err := s.repo.EnqueueArtworks(ctx, arts); err != nil {
		return nil, err
	}
	s.kick()
	out := make([]MaterialArtwork, len(arts))
	for i, a := range arts {
		out[i] = *a
	}
	return out, nil
}

// Redraw queues one more drawing with the same prompt. The prompt is checked against today's
// blocklist again, since terms may have been added since.
func (s *MaterialService) Redraw(ctx context.Context, adminID, artworkID int64) (*MaterialArtwork, error) {
	if _, err := s.drawingOrErr(ctx); err != nil {
		return nil, err
	}
	src, err := s.repo.GetArtwork(ctx, artworkID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, ErrMaterialArtworkNotFound
	}
	terms, err := s.repo.ListBlocklist(ctx, true)
	if err != nil {
		return nil, err
	}
	check := CheckMaterialBlocklist(strings.TrimSuffix(strings.TrimSpace(src.PromptFinal), materialNoText), "prompt", terms)
	if check.Blocked {
		return nil, materialBlockedError(check.Hits)
	}
	if err := s.lyrics.Refuse(ctx, src.PromptFinal); err != nil {
		return nil, err
	}
	parent, by := src.ID, adminID
	a := &MaterialArtwork{SegmentID: src.SegmentID, StyleID: src.StyleID, Options: src.Options, PromptFinal: src.PromptFinal,
		PromptParts: src.PromptParts, Model: src.Model, Size: src.Size, Quality: src.Quality, Status: MaterialArtworkQueued,
		ParentID: &parent, Source: src.Source, ReviewStatus: MaterialReviewPending, TextOverlayAllowed: src.TextOverlayAllowed,
		CreatedBy: &by, ImageryIDs: src.ImageryIDs}
	if err := s.repo.EnqueueArtworks(ctx, []*MaterialArtwork{a}); err != nil {
		return nil, err
	}
	s.kick()
	return a, nil
}

func (s *MaterialService) ListArtworks(ctx context.Context, q MaterialArtworkQuery) ([]MaterialArtwork, int, error) {
	if q.Limit <= 0 || q.Limit > materialMaxArtworksPage {
		q.Limit = 60
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	items, total, err := s.repo.ListArtworks(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		s.signArtwork(&items[i])
	}
	return items, total, nil
}

func (s *MaterialService) GetArtwork(ctx context.Context, id int64) (*MaterialArtwork, error) {
	a, err := s.repo.GetArtwork(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrMaterialArtworkNotFound
	}
	s.signArtwork(a)
	return a, nil
}

// MaterialReviewInput is an admin's verdict on a drawing.
type MaterialReviewInput struct {
	ReviewStatus string `json:"review_status"`
	Rating       int    `json:"rating"` // 0 = not rated, 1–5
	RejectReason string `json:"reject_reason"`
}

// ReviewArtwork records the verdict; only finished drawings can be approved.
func (s *MaterialService) ReviewArtwork(ctx context.Context, id int64, in MaterialReviewInput) (*MaterialArtwork, error) {
	switch in.ReviewStatus {
	case MaterialReviewPending, MaterialReviewApproved, MaterialReviewRejected:
	default:
		return nil, ErrMaterialInvalid
	}
	if in.Rating < 0 || in.Rating > 5 {
		return nil, ErrMaterialInvalid
	}
	reason := strings.TrimSpace(in.RejectReason)
	if utf8.RuneCountInString(reason) > materialMaxRejectReason {
		reason = materialTrimRunes(reason, materialMaxRejectReason)
	}
	if in.ReviewStatus != MaterialReviewRejected {
		reason = ""
	}
	a, err := s.repo.GetArtwork(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrMaterialArtworkNotFound
	}
	if in.ReviewStatus == MaterialReviewApproved && a.Status != MaterialArtworkSucceeded {
		return nil, infraerrors.BadRequest("MATERIAL_NOT_DRAWN", "这张还没画好，不能通过审核（Not drawn yet）")
	}
	if _, err := s.repo.ReviewArtwork(ctx, id, in.ReviewStatus, in.Rating, reason); err != nil {
		return nil, err
	}
	return s.GetArtwork(ctx, id)
}

// CancelArtworks cancels the queued drawings among ids.
func (s *MaterialService) CancelArtworks(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 || len(ids) > materialMaxArtworksPage {
		return 0, ErrMaterialInvalid
	}
	return s.repo.CancelQueuedArtworks(ctx, ids)
}

// RetryArtwork puts a failed or canceled drawing back in the queue.
func (s *MaterialService) RetryArtwork(ctx context.Context, id int64) (*MaterialArtwork, error) {
	if _, err := s.drawingOrErr(ctx); err != nil {
		return nil, err
	}
	ok, err := s.repo.RetryArtwork(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, infraerrors.BadRequest("MATERIAL_NOT_RETRYABLE", "只有失败或已取消的作品可以重试（Only failed or canceled drawings can be retried）")
	}
	s.kick()
	return s.GetArtwork(ctx, id)
}

// QueueView reports the queue for the admin page.
func (s *MaterialService) QueueView(ctx context.Context) (*MaterialQueueView, error) {
	stats, err := s.repo.ArtworkQueueStats(ctx, shihuaDayStart(s.now()))
	if err != nil {
		return nil, err
	}
	v := &MaterialQueueView{MaterialQueueStats: stats}
	if s.queue == nil {
		v.Paused = "出图未启用（Drawing is off）"
		return v, nil
	}
	v.Running = max(v.Running, s.queue.runningCount())
	concurrency, limit, ready := s.queue.drawer.DrawLimits(ctx)
	v.Concurrency, v.DailyLimit = concurrency, limit
	switch {
	case !ready:
		v.Paused = "没有可用的 API Key，请在「课本诗画 → 设置」里选一把（No API key）"
	case stats.StartedToday >= limit:
		v.Paused = fmt.Sprintf("今天已画 %d 张，达到每日上限，明天 0 点（北京时间）继续（Daily limit reached）", stats.StartedToday)
	}
	return v, nil
}

// OpenFile checks a signed picture link (for the file route).
func (s *MaterialService) OpenFile(rel, exp, sig string) (string, bool) {
	if s.queue == nil {
		return "", false
	}
	return s.queue.files.Open(rel, exp, sig)
}
