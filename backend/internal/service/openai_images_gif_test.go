//go:build unit

package service

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func parseOpenAIImagesForGIFTest(t *testing.T, path, contentType string, body []byte) *OpenAIImagesRequest {
	t.Helper()
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	parsed, err := (&OpenAIGatewayService{}).ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	return parsed
}

func TestOpenAIImagesGIFFrameOptionsReachTheImageTool(t *testing.T) {
	body := []byte(`{"model":"gpt-image-2.6-flare","prompt":"a cat blinking","output_format":"gif","frame_count":6,"frame_duration_ms":150}`)
	parsed := parseOpenAIImagesForGIFTest(t, "/v1/images/generations", "application/json", body)
	require.Equal(t, "gif", parsed.OutputFormat)
	require.Equal(t, 6, *parsed.FrameCount)
	require.Equal(t, 150, *parsed.FrameDurationMS)

	payload, err := buildOpenAIImagesResponsesRequest(parsed, "gpt-image-2.6-flare")
	require.NoError(t, err)
	tool := gjson.GetBytes(payload, "tools.0")
	require.Equal(t, "gif", tool.Get("output_format").String())
	require.Equal(t, int64(6), tool.Get("frame_count").Int())
	require.Equal(t, int64(150), tool.Get("frame_duration_ms").Int())

	plain := parseOpenAIImagesForGIFTest(t, "/v1/images/generations", "application/json", []byte(`{"model":"gpt-image-2","prompt":"a cat"}`))
	payload, err = buildOpenAIImagesResponsesRequest(plain, "gpt-image-2")
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(payload, "tools.0.frame_count").Exists())
	require.False(t, gjson.GetBytes(payload, "tools.0.frame_duration_ms").Exists())
}

func TestOpenAIImagesGIFFrameOptionsMultipart(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range map[string]string{"model": "gpt-image-2.6-sunburst", "prompt": "make it wave", "output_format": "gif", "frame_count": "8", "frame_duration_ms": "120"} {
		require.NoError(t, w.WriteField(k, v))
	}
	part, err := w.CreateFormFile("image", "a.png")
	require.NoError(t, err)
	_, _ = part.Write([]byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, w.Close())

	parsed := parseOpenAIImagesForGIFTest(t, "/v1/images/edits", w.FormDataContentType(), buf.Bytes())
	require.Equal(t, 8, *parsed.FrameCount)
	require.Equal(t, 120, *parsed.FrameDurationMS)
	require.True(t, parsed.HasNativeOptions)
}

func TestOpenAIImagesGIFNotEnabledIsExplainedInChinese(t *testing.T) {
	up := "GIF output is not enabled for the model 'gpt-image-2-codex'."
	e := &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, ErrorType: "image_generation_user_error", Code: "invalid_value", Message: up}
	msg := e.clientMessage()
	require.Contains(t, msg, "暂不支持生成 GIF 动图")
	require.Contains(t, msg, "本次请求未扣费")
	require.Contains(t, msg, "（"+up+"）")

	other := &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, Message: "Invalid size"}
	require.Equal(t, "Invalid size", other.clientMessage())
}
