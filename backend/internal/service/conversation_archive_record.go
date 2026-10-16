package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

// 对话归档 record building: protocol-aware extraction of the input items and instructions,
// session dedupe against the hashes kept in Redis, and the gzip JSON object written per request.

const (
	archiveRecordVersion     = 1
	archiveDedupeTTL         = 48 * time.Hour
	archiveDedupeMaxItems    = 4000
	archiveDedupeKeyPrefix   = "conv_archive:sess:"
	archiveSeenKeyPrefix     = "conv_archive:seen:"
	archiveRequestItemsMax   = 16 << 20 // raw bytes of new_items per record
	archivePreviewMaxRunes   = 200
	archiveParamMaxBytes     = 2 << 10
	archiveMaxToolNames      = 128
	archiveRequestIDMaxChars = 80
)

// archiveRequestParts is what the archive keeps from a request body.
type archiveRequestParts struct {
	Instructions       string
	HasInstructions    bool
	Items              []json.RawMessage
	Params             map[string]json.RawMessage
	PreviousResponseID string
}

// extractArchiveRequest splits a request body into instructions, input items (raw, as sent) and
// small request parameters. prompt is used for multipart image edits (no body).
func extractArchiveRequest(protocol string, body []byte, prompt string) (out archiveRequestParts) {
	out.Items = []json.RawMessage{}
	out.Params = map[string]json.RawMessage{}
	defer func() {
		if recover() != nil {
			out = archiveRequestParts{Items: []json.RawMessage{}, Params: map[string]json.RawMessage{}}
		}
	}()
	if protocol == ArchiveProtocolImages && len(body) == 0 {
		if strings.TrimSpace(prompt) != "" {
			out.Items = append(out.Items, archiveUserTextItem(prompt))
		}
		return out
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return out
	}
	root := gjson.ParseBytes(body)
	switch protocol {
	case ArchiveProtocolResponses:
		input := root.Get("input")
		switch {
		case input.Type == gjson.String:
			out.Items = append(out.Items, archiveUserTextItem(input.String()))
		case input.IsArray():
			forEachArray(input, func(item gjson.Result) bool {
				out.Items = append(out.Items, json.RawMessage(item.Raw))
				return true
			})
		case input.IsObject():
			out.Items = append(out.Items, json.RawMessage(input.Raw))
		}
		if instr := root.Get("instructions"); instr.Exists() && instr.Type != gjson.Null {
			out.HasInstructions = true
			if instr.Type == gjson.String {
				out.Instructions = instr.String()
			} else {
				out.Instructions = instr.Raw
			}
		}
		out.PreviousResponseID = strings.TrimSpace(root.Get("previous_response_id").String())
		archivePickParams(root, out.Params, "temperature", "top_p", "max_output_tokens", "tool_choice", "reasoning",
			"previous_response_id", "parallel_tool_calls", "service_tier", "store", "stream", "truncation", "text")
		out.Params["tools"] = archiveToolNames(root.Get("tools"), "name", "type")
	case ArchiveProtocolChat:
		var system []string
		forEachArray(root.Get("messages"), func(msg gjson.Result) bool {
			switch msg.Get("role").String() {
			case "system", "developer":
				system = append(system, archiveContentText(msg.Get("content")))
			default:
				out.Items = append(out.Items, json.RawMessage(msg.Raw))
			}
			return true
		})
		if len(system) > 0 {
			out.HasInstructions = true
			out.Instructions = strings.Join(system, "\n\n")
		}
		archivePickParams(root, out.Params, "temperature", "top_p", "max_tokens", "max_completion_tokens", "tool_choice",
			"reasoning_effort", "stream", "n", "service_tier")
		if rf := root.Get("response_format.type"); rf.Exists() {
			out.Params["response_format"], _ = json.Marshal(rf.String())
		}
		out.Params["tools"] = archiveToolNames(root.Get("tools"), "function.name", "type")
	case ArchiveProtocolAnthropic:
		forEachArray(root.Get("messages"), func(msg gjson.Result) bool {
			out.Items = append(out.Items, json.RawMessage(msg.Raw))
			return true
		})
		if system := root.Get("system"); system.Exists() && system.Type != gjson.Null {
			out.HasInstructions = true
			out.Instructions = archiveContentText(system)
		}
		archivePickParams(root, out.Params, "temperature", "top_p", "top_k", "max_tokens", "tool_choice", "thinking",
			"stream", "stop_sequences", "service_tier")
		out.Params["tools"] = archiveToolNames(root.Get("tools"), "name", "type")
	case ArchiveProtocolGemini:
		forEachArray(root.Get("contents"), func(content gjson.Result) bool {
			out.Items = append(out.Items, json.RawMessage(content.Raw))
			return true
		})
		system := root.Get("systemInstruction")
		if !system.Exists() {
			system = root.Get("system_instruction")
		}
		if system.Exists() && system.Type != gjson.Null {
			out.HasInstructions = true
			out.Instructions = archiveGeminiPartsText(system.Get("parts"))
		}
		if cfg := root.Get("generationConfig"); cfg.IsObject() {
			archivePickParams(cfg, out.Params, "temperature", "topP", "topK", "maxOutputTokens", "thinkingConfig", "responseMimeType")
		}
		var names []string
		forEachArray(root.Get("tools"), func(tool gjson.Result) bool {
			forEachArray(tool.Get("functionDeclarations"), func(fn gjson.Result) bool {
				if len(names) < archiveMaxToolNames {
					names = append(names, fn.Get("name").String())
				}
				return true
			})
			tool.ForEach(func(key, _ gjson.Result) bool {
				if k := key.String(); k != "functionDeclarations" && len(names) < archiveMaxToolNames {
					names = append(names, k)
				}
				return true
			})
			return true
		})
		if names != nil {
			out.Params["tools"], _ = json.Marshal(names)
		}
	case ArchiveProtocolImages:
		text := root.Get("prompt").String()
		if text == "" {
			text = prompt
		}
		if strings.TrimSpace(text) != "" {
			out.Items = append(out.Items, archiveUserTextItem(text))
		}
		archivePickParams(root, out.Params, "model", "n", "size", "quality", "background", "output_format",
			"response_format", "style", "moderation", "output_compression", "stream")
	}
	if raw, ok := out.Params["tools"]; ok && (len(raw) == 0 || string(raw) == "null") {
		delete(out.Params, "tools")
	}
	return out
}

