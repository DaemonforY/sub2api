//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// agentGateway is a fake /v1/chat/completions: each request gets the next scripted SSE body.
type agentGateway struct {
	mu       sync.Mutex
	replies  []string
	requests []map[string]any
}

func (g *agentGateway) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		g.mu.Lock()
		n := len(g.requests)
		g.requests = append(g.requests, body)
		g.mu.Unlock()
		if n >= len(g.replies) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(g.replies[n], "HTTP ") {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"message":"upstream down"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, g.replies[n])
	}))
	t.Cleanup(srv.Close)
	return srv
}

func sseText(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		chunk, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"content": p}}}})
		b.WriteString("data: " + string(chunk) + "\n\n")
	}
	b.WriteString(`data: {"choices":[],"usage":{"prompt_tokens":100,"completion_tokens":10},"model":"gpt-test"}` + "\n\ndata: [DONE]\n\n")
	return b.String()
}

// sseToolCall streams one tool call with its arguments split over two chunks, as upstreams do.
func sseToolCall(id, name, args string) string {
	half := len(args) / 2
	first, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{
		map[string]any{"index": 0, "id": id, "type": "function", "function": map[string]any{"name": name, "arguments": args[:half]}}}}}}})
	second, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": map[string]any{"tool_calls": []any{
		map[string]any{"index": 0, "function": map[string]any{"arguments": args[half:]}}}}}}})
	return "data: " + string(first) + "\n\ndata: " + string(second) + "\n\n" +
		`data: {"choices":[],"usage":{"prompt_tokens":50,"completion_tokens":5}}` + "\n\ndata: [DONE]\n\n"
}

func TestRunAgentToolLoop(t *testing.T) {
	gw := &agentGateway{replies: []string{
		sseToolCall("call_1", "lookup", `{"city":"上海"}`),
		sseToolCall("call_2", "missing_tool", `{}`),
		sseText("上海", "今天晴。"),
	}}
	srv := gw.serve(t)

	var gotArgs string
	var labels []string
	var streamed strings.Builder
	res, err := RunAgent(context.Background(), srv.Client(), AgentRunInput{
		GatewayURL: srv.URL, Key: "sk-test", Model: "gpt-test",
		Messages: []AgentMessage{agentText("system", "规则"), agentText("user", "上海天气？")},
		Tools: []AgentTool{{Name: "lookup", Label: "正在查天气", Run: func(_ context.Context, args json.RawMessage) (any, error) {
			gotArgs = string(args)
			return map[string]string{"weather": "晴"}, nil
		}}},
		OnDelta: func(d string) error { streamed.WriteString(d); return nil },
		OnTool:  func(_, label string) error { labels = append(labels, label); return nil },
	})
	require.NoError(t, err)
	require.Equal(t, "上海今天晴。", res.Text)
	require.Equal(t, res.Text, streamed.String())
	require.Equal(t, `{"city":"上海"}`, gotArgs)
	require.Equal(t, []string{"正在查天气"}, labels)
	require.Equal(t, 3, res.ModelCalls)
	require.Equal(t, 200, res.PromptTokens)
	require.Equal(t, "gpt-test", res.Model)
	require.Len(t, res.Steps, 2)
	require.Equal(t, "lookup", res.Steps[0].Tool)
	require.Contains(t, res.Steps[0].Result, "晴")
	require.Contains(t, res.Steps[1].Error, "没有这个工具")

	// The second request carries the assistant's tool call and the tool's JSON result.
	msgs := gw.requests[1]["messages"].([]any)
	require.Len(t, msgs, 4)
	call := msgs[2].(map[string]any)
	require.Equal(t, "assistant", call["role"])
	require.Nil(t, call["content"])
	require.Equal(t, "call_1", call["tool_calls"].([]any)[0].(map[string]any)["id"])
	result := msgs[3].(map[string]any)
	require.Equal(t, "tool", result["role"])
	require.Equal(t, "call_1", result["tool_call_id"])
	require.JSONEq(t, `{"weather":"晴"}`, result["content"].(string))
	require.NotNil(t, gw.requests[0]["tools"])
	require.NotNil(t, gw.requests[2]["tools"], "calls before the last still offer tools")
}

