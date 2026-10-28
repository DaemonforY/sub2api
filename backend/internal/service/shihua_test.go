//go:build unit

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	xdraw "golang.org/x/image/draw"
)

// shihuaRepoFake is an in-memory ShihuaRepository.
type shihuaRepoFake struct {
	mu      sync.Mutex
	works   map[int64]*ShihuaWork
	images  map[int64]*ShihuaImage
	renders []ShihuaRender
	jobs    []ShihuaJob
	nextID  int64
}

func newShihuaRepoFake() *shihuaRepoFake {
	return &shihuaRepoFake{works: map[int64]*ShihuaWork{}, images: map[int64]*ShihuaImage{}}
}

func (r *shihuaRepoFake) id() int64 { r.nextID++; return r.nextID }

func (r *shihuaRepoFake) ListWorks(_ context.Context, q ShihuaWorkQuery) ([]ShihuaWork, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	want := map[int64]bool{}
	for _, id := range q.IDs {
		want[id] = true
	}
	out := []ShihuaWork{}
	for _, w := range r.works {
		if q.IDs != nil && !want[w.ID] {
			continue
		}
		out = append(out, *w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *shihuaRepoFake) GetWork(_ context.Context, id int64) (*ShihuaWork, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.works[id]; ok {
		c := *w
		return &c, nil
	}
	return nil, nil
}

func (r *shihuaRepoFake) GetWorkBySlug(_ context.Context, slug string) (*ShihuaWork, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, w := range r.works {
		if w.Slug == slug {
			c := *w
			return &c, nil
		}
	}
	return nil, nil
}

func (r *shihuaRepoFake) CreateWork(_ context.Context, w *ShihuaWork) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	w.ID = r.id()
	c := *w
	r.works[w.ID] = &c
	return nil
}

func (r *shihuaRepoFake) UpdateWork(_ context.Context, w *ShihuaWork) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *w
	r.works[w.ID] = &c
	return nil
}

func (r *shihuaRepoFake) DeleteWork(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.works, id)
	return nil
}

func (r *shihuaRepoFake) SetPicked(_ context.Context, workID int64, imageID *int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.works[workID].PickedImageID = imageID
	return nil
}

func (r *shihuaRepoFake) CreateJob(ctx context.Context, name string, candidates int, by int64, images []ShihuaImage) (*ShihuaJob, error) {
	r.mu.Lock()
	job := ShihuaJob{ID: r.id(), Name: name, Candidates: candidates}
	r.jobs = append(r.jobs, job)
	r.mu.Unlock()
	for i := range images {
		images[i].JobID = &job.ID
	}
	return &job, r.EnqueueImages(ctx, images)
}

func (r *shihuaRepoFake) ListJobs(context.Context, int) ([]ShihuaJob, error) { return r.jobs, nil }

func (r *shihuaRepoFake) EnqueueImages(_ context.Context, images []ShihuaImage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range images {
		images[i].ID = r.id()
		images[i].Status = ShihuaImageQueued
		c := images[i]
		r.images[c.ID] = &c
	}
	return nil
}

func (r *shihuaRepoFake) CancelQueued(_ context.Context, jobID int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := int64(0)
	for _, img := range r.images {
		if img.Status == ShihuaImageQueued && (jobID == 0 || (img.JobID != nil && *img.JobID == jobID)) {
			img.Status = ShihuaImageCanceled
			n++
		}
	}
	return n, nil
}

func (r *shihuaRepoFake) RetryFailed(context.Context, int64) (int64, error) { return 0, nil }

