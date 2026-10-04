package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterLearnRoutes registers /learn's API: public config, the learner's progress and runs.
func RegisterLearnRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Learn == nil {
		return
	}
	v1.GET("/learn/config", panelRateLimiter.PublicIP(), h.Learn.Config)

	g := v1.Group("/learn")
	g.Use(gin.HandlerFunc(jwtAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.GET("/me", h.Learn.Me)
		g.POST("/progress", h.Learn.Progress)
		g.POST("/run", panelRateLimiter.Heavy(), h.Learn.Run)
	}
}
