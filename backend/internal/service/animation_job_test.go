//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeAnimationRepo struct {
	mu     sync.Mutex
	nextID int64
	jobs   map[int64]*AnimationJob
	owners map[int64]int64
}

func newFakeAnimationRepo() *fakeAnimationRepo {
	return &fakeAnimationRepo{jobs: map[int64]*AnimationJob{}, owners: map[int64]int64{}}
}

func (r *fakeAnimationRepo) Create(_ context.Context, userID, _ int64, model string, meta json.RawMessage) (*AnimationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	job := &AnimationJob{ID: r.nextID, Status: AnimationJobPending, Model: model, Meta: meta, CreatedAt: time.Now()}
	r.jobs[job.ID], r.owners[job.ID] = job, userID
	copied := *job
	return &copied, nil
}

func (r *fakeAnimationRepo) CountActive(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for id, job := range r.jobs {
		if r.owners[id] == userID && job.Active() {
			n++
		}
	}
	return n, nil
}

func (r *fakeAnimationRepo) Get(_ context.Context, userID, id int64) (*AnimationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, ok := r.jobs[id]
	if !ok || r.owners[id] != userID {
		return nil, nil
	}
	copied := *job
	return &copied, nil
}

func (r *fakeAnimationRepo) List(_ context.Context, userID int64, _ int) ([]AnimationJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []AnimationJob{}
	for id, job := range r.jobs {
		if r.owners[id] == userID {
			copied := *job
			copied.SVG = ""
			out = append(out, copied)
		}
	}
	return out, nil
}

func (r *fakeAnimationRepo) MarkRunning(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job := r.jobs[id]; job != nil && job.Status == AnimationJobPending {
		job.Status = AnimationJobRunning
	}
	return nil
}

func (r *fakeAnimationRepo) Finish(_ context.Context, id int64, res AnimationJobResult) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[id]
	if job == nil || !job.Active() {
		return false, nil
	}
	job.Status, job.SVG, job.Error, job.Chars = res.Status, res.SVG, res.Error, res.Chars
	job.PromptTokens, job.CompletionTokens = res.PromptTokens, res.CompletionTokens
	return true, nil
}

func (r *fakeAnimationRepo) FailActive(_ context.Context, message string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, job := range r.jobs {
		if job.Active() {
			job.Status, job.Error = AnimationJobFailed, message
			n++
		}
	}
	return n, nil
}

func (r *fakeAnimationRepo) DeleteBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func animationInput() AnimationJobInput {
	return AnimationJobInput{Model: "gpt-5.5", Messages: []LearnMessage{{Role: "system", Content: "write svg"}, {Role: "user", Content: "三个圆点"}}, Meta: json.RawMessage(`{"logId":"a"}`)}
}

func TestAnimationJobSucceedsWithExtractedSVG(t *testing.T) {
	repo := newFakeAnimationRepo()
	var gotKey, gotModel string
	svc := newAnimationJobService(repo, func(_ context.Context, key, model string, messages []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		gotKey, gotModel = key, model
		require.Len(t, messages, 2)
		for _, part := range []string{"Here:\n```svg\n<svg viewBox=\"0 0 10 10\">", "<rect/></svg>\n```", "\nEnjoy"} {
			require.NoError(t, onDelta(part))
		}
		return &LearnTutorResult{PromptTokens: 12, CompletionTokens: 34}, nil
	})
	job, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk-user"}, animationInput())
	require.NoError(t, err)
	require.Equal(t, AnimationJobPending, job.Status)
	svc.Wait()

	done, err := svc.Get(context.Background(), 7, job.ID)
	require.NoError(t, err)
	require.Equal(t, AnimationJobSucceeded, done.Status)
	require.Equal(t, `<svg viewBox="0 0 10 10"><rect/></svg>`, done.SVG)
	require.Equal(t, 34, done.CompletionTokens)
	require.JSONEq(t, `{"logId":"a"}`, string(done.Meta))
	require.Equal(t, "sk-user", gotKey)
	require.Equal(t, "gpt-5.5", gotModel)

	// Another user cannot read it.
	_, err = svc.Get(context.Background(), 8, job.ID)
	require.ErrorIs(t, err, ErrAnimationJobNotFound)
}

func TestAnimationJobWithoutSVGFails(t *testing.T) {
	repo := newFakeAnimationRepo()
	svc := newAnimationJobService(repo, func(_ context.Context, _, _ string, _ []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		return &LearnTutorResult{}, onDelta("抱歉，我无法生成。")
	})
	job, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk"}, animationInput())
	require.NoError(t, err)
	svc.Wait()
	done, _ := svc.Get(context.Background(), 7, job.ID)
	require.Equal(t, AnimationJobFailed, done.Status)
	require.Contains(t, done.Error, "没有返回 SVG")
}

