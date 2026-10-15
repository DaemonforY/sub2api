//go:build unit

package securityaudit

import (
	"strings"
	"testing"
)

func TestExtractLatestUserText_Responses(t *testing.T) {
	body := `{"model":"gpt-5","instructions":"You are Codex","input":[
		{"type":"message","role":"developer","content":[{"type":"input_text","text":"dev rules"}]},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"# AGENTS.md instructions for /repo\n..."}]},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>cwd</environment_context>"}]},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"fix the failing test"},{"type":"input_image","image_url":"data:image/png;base64,AAA"}]},
		{"type":"function_call","name":"shell","arguments":"{\"command\":[\"ls\"]}","call_id":"c1"},
		{"type":"function_call_output","call_id":"c1","output":"a.go"},
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"done"}]}
	]}`
	if got := ExtractLatestUserText(LatestUserTextResponses, []byte(body)); got != "fix the failing test" {
		t.Fatalf("got %q", got)
	}
	if got := ExtractLatestUserText(LatestUserTextResponses, []byte(`{"input":"hello"}`)); got != "hello" {
		t.Fatalf("string input: got %q", got)
	}
	// Only injected context: fall back to it rather than nothing.
	onlyCtx := `{"input":[{"role":"user","content":[{"type":"input_text","text":"<environment_context>x</environment_context>"}]}]}`
	if got := ExtractLatestUserText(LatestUserTextResponses, []byte(onlyCtx)); got != "<environment_context>x</environment_context>" {
		t.Fatalf("fallback: got %q", got)
	}
}

func TestExtractLatestUserText_Chat(t *testing.T) {
	body := `{"messages":[{"role":"system","content":"sys"},{"role":"user","content":"first"},
		{"role":"assistant","content":"ok"},{"role":"user","content":[{"type":"text","text":"second"},{"type":"image_url","image_url":{"url":"x"}}]},
		{"role":"tool","content":"result"}]}`
	if got := ExtractLatestUserText(LatestUserTextChat, []byte(body)); got != "second" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractLatestUserText_AnthropicSkipsToolResults(t *testing.T) {
	body := `{"system":"sys","messages":[
		{"role":"user","content":[{"type":"text","text":"<system-reminder>ctx</system-reminder>"},{"type":"text","text":"refactor main.go"}]},
		{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{"path":"main.go"}}]},
		{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"package main"}]}
	]}`
	if got := ExtractLatestUserText(LatestUserTextAnthropic, []byte(body)); got != "refactor main.go" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractLatestUserText_GeminiAndImages(t *testing.T) {
	gemini := `{"contents":[{"role":"user","parts":[{"text":"q1"}]},{"role":"model","parts":[{"text":"a1"}]},{"role":"user","parts":[{"text":"q2"}]}]}`
	if got := ExtractLatestUserText(LatestUserTextGemini, []byte(gemini)); got != "q2" {
		t.Fatalf("gemini: got %q", got)
	}
	images := `{"model":"gpt-image-1","prompt":"a cat on the moon","size":"1024x1024"}`
	if got := ExtractLatestUserText(LatestUserTextImages, []byte(images)); got != "a cat on the moon" {
		t.Fatalf("images: got %q", got)
	}
	if got := ExtractLatestUserText(LatestUserTextChat, []byte(`not json`)); got != "" {
		t.Fatalf("invalid json: got %q", got)
	}
}

func TestExtractLatestUserText_LargeHistory(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"model":"gpt-5","input":[`)
	filler := strings.Repeat("x", 4000)
	for i := 0; i < 5000; i++ { // ~20 MB of earlier turns
		b.WriteString(`{"type":"function_call_output","call_id":"c","output":"` + filler + `"},`)
	}
	b.WriteString(`{"role":"user","content":[{"type":"input_text","text":"latest question"}]}]}`)
	if got := ExtractLatestUserText(LatestUserTextResponses, []byte(b.String())); got != "latest question" {
		t.Fatalf("got %q", got)
	}
}

func BenchmarkExtractLatestUserText_20MB(b *testing.B) {
	var sb strings.Builder
	sb.WriteString(`{"messages":[`)
	filler := strings.Repeat("y", 4000)
	for i := 0; i < 5000; i++ {
		sb.WriteString(`{"role":"assistant","content":"` + filler + `"},`)
	}
	sb.WriteString(`{"role":"user","content":"q"}]}`)
	body := []byte(sb.String())
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractLatestUserText(LatestUserTextChat, body)
	}
}
