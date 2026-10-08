//go:build unit

package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// --- fakes -----------------------------------------------------------------------------------

type fakeVideoRepo struct {
	mu       sync.Mutex
	seq      int
	projects map[string]*VideoProject
	events   map[string][]VideoEvent
	versions map[string][]VideoVersion
}

func newFakeVideoRepo() *fakeVideoRepo {
	return &fakeVideoRepo{projects: map[string]*VideoProject{}, events: map[string][]VideoEvent{}, versions: map[string][]VideoVersion{}}
}

func cloneProject(p *VideoProject) *VideoProject {
	b, _ := json.Marshal(p)
	var c VideoProject
	_ = json.Unmarshal(b, &c)
	c.UserID, c.APIKeyID = p.UserID, p.APIKeyID
	return &c
}

func (r *fakeVideoRepo) Create(_ context.Context, p *VideoProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	p.ID = fmt.Sprintf("00000000-0000-0000-0000-%012d", r.seq)
	p.CreatedAt, p.UpdatedAt = time.Now(), time.Now()
	r.projects[p.ID] = cloneProject(p)
	return nil
}
func (r *fakeVideoRepo) Get(_ context.Context, id string) (*VideoProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.projects[id]
	if p == nil {
		return nil, nil
	}
	return cloneProject(p), nil
}
func (r *fakeVideoRepo) Save(_ context.Context, p *VideoProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.projects[p.ID]
	next := cloneProject(p)
	next.Visibility, next.Category, next.Featured = cur.Visibility, cur.Category, cur.Featured
	r.projects[p.ID] = next
	return nil
}
func (r *fakeVideoRepo) Delete(_ context.Context, userID int64, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p := r.projects[id]; p != nil && p.UserID == userID {
		delete(r.projects, id)
		return true, nil
	}
	return false, nil
}
func (r *fakeVideoRepo) ListByUser(_ context.Context, userID int64, _ int) ([]VideoCard, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []VideoCard
	for _, p := range r.projects {
		if p.UserID == userID {
			out = append(out, VideoCard{ID: p.ID, Title: p.Title, Status: p.Status})
		}
	}
	return out, nil
}
func (r *fakeVideoRepo) CountRunning(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.projects {
		if p.UserID == userID && p.Status == VideoStatusRunning {
			n++
		}
	}
	return n, nil
}
func (r *fakeVideoRepo) FailRunning(_ context.Context, msg string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []string
	for id, p := range r.projects {
		if p.Status == VideoStatusRunning {
			p.Status, p.Error = VideoStatusFailed, msg
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (r *fakeVideoRepo) AddEvent(_ context.Context, id string, e *VideoEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e.ID = int64(len(r.events[id]) + 1)
	r.events[id] = append(r.events[id], *e)
	return nil
}
func (r *fakeVideoRepo) ListEvents(_ context.Context, id string, after int64, _ int) ([]VideoEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []VideoEvent
	for _, e := range r.events[id] {
		if e.ID > after {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *fakeVideoRepo) AddVersion(_ context.Context, id, note string, spec *VideoSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, _ := json.Marshal(spec)
	var c VideoSpec
	_ = json.Unmarshal(b, &c)
	r.versions[id] = append(r.versions[id], VideoVersion{ID: int64(len(r.versions[id]) + 1), Note: note, Spec: &c})
	return nil
}
func (r *fakeVideoRepo) ListVersions(_ context.Context, id string, _ int) ([]VideoVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]VideoVersion(nil), r.versions[id]...), nil
}
func (r *fakeVideoRepo) GetVersion(_ context.Context, id string, vid int64) (*VideoVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range r.versions[id] {
		if v.ID == vid {
			c := v
			return &c, nil
		}
	}
	return nil, nil
}
func (r *fakeVideoRepo) SetVisibility(_ context.Context, id, vis, cat, title string, featured *bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.projects[id]
	p.Visibility, p.Category, p.Title = vis, cat, title
	if featured != nil {
		p.Featured = *featured
	}
	return nil
}
func (r *fakeVideoRepo) Gallery(context.Context, VideoGalleryQuery) ([]VideoCard, int, error) {
	return nil, 0, nil
}
func (r *fakeVideoRepo) Pending(context.Context, int) ([]VideoCard, error) { return nil, nil }
func (r *fakeVideoRepo) AddView(context.Context, string) error             { return nil }
func (r *fakeVideoRepo) AddRemix(context.Context, string) error            { return nil }

type fakeTTS struct {
	mu    sync.Mutex
	calls int
}

func (f *fakeTTS) Synthesize(_ context.Context, text, _ string, _ int) (*VideoSpeech, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return &VideoSpeech{Audio: []byte("mp3:" + text), Duration: float64(len([]rune(text))) / 5, Words: []VideoWord{{Text: string([]rune(text)[:1]), Start: 0.1, End: 0.3}}}, nil
}

// fakeModel answers by prompt kind; broken lists scenes whose first code answer has a syntax error.
type fakeModel struct {
	mu        sync.Mutex
	broken    map[string]int
	calls     []string
	block     chan struct{}
	revisions []string
}

const goodScene = "const el = S.h('div', {text: 'hi'});\nS.root.appendChild(el);\nreturn function update(t) { S.set(el, {opacity: S.p(t, 0, 1)}); };"

func (m *fakeModel) stream(ctx context.Context, _, _ string, msgs []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
	if m.block != nil {
		select {
		case <-m.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	sys := msgs[0].Content
	last := msgs[len(msgs)-1].Content
	var ans string
	m.mu.Lock()
	switch {
	case strings.Contains(sys, "multiple-choice questions"):
		m.calls = append(m.calls, "questions")
		ans = `{"questions":[{"id":"depth","title":"深度","options":[{"label":"入门","recommended":true},{"label":"进阶"}]}]}`
	case strings.Contains(sys, "scriptwriter"):
		m.calls = append(m.calls, "script")
		ans = "```json\n" + `{"title":"测试视频","summary":"s","scenes":[{"title":"一","narration":"第一段旁白。","visual":"v1"},{"title":"二","narration":"第二段旁白。","visual":"v2"}]}` + "\n```"
	case strings.Contains(sys, "The user asks for changes"):
		m.calls = append(m.calls, "plan")
		ans = `{"reply":"好的","scenes":[{"id":"s2","instruction":"更大","narration":"新的第二段旁白。"}]}`
	default:
		scene := "s1"
		if strings.Contains(msgs[1].Content, "Scene s2") {
			scene = "s2"
		}
		m.calls = append(m.calls, "code:"+scene)
		if strings.HasPrefix(last, "Change this scene") {
			m.revisions = append(m.revisions, scene)
		}
		if m.broken[scene] > 0 {
			m.broken[scene]--
			ans = "```js\nreturn function update(t) { if ( }\n```"
		} else {
			ans = "```js\n" + goodScene + "\n```"
		}
	}
	m.mu.Unlock()
	if err := onDelta(ans); err != nil {
		return nil, err
	}
	return &LearnTutorResult{PromptTokens: 10, CompletionTokens: 5}, nil
}

func newTestVideoService(t *testing.T, model *fakeModel) (*VideoService, *fakeVideoRepo, *fakeTTS) {
	repo := newFakeVideoRepo()
	tts := &fakeTTS{}
	svc := newVideoService(repo, tts, model.stream, t.TempDir())
	return svc, repo, tts
}

var videoKeyA = &APIKey{ID: 1, UserID: 7, Key: "sk-a"}

// --- pipeline --------------------------------------------------------------------------------

func TestVideoFilmAsksThenBuilds(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, repo, tts := newTestVideoService(t, model)
	ctx := context.Background()

	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "讲讲区块链", Options: VideoOptions{Ask: true, Style: "tech-blogger"}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusQuestions, got.Status)
	require.Nil(t, got.Spec)

	_, err = svc.Answer(ctx, videoKeyA, p.ID, map[string]string{"深度": "入门"})
	require.NoError(t, err)
	svc.Wait()
	got, _ = svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, "测试视频", got.Title)
	require.Len(t, got.Spec.Scenes, 2)
	require.Equal(t, findVideoStyle("tech-blogger").Theme, got.Spec.Theme)
	for _, sc := range got.Spec.Scenes {
		require.NotEmpty(t, sc.Code)
		require.NotNil(t, sc.Audio)
		require.InDelta(t, math64(videoMinSceneSeconds, sc.Audio.Duration+videoSceneTailPause), sc.Duration, 1e-9)
		_, statErr := os.Stat(filepath.Join(svc.audioDir, p.ID, sc.Audio.File))
		require.NoError(t, statErr)
	}
	require.Equal(t, 2, tts.calls)
	require.Equal(t, 4, got.Usage.Calls) // questions, script, 2 scenes
	require.Equal(t, "入门", got.Options.Answers["深度"])
	require.Len(t, repo.versions[p.ID], 1)

	// Answering twice is refused.
	_, err = svc.Answer(ctx, videoKeyA, p.ID, nil)
	require.ErrorIs(t, err, ErrVideoNoQuestion)
}

func math64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func TestVideoMotionIsOneLoopingScene(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, tts := newTestVideoService(t, model)
	p, err := svc.Create(context.Background(), videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "三个圆点跳动", Options: VideoOptions{Seconds: 6, Ask: true}})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(context.Background(), 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.True(t, got.Spec.Loop)
	require.Len(t, got.Spec.Scenes, 1)
	require.Equal(t, 6.0, got.Spec.Scenes[0].Duration)
	require.Zero(t, tts.calls, "animations have no narration")
	require.Equal(t, []string{"code:s1"}, model.calls, "motion never asks questions")
}

func TestVideoSyntaxErrorIsFixedOnce(t *testing.T) {
	model := &fakeModel{broken: map[string]int{"s2": 1}}
	svc, _, _ := newTestVideoService(t, model)
	p, err := svc.Create(context.Background(), videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(context.Background(), 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	n := 0
	for _, c := range model.calls {
		if c == "code:s2" {
			n++
		}
	}
	require.Equal(t, 2, n, "one retry with the syntax error")
}

func TestVideoBrokenSceneFailsThenResumes(t *testing.T) {
	model := &fakeModel{broken: map[string]int{"s1": 2}}
	svc, _, tts := newTestVideoService(t, model)
	ctx := context.Background()
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusFailed, got.Status)
	require.Contains(t, got.Error, "分镜 1")
	require.NotEmpty(t, got.Spec.Scenes[1].Code, "the other scene is kept")

	_, err = svc.Resume(ctx, videoKeyA, p.ID)
	require.NoError(t, err)
	svc.Wait()
	got, _ = svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, 2, tts.calls, "resuming does not re-record narration")
}

func TestVideoMessageRevisesPlannedScenes(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, repo, tts := newTestVideoService(t, model)
	ctx := context.Background()
	p, _ := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	svc.Wait()
	before, _ := svc.Get(ctx, 7, p.ID)

	_, err := svc.Message(ctx, videoKeyA, p.ID, "第二段字大一点", "s2")
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Equal(t, []string{"s2"}, model.revisions)
	require.Equal(t, "新的第二段旁白。", got.Spec.Scenes[1].Narration)
	require.NotEqual(t, before.Spec.Scenes[1].Audio.File, got.Spec.Scenes[1].Audio.File, "changed narration is re-recorded")
	require.Equal(t, before.Spec.Scenes[0].Audio.File, got.Spec.Scenes[0].Audio.File)
	require.Equal(t, 3, tts.calls)
	require.Len(t, repo.versions[p.ID], 2, "a version is kept before the change")

	// Restoring the version before the change brings the old narration back.
	vs, _ := svc.Versions(ctx, 7, p.ID)
	sort.Slice(vs, func(i, j int) bool { return vs[i].ID > vs[j].ID })
	restored, err := svc.Restore(ctx, 7, p.ID, vs[0].ID)
	require.NoError(t, err)
	require.Equal(t, before.Spec.Scenes[1].Narration, restored.Spec.Scenes[1].Narration)
}

func TestVideoStopAndCapacity(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}, block: make(chan struct{})}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	a, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "a"})
	require.NoError(t, err)
	_, err = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "b"})
	require.NoError(t, err)
	_, err = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "c"})
	require.ErrorIs(t, err, ErrVideoTooMany)
	_, err = svc.Message(ctx, videoKeyA, a.ID, "改", "")
	require.ErrorIs(t, err, ErrVideoBusy)

	require.NoError(t, svc.Stop(ctx, 7, a.ID))
	close(model.block)
	svc.Wait()
	got, _ := svc.Get(ctx, 7, a.ID)
	require.Equal(t, VideoStatusStopped, got.Status)
}

