//go:build unit

package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// --- fakes ---

type fakeArchiveObject struct {
	body            []byte
	contentType     string
	contentEncoding string
	sse             bool
}

type fakeArchiveStore struct {
	mu        sync.Mutex
	objects   map[string]fakeArchiveObject
	deleted   [][]string
	prefixes  []string
	lifecycle []string
	putErr    error
	putCalls  int
	deleteErr error
}

func newFakeArchiveStore() *fakeArchiveStore {
	return &fakeArchiveStore{objects: map[string]fakeArchiveObject{}}
}

func (f *fakeArchiveStore) PutObject(_ context.Context, key string, body []byte, contentType, contentEncoding string, sse bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putCalls++
	if f.putErr != nil {
		return f.putErr
	}
	f.objects[key] = fakeArchiveObject{body: append([]byte(nil), body...), contentType: contentType, contentEncoding: contentEncoding, sse: sse}
	return nil
}

func (f *fakeArchiveStore) GetObject(_ context.Context, key string) (io.ReadCloser, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	obj, ok := f.objects[key]
	if !ok {
		return nil, 0, errors.New("NoSuchKey")
	}
	return io.NopCloser(bytes.NewReader(obj.body)), int64(len(obj.body)), nil
}

func (f *fakeArchiveStore) DeleteObjects(_ context.Context, keys []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if len(keys) > 1000 {
		return errors.New("too many keys")
	}
	f.deleted = append(f.deleted, append([]string(nil), keys...))
	for _, k := range keys {
		delete(f.objects, k)
	}
	return nil
}

func (f *fakeArchiveStore) DeletePrefix(_ context.Context, prefix string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prefixes = append(f.prefixes, prefix)
	n := 0
	for k := range f.objects {
		if strings.HasPrefix(k, prefix) {
			delete(f.objects, k)
			n++
		}
	}
	return n, nil
}

func (f *fakeArchiveStore) ApplyLifecycle(_ context.Context, ruleID, prefix string, days int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lifecycle = append(f.lifecycle, ruleID+"|"+prefix+"|"+time.Duration(days).String())
	return nil
}

func (f *fakeArchiveStore) keys(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for k := range f.objects {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func (f *fakeArchiveStore) object(key string) fakeArchiveObject {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.objects[key]
}

type fakeArchiveRepo struct {
	mu        sync.Mutex
	raw       string
	access    []ContentLogAccess
	accessErr error
	turns     []ArchiveTurn
	images    []ArchiveImage
	nextID    int64
	expired   []ArchiveExpiredTurn
	deleted   []int64
	resolve   map[string]int64
	sessions  []ArchiveSession
	lastSF    ArchiveSessionFilter
	insertErr error
	purges    map[int64]ArchivePurge
	afterGet  func()
}

func (r *fakeArchiveRepo) QueuePurge(_ context.Context, userID int64, prefixes []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.purges == nil {
		r.purges = map[int64]ArchivePurge{}
	}
	r.purges[userID] = ArchivePurge{UserID: userID, Prefixes: prefixes, CreatedAt: time.Now()}
	return nil
}

func (r *fakeArchiveRepo) ListPurges(context.Context, int) ([]ArchivePurge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ArchivePurge
	for _, p := range r.purges {
		out = append(out, p)
	}
	return out, nil
}

func (r *fakeArchiveRepo) FinishPurge(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.purges, userID)
	return nil
}

func (r *fakeArchiveRepo) FailPurge(context.Context, int64, string) error { return nil }

func (r *fakeArchiveRepo) GetConfigRaw(context.Context) (string, error) {
	r.mu.Lock()
	raw, hook := r.raw, r.afterGet
	r.afterGet = nil
	r.mu.Unlock()
	if hook != nil {
		hook()
	}
	return raw, nil
}

func (r *fakeArchiveRepo) SaveConfig(_ context.Context, raw string, audit ContentLogAccess) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.accessErr != nil {
		return r.accessErr
	}
	r.raw = raw
	r.access = append(r.access, audit)
	return nil
}

func (r *fakeArchiveRepo) InsertAccess(_ context.Context, a ContentLogAccess) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.accessErr != nil {
		return r.accessErr
	}
	r.access = append(r.access, a)
	return nil
}

func (r *fakeArchiveRepo) InsertTurn(_ context.Context, t *ArchiveTurn, images []ArchiveImage) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.insertErr != nil {
		return 0, r.insertErr
	}
	r.nextID++
	t.ID = r.nextID
	r.turns = append(r.turns, *t)
	for _, img := range images {
		r.nextID++
		img.ID = r.nextID
		img.TurnID = t.ID
		r.images = append(r.images, img)
	}
	return t.ID, nil
}

func (r *fakeArchiveRepo) ListSessions(_ context.Context, f ArchiveSessionFilter) ([]ArchiveSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSF = f
	return r.sessions, nil
}

func (r *fakeArchiveRepo) ListTurns(_ context.Context, userID int64, sessionKey string, beforeID int64, limit int) ([]ArchiveTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ArchiveTurn
	for i := len(r.turns) - 1; i >= 0; i-- {
		t := r.turns[i]
		if t.UserID == userID && t.SessionKey == sessionKey && (beforeID == 0 || t.ID < beforeID) && len(out) < limit {
			out = append(out, t)
		}
	}
	return out, nil
}

func (r *fakeArchiveRepo) GetTurnByRequestID(_ context.Context, requestID string) (*ArchiveTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.turns {
		if t.RequestID == requestID {
			t.Email = "u@test"
			return &t, nil
		}
	}
	return nil, nil
}

func (r *fakeArchiveRepo) ListTurnImages(_ context.Context, ids []int64) (map[int64][]ArchiveImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[int64][]ArchiveImage{}
	for _, img := range r.images {
		for _, id := range ids {
			if img.TurnID == id {
				out[id] = append(out[id], img)
			}
		}
	}
	return out, nil
}

func (r *fakeArchiveRepo) GetImage(_ context.Context, id int64) (*ArchiveImage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, img := range r.images {
		if img.ID == id {
			return &img, nil
		}
	}
	return nil, nil
}

func (r *fakeArchiveRepo) DailyStats(context.Context, time.Time, string) ([]ArchiveDayStats, error) {
	return nil, nil
}

func (r *fakeArchiveRepo) ListExpired(_ context.Context, _ time.Time, limit int) ([]ArchiveExpiredTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := min(limit, len(r.expired))
	return append([]ArchiveExpiredTurn(nil), r.expired[:n]...), nil
}

func (r *fakeArchiveRepo) DeleteTurns(_ context.Context, ids []int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, ids...)
	drop := map[int64]bool{}
	for _, id := range ids {
		drop[id] = true
	}
	var rest []ArchiveExpiredTurn
	for _, t := range r.expired {
		if !drop[t.ID] {
			rest = append(rest, t)
		}
	}
	r.expired = rest
	return int64(len(ids)), nil
}

func (r *fakeArchiveRepo) ResolveUserID(_ context.Context, q string) (int64, error) {
	return r.resolve[q], nil
}

type fakeArchiveDedupe struct {
	mu      sync.Mutex
	states  map[string]*ArchiveDedupeState
	loadErr error
}

func (d *fakeArchiveDedupe) Load(_ context.Context, key string) (*ArchiveDedupeState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.loadErr != nil {
		return nil, d.loadErr
	}
	return d.states[key], nil
}

