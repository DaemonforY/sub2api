//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIImagesSafetyRejectionIsExplainedInChinese(t *testing.T) {
	up := "Your request was rejected by the safety system. If you believe this is an error, contact us at help.openai.com and include the request ID 054d. safety_violations=[sexual]."
	e := &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, ErrorType: "image_generation_user_error", Code: "moderation_blocked", Message: up}
	msg := e.clientMessage()
	require.Contains(t, msg, "触发了 OpenAI 的内容安全审核")
	require.Contains(t, msg, "涉及色情或性暗示")
	require.Contains(t, msg, "本次请求未扣费")
	require.Contains(t, msg, "（"+up+"）")

	// Detected from the text alone (a plain invalid_request_error from a relaying upstream).
	e = &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, ErrorType: "invalid_request_error", Message: "Your request was rejected by the safety system."}
	require.Contains(t, e.clientMessage(), "请换一种说法再试")
	require.NotContains(t, e.clientMessage(), "涉及")

	// Other errors and upstream failures are left alone.
	e = &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, ErrorType: "invalid_request_error", Message: "Invalid size"}
	require.Equal(t, "Invalid size", e.clientMessage())
	e = &OpenAIImagesUpstreamError{StatusCode: http.StatusBadGateway, Message: "moderation service down"}
	require.Equal(t, "moderation service down", e.clientMessage())
	// Already Chinese: shown as is.
	e = &OpenAIImagesUpstreamError{StatusCode: http.StatusBadRequest, Code: "moderation_blocked", Message: "内容审核未通过"}
	require.Equal(t, "内容审核未通过", e.clientMessage())
}
