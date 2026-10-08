package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AI 动画 (canvas): a text model writes an animated SVG, which takes 1–3 minutes. The canvas hands the
// conversation to this service, which calls the site's own gateway with the user's API key (so the
// gateway bills it like any other call) in a goroutine that outlives the browser tab, and stores the
// SVG for the canvas to pick up — on this device or another one.

const (
	AnimationJobPending   = "pending"
	AnimationJobRunning   = "running"
	AnimationJobSucceeded = "succeeded"
	AnimationJobFailed    = "failed"
	AnimationJobCanceled  = "canceled"

	animationJobMaxInputBytes  = 1 << 20 // the conversation, which carries the SVG being revised
	animationJobMaxMetaBytes   = 16 << 10
	animationJobMaxOutputChars = 600_000
	animationJobMaxMessages    = 24
	animationJobMaxActive      = 3  // per user
	animationJobMaxConcurrent  = 12 // site-wide; later jobs wait their turn
	animationJobTimeout        = 10 * time.Minute
	animationJobRetention      = 7 * 24 * time.Hour
	animationJobMaxTokens      = 32000 // reasoning included
	animationJobListLimit      = 30
)

var (
	ErrAnimationJobInvalid  = infraerrors.BadRequest("ANIMATION_JOB_INVALID", "动画请求格式不正确，请刷新画布后重试（Invalid animation request）")
	ErrAnimationJobTooLarge = infraerrors.BadRequest("ANIMATION_JOB_TOO_LARGE", "动画内容太大，请新建一个动画再试（Animation request too large）")
	ErrAnimationJobBusy     = infraerrors.TooManyRequests("ANIMATION_JOB_BUSY", "已有 3 个动画在生成，请等它们完成后再试（Too many animations in progress）")
	ErrAnimationJobNotFound = infraerrors.NotFound("ANIMATION_JOB_NOT_FOUND", "找不到这个动画任务，可能已超过 7 天被清理（Animation job not found）")
	ErrAnimationJobKey      = infraerrors.Unauthorized("ANIMATION_JOB_KEY", "请先在画布里连接 API Key（Connect an API key first）")
)

// Messages stored when a job ends without an SVG. Chinese first, English kept in parentheses.
const (
	animationErrNoSVG       = "模型没有返回 SVG 动画，请重试，或换一个更强的模型（如 GPT-5 / Claude）（The model returned no SVG）"
	animationErrTooLong     = "模型返回的内容太长（超过 60 万字），请把描述写简单一些再试（Answer too long）"
	animationErrTimeout     = "生成超过 10 分钟仍未完成，已停止，请重试（Timed out after 10 minutes）"
	animationErrInterrupted = "服务器重启，生成中断了，请重新生成（Interrupted by a server restart）"
)

type AnimationJob struct {
	ID               int64           `json:"id"`
	Status           string          `json:"status"`
	Model            string          `json:"model"`
	Meta             json.RawMessage `json:"meta"`
	SVG              string          `json:"svg,omitempty"`
	Error            string          `json:"error,omitempty"`
	Chars            int             `json:"chars"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	CreatedAt        time.Time       `json:"created_at"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
}

// Active reports whether the job may still produce a result.
func (j *AnimationJob) Active() bool {
	return j.Status == AnimationJobPending || j.Status == AnimationJobRunning
}

type AnimationJobResult struct {
	Status           string
	SVG              string
	Error            string
	Chars            int
	PromptTokens     int
	CompletionTokens int
}

type AnimationJobRepository interface {
	Create(ctx context.Context, userID, apiKeyID int64, model string, meta json.RawMessage) (*AnimationJob, error)
	CountActive(ctx context.Context, userID int64) (int, error)
	// Get returns nil when the job does not exist or belongs to someone else.
	Get(ctx context.Context, userID, id int64) (*AnimationJob, error)
	// List returns the user's newest jobs without their SVG.
	List(ctx context.Context, userID int64, limit int) ([]AnimationJob, error)
	MarkRunning(ctx context.Context, id int64) error
	// Finish records the outcome unless the job already ended (e.g. it was canceled first).
	Finish(ctx context.Context, id int64, res AnimationJobResult) (bool, error)
	// FailActive ends every pending / running job with message (used at startup).
	FailActive(ctx context.Context, message string) (int64, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

type AnimationJobInput struct {
	Model    string          `json:"model"`
	Messages []LearnMessage  `json:"messages"`
	Meta     json.RawMessage `json:"meta"`
}

type animationStreamFunc func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error)

type animationRun struct {
	userID int64
	cancel context.CancelFunc
	chars  atomic.Int64
}

type AnimationJobService struct {
	repo   AnimationJobRepository
	stream animationStreamFunc
	slots  chan struct{}
	now    func() time.Time

	mu   sync.Mutex
	runs map[int64]*animationRun
	wg   sync.WaitGroup
}

// NewAnimationJobService: gatewayURL is this server's own base URL (e.g. http://127.0.0.1:8080).
func NewAnimationJobService(repo AnimationJobRepository, gatewayURL string) *AnimationJobService {
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	// No client timeout: a run is bounded by its context (animationJobTimeout) and streams for minutes.
	client := &http.Client{}
	stream := func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
		return streamChatCompletion(ctx, client, gatewayURL, "hivegpt-canvas-animation/1", key, model, messages, maxTokens, onDelta)
	}
	return newAnimationJobService(repo, stream)
}