func TestVideoOwnershipPublishReviewAndAudio(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, _, _ := newTestVideoService(t, model)
	ctx := context.Background()
	p, _ := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x"})
	svc.Wait()
	got, _ := svc.Get(ctx, 7, p.ID)
	file := got.Spec.Scenes[0].Audio.File

	_, err := svc.Get(ctx, 8, p.ID)
	require.ErrorIs(t, err, ErrVideoNotFound, "other users cannot open it")
	_, err = svc.AudioPath(ctx, 8, false, p.ID, file)
	require.ErrorIs(t, err, ErrVideoNotFound)
	_, err = svc.AudioPath(ctx, 7, false, p.ID, "../../etc/passwd")
	require.ErrorIs(t, err, ErrVideoNotFound)
	path, err := svc.AudioPath(ctx, 7, false, p.ID, file)
	require.NoError(t, err)
	require.FileExists(t, path)
	_, err = svc.Work(ctx, p.ID)
	require.ErrorIs(t, err, ErrVideoNotFound, "private works are not public")

	pub, err := svc.Publish(ctx, 7, p.ID, "science", "")
	require.NoError(t, err)
	require.Equal(t, VideoVisibilityPending, pub.Visibility)
	_, err = svc.Work(ctx, p.ID)
	require.ErrorIs(t, err, ErrVideoNotFound, "pending works wait for review")

	require.ErrorIs(t, svc.Review(ctx, &User{Role: "user"}, p.ID, "approve", "", nil), ErrVideoForbidden)
	featured := true
	require.NoError(t, svc.Review(ctx, &User{Role: RoleAdmin}, p.ID, "approve", "", &featured))
	w, err := svc.Work(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "science", w.Category)
	require.True(t, w.Featured)
	require.Zero(t, w.Usage.PromptTokens, "usage is hidden from the public")
	_, err = svc.AudioPath(ctx, 0, false, p.ID, file)
	require.NoError(t, err, "anyone may play a public work")
}

