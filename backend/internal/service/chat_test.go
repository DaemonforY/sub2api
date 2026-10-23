//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// memChatRepo is an in-memory ChatRepository.
type memChatRepo struct {
	mu     sync.Mutex
	nextID int64
	convs  map[int64]*ChatConversation
	msgs   map[int64]*ChatMessage
	files  map[string]*ChatFile
}

func newMemChatRepo() *memChatRepo {
	return &memChatRepo{convs: map[int64]*ChatConversation{}, msgs: map[int64]*ChatMessage{}, files: map[string]*ChatFile{}}
}

func (r *memChatRepo) id() int64 { r.nextID++; return r.nextID }

func (r *memChatRepo) ListConversations(_ context.Context, userID int64, query string, limit int) ([]ChatConversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ChatConversation{}
	for _, c := range r.convs {
		if c.UserID == userID && strings.Contains(c.Title, query) {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (r *memChatRepo) GetConversation(_ context.Context, userID, id int64) (*ChatConversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.convs[id]
	if c == nil || c.UserID != userID {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *memChatRepo) CreateConversation(_ context.Context, c *ChatConversation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c.ID = r.id()
	cp := *c
	r.convs[c.ID] = &cp
	return nil
}

func (r *memChatRepo) UpdateConversation(_ context.Context, c *ChatConversation, _ bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.convs[c.ID] = &cp
	return nil
}

func (r *memChatRepo) dropMessages(match func(*ChatMessage) bool, keep map[string]bool) []string {
	var gone []string
	for id, m := range r.msgs {
		if !match(m) {
			continue
		}
		for name, f := range r.files {
			if f.MessageID != nil && *f.MessageID == id {
				if keep[name] {
					f.MessageID = nil
					continue
				}
				gone = append(gone, name)
				delete(r.files, name)
			}
		}
		delete(r.msgs, id)
	}
	return gone
}

func (r *memChatRepo) DeleteConversation(_ context.Context, userID, id int64) (bool, []string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.convs[id]
	if c == nil || c.UserID != userID {
		return false, nil, nil
	}
	delete(r.convs, id)
	return true, r.dropMessages(func(m *ChatMessage) bool { return m.ConversationID == id }, nil), nil
}

func (r *memChatRepo) ListMessages(_ context.Context, convID int64) ([]ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ChatMessage{}
	for _, m := range r.msgs {
		if m.ConversationID == convID {
			cp := *m
			cp.Files = append([]string{}, m.Files...)
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *memChatRepo) CountMessages(ctx context.Context, convID int64) (int, error) {
	m, _ := r.ListMessages(ctx, convID)
	return len(m), nil
}

func (r *memChatRepo) CreateMessage(_ context.Context, m *ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.ID = r.id()
	cp := *m
	cp.Files = append([]string{}, m.Files...)
	r.msgs[m.ID] = &cp
	return nil
}

func (r *memChatRepo) UpdateMessage(_ context.Context, m *ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *m
	cp.Files = append([]string{}, m.Files...)
	r.msgs[m.ID] = &cp
	return nil
}

func (r *memChatRepo) DeleteMessagesFrom(_ context.Context, convID, fromID int64, keep []string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := map[string]bool{}
	for _, n := range keep {
		k[n] = true
	}
	return r.dropMessages(func(m *ChatMessage) bool { return m.ConversationID == convID && m.ID >= fromID }, k), nil
}

func (r *memChatRepo) FailPending(_ context.Context, msg string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, m := range r.msgs {
		if m.Status == ChatStatusPending {
			m.Status, m.Error = ChatStatusError, msg
			n++
		}
	}
	return n, nil
}

func (r *memChatRepo) CreateFile(_ context.Context, f *ChatFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f.CreatedAt = time.Now()
	cp := *f
	r.files[f.Name] = &cp
	return nil
}

func (r *memChatRepo) GetFile(_ context.Context, name string) (*ChatFile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f := r.files[name]
	if f == nil {
		return nil, nil
	}
	cp := *f
	return &cp, nil
}

func (r *memChatRepo) AttachFiles(_ context.Context, messageID int64, names []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range names {
		if f := r.files[n]; f != nil && f.MessageID == nil {
			id := messageID
			f.MessageID = &id
		}
	}
	return nil
}

func (r *memChatRepo) CountFilesSince(_ context.Context, userID int64, since time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, f := range r.files {
		if f.UserID == userID && !f.CreatedAt.Before(since) {
			n++
		}
	}
	return n, nil
}

func (r *memChatRepo) DeleteUnsentFiles(_ context.Context, before time.Time) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for name, f := range r.files {
		if f.MessageID == nil && f.CreatedAt.Before(before) {
			out = append(out, name)
			delete(r.files, name)
		}
	}
	return out, nil
}

func (r *memChatRepo) KnownFiles(_ context.Context, names []string) (map[string]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]bool{}
	for _, n := range names {
		if r.files[n] != nil {
			out[n] = true
		}
	}
	return out, nil
}

// chatKeyRepo serves fixed keys; the other APIKeyRepository methods are not used.
type chatKeyRepo struct {
	APIKeyRepository
	keys []APIKey
}

func (r *chatKeyRepo) GetByID(_ context.Context, id int64) (*APIKey, error) {
	for i := range r.keys {
		if r.keys[i].ID == id {
			k := r.keys[i]
			return &k, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *chatKeyRepo) ListByUserID(_ context.Context, userID int64, _ pagination.PaginationParams, _ APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	var out []APIKey
	for _, k := range r.keys {
		if k.UserID == userID {
			out = append(out, k)
		}
	}
	return out, nil, nil
}

// fakeChatGateway answers /v1/chat/completions with a fixed streamed reply (recording the request),
// /v1/models and /v1/images/*.
type fakeChatGateway struct {
	mu       sync.Mutex
	bodies   []map[string]any
	paths    []string
	reply    []string
	block    chan struct{} // if set, the stream waits on it after the first piece
	imageErr bool
}

func (g *fakeChatGateway) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.paths = append(g.paths, r.URL.Path)
		g.mu.Unlock()
		switch r.URL.Path {
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"gpt-5.5"},{"id":"gpt-image-2"},{"id":"text-embedding-3-small"},{"id":"claude-opus-5-5"},{"id":"gpt-5.5"}]}`)
		case "/v1/chat/completions":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			g.mu.Lock()
			g.bodies = append(g.bodies, body)
			g.mu.Unlock()
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)
			_, _ = io.WriteString(w, `data: {"model":"gpt-5.5","choices":[{"delta":{"reasoning_content":"想一想"}}]}`+"\n\n")
			for i, piece := range g.reply {
				b, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": piece}}}})
				_, _ = io.WriteString(w, "data: "+string(b)+"\n\n")
				flusher.Flush()
				if i == 0 && g.block != nil {
					select {
					case <-g.block:
					case <-r.Context().Done():
						return
					}
				}
			}
			_, _ = io.WriteString(w, `data: {"choices":[],"usage":{"prompt_tokens":12,"completion_tokens":5}}`+"\n\ndata: [DONE]\n\n")
		case "/v1/images/generations", "/v1/images/edits":
			if g.imageErr {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"message":"余额不足"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(testPNG(t))+`"}]}`)
		default:
			http.NotFound(w, r)
		}
	})
}

func testPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

type chatFixture struct {
	svc  *ChatService
	repo *memChatRepo
	gw   *fakeChatGateway
}

func newChatFixture(t *testing.T) *chatFixture {
	gw := &fakeChatGateway{reply: []string{"你好", "，我是助手"}}
	srv := httptest.NewServer(gw.handler(t))
	t.Cleanup(srv.Close)
	repo := newMemChatRepo()
	keys := &chatKeyRepo{keys: []APIKey{
		{ID: 1, UserID: 7, Key: "sk-gpt", Name: "gpt", Status: StatusActive, Group: &Group{ID: 5, Name: "GPT-按量", Platform: PlatformOpenAI, RateMultiplier: 1}},
		{ID: 2, UserID: 7, Key: "sk-claude", Name: "claude", Status: StatusActive, Group: &Group{ID: 11, Name: "Claude", Platform: PlatformAnthropic}},
		{ID: 3, UserID: 7, Key: "sk-cc", Name: "cc", Status: StatusActive, Group: &Group{ID: 9, Name: "CC", Platform: PlatformAnthropic, ClaudeCodeOnly: true}},
		{ID: 4, UserID: 8, Key: "sk-other", Name: "other", Status: StatusActive, Group: &Group{ID: 5, Platform: PlatformOpenAI}},
		{ID: 5, UserID: 7, Key: "sk-off", Name: "off", Status: "disabled", Group: &Group{ID: 5, Platform: PlatformOpenAI}},
	}}
	svc := NewChatService(repo, keys, NewChatFileStore(t.TempDir()), srv.URL)
	svc.now = func() time.Time { return time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC) }
	return &chatFixture{svc: svc, repo: repo, gw: gw}
}

