package handler

import (
	"net/http"
	"strconv"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasSessionHandler signs the canvas in with a HiveGPT account (see service.CanvasSessionService).
type CanvasSessionHandler struct {
	sessions *service.CanvasSessionService
}

func NewCanvasSessionHandler(sessions *service.CanvasSessionService) *CanvasSessionHandler {
	return &CanvasSessionHandler{sessions: sessions}
}

// Sessions is the service behind the handler (the routes build the cookie middleware from it).
func (h *CanvasSessionHandler) Sessions() *service.CanvasSessionService { return h.sessions }

func secureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

func (h *CanvasSessionHandler) setCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     service.CanvasSessionCookie,
		Value:    token,
		Path:     service.CanvasSessionPath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// Create POST /api/v1/user/canvas-sessions — called by the main site's connect popup (panel JWT).
func (h *CanvasSessionHandler) Create(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	token, session, err := h.sessions.Create(c.Request.Context(), subject.UserID, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.setCookie(c, token, int(service.CanvasSessionTTL.Seconds()))
	response.Success(c, session)
}

// List GET /api/v1/user/canvas-sessions — signed-in canvas devices (panel JWT).
func (h *CanvasSessionHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	sessions, err := h.sessions.List(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, sessions)
}

// Revoke DELETE /api/v1/user/canvas-sessions/:id (panel JWT).
func (h *CanvasSessionHandler) Revoke(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_ID", "参数错误（Invalid id）"))
		return
	}
	if _, err := h.sessions.Revoke(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Me GET /api/v1/canvas/me — who is signed in to the canvas (session cookie).
func (h *CanvasSessionHandler) Me(c *gin.Context) {
	_, user, ok := middleware.CanvasSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrCanvasSessionInvalid)
		return
	}
	me, err := h.sessions.Me(c.Request.Context(), user)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, me)
}

// Logout POST /api/v1/canvas/logout — ends this canvas session and clears the cookie.
func (h *CanvasSessionHandler) Logout(c *gin.Context) {
	if token, _ := c.Cookie(service.CanvasSessionCookie); token != "" {
		if session, user, err := h.sessions.Authenticate(c.Request.Context(), token); err == nil {
			if _, err := h.sessions.Revoke(c.Request.Context(), user.ID, session.ID); err != nil {
				response.ErrorFrom(c, err)
				return
			}
		}
	}
	h.setCookie(c, "", -1)
	response.Success(c, gin.H{"ok": true})
}
