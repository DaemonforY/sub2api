package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterPromptLibraryRoutes registers the canvas prompt library. Browsing is anonymous (the canvas
// works before the user connects a key); usage events, recommendations and the user's own prompts
// are API-key authenticated like the other companion-app endpoints (no billing checks).
func RegisterPromptLibraryRoutes(v1 *gin.RouterGroup, h *handler.Handlers, apiKeyAuth middleware.APIKeyAuthMiddleware, settingService *service.SettingService, panelRateLimiter *middleware.PanelRateLimiter) {
	if h.PromptLibrary == nil {
		return
	}
	public := v1.Group("/prompt-library")
	public.Use(panelRateLimiter.PublicIP())
	public.Use(middleware.BackendModeUserGuard(settingService))
	{
		public.GET("/items", h.PromptLibrary.List)
		public.GET("/covers/:file", h.PromptLibrary.Cover)
	}

	keyed := v1.Group("/prompt-library")
	keyed.Use(gin.HandlerFunc(apiKeyAuth))
	keyed.Use(middleware.BackendModeUserGuard(settingService))
	keyed.Use(panelRateLimiter.Global())
	{
		keyed.GET("/recommendations", h.PromptLibrary.Recommendations)
		keyed.POST("/items/:id/use", h.PromptLibrary.Use)
		keyed.PUT("/items/:id/favorite", h.PromptLibrary.Favorite)
		keyed.GET("/mine", h.PromptLibrary.ListMine)
		keyed.POST("/mine", h.PromptLibrary.CreateMine)
		keyed.PUT("/mine/:id", h.PromptLibrary.UpdateMine)
		keyed.DELETE("/mine/:id", h.PromptLibrary.DeleteMine)
		keyed.POST("/covers", h.PromptLibrary.UploadCover)
	}
}