func collect(events *[]ChatEvent) func(ChatEvent) error {
	return func(ev ChatEvent) error {
		*events = append(*events, ev)
		return nil
	}
}

func TestChatModelIDsKeepsChatModels(t *testing.T) {
	raw := []byte(`{"data":[{"id":"gpt-5.5"},{"id":"gpt-image-2"},{"id":"text-embedding-3-small"},{"id":"tts-1"},{"id":"claude-opus-5-5"},{"id":"gpt-5.5"},{"id":"bad model"}]}`)
	require.Equal(t, []string{"gpt-5.5", "claude-opus-5-5"}, chatModelIDs(raw))
	require.Equal(t, []string{}, chatModelIDs([]byte(`nope`)))
}

func TestChatKeysSkipsUnusableKeys(t *testing.T) {
	f := newChatFixture(t)
	keys, err := f.svc.Keys(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.True(t, keys[0].CanDraw)
	require.False(t, keys[1].CanDraw)

	_, err = f.svc.Models(context.Background(), 7, 3)
	require.ErrorIs(t, err, ErrChatKeyCodeOnly)
	_, err = f.svc.Models(context.Background(), 7, 4)
	require.ErrorIs(t, err, ErrChatKeyInvalid)
	m, err := f.svc.Models(context.Background(), 7, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-5.5", "claude-opus-5-5"}, m.Models)
	require.True(t, m.CanDraw)
}

func TestChatSendStreamsAndSaves(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	sys := "你是翻译"
	conv, err := f.svc.Create(ctx, 7, ChatConversationInput{SystemPrompt: &sys})
	require.NoError(t, err)

	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Content: "  帮我翻译一句话  "}, collect(&events)))
	require.NotNil(t, events[0].User)
	require.Equal(t, "帮我翻译一句话", events[0].User.Content)
	require.Equal(t, "帮我翻译一句话", events[0].Conversation.Title)
	last := events[len(events)-1]
	require.NotNil(t, last.Done)
	require.Equal(t, ChatStatusDone, last.Done.Status)
	require.Equal(t, "你好，我是助手", last.Done.Content)
	require.Equal(t, "想一想", last.Done.Reasoning)
	require.Equal(t, 12, last.Done.PromptTokens)

	d, err := f.svc.Get(ctx, 7, conv.ID)
	require.NoError(t, err)
	require.Len(t, d.Messages, 2)
	require.Equal(t, "你好，我是助手", d.Messages[1].Content)
	require.Equal(t, "gpt-5.5", d.Conversation.Model)
	require.Equal(t, int64(1), *d.Conversation.APIKeyID)

	body := f.gw.bodies[0]
	msgs := body["messages"].([]any)
	require.Len(t, msgs, 2)
	sysMsg := msgs[0].(map[string]any)
	require.Equal(t, "system", sysMsg["role"])
	require.True(t, strings.HasPrefix(sysMsg["content"].(string), "你是翻译"))
	require.Contains(t, sysMsg["content"], "2026-10-23")
	require.NotContains(t, body, "max_tokens")

	// Other users can't see or write to it.
	_, err = f.svc.Get(ctx, 8, conv.ID)
	require.ErrorIs(t, err, ErrChatNotFound)
	require.ErrorIs(t, f.svc.Send(ctx, 8, conv.ID, ChatSendInput{KeyID: 4, Model: "gpt-5.5", Content: "hi"}, collect(&events)), ErrChatNotFound)
}

func TestChatSendValidates(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Content: "  "}, collect(&events)), ErrChatEmpty)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Content: "hi"}, collect(&events)), ErrChatModel)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 3, Model: "claude-opus-5-5", Content: "hi"}, collect(&events)), ErrChatKeyCodeOnly)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 2, Mode: "image", Content: "画猫"}, collect(&events)), ErrChatNoDraw)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Content: "hi", Files: []string{"0123456789abcdef0123456789abcdef.png"}}, collect(&events)), ErrChatFileGone)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Regenerate: true}, collect(&events)), ErrChatNothingToRedo)
	require.Empty(t, events)
	msgs, _ := f.repo.ListMessages(ctx, conv.ID)
	require.Empty(t, msgs)
}

