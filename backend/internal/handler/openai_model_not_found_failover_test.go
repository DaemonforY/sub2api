package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// A ChatGPT account asked for an unreleased model answers 400 model_not_found; once every account
// has failed over, the client must hear "check the model name", not "upstream failed, retry later".
func modelNotFoundFailoverTestError() *service.UpstreamFailoverError {
	return &service.UpstreamFailoverError{
		StatusCode:   http.StatusBadRequest,
		ResponseBody: []byte(`{"detail":"The 'gpt-6.2-astra' model is not supported when using Codex with a ChatGPT account.","error":{"type":"invalid_request_error","code":"model_not_found","message":"The 'gpt-6.2-astra' model is not supported when using Codex with a ChatGPT account."}}`),
	}
}

func newModelNotFoundTestContext(path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(nil))
	setOpsRequestContext(c, "gpt-6.2-astra", false)
	return c, rec
}

func TestOpenAIFailoverExhaustedExplainsUpstreamModelNotFound(t *testing.T) {
	c, rec := newModelNotFoundTestContext("/v1/responses")

	(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, modelNotFoundFailoverTestError(), false)

	require.Equal(t, http.StatusNotFound, rec.Code)
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, "invalid_request_error", envelope.Error.Type)
	require.Equal(t, "model_not_found", envelope.Error.Code)
	require.Contains(t, envelope.Error.Message, `上游暂不支持模型 "gpt-6.2-astra"`)
	require.Contains(t, envelope.Error.Message, "/v1/models")
	require.Contains(t, envelope.Error.Message, "does not exist or is not available upstream")
	require.NotContains(t, envelope.Error.Message, "ChatGPT account")
}

func TestOpenAIAnthropicFailoverExhaustedExplainsUpstreamModelNotFound(t *testing.T) {
	c, rec := newModelNotFoundTestContext("/v1/messages")

	(&OpenAIGatewayHandler{}).handleAnthropicFailoverExhausted(c, modelNotFoundFailoverTestError(), false)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "not_found_error")
	require.Contains(t, rec.Body.String(), "上游暂不支持模型")
}

func TestOpenAIFailoverExhaustedOtherBadRequestKeepsDefaultMapping(t *testing.T) {
	c, rec := newModelNotFoundTestContext("/v1/responses")
	failoverErr := &service.UpstreamFailoverError{
		StatusCode:   http.StatusBadRequest,
		ResponseBody: []byte(`{"error":{"type":"invalid_request_error","code":"invalid_value","message":"bad value"}}`),
	}

	(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, failoverErr, false)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotContains(t, rec.Body.String(), "上游暂不支持模型")
}

func TestUpstreamModelNotFoundMessageWithoutModel(t *testing.T) {
	require.Contains(t, upstreamModelNotFoundMessage(" "), "上游暂不支持这个模型")
}
