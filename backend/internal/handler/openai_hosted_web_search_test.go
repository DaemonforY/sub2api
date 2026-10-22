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

	rec := run(true, &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway})
	require.Equal(t, http.StatusBadRequest, rec.Code, "a 4xx so SDKs do not keep retrying a tool that cannot run")
	require.Contains(t, rec.Body.String(), "联网搜索工具（web_search）在当前分组暂时不可用")
	require.Contains(t, rec.Body.String(), "本次请求未扣费")

	rec = run(false, &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway})
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotContains(t, rec.Body.String(), "web_search")

	// Codex sends web_search on every turn: a reset connection must stay a retryable 502,
	// not a 400 telling the user to drop the tool.
	rec = run(true, &service.UpstreamFailoverError{StatusCode: http.StatusBadGateway, ConnectionFailure: true})
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotContains(t, rec.Body.String(), "web_search")

	rec = run(true, &service.UpstreamFailoverError{StatusCode: http.StatusGatewayTimeout, ConnectionFailure: true})
	require.GreaterOrEqual(t, rec.Code, 500)
	require.NotContains(t, rec.Body.String(), "web_search")
}