func TestVideoConcurrencyLimitIsRetried(t *testing.T) {
	saved := videoRetryBackoff
	videoRetryBackoff = []time.Duration{time.Millisecond, time.Millisecond}
	defer func() { videoRetryBackoff = saved }()
	model := &fakeModel{broken: map[string]int{}}
	refusals := 2
	var mu sync.Mutex
	stream := func(ctx context.Context, key, m string, msgs []LearnMessage, n int, onDelta func(string) error) (*LearnTutorResult, error) {
		mu.Lock()
		refuse := refusals > 0 && strings.Contains(msgs[0].Content, "motion designer")
		if refuse {
			refusals--
		}
		mu.Unlock()
		if refuse {
			return nil, errLearnRunFailed("同时进行的请求数已达你的账号的并发上限，请等其它请求完成后再试（Concurrency limit exceeded for user, please retry later）")
		}
		return model.stream(ctx, key, m, msgs, n, onDelta)
	}
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, stream, t.TempDir())
	p, err := svc.Create(context.Background(), videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "x"})
	require.NoError(t, err)
	svc.Wait()
	got, _ := svc.Get(context.Background(), 7, p.ID)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Zero(t, refusals)

	require.False(t, videoRetryable(errLearnRunFailed("余额不足")))
}

func TestVideoRecoverInterrupted(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	svc, repo, _ := newTestVideoService(t, model)
	repo.projects["x"] = &VideoProject{ID: "x", UserID: 7, Status: VideoStatusRunning}
	svc.RecoverInterrupted(context.Background())
	require.Equal(t, VideoStatusFailed, repo.projects["x"].Status)
	require.Equal(t, "error", repo.events["x"][0].Kind)
}

