package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserAppStateHandler serves per-user documents that companion apps (the canvas) sync across
// devices, authenticated with the user's API key.
type UserAppStateHandler struct {
	service *service.UserAppStateService
}

func NewUserAppStateHandler(svc *service.UserAppStateService) *UserAppStateHandler {
	return &UserAppStateHandler{service: svc}
}

type putUserAppStateRequest struct {
	Value       json.RawMessage `json:"value"`
	BaseVersion int64           `json:"base_version"`
}

type putUserAppStateResponse struct {
	*service.UserAppState
	// Conflict is true when another device wrote first; State then holds the current document.
	Conflict bool `json:"conflict"`
}

// Get GET /api/v1/app-state/:namespace
func (h *UserAppStateHandler) Get(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	state, err := h.service.Get(c.Request.Context(), userID, strings.TrimSpace(c.Param("namespace")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

// Put PUT /api/v1/app-state/:namespace  body: {"value": {...}, "base_version": n}
func (h *UserAppStateHandler) Put(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.UserAppStateMaxBytes+4096)
	var req putUserAppStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrUserAppStateInvalid)
		return
	}
	state, applied, err := h.service.Put(c.Request.Context(), userID, strings.TrimSpace(c.Param("namespace")), req.Value, req.BaseVersion)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, putUserAppStateResponse{UserAppState: state, Conflict: !applied})
}

func appStateUserID(c *gin.Context) (int64, bool) {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		return subject.UserID, true
	}
	response.Unauthorized(c, "请先连接 API Key")
	return 0, false
}