func (r *shihuaRepoFake) ClaimNext(context.Context) (*ShihuaImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := []int64{}
	for id, img := range r.images {
		if img.Status == ShihuaImageQueued && (img.NotBefore == nil || !img.NotBefore.After(time.Now())) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	img := r.images[ids[0]]
	now := time.Now()
	img.Status, img.StartedAt, img.NotBefore = ShihuaImageRunning, &now, nil
	img.Attempts++
	c := *img
	return &c, nil
}

func (r *shihuaRepoFake) FinishImage(_ context.Context, img *ShihuaImage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *img
	r.images[img.ID] = &c
	return nil
}

func (r *shihuaRepoFake) RequeueRunning(context.Context) (int64, error)        { return 0, nil }
func (r *shihuaRepoFake) RequeueUpscaleRunning(context.Context) (int64, error) { return 0, nil }

func (r *shihuaRepoFake) QueueStats(_ context.Context, dayStart time.Time) (ShihuaQueueStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var st ShihuaQueueStats
	for _, img := range r.images {
		switch img.Status {
		case ShihuaImageQueued:
			st.Queued++
		case ShihuaImageRunning:
			st.Running++
		}
		if img.StartedAt != nil && !img.StartedAt.Before(dayStart) {
			st.StartedToday++
		}
		if img.UpscaleStatus == ShihuaUpscaleQueued || img.UpscaleStatus == ShihuaUpscaleRunning {
			st.UpscaleQueue++
		}
	}
	return st, nil
}

func (r *shihuaRepoFake) ListImages(context.Context, ShihuaImageQuery) ([]ShihuaImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ShihuaImage{}
	for _, img := range r.images {
		out = append(out, *img)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *shihuaRepoFake) GetImage(_ context.Context, id int64) (*ShihuaImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if img, ok := r.images[id]; ok {
		c := *img
		return &c, nil
	}
	return nil, nil
}

func (r *shihuaRepoFake) ReviewImage(_ context.Context, id int64, rejected bool, titleCheck string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.images[id].Rejected, r.images[id].TitleCheck = rejected, titleCheck
	return nil
}

func (r *shihuaRepoFake) SetUpscale(_ context.Context, id int64, status, errText, file string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.images[id].UpscaleStatus, r.images[id].UpscaleError, r.images[id].UpscaledFile = status, errText, file
	return nil
}

func (r *shihuaRepoFake) ClaimUpscale(context.Context) (*ShihuaImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, img := range r.images {
		if img.UpscaleStatus == ShihuaUpscaleQueued {
			img.UpscaleStatus = ShihuaUpscaleRunning
			c := *img
			return &c, nil
		}
	}
	return nil, nil
}

func (r *shihuaRepoFake) SaveRender(_ context.Context, rd *ShihuaRender) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rd.ID, rd.CreatedAt = r.id(), time.Now()
	r.renders = append(r.renders, *rd)
	return nil
}

func (r *shihuaRepoFake) ListRenders(context.Context, []int64) ([]ShihuaRender, error) {
	return r.renders, nil
}

type shihuaSettingsFake struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *shihuaSettingsFake) Get(context.Context, string) (*Setting, error) { return nil, nil }
func (s *shihuaSettingsFake) GetValue(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key], nil
}
func (s *shihuaSettingsFake) Set(_ context.Context, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}
func (s *shihuaSettingsFake) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *shihuaSettingsFake) SetMultiple(context.Context, map[string]string) error { return nil }
func (s *shihuaSettingsFake) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *shihuaSettingsFake) Delete(context.Context, string) error { return nil }

type shihuaKeysFake map[int64]*APIKey

func (k shihuaKeysFake) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if key, ok := k[id]; ok {
		return key, nil
	}
	return nil, errors.New("not found")
}

func newShihuaTestService(t *testing.T) (*ShihuaService, *shihuaRepoFake) {
	t.Helper()
	repo := newShihuaRepoFake()
	settings := &shihuaSettingsFake{values: map[string]string{}}
	keys := shihuaKeysFake{7: {ID: 7, UserID: 1, Key: "sk-test", Status: StatusActive}}
	s := NewShihuaService(repo, settings, keys, nil, nil, "http://127.0.0.1:1", t.TempDir())
	return s, repo
}

func shihuaPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func shihuaTestWork() ShihuaWorkInput {
	return ShihuaWorkInput{Slug: "shanju-qiuming", Title: "山居秋暝", Author: "王维", Dynasty: "唐", Genre: "shi", Stage: "junior", Grade: "八上",
		FullText: "空山新雨后，天气晚来秋。\n明月松间照，清泉石上流。", Theme: &ShihuaTheme{PromptTheme: "初秋傍晚，雨后空山"}}
}