func TestChatSendAnthropicSetsMaxTokens(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 2, Model: "claude-opus-5-5", Content: "hi"}, collect(&events)))
	require.EqualValues(t, chatAnthropicMaxOut, f.gw.bodies[0]["max_tokens"])
}

func TestChatStopKeepsPartialReply(t *testing.T) {
	f := newChatFixture(t)
	f.gw.block = make(chan struct{})
	conv, _ := f.svc.Create(context.Background(), 7, ChatConversationInput{})
	ctx, cancel := context.WithCancel(context.Background())
	var events []ChatEvent
	err := f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Content: "讲个长故事"}, func(ev ChatEvent) error {
		events = append(events, ev)
		if ev.Delta == "你好" {
			cancel()
		}
		return nil
	})
	require.NoError(t, err)
	last := events[len(events)-1].Done
	require.Equal(t, ChatStatusStopped, last.Status)
	require.Equal(t, "你好", last.Content)
	msgs, _ := f.repo.ListMessages(context.Background(), conv.ID)
	require.Equal(t, ChatStatusStopped, msgs[1].Status)
	// The conversation is free again.
	require.False(t, f.svc.running[conv.ID])
}

func TestChatRegenerateAndEdit(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	send := func(in ChatSendInput) {
		in.KeyID, in.Model = 1, "gpt-5.5"
		require.NoError(t, f.svc.Send(ctx, 7, conv.ID, in, collect(&events)))
	}
	send(ChatSendInput{Content: "第一问"})
	send(ChatSendInput{Content: "第二问"})
	msgs, _ := f.repo.ListMessages(ctx, conv.ID)
	require.Len(t, msgs, 4)

	send(ChatSendInput{Regenerate: true})
	after, _ := f.repo.ListMessages(ctx, conv.ID)
	require.Len(t, after, 4)
	require.Equal(t, msgs[2].ID, after[2].ID, "the question stays")
	require.NotEqual(t, msgs[3].ID, after[3].ID, "the answer is new")
	// The history sent with the regenerated answer ends with the question, not the old answer.
	sent := f.gw.bodies[2]["messages"].([]any)
	require.Len(t, sent, 4)
	require.Equal(t, "第二问", sent[3].(map[string]any)["content"])

	send(ChatSendInput{Content: "改过的第一问", EditID: msgs[0].ID})
	edited, _ := f.repo.ListMessages(ctx, conv.ID)
	require.Len(t, edited, 2)
	require.Equal(t, "改过的第一问", edited[0].Content)
}

func TestChatPicturesAreSentAndCopied(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	up, err := f.svc.Upload(ctx, 7, testPNG(t))
	require.NoError(t, err)
	require.Equal(t, 4, up.Width)

	_, err = f.svc.Upload(ctx, 7, []byte("not an image"))
	require.ErrorIs(t, err, ErrChatFileInvalid)

	// Someone else can't use or read it.
	_, _, err = f.svc.File(ctx, 8, up.Name)
	require.ErrorIs(t, err, ErrChatFileGone)

	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Files: []string{up.Name}}, collect(&events)))
	parts := f.gw.bodies[0]["messages"].([]any)[1].(map[string]any)["content"].([]any)
	require.Equal(t, "image_url", parts[0].(map[string]any)["type"])
	url := parts[0].(map[string]any)["image_url"].(map[string]any)["url"].(string)
	require.True(t, strings.HasPrefix(url, "data:image/png;base64,"))

	// Draw with it as the reference: the sent picture is copied for the new message.
	events = nil
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Mode: "image", Content: "改成蓝色", Files: []string{up.Name}}, collect(&events)))
	require.Equal(t, "/v1/images/edits", f.gw.paths[len(f.gw.paths)-1])
	user := events[0].User
	require.Len(t, user.Files, 1)
	require.NotEqual(t, up.Name, user.Files[0])
	done := events[len(events)-1].Done
	require.Equal(t, ChatStatusDone, done.Status)
	require.Equal(t, ChatKindImage, done.Kind)
	require.Len(t, done.Files, 1)
	data, mime, err := f.svc.File(ctx, 7, done.Files[0])
	require.NoError(t, err)
	require.Equal(t, "image/png", mime)
	require.NotEmpty(t, data)

	// Deleting the conversation removes its pictures from disk.
	require.NoError(t, f.svc.Delete(ctx, 7, conv.ID))
	_, err = f.svc.files.Read(done.Files[0])
	require.Error(t, err)
	_, err = f.svc.files.Read(up.Name)
	require.Error(t, err)
}