func (d *fakeArchiveDedupe) Save(_ context.Context, key string, st *ArchiveDedupeState, _ time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.loadErr != nil {
		return d.loadErr
	}
	d.states[key] = st
	return nil
}

type archivePrefixEncryptor struct{}

func (archivePrefixEncryptor) Encrypt(p string) (string, error) { return "enc:" + p, nil }
func (archivePrefixEncryptor) Decrypt(c string) (string, error) {
	if !strings.HasPrefix(c, "enc:") {
		return "", errors.New("not encrypted")
	}
	return strings.TrimPrefix(c, "enc:"), nil
}

type archiveHarness struct {
	svc     *ConversationArchiveService
	repo    *fakeArchiveRepo
	store   *fakeArchiveStore
	dedupe  *fakeArchiveDedupe
	configs []ArchiveStoreConfig
}

func newArchiveHarness(t *testing.T) *archiveHarness {
	t.Helper()
	h := &archiveHarness{repo: &fakeArchiveRepo{resolve: map[string]int64{}}, store: newFakeArchiveStore(), dedupe: &fakeArchiveDedupe{states: map[string]*ArchiveDedupeState{}}}
	factory := func(_ context.Context, cfg ArchiveStoreConfig) (ArchiveObjectStore, error) {
		h.configs = append(h.configs, cfg)
		return h.store, nil
	}
	h.svc = NewConversationArchiveService(h.repo, factory, archivePrefixEncryptor{}, true, h.dedupe)
	h.svc.sleep = func(context.Context, time.Duration) bool { return true }
	t.Cleanup(func() {
		h.svc.Stop()
		archiveCaptureOn.Store(false)
	})
	return h
}

func (h *archiveHarness) enable(t *testing.T) {
	t.Helper()
	_, err := h.svc.UpdateConfig(context.Background(), 1, ArchiveConfigInput{
		Enabled: true, Endpoint: "https://cos.ap-seoul.myqcloud.com", Region: "ap-seoul", Bucket: "b-123",
		AccessKeyID: "AK", SecretAccessKey: "SECRET",
	}, "1.2.3.4", "ua")
	require.NoError(t, err)
	require.True(t, h.svc.Enabled())
	require.True(t, ArchiveCaptureEnabled())
}

// drain processes everything queued, synchronously.
func (h *archiveHarness) drain() {
	for _, q := range h.svc.queues {
		for {
			select {
			case job := <-q:
				h.svc.process(job)
				h.svc.release(job)
				continue
			default:
			}
			break
		}
	}
}

func readArchiveRecord(t *testing.T, obj fakeArchiveObject) map[string]any {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(obj.body))
	require.NoError(t, err)
	raw, err := io.ReadAll(zr)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func archiveField(t *testing.T, rec map[string]any, key string) map[string]any {
	t.Helper()
	m, ok := rec[key].(map[string]any)
	require.True(t, ok, key)
	return m
}

func archiveUsage(requestID, session string) *UsageLog {
	s := session
	ep := "/v1/responses"
	ua := "codex_cli_rs/0.1"
	ip := "9.9.9.9"
	g := int64(2)
	return &UsageLog{UserID: 45, APIKeyID: 35, GroupID: &g, RequestID: requestID, RequestedModel: "gpt-6.1-sol", SessionID: &s,
		InboundEndpoint: &ep, UserAgent: &ua, IPAddress: &ip, InputTokens: 10, OutputTokens: 5, CacheReadTokens: 3, ActualCost: 0.01,
		CreatedAt: time.Date(2026, 10, 16, 3, 4, 5, 0, time.UTC)}
}

// --- dedupe ---

func archiveParts(items ...string) archiveRequestParts {
	p := archiveRequestParts{Params: map[string]json.RawMessage{}}
	for _, it := range items {
		p.Items = append(p.Items, json.RawMessage(it))
	}
	return p
}

func archiveHashes(p archiveRequestParts) []string {
	out := make([]string, len(p.Items))
	for i, it := range p.Items {
		out[i] = archiveItemHash(it)
	}
	return out
}

func TestArchiveDedupe_PrefixResetInstructionsAndRedisDown(t *testing.T) {
	first := archiveParts(`{"role":"user","content":"a"}`)
	first.HasInstructions, first.Instructions = true, "sys"
	res := computeArchiveDedupe(nil, nil, first, archiveHashes(first), "i1")
	require.False(t, res.Reset)
	require.Equal(t, 0, res.PrefixItems)
	require.Len(t, res.NewItems, 1)
	require.True(t, res.IncludeInstructions)

	// Resend with the history (whitespace differs) plus two new items: only the new ones.
	second := archiveParts(`{ "role": "user", "content": "a" }`, `{"role":"assistant","content":"b"}`, `{"role":"user","content":"c"}`)
	second.HasInstructions, second.Instructions = true, "sys"
	res2 := computeArchiveDedupe(res.Next, nil, second, archiveHashes(second), "i1")
	require.False(t, res2.Reset)
	require.Equal(t, 1, res2.PrefixItems)
	require.Len(t, res2.NewItems, 2)
	require.False(t, res2.IncludeInstructions, "unchanged instructions are omitted")

	// Changed instructions are included again.
	res3 := computeArchiveDedupe(res2.Next, nil, second, archiveHashes(second), "i2")
	require.True(t, res3.IncludeInstructions)
	require.Len(t, res3.NewItems, 0)

	// An edited history is a reset: everything is archived.
	edited := archiveParts(`{"role":"user","content":"A!"}`, `{"role":"user","content":"d"}`)
	res4 := computeArchiveDedupe(res3.Next, nil, edited, archiveHashes(edited), "i2")
	require.True(t, res4.Reset)
	require.Equal(t, 0, res4.PrefixItems)
	require.Len(t, res4.NewItems, 2)

	// previous_response_id: the input is already only the new items.
	chained := archiveParts(`{"role":"user","content":"next"}`)
	chained.PreviousResponseID = "resp_1"
	res5 := computeArchiveDedupe(res4.Next, nil, chained, archiveHashes(chained), "i2")
	require.False(t, res5.Reset)
	require.Len(t, res5.NewItems, 1)
	require.Equal(t, res4.Next.ItemHashes, res5.Next.ItemHashes)

	// Redis down: the full input, marked reset.
	res6 := computeArchiveDedupe(nil, errors.New("down"), second, archiveHashes(second), "i1")
	require.True(t, res6.Reset)
	require.Len(t, res6.NewItems, 3)
	require.True(t, res6.IncludeInstructions)
}

func TestArchiveDedupe_HashCap(t *testing.T) {
	var items []string
	for i := 0; i < archiveDedupeMaxItems+10; i++ {
		items = append(items, `{"i":`+itoa(i)+`}`)
	}
	p := archiveParts(items...)
	res := computeArchiveDedupe(nil, nil, p, archiveHashes(p), "")
	require.Len(t, res.Next.ItemHashes, archiveDedupeMaxItems)
	require.Equal(t, 10, res.Next.Offset)
	// The next turn still matches past the dropped hashes: no perpetual reset.
	p2 := archiveParts(append(items, `{"new":1}`)...)
	res2 := computeArchiveDedupe(res.Next, nil, p2, archiveHashes(p2), "")
	require.False(t, res2.Reset)
	require.Equal(t, archiveDedupeMaxItems+10, res2.PrefixItems)
	require.Len(t, res2.NewItems, 1)
	require.Equal(t, 11, res2.Next.Offset)
	// A shorter (compacted) history than the offset is a reset.
	p3 := archiveParts(`{"summary":1}`)
	require.True(t, computeArchiveDedupe(res2.Next, nil, p3, archiveHashes(p3), "").Reset)
}

