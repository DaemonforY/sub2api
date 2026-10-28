//go:build unit

package service

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// materialQueueRepo adds an in-memory drawing queue to the prompt fixture's repo.
type materialQueueRepo struct {
	*materialPromptRepo
	mu      sync.Mutex
	arts    []*MaterialArtwork
	renders []MaterialRender
	now     func() time.Time
}

func (r *materialQueueRepo) EnqueueArtworks(_ context.Context, arts []*MaterialArtwork) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range arts {
		a.ID = int64(len(r.arts) + 1)
		a.CreatedAt = r.now()
		cp := *a
		r.arts = append(r.arts, &cp)
	}
	return nil
}

func (r *materialQueueRepo) ClaimNextArtwork(context.Context) (*MaterialArtwork, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.arts {
		if a.Status == MaterialArtworkQueued && (a.NotBefore == nil || !a.NotBefore.After(r.now())) {
			now := r.now()
			a.Status, a.Attempts, a.StartedAt, a.Error = MaterialArtworkRunning, a.Attempts+1, &now, ""
			cp := *a
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *materialQueueRepo) FinishArtwork(_ context.Context, a *MaterialArtwork) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *a
	r.arts[a.ID-1] = &cp
	return nil
}

func (r *materialQueueRepo) RequeueRunningArtworks(context.Context) (int64, error) { return 0, nil }

func (r *materialQueueRepo) ArtworkQueueStats(_ context.Context, dayStart time.Time) (MaterialQueueStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var s MaterialQueueStats
	for _, a := range r.arts {
		switch a.Status {
		case MaterialArtworkQueued:
			s.Queued++
		case MaterialArtworkRunning:
			s.Running++
		}
		if a.StartedAt != nil && !a.StartedAt.Before(dayStart) {
			s.StartedToday++
		}
	}
	return s, nil
}

func (r *materialQueueRepo) ListArtworks(context.Context, MaterialArtworkQuery) ([]MaterialArtwork, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []MaterialArtwork{}
	for _, a := range r.arts {
		out = append(out, *a)
	}
	return out, len(out), nil
}

func (r *materialQueueRepo) GetArtwork(_ context.Context, id int64) (*MaterialArtwork, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id < 1 || int(id) > len(r.arts) {
		return nil, nil
	}
	cp := *r.arts[id-1]
	return &cp, nil
}

func (r *materialQueueRepo) ReviewArtwork(_ context.Context, id int64, status string, rating int, reason string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.arts[id-1]
	a.ReviewStatus, a.Rating, a.RejectReason = status, rating, reason
	return true, nil
}

func (r *materialQueueRepo) CancelQueuedArtworks(_ context.Context, ids []int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, id := range ids {
		if a := r.arts[id-1]; a.Status == MaterialArtworkQueued {
			a.Status = MaterialArtworkCanceled
			n++
		}
	}
	return n, nil
}

func (r *materialQueueRepo) RetryArtwork(_ context.Context, id int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a := r.arts[id-1]
	if a.Status != MaterialArtworkFailed && a.Status != MaterialArtworkCanceled {
		return false, nil
	}
	a.Status, a.Attempts, a.NotBefore, a.Error, a.FinishedAt = MaterialArtworkQueued, 0, nil, "", nil
	return true, nil
}

type materialFakeDrawer struct {
	mu          sync.Mutex
	ready       bool
	concurrency int
	limit       int
	results     []error // per call; nil = success
	calls       []string
	png         []byte
	gate        chan struct{} // when set, each drawing waits for it
}

func (d *materialFakeDrawer) AdminDraw(_ context.Context, prompt, size, quality string) (*shihuaDrawn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, size+"|"+quality+"|"+prompt)
	gate := d.gate
	d.mu.Unlock()
	if gate != nil {
		<-gate
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.results) > 0 {
		err := d.results[0]
		d.results = d.results[1:]
		if err != nil {
			return nil, err
		}
	}
	return &shihuaDrawn{Data: d.png, Width: 64, Height: 96}, nil
}

func (d *materialFakeDrawer) DrawLimits(context.Context) (int, int, bool) {
	return d.concurrency, d.limit, d.ready
}