func TestRunAgentLastCallHasNoTools(t *testing.T) {
	gw := &agentGateway{replies: []string{
		sseToolCall("c1", "lookup", `{}`),
		sseText("根据查询结果，", "答案是 42。"),
	}}
	srv := gw.serve(t)
	runs := 0
	res, err := RunAgent(context.Background(), srv.Client(), AgentRunInput{
		GatewayURL: srv.URL, Model: "m", MaxModelCalls: 2,
		Messages: []AgentMessage{agentText("user", "q")},
		Tools: []AgentTool{{Name: "lookup", Run: func(context.Context, json.RawMessage) (any, error) {
			runs++
			return nil, errors.New("暂时查不到")
		}}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, runs)
	require.Equal(t, "暂时查不到", res.Steps[0].Error, "a tool's error goes back to the model")
	require.Nil(t, gw.requests[1]["tools"], "the last call must answer")
	require.Equal(t, "根据查询结果，答案是 42。", res.Text)
}

func TestRunAgentEdges(t *testing.T) {
	ctx := context.Background()

	// Text before a tool call is kept, and the answer after it starts a new paragraph.
	gw := &agentGateway{replies: []string{
		strings.Replace(sseToolCall("c1", "lookup", `not json`), "data: [DONE]", `data: {"choices":[{"delta":{"content":"我查一下。"}}]}`+"\n\ndata: [DONE]", 1),
		sseText("查到了。"),
	}}
	srv := gw.serve(t)
	ran := false
	res, err := RunAgent(ctx, srv.Client(), AgentRunInput{GatewayURL: srv.URL, Model: "m", Messages: []AgentMessage{agentText("user", "q")},
		Tools: []AgentTool{{Name: "lookup", Run: func(context.Context, json.RawMessage) (any, error) { ran = true; return "x", nil }}}})
	require.NoError(t, err)
	require.False(t, ran, "invalid JSON arguments are refused")
	require.Equal(t, "参数不是合法的 JSON", res.Steps[0].Error)
	require.Equal(t, "我查一下。\n\n查到了。", res.Text)

	// An upstream error ends the run with what happened so far.
	gw = &agentGateway{replies: []string{sseToolCall("c1", "lookup", `{}`), "HTTP 503"}}
	srv = gw.serve(t)
	res, err = RunAgent(ctx, srv.Client(), AgentRunInput{GatewayURL: srv.URL, Model: "m", Messages: []AgentMessage{agentText("user", "q")},
		Tools: []AgentTool{{Name: "lookup", Run: func(context.Context, json.RawMessage) (any, error) { return "x", nil }}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "upstream down")
	require.Len(t, res.Steps, 1)
	require.Equal(t, 2, res.ModelCalls)

	// The client going away (OnTool fails) stops the run before the tool runs.
	gw = &agentGateway{replies: []string{sseToolCall("c1", "lookup", `{}`)}}
	srv = gw.serve(t)
	gone := errors.New("client gone")
	_, err = RunAgent(ctx, srv.Client(), AgentRunInput{GatewayURL: srv.URL, Model: "m", Messages: []AgentMessage{agentText("user", "q")},
		Tools:  []AgentTool{{Name: "lookup", Run: func(context.Context, json.RawMessage) (any, error) { t.Fatal("must not run"); return nil, nil }}},
		OnTool: func(string, string) error { return gone }})
	require.ErrorIs(t, err, gone)

	// A tool that panics is reported as a failure, and big results are cut.
	gw = &agentGateway{replies: []string{sseToolCall("c1", "boom", `{}`), sseToolCall("c2", "big", `{}`), sseText("好")}}
	srv = gw.serve(t)
	res, err = RunAgent(ctx, srv.Client(), AgentRunInput{GatewayURL: srv.URL, Model: "m", Messages: []AgentMessage{agentText("user", "q")},
		Tools: []AgentTool{
			{Name: "boom", Run: func(context.Context, json.RawMessage) (any, error) { panic("x") }},
			{Name: "big", Run: func(context.Context, json.RawMessage) (any, error) {
				return strings.Repeat("长", agentMaxToolResult*2), nil
			}},
		}})
	require.NoError(t, err)
	require.Equal(t, "工具内部错误", res.Steps[0].Error)
	sent := gw.requests[2]["messages"].([]any)
	require.LessOrEqual(t, len([]rune(sent[len(sent)-1].(map[string]any)["content"].(string))), agentMaxToolResult+10)
	require.LessOrEqual(t, len([]rune(res.Steps[1].Result)), agentMaxStepLog+10)

	// No text at all is an error.
	gw = &agentGateway{replies: []string{sseText()}}
	srv = gw.serve(t)
	_, err = RunAgent(ctx, srv.Client(), AgentRunInput{GatewayURL: srv.URL, Model: "m", Messages: []AgentMessage{agentText("user", "q")}})
	require.Error(t, err)
}
