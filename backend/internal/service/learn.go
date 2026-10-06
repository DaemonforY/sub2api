package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AI 学习 (/learn, docs/AI_LEARN_PLAN.md): lesson progress per user, and running the examples in a
// lesson. A run is one chat completion through this site's own gateway with the admin's learning
// key (so routing, usage logs and costs work as for any key); the server picks the model and caps
// the output, and the request may only carry messages plus an optional JSON schema or tools.
// Each user gets a number of free runs a day, under a site-wide daily cap.

const (
	settingLearnRunEnabled = "learn_run_enabled"
	settingLearnRunModel   = "learn_run_model"
	settingLearnFreeRuns   = "learn_free_runs_per_day"
	settingLearnDailyCap   = "learn_run_daily_cap"
	settingLearnAPIKey     = "learn_run_api_key_enc"
	settingLearnTutorFree  = "learn_tutor_free_per_day"
	settingLearnInterviews = "learn_interviews_per_day"

	learnDefaultFreeRuns  = 20
	learnDefaultDailyCap  = 1000
	learnDefaultTutorFree = 10
	learnDefaultIntervws  = 3
	learnMaxOutputTokens  = 800
	learnRunTimeout       = 90 * time.Second
	learnMaxMessages      = 16
	learnMaxMessageChars  = 4000
	learnMaxTotalChars    = 12000
	learnMaxSchemaBytes   = 4096
	learnMaxToolsBytes    = 6144
	learnMaxProgressBatch = 100
)

var (
	ErrLearnRunDisabled    = infraerrors.ServiceUnavailable("LEARN_RUN_DISABLED", "在线运行暂未开放，可以复制代码用自己的 Key 运行（Running examples is not available yet）")
	ErrLearnOwnKeyRequired = infraerrors.BadRequest("LEARN_OWN_KEY_REQUIRED", "在线运行使用你自己的 HiveGPT Key，按实际用量计费，请先选择一个 Key（Choose one of your API keys to run）")
	ErrLearnRunQuota       = infraerrors.TooManyRequests("LEARN_RUN_QUOTA", "今天的免费运行次数用完了，明天再来，或者复制代码用自己的 Key 运行（Daily free runs used up）")
	ErrLearnRunBusy        = infraerrors.TooManyRequests("LEARN_RUN_BUSY", "今天的在线运行名额已用完，可以复制代码用自己的 Key 运行（Daily run capacity reached）")
	ErrLearnLessonID       = infraerrors.BadRequest("LEARN_LESSON_INVALID", "课时编号不正确（Invalid lesson）")
)

func errLearnRunInvalid(what string) error {
	return infraerrors.BadRequest("LEARN_RUN_INVALID", fmt.Sprintf("运行内容不符合要求：%s（Invalid run）", what))
}

func errLearnRunFailed(detail string) error {
	return infraerrors.New(http.StatusBadGateway, "LEARN_RUN_FAILED", fmt.Sprintf("运行失败：%s，请稍后再试（Run failed）", detail))
}

// lesson IDs: track letter + number ("a1", "b12").
var learnLessonRe = regexp.MustCompile(`^[a-z][0-9]{1,2}$`)

// learnDayZone: free runs reset at midnight Beijing time.
var learnDayZone = time.FixedZone("CST", 8*3600)

