//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeStills stands in for the renderer's /frames: one tiny "JPEG" per time, plus the given errors.
func fakeStills(t *testing.T, status int, errs []string, jobs chan<- videoStillsJob) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/frames" {
			http.NotFound(w, r)
			return
		}
		var job videoStillsJob
		require.NoError(t, json.NewDecoder(r.Body).Decode(&job))
		if jobs != nil {
			jobs <- job
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(videoStillsResult{Error: "busy"})
			return
		}
		frames := make([]string, len(job.Times))
		for i := range frames {
			frames[i] = "/9j/AA=="
		}
		_ = json.NewEncoder(w).Encode(videoStillsResult{Frames: frames, Errors: errs})
	}))
}

func eventTexts(t *testing.T, svc *VideoService, id string) string {
	events, err := svc.Events(context.Background(), 7, id, 0)
	require.NoError(t, err)
	var b strings.Builder
	for _, e := range events {
		b.WriteString(e.Text + "\n")
	}
	return b.String()
}

func TestVideoReviewFixesWhatItSees(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}, reviewFix: map[string]bool{"s1": true}}
	svc, _, _ := newTestVideoService(t, model)
	jobs := make(chan videoStillsJob, 4)
	srv := fakeStills(t, http.StatusOK, []string{"s2 update: t is not defined (line 9)"}, jobs)
	defer srv.Close()
	svc.SetRenderer(newFakeExportRepo(), srv.URL, "https://video.test")
	ctx := context.Background()

	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x", Options: VideoOptions{Ratio: "9:16"}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)

	job := <-jobs
	require.Equal(t, "https://video.test/render.html", job.URL)
	require.Equal(t, 1080, job.Width)
	require.Equal(t, 1920, job.Height)
	require.Len(t, job.Times, 2*len(videoReviewFractions), "four stills per scene")
	require.Equal(t, VideoLayoutLandscape, job.Setup["layout"], "full frame, subtitles as on the site")
	payload := job.Setup["payload"].(map[string]any)
	require.Equal(t, goodKit, payload["common"])
	// Scene 2's stills are taken inside scene 2.
	s1 := got.Spec.Scenes[0].Duration
	require.Greater(t, job.Times[4], s1)

	require.Equal(t, fixedScene, got.Spec.Scenes[0].Code, "the review's fix replaces scene 1")
	require.Equal(t, goodScene, got.Spec.Scenes[1].Code, "OK leaves scene 2 alone")
	require.Len(t, model.reviews, 2)
	for _, m := range model.reviews {
		require.Len(t, m.Images, 4)
		require.True(t, strings.HasPrefix(m.Images[0], "data:image/jpeg;base64,"))
	}
	var s2 LearnMessage
	for _, m := range model.reviews {
		if strings.Contains(m.Content, "t is not defined") {
			s2 = m
		}
	}
	require.NotEmpty(t, s2.Content, "the player's errors reach the scene's review")
	require.Contains(t, eventTexts(t, svc, p.ID), "修正了 1 个分镜")
}

func TestVideoReviewSkippedWhenRendererBusy(t *testing.T) {
	oldRetry, oldWait := videoStillsRetry, videoStillsWait
	videoStillsRetry, videoStillsWait = 10*time.Millisecond, 50*time.Millisecond
	defer func() { videoStillsRetry, videoStillsWait = oldRetry, oldWait }()

	model := &fakeModel{broken: map[string]int{}, reviewFix: map[string]bool{"s1": true}}
	svc, _, _ := newTestVideoService(t, model)
	srv := fakeStills(t, http.StatusConflict, nil, nil)
	defer srv.Close()
	svc.SetRenderer(newFakeExportRepo(), srv.URL, "https://video.test")
	ctx := context.Background()

	p, _ := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, goodScene, got.Spec.Scenes[0].Code)
	require.Empty(t, model.reviews)
	require.Contains(t, eventTexts(t, svc, p.ID), "画面检查跳过了")
}

func TestVideoReviewOffAndNoRenderer(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	// No renderer: no review.
	p, _ := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Empty(t, model.reviews)

	// Turned off by the admin: the renderer is never asked.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	svc.SetRenderer(newFakeExportRepo(), srv.URL, "https://video.test")
	svc.settingsCache.mu.Lock()
	svc.settingsCache.val, svc.settingsCache.at = &VideoSettings{ReviewOff: true}, time.Now()
	svc.settingsCache.mu.Unlock()
	p, _ = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "y"})
	svc.Wait()
	got, _ = svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Zero(t, calls.Load())
	require.Empty(t, model.reviews)
}

func TestVideoReviewPrompt(t *testing.T) {
	spec := &VideoSpec{Width: 1080, Height: 1920, Common: goodKit, Facts: []string{"订阅费 ¥1,450/月"}}
	brief := videoSceneBrief{Index: 0, Scene: VideoScene{ID: "s1", Title: "开场", Narration: "你好", Visual: "v", Duration: 5}, Outline: []string{"1.开场"}, Spec: spec}
	msgs := videoReviewPrompt(brief, goodScene, []string{"data:image/jpeg;base64,AA"}, []float64{1, 2.25}, []string{"update: boom"})
	require.Contains(t, msgs[0].Content, "checking a scene you wrote")
	require.Contains(t, msgs[0].Content, "## Review checklist")
	require.NotContains(t, msgs[0].Content, "Answer with the function body only")
	require.Contains(t, msgs[0].Content, "## Shared kit")
	require.Contains(t, msgs[0].Content, "订阅费 ¥1,450/月")
	require.Equal(t, "assistant", msgs[len(msgs)-2].Role)
	last := msgs[len(msgs)-1]
	require.Contains(t, last.Content, "t = 1.0 s, 2.2 s")
	require.Contains(t, last.Content, "update: boom")
	require.Len(t, last.Images, 1)
}