func TestVideoCreateValidation(t *testing.T) {
	svc, _, _ := newTestVideoService(t, &fakeModel{})
	_, err := svc.Create(context.Background(), videoKeyA, VideoCreateInput{Prompt: "  "})
	require.ErrorIs(t, err, ErrVideoPrompt)
	_, err = svc.Create(context.Background(), nil, VideoCreateInput{Prompt: "x"})
	require.ErrorIs(t, err, ErrVideoKey)
	o := VideoOptions{Ratio: "7:3", Style: "nope", Voice: "evil", Model: "", Rate: 99}
	normalizeVideoOptions("film", &o)
	require.Equal(t, VideoOptions{Ratio: "16:9", Style: "auto", Voice: DefaultVideoVoice, Model: videoDefaultModel, Rate: 30, Length: "standard"}, o)
}

// --- helpers ---------------------------------------------------------------------------------

func TestCheckSceneCode(t *testing.T) {
	require.Empty(t, checkSceneCode(goodScene))
	require.Contains(t, checkSceneCode("return function(t){ if ( }"), "syntax error")
	require.Contains(t, checkSceneCode("const a = 1;"), "return function update")
	require.Contains(t, checkSceneCode("fetch('/x'); return function(){}"), "network")
	require.Contains(t, checkSceneCode("import('x'); return function(){}"), "network")
	require.Contains(t, checkSceneCode(""), "no code")
}

