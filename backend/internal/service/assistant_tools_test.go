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

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type assistantUsersStub struct{ users map[int64]*User }

func (s assistantUsersStub) GetByID(_ context.Context, id int64) (*User, error) {
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

type assistantUsageStub struct{ logs []UsageLog }

func (s assistantUsageStub) ListByUser(_ context.Context, userID int64, _ pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error) {
	out := []UsageLog{}
	for _, l := range s.logs {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	return out, nil, nil
}

type assistantErrorsStub struct {
	items  map[int64][]*UserErrorRequest
	filter *OpsErrorLogFilter
}

func (s *assistantErrorsStub) ListUserErrorRequests(_ context.Context, userID int64, f *OpsErrorLogFilter) (*UserErrorRequestList, error) {
	s.filter = f
	return &UserErrorRequestList{Items: s.items[userID], Total: len(s.items[userID])}, nil
}

type assistantErrorViewStub bool

func (v assistantErrorViewStub) IsUserErrorViewAllowed(context.Context) bool { return bool(v) }

type agentRunsStub struct {
	mu   sync.Mutex
	runs []AgentRunRecord
}

func (s *agentRunsStub) Create(_ context.Context, run *AgentRunRecord) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs = append(s.runs, *run)
	return int64(len(s.runs)), nil
}

func (s *agentRunsStub) List(_ context.Context, agent string, _, _ int) ([]AgentRunRecord, int, error) {
	return s.runs, len(s.runs), nil
}

func (s *agentRunsStub) DeleteBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func newAssistantAgentForTest(t *testing.T, gw string, errorView bool) (*AssistantService, *assistantErrorsStub, *agentRunsStub) {
	t.Helper()
	gpt := &Group{Name: "GPT 官方", Platform: PlatformOpenAI}
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 1, Name: "客服", Key: "sk-admin", Status: StatusActive},
		{ID: 20, UserID: 5, Name: "codex", Key: "sk-user5-secret", Status: StatusActive, Group: gpt, Quota: 10, QuotaUsed: 2.5},
		{ID: 21, UserID: 5, Name: "旧的", Key: "sk-user5-old", Status: StatusAPIKeyDisabled},
		{ID: 30, UserID: 6, Name: "别人的", Key: "sk-user6-secret", Status: StatusActive, Group: gpt},
	}}
	learn, _ := newL2Service(t, gw, LearnSources{Keys: keys})
	svc := NewAssistantService(learn, mapSettings{SettingKeySiteName: "HiveGPT"}, &assistantQuotaStub{n: map[string]int64{}})
	svc.SetPages(func() []AssistantPage { return assistantTestPages })
	errs := &assistantErrorsStub{items: map[int64][]*UserErrorRequest{
		5: {{CreatedAt: time.Date(2026, 10, 9, 2, 0, 0, 0, time.UTC), Model: "gpt-4o", StatusCode: 400, Category: "model_unsupported", Message: "分组不支持模型 gpt-4o", KeyName: "codex"}},
		6: {{Model: "x", Message: "user 6 only"}},
	}}
	runs := &agentRunsStub{}
	svc.SetSources(AssistantSources{
		Users:     assistantUsersStub{users: map[int64]*User{5: {ID: 5, Balance: 12.345678, Status: StatusActive, Concurrency: 3}, 6: {ID: 6, Balance: 99}}},
		Usage:     assistantUsageStub{logs: []UsageLog{{UserID: 5, APIKeyID: 20, Model: "gpt-5.5", InputTokens: 1200, OutputTokens: 300, ActualCost: 0.0123}, {UserID: 6, Model: "other"}}},
		Errors:    errs,
		ErrorView: assistantErrorViewStub(errorView),
		Runs:      runs,
	})
	return svc, errs, runs
}

func runAssistantTool(t *testing.T, tools []AgentTool, name, args string) (string, error) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			v, err := tool.Run(context.Background(), json.RawMessage(args))
			if err != nil {
				return "", err
			}
			b, _ := json.Marshal(v)
			return string(b), nil
		}
	}
	t.Fatalf("no tool %s", name)
	return "", nil
}

