package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterTutorRoutes registers AI 助教: the teacher's side (/tutors, login) and the students' side
// (/t/:code, no account — a class password and a name).
func RegisterTutorRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h.Tutor == nil {
		return
	}
	t := v1.Group("/tutors")
	t.Use(gin.HandlerFunc(jwtAuth))
	t.Use(middleware.BackendModeUserGuard(settingService))
	t.Use(panelRateLimiter.Global())
	{
		t.GET("/templates", h.Tutor.Templates)
		t.GET("", h.Tutor.List)
		t.POST("", h.Tutor.Create)
		t.POST("/draft", panelRateLimiter.Heavy(), h.Tutor.Draft)
		t.GET("/:id", h.Tutor.Get)
		t.PUT("/:id", h.Tutor.Update)
		t.DELETE("/:id", h.Tutor.Delete)
		t.POST("/:id/materials", panelRateLimiter.Heavy(), h.Tutor.AddMaterial)
		t.DELETE("/:id/materials/:mid", h.Tutor.DeleteMaterial)
		t.POST("/:id/preview", panelRateLimiter.Heavy(), h.Tutor.Preview)
		t.GET("/:id/stats", h.Tutor.Stats)
		t.POST("/:id/insights", panelRateLimiter.Heavy(), h.Tutor.Insights)
	}

	s := v1.Group("/t/:code")
	s.Use(panelRateLimiter.PublicIP())
	s.Use(middleware.BackendModeUserGuard(settingService))
	{
		s.GET("", h.Tutor.Public)
		s.POST("/join", h.Tutor.Join)
		s.POST("/chat", h.Tutor.Chat)
	}
}
