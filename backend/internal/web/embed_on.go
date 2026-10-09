//go:build embed

package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	htmlpkg "html"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// NonceHTMLPlaceholder is the placeholder for nonce in HTML script tags
	NonceHTMLPlaceholder = "__CSP_NONCE_VALUE__"
)

//go:embed all:dist
var frontendFS embed.FS

// PublicSettingsProvider is an interface to fetch public settings
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the embedded frontend with settings injection
type FrontendServer struct {
	distFS      fs.FS
	fileServer  http.Handler
	baseHTML    []byte
	cache       *HTMLCache
	settings    PublicSettingsProvider
	overrideDir string // local file override directory
	videoHost   string // HiveGPT 视频 is served for this host (e.g. video.hivegpt.cn)

	seo        SEOContent // dynamic public pages for meta tags and the sitemap; may be nil
	site       atomic.Pointer[seoSiteInfo]
	videoHTML  []byte    // dist/video/index.html, nil when the video app is not built
	videoTools []SEOTool // dist/video/seo-tools.json
	learnOnce  sync.Once
	learn      []SEOLearnPage
	textCache  sync.Map // robots / sitemap / llms bodies: key -> seoCachedText
}

// seoSiteInfo is the site name and logo from the public settings.
type seoSiteInfo struct {
	name string
	logo string
}

type seoCachedText struct {
	body    []byte
	expires time.Time
}

const seoTextTTL = 10 * time.Minute

// SetSEOContent supplies the courses, gallery works and FAQ for meta tags, sitemaps and llms.txt.
func (s *FrontendServer) SetSEOContent(content SEOContent) {
	s.seo = content
}

// SetVideoHost makes requests for host (without port) serve the HiveGPT 视频 app (dist/video)
// instead of the main site. Empty disables it.
func (s *FrontendServer) SetVideoHost(host string) {
	s.videoHost = strings.ToLower(strings.TrimSpace(host))
}

// NewFrontendServer creates a new frontend server with settings injection
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil, err
	}

	// Read base HTML once
	file, err := distFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	baseHTML, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	s := &FrontendServer{
		distFS:      distFS,
		fileServer:  http.FileServer(http.FS(distFS)),
		baseHTML:    baseHTML,
		cache:       cache,
		settings:    settingsProvider,
		overrideDir: filepath.Join("data", "public"),
	}
	if raw, err := fs.ReadFile(distFS, "video/index.html"); err == nil {
		s.videoHTML = raw
	}
	if raw, err := fs.ReadFile(distFS, "video/seo-tools.json"); err == nil {
		_ = json.Unmarshal(raw, &s.videoTools)
	}
	return s, nil
}

// InvalidateCache invalidates the HTML cache (call when settings change)
func (s *FrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
	if s != nil {
		s.site.Store(nil)
		s.textCache.Range(func(k, _ any) bool { s.textCache.Delete(k); return true })
	}
}

// Middleware returns the Gin middleware handler
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes
		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		video := s.videoHost != "" && requestHost(c.Request) == s.videoHost
		if s.serveSEOText(c, cleanPath, video) {
			return
		}
		if video {
			s.serveVideo(c, cleanPath)
			return
		}

		// /learn is a separate static site (VitePress), not part of the SPA.
		if cleanPath == "learn" || strings.HasPrefix(cleanPath, "learn/") {
			s.serveLearn(c, cleanPath)
			return
		}
		// /editor is the 公众号 editor (its own Vite app).
		if cleanPath == "editor" || strings.HasPrefix(cleanPath, "editor/") {
			s.serveEditor(c, cleanPath)
			return
		}

		// For index.html or SPA routes, serve with injected settings
		if cleanPath == "index.html" || !s.fileExists(cleanPath) {
			s.serveIndexHTML(c)
			return
		}

		// Try local override first
		if s.tryServeOverride(c, cleanPath) {
			return
		}

		// Serve static files normally (hashed assets get long-lived cache headers)
		applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
		s.fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// serveLearn serves the /learn static site: pages (also without ".html"), its assets (hashed
// names under learn/assets/ are cached for good), and its own 404 page.
func (s *FrontendServer) serveLearn(c *gin.Context, cleanPath string) {
	if cleanPath == "learn" {
		c.Redirect(http.StatusMovedPermanently, "/learn/")
		c.Abort()
		return
	}
	name := cleanPath
	if strings.HasSuffix(name, "/") {
		name += "index.html"
	}
	candidates := []string{name}
	if path.Ext(name) == "" {
		candidates = append(candidates, name+".html", name+"/index.html")
	}
	for _, candidate := range candidates {
		if !s.isFile(candidate) {
			continue
		}
		if strings.HasPrefix(candidate, "learn/assets/") {
			c.Header("Cache-Control", staticAssetsCacheControl)
		} else if strings.HasSuffix(candidate, ".html") {
			c.Header("Cache-Control", "no-cache")
		}
		http.ServeFileFS(c.Writer, c.Request, s.distFS, candidate)
		c.Abort()
		return
	}
	body, err := fs.ReadFile(s.distFS, "learn/404.html")
	if err != nil {
		c.String(http.StatusNotFound, "Not found")
	} else {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", body)
	}
	c.Abort()
}

