package middleware

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasRequestHeader must accompany every canvas session request. Together with the Origin check
// it keeps other pages from riding on the cookie (a cross-origin request with a custom header needs
// a CORS preflight, which only the canvas origins pass).
const CanvasRequestHeader = "X-HiveGPT-Canvas"

const contextKeyCanvasSession = "canvas_session"

// CanvasOriginGuard lets through only requests from an allowed canvas origin carrying CanvasRequestHeader.
func CanvasOriginGuard(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin = strings.TrimRight(strings.TrimSpace(origin), "/"); origin != "" && origin != "*" {
			allowed[origin] = true
		}
	}
	return func(c *gin.Context) {
		if !allowed[c.GetHeader("Origin")] || c.GetHeader(CanvasRequestHeader) != "1" {
			response.ErrorFrom(c, infraerrors.Forbidden("CANVAS_ORIGIN_REJECTED", "只接受来自无限画布的请求（Requests must come from the canvas）"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// CanvasSessionAuth authenticates canvas requests by the session cookie (use after CanvasOriginGuard).
func CanvasSessionAuth(sessions *service.CanvasSessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(service.CanvasSessionCookie)
		session, user, err := sessions.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.ErrorFrom(c, err)
			c.Abort()
			return
		}
		c.Set(string(ContextKeyUser), AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Set(contextKeyCanvasSession, session)
		c.Set(contextKeyCanvasUser, user)
		c.Next()
	}
}

const contextKeyCanvasUser = "canvas_user"

// CanvasSessionFromContext returns the session and user set by CanvasSessionAuth.
func CanvasSessionFromContext(c *gin.Context) (*service.CanvasSession, *service.User, bool) {
	s, ok1 := c.Get(contextKeyCanvasSession)
	u, ok2 := c.Get(contextKeyCanvasUser)
	session, _ := s.(*service.CanvasSession)
	user, _ := u.(*service.User)
	return session, user, ok1 && ok2 && session != nil && user != nil
}

// CanvasSessionOptional signs the request in when the cookie holds a live session and otherwise
// lets it through anonymously (public community pages).
func CanvasSessionOptional(sessions *service.CanvasSessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, _ := c.Cookie(service.CanvasSessionCookie); token != "" {
			if session, user, err := sessions.Authenticate(c.Request.Context(), token); err == nil {
				c.Set(string(ContextKeyUser), AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
				c.Set(string(ContextKeyUserRole), user.Role)
				c.Set(contextKeyCanvasSession, session)
				c.Set(contextKeyCanvasUser, user)
			}
		}
		c.Next()
	}
}
