package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterCourseRoutes registers the paid course pages (anonymous browsing, optional login to mark
// owned courses) and the buyer's courses with their netdisk delivery.
func RegisterCourseRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	optionalJWT middleware.OptionalJWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Course == nil {
		return
	}
	v1.GET("/courses/media/:file", panelRateLimiter.PublicIP(), h.Course.Media)

	public := v1.Group("/courses")
	public.Use(panelRateLimiter.PublicIP())
	public.Use(gin.HandlerFunc(optionalJWT))
	public.Use(middleware.BackendModeUserGuard(settingService))
	{
		public.GET("", h.Course.List)
		public.GET("/:slug", h.Course.Get)
	}

	mine := v1.Group("/user/courses")
	mine.Use(gin.HandlerFunc(jwtAuth))
	mine.Use(middleware.BackendModeUserGuard(settingService))
	mine.Use(panelRateLimiter.Global())
	{
		mine.GET("", h.Course.Mine)
		mine.GET("/:id/delivery", h.Course.Delivery)
	}
}
