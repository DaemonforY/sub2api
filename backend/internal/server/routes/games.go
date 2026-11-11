package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGamesRoutes registers the 小游戏 APIs: 「AI 万物合成」 is open to visitors (limited per IP,
// per user when signed in, and site-wide per day in the service).
func RegisterGamesRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.GameCraft == nil {
		return
	}
	g := v1.Group("/games/craft")
	g.Use(panelRateLimiter.PublicIP())
	g.Use(gin.HandlerFunc(optionalJWT))
	g.Use(middleware.BackendModeUserGuard(settingService))
	{
		g.GET("/start", h.GameCraft.Start)
		g.POST("/combine", h.GameCraft.Combine)
	}
}