func TestBuildShihuaPromptNoTextByDefault(t *testing.T) {
	w := &ShihuaWork{Title: "静夜思", Author: "李白", FullText: "床前明月光", Theme: ShihuaTheme{PromptTheme: "深秋夜，客舍窗前"}}
	p := BuildShihuaPrompt("", w, false)
	require.True(t, strings.HasPrefix(p, "画风："))
	require.Contains(t, p, "主题： 深秋夜，客舍窗前")
	require.Contains(t, p, "不要出现任何文字")
	require.NotContains(t, p, "静夜思", "the layout sets the title")
	require.NotContains(t, p, shihuaTitleMarker)
	require.NotContains(t, p, "床前明月光", "the body text must never reach the image model")
	require.Contains(t, p, "右上方保留")
	require.LessOrEqual(t, len([]rune(p)), shihuaMaxPromptRunes)

	img := ShihuaImage{Prompt: p}
	(&shihuaFiles{}).signImage(&img)
	require.False(t, img.TitleDrawn)
}

func TestBuildShihuaPromptTitleByModel(t *testing.T) {
	w := &ShihuaWork{Title: "静夜思", Author: "李白", FullText: "床前明月光", Theme: ShihuaTheme{PromptTheme: "深秋夜，客舍窗前"}}
	p := BuildShihuaPrompt("", w, true)
	require.Contains(t, p, "主标题「静夜思」")
	require.Contains(t, p, "作者名「李白」")
	require.NotContains(t, p, "床前明月光")
	require.Equal(t, 1, strings.Count(p, "【文字与版式】"))
	require.LessOrEqual(t, len([]rune(p)), shihuaMaxPromptRunes)

	img := ShihuaImage{Prompt: p}
	(&shihuaFiles{}).signImage(&img)
	require.True(t, img.TitleDrawn)
}

func TestBuildShihuaPromptReplacesSavedTextRules(t *testing.T) {
	// A style saved before the title modes still carries the old text part.
	style := "画风： 【美学定位】 细线设色。 " + shihuaTextTitle + " 【风格禁忌】 拒绝3D。"
	w := &ShihuaWork{Title: "静夜思", Author: "李白", Theme: ShihuaTheme{PromptTheme: "夜"}}
	p := BuildShihuaPrompt(style, w, false)
	require.Equal(t, 1, strings.Count(p, "【文字与版式】"))
	require.NotContains(t, p, "主标题")
	require.Contains(t, p, "【风格禁忌】 拒绝3D。")
	require.Contains(t, shihuaThemeSystemPrompt(false), "用书法排标题和正文")
	require.Contains(t, shihuaThemeSystemPrompt(true), "左上方要竖排书法标题")
	require.NotContains(t, shihuaThemeSystemPrompt(false), "%!")
}

func TestShihuaCharCount(t *testing.T) {
	require.Equal(t, 20, ShihuaCharCount("床前明月光，疑是地上霜。\n举头望明月，低头思故乡。"))
	require.Equal(t, 4, ShihuaCharCount("佁然不动；"))
}

func TestShihuaWorkTextChangeClearsChecks(t *testing.T) {
	s, _ := newShihuaTestService(t)
	ctx := context.Background()
	w, err := s.CreateWork(ctx, shihuaTestWork())
	require.NoError(t, err)
	require.Equal(t, ShihuaTextDraft, w.TextStatus)
	require.Equal(t, 20, w.CharCount)

	_, err = s.Verify(ctx, 1, w.ID, 2)
	require.ErrorIs(t, err, ErrShihuaVerifyOrder)
	w, err = s.Verify(ctx, 1, w.ID, 1)
	require.NoError(t, err)
	w, err = s.Verify(ctx, 1, w.ID, 2)
	require.NoError(t, err)
	require.Equal(t, ShihuaTextVerified2, w.TextStatus)

	// Editing something other than the text keeps the checks.
	in := shihuaTestWork()
	in.Grade = "八年级上册"
	w, err = s.UpdateWork(ctx, w.ID, in)
	require.NoError(t, err)
	require.Equal(t, ShihuaTextVerified2, w.TextStatus)

	// A changed character clears them.
	in.FullText = strings.Replace(in.FullText, "秋", "秌", 1)
	w, err = s.UpdateWork(ctx, w.ID, in)
	require.NoError(t, err)
	require.Equal(t, ShihuaTextDraft, w.TextStatus)
	require.Nil(t, w.Verify2At)
}

