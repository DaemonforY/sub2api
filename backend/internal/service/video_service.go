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
	"slices"
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
	videoDefaultModel     = "gpt-6.1-sol"
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
	images  map[int64][]time.Time // staged picture uploads per user (daily limit)
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

	// paid narrators (SetPaidTTS)
	paidTTS      VideoPaidTTS
	ttsBilling   VideoTTSBilling
	balanceCache videoBalanceCache

	// 我的音色 (SetVoiceClones)
	clones   VideoCloneRepository
	trainer  VideoVoiceTrainer
	previews map[int64][]time.Time // guarded by mu

	// MP4 export and the publishing kit (SetRenderer)
	exports  VideoExportRepository
	renderer *videoRenderer
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
	// Images: tokens of pictures staged with POST /video/images, in the order the user added them.
	Images []string `json:"images,omitempty"`
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
	in.Options.Images = nil // only from staged uploads, below
	if len(in.Images) > VideoImagesMax {
		return nil, ErrVideoImageCount
	}
	if videoVoicePaid(in.Options.Voice) && s.paidTTS == nil {
		return nil, ErrVideoPaidVoiceOff
	}
	if err := s.checkCloneVoice(ctx, key.UserID, in.Options.Voice); err != nil {
		return nil, err
	}
	if m, ok := s.videoModel(ctx, in.Options.Model); ok {
		in.Options.Model = m.ID
	}
	if err := s.checkCapacity(ctx, key.UserID); err != nil {
		return nil, err
	}
	if err := s.checkBalance(ctx, key); err != nil {
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
	if len(in.Images) > 0 {
		images, err := s.attachImages(p, in.Images)
		if err == nil {
			p.Options.Images = images
			err = s.repo.Save(ctx, p)
		}
		if err != nil {
			_, _ = s.repo.Delete(ctx, p.UserID, p.ID)
			_ = os.RemoveAll(s.projectDir(p.ID))
			return nil, err
		}
	}
	if p.RemixOf != "" {
		_ = s.repo.AddRemix(ctx, p.RemixOf)
	}
	text := p.Prompt
	if n := len(p.Options.Images); n > 0 {
		text += fmt.Sprintf("\n（附 %d 张图片）", n)
	}
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "user", Text: text})
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

// checkBalance refuses a run up front when the site's models bill the balance and it is used up;
// otherwise the run would stop at its first model call with a less clear error.
func (s *VideoService) checkBalance(ctx context.Context, key *APIKey) error {
	if key.User == nil || key.User.Balance > 0 {
		return nil
	}
	if _, ok := s.videoModel(ctx, ""); !ok {
		return nil // runs use the user's own key, which may be on a subscription
	}
	return ErrVideoBalance
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
		slog.Warn("video: run failed", "id", p.ID, "err", err)
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
	waited := time.Duration(0)
	for attempt := 0; ; attempt++ {
		answer, err := s.chatOnce(ctx, r, msgs, maxTokens)
		if err != nil && videoBalanceError(err) {
			return answer, ErrVideoBalance
		}
		if err != nil && videoConcurrencyError(err) && waited < videoConcurrencyMaxWait {
			// The user's own concurrency limit (2 for new accounts) is lower than the scenes
			// written at once: wait for a slot as long as it takes the other scenes to finish.
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(videoConcurrencyRetry):
			}
			waited += videoConcurrencyRetry
			continue
		}
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

const (
	videoSilentSceneMin float64 = 2.5
	videoSilentSceneMax float64 = 15
)

var videoRetryBackoff = []time.Duration{4 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second}

var (
	videoConcurrencyRetry   = 10 * time.Second
	videoConcurrencyMaxWait = 20 * time.Minute
)

// videoConcurrencyError: the gateway refused the call because the user (or key) already has as
// many requests running as allowed.
func videoConcurrencyError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "并发上限") || strings.Contains(msg, "concurrency limit")
}

// videoBalanceError: the gateway refused the call for lack of balance.
func videoBalanceError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "余额不足") || strings.Contains(msg, "insufficient balance") || strings.Contains(msg, "insufficient account balance")
}

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
	if res != nil && (res.FinishReason == "length" || (maxTokens > 0 && res.CompletionTokens >= maxTokens)) {
		return out.String(), errVideoTruncated
	}
	return out.String(), nil
}