func archiveUserTextItem(text string) json.RawMessage {
	raw, _ := json.Marshal(map[string]string{"role": "user", "content": text})
	return raw
}

func archivePickParams(root gjson.Result, dst map[string]json.RawMessage, keys ...string) {
	for _, key := range keys {
		value := root.Get(key)
		if !value.Exists() || len(value.Raw) > archiveParamMaxBytes {
			continue
		}
		dst[key] = json.RawMessage(value.Raw)
	}
}

// archiveToolNames lists tool names (no schemas): the first non-empty of the given paths per tool.
func archiveToolNames(tools gjson.Result, paths ...string) json.RawMessage {
	if !tools.IsArray() {
		return nil
	}
	names := []string{}
	forEachArray(tools, func(tool gjson.Result) bool {
		if len(names) >= archiveMaxToolNames {
			return false
		}
		for _, p := range paths {
			if name := tool.Get(p).String(); name != "" {
				names = append(names, truncateBytesUTF8(name, 200))
				break
			}
		}
		return true
	})
	raw, _ := json.Marshal(names)
	return raw
}

// archiveContentText joins the text of a string or an array of text parts/blocks.
func archiveContentText(content gjson.Result) string {
	if content.Type == gjson.String {
		return content.String()
	}
	if !content.IsArray() {
		return ""
	}
	var parts []string
	forEachArray(content, func(part gjson.Result) bool {
		if part.Type == gjson.String {
			parts = append(parts, part.String())
			return true
		}
		if text := part.Get("text"); text.Type == gjson.String {
			parts = append(parts, text.String())
		}
		return true
	})
	return strings.Join(parts, "\n\n")
}

func archiveGeminiPartsText(parts gjson.Result) string {
	var out []string
	forEachArray(parts, func(part gjson.Result) bool {
		if text := part.Get("text"); text.Type == gjson.String {
			out = append(out, text.String())
		}
		return true
	})
	return strings.Join(out, "\n\n")
}

