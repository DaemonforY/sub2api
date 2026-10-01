package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterImageToolsRoutes registers the canvas' server-side image tools (background removal,
// super-resolution). API-key authenticated; the tools bill the balance themselves.
func RegisterImageToolsRoutes(v1 *gin.RouterGroup, h *handler.Handlers, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.ImageTools == nil {
		return
	}
	g := v1.Group("/image-tools")
	g.Use(gin.HandlerFunc(apiKeyAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.GET("/quota", h.ImageTools.Quota)
		g.POST("/remove-bg", h.ImageTools.RemoveBackground)
		g.POST("/upscale", h.ImageTools.Upscale)
	}
}
