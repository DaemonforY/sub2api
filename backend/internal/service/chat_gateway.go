package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// Model calls of AI 对话 through this site's gateway (billed on the user's key as usual).

type chatStreamResult struct {
	model            string
	promptTokens     int
	completionTokens int
}

// stream makes a streamed chat completion, passing each piece of answer text and of reasoning
// (models that stream their thinking as reasoning_content / reasoning) to onDelta.
func (s *ChatService) stream(ctx context.Context, key string, body map[string]any, onDelta func(delta, reasoning string) error) (*chatStreamResult, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gatewayURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", chatUserAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, chatFailed("连接模型服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, chatUpstreamFailed(raw, resp.StatusCode)
	}
	out := &chatStreamResult{}
	got := false
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
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
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
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
			return out, chatUpstreamFailed([]byte(data), http.StatusBadGateway)
		}
		if chunk.Model != "" {
			out.model = chunk.Model
		}
		if chunk.Usage != nil {
			out.promptTokens, out.completionTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
		}
		for _, c := range chunk.Choices {
			reasoning := c.Delta.ReasoningContent
			if reasoning == "" {
				reasoning = c.Delta.Reasoning
			}
			if c.Delta.Content == "" && reasoning == "" {
				continue
			}
			if c.Delta.Content != "" {
				got = true
			}
			if err := onDelta(c.Delta.Content, reasoning); err != nil {
				return out, err
			}
		}
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if err := scanner.Err(); err != nil {
		return out, chatFailed("读取回答中断")
	}
	if !got {
		return out, chatFailed("模型没有返回内容")
	}
	return out, nil
}

type chatImageRef struct {
	name string
	data []byte
}

// gatewayDraw draws one picture with gpt-image-2: from the prompt alone, or as an edit of the
// reference pictures. It returns the image bytes.
func (s *ChatService) gatewayDraw(ctx context.Context, key, prompt, size string, refs []chatImageRef) ([]byte, error) {
	var (
		body        io.Reader
		contentType string
		path        string
	)
	if len(refs) == 0 {
		raw, _ := json.Marshal(map[string]any{"model": chatImageModel, "prompt": prompt, "size": size, "quality": "medium", "n": 1})
		body, contentType, path = bytes.NewReader(raw), "application/json", "/v1/images/generations"
	} else {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for k, v := range map[string]string{"model": chatImageModel, "prompt": prompt, "size": size, "quality": "medium", "n": "1"} {
			_ = w.WriteField(k, v)
		}
		for i, ref := range refs {
			mime := http.DetectContentType(ref.data)
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image[]"; filename="ref%d.%s"`, i+1, chatFileExt[mime]))
			h.Set("Content-Type", mime)
			part, err := w.CreatePart(h)
			if err != nil {
				return nil, err
			}
			if _, err := part.Write(ref.data); err != nil {
				return nil, err
			}
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		body, contentType, path = &buf, w.FormDataContentType(), "/v1/images/edits"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.gatewayURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", chatUserAgent)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, chatFailed("连接画图服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 60<<20))
	if err != nil {
		return nil, chatFailed("读取图片中断")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, chatUpstreamFailed(raw, resp.StatusCode)
	}
	var r struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Data) == 0 {
		return nil, chatFailed("画图服务没有返回图片")
	}
	if r.Data[0].B64JSON == "" && r.Data[0].URL != "" {
		return gatewayImageURL(ctx, s.client, r.Data[0].URL)
	}
	data, err := base64.StdEncoding.DecodeString(r.Data[0].B64JSON)
	if err != nil || len(data) == 0 {
		return nil, chatFailed("画图服务返回的图片无法识别")
	}
	return data, nil
}

// gatewayVideo creates a video task on the gateway's /v1/videos, polls it until it is done, and
// downloads the MP4. firstFrame is an optional data URL used as the opening frame.
func (s *ChatService) gatewayVideo(ctx context.Context, key, prompt, aspect, firstFrame string) ([]byte, error) {
	body := map[string]any{"model": chatVideoModel, "prompt": prompt, "aspect_ratio": aspect}
	if firstFrame != "" {
		body["image"] = firstFrame
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	raw, status, err := s.gatewayCall(ctx, key, http.MethodPost, "/v1/videos", payload, 1<<20)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, chatUpstreamFailed(raw, status)
	}
	var created struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &created) != nil || created.ID == "" {
		return nil, chatFailed("视频任务创建失败")
	}
	taskPath := "/v1/videos/" + url.PathEscape(created.ID)
	for {
		select {
		case <-ctx.Done():
			return nil, chatFailed("视频生成超时")
		case <-time.After(s.videoPoll):
		}
		raw, status, err = s.gatewayCall(ctx, key, http.MethodGet, taskPath, nil, 1<<20)
		if err != nil {
			if ctx.Err() != nil {
				return nil, chatFailed("视频生成超时")
			}
			continue // a lost poll is retried on the next tick
		}
		if status != http.StatusOK {
			return nil, chatUpstreamFailed(raw, status)
		}
		var task struct {
			Status string          `json:"status"`
			Error  json.RawMessage `json:"error"`
		}
		_ = json.Unmarshal(raw, &task)
		switch task.Status {
		case "done":
			data, status, err := s.gatewayCall(ctx, key, http.MethodGet, taskPath+"/content", nil, ChatVideoMaxBytes+1)
			if err != nil {
				return nil, chatFailed("视频下载失败")
			}
			if status != http.StatusOK {
				return nil, chatUpstreamFailed(data, status)
			}
			return data, nil
		case "failed", "expired":
			return nil, chatFailed("视频生成失败：" + chatVideoTaskError(task.Error) + "。这次不扣次数")
		}
	}
}

// chatVideoTaskError reads a failed task's error, which muse2api sends as a string and xAI as an object.
func chatVideoTaskError(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil && strings.TrimSpace(text) != "" {
		return chatCutError(text)
	}
	var obj struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &obj) == nil && strings.TrimSpace(obj.Message) != "" {
		return chatCutError(obj.Message)
	}
	return "上游没有返回原因"
}

func chatCutError(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > 120 {
		return string(r[:120]) + "…"
	}
	return text
}

// gatewayCall is one plain request to the gateway on the user's key; the body is read up to limit bytes.
func (s *ChatService) gatewayCall(ctx context.Context, key, method, path string, payload []byte, limit int64) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.gatewayURL+path, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", chatUserAgent)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, 0, err
	}
	return data, resp.StatusCode, nil
}