func TestChatDrawFailureIsSaved(t *testing.T) {
	f := newChatFixture(t)
	f.gw.imageErr = true
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Mode: "image", Content: "一只猫", ImageSize: "1536x1024"}, collect(&events)))
	last := events[len(events)-1]
	require.Contains(t, last.Error, "余额不足")
	require.Equal(t, ChatStatusError, last.Done.Status)
	require.Equal(t, "/v1/images/generations", f.gw.paths[len(f.gw.paths)-1])
	d, _ := f.svc.Get(ctx, 7, conv.ID)
	require.Equal(t, "画图：一只猫", d.Conversation.Title)
	require.Contains(t, d.Messages[1].Error, "余额不足")

	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Mode: "image", Content: "猫", ImageSize: "999x1"}, collect(&events)), ErrChatImageSize)
}

func TestChatContextBudgetAndImages(t *testing.T) {
	f := newChatFixture(t)
	conv := &ChatConversation{ID: 1}
	var msgs []ChatMessage
	for i := 0; i < 60; i++ {
		role := ChatRoleUser
		if i%2 == 1 {
			role = ChatRoleAssistant
		}
		msgs = append(msgs, ChatMessage{ID: int64(i + 1), Role: role, Content: "m", Status: ChatStatusDone})
	}
	out := f.svc.buildContext(conv, msgs, 61)
	require.Len(t, out, 1+chatContextMessages)
	require.Equal(t, "user", out[1]["role"], "history starts with a question")

	// Failed replies are left out; old pictures are only mentioned.
	msgs = []ChatMessage{
		{ID: 1, Role: ChatRoleUser, Content: "看图", Files: []string{"a.png"}, Status: ChatStatusDone},
		{ID: 2, Role: ChatRoleAssistant, Content: "", Status: ChatStatusError},
	}
	for i := 0; i < chatContextImageTurns; i++ {
		msgs = append(msgs, ChatMessage{ID: int64(10 + i), Role: ChatRoleUser, Content: "再看", Files: []string{"b.png"}, Status: ChatStatusDone})
	}
	out = f.svc.buildContext(conv, msgs, 99)
	require.Equal(t, "[附了 1 张图片] 看图", out[1]["content"])
	require.Equal(t, "user", out[2]["role"])
	_, isParts := out[2]["content"].([]map[string]any)
	require.True(t, isParts)
}

func TestChatBusyConversation(t *testing.T) {
	f := newChatFixture(t)
	conv, _ := f.svc.Create(context.Background(), 7, ChatConversationInput{})
	require.True(t, f.svc.claim(conv.ID))
	var events []ChatEvent
	require.ErrorIs(t, f.svc.Send(context.Background(), 7, conv.ID, ChatSendInput{KeyID: 1, Model: "gpt-5.5", Content: "hi"}, collect(&events)), ErrChatBusy)
	f.svc.release(conv.ID)
}

func TestChatSweepRemovesUnsentAndOrphans(t *testing.T) {
	f := newChatFixture(t)
	ctx := context.Background()
	up, err := f.svc.Upload(ctx, 7, testPNG(t))
	require.NoError(t, err)
	orphan, err := f.svc.files.Save(testPNG(t))
	require.NoError(t, err)
	f.svc.now = func() time.Time { return time.Now().Add(48 * time.Hour) }
	f.repo.files[up.Name].CreatedAt = time.Now().Add(-25 * time.Hour)
	f.svc.Sweep(ctx)
	_, err = f.svc.files.Read(up.Name)
	require.Error(t, err)
	_, err = f.svc.files.Read(orphan.Name)
	require.Error(t, err)
}

func TestChatUpstreamFailedKeepsChineseGatewayMessages(t *testing.T) {
	err := chatUpstreamFailed([]byte(`{"error":{"message":"余额不足，请充值后再试（Insufficient balance）"}}`), 402)
	require.Equal(t, "余额不足，请充值后再试（Insufficient balance）", chatErrText(err))
	err = chatUpstreamFailed([]byte(`{"error":{"message":"upstream overloaded"}}`), 529)
	require.Equal(t, "模型服务出错：upstream overloaded，请稍后再试（Chat failed）", chatErrText(err))
}
