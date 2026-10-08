package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterChannelLinkRoutes serves 渠道链接 short links at the site root: hivegpt.cn/go/<code>.
func RegisterChannelLinkRoutes(r *gin.Engine, h *handler.Handlers) {
	if h.Analytics == nil {
		return
	}
	r.GET("/go/:code", h.Analytics.ChannelRedirect)
}

// RegisterAnalyticsRoutes registers the event collector (埋点): open to visitors, optional login,
// 120 batches a minute per IP (fail-open — losing a few events is fine).
func RegisterAnalyticsRoutes(v1 *gin.RouterGroup, h *handler.Handlers, optionalJWT servermiddleware.OptionalJWTAuthMiddleware, redisClient *redis.Client) {
	if h.Analytics == nil {
		return
	}
	limiter := middleware.NewRateLimiter(redisClient)
	v1.POST("/events", limiter.LimitWithOptions("analytics-events", 120, time.Minute, middleware.RateLimitOptions{
		FailureMode: middleware.RateLimitFailOpen,
	}), gin.HandlerFunc(optionalJWT), h.Analytics.Collect)
}
