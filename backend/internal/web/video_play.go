//go:build embed

package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// /play/<work id> on the video host: player.html with one public work's scene data inlined and
// versioned asset URLs, so a gallery card shows its first frame after a single request. The page is
// cacheable (ETag + stale-while-revalidate) and the assets it names are immutable.

// SetVideoPlayer supplies public works for /play/<id>.
func (s *FrontendServer) SetVideoPlayer(src VideoPlayerSource) { s.player = src }

var playPath = regexp.MustCompile(`^play/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

type playerAssets struct {
	once     sync.Once
	shell    []byte // player.html with versioned asset URLs and a data placeholder
	versions map[string]string
}

const playDataMarker = "<!--hivegpt-play-data-->"

// assetVersion is a short content hash of a file in dist/video.
func assetVersion(fsys fs.FS, name string) string {
	raw, err := fs.ReadFile(fsys, "video/"+name)
	if err != nil {
		return "0"
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:5])
}

func (s *FrontendServer) playShell() []byte {
	s.playAssets.once.Do(func() {
		raw, err := fs.ReadFile(s.distFS, "video/player.html")
		if err != nil {
			return
		}
		v := map[string]string{}
		for _, f := range []string{"player/runtime.js", "player/icons.js", "fonts/fonts.css", "player/vendor/three.js"} {
			v[f] = assetVersion(s.distFS, f)
		}
		html := string(raw)
		html = strings.Replace(html, `href="/fonts/fonts.css"`, `href="/fonts/fonts.css?v=`+v["fonts/fonts.css"]+`"`, 1)
		html = strings.Replace(html, `src="/player/runtime.js"`, `src="/player/runtime.js?v=`+v["player/runtime.js"]+`"`, 1)
		html = strings.Replace(html, `<script src="/player/icons.js"></script>`,
			`<script>window.__PLAYER_THREE__="/player/vendor/three.js?v=`+v["player/vendor/three.js"]+`";</script>`+playDataMarker+
				`<script src="/player/icons.js?v=`+v["player/icons.js"]+`"></script>`, 1)
		s.playAssets.shell = []byte(html)
		s.playAssets.versions = v
	})
	return s.playAssets.shell
}

func (s *FrontendServer) servePlay(c *gin.Context, id string) {
	shell := s.playShell()
	if s.player == nil || shell == nil {
		c.String(http.StatusNotFound, "Not found")
		c.Abort()
		return
	}
	data, ok := s.player.PlayerData(c.Request.Context(), id)
	if !ok {
		c.String(http.StatusNotFound, "Not found")
		c.Abort()
		return
	}
	// json.Marshal escapes <, > and &, so the data cannot close the script element.
	page := bytes.Replace(shell, []byte(playDataMarker), append(append([]byte("<script>window.__FILM_DATA__="), data...), []byte(";</script>")...), 1)
	sum := sha256.Sum256(page)
	etag := `"` + hex.EncodeToString(sum[:10]) + `"`
	setPlayerHeaders(c)
	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=300, stale-while-revalidate=604800")
	if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, etag) {
		c.Status(http.StatusNotModified)
		c.Abort()
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	c.Abort()
}

// setPlayerHeaders is the sandboxed player's policy: scripts may run but nothing may reach the network.
func setPlayerHeaders(c *gin.Context) {
	origin := requestOrigin(c.Request)
	c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline' 'unsafe-eval' "+origin+"; style-src 'unsafe-inline' "+origin+
		"; img-src data: blob: "+origin+"; font-src data: "+origin+"; media-src 'none'; connect-src 'none'; form-action 'none'; base-uri 'none'; frame-ancestors "+origin)
	c.Header("X-Frame-Options", "SAMEORIGIN")
}
