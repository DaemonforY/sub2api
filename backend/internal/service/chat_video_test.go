//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testMP4 is the start of a real MP4 (an ftyp box), enough for content sniffing.
var testMP4 = append([]byte("\x00\x00\x00\x20ftypisom\x00\x00\x02\x00isomiso2avc1mp41"), make([]byte, 64)...)

// fakeVideoGateway answers /v1/videos like sub2api does for a muse2api account: create returns a
// pending task, status turns done after `pendingPolls` polls, and content serves the MP4.
type fakeVideoGateway struct {
	mu           sync.Mutex
	created      []map[string]any
	paths        []string
	pendingPolls int
	polls        int
	fail         string // if set, the task ends failed with this error
	createStatus int    // if set, create answers with this status and an error body
}

func (g *fakeVideoGateway) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.paths = append(g.paths, r.Method+" "+r.URL.Path)
		require.Equal(t, "Bearer sk-muse", r.Header.Get("Authorization"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/videos":
			if g.createStatus != 0 {
				w.WriteHeader(g.createStatus)
				_, _ = io.WriteString(w, `{"error":{"message":"这个 Key 属于订阅分组，但你没有该分组的有效订阅"}}`)
				return
			}
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			g.created = append(g.created, body)
			_, _ = io.WriteString(w, `{"id":"task_1","status":"pending"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/videos/task_1":
			g.polls++
			switch {
			case g.polls <= g.pendingPolls:
				_, _ = io.WriteString(w, `{"id":"task_1","status":"pending","progress":20}`)
			case g.fail != "":
				b, _ := json.Marshal(map[string]any{"id": "task_1", "status": "failed", "error": g.fail})
				_, _ = w.Write(b)
			default:
				_, _ = io.WriteString(w, `{"id":"task_1","status":"done","video":{"url":"/v1/videos/task_1/content"}}`)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v1/videos/task_1/content":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write(testMP4)
		default:
			http.NotFound(w, r)
		}
	})
}

type chatVideoFixture struct {
	svc  *ChatService
	repo *memChatRepo
	gw   *fakeVideoGateway
}

func newChatVideoFixture(t *testing.T) *chatVideoFixture {
	gw := &fakeVideoGateway{pendingPolls: 2}
	srv := httptest.NewServer(gw.handler(t))
	t.Cleanup(srv.Close)
	repo := newMemChatRepo()
	keys := &chatKeyRepo{keys: []APIKey{
		{ID: 1, UserID: 7, Key: "sk-gpt", Name: "gpt", Status: StatusActive, Group: &Group{ID: 5, Name: "GPT-按量", Platform: PlatformOpenAI, RateMultiplier: 1}},
		{ID: 6, UserID: 7, Key: "sk-muse", Name: "muse", Status: StatusActive, Group: &Group{ID: 13, Name: "Muse 视频", Platform: PlatformGrok, RateMultiplier: 1}},
	}}
	svc := NewChatService(repo, keys, NewChatFileStore(t.TempDir()), srv.URL)
	svc.now = func() time.Time { return time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC) }
	svc.videoPoll = time.Millisecond
	return &chatVideoFixture{svc: svc, repo: repo, gw: gw}
}

func TestChatKeysAndModelsMarkVideoKeys(t *testing.T) {
	f := newChatVideoFixture(t)
	ctx := context.Background()
	keys, err := f.svc.Keys(ctx, 7)
	require.NoError(t, err)
	require.Len(t, keys, 2)
	require.False(t, keys[0].CanVideo)
	require.True(t, keys[0].CanDraw)
	require.True(t, keys[1].CanVideo)
	require.False(t, keys[1].CanDraw)

	m, err := f.svc.Models(ctx, 7, 6)
	require.NoError(t, err)
	require.True(t, m.CanVideo)
	require.Empty(t, m.Models)
	require.Empty(t, f.gw.paths, "the video key's models are not fetched from the gateway")
}

func TestChatVideoTextToVideoSavesTheClip(t *testing.T) {
	f := newChatVideoFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "江南水乡，乌篷船划过石桥", VideoAspect: "16:9"}, collect(&events)))

	last := events[len(events)-1]
	require.Empty(t, last.Error)
	require.Equal(t, ChatStatusDone, last.Done.Status)
	require.Equal(t, ChatKindVideo, last.Done.Kind)
	require.Equal(t, chatVideoModel, last.Done.Model)
	require.Len(t, last.Done.Files, 1)
	require.True(t, strings.HasSuffix(last.Done.Files[0], ".mp4"))

	require.Len(t, f.gw.created, 1)
	require.Equal(t, "muse-video", f.gw.created[0]["model"])
	require.Equal(t, "16:9", f.gw.created[0]["aspect_ratio"])
	require.Equal(t, "江南水乡，乌篷船划过石桥", f.gw.created[0]["prompt"])
	require.NotContains(t, f.gw.created[0], "image")
	require.Equal(t, 3, f.gw.polls, "polled until done")
	require.Equal(t, "GET /v1/videos/task_1/content", f.gw.paths[len(f.gw.paths)-1])

	data, mime, err := f.svc.File(ctx, 7, last.Done.Files[0], true)
	require.NoError(t, err)
	require.Equal(t, "video/mp4", mime)
	require.Equal(t, testMP4, data)

	d, _ := f.svc.Get(ctx, 7, conv.ID)
	require.Equal(t, "视频：江南水乡，乌篷船划过石桥", d.Conversation.Title)
	require.Empty(t, d.Conversation.Model, "video turns don't change the chat model")
}

func TestChatVideoImageToVideoSendsTheFirstFrame(t *testing.T) {
	f := newChatVideoFixture(t)
	ctx := context.Background()
	up, err := f.svc.Upload(ctx, 7, testPNG(t))
	require.NoError(t, err)
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "让猫眨眼", Files: []string{up.Name}}, collect(&events)))
	require.Equal(t, ChatStatusDone, events[len(events)-1].Done.Status)
	require.Equal(t, "9:16", f.gw.created[0]["aspect_ratio"], "portrait by default")
	require.True(t, strings.HasPrefix(f.gw.created[0]["image"].(string), "data:image/png;base64,"))

	// A generated video can't be sent back as a reference.
	video := events[len(events)-1].Done.Files[0]
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "再来", Files: []string{video}}, collect(&events)), ErrChatVideoAsRef)
}

func TestChatVideoFailureIsSavedWithTheReason(t *testing.T) {
	f := newChatVideoFixture(t)
	f.gw.fail = "提示词未成功送达后端（Still sending / Delivery not confirmed）"
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "一只猫"}, collect(&events)))
	last := events[len(events)-1]
	require.Equal(t, ChatStatusError, last.Done.Status)
	require.Contains(t, last.Error, "视频生成失败：提示词未成功送达后端")
	require.Contains(t, last.Error, "不扣次数")
	require.Empty(t, last.Done.Files)
}

func TestChatVideoGatewayRejectionIsShown(t *testing.T) {
	f := newChatVideoFixture(t)
	f.gw.createStatus = http.StatusForbidden
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "一只猫"}, collect(&events)))
	require.Contains(t, events[len(events)-1].Error, "没有该分组的有效订阅")
}

func TestChatVideoValidates(t *testing.T) {
	f := newChatVideoFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	noop := func(ChatEvent) error { return nil }
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 1, Mode: "video", Content: "猫"}, noop), ErrChatNoVideo)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "猫", VideoAspect: "1:1"}, noop), ErrChatVideoAspect)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "猫", Files: []string{"a", "b"}}, noop), ErrChatVideoRefs)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video"}, noop), ErrChatEmpty)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "text", Model: "gpt-5.5", Content: "你好"}, noop), ErrChatVideoKeyOnly)
	require.ErrorIs(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "image", Content: "猫"}, noop), ErrChatNoDraw)
	require.Empty(t, f.gw.created)
}

func TestChatVideoRegenerateKeepsVideoMode(t *testing.T) {
	f := newChatVideoFixture(t)
	ctx := context.Background()
	conv, _ := f.svc.Create(ctx, 7, ChatConversationInput{})
	var events []ChatEvent
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Content: "一只猫", VideoAspect: "16:9"}, collect(&events)))
	f.gw.polls = 0
	require.NoError(t, f.svc.Send(ctx, 7, conv.ID, ChatSendInput{KeyID: 6, Mode: "video", Regenerate: true, VideoAspect: "16:9"}, collect(&events)))
	last := events[len(events)-1]
	require.Equal(t, ChatKindVideo, last.Done.Kind)
	require.Equal(t, ChatStatusDone, last.Done.Status)
	require.Len(t, f.gw.created, 2)
	require.Equal(t, "一只猫", f.gw.created[1]["prompt"])
	d, _ := f.svc.Get(ctx, 7, conv.ID)
	require.Len(t, d.Messages, 2, "the earlier reply was replaced")
}

func TestChatContextDescribesVideoTurns(t *testing.T) {
	f := newChatVideoFixture(t)
	conv := &ChatConversation{ID: 1, UserID: 7}
	msgs := []ChatMessage{
		{ID: 1, Role: ChatRoleUser, Kind: ChatKindVideo, Content: "一只猫", Status: ChatStatusDone},
		{ID: 2, Role: ChatRoleAssistant, Kind: ChatKindVideo, Files: []string{"x.mp4"}, Status: ChatStatusDone},
		{ID: 3, Role: ChatRoleUser, Kind: ChatKindText, Content: "这个视频怎么样", Status: ChatStatusDone},
	}
	out := f.svc.buildContext(conv, msgs, 4)
	require.Equal(t, "（请求生成视频）一只猫", out[1]["content"])
	require.Equal(t, "（已按要求生成视频）", out[2]["content"])
}
