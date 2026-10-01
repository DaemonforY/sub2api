package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ImageToolsHandler serves the canvas' server-side image tools (API-key authenticated).
type ImageToolsHandler struct {
	service *service.ImageToolsService
}

func NewImageToolsHandler(svc *service.ImageToolsService) *ImageToolsHandler {
	return &ImageToolsHandler{service: svc}
}

// Quota GET /api/v1/image-tools/quota
func (h *ImageToolsHandler) Quota(c *gin.Context) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	q, err := h.service.Quota(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, q)
}

// RemoveBackground POST /api/v1/image-tools/remove-bg (multipart "file", optional "model")
func (h *ImageToolsHandler) RemoveBackground(c *gin.Context) {
	h.run(c, service.ImageToolRemoveBg, service.ImageToolOptions{Model: c.PostForm("model")})
}

// Upscale POST /api/v1/image-tools/upscale (multipart "file", optional "scale" 2|4)
func (h *ImageToolsHandler) Upscale(c *gin.Context) {
	scale, _ := strconv.Atoi(c.PostForm("scale"))
	h.run(c, service.ImageToolUpscale, service.ImageToolOptions{Scale: scale})
}

// run answers with the image itself; X-Image-Tools-Cost / -Free-Left tell what the run cost.
func (h *ImageToolsHandler) run(c *gin.Context, tool string, opts service.ImageToolOptions) {
	userID, ok := appStateUserID(c)
	if !ok {
		return
	}
	var apiKeyID int64
	if key, ok := middleware.GetAPIKeyFromContext(c); ok && key != nil {
		apiKeyID = key.ID
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ImageToolMaxInputBytes+1<<20)
	file, err := c.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.ErrorFrom(c, service.ErrImageToolsTooLarge)
			return
		}
		response.ErrorFrom(c, service.ErrImageToolsNoImage)
		return
	}
	if file.Size > service.ImageToolMaxInputBytes {
		response.ErrorFrom(c, service.ErrImageToolsTooLarge)
		return
	}
	f, err := file.Open()
	if err != nil {
		response.ErrorFrom(c, service.ErrImageToolsNoImage)
		return
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, service.ImageToolMaxInputBytes+1))
	if err != nil {
		response.ErrorFrom(c, service.ErrImageToolsNoImage)
		return
	}
	res, err := h.service.Run(c.Request.Context(), userID, apiKeyID, tool, opts, data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("X-Image-Tools-Cost", strconv.FormatFloat(res.Cost, 'f', -1, 64))
	c.Header("X-Image-Tools-Free-Left", strconv.Itoa(res.FreeLeft))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, res.ContentType, res.Data)
}
