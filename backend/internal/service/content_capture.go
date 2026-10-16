package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// Content capture for 请求内容日志. Only requests of accounts an admin turned on carry a
// contentCaptureRequest in their gin.Context (set by the gateway capture middleware); for every
// other request the observer has no accumulator and nothing here runs. Capture never changes what
// is forwarded or returned, and every entry point is nil-safe and recovers from panics.

const (
	contentCaptureContextKey = "content_log_capture_request"

	// Raw (unredacted) amounts kept in memory. Redaction runs later in the log writer, then the
	// stored text is cut to the ContentLog*Runes limits; the slack keeps a secret that straddles
	// the cut from surviving half-masked.
	contentCaptureUserTextMaxBytes  = 16 << 10
	contentCaptureReplyMaxBytes     = 8 << 10
	contentCaptureToolArgsMaxBytes  = 4 << 10
	contentCaptureToolNameMaxBytes  = 200
	ContentLogMaxToolCalls          = 20
	ContentLogUserTextMaxRunes      = 2000
	ContentLogReplyHeadMaxRunes     = 1000
	ContentLogToolArgumentsMaxRunes = 300
)

// ContentToolCall is one tool/function call the model emitted.
type ContentToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ContentCapture is what one request contributed to the content log: the latest user message,
// the model's tool calls and the head of its text reply. Values are raw and bounded; the content
// log writer redacts and truncates them before storage.
type ContentCapture struct {
	UserText  string
	ToolCalls []ContentToolCall
	ReplyHead string

	// Archive is the 对话归档 capture (request body reference + the full response); nil unless
	// archiving was on when the request arrived.
	Archive *ArchiveCapture
	// archiveOnly is set when the request was captured for the archive only (not a 请求内容日志
	// target): the content log ignores it.
	archiveOnly bool
}

type contentCaptureRequest struct {
	userText string
}

// SetContentCaptureRequest marks the request for content capture (gateway middleware, target
// accounts only). userText is the latest user message, already extracted from the request body.
func SetContentCaptureRequest(c *gin.Context, userText string) {
	if c == nil {
		return
	}
	c.Set(contentCaptureContextKey, &contentCaptureRequest{userText: truncateBytesUTF8(userText, contentCaptureUserTextMaxBytes)})
}

func contentCaptureRequestFromContext(c *gin.Context) *contentCaptureRequest {
	if c == nil {
		return nil
	}
	value, ok := c.Get(contentCaptureContextKey)
	if !ok {
		return nil
	}
	req, _ := value.(*contentCaptureRequest)
	return req
}

// ContentCaptureSnapshot returns the request's capture (latest user message + what the final
// upstream attempt returned), or nil when the request isn't being captured. Call it after the
// forward finished, from the request goroutine.
func ContentCaptureSnapshot(c *gin.Context) (out *ContentCapture) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	req := contentCaptureRequestFromContext(c)
	archive := archiveRequestFromContext(c)
	if req == nil && archive == nil {
		return nil
	}
	out = &ContentCapture{}
	if req != nil {
		out.UserText = req.userText
	} else {
		out.archiveOnly = true
	}
	var acc *contentAccumulator
	if observer := upstreamResponseModelObserverFromContext(c); observer != nil && observer.content != nil {
		acc = observer.content
		if req != nil {
			out.ToolCalls, out.ReplyHead = acc.snapshot()
		}
	}
	if archive != nil {
		out.Archive = archive.snapshot(acc)
	}
	return out
}

// contentCaptureActive reports whether the request needs a response accumulator (请求内容日志
// target or 对话归档 on).
func contentCaptureActive(c *gin.Context) bool {
	return contentCaptureRequestFromContext(c) != nil || archiveRequestFromContext(c) != nil
}

// ObserveContentFinalResponse feeds a buffered terminal Responses object (the OpenAI-compat
// buffered chat/completions and /v1/messages paths read the whole upstream stream first and never
// pass its frames to the observer) to the content capture. No-op unless the request is captured.
func (o *upstreamResponseModelObserver) ObserveContentFinalResponse(final any) {
	if o == nil || o.content == nil || final == nil {
		return
	}
	defer func() { _ = recover() }()
	body, err := json.Marshal(final)
	if err != nil {
		return
	}
	o.content.feedOpenAI(body, "")
}

