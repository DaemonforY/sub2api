package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGrowthRoutes registers the public growth-program config (invitee bonus, education
// discount, leaderboard switches) read by the register, referral and purchase pages.
func RegisterGrowthRoutes(v1 *gin.RouterGroup, h *handler.Handlers, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.Growth == nil {
		return
	}
	public := v1.Group("/growth")
	public.Use(panelRateLimiter.PublicIP())
	public.Use(middleware.BackendModeUserGuard(settingService))
	public.GET("/config", h.Growth.PublicConfig)
}
