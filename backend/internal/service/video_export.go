package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// MP4 export for the publishing kit: a film is rendered frame by frame at 1080p 60 fps by the
// videorender container (deploy/videorender), which opens the video site's /render.html, mixes the
// narration and writes the MP4 and a cover next to the project's audio. Landscape is the film as on
// the site (1920×1080); portrait (1080×1920, for 抖音 / 视频号 / 小红书) puts the title above the
// film and large subtitles below it. One render runs at a time, site-wide; the rest wait in order.

const (
	VideoLayoutLandscape = "landscape"
	VideoLayoutPortrait  = "portrait"

	VideoExportQueued  = "queued"
	VideoExportRunning = "running"
	VideoExportReady   = "ready"
	VideoExportFailed  = "failed"

	videoExportFPS       = 60
	videoExportsPerDay   = 40
	videoExportTimeout   = 3 * time.Hour
	videoExportQueueSize = 200
	// Bump when render.html / the render settings change, so films are rendered again.
	videoRenderVersion = "1"
)

var (
	ErrVideoExportOff    = infraerrors.BadRequest("VIDEO_EXPORT_OFF", "视频导出暂未开通，请稍后再试（MP4 export is not available）")
	ErrVideoExportLayout = infraerrors.BadRequest("VIDEO_EXPORT_LAYOUT", "导出格式只能是横屏或竖屏（Unknown layout）")
	ErrVideoExportLimit  = infraerrors.TooManyRequests("VIDEO_EXPORT_LIMIT", fmt.Sprintf("今天的导出次数已用完（每天 %d 次），请明天再试（Daily export limit reached）", videoExportsPerDay))
	ErrVideoExportQueue  = infraerrors.TooManyRequests("VIDEO_EXPORT_QUEUE", "导出排队的作品太多了，请过一会儿再试（The export queue is full）")
	ErrVideoExportFilm   = infraerrors.BadRequest("VIDEO_EXPORT_FILM", "只有生成完成的 HTML 视频可以导出 MP4（Only finished films can be exported）")
)

