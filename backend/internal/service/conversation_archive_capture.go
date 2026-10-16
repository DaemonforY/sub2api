package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// 对话归档 capture on the request path. Everything here is a no-op costing one atomic load while
// archiving is off; when it is on, the handlers stash a reference to the request body they
// already hold, the upstream response observer keeps the full reply (archive mode of the content
// accumulator) and the images endpoints hand over their response payloads. The snapshot rides on
// the existing ContentCapture to usage recording, where the archive job is queued.

const (
	archiveRequestContextKey = "conversation_archive_request"

	archiveResponseTextMaxBytes = 256 << 10
	archiveToolArgsMaxBytes     = 64 << 10
	archiveMaxToolCalls         = 200
	archiveResponseItemsMax     = 1 << 20
	archiveImagesMaxBytes       = 32 << 20 // base64 bytes per request

	ArchiveProtocolResponses = "openai_responses"
	ArchiveProtocolChat      = "openai_chat"
	ArchiveProtocolAnthropic = "anthropic_messages"
	ArchiveProtocolGemini    = "gemini"
	ArchiveProtocolImages    = "openai_images"

	archiveImageSourceResponse        = "response"
	archiveImageSourceImageGeneration = "image_generation_call"
)

// archiveCaptureOn mirrors ConversationArchiveService.Enabled() for the request path.
var archiveCaptureOn atomic.Bool

// ArchiveToolCall is one tool call of an archived response.
type ArchiveToolCall struct {
	Name      string `json:"name"`
	CallID    string `json:"call_id,omitempty"`
	Arguments string `json:"arguments"`
}

// ArchiveResponse is the model's response as archived.
type ArchiveResponse struct {
	Status    string            `json:"status"`
	Text      string            `json:"text"`
	ToolCalls []ArchiveToolCall `json:"tool_calls"`
	Items     []json.RawMessage `json:"items"`
}

// ArchiveImagePayload is one generated image still in base64 (decoded by the archive worker).
type ArchiveImagePayload struct {
	B64    string
	Format string // output_format when the upstream declared one (png / jpeg / webp)
	Source string
}

// ArchiveCapture is what one request hands to the archive: a reference to the request body as
// the handler read it (never copied), and the response collected during the forward.
type ArchiveCapture struct {
	Protocol    string
	RequestBody []byte
	// Prompt is set instead of RequestBody for multipart image edits (input images aren't kept).
	Prompt string

	Response          ArchiveResponse
	Images            []ArchiveImagePayload
	ImageURLs         []string
	ResponseTruncated bool
}

// SizeBytes is what the capture holds in memory (for the queue's byte budget).
func (a *ArchiveCapture) SizeBytes() int64 {
	if a == nil {
		return 0
	}
	n := len(a.RequestBody) + len(a.Prompt) + len(a.Response.Text)
	for _, t := range a.Response.ToolCalls {
		n += len(t.Name) + len(t.CallID) + len(t.Arguments)
	}
	for _, item := range a.Response.Items {
		n += len(item)
	}
	for _, img := range a.Images {
		n += len(img.B64)
	}
	for _, u := range a.ImageURLs {
		n += len(u)
	}
	return int64(n)
}

type archiveRequest struct {
	protocol string
	body     []byte
	prompt   string

	// Images endpoint payloads (per request, not per attempt: those endpoints only produce output
	// on the successful attempt).
	mu         sync.Mutex
	images     []ArchiveImagePayload
	imageBytes int
	imagesCut  bool
	urls       []string
}

// ArchiveCaptureEnabled reports whether 对话归档 is on (one atomic load).
func ArchiveCaptureEnabled() bool {
	return archiveCaptureOn.Load()
}

// SetArchiveRequestBody marks the request for archiving and keeps a reference to the body the
// handler read (not a copy; the archive only reads it, after the response). No-op while archiving
// is off or for endpoints that aren't archived.
func SetArchiveRequestBody(c *gin.Context, body []byte) {
	if !archiveCaptureOn.Load() || c == nil || c.Request == nil || len(body) == 0 {
		return
	}
	defer func() { _ = recover() }()
	protocol := ArchiveProtocolForPath(c.Request.URL.Path)
	if protocol == "" {
		return
	}
	c.Set(archiveRequestContextKey, &archiveRequest{protocol: protocol, body: body})
}

