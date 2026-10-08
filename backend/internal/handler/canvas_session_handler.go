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
	sessions   *service.CanvasSessionService
	affiliates *service.AffiliateService
	apiKeys    *service.APIKeyService
	membership *service.CanvasMembershipService
}

func NewCanvasSessionHandler(sessions *service.CanvasSessionService, affiliates *service.AffiliateService, apiKeys *service.APIKeyService, membership *service.CanvasMembershipService) *CanvasSessionHandler {
	return &CanvasSessionHandler{sessions: sessions, affiliates: affiliates, apiKeys: apiKeys, membership: membership}
}

type createCanvasSessionRequest struct {
	// APIKeyID: the key picked on the connect page, handed to the canvas once (redirect sign-in).
	APIKeyID int64 `json:"api_key_id"`
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
	var req createCanvasSessionRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}
	ctx := c.Request.Context()
	if req.APIKeyID > 0 {
		if h.apiKeys == nil {
			response.ErrorFrom(c, errCanvasKeyNotYours)
			return
		}
		key, err := h.apiKeys.GetByID(ctx, req.APIKeyID)
		if err != nil || key == nil || key.UserID != subject.UserID {
			response.ErrorFrom(c, errCanvasKeyNotYours)
			return
		}
	}
	token, session, err := h.sessions.Create(ctx, subject.UserID, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.APIKeyID > 0 {
		if err := h.sessions.SetHandoffKey(ctx, session.ID, req.APIKeyID); err != nil {
			response.ErrorFrom(c, err)
			return
		}
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
	// The canvas appends the invite code to the links it shares, so sign-ups through them count as invites.
	if h.affiliates != nil && h.affiliates.IsEnabled(c.Request.Context()) {
		if summary, err := h.affiliates.EnsureUserAffiliate(c.Request.Context(), user.ID); err == nil && summary != nil {
			me.AffCode = summary.AffCode
		}
	}
	if h.membership != nil {
		me.MembershipOnSale = h.membership.OnSale(c.Request.Context())
		if until, err := h.membership.Until(c.Request.Context(), user.ID); err == nil {
			me.NoWatermarkUntil = until
		}
	}
	response.Success(c, me)
}

var errCanvasKeyNotYours = infraerrors.BadRequest("CANVAS_KEY_INVALID", "这个 API Key 不属于当前账号，请重新选择（The API key does not belong to you）")

// ConnectKey GET /api/v1/canvas/connect-key — the key picked on the connect page, once (session
// cookie). Empty when no key was picked, it was already taken, or it was deleted / disabled since.
func (h *CanvasSessionHandler) ConnectKey(c *gin.Context) {
	session, user, ok := middleware.CanvasSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrCanvasSessionInvalid)
		return
	}
	ctx := c.Request.Context()
	keyID, err := h.sessions.TakeHandoffKey(ctx, session.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if keyID == 0 || h.apiKeys == nil {
		response.Success(c, gin.H{})
		return
	}
	key, err := h.apiKeys.GetByID(ctx, keyID)
	if err != nil || key == nil || key.UserID != user.ID || key.Status != service.StatusActive {
		response.Success(c, gin.H{})
		return
	}
	response.Success(c, gin.H{"api_key": key.Key, "name": key.Name})
}

type unmarkedSaveRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UnmarkedSave POST /api/v1/canvas/unmarked-saves — a member saved an image without the watermark
// (logged for 《人工智能生成合成内容标识办法》第九条). 403 when the membership has ended.
func (h *CanvasSessionHandler) UnmarkedSave(c *gin.Context) {
	_, user, ok := middleware.CanvasSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrCanvasSessionInvalid)
		return
	}
	if h.membership == nil {
		response.ErrorFrom(c, service.ErrCanvasMembershipUnavailable)
		return
	}
	var req unmarkedSaveRequest
	_ = c.ShouldBindJSON(&req)
	err := h.membership.RecordUnmarkedSave(c.Request.Context(), service.CanvasUnmarkedSave{UserID: user.ID, Width: req.Width, Height: req.Height, ClientIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
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
