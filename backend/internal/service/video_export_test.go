//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeExportRepo struct {
	mu      sync.Mutex
	exports map[string]*VideoExport
	order   []string
	copy    map[string]json.RawMessage
}

func newFakeExportRepo() *fakeExportRepo {
	return &fakeExportRepo{exports: map[string]*VideoExport{}, copy: map[string]json.RawMessage{}}
}

func (f *fakeExportRepo) CreateExport(_ context.Context, e *VideoExport) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	e.CreatedAt, e.UpdatedAt = time.Now(), time.Now()
	c := *e
	f.exports[e.ID] = &c
	f.order = append([]string{e.ID}, f.order...)
	return nil
}

func (f *fakeExportRepo) UpdateExport(_ context.Context, e *VideoExport) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *e
	f.exports[e.ID] = &c
	return nil
}

func (f *fakeExportRepo) GetExport(_ context.Context, id string) (*VideoExport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.exports[id]; ok {
		c := *e
		return &c, nil
	}
	return nil, nil
}

func (f *fakeExportRepo) ListExports(_ context.Context, projectID string) ([]VideoExport, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []VideoExport
	for _, id := range f.order {
		if e, ok := f.exports[id]; ok && e.ProjectID == projectID {
			out = append(out, *e)
		}
	}
	return out, nil
}

func (f *fakeExportRepo) DeleteExport(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.exports, id)
	return nil
}

func (f *fakeExportRepo) UnfinishedExports(context.Context) ([]string, error) { return nil, nil }

func (f *fakeExportRepo) CountExportsSince(_ context.Context, userID int64, _ time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, e := range f.exports {
		if e.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (f *fakeExportRepo) GetPublishCopy(_ context.Context, projectID string) (json.RawMessage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.copy[projectID], nil
}

func (f *fakeExportRepo) SetPublishCopy(_ context.Context, projectID string, raw json.RawMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.copy[projectID] = raw
	return nil
}

// fakeRenderer stands in for the videorender container: it writes the MP4 and cover it was asked for.
func fakeRenderer(t *testing.T, root string, jobs chan<- videoRenderJob) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var job videoRenderJob
		require.NoError(t, json.NewDecoder(r.Body).Decode(&job))
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, job.Out)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, job.Out), []byte("mp4"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(root, job.Cover), []byte("jpg"), 0o644))
		jobs <- job
		_ = json.NewEncoder(w).Encode(videoRenderResult{Size: 3, Duration: 4.2, Frames: 252})
	}))
}

func waitExport(t *testing.T, svc *VideoService, projectID, layout, status string) *VideoExport {
	t.Helper()
	var got *VideoExport
	require.Eventually(t, func() bool {
		list, err := svc.Exports(context.Background(), 7, projectID)
		require.NoError(t, err)
		for _, e := range list {
			if e.Layout == layout && e.Status == status {
				got = e
				return true
			}
		}
		return false
	}, 5*time.Second, 20*time.Millisecond)
	return got
}

func TestVideoExportRendersOncePerFilmAndLayout(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链"})
	require.NoError(t, err)
	svc.Wait()

	_, err = svc.StartExport(ctx, 7, p.ID, VideoLayoutPortrait)
	require.ErrorIs(t, err, ErrVideoExportOff)

	jobs := make(chan videoRenderJob, 10)
	srv := fakeRenderer(t, svc.audioDir, jobs)
	defer srv.Close()
	repo := newFakeExportRepo()
	svc.SetRenderer(repo, srv.URL, "https://video.example.com")
	require.True(t, svc.ExportEnabled())

	_, err = svc.StartExport(ctx, 7, p.ID, "square")
	require.ErrorIs(t, err, ErrVideoExportLayout)
	_, err = svc.StartExport(ctx, 8, p.ID, VideoLayoutPortrait)
	require.Error(t, err) // not the owner

	e, err := svc.StartExport(ctx, 7, p.ID, VideoLayoutPortrait)
	require.NoError(t, err)
	require.Equal(t, 1080, e.Width)
	require.Equal(t, 1920, e.Height)
	require.Equal(t, 60, e.FPS)
	job := <-jobs
	require.Equal(t, "https://video.example.com/render.html", job.URL)
	require.Equal(t, 1080, job.Width)
	require.Equal(t, "portrait", job.Setup["layout"])
	require.Len(t, job.Audio, 2) // one clip per narrated scene, the second at the first one's end
	require.Zero(t, job.Audio[0].Start)
	require.Greater(t, job.Audio[1].Start, 0.0)
	require.True(t, strings.HasPrefix(job.Out, p.ID+"/exports/"))

	ready := waitExport(t, svc, p.ID, VideoLayoutPortrait, VideoExportReady)
	require.True(t, ready.Current)
	require.Equal(t, 1.0, ready.Progress)
	require.NotEmpty(t, ready.URL)
	require.NotEmpty(t, ready.CoverURL)

	// The same film and layout is not rendered again.
	again, err := svc.StartExport(ctx, 7, p.ID, VideoLayoutPortrait)
	require.NoError(t, err)
	require.Equal(t, e.ID, again.ID)
	require.Len(t, jobs, 0)

	// The signed link serves the file under a readable name; a tampered one does not.
	u, err := url.Parse(ready.URL)
	require.NoError(t, err)
	parts := strings.Split(u.Path, "/")
	f, err := svc.ExportFile(ctx, parts[5], parts[6], u.Query().Get("exp"), u.Query().Get("sig"))
	require.NoError(t, err)
	require.Equal(t, "video/mp4", f.Mime)
	require.True(t, strings.HasSuffix(f.Name, "-竖屏-1080p60.mp4"), f.Name)
	_, err = svc.ExportFile(ctx, parts[5], parts[6], u.Query().Get("exp"), strings.Repeat("0", 32))
	require.ErrorIs(t, err, ErrVideoNotFound)
	_, err = svc.ExportFile(ctx, parts[5], "jpg", u.Query().Get("exp"), u.Query().Get("sig")) // signed for mp4 only
	require.ErrorIs(t, err, ErrVideoNotFound)

	// A changed film is rendered again, and the old render of that layout is removed.
	stored, _ := svc.repo.Get(ctx, p.ID)
	stored.Spec.Scenes[0].Code += "\n// changed"
	require.NoError(t, svc.repo.Save(ctx, stored))
	list, _ := svc.Exports(ctx, 7, p.ID)
	require.False(t, list[0].Current)
	second, err := svc.StartExport(ctx, 7, p.ID, VideoLayoutPortrait)
	require.NoError(t, err)
	require.NotEqual(t, e.ID, second.ID)
	<-jobs
	waitExport(t, svc, p.ID, VideoLayoutPortrait, VideoExportReady)
	require.Eventually(t, func() bool {
		list, _ := svc.Exports(ctx, 7, p.ID)
		return len(list) == 1 && list[0].ID == second.ID
	}, 5*time.Second, 20*time.Millisecond)
	require.NotEmpty(t, ready.File)
	_, statErr := os.Stat(filepath.Join(svc.audioDir, ready.File))
	require.True(t, os.IsNotExist(statErr), "the replaced render's file is deleted")
}