func (d *materialFakeDrawer) ImageCost(w, h int) float64 { return float64(w*h) / 1e6 }

func newMaterialQueueFixture(t *testing.T) (*MaterialService, *materialQueueRepo, *materialFakeDrawer, *time.Time) {
	seg, links, byID, styles := materialPromptFixture()
	clock := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	repo := &materialQueueRepo{materialPromptRepo: &materialPromptRepo{materialMemRepo: newMaterialMemRepo(), styles: styles},
		now: func() time.Time { return clock }}
	for id := int64(1); id <= 4; id++ {
		repo.imagery = append(repo.imagery, byID[id])
	}
	repo.segments[1] = seg
	repo.links[1] = links
	drawer := &materialFakeDrawer{ready: true, concurrency: 2, limit: 10, png: shihuaPNG(t, 64, 96)}
	svc := &MaterialService{repo: repo, now: func() time.Time { return clock }}
	svc.EnableDrawing(drawer, t.TempDir())
	return svc, repo, drawer, &clock
}

// runQueue dispatches until nothing more starts and waits for the drawings.
func runQueue(svc *MaterialService) {
	for i := 0; i < 10; i++ {
		svc.dispatch()
		svc.queue.wg.Wait()
	}
}

// drawOne claims and draws exactly one due drawing, synchronously; false when none is due.
func drawOne(t *testing.T, svc *MaterialService) bool {
	t.Helper()
	art, err := svc.repo.ClaimNextArtwork(context.Background())
	require.NoError(t, err)
	if art == nil {
		return false
	}
	q := svc.queue
	q.mu.Lock()
	q.running++
	q.mu.Unlock()
	q.wg.Add(1)
	svc.runDraw(art)
	return true
}

func TestMaterialDrawSegment(t *testing.T) {
	svc, repo, drawer, _ := newMaterialQueueFixture(t)
	ctx := context.Background()

	arts, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20, Mood: "冷白月光"}, 2)
	require.NoError(t, err)
	require.Len(t, arts, 2)
	a := arts[0]
	require.Equal(t, MaterialArtworkQueued, a.Status)
	require.Equal(t, int64(20), *a.StyleID)
	require.Equal(t, int64(1), *a.SegmentID)
	require.Equal(t, int64(7), *a.CreatedBy)
	require.Equal(t, "1024x1536", a.Size)
	require.Equal(t, []int64{1, 2, 3}, a.ImageryIDs)
	require.True(t, a.TextOverlayAllowed)
	require.Contains(t, a.PromptFinal, "Cool silver moonlight")
	require.Contains(t, string(a.Options), `"mood":"冷白月光"`)
	require.Contains(t, string(a.PromptParts), `"frame":`)

	_, err = svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 5)
	require.ErrorIs(t, err, ErrMaterialInvalid)

	repo.terms = []MaterialBlockTerm{{Term: "(?i)ghibli", Kind: "studio", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true}}
	_, err = svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20, Extra: "like Ghibli"}, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Ghibli")

	drawer.ready = false
	_, err = svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 1)
	require.ErrorIs(t, err, ErrMaterialNoKey)
	require.Len(t, repo.arts, 2)
}

func TestMaterialQueueDrawsAndStores(t *testing.T) {
	svc, repo, drawer, _ := newMaterialQueueFixture(t)
	ctx := context.Background()
	_, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 3)
	require.NoError(t, err)

	runQueue(svc)
	require.Len(t, drawer.calls, 3)
	require.True(t, strings.HasPrefix(drawer.calls[0], "1024x1536|high|Tall vertical"))

	for _, a := range repo.arts {
		require.Equal(t, MaterialArtworkSucceeded, a.Status)
		require.Equal(t, 1, a.Attempts)
		require.Equal(t, 64, a.Width)
		require.Equal(t, 96, a.Height)
		require.InDelta(t, 64*96/1e6, a.CostUSD, 1e-9)
		require.NotNil(t, a.FinishedAt)
	}
	require.Equal(t, "artworks/0/1.png", repo.arts[0].File)

	items, _, err := svc.ListArtworks(ctx, MaterialArtworkQuery{})
	require.NoError(t, err)
	link, err := url.Parse(items[0].URL)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(link.Path, materialFileRoute+"artworks/0/1.png"))
	path, ok := svc.OpenFile(strings.TrimPrefix(link.Path, materialFileRoute), link.Query().Get("e"), link.Query().Get("s"))
	require.True(t, ok)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, drawer.png, data)
	thumb, err := url.Parse(items[0].ThumbURL)
	require.NoError(t, err)
	_, ok = svc.OpenFile(strings.TrimPrefix(thumb.Path, materialFileRoute), thumb.Query().Get("e"), thumb.Query().Get("s"))
	require.True(t, ok)

	// A wrong signature, a 课本诗画 path and a path outside the store are refused.
	_, ok = svc.OpenFile("artworks/0/1.png", link.Query().Get("e"), "00")
	require.False(t, ok)
	_, ok = svc.OpenFile("works/1/1.png", link.Query().Get("e"), link.Query().Get("s"))
	require.False(t, ok)
	_, ok = svc.OpenFile("../secret.png", link.Query().Get("e"), link.Query().Get("s"))
	require.False(t, ok)
}