// serveEditor serves the /editor single-page app: its files (hashed assets cached for good) and
// index.html for everything else.
func (s *FrontendServer) serveEditor(c *gin.Context, cleanPath string) {
	if cleanPath == "editor" {
		c.Redirect(http.StatusMovedPermanently, "/editor/")
		c.Abort()
		return
	}
	name := cleanPath
	if strings.HasSuffix(name, "/") || !s.isFile(name) {
		name = "editor/index.html"
	}
	if !s.isFile(name) {
		c.String(http.StatusNotFound, "Not found")
		c.Abort()
		return
	}
	if strings.HasPrefix(name, "editor/assets/") {
		c.Header("Cache-Control", staticAssetsCacheControl)
	} else if strings.HasSuffix(name, ".html") {
		c.Header("Cache-Control", "no-cache")
	}
	http.ServeFileFS(c.Writer, c.Request, s.distFS, name)
	c.Abort()
}

// serveVideo serves the HiveGPT 视频 single-page app on its own host. player.html runs AI-written
// scene code inside a sandboxed iframe, so it gets its own policy: scripts may run but nothing may
// reach the network.
func (s *FrontendServer) serveVideo(c *gin.Context, cleanPath string) {
	name := "video/" + cleanPath
	if cleanPath == "index.html" || !s.isFile(name) {
		name = "video/index.html"
	}
	if !s.isFile(name) {
		c.String(http.StatusNotFound, "Not found")
		c.Abort()
		return
	}
	switch {
	case name == "video/index.html" && s.videoHTML != nil:
		site := s.seoSite(c, true)
		meta := videoPageMeta(c.Request.Context(), site, s.seo, s.videoTools, c.Request.URL.Path)
		c.Header("Cache-Control", "no-cache")
		if !meta.Index {
			c.Header("X-Robots-Tag", "noindex")
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", applyPageMeta(s.videoHTML, site, meta, middleware.GetNonceFromContext(c)))
		c.Abort()
		return
	case name == "video/player.html":
		origin := requestOrigin(c.Request)
		c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline' 'unsafe-eval' "+origin+"; style-src 'unsafe-inline' "+origin+
			"; img-src data: blob: "+origin+"; font-src data: "+origin+"; media-src 'none'; connect-src 'none'; form-action 'none'; base-uri 'none'; frame-ancestors "+origin)
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("Cache-Control", "no-cache")
	case strings.HasPrefix(name, "video/fonts/"):
		// Loaded from the sandboxed player (an opaque origin), so fonts need CORS.
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Cache-Control", staticAssetsCacheControl)
	case strings.HasPrefix(name, "video/assets/"):
		c.Header("Cache-Control", staticAssetsCacheControl)
	case strings.HasSuffix(name, ".html"):
		c.Header("Cache-Control", "no-cache")
	}
	http.ServeFileFS(c.Writer, c.Request, s.distFS, name)
	c.Abort()
}

// requestHost is the request's host without port, lower-cased.
func requestHost(r *http.Request) string {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.ToLower(host)
}

// requestOrigin is the browser-visible origin of the request (TLS ends at the proxy).
func requestOrigin(r *http.Request) string {
	scheme := "http"
	if proto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])); proto == "https" || proto == "http" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (s *FrontendServer) isFile(name string) bool {
	info, err := fs.Stat(s.distFS, name)
	return err == nil && !info.IsDir()
}

