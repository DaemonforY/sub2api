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

	learnDefaultFreeRuns  = 20
	learnDefaultDailyCap  = 1000
	learnMaxOutputTokens  = 800
	learnRunTimeout       = 90 * time.Second
	learnMaxMessages      = 12
	learnMaxMessageChars  = 4000
	learnMaxTotalChars    = 12000
	learnMaxSchemaBytes   = 4096
	learnMaxToolsBytes    = 6144
	learnMaxProgressBatch = 100
)

var (
	ErrLearnRunDisabled = infraerrors.ServiceUnavailable("LEARN_RUN_DISABLED", "在线运行暂未开放，可以复制代码用自己的 Key 运行（Running examples is not available yet）")
	ErrLearnRunQuota    = infraerrors.TooManyRequests("LEARN_RUN_QUOTA", "今天的免费运行次数用完了，明天再来，或者复制代码用自己的 Key 运行（Daily free runs used up）")
	ErrLearnRunBusy     = infraerrors.TooManyRequests("LEARN_RUN_BUSY", "今天的在线运行名额已用完，可以复制代码用自己的 Key 运行（Daily run capacity reached）")
	ErrLearnLessonID    = infraerrors.BadRequest("LEARN_LESSON_INVALID", "课时编号不正确（Invalid lesson）")
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
	CountRuns(ctx context.Context, userID int64, since time.Time) (int, error)
	StartRun(ctx context.Context, userID int64, lessonID string) (int64, error)
	FinishRun(ctx context.Context, runID int64, status string, latencyMs, promptTokens, completionTokens int) error
	Stats(ctx context.Context, since time.Time) (*LearnStats, error)
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
	APIKeySet      bool   `json:"api_key_set"`
	APIKey         string `json:"api_key,omitempty"`
}

// LearnConfig is public: whether runs work and how many are free.
type LearnConfig struct {
	RunEnabled     bool   `json:"run_enabled"`
	FreeRunsPerDay int    `json:"free_runs_per_day"`
	Model          string `json:"model"`
}

// LearnMe is the signed-in learner's state.
type LearnMe struct {
	Completed map[string]time.Time `json:"completed"`
	RunsLeft  int                  `json:"runs_left"`
}

type LearnMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LearnRunInput is what a lesson's example may send.
type LearnRunInput struct {
	Lesson         string          `json:"lesson"`
	Messages       []LearnMessage  `json:"messages"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Tools          json.RawMessage `json:"tools,omitempty"`
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
}

type LearnLessonStat struct {
	LessonID  string `json:"lesson_id"`
	Completed int    `json:"completed"`
	Runs      int    `json:"runs"`
}

type LearnStats struct {
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
}

// NewLearnService: gatewayURL is this server's own base URL (e.g. http://127.0.0.1:8080).
func NewLearnService(repo LearnRepository, settings learnSettingStore, encryptor SecretEncryptor, gatewayURL string) *LearnService {
	return &LearnService{repo: repo, settings: settings, encryptor: encryptor, gatewayURL: strings.TrimRight(gatewayURL, "/"),
		client: &http.Client{Timeout: learnRunTimeout}, now: time.Now}
}

func learnDayStart(t time.Time) time.Time {
	local := t.In(learnDayZone)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, learnDayZone)
}

func (s *LearnService) loadSettings(ctx context.Context) (LearnSettings, string) {
	out := LearnSettings{FreeRunsPerDay: learnDefaultFreeRuns, DailyCap: learnDefaultDailyCap}
	if s.settings == nil {
		return out, ""
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingLearnRunEnabled, settingLearnRunModel, settingLearnFreeRuns, settingLearnDailyCap, settingLearnAPIKey})
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
	key := ""
	if enc := values[settingLearnAPIKey]; enc != "" && s.encryptor != nil {
		if plain, err := s.encryptor.Decrypt(enc); err == nil {
			key = plain
		}
	}
	out.APIKeySet = key != ""
	return out, key
}

// Settings (admin; the key itself is never returned).
func (s *LearnService) Settings(ctx context.Context) LearnSettings {
	out, _ := s.loadSettings(ctx)
	return out
}

// SaveSettings stores the run settings; an empty APIKey keeps the stored one.
func (s *LearnService) SaveSettings(ctx context.Context, in LearnSettings) (LearnSettings, error) {
	if in.FreeRunsPerDay < 0 || in.FreeRunsPerDay > 500 || in.DailyCap < 0 || in.DailyCap > 100000 {
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
	out, key := s.loadSettings(ctx)
	if out.RunEnabled && key == "" {
		return out, errLearnRunInvalid("请填写学习专用 Key")
	}
	return out, nil
}

// Config is public.
func (s *LearnService) Config(ctx context.Context) LearnConfig {
	st, key := s.loadSettings(ctx)
	return LearnConfig{RunEnabled: st.RunEnabled && key != "" && st.Model != "", FreeRunsPerDay: st.FreeRunsPerDay, Model: st.Model}
}

func (s *LearnService) runsLeft(ctx context.Context, userID int64, st LearnSettings) (int, error) {
	used, err := s.repo.CountRuns(ctx, userID, learnDayStart(s.now()))
	if err != nil {
		return 0, err
	}
	return max(st.FreeRunsPerDay-used, 0), nil
}

// Me: completed lessons and today's free runs left.
func (s *LearnService) Me(ctx context.Context, userID int64) (*LearnMe, error) {
	done, err := s.repo.Progress(ctx, userID)
	if err != nil {
		return nil, err
	}
	st, _ := s.loadSettings(ctx)
	left, err := s.runsLeft(ctx, userID, st)
	if err != nil {
		return nil, err
	}
	return &LearnMe{Completed: done, RunsLeft: left}, nil
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

func validateLearnRun(in *LearnRunInput) error {
	if !learnLessonRe.MatchString(in.Lesson) {
		return ErrLearnLessonID
	}
	if len(in.Messages) == 0 || len(in.Messages) > learnMaxMessages {
		return errLearnRunInvalid("消息数量")
	}
	total := 0
	for _, m := range in.Messages {
		switch m.Role {
		case "system", "user", "assistant":
		default:
			return errLearnRunInvalid("消息角色")
		}
		n := utf8.RuneCountInString(m.Content)
		if n == 0 || n > learnMaxMessageChars {
			return errLearnRunInvalid(fmt.Sprintf("每条消息 1–%d 字", learnMaxMessageChars))
		}
		total += n
	}
	if total > learnMaxTotalChars {
		return errLearnRunInvalid("内容太长")
	}
	if in.Messages[len(in.Messages)-1].Role != "user" {
		return errLearnRunInvalid("最后一条须是用户消息")
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

// Run runs a lesson example for userID.
func (s *LearnService) Run(ctx context.Context, userID int64, in LearnRunInput) (*LearnRunResult, error) {
	st, key := s.loadSettings(ctx)
	if !st.RunEnabled || key == "" || st.Model == "" {
		return nil, ErrLearnRunDisabled
	}
	if err := validateLearnRun(&in); err != nil {
		return nil, err
	}
	since := learnDayStart(s.now())
	used, err := s.repo.CountRuns(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	if used >= st.FreeRunsPerDay {
		return nil, ErrLearnRunQuota
	}
	if st.DailyCap > 0 {
		all, err := s.repo.CountRuns(ctx, 0, since)
		if err != nil {
			return nil, err
		}
		if all >= st.DailyCap {
			return nil, ErrLearnRunBusy
		}
	}
	runID, err := s.repo.StartRun(ctx, userID, in.Lesson)
	if err != nil {
		return nil, err
	}
	started := s.now()
	result, callErr := s.callGateway(ctx, key, st.Model, in)
	latency := int(s.now().Sub(started) / time.Millisecond)
	if callErr != nil {
		_ = s.repo.FinishRun(context.WithoutCancel(ctx), runID, "failed", latency, 0, 0)
		return nil, callErr
	}
	_ = s.repo.FinishRun(context.WithoutCancel(ctx), runID, "ok", latency, result.PromptTokens, result.CompletionTokens)
	result.LatencyMs = latency
	result.RunsLeft = max(st.FreeRunsPerDay-used-1, 0)
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
