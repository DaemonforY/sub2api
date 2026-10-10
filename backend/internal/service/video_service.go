package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/errgroup"
)

const (
	videoMaxRunsPerUser   = 2
	videoMaxRunsSiteWide  = 8
	videoRunTimeout       = 30 * time.Minute
	videoSceneParallelism = 3
	videoDefaultModel     = "gpt-5.5"
	videoSceneTailPause   = 0.6 // seconds of silence after each scene's narration
	videoMinSceneSeconds  = 3.0
	videoMaxScenes        = 12
	videoEventsPageSize   = 200
	videoVersionsKept     = 30
)

// VideoService owns HiveGPT 视频 projects and runs the agent that writes them. Model calls go through
// this site's gateway with the user's API key, so they are billed like any other call.
type VideoService struct {
	repo     VideoRepository
	tts      VideoTTS
	stream   animationStreamFunc
	audioDir string
	mediaKey []byte // signs links to private uploaded files
	now      func() time.Time
	slots    chan struct{}

	mu      sync.Mutex
	runs    map[string]*videoRun
	uploads map[int64][]time.Time // upload times per user, so deleting does not reset the daily limit
	wg      sync.WaitGroup

	// models offered on the site and the per-user keys of their groups (SetModelBilling)
	settings      SettingRepository
	groups        GroupRepository
	keyStore      VideoKeyStore
	keyMaker      VideoKeyMaker
	grant         VideoGroupGrant
	newKey        func() (string, error)
	keyMu         sync.Mutex
	settingsCache videoSettingsCache
}

// SetModelBilling enables the admin-configured model list: each model runs through its own group
// with a per-user key created on first use.
func (s *VideoService) SetModelBilling(settings SettingRepository, groups GroupRepository, keys VideoKeyStore, maker VideoKeyMaker, grant VideoGroupGrant, newKey func() (string, error)) {
	s.settings, s.groups, s.keyStore, s.keyMaker, s.grant, s.newKey = settings, groups, keys, maker, grant, newKey
}

type videoRun struct {
	cancel context.CancelFunc
	userID int64
	key    string
	model  string // the model this run calls (set with the key when the run starts)

	mu sync.Mutex // guards p while scenes are built in parallel
	p  *VideoProject
}

func NewVideoService(repo VideoRepository, tts VideoTTS, gatewayURL, audioDir string) *VideoService {
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	client := &http.Client{}
	stream := func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
		return streamChatCompletion(ctx, client, gatewayURL, "hivegpt-video/1", key, model, messages, maxTokens, onDelta)
	}
	return newVideoService(repo, tts, stream, audioDir)
}

func newVideoService(repo VideoRepository, tts VideoTTS, stream animationStreamFunc, audioDir string) *VideoService {
	return &VideoService{repo: repo, tts: tts, stream: stream, audioDir: audioDir, mediaKey: newVideoMediaKey(), now: time.Now, slots: make(chan struct{}, videoMaxRunsSiteWide), runs: map[string]*videoRun{}}
}

// RecoverInterrupted marks projects a previous process left running; the user can resume them.
func (s *VideoService) RecoverInterrupted(ctx context.Context) {
	ids, err := s.repo.FailRunning(ctx, "服务器重启，生成中断了，点「继续生成」接着做（Interrupted by a server restart）")
	if err != nil {
		slog.Warn("video: failing interrupted projects", "err", err)
		return
	}
	for _, id := range ids {
		_ = s.repo.AddEvent(ctx, id, &VideoEvent{Kind: "error", Text: "服务器重启，生成中断了。点「继续生成」可以从中断处接着做。"})
	}
}

// --- create ----------------------------------------------------------------------------------

type VideoCreateInput struct {
	Mode    string       `json:"mode"`
	Prompt  string       `json:"prompt"`
	Options VideoOptions `json:"options"`
	RemixOf string       `json:"remix_of,omitempty"`
}

func normalizeVideoOptions(mode string, o *VideoOptions) {
	o.Ratio = findVideoRatio(o.Ratio).ID
	o.Style = findVideoStyle(o.Style).ID
	if _, ok := findVideoGenre(o.Genre); !ok {
		o.Genre = ""
	}
	if _, ok := findVideoCategory(o.Category); !ok {
		o.Category = ""
	}
	if !videoVoiceKnown(o.Voice) {
		o.Voice = DefaultVideoVoice
	}
	o.Rate = max(-30, min(o.Rate, 30))
	o.Model = strings.TrimSpace(o.Model)
	if o.Model == "" || len(o.Model) > 100 {
		o.Model = videoDefaultModel
	}
	if o.Length != "short" && o.Length != "long" {
		o.Length = "standard"
	}
	if mode == VideoModeMotion {
		o.Ask = false
		if o.Seconds < 3 || o.Seconds > 20 {
			o.Seconds = 8
		}
	}
	if len(o.Answers) > 5 {
		o.Answers = nil
	}
}