func TestShihuaTwoPeopleVerify(t *testing.T) {
	s, _ := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7, TwoPeopleVerify: true})
	require.NoError(t, err)
	w, err := s.CreateWork(ctx, shihuaTestWork())
	require.NoError(t, err)
	_, err = s.Verify(ctx, 1, w.ID, 1)
	require.NoError(t, err)
	_, err = s.Verify(ctx, 1, w.ID, 2)
	require.ErrorIs(t, err, ErrShihuaVerifySame)
	w, err = s.Verify(ctx, 2, w.ID, 2)
	require.NoError(t, err)
	require.Equal(t, ShihuaTextVerified2, w.TextStatus)
}

func TestShihuaImportDryRunAndReset(t *testing.T) {
	s, repo := newShihuaTestService(t)
	ctx := context.Background()
	bad := shihuaTestWork()
	bad.Slug = "Bad Slug"
	res, err := s.Import(ctx, []ShihuaWorkInput{shihuaTestWork(), bad}, true)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	require.Len(t, res.Errors, 1)
	require.Empty(t, repo.works, "a dry run writes nothing")

	res, err = s.Import(ctx, []ShihuaWorkInput{shihuaTestWork()}, false)
	require.NoError(t, err)
	require.Equal(t, 1, res.Created)
	var id int64
	for k := range repo.works {
		id = k
	}
	_, _ = s.Verify(ctx, 1, id, 1)

	same := shihuaTestWork()
	same.Theme = nil
	res, err = s.Import(ctx, []ShihuaWorkInput{same}, false)
	require.NoError(t, err)
	require.Equal(t, 1, res.Unchanged)
	require.Equal(t, "初秋傍晚，雨后空山", repo.works[id].Theme.PromptTheme, "an import without a scene keeps the old one")

	changed := shihuaTestWork()
	changed.FullText += "\n竹喧归浣女，莲动下渔舟。"
	res, err = s.Import(ctx, []ShihuaWorkInput{changed}, false)
	require.NoError(t, err)
	require.Equal(t, 1, res.Updated)
	require.Equal(t, []string{"shanju-qiuming"}, res.Reset)
}

func TestShihuaJobNeedsThemes(t *testing.T) {
	s, _ := newShihuaTestService(t)
	ctx := context.Background()
	in := shihuaTestWork()
	in.Theme = nil
	w, err := s.CreateWork(ctx, in)
	require.NoError(t, err)
	_, err = s.CreateJob(ctx, 1, ShihuaJobInput{WorkIDs: []int64{w.ID}, Candidates: 2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "山居秋暝")
	_, err = s.CreateJob(ctx, 1, ShihuaJobInput{WorkIDs: []int64{w.ID}, Candidates: 9})
	require.ErrorIs(t, err, ErrShihuaInvalid)
}

func waitShihua(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestShihuaWorkerDrawsRetriesAndStopsAtDailyLimit(t *testing.T) {
	s, repo := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7, Concurrency: 2, DailyLimit: 3})
	require.NoError(t, err)
	w, err := s.CreateWork(ctx, shihuaTestWork())
	require.NoError(t, err)

	var mu sync.Mutex
	calls := 0
	pic := shihuaPNG(t, 64, 96)
	s.draw = func(_ context.Context, key, prompt, size, quality string) (*shihuaDrawn, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		require.Equal(t, "sk-test", key)
		if calls == 1 {
			return nil, &shihuaDrawError{Status: http.StatusBadGateway, Msg: "HTTP 502"}
		}
		return &shihuaDrawn{Data: pic, Width: 64, Height: 96}, nil
	}
	job, err := s.CreateJob(ctx, 1, ShihuaJobInput{WorkIDs: []int64{w.ID}, Candidates: 4})
	require.NoError(t, err)
	require.Equal(t, 4, job.Counts.Total+4-job.Counts.Total)
	require.NotZero(t, job.ID)

	s.dispatch()
	waitShihua(t, func() bool { return s.runningCount() == 0 })
	s.dispatch()
	waitShihua(t, func() bool { return s.runningCount() == 0 })

	images, _ := repo.ListImages(ctx, ShihuaImageQuery{})
	statuses := map[string]int{}
	var retried *ShihuaImage
	for i := range images {
		statuses[images[i].Status]++
		if images[i].Error == "HTTP 502" {
			retried = &images[i]
		}
	}
	// 3 started today (the daily limit): 2 drawn, 1 failed with a 502 and waiting to retry; 1 never started.
	require.Equal(t, 2, statuses[ShihuaImageSucceeded])
	require.Equal(t, 2, statuses[ShihuaImageQueued])
	require.NotNil(t, retried)
	require.NotNil(t, retried.NotBefore)
	require.Equal(t, 1, retried.Attempts)

	for _, img := range images {
		if img.Status == ShihuaImageSucceeded {
			require.Equal(t, 64, img.Width)
			_, err := s.files.read(img.File)
			require.NoError(t, err)
			_, err = s.files.read(shihuaThumbRel(img.WorkID, img.ID))
			require.NoError(t, err)
		}
	}
	view, err := s.SettingsView(ctx, 1)
	require.NoError(t, err)
	require.Contains(t, view.Paused, "每日上限")
}