// SetArchiveRequestPrompt marks a multipart image edit for archiving with only its prompt.
func SetArchiveRequestPrompt(c *gin.Context, prompt string) {
	if !archiveCaptureOn.Load() || c == nil {
		return
	}
	c.Set(archiveRequestContextKey, &archiveRequest{protocol: ArchiveProtocolImages, prompt: truncateBytesUTF8(prompt, 64<<10)})
}

// archiveRequestFromContext returns the request's archive marker; while archiving is off this is
// the single atomic load the request path pays (a request marked before a pause is dropped too).
func archiveRequestFromContext(c *gin.Context) *archiveRequest {
	if c == nil || !archiveCaptureOn.Load() {
		return nil
	}
	value, ok := c.Get(archiveRequestContextKey)
	if !ok {
		return nil
	}
	req, _ := value.(*archiveRequest)
	return req
}

// ArchiveProtocolForPath maps an inbound path to the archived protocol ("" = not archived).
func ArchiveProtocolForPath(path string) string {
	p := strings.ToLower(strings.TrimRight(path, "/"))
	switch {
	case strings.HasSuffix(p, "/count_tokens"), strings.HasSuffix(p, "/async"), strings.HasSuffix(p, "/input_tokens"):
		return ""
	case strings.HasSuffix(p, "/images/generations"), strings.HasSuffix(p, "/images/edits"):
		return ArchiveProtocolImages
	case strings.HasSuffix(p, "/chat/completions"):
		return ArchiveProtocolChat
	case strings.HasSuffix(p, "/responses"), strings.HasSuffix(p, "/responses/compact"):
		return ArchiveProtocolResponses
	case strings.HasSuffix(p, "/messages"):
		return ArchiveProtocolAnthropic
	case strings.Contains(p, "/v1beta/") && (strings.HasSuffix(p, ":generatecontent") || strings.HasSuffix(p, ":streamgeneratecontent")):
		return ArchiveProtocolGemini
	default:
		return ""
	}
}

// ObserveArchiveImagesPayload collects generated images from an images-endpoint response body or
// SSE data payload (data[].b64_json / data[].url, or a top-level b64_json on *.completed stream
// events). Partial images are ignored. No-op unless the request is archived.
func ObserveArchiveImagesPayload(c *gin.Context, payload []byte) {
	req := archiveRequestFromContext(c)
	if req == nil || len(payload) == 0 {
		return
	}
	defer func() { _ = recover() }()
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return
	}
	typ := gjson.GetBytes(trimmed, "type").String()
	if strings.Contains(typ, "partial_image") {
		return
	}
	req.mu.Lock()
	defer req.mu.Unlock()
	forEachArray(gjson.GetBytes(trimmed, "data"), func(item gjson.Result) bool {
		req.addImageLocked(item.Get("b64_json").String(), item.Get("output_format").String(), item.Get("url").String())
		return true
	})
	if b64 := gjson.GetBytes(trimmed, "b64_json"); b64.Type == gjson.String {
		req.addImageLocked(b64.String(), gjson.GetBytes(trimmed, "output_format").String(), "")
	}
}

func (r *archiveRequest) addImageLocked(b64, format, url string) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		if url = strings.TrimSpace(url); url != "" && len(r.urls) < 64 {
			r.urls = append(r.urls, truncateBytesUTF8(url, 4096))
		}
		return
	}
	if r.imageBytes+len(b64) > archiveImagesMaxBytes {
		r.imagesCut = true
		return
	}
	r.imageBytes += len(b64)
	r.images = append(r.images, ArchiveImagePayload{B64: b64, Format: format, Source: archiveImageSourceResponse})
}