func (s *VideoService) Create(ctx context.Context, key *APIKey, in VideoCreateInput) (*VideoProject, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	if in.Mode != VideoModeMotion {
		in.Mode = VideoModeFilm
	}
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.Prompt == "" || len([]rune(in.Prompt)) > videoPromptMaxChars {
		return nil, ErrVideoPrompt
	}
	normalizeVideoOptions(in.Mode, &in.Options)
	if m, ok := s.videoModel(ctx, in.Options.Model); ok {
		in.Options.Model = m.ID
	}
	if err := s.checkCapacity(ctx, key.UserID); err != nil {
		return nil, err
	}
	ratio := findVideoRatio(in.Options.Ratio)
	p := &VideoProject{
		UserID: key.UserID, APIKeyID: key.ID, Mode: in.Mode, Title: videoTitleFromPrompt(in.Prompt), Prompt: in.Prompt,
		Options: in.Options, Status: VideoStatusRunning, Width: ratio.Width, Height: ratio.Height, Visibility: VideoVisibilityPrivate,
		Category: in.Options.Category, RemixOf: strings.TrimSpace(in.RemixOf),
	}
	if p.Category == "" && p.Mode == VideoModeFilm {
		p.Category = "film"
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	if p.RemixOf != "" {
		_ = s.repo.AddRemix(ctx, p.RemixOf)
	}
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: p.Prompt})
	s.start(p, key.Key, s.build)
	return p, nil
}

func (s *VideoService) checkCapacity(ctx context.Context, userID int64) error {
	n, err := s.repo.CountRunning(ctx, userID)
	if err != nil {
		return err
	}
	if n >= videoMaxRunsPerUser {
		return ErrVideoTooMany
	}
	return nil
}

// --- running the agent -----------------------------------------------------------------------

type videoTask func(ctx context.Context, r *videoRun) error

func (s *VideoService) start(p *VideoProject, key string, task videoTask) {
	ctx, cancel := context.WithTimeout(context.Background(), videoRunTimeout)
	r := &videoRun{cancel: cancel, userID: p.UserID, key: key, p: p}
	s.mu.Lock()
	s.runs[p.ID] = r
	s.mu.Unlock()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			cancel()
			s.mu.Lock()
			if s.runs[p.ID] == r {
				delete(s.runs, p.ID)
			}
			s.mu.Unlock()
		}()
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		case <-ctx.Done():
		}
		err := ctx.Err()
		if err == nil {
			var key, model string
			if key, model, err = s.runKey(ctx, p, r.key); err == nil {
				r.key, r.model = key, model
				err = task(ctx, r)
			}
		}
		s.finish(r, err, ctx)
	}()
}

func (s *VideoService) finish(r *videoRun, err error, ctx context.Context) {
	r.mu.Lock()
	p := r.p
	switch {
	case err == nil:
		if p.Status == VideoStatusRunning {
			p.Status, p.Stage, p.Error = VideoStatusReady, "", ""
		}
	case errors.Is(ctx.Err(), context.Canceled):
		p.Status, p.Stage, p.Error = VideoStatusStopped, "", "已停止"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		p.Status, p.Error = VideoStatusFailed, "生成超过 30 分钟仍未完成，已停止，可以点「继续生成」（Timed out）"
	default:
		p.Status, p.Error = VideoStatusFailed, videoErrorText(err)
	}
	status, msg := p.Status, p.Error
	r.mu.Unlock()
	wctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s.save(wctx, r)
	switch status {
	case VideoStatusFailed:
		_ = s.repo.AddEvent(wctx, p.ID, &VideoEvent{Kind: "error", Text: msg})
	case VideoStatusStopped:
		_ = s.repo.AddEvent(wctx, p.ID, &VideoEvent{Kind: "step", Text: "已停止生成", Data: videoStepData("stopped")})
	}
}

func videoErrorText(err error) string {
	if errors.Is(err, errEdgeTTS) {
		return "配音服务暂时不可用，请稍后点「继续生成」重试（TTS unavailable）"
	}
	if msg := infraerrors.Message(err); msg != "" {
		return msg
	}
	return "生成失败，请重试（Generation failed）"
}

func videoStepData(state string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"state": state})
	return b
}

func (s *VideoService) save(ctx context.Context, r *videoRun) {
	r.mu.Lock()
	snapshot := *r.p
	if r.p.Spec != nil {
		spec := *r.p.Spec
		spec.Scenes = append([]VideoScene(nil), r.p.Spec.Scenes...)
		snapshot.Spec = &spec
		snapshot.Duration = spec.TotalDuration()
	}
	r.mu.Unlock()
	if err := s.repo.Save(ctx, &snapshot); err != nil {
		slog.Warn("video: save project", "id", snapshot.ID, "err", err)
	}
}

func (s *VideoService) step(ctx context.Context, r *videoRun, stage, text string) {
	r.mu.Lock()
	r.p.Stage = stage
	id := r.p.ID
	r.mu.Unlock()
	s.save(ctx, r)
	_ = s.repo.AddEvent(ctx, id, &VideoEvent{Kind: "step", Text: text, Data: videoStepData("running")})
}

