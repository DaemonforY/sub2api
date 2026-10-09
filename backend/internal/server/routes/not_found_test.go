//go:build unit

package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPINotFoundMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/v1/chat/completions", func(c *gin.Context) { c.Status(http.StatusOK) })
	RegisterAPINotFound(r)

	cases := []struct {
		method, path, code, contains string
	}{
		{http.MethodPost, "/completions", "API_PATH_MISSING_V1", "https://hivegpt.cn/v1/completions"},
		{http.MethodPost, "/messages", "API_PATH_MISSING_V1", "少了 /v1"},
		{http.MethodPost, "/audio/speech", "API_PATH_MISSING_V1", "少了 /v1"},
		{http.MethodPost, "/v1/v1/chat/completions", "API_PATH_DUPLICATED_V1", "API 地址改成 https://hivegpt.cn"},
		{http.MethodGet, "/v1", "API_BASE_URL", "POST https://hivegpt.cn/v1/chat/completions"},
		{http.MethodGet, "/v1/chat/completions", "API_NOT_FOUND", "GET /v1/chat/completions"},
		{http.MethodPost, "/v1/nope", "API_NOT_FOUND", "接口不存在"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		req.Host = "hivegpt.cn"
		req.Header.Set("X-Forwarded-Proto", "https")
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code, tc.path)
		require.Contains(t, w.Header().Get("Content-Type"), "application/json", tc.path)
		var body struct {
			Code, Message string
			Error         *struct{ Message, Type, Code string }
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), tc.path)
		require.Equal(t, tc.code, body.Code, tc.path)
		require.Contains(t, body.Message, tc.contains, tc.path)
		require.Contains(t, body.Message, "（", tc.path) // English kept in parentheses
		require.NotNil(t, body.Error, tc.path) // OpenAI-style error object for SDKs
		require.Equal(t, body.Message, body.Error.Message, tc.path)
		require.Equal(t, "not_found_error", body.Error.Type, tc.path)
	}
}