type LearnRepository interface {
	Progress(ctx context.Context, userID int64) (map[string]time.Time, error)
	MarkDone(ctx context.Context, userID int64, lessonIDs []string) error
	// CountRuns counts runs not failed since the time (userID 0: everyone).
	// Only free calls (no key_id) count; kind "" counts every kind.
	CountRuns(ctx context.Context, userID int64, kind string, since time.Time) (int, error)
	// StartRun records a call (kind run | tutor | interview; keyID: the learner's own key, else 0).
	StartRun(ctx context.Context, userID int64, lessonID, kind string, keyID int64) (int64, error)
	FinishRun(ctx context.Context, runID int64, status string, latencyMs, promptTokens, completionTokens int) error
	Stats(ctx context.Context, since time.Time) (*LearnStats, error)

	QuizResults(ctx context.Context, userID int64) (map[string]LearnQuizResult, error)
	// SaveQuizResult counts an attempt and keeps the best score; returns the stored result.
	SaveQuizResult(ctx context.Context, userID int64, lessonID string, correct, total int) (LearnQuizResult, error)
	Checkpoints(ctx context.Context, userID int64) (map[string]time.Time, error)
	PassCheckpoint(ctx context.Context, userID int64, checkpointID string) error

	// CreateCertificate stores c unless the learner already holds a live one for the track.
	CreateCertificate(ctx context.Context, c *LearnCertificate) error
	CertificateByCode(ctx context.Context, code string) (*LearnCertificate, error)
	CertificatesByUser(ctx context.Context, userID int64) ([]LearnCertificate, error)
	ListCertificates(ctx context.Context, limit, offset int) ([]LearnCertificate, int, error)
	SetCertificateRevoked(ctx context.Context, code string, revoked bool) error

	CreateInterview(ctx context.Context, iv *LearnInterview) error
	GetInterview(ctx context.Context, id int64) (*LearnInterview, error)
	// SaveInterview stores the answers, status, score and finish time.
	SaveInterview(ctx context.Context, iv *LearnInterview) error
	ListInterviews(ctx context.Context, userID int64, limit int) ([]LearnInterview, error)
	// CountInterviews counts free interviews started since the time.
	CountInterviews(ctx context.Context, userID int64, since time.Time) (int, error)
	// BestInterviewScores: the best finished score per topic.
	BestInterviewScores(ctx context.Context, userID int64) (map[string]int, error)

	// SetShowcase shows / hides the learner's live certificate of a track on the wall.
	SetShowcase(ctx context.Context, userID int64, track string, on bool) error
	SetShowcaseHidden(ctx context.Context, code string, hidden bool) error
	// Showcase lists wall entries (track "" = all), newest first.
	Showcase(ctx context.Context, track string, limit int) ([]LearnCertificate, error)
	// Insights for the admin page.
	TrackProgress(ctx context.Context) ([]LearnTrackProgress, error)
	CertificateCounts(ctx context.Context) (map[string]int, error)
	Daily(ctx context.Context, since time.Time) ([]LearnDay, error)
	QuizStats(ctx context.Context) (map[string]LearnQuizStat, error)
}

type learnSettingStore interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, values map[string]string) error
}

// LearnSettings (admin). APIKey is write-only: reads report APIKeySet.
type LearnSettings struct {
	RunEnabled     bool   `json:"run_enabled"`
	Model          string `json:"model"`
	FreeRunsPerDay int    `json:"free_runs_per_day"`
	DailyCap       int    `json:"daily_cap"`
	TutorFree      int    `json:"tutor_free_per_day"`
	InterviewsFree int    `json:"interviews_per_day"`
	APIKeySet      bool   `json:"api_key_set"`
	APIKey         string `json:"api_key,omitempty"`
}

// LearnConfig is public: whether runs work and how many are free.
type LearnConfig struct {
	RunEnabled     bool   `json:"run_enabled"`
	FreeRunsPerDay int    `json:"free_runs_per_day"`
	TutorFree      int    `json:"tutor_free_per_day"`
	InterviewsFree int    `json:"interviews_per_day"`
	Model          string `json:"model"`
	// OwnKeyOnly: no site learning key, so every run uses one of the learner's own keys.
	OwnKeyOnly bool `json:"own_key_only"`
}

// LearnMe is the signed-in learner's state.
type LearnMe struct {
	Completed      map[string]time.Time       `json:"completed"`
	RunsLeft       int                        `json:"runs_left"`
	TutorLeft      int                        `json:"tutor_left"`
	InterviewsLeft int                        `json:"interviews_left"`
	Quizzes        map[string]LearnQuizResult `json:"quizzes"`
	Checkpoints    map[string]time.Time       `json:"checkpoints"`
	Certificates   []LearnCertificate         `json:"certificates"`
	// Usage today: free runs / tutor questions / interviews used.
	RunsToday       int `json:"runs_today"`
	TutorToday      int `json:"tutor_today"`
	InterviewsToday int `json:"interviews_today"`
}

// LearnMessage is one chat message of an example (assistant tool calls and tool results too).
type LearnMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

// LearnRunInput is what a lesson's example may send.
type LearnRunInput struct {
	Lesson         string          `json:"lesson"`
	Messages       []LearnMessage  `json:"messages"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Tools          json.RawMessage `json:"tools,omitempty"`
	// KeyID: run on the learner's own key (billed as usual) instead of a free run.
	KeyID int64 `json:"key_id,omitempty"`
}

type LearnRunResult struct {
	Content          string          `json:"content"`
	ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
	FinishReason     string          `json:"finish_reason"`
	Model            string          `json:"model"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	LatencyMs        int             `json:"latency_ms"`
	RunsLeft         int             `json:"runs_left"`
	OwnKey           bool            `json:"own_key,omitempty"`
}