func (s *VideoService) done(ctx context.Context, r *videoRun, text string) {
	_ = s.repo.AddEvent(ctx, r.p.ID, &VideoEvent{Kind: "step", Text: text, Data: videoStepData("done")})
}

// chat runs one model call and adds its usage to the project.
func (s *VideoService) chat(ctx context.Context, r *videoRun, msgs []LearnMessage, maxTokens int) (string, error) {
	for attempt := 0; ; attempt++ {
		answer, err := s.chatOnce(ctx, r, msgs, maxTokens)
		if err == nil || attempt >= len(videoRetryBackoff) || !videoRetryable(err) {
			return answer, err
		}
		// Scenes are written in parallel, so the account's concurrency limit (or a rate limit) can
		// refuse one of them: wait and try again instead of failing the scene.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(videoRetryBackoff[attempt]):
		}
	}
}

var videoRetryBackoff = []time.Duration{4 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second}

// videoRetryable reports errors that go away by waiting: the per-account concurrency limit, rate
// limits and an overloaded upstream.
func videoRetryable(err error) bool {
	msg := strings.ToLower(err.Error())
	for _, s := range []string{"并发上限", "concurrency limit", "rate limit", "too many requests", "请求太频繁", "temporarily unavailable", "overloaded", "529"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

func (s *VideoService) chatOnce(ctx context.Context, r *videoRun, msgs []LearnMessage, maxTokens int) (string, error) {
	var out strings.Builder
	model := r.model
	if model == "" {
		model = r.p.Options.Model
	}
	res, err := s.stream(ctx, r.key, model, msgs, maxTokens, func(d string) error {
		_, _ = out.WriteString(d)
		return nil
	})
	r.mu.Lock()
	r.p.Usage.Calls++
	if res != nil {
		r.p.Usage.PromptTokens += res.PromptTokens
		r.p.Usage.CompletionTokens += res.CompletionTokens
	}
	r.mu.Unlock()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// build is the full pipeline. Every step skips work already done, so it also resumes a project.
func (s *VideoService) build(ctx context.Context, r *videoRun) error {
	p := r.p
	if p.Mode == VideoModeFilm && p.Spec == nil {
		if p.Options.Ask && p.Options.Answers == nil {
			return s.askQuestions(ctx, r)
		}
		if err := s.writeScript(ctx, r); err != nil {
			return err
		}
	}
	if p.Mode == VideoModeMotion && p.Spec == nil {
		s.motionSpec(r)
	}
	if err := s.voiceScenes(ctx, r); err != nil {
		return err
	}
	if err := s.codeScenes(ctx, r, nil); err != nil {
		return err
	}
	s.done(ctx, r, fmt.Sprintf("完成：%d 个分镜，共 %.0f 秒。可以在右边继续让 AI 修改。", len(p.Spec.Scenes), p.Spec.TotalDuration()))
	s.save(ctx, r)
	_ = s.repo.AddVersion(ctx, p.ID, "初版", p.Spec)
	return nil
}

func (s *VideoService) askQuestions(ctx context.Context, r *videoRun) error {
	s.step(ctx, r, "questions", "正在理解你的需求…")
	answer, err := s.chat(ctx, r, videoQuestionsPrompt(r.p), 3000)
	if err != nil {
		return err
	}
	var parsed struct {
		Questions []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Options []struct {
				Label       string `json:"label"`
				Recommended bool   `json:"recommended"`
			} `json:"options"`
		} `json:"questions"`
	}
	if raw := extractJSONObject(answer); raw == "" || json.Unmarshal([]byte(raw), &parsed) != nil || len(parsed.Questions) == 0 {
		// No usable questions: go straight on with the defaults.
		r.p.Options.Answers = map[string]string{}
		return s.build(ctx, r)
	}
	if len(parsed.Questions) > 3 {
		parsed.Questions = parsed.Questions[:3]
	}
	data, _ := json.Marshal(parsed)
	_ = s.repo.AddEvent(ctx, r.p.ID, &VideoEvent{Kind: "question", Text: "为了更贴合你的需求，请确认以下方向（直接点「按推荐继续」也可以）：", Data: data})
	r.mu.Lock()
	r.p.Status, r.p.Stage = VideoStatusQuestions, "questions"
	r.mu.Unlock()
	return nil
}

func (s *VideoService) writeScript(ctx context.Context, r *videoRun) error {
	s.step(ctx, r, "script", "正在写脚本和分镜…")
	answer, err := s.chat(ctx, r, videoScriptPrompt(r.p), 12000)
	if err != nil {
		return err
	}
	var spec VideoSpec
	raw := extractJSONObject(answer)
	if raw == "" || json.Unmarshal([]byte(raw), &spec) != nil || len(spec.Scenes) == 0 {
		return errLearnRunFailed("模型没有写出可用的脚本，请重试或换一个更强的模型")
	}
	if len(spec.Scenes) > videoMaxScenes {
		spec.Scenes = spec.Scenes[:videoMaxScenes]
	}
	style := findVideoStyle(r.p.Options.Style)
	spec.Theme = style.Theme
	spec.Width, spec.Height = r.p.Width, r.p.Height
	for i := range spec.Scenes {
		sc := &spec.Scenes[i]
		sc.ID = fmt.Sprintf("s%d", i+1)
		sc.Code, sc.Audio, sc.Error = "", nil, ""
		sc.Duration = math.Max(videoMinSceneSeconds, float64(len([]rune(sc.Narration)))/4.5+videoSceneTailPause)
	}
	spec.Title = clipRunes(spec.Title, videoTitleMaxChars)
	r.mu.Lock()
	r.p.Spec = &spec
	if spec.Title != "" {
		r.p.Title = spec.Title
	}
	r.mu.Unlock()
	s.save(ctx, r)
	s.done(ctx, r, fmt.Sprintf("脚本完成：%d 个分镜，约 %.0f 秒", len(spec.Scenes), spec.TotalDuration()))
	return nil
}

func (s *VideoService) motionSpec(r *videoRun) {
	p := r.p
	style := findVideoStyle(p.Options.Style)
	r.mu.Lock()
	defer r.mu.Unlock()
	p.Spec = &VideoSpec{Title: p.Title, Theme: style.Theme, Width: p.Width, Height: p.Height, Loop: true,
		Scenes: []VideoScene{{ID: "s1", Title: p.Title, Visual: p.Prompt, Duration: float64(p.Options.Seconds)}}}
}

func videoAudioName(projectID string, sc VideoScene, voice string, rate int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", voice, rate, sc.Narration)))
	return fmt.Sprintf("%s-%s.mp3", sc.ID, hex.EncodeToString(sum[:6]))
}

func (s *VideoService) voiceScenes(ctx context.Context, r *videoRun) error {
	p := r.p
	if p.Mode != VideoModeFilm || p.Spec == nil {
		return nil
	}
	var todo []int
	for i, sc := range p.Spec.Scenes {
		if strings.TrimSpace(sc.Narration) == "" {
			continue
		}
		if sc.Audio == nil || sc.Audio.File != videoAudioName(p.ID, sc, p.Options.Voice, p.Options.Rate) {
			todo = append(todo, i)
		}
	}
	if len(todo) == 0 {
		return nil
	}
	s.step(ctx, r, "voice", fmt.Sprintf("正在配音（%d 段）…", len(todo)))
	dir := filepath.Join(s.audioDir, p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoSceneParallelism)
	for _, i := range todo {
		g.Go(func() error {
			r.mu.Lock()
			sc := p.Spec.Scenes[i]
			r.mu.Unlock()
			speech, err := s.tts.Synthesize(gctx, sc.Narration, p.Options.Voice, p.Options.Rate)
			if err != nil {
				return err
			}
			name := videoAudioName(p.ID, sc, p.Options.Voice, p.Options.Rate)
			if err := os.WriteFile(filepath.Join(dir, name), speech.Audio, 0o644); err != nil {
				return err
			}
			r.mu.Lock()
			cur := &p.Spec.Scenes[i]
			cur.Audio = &VideoSceneAudio{File: name, Duration: speech.Duration, Words: speech.Words}
			cur.Duration = math.Max(videoMinSceneSeconds, speech.Duration+videoSceneTailPause)
			p.Usage.TTSChars += len([]rune(sc.Narration))
			r.mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	s.save(ctx, r)
	s.done(ctx, r, fmt.Sprintf("配音完成，全片 %.0f 秒", p.Spec.TotalDuration()))
	return nil
}

// codeScenes writes the code of every scene without code, and rewrites the scenes in redo with
// their instruction (or failure, prefixed "!error:").
func (s *VideoService) codeScenes(ctx context.Context, r *videoRun, redo map[string]string) error {
	p := r.p
	var todo []int
	for i, sc := range p.Spec.Scenes {
		if _, again := redo[sc.ID]; again || strings.TrimSpace(sc.Code) == "" {
			todo = append(todo, i)
		}
	}
	if len(todo) == 0 {
		return nil
	}
	if len(todo) > 1 {
		s.step(ctx, r, "code", fmt.Sprintf("正在制作 %d 个分镜的动画（并行）…", len(todo)))
	}
	style := findVideoStyle(p.Options.Style)
	category, _ := findVideoCategory(p.Options.Category)
	outline := make([]string, len(p.Spec.Scenes))
	for i, sc := range p.Spec.Scenes {
		outline[i] = fmt.Sprintf("%d.%s", i+1, sc.Title)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoSceneParallelism)
	var failed []string
	var failedMu sync.Mutex
	for _, i := range todo {
		g.Go(func() error {
			r.mu.Lock()
			sc := p.Spec.Scenes[i]
			spec := *p.Spec
			r.mu.Unlock()
			if len(todo) == 1 {
				s.step(gctx, r, "code", fmt.Sprintf("正在制作分镜 %d「%s」…", i+1, sc.Title))
			}
			brief := videoSceneBrief{Index: i, Scene: sc, Outline: outline, Spec: &spec, StyleBrief: style.Brief, Category: category.Brief, Motion: p.Mode == VideoModeMotion, Prompt: p.Prompt}
			instruction, failure := redo[sc.ID], ""
			if strings.HasPrefix(instruction, "!error:") {
				failure, instruction = strings.TrimPrefix(instruction, "!error:"), ""
			}
			previous := ""
			if instruction != "" || failure != "" {
				previous = sc.Code
			}
			code, err := s.writeSceneCode(gctx, r, brief, previous, instruction, failure)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err()
				}
				failedMu.Lock()
				failed = append(failed, fmt.Sprintf("分镜 %d", i+1))
				failedMu.Unlock()
				r.mu.Lock()
				p.Spec.Scenes[i].Error = videoErrorText(err)
				r.mu.Unlock()
				return nil
			}
			r.mu.Lock()
			p.Spec.Scenes[i].Code, p.Spec.Scenes[i].Error = code, ""
			r.mu.Unlock()
			s.save(gctx, r)
			s.done(gctx, r, fmt.Sprintf("分镜 %d「%s」完成", i+1, sc.Title))
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	if len(failed) > 0 {
		return errLearnRunFailed(strings.Join(failed, "、") + " 没有做出来，可以点「继续生成」重试")
	}
	return nil
}

// writeSceneCode asks for the scene's code, checks its syntax and lets the model fix a syntax error once.
func (s *VideoService) writeSceneCode(ctx context.Context, r *videoRun, brief videoSceneBrief, previous, instruction, failure string) (string, error) {
	answer, err := s.chat(ctx, r, videoScenePrompt(brief, previous, instruction, failure), 24000)
	if err != nil {
		return "", err
	}
	code := extractCodeBlock(answer)
	if problem := checkSceneCode(code); problem != "" {
		answer, err = s.chat(ctx, r, videoScenePrompt(brief, code, "", problem), 24000)
		if err != nil {
			return "", err
		}
		code = extractCodeBlock(answer)
		if problem = checkSceneCode(code); problem != "" {
			return "", errLearnRunFailed("动画代码有语法错误（" + clipRunes(problem, 80) + "）")
		}
	}
	return code, nil
}

var videoForbiddenCode = regexp.MustCompile(`\b(fetch|XMLHttpRequest|WebSocket|EventSource|importScripts|navigator\.sendBeacon)\b|\bimport\s*\(`)

// checkSceneCode returns "" when the scene body parses as a function and avoids the network.
func checkSceneCode(code string) string {
	if strings.TrimSpace(code) == "" {
		return "the answer contained no code"
	}
	if !strings.Contains(code, "return") {
		return "the body must end with `return function update(t) { … }`"
	}
	if m := videoForbiddenCode.FindString(code); m != "" {
		return "network access is not allowed (" + m + ")"
	}
	res := api.Transform("(function(S){\n"+code+"\n})", api.TransformOptions{Loader: api.LoaderJS, Target: api.ES2022, LogLevel: api.LogLevelSilent})
	if len(res.Errors) > 0 {
		e := res.Errors[0]
		line := 0
		if e.Location != nil {
			line = e.Location.Line - 1
		}
		return fmt.Sprintf("syntax error at line %d: %s", line, e.Text)
	}
	return ""
}

// --- owner actions ---------------------------------------------------------------------------

func (s *VideoService) own(ctx context.Context, userID int64, id string) (*VideoProject, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.UserID != userID {
		return nil, ErrVideoNotFound
	}
	return s.withMedia(p), nil
}

func (s *VideoService) running(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.runs[id]
	return ok
}

func (s *VideoService) Get(ctx context.Context, userID int64, id string) (*VideoProject, error) {
	return s.own(ctx, userID, id)
}

func (s *VideoService) List(ctx context.Context, userID int64) ([]VideoCard, error) {
	cards, err := s.repo.ListByUser(ctx, userID, 100)
	return s.cardsWithMedia(cards), err
}

func (s *VideoService) Events(ctx context.Context, userID int64, id string, after int64) ([]VideoEvent, error) {
	if _, err := s.own(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, id, after, videoEventsPageSize)
}

// startOn re-runs the agent on an existing project with task, after the checks every action shares.
func (s *VideoService) startOn(ctx context.Context, key *APIKey, id string, prepare func(p *VideoProject) error, task videoTask) (*VideoProject, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	p, err := s.own(ctx, key.UserID, id)
	if err != nil {
		return nil, err
	}
	if p.Mode == VideoModeUpload {
		return nil, ErrVideoUploadAgent
	}
	if s.running(id) || p.Status == VideoStatusRunning {
		return nil, ErrVideoBusy
	}
	if err := s.checkCapacity(ctx, key.UserID); err != nil {
		return nil, err
	}
	if prepare != nil {
		if err := prepare(p); err != nil {
			return nil, err
		}
	}
	p.Status, p.Error, p.APIKeyID = VideoStatusRunning, "", key.ID
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	s.start(p, key.Key, task)
	return p, nil
}

// Answer records the user's choices for the clarifying questions and writes the video.
func (s *VideoService) Answer(ctx context.Context, key *APIKey, id string, answers map[string]string) (*VideoProject, error) {
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Status != VideoStatusQuestions {
			return ErrVideoNoQuestion
		}
		clean := map[string]string{}
		var lines []string
		for k, v := range answers {
			k, v = clipRunes(k, 40), clipRunes(v, 60)
			if k != "" && v != "" && len(clean) < 5 {
				clean[k] = v
				lines = append(lines, k+"："+v)
			}
		}
		p.Options.Answers = clean
		text := "按推荐继续"
		if len(lines) > 0 {
			text = strings.Join(lines, "\n")
		}
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "answer", Text: text})
	}, s.build)
}

// Resume continues a stopped or failed project from where it ended.
func (s *VideoService) Resume(ctx context.Context, key *APIKey, id string) (*VideoProject, error) {
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Status == VideoStatusReady || p.Status == VideoStatusQuestions {
			return ErrVideoInvalid
		}
		if p.Spec != nil {
			for i := range p.Spec.Scenes {
				if p.Spec.Scenes[i].Error != "" {
					p.Spec.Scenes[i].Code, p.Spec.Scenes[i].Error = "", ""
				}
			}
		}
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: "继续生成"})
	}, s.build)
}

