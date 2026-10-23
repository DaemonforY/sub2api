package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterChatRoutes registers AI 对话 (/chat): conversations kept on the server and model calls on
// the user's own keys. All of it needs login.
func RegisterChatRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Chat == nil {
		return
	}
	g := v1.Group("/chat")
	g.Use(gin.HandlerFunc(jwtAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.GET("/keys", h.Chat.Keys)
		g.GET("/models", h.Chat.Models)
		g.GET("/conversations", h.Chat.List)
		g.POST("/conversations", h.Chat.Create)
		g.GET("/conversations/:id", h.Chat.Get)
		g.PUT("/conversations/:id", h.Chat.Update)
		g.DELETE("/conversations/:id", h.Chat.Delete)
		g.POST("/conversations/:id/send", panelRateLimiter.Heavy(), h.Chat.Send)
		g.POST("/files", panelRateLimiter.Heavy(), h.Chat.Upload)
		g.GET("/files/:name", h.Chat.File)
	}
}