func (s *FrontendServer) fileExists(path string) bool {
	file, err := s.distFS.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

// tryServeOverride checks if a local override file exists and serves it.
// Files in overrideDir take precedence over embedded files.
func (s *FrontendServer) tryServeOverride(c *gin.Context, cleanPath string) bool {
	if s.overrideDir == "" {
		return false
	}
	filePath := filepath.Join(s.overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func (s *FrontendServer) serveIndexHTML(c *gin.Context) {
	// Get nonce from context (generated by SecurityHeaders middleware)
	nonce := middleware.GetNonceFromContext(c)

	rendered, ok := s.renderedIndex(c.Request.Context())
	if !ok {
		// Fallback: serve without injection
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	// Per-page title, meta tags and text for crawlers; the ETag covers them (but not the nonce).
	site := s.seoSite(c, false)
	meta := mainPageMeta(c.Request.Context(), site, s.seo, c.Request.URL.Path)
	page := applyPageMeta(rendered.Content, site, meta, NonceHTMLPlaceholder)
	pageHash := sha256.Sum256(page)
	etag := strings.TrimSuffix(rendered.ETag, `"`) + "-" + hex.EncodeToString(pageHash[:6]) + `"`

	if !meta.Index {
		c.Header("X-Robots-Tag", "noindex")
	}
	// Check If-None-Match for 304 response
	if match := c.GetHeader("If-None-Match"); match == etag {
		c.Status(http.StatusNotModified)
		c.Abort()
		return
	}

	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache") // Must revalidate
	c.Data(http.StatusOK, "text/html; charset=utf-8", replaceNoncePlaceholder(page, nonce))
	c.Abort()
}

// renderedIndex is index.html with the public settings injected (cached until settings change).
func (s *FrontendServer) renderedIndex(ctx context.Context) (*CachedHTML, bool) {
	if cached := s.cache.Get(); cached != nil {
		return cached, true
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		return nil, false
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, false
	}
	s.cache.Set(s.injectSettings(settingsJSON), settingsJSON)
	cached := s.cache.Get()
	return cached, cached != nil
}

// seoSite is the host's name, origin and logo for meta tags.
func (s *FrontendServer) seoSite(c *gin.Context, video bool) seoSite {
	info := s.site.Load()
	if info == nil {
		_, _ = s.renderedIndex(c.Request.Context()) // rendering records the site name and logo
		if info = s.site.Load(); info == nil {
			info = &seoSiteInfo{name: "HiveGPT"}
		}
	}
	site := seoSite{Name: info.name, Origin: requestOrigin(c.Request), Logo: info.logo}
	if site.Logo == "" && !video {
		site.Logo = "/apple-touch-icon.png"
	}
	site.Logo = site.abs(site.Logo)
	if video {
		site.Name += " 视频"
	}
	return site
}

// storeSiteInfo keeps the site name and logo of the public settings for meta tags.
func (s *FrontendServer) storeSiteInfo(settingsJSON []byte) {
	var cfg struct {
		SiteName string `json:"site_name"`
		SiteLogo string `json:"site_logo"`
	}
	info := &seoSiteInfo{name: "HiveGPT"}
	if json.Unmarshal(settingsJSON, &cfg) == nil {
		if name := strings.TrimSpace(cfg.SiteName); name != "" {
			info.name = name
		}
		if logo := safeImageURL(cfg.SiteLogo); logo != "" && !strings.HasPrefix(strings.ToLower(logo), "data:") {
			info.logo = logo
		}
	}
	s.site.Store(info)
}

func (s *FrontendServer) learnPages() []SEOLearnPage {
	s.learnOnce.Do(func() {
		for _, p := range learnPagesFrom(s.distFS) {
			s.learn = append(s.learn, SEOLearnPage(p))
		}
	})
	return s.learn
}

// serveSEOText answers /robots.txt, /sitemap.xml, /llms.txt and /llms-full.txt for the host.
func (s *FrontendServer) serveSEOText(c *gin.Context, cleanPath string, video bool) bool {
	var contentType string
	switch cleanPath {
	case "robots.txt", "llms.txt", "llms-full.txt":
		contentType = "text/plain; charset=utf-8"
	case "sitemap.xml":
		contentType = "application/xml; charset=utf-8"
	default:
		return false
	}
	if video && cleanPath == "llms-full.txt" {
		return false
	}
	site := s.seoSite(c, video)
	key := site.Origin + "|" + cleanPath
	if v, ok := s.textCache.Load(key); ok {
		if cached := v.(seoCachedText); time.Now().Before(cached.expires) {
			s.writeSEOText(c, contentType, cached.body)
			return true
		}
	}
	ctx := c.Request.Context()
	var body []byte
	switch {
	case cleanPath == "robots.txt" && video:
		body = []byte(robotsTxt(site, videoRobotsDisallow, []string{"/sitemap.xml"}))
	case cleanPath == "robots.txt":
		sitemaps := []string{"/sitemap.xml"}
		if s.isFile("learn/sitemap.xml") {
			sitemaps = append(sitemaps, "/learn/sitemap.xml")
		}
		body = []byte(robotsTxt(site, mainRobotsDisallow, sitemaps))
	case cleanPath == "sitemap.xml" && video:
		body = sitemapXML(site, videoSitemapPages(ctx, s.seo, s.videoTools))
	case cleanPath == "sitemap.xml":
		body = sitemapXML(site, mainSitemapPages(ctx, s.seo))
	case video:
		body = []byte(videoLLMsTxt(ctx, site, s.seo, s.videoTools))
	default:
		body = []byte(mainLLMsTxt(ctx, site, s.seo, s.learnPages(), cleanPath == "llms-full.txt"))
	}
	s.textCache.Store(key, seoCachedText{body: body, expires: time.Now().Add(seoTextTTL)})
	s.writeSEOText(c, contentType, body)
	return true
}

func (s *FrontendServer) writeSEOText(c *gin.Context, contentType string, body []byte) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, contentType, body)
	c.Abort()
}

func (s *FrontendServer) injectSettings(settingsJSON []byte) []byte {
	// Create the script tag to inject with nonce placeholder
	// The placeholder will be replaced with actual nonce at request time
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)

	// Inject before </head>
	headClose := []byte("</head>")
	result := bytes.Replace(s.baseHTML, headClose, append(script, headClose...), 1)

	s.storeSiteInfo(settingsJSON)

	// Apply custom branding before the browser paints the static defaults.
	result = injectSiteTitle(result, settingsJSON)
	result = injectSiteFavicon(result, settingsJSON)

	return result
}

// injectSiteFavicon replaces the static favicon with a configured, browser-safe image URL.
func injectSiteFavicon(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteLogo string `json:"site_logo"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil {
		return html
	}

	logoURL := safeImageURL(cfg.SiteLogo)
	if logoURL == "" {
		return html
	}

	linkStart := bytes.Index(html, []byte(`<link rel="icon"`))
	if linkStart == -1 {
		return html
	}
	linkEndOffset := bytes.IndexByte(html[linkStart:], '>')
	if linkEndOffset == -1 {
		return html
	}
	linkEnd := linkStart + linkEndOffset + 1
	replacement := []byte(`<link rel="icon" href="` + htmlpkg.EscapeString(logoURL) + `" />`)

	var buf bytes.Buffer
	buf.Write(html[:linkStart])
	buf.Write(replacement)
	buf.Write(html[linkEnd:])
	return buf.Bytes()
}

func safeImageURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return trimmed
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "data:image/") {
		return trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ""
	}
	return trimmed
}

// injectSiteTitle replaces the static <title> in HTML with the configured site name.
// This ensures the browser tab shows the correct title before JS executes.
func injectSiteTitle(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil || cfg.SiteName == "" {
		return html
	}

	// Find and replace the existing <title>...</title>
	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	newTitle := []byte("<title>" + htmlpkg.EscapeString(cfg.SiteName) + " - AI API Gateway</title>")
	var buf bytes.Buffer
	buf.Write(html[:titleStart])
	buf.Write(newTitle)
	buf.Write(html[titleEnd+len("</title>"):])
	return buf.Bytes()
}

// replaceNoncePlaceholder replaces the nonce placeholder with actual nonce value
func replaceNoncePlaceholder(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(NonceHTMLPlaceholder), []byte(nonce))
}

// ServeEmbeddedFrontend returns a middleware for serving embedded frontend
// This is the legacy function for backward compatibility when no settings provider is available
func ServeEmbeddedFrontend() gin.HandlerFunc {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))
	overrideDir := filepath.Join("data", "public")

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			// Try local override first
			if tryServeOverrideFile(c, overrideDir, cleanPath) {
				return
			}
			applyStaticAssetCacheHeaders(c.Writer.Header(), cleanPath)
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		serveIndexHTML(c, distFS)
	}
}

// tryServeOverrideFile is a standalone version of tryServeOverride for legacy usage.
func tryServeOverrideFile(c *gin.Context, overrideDir, cleanPath string) bool {
	if overrideDir == "" {
		return false
	}
	filePath := filepath.Join(overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func shouldBypassEmbeddedFrontend(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/api/") ||
		strings.HasPrefix(trimmed, "/v1/") ||
		strings.HasPrefix(trimmed, "/v1beta/") ||
		strings.HasPrefix(trimmed, "/backend-api/") ||
		strings.HasPrefix(trimmed, "/antigravity/") ||
		strings.HasPrefix(trimmed, "/setup/") ||
		trimmed == "/health" ||
		strings.HasPrefix(trimmed, "/go/") ||
		trimmed == "/models" ||
		trimmed == "/responses" ||
		strings.HasPrefix(trimmed, "/responses/") ||
		trimmed == "/alpha/search" ||
		strings.HasPrefix(trimmed, "/images/") ||
		strings.HasPrefix(trimmed, "/videos/")
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html")
		c.Abort()
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func HasEmbeddedFrontend() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}

// LearnLessonText is a built lesson page's title and text (for the AI tutor).
func LearnLessonText(lessonID string) (string, string) {
	sub, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return "", ""
	}
	return lessonTextFrom(sub, lessonID)
}

// LearnPages is every built page of the learning site as text (for the support assistant).
func LearnPages() []LearnPage {
	sub, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil
	}
	return learnPagesFrom(sub)
}