// Message asks the agent to change the video.
func (s *VideoService) Message(ctx context.Context, key *APIKey, id, text, focusScene string) (*VideoProject, error) {
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) > 2000 {
		return nil, ErrVideoInvalid
	}
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil || p.Status == VideoStatusQuestions {
			return ErrVideoNotReady
		}
		_ = s.repo.AddVersion(ctx, p.ID, "修改前："+clipRunes(text, 40), p.Spec)
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: text})
	}, func(ctx context.Context, r *videoRun) error { return s.revise(ctx, r, text, focusScene) })
}

func (s *VideoService) revise(ctx context.Context, r *videoRun, text, focus string) error {
	p := r.p
	redo := map[string]string{}
	if p.Mode == VideoModeMotion {
		redo[p.Spec.Scenes[0].ID] = text
		_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "assistant", Text: "好的，我来按你的要求修改。"})
	} else {
		s.step(ctx, r, "plan", "正在理解修改意见…")
		answer, err := s.chat(ctx, r, videoRevisionPrompt(p, text, focus), 4000)
		if err != nil {
			return err
		}
		var plan videoRevisionPlan
		if raw := extractJSONObject(answer); raw == "" || json.Unmarshal([]byte(raw), &plan) != nil {
			return errLearnRunFailed("没能理解这次修改，请换个说法再试")
		}
		r.mu.Lock()
		for _, change := range plan.Scenes {
			for i := range p.Spec.Scenes {
				sc := &p.Spec.Scenes[i]
				if sc.ID != change.ID {
					continue
				}
				if n := strings.TrimSpace(change.Narration); n != "" && n != sc.Narration {
					sc.Narration = n
				}
				if t := strings.TrimSpace(change.Title); t != "" {
					sc.Title = clipRunes(t, 40)
				}
				redo[sc.ID] = firstNonEmpty(strings.TrimSpace(change.Instruction), text)
			}
		}
		if t := strings.TrimSpace(plan.Title); t != "" {
			p.Spec.Title, p.Title = clipRunes(t, videoTitleMaxChars), clipRunes(t, videoTitleMaxChars)
		}
		r.mu.Unlock()
		if reply := strings.TrimSpace(plan.Reply); reply != "" {
			_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "assistant", Text: reply})
		}
		if len(redo) == 0 {
			return nil
		}
		if err := s.voiceScenes(ctx, r); err != nil {
			return err
		}
	}
	if err := s.codeScenes(ctx, r, redo); err != nil {
		return err
	}
	s.done(ctx, r, fmt.Sprintf("修改完成（%d 个分镜）", len(redo)))
	return nil
}