// archiveVolatileKeys are per-turn markers clients move between resends of the same item:
// Claude Code puts the prompt-cache breakpoint (cache_control) on the newest message each turn.
var archiveVolatileKeys = map[string]bool{"cache_control": true}

// archiveCanonicalJSON is the item's JSON without whitespace (like gjson @ugly, key order kept)
// and without volatile keys at any depth, so a resent item hashes the same.
func archiveCanonicalJSON(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if !gjson.ValidBytes(trimmed) {
		return string(trimmed)
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	writeArchiveCanonical(&b, gjson.ParseBytes(trimmed))
	return b.String()
}

func writeArchiveCanonical(b *strings.Builder, v gjson.Result) {
	switch {
	case v.IsObject():
		_ = b.WriteByte('{')
		first := true
		v.ForEach(func(key, value gjson.Result) bool {
			if archiveVolatileKeys[key.String()] {
				return true
			}
			if !first {
				_ = b.WriteByte(',')
			}
			first = false
			_, _ = b.WriteString(key.Raw)
			_ = b.WriteByte(':')
			writeArchiveCanonical(b, value)
			return true
		})
		_ = b.WriteByte('}')
	case v.IsArray():
		_ = b.WriteByte('[')
		first := true
		v.ForEach(func(_, value gjson.Result) bool {
			if !first {
				_ = b.WriteByte(',')
			}
			first = false
			writeArchiveCanonical(b, value)
			return true
		})
		_ = b.WriteByte(']')
	default:
		_, _ = b.WriteString(strings.TrimSpace(v.Raw))
	}
}

// archiveItemHash hashes an item's canonical JSON, so whitespace differences and moved
// cache breakpoints between resends don't break the prefix match.
func archiveItemHash(raw json.RawMessage) string {
	sum := sha256.Sum256([]byte(archiveCanonicalJSON(raw)))
	return hex.EncodeToString(sum[:12])
}

func archiveSHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// archiveSessionKey is usage_logs.session_id when the request had one, otherwise
// "h:" + sha256(api_key_id + first input item)[:24] so clients without a session id still share
// their first message.
func archiveSessionKey(sessionID string, apiKeyID int64, items []json.RawMessage, requestID string) string {
	if s := strings.TrimSpace(sessionID); s != "" {
		return truncateBytesUTF8(s, 256)
	}
	seed := ""
	if len(items) > 0 {
		seed = archiveCanonicalJSON(items[0])
	} else {
		seed = "req:" + requestID
	}
	return "h:" + archiveSHA256Hex(fmt.Sprintf("%d:%s", apiKeyID, seed))[:24]
}

// archiveDedupeKey keys the dedupe state by session AND thread: Claude Code sub-agents and side
// calls share the session id but each has its own first message, so they dedupe independently.
func archiveDedupeKey(userID int64, sessionKey, thread string) string {
	return fmt.Sprintf("%s%d:%s", archiveDedupeKeyPrefix, userID, archiveSHA256Hex(sessionKey + "\x00" + thread)[:24])
}

// archiveSeenKey marks that a session already has an archived thread: a new thread in it starts
// a fresh context (reset), not the session's first turn.
func archiveSeenKey(userID int64, sessionKey string) string {
	return fmt.Sprintf("%s%d:%s", archiveSeenKeyPrefix, userID, archiveSHA256Hex(sessionKey)[:24])
}

// ArchiveDedupeState is the per-thread state kept in Redis
// (conv_archive:sess:{user_id}:{sha256(session_key + thread)[:24]}). Offset counts the leading
// items whose hashes were dropped to stay under 4000; ItemHashes start at item Offset.
type ArchiveDedupeState struct {
	InstrHash  string   `json:"instr_hash"`
	ItemHashes []string `json:"item_hashes"`
	Offset     int      `json:"offset,omitempty"`
}

// ArchiveDedupeStore keeps the per-session dedupe state (Redis; implemented in repository).
type ArchiveDedupeStore interface {
	Load(ctx context.Context, key string) (*ArchiveDedupeState, error) // nil, nil when absent
	Save(ctx context.Context, key string, state *ArchiveDedupeState, ttl time.Duration) error
}

type archiveDedupeResult struct {
	NewItems            []json.RawMessage
	PrefixItems         int
	Reset               bool
	IncludeInstructions bool
	Next                *ArchiveDedupeState
}

// computeArchiveDedupe applies the session rules: new items are those after the stored prefix;
// a history that no longer matches is a reset (everything is archived again); with Redis down
// everything is archived and marked reset; with previous_response_id the input already holds only
// the new items.
func computeArchiveDedupe(prev *ArchiveDedupeState, loadErr error, parts archiveRequestParts, hashes []string, instrHash string) archiveDedupeResult {
	res := archiveDedupeResult{NewItems: parts.Items}
	next := &ArchiveDedupeState{InstrHash: instrHash, ItemHashes: hashes}
	if len(next.ItemHashes) > archiveDedupeMaxItems {
		next.Offset = len(next.ItemHashes) - archiveDedupeMaxItems
		next.ItemHashes = next.ItemHashes[next.Offset:]
	}
	res.Next = next
	instrChanged := prev == nil || prev.InstrHash != instrHash
	switch {
	case loadErr != nil:
		res.Reset = true
		res.IncludeInstructions = parts.HasInstructions
		return res
	case parts.PreviousResponseID != "":
		if prev != nil {
			next.ItemHashes, next.Offset = prev.ItemHashes, prev.Offset
		} else {
			next.ItemHashes, next.Offset = nil, 0
		}
		res.IncludeInstructions = parts.HasInstructions && instrChanged
		return res
	case prev == nil:
		res.IncludeInstructions = parts.HasInstructions
		return res
	}
	// The stored hashes describe items[prev.Offset:]; the first prev.Offset items were dropped
	// from the state and are not compared again.
	k := 0
	if prev.Offset <= len(hashes) {
		current := hashes[prev.Offset:]
		for k < len(prev.ItemHashes) && k < len(current) && prev.ItemHashes[k] == current[k] {
			k++
		}
	}
	if prev.Offset <= len(hashes) && k == len(prev.ItemHashes) {
		res.NewItems = parts.Items[prev.Offset+k:]
		res.PrefixItems = prev.Offset + k
	} else {
		res.Reset = true
	}
	res.IncludeInstructions = parts.HasInstructions && (res.Reset || instrChanged)
	return res
}

// archiveRecord is the JSON object stored per request.
type archiveRecord struct {
	V               int                        `json:"v"`
	RequestID       string                     `json:"request_id"`
	UserID          int64                      `json:"user_id"`
	APIKeyID        int64                      `json:"api_key_id"`
	GroupID         *int64                     `json:"group_id"`
	SessionKey      string                     `json:"session_key"`
	Thread          string                     `json:"thread,omitempty"`
	CreatedAt       string                     `json:"created_at"`
	Model           string                     `json:"model"`
	UpstreamModel   string                     `json:"upstream_model"`
	InboundEndpoint string                     `json:"inbound_endpoint"`
	Protocol        string                     `json:"protocol"`
	UserAgent       string                     `json:"user_agent"`
	IPAddress       string                     `json:"ip_address"`
	Usage           archiveRecordUsage         `json:"usage"`
	Instructions    *string                    `json:"instructions,omitempty"`
	Reset           bool                       `json:"reset"`
	PrefixItems     int                        `json:"prefix_items"`
	NewItems        []json.RawMessage          `json:"new_items"`
	RequestParams   map[string]json.RawMessage `json:"request_params"`
	Response        ArchiveResponse            `json:"response"`
	Images          []archiveRecordImage       `json:"images"`
	Truncated       archiveRecordTruncated     `json:"truncated"`
}

type archiveRecordUsage struct {
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	CacheReadTokens int     `json:"cache_read_tokens"`
	CostUSD         float64 `json:"cost_usd"`
}

// archiveRecordImage: the stored object has no index id (it is written before the index row);
// the admin API fills ids from conversation_archive_images.
type archiveRecordImage struct {
	ID     int64  `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	URL    string `json:"url,omitempty"`
	Mime   string `json:"mime,omitempty"`
	Bytes  int    `json:"bytes,omitempty"`
	Source string `json:"source"`
}

type archiveRecordTruncated struct {
	Request  bool `json:"request"`
	Response bool `json:"response"`
}

// capArchiveItems keeps the newest items that fit the per-record byte cap.
func capArchiveItems(items []json.RawMessage, maxBytes int) ([]json.RawMessage, bool) {
	total := 0
	for _, item := range items {
		total += len(item)
	}
	if total <= maxBytes {
		return items, false
	}
	start := len(items)
	size := 0
	for start > 0 && size+len(items[start-1]) <= maxBytes {
		start--
		size += len(items[start])
	}
	return items[start:], true
}

func gzipJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Grow(len(raw)/4 + 64)
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// archiveSanitizeRequestID keeps request ids safe as an object-key segment.
func archiveSanitizeRequestID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if b.Len() >= archiveRequestIDMaxChars {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			_, _ = b.WriteRune(r)
		default:
			_ = b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "noid"
	}
	return b.String()
}

func archiveTurnKey(prefix string, userID int64, at time.Time, sessionKey, requestID string) string {
	return fmt.Sprintf("%sconv/%d/%s/%s/%d_%s.json.gz", prefix, userID, at.Format("20060102"),
		archiveSHA256Hex(sessionKey)[:16], at.UnixMilli(), archiveSanitizeRequestID(requestID))
}

func archiveImageKey(prefix string, userID int64, at time.Time, requestID string, n int, ext string) string {
	return fmt.Sprintf("%simg/%d/%s/%s_%d.%s", prefix, userID, at.Format("20060102"), archiveSanitizeRequestID(requestID), n, ext)
}

// decodeArchiveImage decodes a base64 payload (optionally a data URL) and detects its type.
func decodeArchiveImage(b64 string) ([]byte, string, string, error) {
	s := strings.TrimSpace(b64)
	if strings.HasPrefix(s, "data:") {
		if i := strings.Index(s, ","); i >= 0 {
			s = s[i+1:]
		}
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(s, "="))
		if err != nil {
			return nil, "", "", fmt.Errorf("decode image: %w", err)
		}
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png":
		return data, mime, "png", nil
	case "image/jpeg":
		return data, mime, "jpg", nil
	case "image/webp":
		return data, mime, "webp", nil
	case "image/gif":
		return data, mime, "gif", nil
	default:
		return data, "application/octet-stream", "bin", nil
	}
}

// archiveLatestUserText returns the latest user-typed text among the items (Codex environment /
// AGENTS.md blocks and Claude Code reminders skipped), cut to the preview length.
func archiveLatestUserText(protocol string, items []json.RawMessage) string {
	for i := len(items) - 1; i >= 0; i-- {
		item := gjson.ParseBytes(items[i])
		role := item.Get("role").String()
		if role != "user" {
			continue
		}
		if protocol == ArchiveProtocolResponses {
			if t := item.Get("type").String(); t != "" && t != "message" {
				continue
			}
		}
		var texts []string
		collect := func(text string) {
			trimmed := strings.TrimSpace(text)
			if trimmed == "" || archiveIsBoilerplate(trimmed) {
				return
			}
			texts = append(texts, trimmed)
		}
		if protocol == ArchiveProtocolGemini {
			forEachArray(item.Get("parts"), func(part gjson.Result) bool {
				collect(part.Get("text").String())
				return true
			})
		} else {
			content := item.Get("content")
			if content.Type == gjson.String {
				collect(content.String())
			} else {
				forEachArray(content, func(part gjson.Result) bool {
					switch part.Get("type").String() {
					case "text", "input_text", "":
						collect(part.Get("text").String())
					}
					return true
				})
			}
		}
		if len(texts) > 0 {
			return archiveHeadRunes(strings.Join(texts, "\n"), archivePreviewMaxRunes)
		}
	}
	return ""
}

func archiveIsBoilerplate(text string) bool {
	for _, prefix := range []string{"<environment_context>", "<user_instructions>", "# AGENTS.md", "<INSTRUCTIONS>", "<system-reminder>", "<permissions instructions>"} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

func archiveHeadRunes(s string, n int) string {
	s = strings.ToValidUTF8(s, "")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
