package handler

import (
	"encoding/json"
	"io"
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
	blobs   *service.UserAppBlobService
}

func NewUserAppStateHandler(svc *service.UserAppStateService, blobs *service.UserAppBlobService) *UserAppStateHandler {
	return &UserAppStateHandler{service: svc, blobs: blobs}
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

// UploadBlob POST /api/v1/app-state/blobs  multipart "file" — stores an image for syncing
// (e.g. a draft reference image) and returns its id. Same bytes return the same id.
func (h *UserAppStateHandler) UploadBlob(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.UserAppBlobMaxBytes+1<<20)
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, service.ErrUserAppBlobInvalid)
		return
	}
	if file.Size > service.UserAppBlobMaxBytes {
		response.ErrorFrom(c, service.ErrUserAppBlobTooLarge)
		return
	}
	f, err := file.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrUserAppBlobInvalid)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, service.UserAppBlobMaxBytes+1))
	if err != nil {
		response.ErrorFrom(c, service.ErrUserAppBlobInvalid)
		return
	}
	blob, err := h.blobs.Upload(c.Request.Context(), userID, data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, blob)
}

// GetBlob GET /api/v1/app-state/blobs/:id — only the owner can read it.
func (h *UserAppStateHandler) GetBlob(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	path, blob, err := h.blobs.Open(c.Request.Context(), userID, strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", blob.MimeType)
	c.File(path)
}

func appStateUserID(c *gin.Context) (int64, bool) {
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		return subject.UserID, true
	}
	response.Unauthorized(c, "请先连接 API Key")
	return 0, false
}
