//go:build unit

package service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Shape captured from the ChatGPT backend on 2026-11-12: a safety block arrives
// as an image_generation_call item with status "failed", not as response.failed.
const failedModerationItemSSE = "event: response.output_item.done\n" +
	"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_1\",\"type\":\"image_generation_call\",\"status\":\"failed\",\"error\":{\"type\":\"image_generation_user_error\",\"code\":\"moderation_blocked\",\"message\":\"Your request was rejected by the safety system. safety_violations=[sexual].\",\"moderation_details\":{\"categories\":[\"sexual\"],\"moderation_stage\":\"output\"},\"param\":null}},\"output_index\":0}\n\n"

const emptyFinalMessageSSE = "event: response.output_item.done\n" +
	"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"msg_1\",\"type\":\"message\",\"status\":\"completed\",\"content\":[{\"type\":\"output_text\",\"text\":\"\"}],\"role\":\"assistant\"},\"output_index\":1}\n\n" +
	"event: response.completed\n" +
	"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"status\":\"completed\",\"model\":\"gpt-5.6-luna\",\"output\":[]}}\n\n"

// 1x1 transparent PNG.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

func newImagesTestContext() (*httptest.ResponseRecorder, *gin.Context) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	return rec, c
}

func sseResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

func requireModerationUserError(t *testing.T, err error) {
	t.Helper()
	var failoverErr *UpstreamFailoverError
	require.False(t, errors.As(err, &failoverErr), "a safety block must not fail over to another account")
	var imgErr *OpenAIImagesUpstreamError
	require.True(t, errors.As(err, &imgErr), "got %T: %v", err, err)
	require.Equal(t, http.StatusBadRequest, imgErr.StatusCode)
	require.Equal(t, "moderation_blocked", imgErr.Code)
	require.False(t, IsOpenAIImagesRetryableUpstreamError(imgErr))
}

func TestImagesOAuthNonStreaming_FailedImageCallReturnsModerationError(t *testing.T) {
	_, c := newImagesTestContext()
	svc := &OpenAIGatewayService{}
	_, _, _, err := svc.handleOpenAIImagesOAuthNonStreamingResponse(sseResponse(failedModerationItemSSE+emptyFinalMessageSSE), c, "b64_json", "gpt-image-2.6-sunburst")
	requireModerationUserError(t, err)
}

func TestImagesOAuthStreaming_FailedImageCallReturnsModerationError(t *testing.T) {
	rec, c := newImagesTestContext()
	svc := &OpenAIGatewayService{}
	_, _, _, _, err := svc.handleOpenAIImagesOAuthStreamingResponse(sseResponse(failedModerationItemSSE+emptyFinalMessageSSE), c, time.Now(), "b64_json", "image_generation", "gpt-image-2.6-sunburst")
	requireModerationUserError(t, err)
	require.Contains(t, rec.Body.String(), "event: error")
}

func TestImagesOAuthNonStreaming_FailedCallThenSuccessStillReturnsImage(t *testing.T) {
	success := "event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_2\",\"type\":\"image_generation_call\",\"status\":\"completed\",\"output_format\":\"png\",\"result\":\"" + tinyPNGBase64 + "\"},\"output_index\":2}\n\n"
	_, c := newImagesTestContext()
	svc := &OpenAIGatewayService{}
	_, count, _, err := svc.handleOpenAIImagesOAuthNonStreamingResponse(sseResponse(failedModerationItemSSE+success+emptyFinalMessageSSE), c, "b64_json", "gpt-image-2.6-sunburst")
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestImagesOAuthStreaming_FailedCallThenSuccessStillReturnsImage(t *testing.T) {
	success := "event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"ig_2\",\"type\":\"image_generation_call\",\"status\":\"completed\",\"output_format\":\"png\",\"result\":\"" + tinyPNGBase64 + "\"},\"output_index\":2}\n\n"
	rec, c := newImagesTestContext()
	svc := &OpenAIGatewayService{}
	_, count, _, _, err := svc.handleOpenAIImagesOAuthStreamingResponse(sseResponse(failedModerationItemSSE+success+emptyFinalMessageSSE), c, time.Now(), "b64_json", "image_generation", "gpt-image-2.6-sunburst")
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NotContains(t, rec.Body.String(), "event: error")
}
