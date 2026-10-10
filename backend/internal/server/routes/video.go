package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterVideoRoutes registers HiveGPT 视频 (video.<domain>, same server, so same origin): the public
// gallery, and the signed-in user's projects with API-key auth (the agent bills that key).
func RegisterVideoRoutes(v1 *gin.RouterGroup, h *handler.Handlers, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.Video == nil {
		return
	}
	pub := v1.Group("/video")
	pub.Use(panelRateLimiter.PublicIP())
	{
		pub.GET("/catalog", h.Video.Catalog)
		pub.GET("/gallery", h.Video.Gallery)
		pub.GET("/works/:id", h.Video.Work)
		pub.GET("/works/:id/audio/:file", h.Video.WorkAudio)
		pub.GET("/works/:id/media/:file", h.Video.Media)
		pub.POST("/works/:id/view", h.Video.View)
		pub.GET("/models", h.Video.Models)
		pub.GET("/ads", h.Video.Ads)
		pub.GET("/ads/:file", h.Video.AdImage)
	}

	g := v1.Group("/video")
	g.Use(gin.HandlerFunc(apiKeyAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.GET("/me", h.Video.Me)
		g.GET("/projects", h.Video.List)
		g.POST("/projects", h.Video.Create)
		g.POST("/uploads", h.Video.Upload)
		g.POST("/projects/:id/poster", h.Video.SetPoster)
		g.GET("/projects/:id", h.Video.Get)
		g.DELETE("/projects/:id", h.Video.Delete)
		g.GET("/projects/:id/events", h.Video.Events)
		g.POST("/projects/:id/answer", h.Video.Answer)
		g.POST("/projects/:id/message", h.Video.Message)
		g.POST("/projects/:id/repair", h.Video.Repair)
		g.POST("/projects/:id/resume", h.Video.Resume)
		g.POST("/projects/:id/stop", h.Video.Stop)
		g.PUT("/projects/:id/script", h.Video.EditScript)
		g.POST("/projects/:id/voice", h.Video.SetVoice)
		g.GET("/projects/:id/versions", h.Video.Versions)
		g.POST("/projects/:id/versions/:vid/restore", h.Video.Restore)
		g.POST("/projects/:id/publish", h.Video.Publish)
		g.POST("/projects/:id/unpublish", h.Video.Unpublish)
		g.GET("/projects/:id/audio/:file", h.Video.OwnAudio)
		g.GET("/admin/pending", h.Video.Pending)
		g.GET("/admin/works/:id", h.Video.AdminWork)
		g.POST("/admin/works/:id/review", h.Video.Review)
		g.PUT("/admin/works/:id", h.Video.AdminEdit)
		g.POST("/admin/works/:id/poster", h.Video.AdminSetPoster)
		g.POST("/admin/import", h.Video.Import)
		g.GET("/admin/settings", h.Video.AdminSettings)
		g.PUT("/admin/settings", h.Video.SaveSettings)
		g.POST("/admin/ads/image", h.Video.UploadAdImage)
	}
}