type LearnLessonStat struct {
	LessonID  string `json:"lesson_id"`
	Completed int    `json:"completed"`
	Runs      int    `json:"runs"`
}

type LearnStats struct {
	Certificates  int               `json:"certificates"`
	QuizPassed    int               `json:"quiz_passed"`
	Tutor7d       int               `json:"tutor_7d"`
	Interviews7d  int               `json:"interviews_7d"`
	OwnKeyRuns7d  int               `json:"own_key_runs_7d"`
	Learners      int               `json:"learners"`
	RunsToday     int               `json:"runs_today"`
	Runs7d        int               `json:"runs_7d"`
	FailedRuns7d  int               `json:"failed_runs_7d"`
	Tokens7d      int               `json:"tokens_7d"`
	Lessons       []LearnLessonStat `json:"lessons"`
	LearnersToday int               `json:"learners_today"`
}

type LearnService struct {
	repo       LearnRepository
	settings   learnSettingStore
	encryptor  SecretEncryptor
	gatewayURL string
	client     *http.Client
	now        func() time.Time
	catalog    *learnCatalog
	sources    LearnSources
	// lessonText is a lesson page's title and text (for the tutor); nil without the built site.
	lessonText func(lessonID string) (title, text string)
	showcase   learnShowcaseCache
}

// NewLearnService: gatewayURL is this server's own base URL (e.g. http://127.0.0.1:8080).
func NewLearnService(repo LearnRepository, settings learnSettingStore, encryptor SecretEncryptor, gatewayURL string, sources LearnSources) *LearnService {
	return &LearnService{repo: repo, settings: settings, encryptor: encryptor, gatewayURL: strings.TrimRight(gatewayURL, "/"),
		client: &http.Client{Timeout: learnRunTimeout}, now: time.Now, catalog: mustLearnCatalog(), sources: sources}
}

// SetLessonText gives the tutor the lesson pages (set once the built site is loaded).
func (s *LearnService) SetLessonText(fn func(lessonID string) (title, text string)) {
	s.lessonText = fn
}

func learnDayStart(t time.Time) time.Time {
	local := t.In(learnDayZone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, learnDayZone)
}

func (s *LearnService) loadSettings(ctx context.Context) (LearnSettings, string) {
	out := LearnSettings{FreeRunsPerDay: learnDefaultFreeRuns, DailyCap: learnDefaultDailyCap, TutorFree: learnDefaultTutorFree, InterviewsFree: learnDefaultIntervws}
	if s.settings == nil {
		return out, ""
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingLearnRunEnabled, settingLearnRunModel, settingLearnFreeRuns, settingLearnDailyCap, settingLearnAPIKey, settingLearnTutorFree, settingLearnInterviews})
	if err != nil {
		return out, ""
	}
	out.RunEnabled = values[settingLearnRunEnabled] == "true"
	out.Model = strings.TrimSpace(values[settingLearnRunModel])
	if n, err := strconv.Atoi(values[settingLearnFreeRuns]); err == nil && n >= 0 {
		out.FreeRunsPerDay = n
	}
	if n, err := strconv.Atoi(values[settingLearnDailyCap]); err == nil && n >= 0 {
		out.DailyCap = n
	}
	if n, err := strconv.Atoi(values[settingLearnTutorFree]); err == nil && n >= 0 {
		out.TutorFree = n
	}
	if n, err := strconv.Atoi(values[settingLearnInterviews]); err == nil && n >= 0 {
		out.InterviewsFree = n
	}
	key := ""
	if enc := values[settingLearnAPIKey]; enc != "" && s.encryptor != nil {
		if plain, err := s.encryptor.Decrypt(enc); err == nil {
			key = plain
		}
	}
	out.APIKeySet = key != ""
	return out, key
}

// freeOnSiteKey: free runs, tutor questions and interviews are paid with the site's learning key.
// Without one, every call runs on the learner's own key (own-key-only mode), so nothing is free.
func freeOnSiteKey(st LearnSettings, siteKey string) LearnSettings {
	if siteKey == "" {
		st.FreeRunsPerDay, st.TutorFree, st.InterviewsFree = 0, 0, 0
	}
	return st
}

