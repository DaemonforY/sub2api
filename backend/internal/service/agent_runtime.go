package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// A small tool-calling loop for features where the model looks things up before it answers (first
// the support assistant). Each model call is one streamed chat completion through this site's own
// gateway with a key the caller picked, so usage and billing go through the normal path. The model
// may call tools for up to MaxModelCalls-1 rounds; the last call is made without tools, so a run
// always ends in an answer. Tools run in-process, one at a time, each under a timeout; what they
// return is sent back to the model as JSON, cut to a length.

const (
	agentDefaultModelCalls = 4
	agentToolTimeout       = 10 * time.Second
	agentMaxToolResult     = 6000 // runes sent back to the model per tool call
	agentMaxStepLog        = 2000 // runes of a tool result kept in the run log
)

// AgentTool is a function the model may call. Parameters is a JSON schema (an object). Label is
// what the user sees while it runs (正在查询…). Run gets the model's arguments as JSON and returns
// something JSON-encodable; an error is reported to the model, which can then answer without it.
type AgentTool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Label       string
	Run         func(ctx context.Context, args json.RawMessage) (any, error)
}

// AgentMessage is a chat-completions message, including assistant tool calls and tool results.
type AgentMessage struct {
	Role       string          `json:"role"`
	Content    *string         `json:"content"`
	ToolCalls  []AgentToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

func agentText(role, text string) AgentMessage { return AgentMessage{Role: role, Content: &text} }

type AgentToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function AgentFunctionCall `json:"function"`
}

type AgentFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AgentRunInput struct {
	GatewayURL      string
	UserAgent       string
	Key             string
	Model           string
	Messages        []AgentMessage
	Tools           []AgentTool
	MaxModelCalls   int
	MaxOutputTokens int
	// OnDelta gets the answer text as it streams. OnTool is called as a tool starts (may be nil).
	OnDelta func(string) error
	OnTool  func(name, label string) error
}

