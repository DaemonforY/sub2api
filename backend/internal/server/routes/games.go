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
	// 联机双排 relay shared by the co-op games (no login; capped per IP and site-wide in the handler).
	rooms := handler.NewGameRoomHandler()
	v1.GET("/games/rooms/ws", panelRateLimiter.PublicIP(), rooms.ServeWS)

	// 小游戏可选登录: cloud saves and scores need a signed-in user (checked in the handler); the
	// leaderboard is public and marks the viewer's own row when signed in.
	if h.GameProgress != nil {
		p := v1.Group("/games/progress/:game")
		p.Use(panelRateLimiter.PublicIP())
		p.Use(gin.HandlerFunc(optionalJWT))
		p.Use(middleware.BackendModeUserGuard(settingService))
		{
			p.GET("/save", h.GameProgress.GetSave)
			p.PUT("/save", h.GameProgress.PutSave)
			p.POST("/score", h.GameProgress.SubmitScore)
			p.GET("/leaderboard", h.GameProgress.Leaderboard)
		}
	}

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