// earlierRequests returns the user's last few messages before this one (oldest first), so the
// planner can redo a change the user says did not happen.
func (s *VideoService) earlierRequests(ctx context.Context, id, current string) []string {
	events, err := s.repo.ListEvents(ctx, id, 0, 500)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range events {
		if e.Kind == "user" {
			out = append(out, clipRunes(e.Text, 300))
		}
	}
	if n := len(out); n > 0 && out[n-1] == clipRunes(current, 300) {
		out = out[:n-1] // the current message was already added as an event
	}
	if len(out) > 4 {
		out = out[len(out)-4:]
	}
	return out
}

// errVideoTruncated: the answer stopped at the output limit (cut off mid-code / mid-JSON).
// videoPendingChange (scene meta): a requested change that failed, applied by 继续生成.
const videoPendingChange = "pending_change"

var errVideoSceneTooLong = errLearnRunFailed("这个分镜的动画代码太长，超过了单次输出上限。可以把这个分镜的要求说得简单些（比如少几个部件或动作）再试")

var errVideoTruncated = errLearnRunFailed("回答超过了单次输出上限被截断")

const (
	videoSceneMaxTokens = 32000
	videoKitMaxTokens   = 32000
	videoKitCompactNote = "\n\nIMPORTANT: a previous answer was cut off at the output limit. Write a much more compact kit: well under 200 lines, the character as one short SVG with few paths, helpers in a line or two each, no comments except the K member list."
	videoCompactNote    = "\n\nIMPORTANT: a previous answer for this scene was cut off at the output limit. Write a much more compact version: well under 500 lines, build repeated parts with loops and small helper functions, keep SVG paths short, no comments. A complete shorter scene is better than a long unfinished one."
)

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
	if err := s.voiceAndKit(ctx, r); err != nil {
		return err
	}
	if err := s.codeScenes(ctx, r, nil); err != nil {
		return err
	}
	s.done(ctx, r, fmt.Sprintf("完成：%d 个分镜，共 %.0f 秒。可以在右边继续让 AI 修改。", len(p.Spec.Scenes), p.Spec.TotalDuration()))
	s.save(ctx, r)
	note := "初版"
	if earlier, _ := s.repo.ListVersions(ctx, p.ID, 1); len(earlier) > 0 {
		note = "继续生成后"
	}
	_ = s.repo.AddVersion(ctx, p.ID, note, p.Spec)
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
	msgs := videoScriptPrompt(r.p)
	if len(r.p.Options.Images) > 0 {
		msgs[len(msgs)-1].Images = orderedImages(r.p, s.imagesForModel(r.p))
	}
	answer, err := s.chat(ctx, r, msgs, 12000)
	if err != nil {
		return err
	}
	var spec VideoSpec
	var notes struct {
		ImageNotes []struct {
			ID   string `json:"id"`
			Desc string `json:"desc"`
		} `json:"image_notes"`
	}
	raw := extractJSONObject(answer)
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &notes)
	}
	if raw == "" || json.Unmarshal([]byte(raw), &spec) != nil || len(spec.Scenes) == 0 {
		return errLearnRunFailed("模型没有写出可用的脚本，请重试或换一个更强的模型")
	}
	if len(spec.Scenes) > videoMaxScenes {
		spec.Scenes = spec.Scenes[:videoMaxScenes]
	}
	style := findVideoStyle(r.p.Options.Style)
	spec.Theme = style.Theme
	spec.Width, spec.Height = r.p.Width, r.p.Height
	descs := map[string]string{}
	for _, n := range notes.ImageNotes {
		descs[n.ID] = n.Desc
	}
	spec.Images = specImages(r.p, descs)
	known := map[string]bool{}
	for _, img := range spec.Images {
		known[img.ID] = true
	}
	for i := range spec.Scenes {
		sc := &spec.Scenes[i]
		sc.ID = fmt.Sprintf("s%d", i+1)
		var ids []string
		for _, id := range sc.Images {
			if known[id] {
				ids = append(ids, id)
			}
		}
		sc.Images = ids
		sc.Code, sc.Audio, sc.Error = "", nil, ""
		if videoSilent(r.p) {
			// No voice to follow: the script gives each scene's length.
			sc.Narration, sc.Tone = "", ""
			if sc.Duration <= 0 {
				sc.Duration = 5
			}
			sc.Duration = math.Round(math.Min(videoSilentSceneMax, math.Max(videoSilentSceneMin, sc.Duration))*10) / 10
		} else {
			sc.Duration = math.Max(videoMinSceneSeconds, float64(len([]rune(sc.Narration)))/4.5+videoSceneTailPause)
		}
	}
	spec.Title = clipRunes(spec.Title, videoTitleMaxChars)
	spec.Character = clipRunes(strings.TrimSpace(spec.Character), 300)
	var facts []string
	for _, f := range spec.Facts {
		if f = clipRunes(strings.TrimSpace(f), 80); f != "" && len(facts) < 16 {
			facts = append(facts, f)
		}
	}
	spec.Facts, spec.Common = facts, ""
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
	p.Spec = &VideoSpec{Title: p.Title, Theme: style.Theme, Width: p.Width, Height: p.Height, Loop: true, Images: specImages(p, nil),
		Scenes: []VideoScene{{ID: "s1", Title: p.Title, Visual: p.Prompt, Duration: float64(p.Options.Seconds)}}}
	for _, img := range p.Spec.Images {
		p.Spec.Scenes[0].Images = append(p.Spec.Scenes[0].Images, img.ID)
	}
}