// Repair rewrites a scene whose code failed in the player.
func (s *VideoService) Repair(ctx context.Context, key *APIKey, id, sceneID, failure string) (*VideoProject, error) {
	failure = clipRunes(failure, 1500)
	if sceneID == "" || failure == "" {
		return nil, ErrVideoInvalid
	}
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil {
			return ErrVideoNotReady
		}
		for _, sc := range p.Spec.Scenes {
			if sc.ID == sceneID {
				return nil
			}
		}
		return ErrVideoInvalid
	}, func(ctx context.Context, r *videoRun) error {
		s.step(ctx, r, "repair", "播放时发现错误，正在修复…")
		if err := s.codeScenes(ctx, r, map[string]string{sceneID: "!error:" + failure}); err != nil {
			return err
		}
		s.done(ctx, r, "已修复")
		return nil
	})
}

// VideoSceneEdit is a script change made in the 脚本 tab.
type VideoSceneEdit struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Narration string `json:"narration"`
	Visual    string `json:"visual"`
}

// EditScript applies script edits and rebuilds the scenes they touch (re-recording changed narration).
func (s *VideoService) EditScript(ctx context.Context, key *APIKey, id string, edits []VideoSceneEdit) (*VideoProject, error) {
	redo := map[string]string{}
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil || p.Mode != VideoModeFilm {
			return ErrVideoNotReady
		}
		_ = s.repo.AddVersion(ctx, p.ID, "修改脚本前", p.Spec)
		for _, e := range edits {
			for i := range p.Spec.Scenes {
				sc := &p.Spec.Scenes[i]
				if sc.ID != e.ID {
					continue
				}
				changed := []string{}
				if t := clipRunes(e.Title, 40); t != "" && t != sc.Title {
					sc.Title = t
					changed = append(changed, "title")
				}
				if n := clipRunes(e.Narration, 1500); n != "" && n != sc.Narration {
					sc.Narration = n
					changed = append(changed, "narration")
				}
				if v := clipRunes(e.Visual, 3000); v != "" && v != sc.Visual {
					sc.Visual = v
					changed = append(changed, "visual plan")
				}
				if len(changed) > 0 {
					redo[sc.ID] = "The scene's " + strings.Join(changed, " and ") + " changed (see the user message above). Rebuild the animation so it matches them and the narration timing."
				}
			}
		}
		if len(redo) == 0 {
			return ErrVideoInvalid
		}
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: fmt.Sprintf("修改了脚本（%d 个分镜）", len(redo))})
	}, func(ctx context.Context, r *videoRun) error {
		if err := s.voiceScenes(ctx, r); err != nil {
			return err
		}
		if err := s.codeScenes(ctx, r, redo); err != nil {
			return err
		}
		s.done(ctx, r, "已按新脚本更新")
		return nil
	})
}

