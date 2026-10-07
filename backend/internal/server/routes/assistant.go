package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterAssistantRoutes registers the homepage support assistant: open to visitors (counted by IP),
// with optional login (signed-in users get their own daily allowance).
func RegisterAssistantRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Assistant == nil {
		return
	}
	g := v1.Group("/assistant")
	g.Use(panelRateLimiter.PublicIP())
	g.Use(gin.HandlerFunc(optionalJWT))
	g.Use(middleware.BackendModeUserGuard(settingService))
	{
		g.GET("/config", h.Assistant.Config)
		g.POST("/chat", h.Assistant.Chat)
	}
}