func itoa(i int) string { return strconv.Itoa(i) }

// --- extraction ---

func TestExtractArchiveRequest_Protocols(t *testing.T) {
	p := extractArchiveRequest(ArchiveProtocolResponses, []byte(`{"model":"m","instructions":"be brief","input":"hi","previous_response_id":"r1","tools":[{"type":"function","name":"shell","parameters":{"x":1}},{"type":"web_search"}],"temperature":0.2}`), "")
	require.Len(t, p.Items, 1)
	require.JSONEq(t, `{"role":"user","content":"hi"}`, string(p.Items[0]))
	require.True(t, p.HasInstructions)
	require.Equal(t, "be brief", p.Instructions)
	require.Equal(t, "r1", p.PreviousResponseID)
	require.JSONEq(t, `["shell","web_search"]`, string(p.Params["tools"]))
	require.Equal(t, "0.2", string(p.Params["temperature"]))

	p = extractArchiveRequest(ArchiveProtocolResponses, []byte(`{"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"x"}]},{"type":"function_call_output","call_id":"c","output":"ok"}]}`), "")
	require.Len(t, p.Items, 2)
	require.False(t, p.HasInstructions)

	p = extractArchiveRequest(ArchiveProtocolChat, []byte(`{"messages":[{"role":"system","content":"S1"},{"role":"developer","content":[{"type":"text","text":"S2"}]},{"role":"user","content":"q"}],"tools":[{"type":"function","function":{"name":"f"}}]}`), "")
	require.Len(t, p.Items, 1)
	require.Equal(t, "S1\n\nS2", p.Instructions)
	require.JSONEq(t, `["f"]`, string(p.Params["tools"]))

	p = extractArchiveRequest(ArchiveProtocolAnthropic, []byte(`{"system":[{"type":"text","text":"You are Claude Code"},{"type":"text","text":"more"}],"messages":[{"role":"user","content":"a"},{"role":"assistant","content":"b"}],"thinking":{"type":"enabled"}}`), "")
	require.Len(t, p.Items, 2)
	require.Equal(t, "You are Claude Code\n\nmore", p.Instructions)
	require.Contains(t, p.Params, "thinking")

	p = extractArchiveRequest(ArchiveProtocolGemini, []byte(`{"systemInstruction":{"parts":[{"text":"G"}]},"contents":[{"role":"user","parts":[{"text":"hello"}]}],"tools":[{"functionDeclarations":[{"name":"fd"}]},{"googleSearch":{}}],"generationConfig":{"temperature":1,"responseSchema":{}}}`), "")
	require.Len(t, p.Items, 1)
	require.Equal(t, "G", p.Instructions)
	require.JSONEq(t, `["fd","googleSearch"]`, string(p.Params["tools"]))
	require.Contains(t, p.Params, "temperature")
	require.NotContains(t, p.Params, "responseSchema")

	p = extractArchiveRequest(ArchiveProtocolImages, []byte(`{"model":"gpt-image-2","prompt":"a cat","size":"1024x1024","image":"data:image/png;base64,AAAA"}`), "")
	require.Len(t, p.Items, 1)
	require.JSONEq(t, `{"role":"user","content":"a cat"}`, string(p.Items[0]))
	require.Contains(t, p.Params, "size")
	require.NotContains(t, p.Params, "image")

	p = extractArchiveRequest(ArchiveProtocolImages, nil, "edit this")
	require.Len(t, p.Items, 1)

	// Garbage never panics.
	p = extractArchiveRequest(ArchiveProtocolChat, []byte(`not json`), "")
	require.Empty(t, p.Items)
}

func TestArchiveCanonicalJSON(t *testing.T) {
	a := archiveCanonicalJSON([]byte(` {"role":"user", "content":[{"type":"text","text":"a b","cache_control":{"type":"ephemeral"}}]} `))
	b := archiveCanonicalJSON([]byte(`{"role":"user","content":[{"type":"text","text":"a b"}]}`))
	require.Equal(t, b, a)
	require.Equal(t, `{"role":"user","content":[{"type":"text","text":"a b"}]}`, a)
	require.NotEqual(t, archiveItemHash(json.RawMessage(`{"a":1}`)), archiveItemHash(json.RawMessage(`{"a":2}`)))
}

func TestArchiveSessionKeyAndPreview(t *testing.T) {
	items := []json.RawMessage{json.RawMessage(`{"role":"user","content":"first"}`)}
	require.Equal(t, "sess-1", archiveSessionKey(" sess-1 ", 35, items, "r"))
	k1 := archiveSessionKey("", 35, items, "r1")
	k2 := archiveSessionKey("", 35, []json.RawMessage{json.RawMessage(`{ "role":"user", "content":"first" }`)}, "r2")
	require.True(t, strings.HasPrefix(k1, "h:"))
	require.Len(t, k1, 26)
	require.Equal(t, k1, k2, "chat clients without a session id share their first message")
	require.NotEqual(t, k1, archiveSessionKey("", 36, items, "r1"))

	preview := archiveLatestUserText(ArchiveProtocolResponses, []json.RawMessage{
		json.RawMessage(`{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>cwd</environment_context>"},{"type":"input_text","text":"fix the bug"}]}`),
		json.RawMessage(`{"type":"function_call_output","call_id":"c","output":"x"}`),
	})
	require.Equal(t, "fix the bug", preview)
	require.Equal(t, "hello", archiveLatestUserText(ArchiveProtocolGemini, []json.RawMessage{json.RawMessage(`{"role":"user","parts":[{"text":"hello"}]}`)}))
	long := archiveLatestUserText(ArchiveProtocolChat, []json.RawMessage{json.RawMessage(`{"role":"user","content":"` + strings.Repeat("字", 500) + `"}`)})
	require.Equal(t, 200, len([]rune(long)))
}

// --- capture (accumulator archive mode) ---

func newArchiveCaptureContext(t *testing.T, path string, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	archiveCaptureOn.Store(true)
	t.Cleanup(func() { archiveCaptureOn.Store(false) })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	SetArchiveRequestBody(c, []byte(body))
	return c
}

func TestArchiveCapture_OffCostsNothing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	archiveCaptureOn.Store(false)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	SetArchiveRequestBody(c, []byte(`{"input":"x"}`))
	SetArchiveRequestPrompt(c, "x")
	require.Nil(t, archiveRequestFromContext(c))
	require.Nil(t, beginUpstreamResponseModelObservation(c).content)
	require.Nil(t, ContentCaptureSnapshot(c))
}

