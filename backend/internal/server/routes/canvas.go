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
	devices := v1.Group("/user/canvas-sessions")
	devices.Use(gin.HandlerFunc(jwtAuth))
	devices.Use(middleware.BackendModeUserGuard(settingService))
	devices.Use(panelRateLimiter.Global())
	{
		devices.POST("", h.CanvasSession.Create)
		devices.GET("", h.CanvasSession.List)
		devices.DELETE("/:id", h.CanvasSession.Revoke)
	}
	if h.Community != nil {
		// Work images are plain public files (loaded by <img> from the canvas).
		v1.GET("/community/media/:file", h.Community.Media)
		// Share-card <head> tags for the canvas site's nginx (public content only, cached briefly).
		v1.GET("/community/share-meta/*path", func(c *gin.Context) { h.Community.ShareMeta(c, canvasOrigins) })
		// The main site's home page wall (public works, no viewer state).
		v1.GET("/community/works", panelRateLimiter.PublicIP(), h.Community.PublicWorks)
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

	if h.Community == nil {
		return
	}
	cm := h.Community
	// Public community pages: anyone may read; a signed-in viewer also sees their likes / follows.
	public := canvas.Group("/community")
	public.Use(middleware.CanvasSessionOptional(sessions))
	public.Use(panelRateLimiter.PublicIP())
	{
		public.GET("/works", cm.Works)
		public.GET("/works/:id", cm.Work)
		public.POST("/works/:id/remix", cm.Remix)
		public.POST("/works/:id/report", cm.Report)
		public.GET("/users/:handle", cm.Profile)
		public.GET("/users/:handle/collections", cm.Collections)
		public.GET("/users/:handle/followers", cm.Followers)
		public.GET("/users/:handle/following", cm.Following)
		public.GET("/collections/:id", cm.Collection)
	}
	mine := canvas.Group("/community")
	mine.Use(middleware.CanvasSessionAuth(sessions))
	mine.Use(middleware.BackendModeUserGuard(settingService))
	mine.Use(panelRateLimiter.Global())
	{
		mine.GET("/me", cm.Me)
		mine.PUT("/me/profile", cm.SaveProfile)
		mine.POST("/works", cm.Publish)
		mine.PUT("/works/:id", cm.UpdateWork)
		mine.DELETE("/works/:id", cm.DeleteWork)
		mine.PUT("/works/:id/like", cm.Like)
		mine.DELETE("/works/:id/like", cm.Like)
		mine.PUT("/works/:id/favorite", cm.Favorite)
		mine.DELETE("/works/:id/favorite", cm.Favorite)
		mine.GET("/works/:id/collections", cm.WorkCollections)
		mine.PUT("/users/:handle/follow", cm.Follow)
		mine.DELETE("/users/:handle/follow", cm.Follow)
		mine.POST("/collections", cm.CreateCollection)
		mine.PUT("/collections/:id", cm.UpdateCollection)
		mine.DELETE("/collections/:id", cm.DeleteCollection)
		mine.PUT("/collections/:id/works/:work_id", cm.CollectionItem)
		mine.DELETE("/collections/:id/works/:work_id", cm.CollectionItem)
		mine.GET("/notifications", cm.Notifications)
		mine.POST("/notifications/read", cm.ReadNotifications)
	}
}
