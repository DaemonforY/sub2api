//go:build unit

package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCaptureTestContext(capture bool) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if capture {
		SetContentCaptureRequest(c, "user asks")
	}
	return c
}

func TestContentCapture_OnlyForMarkedRequests(t *testing.T) {
	c := newCaptureTestContext(false)
	observer := beginUpstreamResponseModelObservation(c)
	require.Nil(t, observer.content)
	observer.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"hi"}`), "response.output_text.delta")
	require.Nil(t, ContentCaptureSnapshot(c))

	c = newCaptureTestContext(true)
	first := beginUpstreamResponseModelObservation(c)
	first.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"from failed attempt"}`), "response.output_text.delta")
	// Failover: a new attempt starts a fresh accumulator.
	second := beginUpstreamResponseModelObservation(c)
	require.NotNil(t, second.content)
	second.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"final"}`), "response.output_text.delta")
	snap := ContentCaptureSnapshot(c)
	require.NotNil(t, snap)
	require.Equal(t, "user asks", snap.UserText)
	require.Equal(t, "final", snap.ReplyHead)
}

func TestContentCapture_ResponsesStream(t *testing.T) {
	c := newCaptureTestContext(true)
	o := beginUpstreamResponseModelObservation(c)
	events := []struct{ typ, data string }{
		{"response.created", `{"type":"response.created","response":{"model":"gpt-5","output":[]}}`},
		{"response.output_item.added", `{"type":"response.output_item.added","item":{"type":"function_call","name":"shell","arguments":""}}`},
		{"response.function_call_arguments.delta", `{"type":"response.function_call_arguments.delta","delta":"{\"comm"}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"function_call","name":"shell","arguments":"{\"command\":[\"ls\",\"-la\"]}","call_id":"c1"}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"custom_tool_call","name":"apply_patch","input":"*** Begin Patch"}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"local_shell_call","action":{"type":"exec","command":["git","status"]}}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"web_search_call","action":{"type":"search","query":"golang generics"}}}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"reasoning","summary":[]}}`},
		{"response.output_text.delta", `{"type":"response.output_text.delta","delta":"Hello "}`},
		{"response.output_text.delta", `{"type":"response.output_text.delta","delta":"world"}`},
		{"response.output_item.done", `{"type":"response.output_item.done","item":{"type":"message","content":[{"type":"output_text","text":"Hello world"}]}}`},
		// Passthrough: the terminal event may carry an empty output; nothing must be duplicated or lost.
		{"response.completed", `{"type":"response.completed","response":{"model":"gpt-5","output":[]}}`},
	}
	for _, e := range events {
		o.ObserveOpenAI([]byte(e.data), e.typ)
	}
	snap := ContentCaptureSnapshot(c)
	require.Equal(t, "Hello world", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{
		{Name: "shell", Arguments: `{"command":["ls","-la"]}`},
		{Name: "apply_patch", Arguments: "*** Begin Patch"},
		{Name: "local_shell", Arguments: `["git","status"]`},
		{Name: "web_search", Arguments: "golang generics"},
	}, snap.ToolCalls)
}

func TestContentCapture_ResponsesCompletedFallbackAndNonStream(t *testing.T) {
	c := newCaptureTestContext(true)
	o := beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"type":"response.completed","response":{"output":[
		{"type":"function_call","name":"read_file","arguments":"{\"path\":\"a\"}"},
		{"type":"message","content":[{"type":"output_text","text":"done"}]}]}}`), "response.completed")
	snap := ContentCaptureSnapshot(c)
	require.Equal(t, "done", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{{Name: "read_file", Arguments: `{"path":"a"}`}}, snap.ToolCalls)

	c = newCaptureTestContext(true)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"object":"response","model":"gpt-5","output":[{"type":"message","content":[{"type":"output_text","text":"plain"}]},
		{"type":"image_generation_call","result":"BASE64DATA"}]}`), "")
	snap = ContentCaptureSnapshot(c)
	require.Equal(t, "plain", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{{Name: "image_generation"}}, snap.ToolCalls)
}

func TestContentCapture_ChatStreamAndNonStream(t *testing.T) {
	c := newCaptureTestContext(true)
	o := beginUpstreamResponseModelObservation(c)
	chunks := []string{
		`{"object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"Let me "}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"check."}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"a","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"b","type":"function","function":{"name":"search","arguments":"{\"q\":"}}]}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":"}}]}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]}}]}`,
		`{"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"function":{"arguments":"\"x\"}"}}]}}]}`,
		`[DONE]`,
	}
	for _, chunk := range chunks {
		o.ObserveOpenAI([]byte(chunk), "")
	}
	snap := ContentCaptureSnapshot(c)
	require.Equal(t, "Let me check.", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{
		{Name: "get_weather", Arguments: `{"city":"Paris"}`},
		{Name: "search", Arguments: `{"q":"x"}`},
	}, snap.ToolCalls)

	c = newCaptureTestContext(true)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"hi",
		"tool_calls":[{"id":"x","type":"function","function":{"name":"f","arguments":"{}"}}]}}]}`), "")
	snap = ContentCaptureSnapshot(c)
	require.Equal(t, "hi", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{{Name: "f", Arguments: "{}"}}, snap.ToolCalls)
}

