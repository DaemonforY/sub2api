//go:build !embed

// Package web provides embedded web assets for the application.
package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PublicSettingsProvider is an interface to fetch public settings
// This stub is needed for compilation when frontend is not embedded
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer is a stub for non-embed builds
type FrontendServer struct{}

// NewFrontendServer returns an error when frontend is not embedded
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	return nil, errors.New("frontend not embedded")
}

// SetVideoPlayer is a no-op for non-embed builds.
func (s *FrontendServer) SetVideoPlayer(VideoPlayerSource) {}

// SetVideoHost is a no-op for non-embed builds.
func (s *FrontendServer) SetVideoHost(string) {}

// SetIndexNowKey is a no-op for non-embed builds.
func (s *FrontendServer) SetIndexNowKey(string) {}

// IndexNowPages is empty for non-embed builds.
func (s *FrontendServer) IndexNowPages(context.Context) map[string]string { return nil }

// InvalidateCache is a no-op for non-embed builds
func (s *FrontendServer) InvalidateCache() {}

// Middleware returns a handler that returns 404 for non-embed builds
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Frontend not embedded. Build with -tags embed to include frontend.")
		c.Abort()
	}
}

func ServeEmbeddedFrontend() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Frontend not embedded. Build with -tags embed to include frontend.")
		c.Abort()
	}
}

func HasEmbeddedFrontend() bool {
	return false
}

// LearnLessonText: no built site without the embedded frontend.
func LearnLessonText(lessonID string) (string, string) { return lessonTextFrom(nil, lessonID) }

// LearnPages: no built site without the embedded frontend.
func LearnPages() []LearnPage { return learnPagesFrom(nil) }

// SetSEOContent is a no-op for non-embed builds.
func (s *FrontendServer) SetSEOContent(SEOContent) {}

// The SEO pages (seo.go) are only served by embed builds; this keeps them compiled and linted here.
var _ = []any{
	mainPageMeta, videoPageMeta, applyPageMeta, robotsTxt, sitemapXML, mainSitemapPages, videoSitemapPages,
	mainLLMsTxt, videoLLMsTxt, mainRobotsDisallow, videoRobotsDisallow,
}