// SetVoice changes the narrator and re-records every scene.
func (s *VideoService) SetVoice(ctx context.Context, key *APIKey, id, voice string, rate int) (*VideoProject, error) {
	if !videoVoiceKnown(voice) {
		return nil, ErrVideoInvalid
	}
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil || p.Mode != VideoModeFilm {
			return ErrVideoNotReady
		}
		p.Options.Voice, p.Options.Rate = voice, max(-30, min(rate, 30))
		return s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: "换配音：" + voice})
	}, func(ctx context.Context, r *videoRun) error {
		// Scene lengths follow the new audio; the code reads S.duration, so it keeps fitting.
		if err := s.voiceScenes(ctx, r); err != nil {
			return err
		}
		s.done(ctx, r, "已换好配音")
		return nil
	})
}

func (s *VideoService) Stop(ctx context.Context, userID int64, id string) error {
	if _, err := s.own(ctx, userID, id); err != nil {
		return err
	}
	s.mu.Lock()
	r := s.runs[id]
	s.mu.Unlock()
	if r != nil {
		r.cancel()
	}
	return nil
}

func (s *VideoService) Delete(ctx context.Context, userID int64, id string) error {
	_ = s.Stop(ctx, userID, id)
	ok, err := s.repo.Delete(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrVideoNotFound
	}
	_ = os.RemoveAll(filepath.Join(s.audioDir, filepath.Base(id)))
	return nil
}