func TestContentCapture_AnthropicStreamAndNonStream(t *testing.T) {
	c := newCaptureTestContext(true)
	o := beginUpstreamResponseModelObservation(c)
	events := []string{
		`{"type":"message_start","message":{"model":"claude","content":[]}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"secret thoughts"}}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"I'll run "}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"it."}}`,
		`{"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"t","name":"Bash","input":{}}}`,
		`{"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"command\":"}}`,
		`{"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"\"ls\"}"}}`,
		`{"type":"content_block_stop","index":2}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use"}}`,
		`{"type":"message_stop"}`,
	}
	for _, e := range events {
		o.ObserveAnthropic([]byte(e))
	}
	snap := ContentCaptureSnapshot(c)
	require.Equal(t, "I'll run it.", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{{Name: "Bash", Arguments: `{"command":"ls"}`}}, snap.ToolCalls)

	c = newCaptureTestContext(true)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveAnthropic([]byte(`{"type":"message","model":"claude","content":[{"type":"text","text":"ok"},{"type":"tool_use","name":"Read","input":{"file":"a.go"}}]}`))
	snap = ContentCaptureSnapshot(c)
	require.Equal(t, "ok", snap.ReplyHead)
	require.Equal(t, []ContentToolCall{{Name: "Read", Arguments: `{"file":"a.go"}`}}, snap.ToolCalls)
}

func TestContentCapture_Caps(t *testing.T) {
	c := newCaptureTestContext(true)
	o := beginUpstreamResponseModelObservation(c)
	big := strings.Repeat("界", 10000) // 30000 bytes
	for i := 0; i < 5; i++ {
		o.ObserveOpenAI([]byte(`{"type":"response.output_text.delta","delta":"`+big+`"}`), "response.output_text.delta")
	}
	longArgs := strings.Repeat("a", 20000)
	for i := 0; i < 30; i++ {
		o.ObserveOpenAI([]byte(`{"type":"response.output_item.done","item":{"type":"function_call","name":"f","arguments":"`+longArgs+`"}}`), "response.output_item.done")
	}
	snap := ContentCaptureSnapshot(c)
	require.LessOrEqual(t, len(snap.ReplyHead), contentCaptureReplyMaxBytes)
	require.True(t, utf8.ValidString(snap.ReplyHead))
	require.Len(t, snap.ToolCalls, ContentLogMaxToolCalls)
	for _, call := range snap.ToolCalls {
		require.LessOrEqual(t, len(call.Arguments), contentCaptureToolArgsMaxBytes)
	}

	// Garbage never panics and never records anything.
	c = newCaptureTestContext(true)
	o = beginUpstreamResponseModelObservation(c)
	o.ObserveOpenAI([]byte(`{"choices":[{"delta":{"tool_calls":"nope"}}]`), "")
	o.ObserveAnthropic([]byte(`{"type":"content_block_delta","index":"x","delta":{"type":"input_json_delta"}}`))
	o.ObserveOpenAI(nil, "")
	snap = ContentCaptureSnapshot(c)
	require.Empty(t, snap.ReplyHead)
	require.Empty(t, snap.ToolCalls)
}

