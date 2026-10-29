package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Review: after a film's scenes are written, the renderer takes stills of every new scene as the
// viewer sees it (subtitles drawn) and the model checks its own scene against them — overlapping or
// cut-off text, empty stages, a character drawn off-model, player errors — and fixes what is wrong.

// videoReviewFractions are the moments of a scene the model looks at (share of its duration).
var videoReviewFractions = []float64{0.2, 0.45, 0.7, 0.95}

const (
	videoStillsScale   = 0.5 // 1080×1920 → 540×960: enough to see layout, cheap in image tokens
	videoStillsTimeout = 3 * time.Minute
)

var (
	videoStillsRetry = 10 * time.Second
	videoStillsWait  = 2 * time.Minute // how long to wait for a busy renderer before skipping the review
)

type videoStillsJob struct {
	URL    string         `json:"url"`
	Width  int            `json:"width"`
	Height int            `json:"height"`
	Scale  float64        `json:"scale"`
	Times  []float64      `json:"times"`
	Setup  map[string]any `json:"setup"`
}

type videoStillsResult struct {
	Frames []string `json:"frames"` // base64 JPEG, one per requested time
	Errors []string `json:"errors"` // "<scene id> <phase> <message>" from the player
	Error  string   `json:"error"`
}

var errVideoRendererBusy = errors.New("renderer busy")

// stillsJob asks for the review moments of the scenes at the given indices, full frame at the film's
// own size; times returns, per scene index, the scene-time of each still in order.
func (s *VideoService) stillsJob(p *VideoProject, indices []int) (videoStillsJob, map[int][]float64) {
	s.fillImageURLs(p)
	scenes := make([]map[string]any, 0, len(p.Spec.Scenes))
	sceneData := make([]map[string]any, 0, len(p.Spec.Scenes))
	starts := make([]float64, len(p.Spec.Scenes))
	at := 0.0
	for i, sc := range p.Spec.Scenes {
		var words []VideoWord
		if sc.Audio != nil {
			words = sc.Audio.Words
		}
		scenes = append(scenes, map[string]any{"id": sc.ID, "duration": sc.Duration, "code": sc.Code, "words": words})
		sceneData = append(sceneData, map[string]any{"narration": sc.Narration, "duration": sc.Duration, "audio": sc.Audio})
		starts[i] = at
		at += sc.Duration
	}
	local := map[int][]float64{}
	var times []float64
	for _, i := range indices {
		d := p.Spec.Scenes[i].Duration
		for _, f := range videoReviewFractions {
			t := float64(int(d*f*100)) / 100
			local[i] = append(local[i], t)
			times = append(times, starts[i]+t)
		}
	}
	w, h := p.Spec.Width, p.Spec.Height
	return videoStillsJob{
		URL: s.renderer.origin + "/render.html", Width: w, Height: h, Scale: videoStillsScale, Times: times,
		Setup: map[string]any{
			// "landscape" is the full-frame layout (a portrait film is not wrapped), as on the site.
			"layout": VideoLayoutLandscape, "width": w, "height": h, "title": p.Title,
			"payload":   map[string]any{"width": w, "height": h, "theme": p.Spec.Theme, "loop": p.Spec.Loop, "common": p.Spec.Common, "scenes": scenes, "images": PlayerImages(p.Spec)},
			"scenes":    sceneData,
			"subtitles": p.Spec.Subtitles,
		},
	}, local
}