func TestMaterialQueueRetriesAndLimits(t *testing.T) {
	svc, repo, drawer, clock := newMaterialQueueFixture(t)
	ctx := context.Background()
	drawer.concurrency = 1
	drawer.results = []error{
		&shihuaDrawError{Status: http.StatusServiceUnavailable, Msg: "HTTP 503：busy"},
		&shihuaDrawError{Status: http.StatusBadRequest, Msg: "HTTP 400：bad prompt"},
	}
	_, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 2)
	require.NoError(t, err)

	require.True(t, drawOne(t, svc))
	first := repo.arts[0]
	require.Equal(t, MaterialArtworkQueued, first.Status, "503 is retried")
	require.Equal(t, "HTTP 503：busy", first.Error)
	require.Equal(t, clock.Add(30*time.Second), *first.NotBefore)

	require.True(t, drawOne(t, svc), "the first is not due yet, so the second goes")
	second := repo.arts[1]
	require.Equal(t, MaterialArtworkFailed, second.Status, "400 is not retried")
	require.NotNil(t, second.FinishedAt)

	require.False(t, drawOne(t, svc))
	*clock = clock.Add(31 * time.Second)
	require.True(t, drawOne(t, svc))
	require.Equal(t, MaterialArtworkSucceeded, repo.arts[0].Status)
	require.Equal(t, 2, repo.arts[0].Attempts)

	// Retry a failed one; then the daily limit stops the queue.
	a, err := svc.RetryArtwork(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, MaterialArtworkQueued, a.Status)
	_, err = svc.RetryArtwork(ctx, 1)
	require.Error(t, err, "a finished drawing cannot be retried")
	// The daily count is of drawings started today, not attempts: the retried one counts once.
	drawer.limit = 2
	v, err := svc.QueueView(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, v.StartedToday)
	require.Contains(t, v.Paused, "每日上限")
	runQueue(svc)
	require.Equal(t, MaterialArtworkQueued, repo.arts[1].Status)
	require.Len(t, drawer.calls, 3)

	n, err := svc.CancelArtworks(ctx, []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.Equal(t, MaterialArtworkCanceled, repo.arts[1].Status)
	_, err = svc.CancelArtworks(ctx, nil)
	require.ErrorIs(t, err, ErrMaterialInvalid)

	drawer.ready = false
	v, err = svc.QueueView(ctx)
	require.NoError(t, err)
	require.Contains(t, v.Paused, "API Key")
	runQueue(svc)
	require.Len(t, drawer.calls, 3)
}