func (s *VideoService) Versions(ctx context.Context, userID int64, id string) ([]VideoVersion, error) {
	if _, err := s.own(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.repo.ListVersions(ctx, id, videoVersionsKept)
}

// Restore puts an earlier version back (the current one is kept as a version first).
func (s *VideoService) Restore(ctx context.Context, userID int64, id string, versionID int64) (*VideoProject, error) {
	p, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Mode == VideoModeUpload {
		return nil, ErrVideoUploadAgent
	}
	if s.running(id) {
		return nil, ErrVideoBusy
	}
	v, err := s.repo.GetVersion(ctx, id, versionID)
	if err != nil {
		return nil, err
	}
	if v == nil || v.Spec == nil {
		return nil, ErrVideoNotFound
	}
	if p.Spec != nil {
		_ = s.repo.AddVersion(ctx, id, "回退前", p.Spec)
	}
	p.Spec, p.Status, p.Error, p.Stage = v.Spec, VideoStatusReady, "", ""
	p.Duration = v.Spec.TotalDuration()
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.AddEvent(ctx, id, &VideoEvent{Kind: "step", Text: "已回退到「" + v.Note + "」", Data: videoStepData("done")})
	return p, nil
}

// AudioPath returns the file of a scene's narration when the caller may play it: the owner, or anyone
// for a public work.
func (s *VideoService) AudioPath(ctx context.Context, userID int64, admin bool, id, file string) (string, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if p == nil || (p.UserID != userID && p.Visibility != VideoVisibilityPublic && !admin) {
		return "", ErrVideoNotFound
	}
	name := filepath.Base(file)
	if name != file || !strings.HasSuffix(name, ".mp3") {
		return "", ErrVideoNotFound
	}
	path := filepath.Join(s.audioDir, filepath.Base(p.ID), name)
	if _, err := os.Stat(path); err != nil {
		return "", ErrVideoNotFound
	}
	return path, nil
}

// --- gallery ---------------------------------------------------------------------------------

// Publish submits a finished work to the gallery; it is shown after review.
func (s *VideoService) Publish(ctx context.Context, userID int64, id, category, title string) (*VideoProject, error) {
	p, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if p.Status != VideoStatusReady || p.Spec == nil {
		return nil, ErrVideoNotReady
	}
	if _, ok := findVideoCategory(category); !ok {
		category = p.Category
	}
	title = clipRunes(firstNonEmpty(title, p.Title), videoTitleMaxChars)
	if err := s.repo.SetVisibility(ctx, id, VideoVisibilityPending, category, title, nil); err != nil {
		return nil, err
	}
	return s.own(ctx, userID, id)
}

func (s *VideoService) Unpublish(ctx context.Context, userID int64, id string) (*VideoProject, error) {
	p, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetVisibility(ctx, id, VideoVisibilityPrivate, p.Category, p.Title, nil); err != nil {
		return nil, err
	}
	return s.own(ctx, userID, id)
}

func (s *VideoService) Gallery(ctx context.Context, q VideoGalleryQuery) ([]VideoCard, int, error) {
	q.PageSize = max(1, min(q.PageSize, 48))
	q.Page = max(1, q.Page)
	q.Search = clipRunes(q.Search, 40)
	cards, total, err := s.repo.Gallery(ctx, q)
	return s.cardsWithMedia(cards), total, err
}

// Work returns a public work for the gallery player.
func (s *VideoService) Work(ctx context.Context, id string) (*VideoProject, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Visibility != VideoVisibilityPublic {
		return nil, ErrVideoNotFound
	}
	p.Usage = VideoUsage{}
	p.Options.Model = ""
	return s.withMedia(p), nil
}

func (s *VideoService) View(ctx context.Context, id string) {
	_ = s.repo.AddView(ctx, id)
}

// AdminWork returns any project for review.
func (s *VideoService) AdminWork(ctx context.Context, admin *User, id string) (*VideoProject, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return nil, ErrVideoForbidden
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrVideoNotFound
	}
	return s.withMedia(p), nil
}