// beginContentCaptureOnly starts a fresh observation for forward paths that don't track the
// upstream response model (Anthropic upstream behind /v1/chat/completions and /v1/responses), and
// only when the request is captured, so model observation and billing there stay as they were.
func beginContentCaptureOnly(c *gin.Context) {
	if contentCaptureActive(c) {
		beginUpstreamResponseModelObservation(c)
	}
}

// contentAccumulatorFromContext returns the current attempt's accumulator (nil when not captured;
// feeding nil is a no-op).
func contentAccumulatorFromContext(c *gin.Context) *contentAccumulator {
	if observer := upstreamResponseModelObserverFromContext(c); observer != nil {
		return observer.content
	}
	return nil
}

// contentAccumulator collects the model's reply from raw upstream payloads (SSE data frames or a
// whole non-stream body), within fixed memory caps. One per upstream attempt.
type contentAccumulator struct {
	mu sync.Mutex

	text         []byte
	textFull     bool
	textByDeltas bool

	tools      []*contentToolAcc
	toolsByKey map[int64]*contentToolAcc
	toolsCut   bool // a tool call was not kept because of the count cap
	sawItems   bool // Responses output items were seen (streamed); the terminal event is then redundant

	// Caps: the content-log sizes, or the larger archive sizes when arch is set.
	textMax  int
	argsMax  int
	maxTools int
	// arch keeps what only 对话归档 needs (raw output items, images, status); nil otherwise.
	arch *archiveResponseAcc
}

type contentToolAcc struct {
	name   string
	callID string
	args   []byte
	max    int
	full   bool
}

func newContentAccumulator() *contentAccumulator {
	return &contentAccumulator{
		textMax:  contentCaptureReplyMaxBytes,
		argsMax:  contentCaptureToolArgsMaxBytes,
		maxTools: ContentLogMaxToolCalls,
	}
}

// newArchiveContentAccumulator keeps the full reply within the archive caps; the content log
// still trims its copy to its own sizes in snapshot().
func newArchiveContentAccumulator() *contentAccumulator {
	return &contentAccumulator{
		textMax:  archiveResponseTextMaxBytes,
		argsMax:  archiveToolArgsMaxBytes,
		maxTools: archiveMaxToolCalls,
		arch:     &archiveResponseAcc{},
	}
}

func (a *contentAccumulator) appendText(s string) {
	if a.textFull || s == "" {
		return
	}
	room := a.textMax - len(a.text)
	if len(s) >= room {
		s = s[:room]
		a.textFull = true
	}
	a.text = append(a.text, s...)
}

func (t *contentToolAcc) appendArgs(s string) {
	if t == nil || t.full || s == "" {
		return
	}
	room := t.max - len(t.args)
	if len(s) >= room {
		s = s[:room]
		t.full = true
	}
	t.args = append(t.args, s...)
}

// addTool starts a new tool call; nil when the cap is reached.
func (a *contentAccumulator) addTool(name string) *contentToolAcc {
	if len(a.tools) >= a.maxTools {
		a.toolsCut = true
		return nil
	}
	t := &contentToolAcc{name: truncateBytesUTF8(strings.TrimSpace(name), contentCaptureToolNameMaxBytes), max: a.argsMax}
	a.tools = append(a.tools, t)
	return t
}

func (a *contentAccumulator) toolForKey(key int64, name string) *contentToolAcc {
	if a.toolsByKey == nil {
		a.toolsByKey = map[int64]*contentToolAcc{}
	}
	if t, ok := a.toolsByKey[key]; ok {
		if t != nil && t.name == "" && name != "" {
			t.name = truncateBytesUTF8(strings.TrimSpace(name), contentCaptureToolNameMaxBytes)
		}
		return t
	}
	t := a.addTool(name)
	a.toolsByKey[key] = t // nil once the cap is reached: later deltas for this key are ignored
	return t
}