// callStills posts a stills job, waiting a while if the renderer is busy with an MP4.
func (s *VideoService) callStills(ctx context.Context, job videoStillsJob) (*videoStillsResult, error) {
	body, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(videoStillsWait)
	for {
		reqCtx, cancel := context.WithTimeout(ctx, videoStillsTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, s.renderer.url+"/frames", bytes.NewReader(body))
		if err != nil {
			cancel()
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.renderer.client.Do(req)
		if err != nil {
			cancel()
			return nil, err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		_ = resp.Body.Close()
		cancel()
		if resp.StatusCode == http.StatusConflict {
			if time.Now().Add(videoStillsRetry).After(deadline) {
				return nil, errVideoRendererBusy
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(videoStillsRetry):
			}
			continue
		}
		var res videoStillsResult
		if err := json.Unmarshal(raw, &res); err != nil {
			return nil, fmt.Errorf("renderer HTTP %d: %s", resp.StatusCode, clipRunes(string(raw), 200))
		}
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New(res.Error)
		}
		if len(res.Frames) != len(job.Times) {
			return nil, fmt.Errorf("renderer returned %d stills for %d times", len(res.Frames), len(job.Times))
		}
		return &res, nil
	}
}

// reviewEnabled: the renderer is there and the admin has not turned the review off.
func (s *VideoService) reviewEnabled(ctx context.Context, p *VideoProject) bool {
	return s.renderer != nil && p.Mode != VideoModeUpload && p.Spec != nil && !s.Settings(ctx).ReviewOff
}

// reviewScenes lets the model check the scenes at the given indices against stills of them and fix
// what it sees. Anything that goes wrong here leaves the scenes as they were written.
func (s *VideoService) reviewScenes(ctx context.Context, r *videoRun, indices []int) error {
	p := r.p
	r.mu.Lock()
	var todo []int
	for _, i := range indices {
		if i < len(p.Spec.Scenes) && strings.TrimSpace(p.Spec.Scenes[i].Code) != "" {
			todo = append(todo, i)
		}
	}
	// A copy to render from: fillImageURLs writes into the image lists.
	snapshot := *p
	snapshot.Options.Images = append([]VideoImage(nil), p.Options.Images...)
	spec := *p.Spec
	spec.Scenes = append([]VideoScene(nil), p.Spec.Scenes...)
	spec.Images = append([]VideoImage(nil), p.Spec.Images...)
	snapshot.Spec = &spec
	r.mu.Unlock()
	if len(todo) == 0 || !s.reviewEnabled(ctx, &snapshot) {
		return nil
	}
	s.step(ctx, r, "review", fmt.Sprintf("正在逐个检查 %d 个分镜的画面…", len(todo)))
	job, local := s.stillsJob(&snapshot, todo)
	res, err := s.callStills(ctx, job)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Warn("video: review skipped", "id", p.ID, "err", err)
		s.done(ctx, r, "画面检查跳过了（渲染服务正忙），成片不受影响")
		return nil
	}
	playerErrors := map[string][]string{}
	for _, e := range res.Errors {
		id, rest, _ := strings.Cut(strings.TrimSpace(e), " ")
		playerErrors[id] = append(playerErrors[id], rest)
	}

	style := findVideoStyle(p.Options.Style)
	category, _ := findVideoCategory(p.Options.Category)
	outline := make([]string, len(spec.Scenes))
	for i, sc := range spec.Scenes {
		outline[i] = fmt.Sprintf("%d.%s", i+1, sc.Title)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoSceneParallelism)
	var fixedMu sync.Mutex
	fixed := 0
	next := 0
	for _, i := range todo {
		frames := res.Frames[next : next+len(local[i])]
		next += len(local[i])
		g.Go(func() error {
			sc := spec.Scenes[i]
			brief := videoSceneBrief{Index: i, Scene: sc, Outline: outline, Spec: &spec, StyleBrief: style.Brief, Category: category.Brief, Motion: p.Mode == VideoModeMotion, Prompt: p.Prompt, Silent: videoSilent(p)}
			stills := make([]string, len(frames))
			for k, f := range frames {
				stills[k] = "data:image/jpeg;base64," + f
			}
			answer, err := s.chat(gctx, r, videoReviewPrompt(brief, sc.Code, stills, local[i], playerErrors[sc.ID]), videoSceneMaxTokens)
			if err != nil {
				if errors.Is(err, ErrVideoBalance) {
					return err
				}
				if gctx.Err() == nil {
					slog.Warn("video: scene review failed", "id", p.ID, "scene", sc.ID, "err", err)
				}
				return nil
			}
			if !strings.Contains(answer, "```") {
				return nil // OK
			}
			code := extractCodeBlock(answer)
			if problem := checkSceneWithKit(brief, code); problem != "" || code == sc.Code {
				return nil
			}
			r.mu.Lock()
			if i < len(p.Spec.Scenes) && p.Spec.Scenes[i].ID == sc.ID {
				p.Spec.Scenes[i].Code, p.Spec.Scenes[i].Error = code, ""
			}
			r.mu.Unlock()
			s.save(gctx, r)
			fixedMu.Lock()
			fixed++
			fixedMu.Unlock()
			s.done(gctx, r, fmt.Sprintf("分镜 %d「%s」检查后修正了画面", i+1, sc.Title))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if fixed == 0 {
		s.done(ctx, r, "画面检查完成，没有发现问题")
	} else {
		s.done(ctx, r, fmt.Sprintf("画面检查完成：修正了 %d 个分镜", fixed))
	}
	return nil
}
