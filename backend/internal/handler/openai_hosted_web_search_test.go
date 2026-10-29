//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIFailoverExhaustedExplainsHostedWebSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	run := func(webSearch bool, failoverErr *service.UpstreamFailoverError) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
		if webSearch {
			c.Set(ctxKeyOpenAIHostedWebSearch, true)
		}
		(&OpenAIGatewayHandler{}).handleFailoverExhausted(c, failoverErr, false)
		return rec
	}

	// OpenAI's own server_error, passed through by a sub2api upstream: the tool is the likely cause.
	serverError := []byte(`{"error":{"type":"api_error","code":"server_error","message":"An error occurred while processing your request. You can retry your request, or contact us through our help center at help.openai.com if the error persists."}}`)
	rec := run(true, &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ResponseBody: serverError})
	require.Equal(t, http.StatusBadRequest, rec.Code, "a 4xx so SDKs do not keep retrying a tool that cannot run")
	require.Contains(t, rec.Body.String(), "联网搜索工具（web_search）在当前分组暂时不可用")
	require.Contains(t, rec.Body.String(), "本次请求未扣费")

	// Same message without the code (upstream sub2api rewrote the type) still counts.
	rewritten := []byte(`{"error":{"type":"api_error","message":"An error occurred while processing your request. You can retry your request."}}`)
	rec = run(true, &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ResponseBody: rewritten})
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "web_search")

	// An error that names the tool.
	named := []byte(`{"error":{"type":"server_error","message":"web_search call failed"}}`)
	rec = run(true, &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: named})
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// Without the tool in the request nothing is blamed on it.
	rec = run(false, &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ResponseBody: serverError})
	require.GreaterOrEqual(t, rec.Code, 500)
	require.NotContains(t, rec.Body.String(), "web_search")

	// Codex sends web_search on every turn: these say nothing about the tool and must stay
	// retryable 5xx (seen in prod 10-23..10-29 as 「web_search 不可用」 misreports).
	unrelated := []struct {
		name string
		err  *service.UpstreamFailoverError
	}{
		{"connection reset", &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway, ConnectionFailure: true}},
		{"first output timeout", &service.UpstreamFailoverError{StatusCode: http.StatusGatewayTimeout, ConnectionFailure: true}},
		{"empty body", &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway}},
		{"upstream concurrency full", &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway,
			ResponseBody: []byte(`{"error":{"type":"upstream_error","message":"Concurrency limit exceeded for account, please retry later"}}`)}},
		{"upstream temporarily unavailable", &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway,
			ResponseBody: []byte(`{"error":{"type":"upstream_error","message":"Upstream service temporarily unavailable"}}`)}},
		{"cloudflare 524", &service.UpstreamFailoverError{StatusCode: 524, ResponseBody: []byte(`error code: 524`)}},
	}
	for _, tc := range unrelated {
		rec = run(true, tc.err)
		require.GreaterOrEqual(t, rec.Code, 500, tc.name)
		require.NotContains(t, rec.Body.String(), "web_search", tc.name)
	}
}

func TestIsOpenAIHostedWebSearchFailureBody(t *testing.T) {
	require.False(t, isOpenAIHostedWebSearchFailureBody(nil))
	require.True(t, isOpenAIHostedWebSearchFailureBody([]byte(`{"response":{"error":{"code":"server_error","message":"x"}}}`)))
	require.False(t, isOpenAIHostedWebSearchFailureBody([]byte(`{"error":{"code":"server_is_overloaded","message":"Our servers are currently overloaded. Please try again later."}}`)))
	require.False(t, isOpenAIHostedWebSearchFailureBody([]byte(`<html>Bad gateway</html>`)))
}
