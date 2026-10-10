package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterSiteRoutes registers static-site hosting: the signed-in user's sites (panel JWT), the
// API-key publishing API, the public abuse report form and Caddy's on-demand TLS check. The sites themselves are served by
// the host middleware, not by these routes.
func RegisterSiteRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.SiteHosting == nil {
		return
	}
	v1.GET("/sites/tls-check", h.SiteHosting.TLSCheck)
	public := v1.Group("")
	public.Use(panelRateLimiter.PublicIP())
	public.POST("/site-reports", h.SiteHosting.Report)

	mine := v1.Group("/sites")
	mine.Use(gin.HandlerFunc(jwtAuth))
	mine.Use(middleware.BackendModeUserGuard(settingService))
	mine.Use(panelRateLimiter.Global())
	{
		mine.GET("", h.SiteHosting.Mine)
		mine.GET("/name-check", h.SiteHosting.CheckName)
		mine.POST("", h.SiteHosting.Create)
		mine.PUT("/:id", h.SiteHosting.Update)
		mine.DELETE("/:id", h.SiteHosting.Delete)
		mine.POST("/:id/renew", h.SiteHosting.Renew)
		mine.GET("/:id/versions", h.SiteHosting.Versions)
		mine.POST("/:id/rollback", h.SiteHosting.Rollback)
		mine.PUT("/:id/password", h.SiteHosting.SetPassword)
		mine.PUT("/:id/name", h.SiteHosting.Rename)
		mine.GET("/:id/stats", h.SiteHosting.Stats)
	}

	// AI 建站: generation is billed to the user's own key; publishing goes through site hosting.
	if h.SiteBuilder != nil {
		drafts := v1.Group("/site-drafts")
		drafts.Use(gin.HandlerFunc(jwtAuth))
		drafts.Use(middleware.BackendModeUserGuard(settingService))
		drafts.Use(panelRateLimiter.Global())
		drafts.GET("/config", h.SiteBuilder.Config)
		drafts.GET("", h.SiteBuilder.List)
		drafts.POST("", panelRateLimiter.Heavy(), h.SiteBuilder.Create)
		drafts.GET("/:id", h.SiteBuilder.Get)
		drafts.POST("/:id/revise", panelRateLimiter.Heavy(), h.SiteBuilder.Revise)
		drafts.POST("/:id/retry", panelRateLimiter.Heavy(), h.SiteBuilder.Retry)
		drafts.POST("/:id/undo", h.SiteBuilder.Undo)
		drafts.POST("/:id/cancel", h.SiteBuilder.Cancel)
		drafts.POST("/:id/publish", panelRateLimiter.Heavy(), h.SiteBuilder.Publish)
		drafts.DELETE("/:id", h.SiteBuilder.Delete)
		drafts.GET("/:id/images/:n", h.SiteBuilder.Image)
	}

	// Open API for scripts, agents and the canvas ("发布为网页"), authenticated with an API key.
	keyed := v1.Group("/hosting/sites")
	keyed.Use(gin.HandlerFunc(apiKeyAuth))
	keyed.Use(middleware.BackendModeUserGuard(settingService))
	keyed.Use(panelRateLimiter.Global())
	{
		keyed.GET("", h.SiteHosting.KeyList)
		keyed.POST("", h.SiteHosting.KeyCreate)
		keyed.PUT("/:id", h.SiteHosting.KeyUpdate)
	}
}