func anthropicToolSSE() string {
	return strings.Join([]string{
		`event: message_start`,
		`data: {"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"model":"claude-x","usage":{"input_tokens":3}}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Running it."}}`,
		``,
		`event: content_block_start`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"t1","name":"Bash","input":{}}}`,
		``,
		`event: content_block_delta`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\"}"}}`,
		``,
		`event: message_delta`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}`,
		``,
		`event: message_stop`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")
}

// Anthropic upstream behind /v1/chat/completions and /v1/responses feeds the capture too, without
// starting model observation for requests that aren't captured.
func TestContentCapture_AnthropicUpstreamConversionPaths(t *testing.T) {
	want := []ContentToolCall{{Name: "Bash", Arguments: `{"command":"ls"}`}}
	run := func(name string, call func(c *gin.Context, resp *http.Response) error) {
		t.Run(name, func(t *testing.T) {
			c := newCaptureTestContext(true)
			beginContentCaptureOnly(c)
			resp := &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(anthropicToolSSE()))}
			require.NoError(t, call(c, resp))
			snap := ContentCaptureSnapshot(c)
			require.NotNil(t, snap)
			require.Equal(t, "Running it.", snap.ReplyHead)
			require.Equal(t, want, snap.ToolCalls)
		})
	}
	svc := &GatewayService{}
	run("cc stream", func(c *gin.Context, resp *http.Response) error {
		_, err := svc.handleCCStreamingFromAnthropic(resp, c, "gpt-5", "claude-x", nil, time.Now(), false)
		return err
	})
	run("cc buffered", func(c *gin.Context, resp *http.Response) error {
		_, err := svc.handleCCBufferedFromAnthropic(resp, c, "gpt-5", "claude-x", nil, time.Now())
		return err
	})
	run("responses stream", func(c *gin.Context, resp *http.Response) error {
		_, err := svc.handleResponsesStreamingResponse(resp, c, "gpt-5", "claude-x", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
		return err
	})
	run("responses buffered", func(c *gin.Context, resp *http.Response) error {
		_, err := svc.handleResponsesBufferedStreamingResponse(resp, c, "gpt-5", "claude-x", nil, time.Now(), apicompat.ResponsesClientToolMapping{})
		return err
	})

	c := newCaptureTestContext(false)
	beginContentCaptureOnly(c)
	require.Nil(t, upstreamResponseModelObserverFromContext(c), "uncaptured requests get no observer here")
}

// The buffered OpenAI-compat paths (non-stream chat/completions and /v1/messages on OpenAI groups)
// read the whole upstream stream themselves and hand the terminal Responses object over at the end.
func TestContentCapture_BufferedFinalResponse(t *testing.T) {
	c := newCaptureTestContext(true)
	observer := beginUpstreamResponseModelObservation(c)
	observer.ObserveContentFinalResponse(&apicompat.ResponsesResponse{
		Object: "response",
		Model:  "gpt-6-luna",
		Status: "completed",
		Output: []apicompat.ResponsesOutput{
			{Type: "reasoning"},
			{Type: "message", Role: "assistant", Content: []apicompat.ResponsesContentPart{{Type: "output_text", Text: "杭州是一座城市。"}}},
			{Type: "function_call", Name: "run_shell", Arguments: `{"command":"ls"}`},
		},
	})
	snap := ContentCaptureSnapshot(c)
	require.NotNil(t, snap)
	require.Equal(t, "杭州是一座城市。", snap.ReplyHead)
	require.Len(t, snap.ToolCalls, 1)
	require.Equal(t, "run_shell", snap.ToolCalls[0].Name)

	// Not captured: nothing happens.
	plain := newCaptureTestContext(false)
	beginUpstreamResponseModelObservation(plain).ObserveContentFinalResponse(&apicompat.ResponsesResponse{Object: "response"})
	require.Nil(t, ContentCaptureSnapshot(plain))
}
