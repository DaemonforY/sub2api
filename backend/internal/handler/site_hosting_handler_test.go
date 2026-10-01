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

func TestSiteHostMiddlewareKeepsTheMainSiteOffSiteHosts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewSiteHostingService(nil, nil, nil, nil, service.DefaultSiteHostingConfig("s.example.test"), t.TempDir(), "https://hivegpt.cn")
	h := NewSiteHostingHandler(svc)
	r := gin.New()
	r.Use(h.HostMiddleware)
	r.GET("/api/v1/user/profile", func(c *gin.Context) { c.String(http.StatusOK, "secret profile") })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
	req.Host = "hivegpt.cn"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, "secret profile", w.Body.String())

	for _, host := range []string{"s.example.test", "bad_name.s.example.test", "s.example.test:8080"} {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/user/profile", nil)
		req.Host = host
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNotFound, w.Code, host)
		require.NotContains(t, w.Body.String(), "secret", host)
		require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	}
}