// Settings (admin; the key itself is never returned).
func (s *LearnService) Settings(ctx context.Context) LearnSettings {
	out, _ := s.loadSettings(ctx)
	return out
}

// SaveSettings stores the run settings; an empty APIKey keeps the stored one.
func (s *LearnService) SaveSettings(ctx context.Context, in LearnSettings) (LearnSettings, error) {
	if in.FreeRunsPerDay < 0 || in.FreeRunsPerDay > 500 || in.DailyCap < 0 || in.DailyCap > 100000 ||
		in.TutorFree < 0 || in.TutorFree > 500 || in.InterviewsFree < 0 || in.InterviewsFree > 50 {
		return LearnSettings{}, errLearnRunInvalid("次数超出范围")
	}
	model := strings.TrimSpace(in.Model)
	if len(model) > 100 {
		return LearnSettings{}, errLearnRunInvalid("模型名太长")
	}
	if in.RunEnabled && model == "" {
		return LearnSettings{}, errLearnRunInvalid("开启前请填写模型")
	}
	values := map[string]string{
		settingLearnRunEnabled: strconv.FormatBool(in.RunEnabled),
		settingLearnRunModel:   model,
		settingLearnFreeRuns:   strconv.Itoa(in.FreeRunsPerDay),
		settingLearnDailyCap:   strconv.Itoa(in.DailyCap),
		settingLearnTutorFree:  strconv.Itoa(in.TutorFree),
		settingLearnInterviews: strconv.Itoa(in.InterviewsFree),
	}
	if key := strings.TrimSpace(in.APIKey); key != "" {
		if s.encryptor == nil {
			return LearnSettings{}, errLearnRunInvalid("服务端未配置加密")
		}
		enc, err := s.encryptor.Encrypt(key)
		if err != nil {
			return LearnSettings{}, err
		}
		values[settingLearnAPIKey] = enc
	}
	if s.settings != nil {
		if err := s.settings.SetMultiple(ctx, values); err != nil {
			return LearnSettings{}, err
		}
	}
	out, _ := s.loadSettings(ctx)
	return out, nil
}

// Config is public.
func (s *LearnService) Config(ctx context.Context) LearnConfig {
	st, key := s.loadSettings(ctx)
	st = freeOnSiteKey(st, key)
	return LearnConfig{RunEnabled: st.RunEnabled && st.Model != "", FreeRunsPerDay: st.FreeRunsPerDay,
		TutorFree: st.TutorFree, InterviewsFree: st.InterviewsFree, Model: st.Model, OwnKeyOnly: key == ""}
}

// Me: progress, quiz results, checkpoints, certificates and what is left free today.
func (s *LearnService) Me(ctx context.Context, userID int64) (*LearnMe, error) {
	out := &LearnMe{}
	var err error
	if out.Completed, err = s.repo.Progress(ctx, userID); err != nil {
		return nil, err
	}
	if out.Quizzes, err = s.repo.QuizResults(ctx, userID); err != nil {
		return nil, err
	}
	if out.Checkpoints, err = s.repo.Checkpoints(ctx, userID); err != nil {
		return nil, err
	}
	if out.Certificates, err = s.repo.CertificatesByUser(ctx, userID); err != nil {
		return nil, err
	}
	live := out.Certificates[:0]
	for _, c := range out.Certificates {
		if c.RevokedAt == nil {
			if t := s.catalog.track(c.Track); t != nil {
				c.TrackTitle = t.Title
			}
			live = append(live, c)
		}
	}
	out.Certificates = live
	st := freeOnSiteKey(s.loadSettings(ctx))
	since := learnDayStart(s.now())
	if out.RunsToday, err = s.repo.CountRuns(ctx, userID, learnKindRun, since); err != nil {
		return nil, err
	}
	if out.TutorToday, err = s.repo.CountRuns(ctx, userID, learnKindTutor, since); err != nil {
		return nil, err
	}
	if out.InterviewsToday, err = s.repo.CountInterviews(ctx, userID, since); err != nil {
		return nil, err
	}
	out.RunsLeft = max(st.FreeRunsPerDay-out.RunsToday, 0)
	out.TutorLeft = max(st.TutorFree-out.TutorToday, 0)
	out.InterviewsLeft = max(st.InterviewsFree-out.InterviewsToday, 0)
	return out, nil
}