func (a *contentAccumulator) snapshot() ([]ContentToolCall, string) {
	if a == nil {
		return nil, ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Always the content-log sizes, whatever the accumulator kept for the archive.
	calls := make([]ContentToolCall, 0, min(len(a.tools), ContentLogMaxToolCalls))
	for _, t := range a.tools {
		if t == nil {
			continue
		}
		if len(calls) >= ContentLogMaxToolCalls {
			break
		}
		args := t.args
		if len(args) > contentCaptureToolArgsMaxBytes {
			args = args[:contentCaptureToolArgsMaxBytes]
		}
		calls = append(calls, ContentToolCall{Name: t.name, Arguments: strings.ToValidUTF8(string(args), "")})
	}
	return calls, strings.ToValidUTF8(truncateBytesUTF8(string(a.text), contentCaptureReplyMaxBytes), "")
}

// feedOpenAI handles OpenAI Responses events (typed), chat completion chunks and non-stream bodies
// of either API.
func (a *contentAccumulator) feedOpenAI(payload []byte, eventType string) {
	if a == nil || len(payload) == 0 {
		return
	}
	defer func() { _ = recover() }()
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	typ := strings.TrimSpace(eventType)
	if typ == "" || !strings.HasPrefix(typ, "response.") {
		if t := gjson.GetBytes(trimmed, "type").String(); strings.HasPrefix(t, "response.") {
			typ = t
		}
	}
	if strings.HasPrefix(typ, "response.") {
		a.feedResponsesEvent(trimmed, typ)
		return
	}
	if choices := gjson.GetBytes(trimmed, "choices"); choices.IsArray() {
		a.feedChatChoices(choices)
		return
	}
	if output := gjson.GetBytes(trimmed, "output"); output.IsArray() {
		// Non-stream Responses body.
		if len(a.tools) == 0 && len(a.text) == 0 {
			a.arch.setStatus(archiveStatusForResponsesEvent("", gjson.GetBytes(trimmed, "status").String()))
			a.feedResponsesOutput(output)
		}
	}
}

func (a *contentAccumulator) feedResponsesEvent(payload []byte, typ string) {
	switch typ {
	case "response.output_text.delta":
		a.textByDeltas = true
		a.appendText(gjson.GetBytes(payload, "delta").String())
	case "response.output_item.done":
		a.sawItems = true
		a.feedResponsesItem(gjson.GetBytes(payload, "item"))
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		a.arch.setStatus(archiveStatusForResponsesEvent(typ, gjson.GetBytes(payload, "response.status").String()))
		// Passthrough streams may end with an empty output; only use it when nothing streamed.
		if a.sawItems || a.textByDeltas || len(a.tools) > 0 {
			return
		}
		a.feedResponsesOutput(gjson.GetBytes(payload, "response.output"))
	}
}

func (a *contentAccumulator) feedResponsesOutput(output gjson.Result) {
	if !output.IsArray() {
		return
	}
	forEachArray(output, func(item gjson.Result) bool {
		a.feedResponsesItem(item)
		return true
	})
}

func (a *contentAccumulator) feedResponsesItem(item gjson.Result) {
	if !item.IsObject() {
		return
	}
	a.arch.addResponsesItem(item)
	itemType := item.Get("type").String()
	switch itemType {
	case "message":
		if a.textByDeltas {
			return
		}
		forEachArray(item.Get("content"), func(part gjson.Result) bool {
			if pt := part.Get("type").String(); pt == "output_text" || pt == "text" {
				a.appendText(part.Get("text").String())
			}
			return !a.textFull
		})
		return
	case "reasoning", "":
		return
	case "local_shell_call":
		if t := a.addTool("local_shell"); t != nil {
			t.appendArgs(item.Get("action.command").Raw)
		}
		return
	case "web_search_call":
		if t := a.addTool("web_search"); t != nil {
			t.appendArgs(item.Get("action.query").String())
		}
		return
	case "image_generation_call":
		// Never keep the image itself.
		a.addTool("image_generation")
		return
	}
	name := item.Get("name").String()
	if name == "" {
		if !strings.HasSuffix(itemType, "_call") {
			return
		}
		name = strings.TrimSuffix(itemType, "_call")
	}
	t := a.addTool(name)
	if t == nil {
		return
	}
	t.callID = item.Get("call_id").String()
	switch {
	case item.Get("arguments").Exists():
		t.appendArgs(item.Get("arguments").String())
	case item.Get("input").Exists():
		t.appendArgs(item.Get("input").String())
	case item.Get("code").Exists():
		t.appendArgs(item.Get("code").String())
	case item.Get("queries").Exists():
		t.appendArgs(item.Get("queries").Raw)
	case item.Get("action").Exists():
		t.appendArgs(item.Get("action").Raw)
	}
}

func (a *contentAccumulator) feedChatChoices(choices gjson.Result) {
	forEachArray(choices, func(choice gjson.Result) bool {
		choiceIndex := choice.Get("index").Int()
		if reason := choice.Get("finish_reason").String(); reason != "" {
			a.arch.setStatus(archiveStatusForFinishReason(reason))
		}
		if delta := choice.Get("delta"); delta.Exists() {
			if content := delta.Get("content"); content.Type == gjson.String {
				a.appendText(content.String())
			}
			forEachArray(delta.Get("tool_calls"), func(call gjson.Result) bool {
				key := choiceIndex<<20 | call.Get("index").Int()
				if t := a.toolForKey(key, call.Get("function.name").String()); t != nil {
					if id := call.Get("id").String(); id != "" && t.callID == "" {
						t.callID = id
					}
					t.appendArgs(call.Get("function.arguments").String())
				}
				return true
			})
			return true
		}
		message := choice.Get("message")
		if !message.Exists() {
			return true
		}
		if content := message.Get("content"); content.Type == gjson.String {
			a.appendText(content.String())
		} else if content.IsArray() {
			forEachArray(content, func(part gjson.Result) bool {
				a.appendText(part.Get("text").String())
				return !a.textFull
			})
		}
		forEachArray(message.Get("tool_calls"), func(call gjson.Result) bool {
			if t := a.addTool(call.Get("function.name").String()); t != nil {
				t.callID = call.Get("id").String()
				t.appendArgs(call.Get("function.arguments").String())
			}
			return true
		})
		return true
	})
}

// feedAnthropic handles Anthropic Messages SSE events and non-stream bodies.
func (a *contentAccumulator) feedAnthropic(payload []byte) {
	if a == nil || len(payload) == 0 {
		return
	}
	defer func() { _ = recover() }()
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	switch gjson.GetBytes(trimmed, "type").String() {
	case "content_block_start":
		block := gjson.GetBytes(trimmed, "content_block")
		index := gjson.GetBytes(trimmed, "index").Int()
		switch block.Get("type").String() {
		case "text":
			a.appendText(block.Get("text").String())
		case "thinking", "redacted_thinking":
		default:
			if name := block.Get("name").String(); name != "" {
				t := a.toolForKey(index, name)
				if t != nil && t.callID == "" {
					t.callID = block.Get("id").String()
				}
				if input := block.Get("input"); t != nil && input.IsObject() && len(input.Map()) > 0 {
					t.appendArgs(input.Raw)
				}
			}
		}
	case "content_block_delta":
		delta := gjson.GetBytes(trimmed, "delta")
		switch delta.Get("type").String() {
		case "text_delta":
			a.appendText(delta.Get("text").String())
		case "input_json_delta":
			if t, ok := a.toolsByKey[gjson.GetBytes(trimmed, "index").Int()]; ok && t != nil {
				t.appendArgs(delta.Get("partial_json").String())
			}
		}
	case "message_delta":
		if reason := gjson.GetBytes(trimmed, "delta.stop_reason").String(); reason != "" {
			a.arch.setStatus(archiveStatusForFinishReason(reason))
		}
	case "message":
		// Non-stream body.
		if len(a.tools) > 0 || len(a.text) > 0 {
			return
		}
		if reason := gjson.GetBytes(trimmed, "stop_reason").String(); reason != "" {
			a.arch.setStatus(archiveStatusForFinishReason(reason))
		}
		forEachArray(gjson.GetBytes(trimmed, "content"), func(block gjson.Result) bool {
			a.arch.addAnthropicBlock(block)
			switch block.Get("type").String() {
			case "text":
				a.appendText(block.Get("text").String())
			case "thinking", "redacted_thinking":
			default:
				if name := block.Get("name").String(); name != "" {
					if t := a.addTool(name); t != nil {
						t.callID = block.Get("id").String()
						t.appendArgs(block.Get("input").Raw)
					}
				}
			}
			return true
		})
	case "error":
		a.arch.setStatus("error")
	}
}

// forEachArray iterates a JSON array; anything else is ignored (gjson's ForEach would visit a
// scalar once).
func forEachArray(value gjson.Result, fn func(gjson.Result) bool) {
	if !value.IsArray() {
		return
	}
	value.ForEach(func(_, item gjson.Result) bool { return fn(item) })
}

// truncateBytesUTF8 cuts s to at most max bytes without splitting a UTF-8 sequence.
func truncateBytesUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