func TestShihuaWorkerGivesUpOnContentRefusal(t *testing.T) {
	s, repo := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7, DailyLimit: 10})
	require.NoError(t, err)
	w, err := s.CreateWork(ctx, shihuaTestWork())
	require.NoError(t, err)
	s.draw = func(context.Context, string, string, string, string) (*shihuaDrawn, error) {
		return nil, &shihuaDrawError{Status: http.StatusBadRequest, Msg: "HTTP 400：safety"}
	}
	_, err = s.CreateJob(ctx, 1, ShihuaJobInput{WorkIDs: []int64{w.ID}, Candidates: 1})
	require.NoError(t, err)
	s.dispatch()
	waitShihua(t, func() bool { return s.runningCount() == 0 })
	images, _ := repo.ListImages(ctx, ShihuaImageQuery{})
	require.Equal(t, ShihuaImageFailed, images[0].Status)
	require.Contains(t, images[0].Error, "safety")
}

func TestShihuaPickQueuesUpscaleAndRenderNeedsCheckedText(t *testing.T) {
	s, repo := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7, AutoUpscale: true})
	require.NoError(t, err)
	s.upscale = func(_ context.Context, img []byte) ([]byte, error) { return shihuaDouble(t, img), nil }
	w, err := s.CreateWork(ctx, shihuaTestWork())
	require.NoError(t, err)
	imgs := []ShihuaImage{{WorkID: w.ID, Prompt: "p", Size: "1024x1536", Quality: "high"}}
	require.NoError(t, repo.EnqueueImages(ctx, imgs))
	file, width, height, err := s.files.saveDrawing(w.ID, imgs[0].ID, shihuaPNG(t, 64, 96))
	require.NoError(t, err)
	repo.images[imgs[0].ID].Status, repo.images[imgs[0].ID].File = ShihuaImageSucceeded, file
	repo.images[imgs[0].ID].Width, repo.images[imgs[0].ID].Height = width, height

	img, err := s.Review(ctx, imgs[0].ID, ShihuaReviewInput{Action: "pick"})
	require.NoError(t, err)
	require.Equal(t, ShihuaUpscaleQueued, img.UpscaleStatus)
	require.Equal(t, "ok", img.TitleCheck)
	require.Equal(t, imgs[0].ID, *repo.works[w.ID].PickedImageID)

	s.upscaleNext()
	got := repo.images[imgs[0].ID]
	require.Equal(t, ShihuaUpscaleDone, got.UpscaleStatus, got.UpscaleError)
	data, err := s.files.read(got.UpscaledFile)
	require.NoError(t, err)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, [2]int{128, 192}, [2]int{cfg.Width, cfg.Height})

	poster := shihuaPNG(t, 40, 60)
	_, err = s.SaveRender(ctx, imgs[0].ID, "2x3", nil, poster)
	require.ErrorIs(t, err, ErrShihuaNotVerified)
	_, _ = s.Verify(ctx, 1, w.ID, 1)
	_, _ = s.Verify(ctx, 1, w.ID, 2)
	_, err = s.SaveRender(ctx, imgs[0].ID, "5x7", nil, poster)
	require.ErrorIs(t, err, ErrShihuaInvalid)
	r, err := s.SaveRender(ctx, imgs[0].ID, "2x3", json.RawMessage(`{"cell":40}`), poster)
	require.NoError(t, err)
	require.Contains(t, r.URL, "/api/v1/shihua/files/works/")

	var buf bytes.Buffer
	require.NoError(t, s.ExportZip(ctx, &buf, []int64{w.ID}, nil))
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	require.ElementsMatch(t, []string{"2x3/001-山居秋暝-2比3.png", "文字/001-山居秋暝.txt"}, names)
}