func videoAudioName(projectID string, sc VideoScene, voice string, rate int) string {
	key := fmt.Sprintf("%s|%d|%s", voice, rate, sc.Narration)
	if videoVoicePaid(voice) {
		key += "|" + sc.Tone // only the paid voices follow the tone hint
	}
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%s-%s.mp3", sc.ID, hex.EncodeToString(sum[:6]))
}

func (s *VideoService) voiceScenes(ctx context.Context, r *videoRun) error {
	p := r.p
	if p.Mode != VideoModeFilm || p.Spec == nil || videoSilent(p) {
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
	synthesize, err := s.videoTTSFor(ctx, p.UserID, p.Options.Voice)
	if err != nil {
		return err
	}
	paid := videoVoicePaid(p.Options.Voice)
	price := s.Settings(ctx).ttsPrice()
	if paid {
		chars := 0
		for _, i := range todo {
			chars += len([]rune(p.Spec.Scenes[i].Narration))
		}
		need := videoTTSCost(chars, price)
		balance, err := s.ttsBilling.Balance(ctx, p.UserID)
		if err != nil {
			return err
		}
		if balance < need {
			return infraerrors.Forbidden("VIDEO_TTS_BALANCE", fmt.Sprintf("余额不足：这次付费配音 %d 字约需 $%.2f，当前余额 $%.2f。请到 hivegpt.cn 充值，或换成免费音色后点「继续生成」（Insufficient balance for paid voice）", chars, need, balance))
		}
	}
	s.step(ctx, r, "voice", fmt.Sprintf("正在配音（%d 段）…", len(todo)))
	dir := filepath.Join(s.audioDir, p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	charged := 0.0 // guarded by r.mu
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoSceneParallelism)
	for _, i := range todo {
		g.Go(func() error {
			r.mu.Lock()
			sc := p.Spec.Scenes[i]
			r.mu.Unlock()
			speech, err := synthesize(gctx, sc.Narration, p.Options.Voice, p.Options.Rate, sc.Tone)
			if err != nil {
				return err
			}
			chars := len([]rune(sc.Narration))
			cost := 0.0
			if paid {
				// Charged once the audio exists, so a failed recording costs nothing.
				cost = videoTTSCost(chars, price)
				if err := s.ttsBilling.ChargeTTS(context.WithoutCancel(ctx), VideoTTSCharge{UserID: p.UserID, ProjectID: p.ID, SceneID: sc.ID, Voice: p.Options.Voice, Chars: chars, Cost: cost}); err != nil {
					slog.Warn("video: charging paid narration", "id", p.ID, "scene", sc.ID, "err", err)
					cost = 0
				}
			}
			name := videoAudioName(p.ID, sc, p.Options.Voice, p.Options.Rate)
			if err := os.WriteFile(filepath.Join(dir, name), speech.Audio, 0o644); err != nil {
				return err
			}
			r.mu.Lock()
			cur := &p.Spec.Scenes[i]
			cur.Audio = &VideoSceneAudio{File: name, Duration: speech.Duration, Words: speech.Words}
			cur.Duration = math.Max(videoMinSceneSeconds, speech.Duration+videoSceneTailPause)
			p.Usage.TTSChars += chars
			p.Usage.TTSCost += cost
			charged += cost
			r.mu.Unlock()
			return nil
		})
	}
	err = g.Wait()
	if charged > 0 {
		s.afterTTSCharge(p.UserID)
	}
	if err != nil {
		return err
	}
	s.save(ctx, r)
	if charged > 0 {
		s.done(ctx, r, fmt.Sprintf("配音完成，全片 %.0f 秒（豆包配音，本次从余额扣 $%.4f）", p.Spec.TotalDuration(), charged))
	} else {
		s.done(ctx, r, fmt.Sprintf("配音完成，全片 %.0f 秒", p.Spec.TotalDuration()))
	}
	return nil
}

// voiceAndKit records the narration and, for a film whose scenes are all still to be written, writes the
// shared kit at the same time (neither needs the other).
func (s *VideoService) voiceAndKit(ctx context.Context, r *videoRun) error {
	p := r.p
	r.mu.Lock()
	needKit := p.Mode == VideoModeFilm && p.Spec != nil && p.Spec.Common == "" && len(p.Spec.Scenes) > 1
	for _, sc := range p.Spec.Scenes {
		if strings.TrimSpace(sc.Code) != "" {
			needKit = false // an older film being resumed: new scenes must match the ones it has
		}
	}
	r.mu.Unlock()
	if !needKit {
		return s.voiceScenes(ctx, r)
	}
	kitDone := make(chan error, 1)
	go func() { kitDone <- s.writeKit(ctx, r, "") }()
	voiceErr := s.voiceScenes(ctx, r)
	kitErr := <-kitDone
	if voiceErr != nil {
		return voiceErr
	}
	if errors.Is(kitErr, ErrVideoBalance) || ctx.Err() != nil {
		return firstErr(kitErr, ctx.Err())
	}
	if kitErr != nil {
		// Without a kit the scenes style themselves, as films did before kits.
		slog.Warn("video: kit failed, scenes go without it", "id", p.ID, "err", kitErr)
		why := strings.TrimSuffix(strings.TrimPrefix(videoErrorText(kitErr), "运行失败："), "，请稍后再试（Run failed）")
		s.done(ctx, r, "整片统一画风没做成，各分镜各自设计画面（"+clipRunes(why, 80)+"）")
	}
	return nil
}

// writeKit writes the film's shared kit (spec.common), or changes it by instruction.
func (s *VideoService) writeKit(ctx context.Context, r *videoRun, instruction string) error {
	p := r.p
	r.mu.Lock()
	spec := *p.Spec
	spec.Scenes = append([]VideoScene(nil), p.Spec.Scenes...)
	r.mu.Unlock()
	previous := ""
	if instruction != "" {
		previous = spec.Common
		s.step(ctx, r, "kit", "正在修改整片的画面风格…")
	} else {
		s.step(ctx, r, "kit", "正在设计整片统一的画面风格…")
	}
	style := findVideoStyle(p.Options.Style)
	ask := videoKitPrompt(&spec, style.Brief, videoSilent(p), previous, instruction)
	answer, err := s.chat(ctx, r, ask, videoKitMaxTokens)
	if errors.Is(err, errVideoTruncated) {
		ask[len(ask)-1].Content += videoKitCompactNote
		answer, err = s.chat(ctx, r, ask, videoKitMaxTokens)
	}
	if err != nil {
		return err
	}
	code := extractCodeBlock(answer)
	if problem := checkKitCode(code); problem != "" {
		// One chance to fix it, as for scenes.
		msgs := videoKitPrompt(&spec, style.Brief, videoSilent(p), "", "")
		msgs = append(msgs, LearnMessage{Role: "assistant", Content: "```js\n" + code + "\n```"},
			LearnMessage{Role: "user", Content: "This kit does not work: " + problem + "\nFix the cause and return the complete corrected kit."})
		if answer, err = s.chat(ctx, r, msgs, videoKitMaxTokens); err != nil {
			return err
		}
		code = extractCodeBlock(answer)
		if problem = checkKitCode(code); problem != "" {
			return errLearnRunFailed("统一画面风格的代码不可用（" + clipRunes(problem, 80) + "）")
		}
	}
	r.mu.Lock()
	p.Spec.Common = code
	r.mu.Unlock()
	s.save(ctx, r)
	s.done(ctx, r, "整片画面风格已定好")
	return nil
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

var videoKitDeclares = regexp.MustCompile(`\b(?:var|let|const)\s+K\s*=`)

// checkKitCode returns "" when the kit parses, declares K and avoids the network.
func checkKitCode(code string) string {
	if !videoKitDeclares.MatchString(code) {
		return "the kit must declare var K = …"
	}
	return checkSceneCode(code + "\nreturn function update(t) {};")
}

// codeScenes writes the code of every scene without code, and rewrites the scenes in redo with
// their instruction (or failure, prefixed "!error:").
func (s *VideoService) codeScenes(ctx context.Context, r *videoRun, redo map[string]string) error {
	p := r.p
	// A change the user asked for that did not get done last time (the scene failed) is still owed:
	// 继续生成 applies it to the scene's current code instead of rebuilding from the old plan.
	for _, sc := range p.Spec.Scenes {
		if pending := sc.Meta[videoPendingChange]; pending != "" {
			if _, set := redo[sc.ID]; !set {
				if redo == nil {
					redo = map[string]string{}
				}
				redo[sc.ID] = pending
			}
		}
	}
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
	// The designer of a scene sees the pictures it shows (all of them for a single animation).
	pictures := map[string]string{}
	if len(p.Options.Images) > 0 {
		pictures = s.imagesForModel(p)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(videoSceneParallelism)
	var failed []string
	var reason string // the first failure's cause, shown with the list
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
			var imageData []string
			for _, img := range spec.Images {
				if d := pictures[img.ID]; d != "" && (p.Mode == VideoModeMotion || slices.Contains(sc.Images, img.ID)) {
					imageData = append(imageData, d)
				}
			}
			brief := videoSceneBrief{Index: i, Scene: sc, Outline: outline, Spec: &spec, StyleBrief: style.Brief, Category: category.Brief, Motion: p.Mode == VideoModeMotion, Prompt: p.Prompt, ImageData: imageData, Silent: videoSilent(p)}
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
				if errors.Is(err, ErrVideoBalance) {
					return err // stops the other scenes too: each would go further below zero
				}
				slog.Warn("video: scene failed", "id", p.ID, "scene", sc.ID, "err", err)
				failedMu.Lock()
				failed = append(failed, fmt.Sprintf("分镜 %d", i+1))
				if reason == "" {
					reason = strings.TrimSuffix(strings.TrimPrefix(videoErrorText(err), "运行失败："), "，请稍后再试（Run failed）")
				}
				failedMu.Unlock()
				r.mu.Lock()
				p.Spec.Scenes[i].Error = videoErrorText(err)
				if instruction != "" && sc.Code != "" {
					// Keep the scene as it was (still playable) and remember the change for 继续生成.
					if p.Spec.Scenes[i].Meta == nil {
						p.Spec.Scenes[i].Meta = map[string]string{}
					}
					p.Spec.Scenes[i].Meta[videoPendingChange] = instruction
				}
				r.mu.Unlock()
				return nil
			}
			r.mu.Lock()
			p.Spec.Scenes[i].Code, p.Spec.Scenes[i].Error = code, ""
			delete(p.Spec.Scenes[i].Meta, videoPendingChange)
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
		msg := strings.Join(failed, "、") + " 没有做出来"
		if reason != "" {
			msg += "（" + clipRunes(reason, 120) + "）"
		}
		return errLearnRunFailed(msg + "，可以点「继续生成」重试")
	}
	return nil
}

// writeSceneCode asks for the scene's code, checks its syntax and lets the model fix a syntax error once.
func (s *VideoService) writeSceneCode(ctx context.Context, r *videoRun, brief videoSceneBrief, previous, instruction, failure string) (string, error) {
	msgs := videoScenePrompt(brief, previous, instruction, failure)
	answer, err := s.chat(ctx, r, msgs, videoSceneMaxTokens)
	if errors.Is(err, errVideoTruncated) {
		// Too long to finish: ask again for a compact version (the cut-off code is not sent back,
		// and is not "repaired" — that only finishes the same oversized scene).
		msgs[len(msgs)-1].Content += videoCompactNote
		answer, err = s.chat(ctx, r, msgs, videoSceneMaxTokens)
		if errors.Is(err, errVideoTruncated) {
			return "", errVideoSceneTooLong
		}
	}
	if err != nil {
		return "", err
	}
	code := extractCodeBlock(answer)
	if problem := checkSceneWithKit(brief, code); problem != "" {
		answer, err = s.chat(ctx, r, videoScenePrompt(brief, code, "", problem), videoSceneMaxTokens)
		if errors.Is(err, errVideoTruncated) {
			return "", errVideoSceneTooLong
		}
		if err != nil {
			return "", err
		}
		code = extractCodeBlock(answer)
		if problem = checkSceneWithKit(brief, code); problem != "" {
			return "", errLearnRunFailed("动画代码有语法错误（" + clipRunes(problem, 80) + "）")
		}
	}
	return code, nil
}

// checkSceneWithKit checks the scene alone, then together with the film's kit (which runs before it in
// the player), so a scene that re-declares one of the kit's names is caught here.
func checkSceneWithKit(brief videoSceneBrief, code string) string {
	if problem := checkSceneCode(code); problem != "" || brief.Spec.Common == "" || brief.Motion {
		return problem
	}
	if problem := checkSceneCode(brief.Spec.Common + "\n" + code); problem != "" {
		return fmt.Sprintf("together with the shared kit (its %d lines come first, so subtract them from the line number): %s", strings.Count(brief.Spec.Common, "\n")+1, problem)
	}
	return ""
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
	if err := s.checkBalance(ctx, key); err != nil {
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
					if p.Spec.Scenes[i].Meta[videoPendingChange] == "" {
						p.Spec.Scenes[i].Code = "" // rebuilt from the plan; a pending change keeps the code
					}
					p.Spec.Scenes[i].Error = ""
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
	kitChanged := false
	if p.Mode == VideoModeMotion {
		redo[p.Spec.Scenes[0].ID] = text
		_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "assistant", Text: "好的，我来按你的要求修改。"})
	} else {
		s.step(ctx, r, "plan", "正在理解修改意见…")
		answer, err := s.chat(ctx, r, videoRevisionPrompt(p, text, focus, s.earlierRequests(ctx, p.ID, text)), 4000)
		if err != nil {
			return err
		}
		var plan videoRevisionPlan
		if raw := extractJSONObject(answer); raw == "" || json.Unmarshal([]byte(raw), &plan) != nil {
			return errLearnRunFailed("没能理解这次修改，请换个说法再试")
		}
		if kit := strings.TrimSpace(plan.Kit); kit != "" && p.Spec.Common != "" {
			// The whole look lives in the kit: change it once instead of in every scene.
			if err := s.writeKit(ctx, r, kit); err != nil {
				return err
			}
			kitChanged = true
		}
		r.mu.Lock()
		for _, change := range plan.Scenes {
			for i := range p.Spec.Scenes {
				sc := &p.Spec.Scenes[i]
				if sc.ID != change.ID {
					continue
				}
				if n := strings.TrimSpace(change.Narration); n != "" && n != sc.Narration && !videoSilent(p) {
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
			if kitChanged {
				s.done(ctx, r, "修改完成（整片画面风格）")
			}
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
				if n := clipRunes(e.Narration, 1500); n != "" && n != sc.Narration && !videoSilent(p) {
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
	if videoVoicePaid(voice) && s.paidTTS == nil {
		return nil, ErrVideoPaidVoiceOff
	}
	if err := s.checkCloneVoice(ctx, key.UserID, voice); err != nil {
		return nil, err
	}
	return s.startOn(ctx, key, id, func(p *VideoProject) error {
		if p.Spec == nil || p.Mode != VideoModeFilm {
			return ErrVideoNotReady
		}
		if videoSilent(p) || voice == VideoVoiceNone {
			return ErrVideoSilentVoice
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
	// The version's pictures come back too (换图 keeps the old files).
	p.Options.Images = nil
	for _, img := range v.Spec.Images {
		img.URL, img.Desc = "", ""
		p.Options.Images = append(p.Options.Images, img)
	}
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
