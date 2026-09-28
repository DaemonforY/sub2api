package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type captureGatewayRecorder struct {
	mu      sync.Mutex
	entries []*service.GatewayRequestLog
}

func (r *captureGatewayRecorder) Enabled() bool { return true }
func (r *captureGatewayRecorder) Record(e *service.GatewayRequestLog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

func newGatewayLogEngine(t *testing.T, rec *captureGatewayRecorder) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	SetGatewayRequestLogRecorder(rec)
	t.Cleanup(func() { SetGatewayRequestLogRecorder(nil) })

	r := gin.New()
	r.Use(GatewayRequestLog())
	r.POST("/v1/images/generations", func(c *gin.Context) {
		uid, gid := int64(7), int64(3)
		c.Set(string(ContextKeyAPIKey), &service.APIKey{ID: 42, UserID: uid, GroupID: &gid, Key: "sk-authoritative"})
		c.Set(service.OpsModelContextKey, "gpt-image-2")
		c.Set(service.OpsAccountIDContextKey, int64(9))
		c.Status(http.StatusOK)
	})
	r.POST("/chat/completions", func(c *gin.Context) {
		MarkIngressRejected(c, IngressRejectInvalidAPIKey)
		c.AbortWithStatus(http.StatusUnauthorized)
	})
	r.GET("/api/v1/admin/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/assets/app.js", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestGatewayRequestLog_RecordsSuccessWithContextIdentity(t *testing.T) {
	rec := &captureGatewayRecorder{}
	r := newGatewayLogEngine(t, rec)

	req := httptest.NewRequest(http.MethodPost, "https://hivegpt.cn/v1/images/generations?foo=bar", nil)
	req.Header.Set("Authorization", "Bearer sk-presented")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("User-Agent", "curl/8")
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.Len(t, rec.entries, 1)
	e := rec.entries[0]
	require.Equal(t, "POST", e.Method)
	require.Equal(t, "https://hivegpt.cn/v1/images/generations?foo=bar", e.URL)
	require.Equal(t, "/v1/images/generations", e.Path)
	require.Equal(t, http.StatusOK, e.StatusCode)
	require.True(t, e.Success)
	require.Equal(t, "sk-authoritative", e.APIKey, "context key wins over the header value")
	require.Equal(t, int64(42), *e.APIKeyID)
	require.Equal(t, int64(7), *e.UserID)
	require.Equal(t, int64(3), *e.GroupID)
	require.Equal(t, int64(9), *e.AccountID)
	require.Equal(t, "gpt-image-2", e.Model)
	require.Equal(t, "curl/8", e.UserAgent)
}

func TestGatewayRequestLog_RecordsRejectedRequestWithPresentedKey(t *testing.T) {
	rec := &captureGatewayRecorder{}
	r := newGatewayLogEngine(t, rec)

	req := httptest.NewRequest(http.MethodPost, "/chat/completions", nil)
	req.Header.Set("x-api-key", "sk-does-not-exist")
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.Len(t, rec.entries, 1)
	e := rec.entries[0]
	require.Equal(t, http.StatusUnauthorized, e.StatusCode)
	require.False(t, e.Success)
	require.Equal(t, "sk-does-not-exist", e.APIKey)
	require.Equal(t, string(IngressRejectInvalidAPIKey), e.ErrorCode)
	require.Nil(t, e.UserID)
}

func TestGatewayRequestLog_RecordsUnmatchedGatewayRoute(t *testing.T) {
	rec := &captureGatewayRecorder{}
	r := newGatewayLogEngine(t, rec)

	req := httptest.NewRequest(http.MethodPost, "/v1/typo/endpoint", nil)
	req.Header.Set("Authorization", "Bearer sk-typo")
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.Len(t, rec.entries, 1)
	require.Equal(t, http.StatusNotFound, rec.entries[0].StatusCode)
	require.Equal(t, "sk-typo", rec.entries[0].APIKey)
}

func TestGatewayRequestLog_RecordsCredentialedRequestOutsideGatewayPrefixes(t *testing.T) {
	rec := &captureGatewayRecorder{}
	r := newGatewayLogEngine(t, rec)

	req := httptest.NewRequest(http.MethodGet, "/some/other/path?key=AIza-query-key", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.Len(t, rec.entries, 1)
	require.Equal(t, "AIza-query-key", rec.entries[0].APIKey)
}

func TestGatewayRequestLog_SkipsWebAPIStaticAndPreflight(t *testing.T) {
	rec := &captureGatewayRecorder{}
	r := newGatewayLogEngine(t, rec)

	webAPI := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	webAPI.Header.Set("Authorization", "Bearer jwt-token")
	r.ServeHTTP(httptest.NewRecorder(), webAPI)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodOptions, "/v1/chat/completions", nil))

	require.Empty(t, rec.entries)
}

func TestGatewayRequestLog_NoRecorderIsNoop(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetGatewayRequestLogRecorder(nil)
	r := gin.New()
	r.Use(GatewayRequestLog())
	r.GET("/v1/models", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	require.Equal(t, http.StatusOK, w.Code)
}
