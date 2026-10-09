//go:build unit

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAbortWithErrorAddsOpenAIErrorOnGatewayPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/*path", func(c *gin.Context) {
		AbortWithError(c, http.StatusUnauthorized, "INVALID_API_KEY", "API Key 无效（Invalid API key）")
	})

	type body struct {
		Code    string           `json:"code"`
		Message string           `json:"message"`
		Error   *CompatErrorBody `json:"error"`
	}
	get := func(path string) body {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		require.Equal(t, http.StatusUnauthorized, w.Code)
		var b body
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b))
		// The top-level fields stay for existing clients.
		require.Equal(t, "INVALID_API_KEY", b.Code, path)
		require.Equal(t, "API Key 无效（Invalid API key）", b.Message, path)
		return b
	}

	for _, p := range []string{"/v1/chat/completions", "/v1/messages", "/v1/models", "/chat/completions", "/responses", "/images/generations", "/backend-api/codex/responses", "/antigravity/v1/messages"} {
		b := get(p)
		require.NotNil(t, b.Error, p)
		require.Equal(t, CompatErrorBody{Message: "API Key 无效（Invalid API key）", Type: "authentication_error", Code: "INVALID_API_KEY"}, *b.Error, p)
	}
	for _, p := range []string{"/v1beta/models/gemini:generateContent", "/api/v1/auth/login", "/antigravity/v1beta/models", "/v1betax"} {
		require.Nil(t, get(p).Error, p)
	}
}

func TestCompatErrorType(t *testing.T) {
	require.Equal(t, "authentication_error", CompatErrorType(401))
	require.Equal(t, "permission_error", CompatErrorType(403))
	require.Equal(t, "not_found_error", CompatErrorType(404))
	require.Equal(t, "rate_limit_error", CompatErrorType(429))
	require.Equal(t, "api_error", CompatErrorType(503))
	require.Equal(t, "invalid_request_error", CompatErrorType(400))
}
