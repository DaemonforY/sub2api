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

	// Creator center: courses reviewed by an admin before they go live; sales and withdrawals.
	creator := v1.Group("/user/creator")
	creator.Use(gin.HandlerFunc(jwtAuth))
	creator.Use(middleware.BackendModeUserGuard(settingService))
	creator.Use(panelRateLimiter.Global())
	{
		creator.GET("", h.Course.CreatorHome)
		creator.POST("/apply", h.Course.CreatorApply)
		creator.POST("/images", h.Course.CreatorImage)
		creator.GET("/courses", h.Course.CreatorCourses)
		creator.POST("/courses", h.Course.CreatorCreate)
		creator.GET("/courses/:id", h.Course.CreatorGet)
		creator.PUT("/courses/:id", h.Course.CreatorUpdate)
		creator.DELETE("/courses/:id", h.Course.CreatorDelete)
		creator.POST("/courses/:id/cover", h.Course.CreatorCover)
		creator.POST("/courses/:id/submit", h.Course.CreatorSubmit)
		creator.POST("/courses/:id/sale", h.Course.CreatorSale)
		creator.GET("/courses/:id/deliveries", h.Course.CreatorDeliveries)
		creator.POST("/courses/:id/deliveries", h.Course.CreatorSaveDelivery)
		creator.GET("/courses/:id/students", h.Course.CreatorStudents)
		creator.GET("/sales", h.Course.CreatorSales)
		creator.POST("/withdrawals", h.Course.CreatorWithdraw)
		creator.POST("/withdrawals/:id/cancel", h.Course.CreatorCancelWithdraw)
	}
}
