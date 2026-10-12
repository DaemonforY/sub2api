//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestServeImageFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "images"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "images", "imgtask_a-0.png"), []byte("\x89PNG\r\n\x1a\nrest"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(dir), "secret.png"), []byte("nope"), 0o644))
	r := gin.New()
	r.GET("/api/v1/image-files/*key", ServeImageFile(dir))

	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}
	w := get("/api/v1/image-files/images/imgtask_a-0.png")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, w.Header().Get("Cache-Control"), "immutable")
	require.Equal(t, "\x89PNG\r\n\x1a\nrest", w.Body.String())

	for _, p := range []string{
		"/api/v1/image-files/images/missing.png",
		"/api/v1/image-files/../secret.png",
		"/api/v1/image-files/images/%2e%2e/%2e%2e/secret.png",
		"/api/v1/image-files/images/imgtask_a-0.txt",
		"/api/v1/image-files/images",
	} {
		require.Equal(t, http.StatusNotFound, get(p).Code, p)
	}
}
