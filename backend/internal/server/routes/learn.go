package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterLearnRoutes registers /learn's API: public config / quiz questions / checkpoints /
// certificates, and the learner's progress, runs, quizzes, certificates, tutor and interviews.
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
	v1.GET("/learn/quiz/:lesson", panelRateLimiter.PublicIP(), h.Learn.Quiz)
	v1.GET("/learn/checkpoints/:id", panelRateLimiter.PublicIP(), h.Learn.Checkpoint)
	v1.GET("/learn/cert/:code", panelRateLimiter.PublicIP(), h.Learn.Certificate)
	v1.GET("/learn/showcase", panelRateLimiter.PublicIP(), h.Learn.Showcase)

	g := v1.Group("/learn")
	g.Use(gin.HandlerFunc(jwtAuth))
	g.Use(middleware.BackendModeUserGuard(settingService))
	g.Use(panelRateLimiter.Global())
	{
		g.GET("/me", h.Learn.Me)
		g.POST("/progress", h.Learn.Progress)
		g.POST("/run", panelRateLimiter.Heavy(), h.Learn.Run)
		g.GET("/keys", h.Learn.Keys)
		g.POST("/quiz/:lesson", h.Learn.SubmitQuiz)
		g.POST("/checkpoints/:id/verify", h.Learn.VerifyCheckpoint)
		g.GET("/certificates/:track", h.Learn.CertStatus)
		g.POST("/certificates/:track", h.Learn.ClaimCert)
		g.PUT("/certificates/:track/showcase", h.Learn.SetShowcase)
		g.POST("/tutor", panelRateLimiter.Heavy(), h.Learn.Tutor)
		g.GET("/interviews", h.Learn.Interviews)
		g.POST("/interviews", panelRateLimiter.Heavy(), h.Learn.StartInterview)
		g.GET("/interviews/:id", h.Learn.Interview)
		g.POST("/interviews/:id/answer", panelRateLimiter.Heavy(), h.Learn.AnswerInterview)
	}
}
