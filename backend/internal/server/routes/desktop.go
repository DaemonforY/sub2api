package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterDesktopPublicRoutes serves what HiveGPT 桌面助手 fetches without a session: the published
// skills (cards + SKILL.md) on every launch. Sign-in lives under /auth/desktop (auth.go).
func RegisterDesktopPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers, redisClient *redis.Client) {
	if h.Desktop == nil {
		return
	}
	limiter := middleware.NewRateLimiter(redisClient)
	v1.GET("/desktop/skills", limiter.LimitWithOptions("desktop-skills", 120, time.Minute, middleware.RateLimitOptions{
		FailureMode: middleware.RateLimitFailOpen,
	}), h.Desktop.Skills)
}
