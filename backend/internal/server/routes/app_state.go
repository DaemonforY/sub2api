package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterAppStateRoutes registers the synced per-user documents of companion apps. The canvas
// (canvas.<domain>) holds the user's API key but no web session, so these are API-key authenticated
// (same checks as gateway calls, minus billing) and reached cross-origin via CORS_ALLOWED_ORIGINS.
func RegisterAppStateRoutes(v1 *gin.RouterGroup, h *handler.Handlers, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.AppState == nil {
		return
	}
	group := v1.Group("/app-state")
	group.Use(gin.HandlerFunc(apiKeyAuth))
	group.Use(middleware.BackendModeUserGuard(settingService))
	group.Use(panelRateLimiter.Global())
	group.POST("/blobs", h.AppState.UploadBlob)
	group.GET("/blobs/:id", h.AppState.GetBlob)
	group.GET("/:namespace", h.AppState.Get)
	group.PUT("/:namespace", h.AppState.Put)
}
