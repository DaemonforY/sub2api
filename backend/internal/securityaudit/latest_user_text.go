package securityaudit

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

// Protocols understood by ExtractLatestUserText.
const (
	LatestUserTextResponses = "openai_responses"
	LatestUserTextChat      = "openai_chat_completions"
	LatestUserTextAnthropic = "anthropic_messages"
	LatestUserTextGemini    = "gemini"
	LatestUserTextImages    = "openai_images"
)

// latestUserTextMediaFallbackMaxBytes bounds the full decode used only when an image request has
// no top-level prompt.
const latestUserTextMediaFallbackMaxBytes = 1 << 20

// ExtractLatestUserText returns the text of the request's latest user message only — not the
// history, not the system prompt, not tool results. When the last user message carries no text
// (an Anthropic turn that only returns tool_result blocks, a Responses turn that only carries
// function_call_output items), the most recent user message that does have text is used. For image
// endpoints it is the prompt. The text is returned as-is (not redacted, not truncated).
//
// It runs on the request path, so it never decodes the whole document: the message list is scanned
// once with gjson and only the messages it needs, walking from the end, are looked into.
func ExtractLatestUserText(protocol string, body []byte) string {
	if len(body) == 0 {
		return ""
	}
	doc := string(body) // one copy; every gjson result below is a substring of it
	if !strings.HasPrefix(strings.TrimSpace(doc), "{") {
		return ""
	}
	switch protocol {
	case LatestUserTextResponses:
		return latestResponsesUserText(doc)
	case LatestUserTextChat, LatestUserTextAnthropic:
		return latestMessagesUserText(gjson.Get(doc, "messages"))
	case LatestUserTextGemini:
		return latestGeminiUserText(gjson.Get(doc, "contents"))
	case LatestUserTextImages:
		if prompt := gjsonString(gjson.Get(doc, "prompt")); prompt != "" {
			return prompt
		}
		if len(body) > latestUserTextMediaFallbackMaxBytes {
			return ""
		}
		var root map[string]any
		if err := json.Unmarshal(body, &root); err != nil {
			return ""
		}
		return strings.Join(extractMediaPrompts(root), "\n\n")
	default:
		return ""
	}
}

// latestResponsesUserText walks Responses `input` from the end. Codex sends its AGENTS.md and
// environment context as separate user messages ahead of the real prompt; those are only used when
// nothing else has text.
func latestResponsesUserText(doc string) string {
	input := gjson.Get(doc, "input")
	if !input.Exists() {
		input = gjson.Get(doc, "response.input")
	}
	switch {
	case input.Type == gjson.String:
		return strings.TrimSpace(input.String())
	case input.IsObject():
		real, all := splitContextBlobs(gjsonUserMessageTexts(input))
		if real != "" {
			return real
		}
		return all
	case input.IsArray():
		items := input.Array()
		fallback := ""
		for index := len(items) - 1; index >= 0; index-- {
			var texts []string
			switch {
			case items[index].Type == gjson.String:
				texts = []string{items[index].String()}
			case items[index].IsObject():
				texts = gjsonUserMessageTexts(items[index])
			}
			real, all := splitContextBlobs(texts)
			if real != "" {
				return real
			}
			if fallback == "" {
				fallback = all
			}
		}
		return fallback
	}
	return ""
}

func latestMessagesUserText(messages gjson.Result) string {
	if !messages.IsArray() {
		return ""
	}
	items := messages.Array()
	fallback := ""
	for index := len(items) - 1; index >= 0; index-- {
		message := items[index]
		if !message.IsObject() || strings.ToLower(gjsonString(message.Get("role"))) != "user" {
			continue
		}
		real, all := splitContextBlobs(gjsonContentTexts(message.Get("content")))
		if real != "" {
			return real
		}
		if fallback == "" {
			fallback = all
		}
	}
	return fallback
}

func latestGeminiUserText(contents gjson.Result) string {
	if !contents.IsArray() {
		return ""
	}
	items := contents.Array()
	for index := len(items) - 1; index >= 0; index-- {
		content := items[index]
		if !content.IsObject() {
			continue
		}
		role := strings.ToLower(gjsonString(content.Get("role")))
		if role != "" && role != "user" {
			continue
		}
		var texts []string
		if parts := content.Get("parts"); parts.IsArray() {
			for _, part := range parts.Array() {
				if text := gjsonString(part.Get("text")); text != "" {
					texts = append(texts, text)
				}
			}
		}
		if text := joinTexts(texts); text != "" {
			return text
		}
	}
	return ""
}

// gjsonUserMessageTexts returns the text parts of a Responses input item when it is a user message.
func gjsonUserMessageTexts(entry gjson.Result) []string {
	role := strings.ToLower(gjsonString(entry.Get("role")))
	typeName := strings.ToLower(gjsonString(entry.Get("type")))
	if role != "user" && (role != "" || (typeName != "" && typeName != "message" && typeName != "input_text")) {
		return nil
	}
	if content := entry.Get("content"); content.Exists() {
		return gjsonContentTexts(content)
	}
	if text := gjsonString(entry.Get("text")); text != "" {
		return []string{text}
	}
	return nil
}

// gjsonContentTexts mirrors contentTexts: a string, an array of text parts, or an object with text.
func gjsonContentTexts(value gjson.Result) []string {
	switch {
	case value.Type == gjson.String:
		return []string{value.String()}
	case value.IsArray():
		var result []string
		for _, part := range value.Array() {
			if !part.IsObject() {
				continue
			}
			typeName := strings.ToLower(gjsonString(part.Get("type")))
			if typeName != "" && typeName != "text" && typeName != "input_text" && typeName != "output_text" {
				continue
			}
			if text := gjsonString(part.Get("text")); text != "" {
				result = append(result, text)
			}
		}
		return result
	case value.IsObject():
		if text := gjsonString(value.Get("text")); text != "" {
			return []string{text}
		}
	}
	return nil
}

func gjsonString(value gjson.Result) string {
	if value.Type != gjson.String {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func joinTexts(texts []string) string {
	parts := make([]string, 0, len(texts))
	for _, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

// splitContextBlobs joins a message's text parts, leaving out injected client context; all is
// every part joined, for when nothing else in the request has text.
func splitContextBlobs(texts []string) (real string, all string) {
	kept := make([]string, 0, len(texts))
	for _, text := range texts {
		if !isClientContextBlob(strings.TrimSpace(text)) {
			kept = append(kept, text)
		}
	}
	return joinTexts(kept), joinTexts(texts)
}

// isClientContextBlob recognizes context a coding agent injects as a user message on its own
// (Codex environment context / AGENTS.md instructions, Claude Code system reminders).
func isClientContextBlob(text string) bool {
	for _, prefix := range []string{"<environment_context>", "<user_instructions>", "# AGENTS.md instructions", "<system-reminder>", "<permissions instructions>"} {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}