func TestArchiveCapture_ResponsesStreamWithImages(t *testing.T) {
	body := `{"model":"gpt-6.1-sol","input":"draw"}`
	c := newArchiveCaptureContext(t, "/v1/responses", body)
	o := beginUpstreamResponseModelObservation(c)
	require.NotNil(t, o.content)
	require.NotNil(t, o.content.arch)
	img := base64.StdEncoding.EncodeToString(archivePNG())
	for _, ev := range []struct{ typ, data string }{
		{"response.output_text.delta", `{"type":"response.output_text.delta","delta":"Here "}`},
		{"response.output_text.delta", `{"type":"response.output_text.delta","delta":"you go"}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"think"}],"encrypted_content":"SECRETBLOB"}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"function_call","name":"shell","call_id":"call_1","arguments":"{\"cmd\":\"ls\"}"}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"image_generation_call","id":"ig_1","output_format":"png","result":"` + img + `"}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Here you go"}]}}`},
		{"response.completed", `{"type":"response.completed","response":{"status":"completed","output":[{"type":"image_generation_call","result":"` + img + `"}]}}`},
	} {
		o.ObserveOpenAI([]byte(ev.data), ev.typ)
	}
	snap := ContentCaptureSnapshot(c)
	require.NotNil(t, snap)
	require.True(t, snap.archiveOnly)
	require.Empty(t, snap.ReplyHead, "not a content-log target: no content-log fields")
	a := snap.Archive
	require.NotNil(t, a)
	require.Equal(t, ArchiveProtocolResponses, a.Protocol)
	require.Equal(t, body, string(a.RequestBody))
	require.Equal(t, "completed", a.Response.Status)
	require.Equal(t, "Here you go", a.Response.Text)
	require.Contains(t, a.Response.ToolCalls, ArchiveToolCall{Name: "shell", CallID: "call_1", Arguments: `{"cmd":"ls"}`})
	require.Len(t, a.Response.Items, 4)
	require.NotContains(t, string(a.Response.Items[0]), "SECRETBLOB")
	require.Equal(t, "think", gjson.GetBytes(a.Response.Items[0], "summary.0.text").String())
	require.Equal(t, "", gjson.GetBytes(a.Response.Items[2], "result").String())
	require.Equal(t, int64(0), gjson.GetBytes(a.Response.Items[2], "archived_image").Int())
	require.Len(t, a.Images, 1, "the terminal event repeats the image; it is not taken twice")
	require.Equal(t, archiveImageSourceImageGeneration, a.Images[0].Source)
	require.False(t, a.ResponseTruncated)
}

func TestArchiveCapture_ChatAnthropicAndBuffered(t *testing.T) {
	c := newArchiveCaptureContext(t, "/v1/chat/completions", `{"messages":[{"role":"user","content":"q"}]}`)
	o := beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"choices":[{"index":0,"delta":{"content":"Hel"}}]}`), "")
	o.ObserveOpenAI([]byte(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_9","function":{"name":"get","arguments":"{\"a\""}}]}}]}`), "")
	o.ObserveOpenAI([]byte(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":1}"}}]},"finish_reason":"length"}]}`), "")
	a := ContentCaptureSnapshot(c).Archive
	require.Equal(t, "Hel", a.Response.Text)
	require.Equal(t, []ArchiveToolCall{{Name: "get", CallID: "call_9", Arguments: `{"a":1}`}}, a.Response.ToolCalls)
	require.Equal(t, "incomplete", a.Response.Status)

	c = newArchiveCaptureContext(t, "/v1/messages", `{"messages":[{"role":"user","content":"q"}]}`)
	o = beginUpstreamResponseModelObservation(c)
	forEachSSEData(anthropicToolSSE(), func(data string) { o.ObserveAnthropic([]byte(data)) })
	a = ContentCaptureSnapshot(c).Archive
	require.Equal(t, ArchiveProtocolAnthropic, a.Protocol)
	require.Equal(t, "Running it.", a.Response.Text)
	require.Equal(t, []ArchiveToolCall{{Name: "Bash", CallID: "t1", Arguments: `{"command":"ls"}`}}, a.Response.ToolCalls)
	require.Equal(t, "completed", a.Response.Status)

	// Non-stream Anthropic body keeps its blocks as items (thinking signature dropped).
	c = newArchiveCaptureContext(t, "/v1/messages", `{"messages":[]}`)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveAnthropic([]byte(`{"type":"message","stop_reason":"end_turn","content":[{"type":"thinking","thinking":"hmm","signature":"SIG"},{"type":"text","text":"done"}]}`))
	a = ContentCaptureSnapshot(c).Archive
	require.Len(t, a.Response.Items, 2)
	require.NotContains(t, string(a.Response.Items[0]), "SIG")
	require.Equal(t, "done", a.Response.Text)

	// Buffered paths hand over the terminal Responses object.
	c = newArchiveCaptureContext(t, "/v1/chat/completions", `{"messages":[]}`)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveContentFinalResponse(map[string]any{"status": "incomplete", "output": []any{
		map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": "buffered"}}},
	}})
	a = ContentCaptureSnapshot(c).Archive
	require.Equal(t, "buffered", a.Response.Text)
	require.Equal(t, "incomplete", a.Response.Status)
	require.Len(t, a.Response.Items, 1)
}

func forEachSSEData(stream string, fn func(string)) {
	for _, line := range strings.Split(stream, "\n") {
		if strings.HasPrefix(line, "data: ") {
			fn(strings.TrimPrefix(line, "data: "))
		}
	}
}

func TestArchiveCapture_CapsAndContentLogTrim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	archiveCaptureOn.Store(true)
	t.Cleanup(func() { archiveCaptureOn.Store(false) })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	SetContentCaptureRequest(c, "user asks") // also a content-log target
	SetArchiveRequestBody(c, []byte(`{"input":"x"}`))
	o := beginUpstreamResponseModelObservation(c)
	big := strings.Repeat("a", 100<<10)
	for i := 0; i < 4; i++ {
		o.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"`+big+`"}`), "response.output_text.delta")
	}
	args := strings.Repeat("b", 100<<10)
	for i := 0; i < 15; i++ {
		o.ObserveOpenAI([]byte(`{"type":"response.output_item.done","item":{"type":"function_call","name":"f","arguments":"`+args+`"}}`), "response.output_item.done")
	}
	snap := ContentCaptureSnapshot(c)
	require.False(t, snap.archiveOnly)
	// Content log keeps its own sizes.
	require.LessOrEqual(t, len(snap.ReplyHead), contentCaptureReplyMaxBytes)
	for _, call := range snap.ToolCalls {
		require.LessOrEqual(t, len(call.Arguments), contentCaptureToolArgsMaxBytes)
	}
	// The archive keeps the larger caps and says it was cut.
	a := snap.Archive
	require.Equal(t, archiveResponseTextMaxBytes, len(a.Response.Text))
	require.Len(t, a.Response.ToolCalls, 15)
	require.Equal(t, archiveToolArgsMaxBytes, len(a.Response.ToolCalls[0].Arguments))
	total := 0
	for _, it := range a.Response.Items {
		total += len(it)
	}
	require.LessOrEqual(t, total, archiveResponseItemsMax)
	require.True(t, a.ResponseTruncated)
}