// AgentStep is one tool call, for the run log.
type AgentStep struct {
	Tool   string `json:"tool"`
	Args   string `json:"args,omitempty"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
	Ms     int64  `json:"ms"`
}

// AgentRunResult is returned even when the run fails part-way, so what happened can be logged.
type AgentRunResult struct {
	Text             string
	Model            string
	Steps            []AgentStep
	ModelCalls       int
	PromptTokens     int
	CompletionTokens int
}

// RunAgent runs the loop. The error is user-facing (errLearnRunFailed) or the caller's own (OnDelta).
func RunAgent(ctx context.Context, client *http.Client, in AgentRunInput) (*AgentRunResult, error) {
	maxCalls := in.MaxModelCalls
	if maxCalls <= 0 {
		maxCalls = agentDefaultModelCalls
	}
	tools := map[string]AgentTool{}
	var specs []map[string]any
	for _, t := range in.Tools {
		tools[t.Name] = t
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		specs = append(specs, map[string]any{"type": "function", "function": map[string]any{
			"name": t.Name, "description": t.Description, "parameters": params}})
	}

	out := &AgentRunResult{}
	msgs := append([]AgentMessage(nil), in.Messages...)
	// Text from an earlier round (e.g. "我帮你查一下") stays; the next round starts a new paragraph.
	pendingBreak := false
	onDelta := func(d string) error {
		if pendingBreak {
			pendingBreak = false
			if !strings.HasSuffix(out.Text, "\n\n") {
				out.Text += "\n\n"
				if in.OnDelta != nil {
					if err := in.OnDelta("\n\n"); err != nil {
						return err
					}
				}
			}
		}
		out.Text += d
		if in.OnDelta != nil {
			return in.OnDelta(d)
		}
		return nil
	}
	for call := 0; call < maxCalls; call++ {
		offer := specs
		if call == maxCalls-1 {
			offer = nil
		}
		turn, err := agentCompletion(ctx, client, in, msgs, offer, onDelta)
		out.ModelCalls++
		if turn != nil {
			out.PromptTokens += turn.promptTokens
			out.CompletionTokens += turn.completionTokens
			if turn.model != "" {
				out.Model = turn.model
			}
		}
		if err != nil {
			return out, err
		}
		if len(turn.calls) == 0 || offer == nil {
			if strings.TrimSpace(out.Text) == "" {
				return out, errLearnRunFailed("模型没有返回内容")
			}
			return out, nil
		}
		msgs = append(msgs, AgentMessage{Role: "assistant", Content: agentNilIfEmpty(turn.text), ToolCalls: turn.calls})
		for _, tc := range turn.calls {
			result, step, err := runAgentTool(ctx, tools, tc, in.OnTool)
			out.Steps = append(out.Steps, step)
			if err != nil {
				return out, err
			}
			msgs = append(msgs, AgentMessage{Role: "tool", Content: &result, ToolCallID: tc.ID})
		}
		if out.Text != "" {
			pendingBreak = true
		}
	}
	return out, nil
}

func agentNilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// runAgentTool runs one call and returns what goes back to the model plus the log entry. A tool's
// own failure is reported to the model; the error is only onTool's (the client went away).
func runAgentTool(ctx context.Context, tools map[string]AgentTool, tc AgentToolCall, onTool func(name, label string) error) (string, AgentStep, error) {
	step := AgentStep{Tool: tc.Function.Name, Args: agentCut(tc.Function.Arguments, 500)}
	started := time.Now()
	fail := func(msg string) (string, AgentStep, error) {
		step.Error = msg
		step.Ms = time.Since(started).Milliseconds()
		b, _ := json.Marshal(map[string]string{"error": msg})
		return string(b), step, nil
	}
	tool, ok := tools[tc.Function.Name]
	if !ok {
		return fail("没有这个工具：" + tc.Function.Name)
	}
	if onTool != nil {
		if err := onTool(tool.Name, tool.Label); err != nil {
			step.Error = "已中断"
			return "", step, err
		}
	}
	args := json.RawMessage(strings.TrimSpace(tc.Function.Arguments))
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	if !json.Valid(args) {
		return fail("参数不是合法的 JSON")
	}
	tctx, cancel := context.WithTimeout(ctx, agentToolTimeout)
	defer cancel()
	v, err := func() (v any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("工具内部错误")
			}
		}()
		return tool.Run(tctx, args)
	}()
	if err != nil {
		return fail(err.Error())
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fail("结果无法编码")
	}
	step.Ms = time.Since(started).Milliseconds()
	step.Result = agentCut(string(b), agentMaxStepLog)
	return agentCut(string(b), agentMaxToolResult), step, nil
}

func agentCut(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…(已截断)"
}

type agentTurn struct {
	text             string
	calls            []AgentToolCall
	model            string
	promptTokens     int
	completionTokens int
}

// agentCompletion makes one streamed call; text goes to onDelta, tool calls are assembled.
func agentCompletion(ctx context.Context, client *http.Client, in AgentRunInput, msgs []AgentMessage, tools []map[string]any, onDelta func(string) error) (*agentTurn, error) {
	body := map[string]any{
		"model":          in.Model,
		"messages":       msgs,
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if in.MaxOutputTokens > 0 {
		body["max_completion_tokens"] = in.MaxOutputTokens
	}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, in.GatewayURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+in.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", in.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errLearnRunFailed("连接模型服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}

	turn := &agentTurn{}
	type partial struct {
		id, name string
		args     strings.Builder
	}
	parts := map[int]*partial{}
	var text strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}
		if chunk.Error != nil {
			return turn, errLearnRunFailed(upstreamErrorText([]byte(data), http.StatusBadGateway))
		}
		if chunk.Model != "" {
			turn.model = chunk.Model
		}
		if chunk.Usage != nil {
			turn.promptTokens, turn.completionTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
		for _, c := range chunk.Choices {
			for _, tc := range c.Delta.ToolCalls {
				p := parts[tc.Index]
				if p == nil {
					p = &partial{}
					parts[tc.Index] = p
				}
				if tc.ID != "" {
					p.id = tc.ID
				}
				if tc.Function.Name != "" {
					p.name += tc.Function.Name
				}
				_, _ = p.args.WriteString(tc.Function.Arguments)
			}
			if c.Delta.Content == "" {
				continue
			}
			_, _ = text.WriteString(c.Delta.Content)
			if err := onDelta(c.Delta.Content); err != nil {
				return turn, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return turn, ctx.Err()
		}
		return turn, errLearnRunFailed("读取结果中断")
	}
	turn.text = text.String()
	indexes := make([]int, 0, len(parts))
	for i := range parts {
		indexes = append(indexes, i)
	}
	sort.Ints(indexes)
	for n, i := range indexes {
		p := parts[i]
		if p.name == "" {
			continue
		}
		id := p.id
		if id == "" {
			id = fmt.Sprintf("call_%d", n)
		}
		turn.calls = append(turn.calls, AgentToolCall{ID: id, Type: "function", Function: AgentFunctionCall{Name: p.name, Arguments: p.args.String()}})
	}
	return turn, nil
}
