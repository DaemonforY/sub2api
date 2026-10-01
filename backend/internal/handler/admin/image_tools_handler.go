package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ImageToolsHandler manages the canvas image tools: prices, free allowance, on/off, usage records.
type ImageToolsHandler struct {
	service *service.ImageToolsService
}

func NewImageToolsHandler(svc *service.ImageToolsService) *ImageToolsHandler {
	return &ImageToolsHandler{service: svc}
}

// Settings GET /api/v1/admin/image-tools/settings
func (h *ImageToolsHandler) Settings(c *gin.Context) {
	response.Success(c, h.service.Settings(c.Request.Context()))
}

// SaveSettings PUT /api/v1/admin/image-tools/settings
func (h *ImageToolsHandler) SaveSettings(c *gin.Context) {
	var in service.ImageToolsSettingsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	out, err := h.service.SaveSettings(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Stats GET /api/v1/admin/image-tools/stats
func (h *ImageToolsHandler) Stats(c *gin.Context) {
	stats, err := h.service.Stats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// Uses GET /api/v1/admin/image-tools/uses?q=&tool=&user_id=&start_date=&end_date=&page=&page_size=
func (h *ImageToolsHandler) Uses(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	q := service.ImageToolUseQuery{UserID: userID, Keyword: c.Query("q"), Tool: c.Query("tool"), Page: page, PageSize: pageSize}
	if t, err := timezone.ParseInLocation("2006-01-02", c.Query("start_date")); err == nil {
		q.From = &t
	}
	if t, err := timezone.ParseInLocation("2006-01-02", c.Query("end_date")); err == nil {
		end := t.Add(24 * time.Hour)
		q.To = &end
	}
	list, err := h.service.ListUses(c.Request.Context(), q)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}