func TestArchiveCapture_ImagesEndpointPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	archiveCaptureOn.Store(true)
	t.Cleanup(func() { archiveCaptureOn.Store(false) })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
	SetArchiveRequestPrompt(c, "make it blue")
	img := base64.StdEncoding.EncodeToString(archivePNG())
	ObserveArchiveImagesPayload(c, []byte(`{"type":"image_edit.partial_image","b64_json":"`+img+`"}`))
	ObserveArchiveImagesPayload(c, []byte(`{"type":"image_edit.completed","b64_json":"`+img+`"}`))
	ObserveArchiveImagesPayload(c, []byte(`{"data":[{"b64_json":"`+img+`"},{"url":"https://img.example/x.png"}]}`))
	snap := ContentCaptureSnapshot(c)
	require.NotNil(t, snap)
	a := snap.Archive
	require.Equal(t, ArchiveProtocolImages, a.Protocol)
	require.Equal(t, "make it blue", a.Prompt)
	require.Len(t, a.Images, 2)
	require.Equal(t, []string{"https://img.example/x.png"}, a.ImageURLs)

	// Per-request image budget.
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	SetArchiveRequestBody(c, []byte(`{"prompt":"p"}`))
	huge := strings.Repeat("A", 20<<20)
	ObserveArchiveImagesPayload(c, []byte(`{"data":[{"b64_json":"`+huge+`"},{"b64_json":"`+huge+`"}]}`))
	a = ContentCaptureSnapshot(c).Archive
	require.Len(t, a.Images, 1)
	require.True(t, a.ResponseTruncated)

	// Not archived: nothing collected.
	archiveCaptureOn.Store(false)
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	ObserveArchiveImagesPayload(c, []byte(`{"data":[{"b64_json":"`+img+`"}]}`))
	require.Nil(t, ContentCaptureSnapshot(c))
}

func archivePNG() []byte {
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")
}

// --- service end to end (fake store / repo / dedupe) ---

func TestConversationArchive_RecordGzipDedupeAndIndex(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	require.Equal(t, "SECRET", h.configs[len(h.configs)-1].SecretAccessKey, "the store gets the decrypted secret")

	img := base64.StdEncoding.EncodeToString(archivePNG())
	capture1 := &ArchiveCapture{Protocol: ArchiveProtocolResponses,
		RequestBody: []byte(`{"instructions":"sys","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}],"tools":[{"type":"function","name":"shell"}]}`),
		Response:    ArchiveResponse{Status: "completed", Text: "hi", ToolCalls: []ArchiveToolCall{}, Items: []json.RawMessage{json.RawMessage(`{"type":"message"}`)}},
		Images:      []ArchiveImagePayload{{B64: img, Source: archiveImageSourceImageGeneration}},
	}
	h.svc.EnqueueArchive(archiveUsage("client:r1", "sess-A"), capture1)
	require.Equal(t, int64(1), h.svc.Status().QueueLen)
	h.drain()
	require.Equal(t, int64(0), h.svc.Status().QueueLen)
	require.Equal(t, int64(0), h.svc.Status().QueueBytes)

	turnKeys := h.store.keys("archive/conv/45/")
	require.Len(t, turnKeys, 1)
	require.Regexp(t, `^archive/conv/45/\d{8}/[0-9a-f]{16}/\d+_client_r1\.json\.gz$`, turnKeys[0])
	obj := h.store.object(turnKeys[0])
	require.Equal(t, "application/json", obj.contentType)
	require.Equal(t, "gzip", obj.contentEncoding)
	require.True(t, obj.sse)
	rec := readArchiveRecord(t, obj)
	require.EqualValues(t, 1, rec["v"])
	require.Equal(t, "client:r1", rec["request_id"])
	require.Equal(t, "sess-A", rec["session_key"])
	require.Equal(t, "sys", rec["instructions"])
	require.Equal(t, false, rec["reset"])
	require.Len(t, rec["new_items"], 1)
	require.Equal(t, []any{"shell"}, archiveField(t, rec, "request_params")["tools"])
	require.Equal(t, "hi", archiveField(t, rec, "response")["text"])
	require.EqualValues(t, 10, archiveField(t, rec, "usage")["input_tokens"])
	images, ok := rec["images"].([]any)
	require.True(t, ok)
	require.Len(t, images, 1)
	firstImage, ok := images[0].(map[string]any)
	require.True(t, ok)
	imgKey, _ := firstImage["key"].(string)
	require.Regexp(t, `^archive/img/45/\d{8}/client_r1_0\.png$`, imgKey)
	require.Equal(t, "image/png", h.store.object(imgKey).contentType)
	require.True(t, h.store.object(imgKey).sse)

	require.Len(t, h.repo.turns, 1)
	turn := h.repo.turns[0]
	require.Equal(t, "hello", turn.Preview)
	require.Equal(t, 1, turn.NewItemCount)
	require.Equal(t, 1, turn.ImageCount)
	require.Len(t, h.repo.images, 1)
	require.Equal(t, int64(1), h.svc.Status().UploadedToday)

	// Second turn of the same session: only the new items, instructions omitted.
	capture2 := &ArchiveCapture{Protocol: ArchiveProtocolResponses,
		RequestBody: []byte(`{"instructions":"sys","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hi"}]},{"type":"message","role":"user","content":[{"type":"input_text","text":"more"}]}]}`),
		Response:    ArchiveResponse{Status: "completed", Text: "ok"},
	}
	h.svc.EnqueueArchive(archiveUsage("client:r2", "sess-A"), capture2)
	h.drain()
	require.Len(t, h.repo.turns, 2)
	rec2 := readArchiveRecord(t, h.store.object(h.repo.turns[1].ObjectKey))
	require.EqualValues(t, 1, rec2["prefix_items"])
	require.Len(t, rec2["new_items"], 2)
	_, hasInstr := rec2["instructions"]
	require.False(t, hasInstr)
	require.Equal(t, "more", h.repo.turns[1].Preview)

	// Upload failure: no index row, the dedupe state is not advanced, the error is reported.
	h.store.putErr = errors.New("503 SlowDown")
	h.svc.EnqueueArchive(archiveUsage("client:r3", "sess-A"), &ArchiveCapture{Protocol: ArchiveProtocolResponses, RequestBody: capture2.RequestBody})
	h.drain()
	require.Len(t, h.repo.turns, 2)
	st := h.svc.Status()
	require.Contains(t, st.LastError, "SlowDown")
	require.NotNil(t, st.LastErrorAt)
	require.Equal(t, uint64(1), st.FailedTotal)
}

func TestConversationArchive_RedisDownArchivesEverything(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	h.dedupe.loadErr = errors.New("redis down")
	h.svc.EnqueueArchive(archiveUsage("r1", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat,
		RequestBody: []byte(`{"messages":[{"role":"system","content":"S"},{"role":"user","content":"a"},{"role":"assistant","content":"b"},{"role":"user","content":"c"}]}`)})
	h.drain()
	require.Len(t, h.repo.turns, 1)
	rec := readArchiveRecord(t, h.store.object(h.repo.turns[0].ObjectKey))
	require.Equal(t, true, rec["reset"])
	require.Len(t, rec["new_items"], 3)
	require.Equal(t, "S", rec["instructions"])
}

func TestConversationArchive_QueueBoundsDropAndDisabled(t *testing.T) {
	h := newArchiveHarness(t)
	// Disabled: nothing is queued.
	h.svc.EnqueueArchive(archiveUsage("r", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat})
	require.Equal(t, int64(0), h.svc.Status().QueueLen)

	h.enable(t)
	h.svc.maxItems = 3
	for i := 0; i < 5; i++ {
		h.svc.EnqueueArchive(archiveUsage("r", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat})
	}
	st := h.svc.Status()
	require.Equal(t, int64(3), st.QueueLen)
	require.Equal(t, uint64(2), st.DroppedTotal)
	h.drain()

	h.svc.maxItems = 100
	h.svc.maxBytes = 10 << 10
	h.svc.EnqueueArchive(archiveUsage("r", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat, RequestBody: make([]byte, 20<<10)})
	require.Equal(t, int64(0), h.svc.Status().QueueLen)
	require.Equal(t, uint64(3), h.svc.Status().DroppedTotal)
	require.Equal(t, int64(0), h.svc.Status().QueueBytes)

	// nil-safety
	var nilSvc *ConversationArchiveService
	nilSvc.EnqueueArchive(archiveUsage("r", "s"), &ArchiveCapture{})
	require.False(t, nilSvc.Enabled())
	recordConversationArchive(nil, archiveUsage("r", "s"), &ContentCapture{Archive: &ArchiveCapture{}})
	recordConversationArchive(h.svc, archiveUsage("r", "s"), &ContentCapture{})
}