// snapshot builds the archive capture from the request marker and the final attempt's
// accumulator (nil when the forward path had none). It takes the accumulator's buffers over: the
// attempt is finished by the time the handler snapshots.
func (r *archiveRequest) snapshot(acc *contentAccumulator) *ArchiveCapture {
	out := &ArchiveCapture{Protocol: r.protocol, RequestBody: r.body, Prompt: r.prompt}
	out.Response.ToolCalls = []ArchiveToolCall{}
	out.Response.Items = []json.RawMessage{}
	budget := archiveImagesMaxBytes
	if acc != nil {
		acc.mu.Lock()
		out.Response.Text = strings.ToValidUTF8(string(acc.text), "")
		for _, t := range acc.tools {
			if t == nil {
				continue
			}
			out.Response.ToolCalls = append(out.Response.ToolCalls, ArchiveToolCall{Name: t.name, CallID: t.callID, Arguments: strings.ToValidUTF8(string(t.args), "")})
			if t.full {
				out.ResponseTruncated = true
			}
		}
		if acc.textFull || acc.toolsCut {
			out.ResponseTruncated = true
		}
		if arch := acc.arch; arch != nil {
			out.Response.Status = arch.status
			out.Response.Items = append(out.Response.Items, arch.items...)
			if arch.itemsCut || arch.imagesCut {
				out.ResponseTruncated = true
			}
			for _, img := range arch.images {
				budget -= len(img.B64)
				out.Images = append(out.Images, img)
			}
		}
		acc.mu.Unlock()
	}
	r.mu.Lock()
	for _, img := range r.images {
		if len(img.B64) > budget {
			out.ResponseTruncated = true
			continue
		}
		budget -= len(img.B64)
		out.Images = append(out.Images, img)
	}
	out.ImageURLs = append(out.ImageURLs, r.urls...)
	if r.imagesCut {
		out.ResponseTruncated = true
	}
	r.mu.Unlock()
	if out.Response.Status == "" {
		switch r.protocol {
		case ArchiveProtocolImages:
			// The images endpoints have no terminal event; usage is only recorded for a delivered result.
			out.Response.Status = "completed"
		case ArchiveProtocolGemini:
			// Gemini replies are not captured.
			out.Response.Status = "unknown"
		default:
			// A stream that ended without a terminal event / finish reason.
			out.Response.Status = "incomplete"
		}
	}
	return out
}

// archiveResponseAcc is the archive-only part of a content accumulator (guarded by its mutex).
type archiveResponseAcc struct {
	status     string
	items      []json.RawMessage
	itemsBytes int
	itemsCut   bool
	images     []ArchiveImagePayload
	imageBytes int
	imagesCut  bool
}

func (a *archiveResponseAcc) setStatus(status string) {
	if a == nil || status == "" {
		return
	}
	a.status = status
}

func archiveStatusForResponsesEvent(eventType, status string) string {
	switch strings.TrimSpace(status) {
	case "completed", "incomplete", "failed":
		return status
	case "cancelled", "canceled":
		return "incomplete"
	}
	switch eventType {
	case "response.completed", "response.done":
		return "completed"
	case "response.incomplete":
		return "incomplete"
	case "response.failed":
		return "failed"
	}
	return ""
}

func archiveStatusForFinishReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "length", "max_tokens", "model_context_window_exceeded":
		return "incomplete"
	case "":
		return ""
	default:
		return "completed"
	}
}

// addResponsesItem keeps one raw Responses output item: reasoning encrypted_content is dropped,
// an image_generation_call's base64 result moves to the image list (the item keeps a pointer).
func (a *archiveResponseAcc) addResponsesItem(item gjson.Result) {
	if a == nil || !item.IsObject() {
		return
	}
	raw := []byte(item.Raw)
	switch item.Get("type").String() {
	case "reasoning":
		if item.Get("encrypted_content").Exists() {
			if stripped, err := sjson.DeleteBytes(raw, "encrypted_content"); err == nil {
				raw = stripped
			}
		}
	case "image_generation_call":
		if result := item.Get("result"); result.Type == gjson.String && result.String() != "" {
			b64 := result.String()
			if a.imageBytes+len(b64) > archiveImagesMaxBytes {
				a.imagesCut = true
			} else {
				a.imageBytes += len(b64)
				a.images = append(a.images, ArchiveImagePayload{B64: b64, Format: item.Get("output_format").String(), Source: archiveImageSourceImageGeneration})
				raw, _ = sjson.SetBytes(raw, "archived_image", len(a.images)-1)
			}
			raw, _ = sjson.SetBytes(raw, "result", "")
		}
	}
	a.addRawItem(raw)
}

// addAnthropicBlock keeps one content block of a non-stream Anthropic message (thinking
// signatures and redacted thinking data dropped).
func (a *archiveResponseAcc) addAnthropicBlock(block gjson.Result) {
	if a == nil || !block.IsObject() {
		return
	}
	raw := []byte(block.Raw)
	switch block.Get("type").String() {
	case "thinking":
		raw, _ = sjson.DeleteBytes(raw, "signature")
	case "redacted_thinking":
		raw, _ = sjson.DeleteBytes(raw, "data")
	}
	a.addRawItem(raw)
}

func (a *archiveResponseAcc) addRawItem(raw []byte) {
	if len(raw) == 0 {
		return
	}
	if a.itemsBytes+len(raw) > archiveResponseItemsMax {
		a.itemsCut = true
		return
	}
	a.itemsBytes += len(raw)
	a.items = append(a.items, json.RawMessage(raw))
}