func newAnimationJobService(repo AnimationJobRepository, stream animationStreamFunc) *AnimationJobService {
	return &AnimationJobService{repo: repo, stream: stream, slots: make(chan struct{}, animationJobMaxConcurrent), now: time.Now, runs: map[int64]*animationRun{}}
}

// RecoverInterrupted fails the jobs a previous process left behind; their goroutines are gone.
func (s *AnimationJobService) RecoverInterrupted(ctx context.Context) {
	if n, err := s.repo.FailActive(ctx, animationErrInterrupted); err != nil {
		slog.Warn("animation jobs: failing interrupted jobs", "err", err)
	} else if n > 0 {
		slog.Info("animation jobs: marked interrupted jobs failed", "count", n)
	}
}

func validateAnimationInput(in *AnimationJobInput) error {
	in.Model = strings.TrimSpace(in.Model)
	if in.Model == "" || len(in.Model) > 100 || len(in.Messages) == 0 || len(in.Messages) > animationJobMaxMessages {
		return ErrAnimationJobInvalid
	}
	size := 0
	for i := range in.Messages {
		m := &in.Messages[i]
		if m.Role != "system" && m.Role != "user" && m.Role != "assistant" {
			return ErrAnimationJobInvalid
		}
		// Plain text turns only: no tool calls through this endpoint.
		m.ToolCalls, m.ToolCallID = nil, ""
		size += len(m.Content)
	}
	if strings.TrimSpace(in.Messages[len(in.Messages)-1].Content) == "" || in.Messages[len(in.Messages)-1].Role != "user" {
		return ErrAnimationJobInvalid
	}
	if size > animationJobMaxInputBytes {
		return ErrAnimationJobTooLarge
	}
	meta := bytes.TrimSpace(in.Meta)
	if len(meta) == 0 {
		meta = []byte("{}")
	}
	if len(meta) > animationJobMaxMetaBytes {
		return ErrAnimationJobTooLarge
	}
	if meta[0] != '{' || !json.Valid(meta) {
		return ErrAnimationJobInvalid
	}
	in.Meta = meta
	return nil
}

// Create records the job and starts it in the background with the caller's API key.
func (s *AnimationJobService) Create(ctx context.Context, userID int64, key *APIKey, in AnimationJobInput) (*AnimationJob, error) {
	if key == nil || key.Key == "" || userID <= 0 {
		return nil, ErrAnimationJobKey
	}
	if err := validateAnimationInput(&in); err != nil {
		return nil, err
	}
	active, err := s.repo.CountActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active >= animationJobMaxActive {
		return nil, ErrAnimationJobBusy
	}
	job, err := s.repo.Create(ctx, userID, key.ID, in.Model, in.Meta)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(context.Background(), animationJobTimeout)
	run := &animationRun{userID: userID, cancel: cancel}
	s.mu.Lock()
	s.runs[job.ID] = run
	s.mu.Unlock()
	s.wg.Add(1)
	go s.run(runCtx, job.ID, run, key.Key, in.Model, in.Messages)
	go s.cleanup()
	return job, nil
}

func (s *AnimationJobService) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := s.repo.DeleteBefore(ctx, s.now().Add(-animationJobRetention)); err != nil {
		slog.Warn("animation jobs: cleanup", "err", err)
	}
}