func TestConversationArchive_SettingsSecretHandling(t *testing.T) {
	h := newArchiveHarness(t)
	ctx := context.Background()
	view, err := h.svc.GetConfig(ctx)
	require.NoError(t, err)
	require.False(t, view.Enabled)
	require.Equal(t, "archive/", view.Prefix)
	require.Equal(t, 180, view.RetentionDays)
	require.True(t, view.SSE)
	require.False(t, view.SecretConfigured)

	_, err = h.svc.UpdateConfig(ctx, 1, ArchiveConfigInput{Enabled: true, Bucket: "b"}, "", "")
	require.ErrorIs(t, err, ErrArchiveIncomplete)
	_, err = h.svc.UpdateConfig(ctx, 1, ArchiveConfigInput{RetentionDays: 3}, "", "")
	require.ErrorIs(t, err, ErrArchiveRetention)

	view, err = h.svc.UpdateConfig(ctx, 7, ArchiveConfigInput{Enabled: true, Bucket: "b-1", AccessKeyID: "AK", SecretAccessKey: "S3CRET", Prefix: "/arc", RetentionDays: 30}, "1.1.1.1", "ua")
	require.NoError(t, err)
	require.True(t, view.SecretConfigured)
	require.Equal(t, "arc/", view.Prefix)
	require.Contains(t, h.repo.raw, `"secret_access_key":"enc:S3CRET"`)
	viewJSON, _ := json.Marshal(view)
	require.NotContains(t, string(viewJSON), "S3CRET")
	last := h.repo.access[len(h.repo.access)-1]
	require.Equal(t, ContentLogActionArchiveConfig, last.Action)
	require.Equal(t, int64(7), last.AdminID)
	require.Contains(t, last.Detail, `"secret_access_key":"changed"`)
	require.NotContains(t, last.Detail, "S3CRET")

	// Saving without a secret keeps the stored one; pausing turns capture off at once.
	_, err = h.svc.UpdateConfig(ctx, 7, ArchiveConfigInput{Enabled: false, Bucket: "b-1", AccessKeyID: "AK", Prefix: "arc/", RetentionDays: 30}, "", "")
	require.NoError(t, err)
	require.Contains(t, h.repo.raw, `"secret_access_key":"enc:S3CRET"`)
	require.False(t, h.svc.Enabled())
	require.False(t, ArchiveCaptureEnabled())
	require.JSONEq(t, `{"enabled":false}`, h.repo.access[len(h.repo.access)-1].Detail)

	// Without a fixed encryption key a new secret is refused.
	h.svc.encryptionKey = false
	_, err = h.svc.UpdateConfig(ctx, 7, ArchiveConfigInput{Bucket: "b", AccessKeyID: "AK", SecretAccessKey: "new"}, "", "")
	require.ErrorIs(t, err, ErrSecretEncryptionKeyNotConfigured)

	// A failed audit saves nothing.
	h.svc.encryptionKey = true
	h.repo.accessErr = errors.New("db down")
	_, err = h.svc.UpdateConfig(ctx, 7, ArchiveConfigInput{Enabled: true, Bucket: "b-1", AccessKeyID: "AK2", RetentionDays: 30}, "", "")
	require.Error(t, err)
	require.False(t, h.svc.Enabled())
}

func TestConversationArchive_TestConnectionUsesFormOrStored(t *testing.T) {
	h := newArchiveHarness(t)
	ctx := context.Background()
	require.ErrorIs(t, h.svc.TestConnection(ctx, nil), ErrArchiveIncomplete)
	// Unsaved form with a typed secret.
	require.NoError(t, h.svc.TestConnection(ctx, &ArchiveConfigInput{Bucket: "b", AccessKeyID: "AK", SecretAccessKey: "typed"}))
	require.Equal(t, "typed", h.configs[len(h.configs)-1].SecretAccessKey)
	require.Empty(t, h.store.keys(""), "the healthcheck object is deleted")
	h.enable(t)
	// Form without a secret falls back to the stored one (decrypted).
	require.NoError(t, h.svc.TestConnection(ctx, &ArchiveConfigInput{Bucket: "b2", AccessKeyID: "AK"}))
	require.Equal(t, "SECRET", h.configs[len(h.configs)-1].SecretAccessKey)
	require.Equal(t, "b2", h.configs[len(h.configs)-1].Bucket)
	h.store.putErr = errors.New("AccessDenied")
	err := h.svc.TestConnection(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "写入测试对象失败")
	require.Contains(t, err.Error(), "AccessDenied")
}

func TestConversationArchive_RetentionAndPurge(t *testing.T) {
	h := newArchiveHarness(t)
	h.svc.RunRetentionOnce() // not configured: no-op
	h.enable(t)
	for i := int64(1); i <= 3; i++ {
		h.repo.expired = append(h.repo.expired, ArchiveExpiredTurn{ID: i, ObjectKey: "archive/conv/1/x" + itoa(int(i)), ImageKeys: []string{"archive/img/1/y" + itoa(int(i))}})
	}
	h.store.deleteErr = errors.New("boom")
	h.svc.RunRetentionOnce()
	require.Empty(t, h.repo.deleted, "rows stay when the objects could not be deleted")
	h.store.deleteErr = nil
	h.svc.RunRetentionOnce()
	require.ElementsMatch(t, []int64{1, 2, 3}, h.repo.deleted)
	require.Len(t, h.store.deleted, 1)
	require.Len(t, h.store.deleted[0], 6)

	// Purge: the prefix in use and every previous one are recorded before the account goes;
	// queued jobs of the user are skipped; the sweep waits for the account to be deleted.
	ctx := context.Background()
	_, err := h.svc.UpdateConfig(ctx, 1, ArchiveConfigInput{Enabled: true, Bucket: "b-123", AccessKeyID: "AK", Prefix: "arc2/"}, "", "")
	require.NoError(t, err)
	h.svc.EnqueueArchive(archiveUsage("r9", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat, RequestBody: []byte(`{"messages":[{"role":"user","content":"x"}]}`)})
	require.NoError(t, h.svc.QueueUserPurge(ctx, 45))
	h.drain()
	require.Empty(t, h.repo.turns)
	require.ElementsMatch(t, []string{"arc2/", "archive/"}, h.repo.purges[45].Prefixes)
	h.svc.RunPurgesOnce()
	require.Empty(t, h.store.prefixes, "not swept while the account still exists")
	h.repo.mu.Lock()
	p := h.repo.purges[45]
	p.UserDeleted = true
	h.repo.purges[45] = p
	h.repo.mu.Unlock()
	h.svc.RunPurgesOnce()
	require.ElementsMatch(t, []string{"arc2/conv/45/", "arc2/img/45/", "archive/conv/45/", "archive/img/45/"}, h.store.prefixes)
	require.NotContains(t, h.repo.purges, int64(45))

	// Lifecycle: applied automatically on save while enabled, and by hand (audited first).
	before := len(h.store.lifecycle)
	require.GreaterOrEqual(t, before, 1)
	require.True(t, h.svc.Status().LifecycleApplied)
	require.NoError(t, h.svc.ApplyLifecycle(ctx, 1, "", ""))
	require.Len(t, h.store.lifecycle, before+1)
	require.Contains(t, h.store.lifecycle[before], archiveLifecycleRuleID+"|arc2/|")
	require.Equal(t, ContentLogActionArchiveConfig, h.repo.access[len(h.repo.access)-1].Action)
	h.repo.accessErr = errors.New("db down")
	require.Error(t, h.svc.ApplyLifecycle(ctx, 1, "", ""))
	require.Len(t, h.store.lifecycle, before+1, "no audit row, no change")
	h.repo.accessErr = nil
}

