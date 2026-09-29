package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterContestRoutes registers public contest pages (anonymous browsing, optional
// login to show the viewer's votes) and authenticated submit / vote actions.
func RegisterContestRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Contest == nil {
		return
	}
	v1.GET("/contest-images/:file", panelRateLimiter.PublicIP(), h.Contest.Image)

	public := v1.Group("/contests")
	public.Use(panelRateLimiter.PublicIP())
	public.Use(gin.HandlerFunc(optionalJWT))
	public.Use(middleware.BackendModeUserGuard(settingService))
	{
		public.GET("", h.Contest.List)
		public.GET("/:id", h.Contest.Get)
		public.GET("/:id/entries", h.Contest.ListEntries)
		public.GET("/:id/leaderboard", h.Contest.Leaderboard)
	}

	authed := v1.Group("/contests")
	authed.Use(gin.HandlerFunc(jwtAuth))
	authed.Use(middleware.BackendModeUserGuard(settingService))
	authed.Use(panelRateLimiter.Global())
	{
		authed.POST("/:id/entries", h.Contest.SubmitEntry)
		authed.DELETE("/:id/entries/:entryId", h.Contest.WithdrawEntry)
		authed.POST("/:id/entries/:entryId/vote", h.Contest.Vote)
		authed.DELETE("/:id/entries/:entryId/vote", h.Contest.Unvote)
	}
}
