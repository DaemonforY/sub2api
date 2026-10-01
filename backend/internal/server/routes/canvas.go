package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterCanvasRoutes registers the canvas sign-in: the main site's popup creates a session
// (panel JWT) and lists / signs out devices; the canvas itself calls /canvas/* with the session
// cookie (credentials are allowed for those paths only, see middleware.CORS).
func RegisterCanvasRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth middleware.JWTAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter, canvasOrigins []string) {
	if h.CanvasSession == nil {
		return
	}
	sessions := h.CanvasSession.Sessions()
	mine := v1.Group("/user/canvas-sessions")
	mine.Use(gin.HandlerFunc(jwtAuth))
	mine.Use(middleware.BackendModeUserGuard(settingService))
	mine.Use(panelRateLimiter.Global())
	{
		mine.POST("", h.CanvasSession.Create)
		mine.GET("", h.CanvasSession.List)
		mine.DELETE("/:id", h.CanvasSession.Revoke)
	}

	canvas := v1.Group("/canvas")
	canvas.Use(middleware.CanvasOriginGuard(canvasOrigins))
	// Logout works with an expired or revoked cookie too (it still clears it).
	canvas.POST("/logout", panelRateLimiter.PublicIP(), h.CanvasSession.Logout)
	signedIn := canvas.Group("")
	signedIn.Use(middleware.CanvasSessionAuth(sessions))
	signedIn.Use(middleware.BackendModeUserGuard(settingService))
	signedIn.Use(panelRateLimiter.Global())
	{
		signedIn.GET("/me", h.CanvasSession.Me)
	}
}