func TestConversationArchive_PurgeWaitsForStorage(t *testing.T) {
	h := newArchiveHarness(t)
	ctx := context.Background()
	require.NoError(t, h.svc.QueueUserPurge(ctx, 45))
	require.Empty(t, h.repo.purges, "never configured: nothing was archived")
	// Saved but incomplete (no store): the purge waits.
	_, err := h.svc.UpdateConfig(ctx, 1, ArchiveConfigInput{Bucket: "b"}, "", "")
	require.NoError(t, err)
	require.NoError(t, h.svc.QueueUserPurge(ctx, 45))
	h.repo.mu.Lock()
	p := h.repo.purges[45]
	p.UserDeleted = true
	h.repo.purges[45] = p
	h.repo.mu.Unlock()
	h.svc.RunPurgesOnce()
	require.Contains(t, h.repo.purges, int64(45))
	h.enable(t)
	h.svc.RunPurgesOnce()
	require.NotContains(t, h.repo.purges, int64(45))
}

func TestConversationArchive_ClaudeCodeMovingCacheBreakpoint(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	msg := func(role, text string, cache bool) string {
		cc := ""
		if cache {
			cc = `,"cache_control":{"type":"ephemeral"}`
		}
		return `{"role":"` + role + `","content":[{"type":"text","text":"` + text + `"` + cc + `}]}`
	}
	body := func(msgs ...string) []byte {
		return []byte(`{"system":[{"type":"text","text":"You are Claude Code","cache_control":{"type":"ephemeral"}}],"messages":[` + strings.Join(msgs, ",") + `]}`)
	}
	turns := [][]byte{
		body(msg("user", "one", true)),
		body(msg("user", "one", false), msg("assistant", "a1", false), msg("user", "two", true)),
		body(msg("user", "one", false), msg("assistant", "a1", false), msg("user", "two", false), msg("assistant", "a2", false), msg("user", "three", true)),
	}
	for i, b := range turns {
		h.svc.EnqueueArchive(archiveUsage("r"+itoa(i), "cc-session"), &ArchiveCapture{Protocol: ArchiveProtocolAnthropic, RequestBody: b})
		h.drain()
	}
	require.Len(t, h.repo.turns, 3)
	for i, want := range []struct {
		prefix, newItems int
	}{{0, 1}, {1, 2}, {3, 2}} {
		rec := readArchiveRecord(t, h.store.object(h.repo.turns[i].ObjectKey))
		require.Equal(t, false, rec["reset"], "turn %d", i)
		require.EqualValues(t, want.prefix, rec["prefix_items"], "turn %d", i)
		require.Len(t, rec["new_items"], want.newItems, "turn %d", i)
		_, hasInstr := rec["instructions"]
		require.Equal(t, i == 0, hasInstr, "turn %d", i)
	}
	// The stored items are as sent (the breakpoint is only ignored for hashing).
	rec := readArchiveRecord(t, h.store.object(h.repo.turns[2].ObjectKey))
	raw, _ := json.Marshal(rec["new_items"])
	require.Contains(t, string(raw), "cache_control")
}

func TestConversationArchive_SubAgentsDedupeSeparately(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	send := func(id string, msgs ...string) map[string]any {
		h.svc.EnqueueArchive(archiveUsage(id, "shared"), &ArchiveCapture{Protocol: ArchiveProtocolAnthropic,
			RequestBody: []byte(`{"messages":[` + strings.Join(msgs, ",") + `]}`)})
		h.drain()
		return readArchiveRecord(t, h.store.object(h.repo.turns[len(h.repo.turns)-1].ObjectKey))
	}
	u := func(t string) string { return `{"role":"user","content":"` + t + `"}` }
	a := func(t string) string { return `{"role":"assistant","content":"` + t + `"}` }
	main1 := send("m1", u("main task"))
	require.Equal(t, false, main1["reset"])
	sub1 := send("s1", u("sub-agent task"))
	require.Equal(t, true, sub1["reset"], "another context in the same session starts fresh")
	main2 := send("m2", u("main task"), a("ok"), u("next"))
	require.Equal(t, false, main2["reset"])
	require.EqualValues(t, 1, main2["prefix_items"])
	sub2 := send("s2", u("sub-agent task"), a("done"), u("more"))
	require.Equal(t, false, sub2["reset"])
	require.EqualValues(t, 1, sub2["prefix_items"])
	require.NotEqual(t, main2["thread"], sub2["thread"])
	for _, turn := range h.repo.turns {
		require.Equal(t, "shared", turn.SessionKey, "the UI still groups by session")
	}
}

func TestConversationArchive_OrphansCleanedAndStatus(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	img := base64.StdEncoding.EncodeToString(archivePNG())
	h.repo.insertErr = errors.New("db down")
	h.svc.EnqueueArchive(archiveUsage("r1", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat,
		RequestBody: []byte(`{"messages":[{"role":"user","content":"x"}]}`),
		Images:      []ArchiveImagePayload{{B64: img}, {B64: "%%%not base64"}}})
	h.drain()
	require.Empty(t, h.store.keys(""), "uploaded image and turn objects are deleted when indexing fails")
	require.Equal(t, uint64(2), h.svc.Status().FailedTotal, "the bad image and the index failure")

	h.repo.insertErr = nil
	h.svc.EnqueueArchive(archiveUsage("r2", "s"), &ArchiveCapture{Protocol: ArchiveProtocolChat,
		RequestBody: []byte(`{"messages":[{"role":"user","content":"x"}]}`), Images: []ArchiveImagePayload{{B64: "%%%"}}})
	h.drain()
	rec := readArchiveRecord(t, h.store.object(h.repo.turns[0].ObjectKey))
	require.Equal(t, true, archiveField(t, rec, "truncated")["response"])
}

func TestConversationArchive_StopDoesNotWaitOnUploads(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	blocking := &blockingArchiveStore{fakeArchiveStore: h.store}
	h.svc.cfgMu.Lock()
	h.svc.runtime = &archiveRuntime{settings: h.svc.runtime.settings, store: blocking}
	h.svc.cfgMu.Unlock()
	h.svc.drainTimeout = 200 * time.Millisecond
	h.svc.Start()
	for i := 0; i < 4; i++ {
		h.svc.EnqueueArchive(archiveUsage("r"+itoa(i), "s"+itoa(i)), &ArchiveCapture{Protocol: ArchiveProtocolChat, RequestBody: []byte(`{"messages":[]}`)})
	}
	start := time.Now()
	h.svc.Stop()
	require.Less(t, time.Since(start), 3*time.Second)
	require.Equal(t, int64(0), h.svc.Status().QueueLen)
}

type blockingArchiveStore struct{ *fakeArchiveStore }