func TestVideoExportFailureIsReported(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链"})
	require.NoError(t, err)
	svc.Wait()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"chromium crashed"}`))
	}))
	defer srv.Close()
	svc.SetRenderer(newFakeExportRepo(), srv.URL, "https://video.example.com")
	_, err = svc.StartExport(ctx, 7, p.ID, VideoLayoutLandscape)
	require.NoError(t, err)
	failed := waitExport(t, svc, p.ID, VideoLayoutLandscape, VideoExportFailed)
	require.Contains(t, failed.Error, "chromium crashed")
	require.True(t, strings.HasPrefix(failed.Error, "导出失败"))
	// A failed render can be started again.
	retry, err := svc.StartExport(ctx, 7, p.ID, VideoLayoutLandscape)
	require.NoError(t, err)
	require.NotEqual(t, failed.ID, retry.ID)
}

func TestNormalizePublishCopy(t *testing.T) {
	c := normalizePublishCopy(VideoPublishCopy{
		"channels": {Title: "一个非常非常非常非常长的视频号短标题超过十六个字", Tags: []string{"#AI", "＃大模型", "AI", "", "a", "b", "c", "d"}},
		"unknown":  {Title: "ignored"},
	})
	require.Len(t, c, 4)
	require.Len(t, []rune(c["channels"].Title), 16)
	require.Equal(t, []string{"AI", "大模型", "a", "b", "c"}, c["channels"].Tags)
	_, ok := c["unknown"]
	require.False(t, ok)
}

func TestVideoWritePublishCopy(t *testing.T) {
	var gotSys string
	stream := func(_ context.Context, _, _ string, msgs []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		if strings.Contains(msgs[0].Content, "短视频运营") {
			gotSys = msgs[0].Content
			return &LearnTutorResult{}, onDelta("```json\n" + `{"douyin":{"title":"大模型是怎么说话的","desc":"一分钟看懂","tags":["#AI科普"]},"xiaohongshu":{"title":"小红书标题","desc":"正文","tags":["AI"]}}` + "\n```")
		}
		return (&fakeModel{broken: map[string]int{}}).stream(context.Background(), "", "", msgs, 0, onDelta)
	}
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, stream, t.TempDir())
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链"})
	require.NoError(t, err)
	svc.Wait()
	repo := newFakeExportRepo()
	svc.SetRenderer(repo, "http://render.invalid", "https://video.example.com")

	none, err := svc.PublishCopy(ctx, 7, p.ID)
	require.NoError(t, err)
	require.Nil(t, none)
	c, err := svc.WritePublishCopy(ctx, videoKeyA, p.ID)
	require.NoError(t, err)
	require.NotEmpty(t, gotSys)
	require.Equal(t, "大模型是怎么说话的", c["douyin"].Title)
	require.Equal(t, []string{"AI科普"}, c["douyin"].Tags)
	saved, err := svc.PublishCopy(ctx, 7, p.ID)
	require.NoError(t, err)
	require.Equal(t, c, saved)
	_, err = svc.PublishCopy(ctx, 8, p.ID)
	require.Error(t, err)
}
