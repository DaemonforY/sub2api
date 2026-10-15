//go:build unit

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubVerifyFiles map[string]string

func (s stubVerifyFiles) GetWeChatDomainVerifyFile(_ context.Context, name string) (string, bool) {
	v, ok := s[name]
	return v, ok
}

func TestWeChatDomainVerifyFilesServedBeforeSPAFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(wechatDomainVerifyFiles(stubVerifyFiles{"MP_verify_AbCd1234.txt": "AbCd1234"}))
	// Stands in for the embedded frontend, which answers every path with index.html.
	r.Use(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html>spa</html>"))
		c.Abort()
	})

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	rec := get("/MP_verify_AbCd1234.txt")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "AbCd1234", rec.Body.String())
	require.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))

	// Unknown verify files and other paths fall through to the SPA.
	require.Contains(t, get("/MP_verify_Other999.txt").Body.String(), "spa")
	require.Contains(t, get("/dashboard").Body.String(), "spa")
	require.Contains(t, get("/sub/MP_verify_AbCd1234.txt").Body.String(), "spa")
}