func TestAnimationJobGatewayErrorIsKept(t *testing.T) {
	repo := newFakeAnimationRepo()
	svc := newAnimationJobService(repo, func(context.Context, string, string, []LearnMessage, int, func(string) error) (*LearnTutorResult, error) {
		return nil, errLearnRunFailed("余额不足")
	})
	job, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk"}, animationInput())
	require.NoError(t, err)
	svc.Wait()
	done, _ := svc.Get(context.Background(), 7, job.ID)
	require.Equal(t, AnimationJobFailed, done.Status)
	require.Contains(t, done.Error, "余额不足")
}

func TestAnimationJobCancelStopsTheRun(t *testing.T) {
	repo := newFakeAnimationRepo()
	started := make(chan struct{})
	svc := newAnimationJobService(repo, func(ctx context.Context, _, _ string, _ []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		_ = onDelta("<svg ")
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	job, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk"}, animationInput())
	require.NoError(t, err)
	<-started

	running, err := svc.Get(context.Background(), 7, job.ID)
	require.NoError(t, err)
	require.Equal(t, 5, running.Chars, "progress comes from the live run")

	canceled, err := svc.Cancel(context.Background(), 7, job.ID)
	require.NoError(t, err)
	require.Equal(t, AnimationJobCanceled, canceled.Status)
	svc.Wait()
	final, _ := svc.Get(context.Background(), 7, job.ID)
	require.Equal(t, AnimationJobCanceled, final.Status, "the run's own finish must not overwrite the cancel")
}

func TestAnimationJobLimitsActiveJobsPerUser(t *testing.T) {
	repo := newFakeAnimationRepo()
	release := make(chan struct{})
	svc := newAnimationJobService(repo, func(ctx context.Context, _, _ string, _ []LearnMessage, _ int, _ func(string) error) (*LearnTutorResult, error) {
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, errors.New("stopped")
	})
	for i := 0; i < animationJobMaxActive; i++ {
		_, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk"}, animationInput())
		require.NoError(t, err)
	}
	_, err := svc.Create(context.Background(), 7, &APIKey{ID: 3, Key: "sk"}, animationInput())
	require.ErrorIs(t, err, ErrAnimationJobBusy)
	_, err = svc.Create(context.Background(), 9, &APIKey{ID: 4, Key: "sk2"}, animationInput())
	require.NoError(t, err, "other users are not affected")
	close(release)
	svc.Wait()
}

func TestAnimationJobValidation(t *testing.T) {
	svc := newAnimationJobService(newFakeAnimationRepo(), nil)
	key := &APIKey{ID: 1, Key: "sk"}
	cases := map[string]func(*AnimationJobInput){
		"no model":          func(in *AnimationJobInput) { in.Model = " " },
		"no messages":       func(in *AnimationJobInput) { in.Messages = nil },
		"bad role":          func(in *AnimationJobInput) { in.Messages[0].Role = "tool" },
		"last not user":     func(in *AnimationJobInput) { in.Messages = in.Messages[:1] },
		"meta not object":   func(in *AnimationJobInput) { in.Meta = json.RawMessage(`[1]`) },
		"conversation huge": func(in *AnimationJobInput) { in.Messages[0].Content = strings.Repeat("x", animationJobMaxInputBytes+1) },
	}
	for name, mutate := range cases {
		in := animationInput()
		mutate(&in)
		_, err := svc.Create(context.Background(), 7, key, in)
		require.Error(t, err, name)
	}
	_, err := svc.Create(context.Background(), 7, nil, animationInput())
	require.ErrorIs(t, err, ErrAnimationJobKey)
}

func TestAnimationJobRecoverInterrupted(t *testing.T) {
	repo := newFakeAnimationRepo()
	job, _ := repo.Create(context.Background(), 7, 1, "m", json.RawMessage(`{}`))
	svc := newAnimationJobService(repo, nil)
	svc.RecoverInterrupted(context.Background())
	got, _ := repo.Get(context.Background(), 7, job.ID)
	require.Equal(t, AnimationJobFailed, got.Status)
	require.Contains(t, got.Error, "服务器重启")
}

func TestExtractAnimationSVG(t *testing.T) {
	require.Equal(t, "<svg>a</svg>", extractAnimationSVG("x <svg>a</svg> y"))
	require.Equal(t, `<SVG viewBox="0 0 1 1"><svg>b</svg></SVG>`, extractAnimationSVG(`<svgfoo> <SVG viewBox="0 0 1 1"><svg>b</svg></SVG>`))
	require.Equal(t, "", extractAnimationSVG("no svg here"))
	require.Equal(t, "", extractAnimationSVG("</svg> <svg>"))
}