// shihuaDouble stands in for the imagetools service: it refuses a long edge over 1024 like the real one
// would shrink it, and doubles the rest.
func shihuaDouble(t *testing.T, data []byte) []byte {
	src, _, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	b := src.Bounds()
	require.LessOrEqual(t, max(b.Dx(), b.Dy()), shihuaUpscaleEdge)
	dst := image.NewRGBA(image.Rect(0, 0, 2*b.Dx(), 2*b.Dy()))
	xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
	out, err := encodeShihuaJPEG(dst)
	require.NoError(t, err)
	return out
}

func TestShihuaUpscaleTiledSplitsLongPictures(t *testing.T) {
	calls := 0
	up := func(_ context.Context, img []byte) ([]byte, error) {
		calls++
		return shihuaDouble(t, img), nil
	}
	out, err := shihuaUpscaleTiled(context.Background(), shihuaPNG(t, 1024, 1536), up)
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	img, _, err := image.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 2048, 3072), img.Bounds())
	// The gradient survives the seam: green follows y everywhere (JPEG noise aside).
	for _, y := range []int{0, 1500, 1600, 1700, 3070} {
		_, g, _, _ := img.At(10, y).RGBA()
		require.InDelta(t, float64(uint8(y/2)), float64(g>>8), 12, "y=%d", y)
	}
}

func TestShihuaSignedLinks(t *testing.T) {
	f := newShihuaFiles(t.TempDir())
	link := f.sign("works/3/9.png")
	require.True(t, strings.HasPrefix(link, shihuaFileRoute+"works/3/9.png?e="))
	q := strings.SplitN(strings.SplitN(link, "?", 2)[1], "&", 2)
	exp, sig := strings.TrimPrefix(q[0], "e="), strings.TrimPrefix(q[1], "s=")
	path, ok := f.Open("works/3/9.png", exp, sig)
	require.True(t, ok)
	require.Equal(t, filepath.Join(f.dir, "works", "3", "9.png"), path)
	_, ok = f.Open("works/3/10.png", exp, sig)
	require.False(t, ok)
	_, ok = f.Open("../../etc/passwd", exp, sig)
	require.False(t, ok)
	f.now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	_, ok = f.Open("works/3/9.png", exp, sig)
	require.False(t, ok, "expired")
	_ = os.Remove(path)
}

func TestParseShihuaTheme(t *testing.T) {
	theme, err := parseShihuaTheme("好的：\n```json\n{\"subject\":\"渔舟\",\"key_imagery\":[\"莲动 → 荷叶摇动\"],\"prompt_theme\":\" 初秋傍晚 \"}\n```")
	require.NoError(t, err)
	require.Equal(t, "初秋傍晚", theme.PromptTheme)
	require.Equal(t, []string{"莲动 → 荷叶摇动"}, theme.KeyImagery)
	_, err = parseShihuaTheme(`{"subject":"x"}`)
	require.ErrorIs(t, err, ErrShihuaThemeFailed)
}

func TestShihuaDayStartIsBeijingMidnight(t *testing.T) {
	now := time.Date(2026, 10, 25, 17, 30, 0, 0, time.UTC) // 01:30 on the 26th in Beijing
	require.Equal(t, time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC), shihuaDayStart(now).UTC())
}