func (s *VideoService) Pending(ctx context.Context, admin *User) ([]VideoCard, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return nil, ErrVideoForbidden
	}
	cards, err := s.repo.Pending(ctx, 100)
	return s.cardsWithMedia(cards), err
}

// Review approves / rejects a submitted work or changes a public one (feature, category, hide).
func (s *VideoService) Review(ctx context.Context, admin *User, id, decision, category string, featured *bool) error {
	if admin == nil || admin.Role != RoleAdmin {
		return ErrVideoForbidden
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrVideoNotFound
	}
	visibility := p.Visibility
	switch decision {
	case "approve":
		visibility = VideoVisibilityPublic
	case "reject":
		visibility = VideoVisibilityRejected
	case "hide":
		visibility = VideoVisibilityPrivate
	case "", "update":
	default:
		return ErrVideoInvalid
	}
	if _, ok := findVideoCategory(category); !ok {
		category = p.Category
	}
	return s.repo.SetVisibility(ctx, id, visibility, category, p.Title, featured)
}

// VideoAdminEdit changes a work's details from 作品管理 on the review page; nil fields stay.
type VideoAdminEdit struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Category    string  `json:"category"`
	Featured    *bool   `json:"featured"`
	// Visibility: "" keeps it, "public" publishes at once, "private" takes the work off the gallery.
	Visibility string `json:"visibility"`
}

// AdminEdit lets an admin rename, describe, recategorise, feature, publish or hide any work.
func (s *VideoService) AdminEdit(ctx context.Context, admin *User, id string, in VideoAdminEdit) (*VideoProject, error) {
	p, err := s.AdminWork(ctx, admin, id)
	if err != nil {
		return nil, err
	}
	visibility := p.Visibility
	switch in.Visibility {
	case "":
	case VideoVisibilityPublic:
		if p.Status != VideoStatusReady || p.Spec == nil {
			return nil, ErrVideoNotReady
		}
		visibility = VideoVisibilityPublic
	case VideoVisibilityPrivate:
		visibility = VideoVisibilityPrivate
	default:
		return nil, ErrVideoInvalid
	}
	if in.Title != nil || in.Description != nil {
		if in.Title != nil {
			if t := clipRunes(*in.Title, videoTitleMaxChars); t != "" {
				p.Title = t
				if p.Spec != nil {
					p.Spec.Title = t
				}
			}
		}
		if in.Description != nil {
			p.Prompt = clipRunes(*in.Description, videoPromptMaxChars)
			if p.Spec != nil && p.Mode == VideoModeUpload {
				p.Spec.Summary = clipRunes(p.Prompt, 300)
			}
		}
		if err := s.repo.Save(ctx, p); err != nil {
			return nil, err
		}
	}
	category := in.Category
	if _, ok := findVideoCategory(category); !ok {
		category = p.Category
	}
	if err := s.repo.SetVisibility(ctx, id, visibility, category, p.Title, in.Featured); err != nil {
		return nil, err
	}
	return s.AdminWork(ctx, admin, id)
}

// Wait blocks until all agent runs ended (tests).
func (s *VideoService) Wait() { s.wg.Wait() }