func (s *AnimationJobService) run(ctx context.Context, id int64, run *animationRun, key, model string, messages []LearnMessage) {
	defer s.wg.Done()
	defer func() {
		run.cancel()
		s.mu.Lock()
		delete(s.runs, id)
		s.mu.Unlock()
	}()
	finish := func(res AnimationJobResult) {
		// The job's own context may be done (timeout / cancel); the outcome must still be written.
		writeCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := s.repo.Finish(writeCtx, id, res); err != nil {
			slog.Warn("animation jobs: finish", "id", id, "err", err)
		}
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-ctx.Done():
		finish(animationFailure(ctx, nil, 0))
		return
	}
	if err := s.repo.MarkRunning(ctx, id); err != nil {
		slog.Warn("animation jobs: mark running", "id", id, "err", err)
	}

	var answer strings.Builder
	errTooLong := errors.New("too long")
	res, err := s.stream(ctx, key, model, messages, animationJobMaxTokens, func(delta string) error {
		_, _ = answer.WriteString(delta)
		run.chars.Store(int64(answer.Len()))
		if answer.Len() > animationJobMaxOutputChars {
			return errTooLong
		}
		return nil
	})
	chars := answer.Len()
	if errors.Is(err, errTooLong) {
		finish(AnimationJobResult{Status: AnimationJobFailed, Error: animationErrTooLong, Chars: chars})
		return
	}
	if err != nil {
		finish(animationFailure(ctx, err, chars))
		return
	}
	svg := extractAnimationSVG(answer.String())
	out := AnimationJobResult{Status: AnimationJobSucceeded, SVG: svg, Chars: chars}
	if res != nil {
		out.PromptTokens, out.CompletionTokens = res.PromptTokens, res.CompletionTokens
	}
	if svg == "" {
		out.Status, out.Error = AnimationJobFailed, animationErrNoSVG
	}
	finish(out)
}

func animationFailure(ctx context.Context, err error, chars int) AnimationJobResult {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return AnimationJobResult{Status: AnimationJobFailed, Error: animationErrTimeout, Chars: chars}
	case errors.Is(ctx.Err(), context.Canceled):
		return AnimationJobResult{Status: AnimationJobCanceled, Chars: chars}
	case err != nil:
		return AnimationJobResult{Status: AnimationJobFailed, Error: infraerrors.Message(err), Chars: chars}
	}
	return AnimationJobResult{Status: AnimationJobFailed, Error: animationErrNoSVG, Chars: chars}
}

// extractAnimationSVG keeps <svg>…</svg> from the answer (models may add a fence or a sentence). The
// canvas sanitizes it before showing it.
func extractAnimationSVG(answer string) string {
	loc := animationSVGStart.FindStringIndex(answer)
	end := strings.LastIndex(strings.ToLower(answer), "</svg>")
	if loc == nil || end < loc[0] {
		return ""
	}
	return answer[loc[0] : end+len("</svg>")]
}

var animationSVGStart = regexp.MustCompile(`(?i)<svg[\s>]`)

// Get returns one of the user's jobs, with live progress while it runs.
func (s *AnimationJobService) Get(ctx context.Context, userID, id int64) (*AnimationJob, error) {
	job, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrAnimationJobNotFound
	}
	s.withProgress(job)
	return job, nil
}

// List returns the user's recent jobs (newest first, without SVGs) so a canvas can pick up results
// that finished while it was closed.
func (s *AnimationJobService) List(ctx context.Context, userID int64) ([]AnimationJob, error) {
	jobs, err := s.repo.List(ctx, userID, animationJobListLimit)
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		s.withProgress(&jobs[i])
	}
	return jobs, nil
}

func (s *AnimationJobService) withProgress(job *AnimationJob) {
	if !job.Active() {
		return
	}
	s.mu.Lock()
	run := s.runs[job.ID]
	s.mu.Unlock()
	if run != nil {
		job.Chars = int(run.chars.Load())
	}
}

// Cancel stops one of the user's jobs. Ended jobs are returned unchanged.
func (s *AnimationJobService) Cancel(ctx context.Context, userID, id int64) (*AnimationJob, error) {
	job, err := s.Get(ctx, userID, id)
	if err != nil || !job.Active() {
		return job, err
	}
	s.mu.Lock()
	run := s.runs[id]
	s.mu.Unlock()
	if run != nil && run.userID == userID {
		run.cancel()
	}
	if _, err := s.repo.Finish(ctx, id, AnimationJobResult{Status: AnimationJobCanceled, Chars: job.Chars}); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Wait blocks until every running job has ended (tests).
func (s *AnimationJobService) Wait() { s.wg.Wait() }
