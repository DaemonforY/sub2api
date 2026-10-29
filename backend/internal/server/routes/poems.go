package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPoemsRoutes registers 诗词灵感 for users: poem search and pages open to visitors (limited per
// IP, per user when signed in), 按画面找 for signed-in users (free runs, then billed), and the corpus
// half of 按画面找 for the canvas 诗画 (which runs the model on the user's own key). No song data.
func RegisterPoemsRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Inspiration == nil {
		return
	}
	g := v1.Group("/poems")
	g.Use(panelRateLimiter.PublicIP())
	g.Use(gin.HandlerFunc(optionalJWT))
	g.Use(middleware.BackendModeUserGuard(settingService))
	{
		g.GET("/config", h.Inspiration.PublicConfig)
		g.GET("", h.Inspiration.PublicSearch)
		g.GET("/candidates", h.Inspiration.PublicCandidates)
		g.GET("/:id", h.Inspiration.PublicPoem)
		g.POST("/find", panelRateLimiter.Heavy(), h.Inspiration.PublicFind)
	}
}
