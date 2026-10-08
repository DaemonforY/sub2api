package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterAnimationJobRoutes registers the canvas' background AI 动画 jobs. API-key authenticated
// like the gateway (the job calls the gateway with that key, which bills it) and reached
// cross-origin from canvas.<domain> via CORS_ALLOWED_ORIGINS.
func RegisterAnimationJobRoutes(v1 *gin.RouterGroup, h *handler.Handlers, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.AnimationJob == nil {
		return
	}
	g := v1.Group("/animation-jobs")
	g.Use(gin.HandlerFunc(apiKeyAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.POST("", h.AnimationJob.Create)
		g.GET("", h.AnimationJob.List)
		g.GET("/:id", h.AnimationJob.Get)
		g.POST("/:id/cancel", h.AnimationJob.Cancel)
	}
}