func TestExtractJSONAndCode(t *testing.T) {
	require.Equal(t, `{"a":"}{","b":{"c":1}}`, extractJSONObject("好的：\n```json\n{\"a\":\"}{\",\"b\":{\"c\":1}}\n```"))
	require.Empty(t, extractJSONObject("no json"))
	require.Equal(t, "x();", extractCodeBlock("text\n```js\nx();\n```\nmore"))
	require.Equal(t, "y();", extractCodeBlock("```javascript\ny();\n```"))
	require.Equal(t, "z();", extractCodeBlock("z();"))
}

func TestVideoTitleFromPrompt(t *testing.T) {
	require.Equal(t, "HiveGPT", videoTitleFromPrompt("HiveGPT：六边形蜂巢逐格点亮，最后出现字标和一行小字「一个 Key」"))
	require.Equal(t, "用 60 秒讲清楚区块链为什么不能篡改", videoTitleFromPrompt("用 60 秒讲清楚区块链为什么不能篡改\n第二行"))
	require.Equal(t, "未命名作品", videoTitleFromPrompt("  "))
	long := videoTitleFromPrompt("这是一个非常非常长的描述，里面有很多很多的内容，一直写下去没有尽头的那种描述文字")
	require.LessOrEqual(t, len([]rune(long)), 24)
	require.False(t, strings.HasSuffix(long, "，"), "cut at a clause end, without the comma")
}

func TestSplitSpeechText(t *testing.T) {
	text := strings.Repeat("这是一句话。", 400)
	chunks := splitSpeechText(text, 3000)
	require.Greater(t, len(chunks), 1)
	require.Equal(t, text, strings.Join(chunks, ""))
	for _, c := range chunks {
		require.LessOrEqual(t, len(c), 3000)
	}
}

func TestEdgeSecMSGEC(t *testing.T) {
	at := time.Date(2026, 10, 9, 5, 2, 0, 0, time.UTC)
	a := edgeSecMSGEC(at)
	require.Len(t, a, 64)
	require.Equal(t, a, edgeSecMSGEC(at.Add(2*time.Minute)), "same 5-minute window")
	require.NotEqual(t, a, edgeSecMSGEC(at.Add(5*time.Minute)))
}

// TestEdgeTTSProtocol runs the client against a fake read-aloud websocket.
func TestEdgeTTSProtocol(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var gotSSML string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NotEmpty(t, r.URL.Query().Get("Sec-MS-GEC"))
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() { _ = conn.Close() }()
		_, _, _ = conn.ReadMessage() // speech.config
		_, ssml, _ := conn.ReadMessage()
		gotSSML = string(ssml)
		meta := `{"Metadata":[{"Type":"WordBoundary","Data":{"Offset":1000000,"Duration":4000000,"text":{"Text":"你好"}}}]}`
		_ = conn.WriteMessage(websocket.TextMessage, []byte("X-RequestId:1\r\nPath:audio.metadata\r\n\r\n"+meta))
		head := "X-RequestId:1\r\nContent-Type:audio/mpeg\r\nPath:audio\r\n"
		frame := make([]byte, 2)
		binary.BigEndian.PutUint16(frame, uint16(len(head)))
		frame = append(frame, head...)
		frame = append(frame, make([]byte, 6000)...) // 6000 bytes at 48 kbit/s = 1 s
		_ = conn.WriteMessage(websocket.BinaryMessage, frame)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("X-RequestId:1\r\nPath:turn.end\r\n\r\n{}"))
	}))
	defer srv.Close()
	e := NewEdgeTTS()
	e.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	sp, err := e.Synthesize(context.Background(), "你好 <b>&", "zh-CN-YunxiNeural", 10)
	require.NoError(t, err)
	require.Len(t, sp.Audio, 6000)
	require.InDelta(t, 1.0, sp.Duration, 1e-9)
	require.Equal(t, []VideoWord{{Text: "你好", Start: 0.1, End: 0.5}}, sp.Words)
	require.Contains(t, gotSSML, "voice name='zh-CN-YunxiNeural'")
	require.Contains(t, gotSSML, "rate='+10%'")
	require.Contains(t, gotSSML, "你好 &lt;b&gt;&amp;", "narration is XML-escaped")
}