func (b *blockingArchiveStore) PutObject(ctx context.Context, _ string, _ []byte, _, _ string, _ bool) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestConversationArchive_ReloadDoesNotUndoSave(t *testing.T) {
	h := newArchiveHarness(t)
	// A reload reads the (old, empty) settings; before it installs them an admin saves.
	h.repo.mu.Lock()
	h.repo.afterGet = func() { h.enable(t) }
	h.repo.mu.Unlock()
	h.svc.reload(context.Background())
	require.True(t, h.svc.Enabled(), "the stale reload must not undo the save")
	h.svc.reload(context.Background()) // a fresh reload sees the saved value: unchanged
	require.True(t, h.svc.Enabled())
}

func TestConversationArchive_TurnsPageByteBudget(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	for i := 1; i <= 5; i++ {
		h.repo.turns = append(h.repo.turns, ArchiveTurn{ID: int64(i), UserID: 45, SessionKey: "s", ObjectKey: "k" + itoa(i), Bytes: 30 << 20})
	}
	page, err := h.svc.ListTurns(context.Background(), 1, 45, "s", 0, 20, "", "")
	require.NoError(t, err)
	require.Len(t, page.Items, 2, "64 MiB compressed budget")
	require.NotNil(t, page.NextBeforeID)
	require.Equal(t, int64(4), *page.NextBeforeID)
}

func TestArchiveCapture_StreamWithoutTerminalIsIncomplete(t *testing.T) {
	c := newArchiveCaptureContext(t, "/v1/responses", `{"input":"x"}`)
	o := beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"cut off"}`), "response.output_text.delta")
	require.Equal(t, "incomplete", ContentCaptureSnapshot(c).Archive.Response.Status)
	c = newArchiveCaptureContext(t, "/v1beta/models/g:generateContent", `{"contents":[]}`)
	beginUpstreamResponseModelObservation(c)
	require.Equal(t, "unknown", ContentCaptureSnapshot(c).Archive.Response.Status)
}

func TestConversationArchive_BrowseAuditsFirst(t *testing.T) {
	h := newArchiveHarness(t)
	h.enable(t)
	ctx := context.Background()
	img := base64.StdEncoding.EncodeToString(archivePNG())
	h.svc.EnqueueArchive(archiveUsage("req-1", "sess"), &ArchiveCapture{Protocol: ArchiveProtocolChat,
		RequestBody: []byte(`{"messages":[{"role":"user","content":"x"}]}`), Response: ArchiveResponse{Text: "y"},
		Images: []ArchiveImagePayload{{B64: img, Source: "response"}}, ImageURLs: []string{"https://i/x.png"}})
	h.drain()
	before := len(h.repo.access)

	page, err := h.svc.ListTurns(ctx, 9, 45, "sess", 0, 0, "ip", "ua")
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	item := page.Items[0]
	require.Empty(t, item.FetchError)
	require.Equal(t, "y", item.Response.Text)
	require.Equal(t, ArchiveProtocolChat, item.Protocol)
	require.Len(t, item.NewItems, 1)
	require.Len(t, item.Images, 1)
	require.Equal(t, "image/png", item.Images[0].Mime)
	require.Equal(t, []string{"https://i/x.png"}, item.ImageURLs)
	require.Len(t, h.repo.access, before+1)
	require.Equal(t, ContentLogActionArchiveView, h.repo.access[before].Action)
	require.Equal(t, int64(45), h.repo.access[before].TargetUserID)

	byReq, err := h.svc.GetTurnByRequest(ctx, 9, "req-1", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(45), byReq.UserID)
	require.Equal(t, "sess", byReq.SessionKey)
	require.Equal(t, "u@test", byReq.Email)
	_, err = h.svc.GetTurnByRequest(ctx, 9, "nope", "", "")
	require.ErrorIs(t, err, ErrArchiveTurnNotFound)

	rc, meta, err := h.svc.OpenImage(ctx, item.Images[0].ID)
	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	require.Equal(t, archivePNG(), data)
	require.Equal(t, "image/png", meta.Mime)

	// A missing object marks that item instead of failing the page.
	h.store.mu.Lock()
	delete(h.store.objects, h.repo.turns[0].ObjectKey)
	h.store.mu.Unlock()
	page, err = h.svc.ListTurns(ctx, 9, 45, "sess", 0, 0, "", "")
	require.NoError(t, err)
	require.Contains(t, page.Items[0].FetchError, "NoSuchKey")

	// No audit row, no data.
	h.repo.accessErr = errors.New("db down")
	_, err = h.svc.ListTurns(ctx, 9, 45, "sess", 0, 0, "", "")
	require.Error(t, err)
	_, err = h.svc.ListSessions(ctx, 9, ArchiveSessionFilter{}, "", "")
	require.Error(t, err)
	h.repo.accessErr = nil

	// Sessions: q resolves to a user; without a user the default range is the last 30 days.
	h.repo.resolve["a@b.c"] = 45
	_, err = h.svc.ListSessions(ctx, 9, ArchiveSessionFilter{Q: "a@b.c"}, "", "")
	require.NoError(t, err)
	require.Equal(t, int64(45), h.repo.lastSF.UserID)
	require.Nil(t, h.repo.lastSF.From)
	_, err = h.svc.ListSessions(ctx, 9, ArchiveSessionFilter{}, "", "")
	require.NoError(t, err)
	require.NotNil(t, h.repo.lastSF.From)
	require.WithinDuration(t, time.Now().AddDate(0, 0, -30), *h.repo.lastSF.From, time.Minute)
	page2, err := h.svc.ListSessions(ctx, 9, ArchiveSessionFilter{Q: "nobody"}, "", "")
	require.NoError(t, err)
	require.Empty(t, page2.Items)
	_, err = h.svc.ListTurns(ctx, 9, 0, "", 0, 0, "", "")
	require.ErrorIs(t, err, ErrArchiveSessionParams)
}

func TestArchiveProtocolForPath(t *testing.T) {
	require.Equal(t, ArchiveProtocolResponses, ArchiveProtocolForPath("/v1/responses"))
	require.Equal(t, ArchiveProtocolResponses, ArchiveProtocolForPath("/backend-api/codex/responses"))
	require.Equal(t, ArchiveProtocolChat, ArchiveProtocolForPath("/v1/chat/completions"))
	require.Equal(t, ArchiveProtocolAnthropic, ArchiveProtocolForPath("/v1/messages"))
	require.Equal(t, "", ArchiveProtocolForPath("/v1/messages/count_tokens"))
	require.Equal(t, ArchiveProtocolGemini, ArchiveProtocolForPath("/v1beta/models/gemini-3:streamGenerateContent"))
	require.Equal(t, ArchiveProtocolImages, ArchiveProtocolForPath("/v1/images/edits"))
	require.Equal(t, "", ArchiveProtocolForPath("/v1/embeddings"))
}

func TestDecodeArchiveImage(t *testing.T) {
	data, mime, ext, err := decodeArchiveImage("data:image/png;base64," + base64.StdEncoding.EncodeToString(archivePNG()))
	require.NoError(t, err)
	require.Equal(t, archivePNG(), data)
	require.Equal(t, "image/png", mime)
	require.Equal(t, "png", ext)
	_, _, _, err = decodeArchiveImage("!!!")
	require.Error(t, err)
	require.Equal(t, "client_abc_def", archiveSanitizeRequestID("client:abc/def"))
	require.Equal(t, "noid", archiveSanitizeRequestID(""))
}
