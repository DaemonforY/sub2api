//go:build embed

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePlayerSource map[string][]byte

func (f fakePlayerSource) PlayerData(_ context.Context, id string) ([]byte, bool) {
	d, ok := f[id]
	return d, ok
}

func TestVideoPlayPage(t *testing.T) {
	distFS := fstest.MapFS{
		"index.html":                   {Data: []byte("<html>spa</html>")},
		"video/index.html":             {Data: []byte("<html>video</html>")},
		"video/player.html":            {Data: []byte(`<link rel="stylesheet" href="/fonts/fonts.css"><div id="stage"></div><script src="/player/icons.js"></script><script src="/player/runtime.js"></script>`)},
		"video/player/runtime.js":      {Data: []byte("/*runtime*/")},
		"video/player/icons.js":        {Data: []byte("/*icons*/")},
		"video/player/vendor/three.js": {Data: []byte("/*three*/")},
		"video/fonts/fonts.css":        {Data: []byte("/*fonts*/")},
	}
	server := &FrontendServer{distFS: distFS, fileServer: http.FileServer(http.FS(distFS)), baseHTML: []byte("<html>spa</html>"), cache: NewHTMLCache(), settings: &mockSettingsProvider{settings: map[string]string{}}, videoHost: "video.example.com"}
	id := "0d3d66a5-abec-44c8-8ae8-6c5b6cc934db"
	server.SetVideoPlayer(fakePlayerSource{id: []byte(`{"scenes":[{"id":"s1","code":"\u003c/script\u003e"}]}`)})
	router := gin.New()
	router.Use(server.Middleware())
	get := func(path, etag string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "video.example.com"
		req.Header.Set("X-Forwarded-Proto", "https")
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		router.ServeHTTP(w, req)
		return w
	}

	w := get("/play/"+id, "")
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `window.__FILM_DATA__={"scenes"`)
	assert.Contains(t, body, `"code":"\u003c/script\u003e"`, "the data (from json.Marshal) cannot close the script element")
	assert.Regexp(t, `/player/runtime\.js\?v=[0-9a-f]{10}`, body)
	assert.Regexp(t, `/player/icons\.js\?v=[0-9a-f]{10}`, body)
	assert.Regexp(t, `__PLAYER_THREE__="/player/vendor/three\.js\?v=[0-9a-f]{10}"`, body)
	assert.Regexp(t, `/fonts/fonts\.css\?v=`, body)
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "connect-src 'none'")
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "frame-ancestors https://video.example.com")
	assert.Contains(t, w.Header().Get("Cache-Control"), "stale-while-revalidate")
	etag := w.Header().Get("ETag")
	require.NotEmpty(t, etag)

	assert.Equal(t, http.StatusNotModified, get("/play/"+id, etag).Code)
	assert.Equal(t, http.StatusNotFound, get("/play/aaaaaaaa-abec-44c8-8ae8-6c5b6cc934db", "").Code, "not a public work")

	w = get("/player/runtime.js?v=abc", "")
	assert.Equal(t, staticAssetsCacheControl, w.Header().Get("Cache-Control"), "versioned assets are immutable")
	w = get("/player/runtime.js", "")
	assert.NotEqual(t, staticAssetsCacheControl, w.Header().Get("Cache-Control"))
}