// MarkDone records completed lessons (also used to merge progress kept in the browser).
func (s *LearnService) MarkDone(ctx context.Context, userID int64, lessonIDs []string) (map[string]time.Time, error) {
	if len(lessonIDs) == 0 || len(lessonIDs) > learnMaxProgressBatch {
		return nil, ErrLearnLessonID
	}
	seen := map[string]bool{}
	ids := make([]string, 0, len(lessonIDs))
	for _, id := range lessonIDs {
		id = strings.TrimSpace(id)
		if !learnLessonRe.MatchString(id) {
			return nil, ErrLearnLessonID
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if err := s.repo.MarkDone(ctx, userID, ids); err != nil {
		return nil, err
	}
	return s.repo.Progress(ctx, userID)
}

const (
	learnKindRun       = "run"
	learnKindTutor     = "tutor"
	learnKindInterview = "interview"
)

func validateLearnRun(in *LearnRunInput) error {
	if !learnLessonRe.MatchString(in.Lesson) {
		return ErrLearnLessonID
	}
	if len(in.Messages) == 0 || len(in.Messages) > learnMaxMessages {
		return errLearnRunInvalid("消息数量")
	}
	total := 0
	for _, m := range in.Messages {
		n := utf8.RuneCountInString(m.Content)
		switch m.Role {
		case "system", "user":
			if n == 0 {
				return errLearnRunInvalid("消息不能为空")
			}
		case "assistant":
			// An assistant turn that only calls tools has no text.
			if n == 0 && len(m.ToolCalls) == 0 {
				return errLearnRunInvalid("消息不能为空")
			}
			if len(m.ToolCalls) > learnMaxToolsBytes {
				return errLearnRunInvalid("工具调用太长")
			}
		case "tool":
			if m.ToolCallID == "" || len(m.ToolCallID) > 100 {
				return errLearnRunInvalid("工具结果缺少 tool_call_id")
			}
		default:
			return errLearnRunInvalid("消息角色")
		}
		if m.Role != "assistant" && len(m.ToolCalls) > 0 {
			return errLearnRunInvalid("只有助手消息可以带工具调用")
		}
		if n > learnMaxMessageChars {
			return errLearnRunInvalid(fmt.Sprintf("每条消息最多 %d 字", learnMaxMessageChars))
		}
		total += n + len(m.ToolCalls)
	}
	if total > learnMaxTotalChars {
		return errLearnRunInvalid("内容太长")
	}
	if last := in.Messages[len(in.Messages)-1].Role; last != "user" && last != "tool" {
		return errLearnRunInvalid("最后一条须是用户消息或工具结果")
	}
	if len(in.ResponseFormat) > 0 {
		if len(in.ResponseFormat) > learnMaxSchemaBytes {
			return errLearnRunInvalid("JSON Schema 太大")
		}
		var rf struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(in.ResponseFormat, &rf) != nil || (rf.Type != "json_object" && rf.Type != "json_schema") {
			return errLearnRunInvalid("response_format")
		}
	}
	if len(in.Tools) > 0 {
		if len(in.Tools) > learnMaxToolsBytes {
			return errLearnRunInvalid("工具定义太大")
		}
		var tools []struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(in.Tools, &tools) != nil || len(tools) == 0 || len(tools) > 4 {
			return errLearnRunInvalid("tools")
		}
		for _, t := range tools {
			if t.Type != "function" {
				return errLearnRunInvalid("tools 只支持 function")
			}
		}
	}
	return nil
}

// learnCall is a model call that has been allowed: the key to use and what is left free afterwards.
type learnCall struct {
	key   string
	keyID int64
	left  int
	runID int64
}

// acquire allows one call of kind: on the learner's own key (keyID > 0), or as a free call within
// the learner's daily allowance (free; err) and the site-wide cap. It records the call as pending.
func (s *LearnService) acquire(ctx context.Context, userID int64, lesson, kind string, keyID int64, free int, quota error) (*learnCall, LearnSettings, error) {
	st, learnKey := s.loadSettings(ctx)
	if !st.RunEnabled || st.Model == "" {
		return nil, st, ErrLearnRunDisabled
	}
	if keyID == 0 && learnKey == "" {
		return nil, st, ErrLearnOwnKeyRequired
	}
	st = freeOnSiteKey(st, learnKey)
	call := &learnCall{key: learnKey}
	since := learnDayStart(s.now())
	if free < 0 {
		free = st.FreeRunsPerDay
		if kind == learnKindTutor {
			free = st.TutorFree
		}
	}
	if keyID > 0 {
		k, err := s.ownKey(ctx, userID, keyID)
		if err != nil {
			return nil, st, err
		}
		call.key, call.keyID = k.Key, k.ID
	} else if quota != nil {
		used, err := s.repo.CountRuns(ctx, userID, kind, since)
		if err != nil {
			return nil, st, err
		}
		if used >= free {
			return nil, st, quota
		}
		call.left = free - used - 1
	}
	if call.keyID == 0 && st.DailyCap > 0 {
		all, err := s.repo.CountRuns(ctx, 0, "", since)
		if err != nil {
			return nil, st, err
		}
		if all >= st.DailyCap {
			return nil, st, ErrLearnRunBusy
		}
	}
	if call.keyID > 0 && quota != nil {
		used, err := s.repo.CountRuns(ctx, userID, kind, since)
		if err != nil {
			return nil, st, err
		}
		call.left = max(free-used, 0)
	}
	runID, err := s.repo.StartRun(ctx, userID, lesson, kind, call.keyID)
	if err != nil {
		return nil, st, err
	}
	call.runID = runID
	return call, st, nil
}

func (s *LearnService) finish(ctx context.Context, call *learnCall, started time.Time, err error, promptTokens, completionTokens int) int {
	latency := int(s.now().Sub(started) / time.Millisecond)
	status := "ok"
	if err != nil {
		status = "failed"
	}
	_ = s.repo.FinishRun(context.WithoutCancel(ctx), call.runID, status, latency, promptTokens, completionTokens)
	return latency
}

// Run runs a lesson example for userID.
func (s *LearnService) Run(ctx context.Context, userID int64, in LearnRunInput) (*LearnRunResult, error) {
	if err := validateLearnRun(&in); err != nil {
		if st, _ := s.loadSettings(ctx); !st.RunEnabled || st.Model == "" {
			return nil, ErrLearnRunDisabled
		}
		return nil, err
	}
	call, st, err := s.acquire(ctx, userID, in.Lesson, learnKindRun, in.KeyID, -1, ErrLearnRunQuota)
	if err != nil {
		return nil, err
	}
	started := s.now()
	result, callErr := s.callGateway(ctx, call.key, st.Model, in)
	if callErr != nil {
		s.finish(ctx, call, started, callErr, 0, 0)
		return nil, callErr
	}
	result.LatencyMs = s.finish(ctx, call, started, nil, result.PromptTokens, result.CompletionTokens)
	result.RunsLeft = call.left
	result.OwnKey = call.keyID > 0
	return result, nil
}

func (s *LearnService) callGateway(ctx context.Context, key, model string, in LearnRunInput) (*LearnRunResult, error) {
	body := map[string]any{
		"model":                 model,
		"messages":              in.Messages,
		"max_completion_tokens": learnMaxOutputTokens,
		"stream":                false,
	}
	if len(in.ResponseFormat) > 0 {
		body["response_format"] = in.ResponseFormat
	}
	if len(in.Tools) > 0 {
		body["tools"] = in.Tools
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gatewayURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hivegpt-learn/1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errLearnRunFailed("连接模型服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, errLearnRunFailed("读取结果失败")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content   string          `json:"content"`
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return nil, errLearnRunFailed("返回格式无法识别")
	}
	choice := out.Choices[0]
	res := &LearnRunResult{Content: choice.Message.Content, FinishReason: choice.FinishReason, Model: out.Model,
		PromptTokens: out.Usage.PromptTokens, CompletionTokens: out.Usage.CompletionTokens}
	if len(choice.Message.ToolCalls) > 0 && string(choice.Message.ToolCalls) != "null" {
		res.ToolCalls = choice.Message.ToolCalls
	}
	return res, nil
}

// upstreamErrorText picks the message of an OpenAI-style error body.
func upstreamErrorText(raw []byte, status int) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	msg := ""
	if json.Unmarshal(raw, &e) == nil {
		msg = firstNonEmpty(e.Error.Message, e.Message)
	}
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	if r := []rune(msg); len(r) > 200 {
		msg = string(r[:200]) + "…"
	}
	return msg
}

// Stats (admin): learners, runs and per-lesson completions.
func (s *LearnService) Stats(ctx context.Context) (*LearnStats, error) {
	return s.repo.Stats(ctx, learnDayStart(s.now()))
}