// VideoExport is one render of a project in one layout.
type VideoExport struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    int64     `json:"-"`
	Layout    string    `json:"layout"`
	Status    string    `json:"status"`
	Signature string    `json:"-"`
	File      string    `json:"-"`
	Cover     string    `json:"-"`
	Size      int64     `json:"size"`
	Duration  float64   `json:"duration"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Filled in for the API.
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	FPS      int     `json:"fps"`
	Progress float64 `json:"progress"`        // 0–1 while rendering
	Position int     `json:"position"`        // place in the queue, 1 = next
	Current  bool    `json:"current"`         // rendered from the film as it is now
	URL      string  `json:"url,omitempty"`   // signed download link (attachment)
	CoverURL string  `json:"cover,omitempty"` // signed cover image link
}

type VideoExportRepository interface {
	CreateExport(ctx context.Context, e *VideoExport) error
	UpdateExport(ctx context.Context, e *VideoExport) error
	GetExport(ctx context.Context, id string) (*VideoExport, error)
	ListExports(ctx context.Context, projectID string) ([]VideoExport, error)
	DeleteExport(ctx context.Context, id string) error
	UnfinishedExports(ctx context.Context) ([]string, error)
	CountExportsSince(ctx context.Context, userID int64, since time.Time) (int, error)
	GetPublishCopy(ctx context.Context, projectID string) (json.RawMessage, error)
	SetPublishCopy(ctx context.Context, projectID string, raw json.RawMessage) error
}

// videoRenderJob is what the videorender container takes (see deploy/videorender/server.mjs).
type videoRenderJob struct {
	URL      string            `json:"url"`
	FPS      int               `json:"fps"`
	Width    int               `json:"width"`
	Height   int               `json:"height"`
	Setup    map[string]any    `json:"setup"`
	Audio    []videoRenderClip `json:"audio"`
	Out      string            `json:"out"`
	Cover    string            `json:"cover"`
	CoverAt  float64           `json:"coverAt"`
	Progress string            `json:"progress"`
}

type videoRenderClip struct {
	File  string  `json:"file"`
	Start float64 `json:"start"`
}

type videoRenderResult struct {
	Size     int64    `json:"size"`
	Duration float64  `json:"duration"`
	Frames   int      `json:"frames"`
	Seconds  int      `json:"seconds"`
	Errors   []string `json:"errors"`
	Error    string   `json:"error"`
}

// videoRenderer posts jobs to the render container.
type videoRenderer struct {
	url    string // e.g. http://videorender:7100
	origin string // the video site, whose /render.html the renderer opens
	client *http.Client
	queue  chan string

	mu      sync.Mutex
	pending []string // queued export ids in order, for the queue position
}

func (r *videoRenderer) enqueue(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case r.queue <- id:
		r.pending = append(r.pending, id)
		return true
	default:
		return false
	}
}

func (r *videoRenderer) taken(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.pending {
		if p == id {
			r.pending = append(r.pending[:i], r.pending[i+1:]...)
			return
		}
	}
}

// position is 1 for the next export to render, 0 when the id is not waiting.
func (r *videoRenderer) position(id string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.pending {
		if p == id {
			return i + 1
		}
	}
	return 0
}

// SetRenderer enables MP4 export. renderURL is the videorender container, siteOrigin the public
// video site (https://video.hivegpt.cn). Interrupted exports are queued again.
func (s *VideoService) SetRenderer(exports VideoExportRepository, renderURL, siteOrigin string) {
	renderURL, siteOrigin = strings.TrimRight(strings.TrimSpace(renderURL), "/"), strings.TrimRight(strings.TrimSpace(siteOrigin), "/")
	if exports == nil || renderURL == "" || siteOrigin == "" {
		return
	}
	s.exports = exports
	s.renderer = &videoRenderer{url: renderURL, origin: siteOrigin, client: &http.Client{Timeout: videoExportTimeout}, queue: make(chan string, videoExportQueueSize)}
	go s.exportWorker()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ids, err := exports.UnfinishedExports(ctx)
	if err != nil {
		slog.Warn("video: listing unfinished exports", "err", err)
	}
	for _, id := range ids {
		s.renderer.enqueue(id)
	}
}

// ExportEnabled reports whether MP4 export is configured.
func (s *VideoService) ExportEnabled() bool { return s.renderer != nil }

func videoLayoutSize(layout string) (int, int, bool) {
	switch layout {
	case VideoLayoutLandscape:
		return 1920, 1080, true
	case VideoLayoutPortrait:
		return 1080, 1920, true
	}
	return 0, 0, false
}

// videoExportSignature identifies what a render shows: the scenes (code, timing, audio), the title
// and the layout.
func videoExportSignature(p *VideoProject, layout string) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%dx%d\n", videoRenderVersion, layout, p.Title, p.Width, p.Height)
	if p.Spec != nil {
		theme, _ := json.Marshal(p.Spec.Theme)
		_, _ = h.Write(theme)
		_, _ = fmt.Fprintf(h, "|%.3f\n", p.Spec.Poster)
		for _, sc := range p.Spec.Scenes {
			audio := ""
			if sc.Audio != nil {
				audio = sc.Audio.File
			}
			_, _ = fmt.Fprintf(h, "%s|%.3f|%s|%s|", sc.ID, sc.Duration, audio, sc.Narration)
			_, _ = io.WriteString(h, sc.Code)
			_, _ = io.WriteString(h, "\n")
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func (s *VideoService) exportable(ctx context.Context, userID int64, projectID string) (*VideoProject, error) {
	if s.renderer == nil {
		return nil, ErrVideoExportOff
	}
	p, err := s.own(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if p.Mode == VideoModeUpload || p.Spec == nil || len(p.Spec.Scenes) == 0 {
		return nil, ErrVideoExportFilm
	}
	return p, nil
}

// StartExport queues a render of the project in a layout, or returns the export that already
// covers the film as it is now.
func (s *VideoService) StartExport(ctx context.Context, userID int64, projectID, layout string) (*VideoExport, error) {
	if _, _, ok := videoLayoutSize(layout); !ok {
		return nil, ErrVideoExportLayout
	}
	p, err := s.exportable(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if p.Status != VideoStatusReady || s.running(p.ID) {
		return nil, ErrVideoExportFilm
	}
	sig := videoExportSignature(p, layout)
	list, err := s.exports.ListExports(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		e := &list[i]
		if e.Layout == layout && e.Signature == sig && e.Status != VideoExportFailed {
			return s.describeExport(p, e), nil
		}
	}
	n, err := s.exports.CountExportsSince(ctx, userID, timezone.StartOfDay(s.now()))
	if err != nil {
		return nil, err
	}
	if n >= videoExportsPerDay {
		return nil, ErrVideoExportLimit
	}
	e := &VideoExport{ID: randomVideoID(12), ProjectID: p.ID, UserID: userID, Layout: layout, Status: VideoExportQueued, Signature: sig}
	if err := s.exports.CreateExport(ctx, e); err != nil {
		return nil, err
	}
	if !s.renderer.enqueue(e.ID) {
		e.Status, e.Error = VideoExportFailed, "导出排队已满（Queue full）"
		_ = s.exports.UpdateExport(ctx, e)
		return nil, ErrVideoExportQueue
	}
	return s.describeExport(p, e), nil
}

// Exports lists a project's exports, newest first, with progress and download links.
func (s *VideoService) Exports(ctx context.Context, userID int64, projectID string) ([]*VideoExport, error) {
	p, err := s.exportable(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	list, err := s.exports.ListExports(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]*VideoExport, 0, len(list))
	for i := range list {
		out = append(out, s.describeExport(p, &list[i]))
	}
	return out, nil
}

func (s *VideoService) exportPaths(projectID, exportID, layout string) (mp4, cover, progress string) {
	base := filepath.Join(filepath.Base(projectID), "exports", exportID+"-"+layout)
	return base + ".mp4", base + ".jpg", base + ".json"
}

func (s *VideoService) describeExport(p *VideoProject, e *VideoExport) *VideoExport {
	out := *e
	out.Width, out.Height, _ = videoLayoutSize(e.Layout)
	out.FPS = videoExportFPS
	out.Current = e.Signature == videoExportSignature(p, e.Layout)
	switch e.Status {
	case VideoExportRunning:
		_, _, progress := s.exportPaths(e.ProjectID, e.ID, e.Layout)
		if raw, err := os.ReadFile(filepath.Join(s.audioDir, progress)); err == nil {
			var pr struct {
				Frame  int `json:"frame"`
				Frames int `json:"frames"`
			}
			if json.Unmarshal(raw, &pr) == nil && pr.Frames > 0 {
				out.Progress = math.Min(0.99, float64(pr.Frame)/float64(pr.Frames))
			}
		}
	case VideoExportQueued:
		out.Position = s.exportPosition(e.ID)
	case VideoExportReady:
		out.Progress = 1
		out.URL = s.exportURL(e.ID, "mp4")
		if e.Cover != "" {
			out.CoverURL = s.exportURL(e.ID, "jpg")
		}
	}
	return &out
}

func (s *VideoService) exportPosition(id string) int {
	if s.renderer == nil {
		return 0
	}
	return s.renderer.position(id)
}

func (s *VideoService) exportURL(exportID, kind string) string {
	exp := s.now().Truncate(time.Hour).Add(videoMediaURLTTL).Unix()
	return "/api/v1/video/exports/" + exportID + "/" + kind + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.mediaSig("export:"+exportID, kind, exp)
}

// VideoExportFile is a finished export's file for download.
type VideoExportFile struct {
	Path string
	Name string // download file name
	Mime string
}

// ExportFile checks a signed export link and returns the file it points to.
func (s *VideoService) ExportFile(ctx context.Context, exportID, kind, exp, sig string) (*VideoExportFile, error) {
	if s.exports == nil || (kind != "mp4" && kind != "jpg") {
		return nil, ErrVideoNotFound
	}
	e64, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || e64 < s.now().Unix() || !hmac.Equal([]byte(sig), []byte(s.mediaSig("export:"+exportID, kind, e64))) {
		return nil, ErrVideoNotFound
	}
	e, err := s.exports.GetExport(ctx, exportID)
	if err != nil {
		return nil, err
	}
	if e == nil || e.Status != VideoExportReady {
		return nil, ErrVideoNotFound
	}
	p, err := s.repo.Get(ctx, e.ProjectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrVideoNotFound
	}
	label := map[string]string{VideoLayoutLandscape: "横屏", VideoLayoutPortrait: "竖屏"}[e.Layout]
	name := videoSafeFileName(p.Title)
	out := &VideoExportFile{}
	if kind == "mp4" {
		out.Path, out.Name, out.Mime = filepath.Join(s.audioDir, e.File), fmt.Sprintf("%s-%s-1080p60.mp4", name, label), "video/mp4"
	} else {
		out.Path, out.Name, out.Mime = filepath.Join(s.audioDir, e.Cover), fmt.Sprintf("%s-%s-封面.jpg", name, label), "image/jpeg"
	}
	if _, err := os.Stat(out.Path); err != nil {
		return nil, ErrVideoNotFound
	}
	return out, nil
}

// videoSafeFileName keeps a title usable as a file name on every OS.
func videoSafeFileName(title string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "：", "*", "", "?", "", "\"", "", "<", "", ">", "", "|", "-", "\n", " ", "\r", " ")
	name := strings.TrimSpace(r.Replace(title))
	if name == "" {
		name = "HiveGPT视频"
	}
	return clipRunes(name, 60)
}

// --- worker -----------------------------------------------------------------------------------

func (s *VideoService) exportWorker() {
	for id := range s.renderer.queue {
		s.renderer.taken(id)
		s.renderExport(id)
	}
}

func (s *VideoService) renderExport(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), videoExportTimeout)
	defer cancel()
	e, err := s.exports.GetExport(ctx, id)
	if err != nil || e == nil || (e.Status != VideoExportQueued && e.Status != VideoExportRunning) {
		return
	}
	p, err := s.repo.Get(ctx, e.ProjectID)
	if err != nil || p == nil || p.Spec == nil {
		e.Status, e.Error = VideoExportFailed, "作品不存在或已删除（Project not found）"
		_ = s.exports.UpdateExport(ctx, e)
		return
	}
	e.Status, e.Error, e.Signature = VideoExportRunning, "", videoExportSignature(p, e.Layout)
	_ = s.exports.UpdateExport(ctx, e)
	job := s.renderJob(p, e)
	res, err := s.callRenderer(ctx, job)
	if err != nil {
		slog.Warn("video: export failed", "export", e.ID, "project", p.ID, "err", err)
		e.Status, e.Error = VideoExportFailed, "导出失败，请稍后点「重新导出」（Render failed: "+clipRunes(err.Error(), 200)+"）"
		_ = s.exports.UpdateExport(context.WithoutCancel(ctx), e)
		return
	}
	if len(res.Errors) > 0 {
		slog.Warn("video: export finished with scene errors", "export", e.ID, "errors", res.Errors)
	}
	e.Status, e.File, e.Cover, e.Size, e.Duration = VideoExportReady, job.Out, job.Cover, res.Size, res.Duration
	if err := s.exports.UpdateExport(ctx, e); err != nil {
		slog.Warn("video: saving export", "export", e.ID, "err", err)
		return
	}
	_ = os.Remove(filepath.Join(s.audioDir, job.Progress))
	s.pruneExports(ctx, p.ID, e)
}

// pruneExports drops older renders of the same layout once a new one is ready.
func (s *VideoService) pruneExports(ctx context.Context, projectID string, keep *VideoExport) {
	list, err := s.exports.ListExports(ctx, projectID)
	if err != nil {
		return
	}
	for _, old := range list {
		if old.ID == keep.ID || old.Layout != keep.Layout || old.Status == VideoExportQueued || old.Status == VideoExportRunning {
			continue
		}
		for _, f := range []string{old.File, old.Cover} {
			if f != "" {
				_ = os.Remove(filepath.Join(s.audioDir, f))
			}
		}
		_ = s.exports.DeleteExport(ctx, old.ID)
	}
}

func (s *VideoService) renderJob(p *VideoProject, e *VideoExport) videoRenderJob {
	w, h, _ := videoLayoutSize(e.Layout)
	mp4, cover, progress := s.exportPaths(p.ID, e.ID, e.Layout)
	scenes := make([]map[string]any, 0, len(p.Spec.Scenes))
	sceneData := make([]map[string]any, 0, len(p.Spec.Scenes))
	var audio []videoRenderClip
	at := 0.0
	for _, sc := range p.Spec.Scenes {
		var words []VideoWord
		if sc.Audio != nil {
			words = sc.Audio.Words
			audio = append(audio, videoRenderClip{File: filepath.Join(filepath.Base(p.ID), filepath.Base(sc.Audio.File)), Start: at})
		}
		scenes = append(scenes, map[string]any{"id": sc.ID, "duration": sc.Duration, "code": sc.Code, "words": words})
		sceneData = append(sceneData, map[string]any{"narration": sc.Narration, "duration": sc.Duration, "audio": sc.Audio})
		at += sc.Duration
	}
	coverAt := at * 0.3
	if p.Spec.Poster > 0 {
		coverAt = math.Min(p.Spec.Poster, at)
	}
	return videoRenderJob{
		URL: s.renderer.origin + "/render.html", FPS: videoExportFPS, Width: w, Height: h,
		Setup: map[string]any{
			"layout": e.Layout, "width": w, "height": h, "title": p.Title, "brand": "HiveGPT 视频",
			"payload": map[string]any{"width": p.Spec.Width, "height": p.Spec.Height, "theme": p.Spec.Theme, "loop": false, "poster": p.Spec.Poster, "scenes": scenes},
			"scenes":  sceneData,
		},
		Audio: audio, Out: mp4, Cover: cover, CoverAt: coverAt, Progress: progress,
	}
}

func (s *VideoService) callRenderer(ctx context.Context, job videoRenderJob) (*videoRenderResult, error) {
	body, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.renderer.url+"/render", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.renderer.client.Do(req)
		if err != nil {
			return nil, err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		// Busy only if another server process shares the renderer; wait and try again.
		if resp.StatusCode == http.StatusConflict && attempt < 60 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(30 * time.Second):
			}
			continue
		}
		var res videoRenderResult
		if err := json.Unmarshal(raw, &res); err != nil {
			return nil, fmt.Errorf("renderer HTTP %d: %s", resp.StatusCode, clipRunes(string(raw), 200))
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(res.Error)
		}
		return &res, nil
	}
}

// --- publishing copy --------------------------------------------------------------------------

// VideoPlatformCopy is what to paste on one platform.
type VideoPlatformCopy struct {
	Title string   `json:"title"`
	Desc  string   `json:"desc"`
	Tags  []string `json:"tags"`
}

// VideoPublishCopy has the copy for each platform, keyed 抖音 douyin / 视频号 channels /
// 小红书 xiaohongshu / B站 bilibili.
type VideoPublishCopy map[string]VideoPlatformCopy

// Per platform: title and description length (characters) and the number of topics.
var videoPlatformLimits = map[string][3]int{
	"douyin":      {30, 1000, 5},
	"channels":    {16, 1000, 5},
	"xiaohongshu": {20, 1000, 10},
	"bilibili":    {80, 2000, 10},
}

// PublishCopy returns the saved copy (nil when none was written yet).
func (s *VideoService) PublishCopy(ctx context.Context, userID int64, projectID string) (VideoPublishCopy, error) {
	p, err := s.own(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if s.exports == nil {
		return nil, nil
	}
	raw, err := s.exports.GetPublishCopy(ctx, p.ID)
	if err != nil || len(raw) == 0 {
		return nil, err
	}
	var c VideoPublishCopy
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, nil
	}
	return c, nil
}

// WritePublishCopy asks the project's model for titles, descriptions and topics for every platform,
// billed like the rest of the project's model calls, and saves them.
func (s *VideoService) WritePublishCopy(ctx context.Context, key *APIKey, projectID string) (VideoPublishCopy, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	p, err := s.own(ctx, key.UserID, projectID)
	if err != nil {
		return nil, err
	}
	if s.exports == nil || s.stream == nil {
		return nil, ErrVideoExportOff
	}
	if p.Spec == nil {
		return nil, ErrVideoNotReady
	}
	runKey, model, err := s.runKey(ctx, p, key.Key)
	if err != nil {
		return nil, err
	}
	var script strings.Builder
	for i, sc := range p.Spec.Scenes {
		fmt.Fprintf(&script, "%d. %s：%s\n", i+1, sc.Title, sc.Narration)
	}
	example := VideoPublishCopy{
		"douyin":      {Title: "一句抓人的标题", Desc: "两三句口语化简介，结尾引导关注或评论", Tags: []string{"大模型", "AI科普"}},
		"channels":    {Title: "6到16字短标题", Desc: "两三句简介，适合在微信里转发", Tags: []string{"人工智能"}},
		"xiaohongshu": {Title: "20字内标题可带emoji", Desc: "小红书笔记正文：分段、口语化、适量emoji、列出3到5个看点", Tags: []string{"AI学习", "干货分享"}},
		"bilibili":    {Title: "B站标题，信息完整", Desc: "B站简介：本期内容、时间轴要点、系列说明", Tags: []string{"科普", "人工智能"}},
	}
	schema, _ := json.MarshalIndent(example, "", "  ")
	sys := "你是短视频运营。根据视频的标题和旁白脚本，为抖音（douyin）、微信视频号（channels）、小红书（xiaohongshu）、B站（bilibili）各写一份发布文案：标题、简介、话题标签。" +
		"要求：标题长度不超过各平台上限（抖音 30 字、视频号短标题 6–16 字、小红书 20 字、B站 80 字）；简介贴合各平台风格；话题标签不带 # 号，抖音和视频号最多 5 个，小红书和B站最多 10 个；" +
		"内容必须与视频一致，不夸大、不编造数据，不写「最全」「第一」这类绝对化用语，不引导站外交易。只回答 JSON，结构如下：\n" + string(schema)
	user := fmt.Sprintf("视频标题：%s\n时长：约 %.0f 秒\n旁白脚本：\n%s", p.Title, p.Spec.TotalDuration(), clipRunes(script.String(), 6000))
	var out strings.Builder
	if _, err := s.stream(ctx, runKey, model, []LearnMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}}, 3000, func(d string) error {
		_, _ = out.WriteString(d)
		return nil
	}); err != nil {
		return nil, err
	}
	var c VideoPublishCopy
	raw := extractJSONObject(out.String())
	if raw == "" || json.Unmarshal([]byte(raw), &c) != nil {
		return nil, infraerrors.ServiceUnavailable("VIDEO_COPY", "文案生成失败，请再试一次（Copy generation failed）")
	}
	c = normalizePublishCopy(c)
	saved, _ := json.Marshal(c)
	if err := s.exports.SetPublishCopy(ctx, p.ID, saved); err != nil {
		return nil, err
	}
	return c, nil
}

func normalizePublishCopy(in VideoPublishCopy) VideoPublishCopy {
	out := VideoPublishCopy{}
	for platform, lim := range videoPlatformLimits {
		c := in[platform]
		c.Title = clipRunes(strings.TrimSpace(c.Title), lim[0])
		c.Desc = clipRunes(strings.TrimSpace(c.Desc), lim[1])
		var tags []string
		seen := map[string]bool{}
		for _, t := range c.Tags {
			t = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(t), "#＃"))
			if t == "" || seen[t] || len(tags) >= lim[2] {
				continue
			}
			seen[t] = true
			tags = append(tags, clipRunes(t, 20))
		}
		c.Tags = tags
		out[platform] = c
	}
	return out
}

// videoSiteOrigin is the public origin of the video site for a configured domain.
func videoSiteOrigin(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return strings.TrimRight(domain, "/")
	}
	return (&url.URL{Scheme: "https", Host: domain}).String()
}