func TestAssistantToolsScopedToAsker(t *testing.T) {
	ctx := context.Background()
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		require.Equal(t, "Bearer sk-user5-secret", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"data":[{"id":"gpt-5.5"},{"id":"gpt-5.6-terra"}]}`)
	}))
	defer gw.Close()
	svc, errs, _ := newAssistantAgentForTest(t, gw.URL, true)

	names := func(tools []AgentTool) []string {
		var out []string
		for _, tool := range tools {
			out = append(out, tool.Name)
		}
		return out
	}
	require.Equal(t, []string{"search_docs"}, names(svc.assistantTools(ctx, 0)), "visitors only search the docs")
	tools := svc.assistantTools(ctx, 5)
	require.ElementsMatch(t, []string{"search_docs", "get_my_account", "list_my_keys", "list_key_models", "get_recent_errors", "get_recent_usage"}, names(tools))

	out, err := runAssistantTool(t, tools, "get_my_account", `{}`)
	require.NoError(t, err)
	require.Contains(t, out, `"balance_usd":12.3457`)
	require.NotContains(t, out, "99")

	out, err = runAssistantTool(t, tools, "list_my_keys", `{}`)
	require.NoError(t, err)
	require.Contains(t, out, `"codex"`)
	require.Contains(t, out, `已停用`)
	require.Contains(t, out, `GPT 官方`)
	require.NotContains(t, out, "别人的")
	require.NotContains(t, out, "sk-", "key secrets never go to the model")

	out, err = runAssistantTool(t, tools, "list_key_models", `{"key_id":20}`)
	require.NoError(t, err)
	require.Contains(t, out, "gpt-5.6-terra")
	require.NotContains(t, out, "sk-")
	_, err = runAssistantTool(t, tools, "list_key_models", `{"key_id":30}`)
	require.ErrorContains(t, err, "没有找到这个 Key", "another user's key")
	out, err = runAssistantTool(t, tools, "list_key_models", `{"key_id":21}`)
	require.NoError(t, err)
	require.Contains(t, out, "当前不可用")

	out, err = runAssistantTool(t, tools, "get_recent_errors", `{"hours":9999,"model":" gpt-4o "}`)
	require.NoError(t, err)
	require.Contains(t, out, "分组不支持模型 gpt-4o")
	require.Contains(t, out, `"time":"2026-10-09 10:00"`, "times are Beijing time")
	require.NotContains(t, out, "user 6 only")
	require.Equal(t, "gpt-4o", errs.filter.Model)
	require.Equal(t, 10, errs.filter.PageSize)

	out, err = runAssistantTool(t, tools, "get_recent_usage", `{}`)
	require.NoError(t, err)
	require.Contains(t, out, `"key":"codex"`)
	require.Contains(t, out, `"cost_usd":0.0123`)
	require.NotContains(t, out, "other")

	out, err = runAssistantTool(t, tools, "search_docs", `{"query":"公众号 AppSecret"}`)
	require.NoError(t, err)
	require.Contains(t, out, "/learn/editor/")

	// With 错误请求 hidden from users, the assistant doesn't show them either.
	svc2, _, _ := newAssistantAgentForTest(t, gw.URL, false)
	require.NotContains(t, names(svc2.assistantTools(ctx, 5)), "get_recent_errors")
}

func TestAssistantChatWithTools(t *testing.T) {
	ctx := context.Background()
	replies := []string{
		sseToolCall("c1", "get_recent_errors", `{}`),
		sseToolCall("c2", "list_my_keys", `{}`),
		sseText("你最近用 codex 这个 Key 调 gpt-4o 时报了「分组不支持模型」。"),
	}
	var mu sync.Mutex
	var bodies []map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		n := len(bodies)
		bodies = append(bodies, body)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		if n >= len(replies) {
			_, _ = io.WriteString(w, sseText("好的。"))
			return
		}
		_, _ = io.WriteString(w, replies[n])
	}))
	defer gw.Close()
	svc, _, runs := newAssistantAgentForTest(t, gw.URL, true)
	_, err := svc.SaveSettings(ctx, 1, AssistantSettings{Enabled: true, Tools: true, Model: "gpt-5.6-terra", KeyID: 7, UserPerDay: 5, GuestPerDay: 2, DailyCap: 10})
	require.NoError(t, err)
	require.True(t, svc.Settings(ctx).Tools)
	require.True(t, svc.Config(ctx, AssistantAsker{UserID: 5}).Tools)
	require.False(t, svc.Config(ctx, AssistantAsker{IP: "203.0.113.1"}).Tools, "visitors can't look up accounts")

	var labels []string
	var text strings.Builder
	res, err := svc.Chat(ctx, AssistantAsker{UserID: 5}, AssistantChatInput{Messages: []LearnMessage{{Role: "user", Content: "为什么报分组不支持模型？"}}},
		func(d string) error { text.WriteString(d); return nil },
		func(label string) error { labels = append(labels, label); return nil })
	require.NoError(t, err)
	require.Equal(t, 4, res.Left)
	require.Contains(t, text.String(), "分组不支持模型")
	require.Equal(t, []string{"正在查你最近的报错", "正在查看你的 API Key"}, labels)
	require.Len(t, bodies, 3)
	system := bodies[0]["messages"].([]any)[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "list_key_models")
	require.NotContains(t, system, "没有开放报错明细查询")

	require.Len(t, runs.runs, 1)
	run := runs.runs[0]
	require.Equal(t, AgentSupport, run.Agent)
	require.Equal(t, int64(5), *run.UserID)
	require.Equal(t, "为什么报分组不支持模型？", run.Question)
	require.Equal(t, "ok", run.Status)
	require.Equal(t, 3, run.ModelCalls)
	require.Len(t, run.Steps, 2)
	require.Contains(t, run.Steps[0].Result, "分组不支持模型 gpt-4o")
	for _, s := range run.Steps {
		require.NotContains(t, s.Result, "sk-")
	}

	// Visitors: one call, no tools, but still logged.
	bodies = nil
	replies = nil
	_, err = svc.Chat(ctx, AssistantAsker{IP: "203.0.113.1"}, AssistantChatInput{Messages: []LearnMessage{{Role: "user", Content: "怎么创建 API Key？"}}},
		func(string) error { return nil }, nil)
	require.NoError(t, err)
	require.Len(t, bodies, 1)
	require.Nil(t, bodies[0]["tools"])
	system = bodies[0]["messages"].([]any)[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "不要声称自己能查询账户")
	require.Len(t, runs.runs, 2)
	require.Nil(t, runs.runs[1].UserID)

	// 账户诊断 off: signed-in users get the plain answer too.
	_, err = svc.SaveSettings(ctx, 1, AssistantSettings{Enabled: true, Tools: false, Model: "gpt-5.6-terra", KeyID: 7, UserPerDay: 5, GuestPerDay: 2, DailyCap: 10})
	require.NoError(t, err)
	bodies = nil
	_, err = svc.Chat(ctx, AssistantAsker{UserID: 5}, AssistantChatInput{Messages: []LearnMessage{{Role: "user", Content: "我的余额"}}},
		func(string) error { return nil }, nil)
	require.NoError(t, err)
	require.Len(t, bodies, 1)
	require.Nil(t, bodies[0]["tools"])

	list, total, err := svc.Runs(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 3, total)
	require.Len(t, list, 3)
}

func TestAssistantPromptWithoutErrorLookup(t *testing.T) {
	ctx := context.Background()
	var body map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sseText("请把状态码和错误信息发给我。"))
	}))
	defer gw.Close()
	svc, _, _ := newAssistantAgentForTest(t, gw.URL, false)
	_, err := svc.SaveSettings(ctx, 1, AssistantSettings{Enabled: true, Tools: true, Model: "m", KeyID: 7, UserPerDay: 5, GuestPerDay: 2, DailyCap: 10})
	require.NoError(t, err)
	_, err = svc.Chat(ctx, AssistantAsker{UserID: 5}, AssistantChatInput{Messages: []LearnMessage{{Role: "user", Content: "为什么报错"}}}, func(string) error { return nil }, nil)
	require.NoError(t, err)
	system := body["messages"].([]any)[0].(map[string]any)["content"].(string)
	require.Contains(t, system, "没有开放报错明细查询", "the model must not claim there were no errors")
	for _, tool := range body["tools"].([]any) {
		require.NotEqual(t, "get_recent_errors", tool.(map[string]any)["function"].(map[string]any)["name"])
	}
}
