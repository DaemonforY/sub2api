package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasCloudHandler serves the canvas cloud sync (session cookie, see service.CanvasCloudService).
type CanvasCloudHandler struct {
	svc *service.CanvasCloudService
}

func NewCanvasCloudHandler(svc *service.CanvasCloudService) *CanvasCloudHandler {
	return &CanvasCloudHandler{svc: svc}
}

func canvasCloudUser(c *gin.Context) (int64, bool) {
	_, user, ok := middleware.CanvasSessionFromContext(c)
	if !ok {
		response.ErrorFrom(c, service.ErrCanvasSessionInvalid)
		return 0, false
	}
	return user.ID, true
}

func canvasCloudPath(c *gin.Context) string { return strings.TrimPrefix(c.Param("path"), "/") }

// Usage GET /api/v1/canvas/cloud/usage
func (h *CanvasCloudHandler) Usage(c *gin.Context) {
	uid, ok := canvasCloudUser(c)
	if !ok {
		return
	}
	usage, err := h.svc.Usage(c.Request.Context(), uid)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, usage)
}

// List GET /api/v1/canvas/cloud/files
func (h *CanvasCloudHandler) List(c *gin.Context) {
	uid, ok := canvasCloudUser(c)
	if !ok {
		return
	}
	files, err := h.svc.List(c.Request.Context(), uid)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"files": files})
}

// Get GET /api/v1/canvas/cloud/files/*path — the raw file, as a download (never rendered here).
func (h *CanvasCloudHandler) Get(c *gin.Context) {
	uid, ok := canvasCloudUser(c)
	if !ok {
		return
	}
	f, file, err := h.svc.Open(c.Request.Context(), uid, canvasCloudPath(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = file.Close() }()
	header := c.Writer.Header()
	header.Set("Content-Type", f.Mime)
	header.Set("Content-Length", strconv.FormatInt(f.Size, 10))
	header.Set("Content-Disposition", "attachment")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	header.Set("Cache-Control", "private, no-store")
	c.Status(http.StatusOK)
	_, _ = file.WriteTo(c.Writer)
}

// Put PUT /api/v1/canvas/cloud/files/*path — the body is the file.
func (h *CanvasCloudHandler) Put(c *gin.Context) {
	uid, ok := canvasCloudUser(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.CanvasCloudMaxFileBytes+1)
	f, err := h.svc.Put(c.Request.Context(), uid, canvasCloudPath(c), c.GetHeader("Content-Type"), c.Request.Body)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, f)
}

// Delete DELETE /api/v1/canvas/cloud/files/*path
func (h *CanvasCloudHandler) Delete(c *gin.Context) {
	uid, ok := canvasCloudUser(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uid, canvasCloudPath(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
