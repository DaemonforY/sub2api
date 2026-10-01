package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterSiteRoutes registers static-site hosting: the signed-in user's sites (panel JWT), the
// public abuse report form and Caddy's on-demand TLS check. The sites themselves are served by
// the host middleware, not by these routes.
func RegisterSiteRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
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
		mine.POST("", h.SiteHosting.Create)
		mine.PUT("/:id", h.SiteHosting.Update)
		mine.DELETE("/:id", h.SiteHosting.Delete)
		mine.POST("/:id/renew", h.SiteHosting.Renew)
	}
}