func TestMaterialRedrawAndReview(t *testing.T) {
	svc, repo, _, _ := newMaterialQueueFixture(t)
	ctx := context.Background()
	_, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 1)
	require.NoError(t, err)

	_, err = svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: MaterialReviewApproved, Rating: 4})
	require.Error(t, err, "not drawn yet")
	require.True(t, drawOne(t, svc))

	a, err := svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: MaterialReviewApproved, Rating: 5, RejectReason: "ignored"})
	require.NoError(t, err)
	require.Equal(t, MaterialReviewApproved, a.ReviewStatus)
	require.Equal(t, 5, a.Rating)
	require.Empty(t, a.RejectReason)
	a, err = svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: MaterialReviewRejected, RejectReason: "  画面里有字 " + strings.Repeat("长", 300)})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(a.RejectReason, "画面里有字"))
	require.Equal(t, materialMaxRejectReason, len([]rune(a.RejectReason)))
	_, err = svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: "maybe"})
	require.ErrorIs(t, err, ErrMaterialInvalid)
	_, err = svc.ReviewArtwork(ctx, 1, MaterialReviewInput{ReviewStatus: MaterialReviewPending, Rating: 6})
	require.ErrorIs(t, err, ErrMaterialInvalid)
	_, err = svc.ReviewArtwork(ctx, 99, MaterialReviewInput{ReviewStatus: MaterialReviewPending})
	require.ErrorIs(t, err, ErrMaterialArtworkNotFound)

	r, err := svc.Redraw(ctx, 8, 1)
	require.NoError(t, err)
	require.Equal(t, int64(1), *r.ParentID)
	require.Equal(t, repo.arts[0].PromptFinal, r.PromptFinal)
	require.Equal(t, int64(8), *r.CreatedBy)
	require.Equal(t, MaterialReviewPending, r.ReviewStatus)

	// A term added since the first drawing blocks the redraw; the fixed no-text rule never does.
	repo.terms = []MaterialBlockTerm{
		{Term: "(?i)calligraphy", Kind: "text", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true},
	}
	_, err = svc.Redraw(ctx, 8, 1)
	require.NoError(t, err)
	repo.terms = append(repo.terms, MaterialBlockTerm{Term: "(?i)ink-wash", Kind: "artist", Match: "regex", Action: MaterialBlockActionBlock, AppliesTo: "prompt", Enabled: true})
	_, err = svc.Redraw(ctx, 8, 1)
	require.Error(t, err)
	_, err = svc.Redraw(ctx, 8, 99)
	require.ErrorIs(t, err, ErrMaterialArtworkNotFound)
}

func TestMaterialDrawingOff(t *testing.T) {
	svc := &MaterialService{repo: &materialQueueRepo{materialPromptRepo: &materialPromptRepo{materialMemRepo: newMaterialMemRepo()},
		now: time.Now}, now: time.Now}
	_, err := svc.DrawSegment(context.Background(), 1, 1, MaterialPromptInput{}, 1)
	require.ErrorIs(t, err, ErrMaterialNoKey)
	v, err := svc.QueueView(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, v.Paused)
	_, ok := svc.OpenFile("artworks/0/1.png", "1", "x")
	require.False(t, ok)
	svc.Start() // no-ops without a queue
	svc.Stop()
}

func TestMaterialDispatchConcurrency(t *testing.T) {
	svc, repo, drawer, _ := newMaterialQueueFixture(t)
	ctx := context.Background()
	drawer.gate = make(chan struct{})
	_, err := svc.DrawSegment(ctx, 7, 1, MaterialPromptInput{StyleID: 20}, 3)
	require.NoError(t, err)

	svc.dispatch() // two start and wait at the gate; the third stays queued
	require.Equal(t, 2, svc.queue.runningCount())
	stats, err := repo.ArtworkQueueStats(ctx, shihuaDayStart(svc.now()))
	require.NoError(t, err)
	require.Equal(t, MaterialQueueStats{Queued: 1, Running: 2, StartedToday: 2}, stats)
	v, err := svc.QueueView(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, v.Running)
	require.Empty(t, v.Paused)

	close(drawer.gate)
	svc.queue.wg.Wait()
	runQueue(svc)
	for _, a := range repo.arts {
		require.Equal(t, MaterialArtworkSucceeded, a.Status)
	}
}

func TestMaterialQueueStartStop(t *testing.T) {
	svc, repo, drawer, _ := newMaterialQueueFixture(t)
	svc.Start()
	_, err := svc.DrawSegment(context.Background(), 7, 1, MaterialPromptInput{StyleID: 20}, 1)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		a, _ := repo.GetArtwork(context.Background(), 1)
		return a != nil && a.Status == MaterialArtworkSucceeded
	}, 5*time.Second, 20*time.Millisecond, "the loop wakes on a new drawing")
	svc.Stop()
	require.Len(t, drawer.calls, 1)
}
