package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CommunityHandler moderates the canvas community: works queue, reports, author restrictions.
type CommunityHandler struct {
	svc *service.CommunityService
}

func NewCommunityHandler(svc *service.CommunityService) *CommunityHandler {
	return &CommunityHandler{svc: svc}
}

// Works GET /api/v1/admin/community/works?status=pending|reported|approved|hidden|rejected&page=
func (h *CommunityHandler) Works(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	works, err := h.svc.AdminWorks(c.Request.Context(), c.Query("status"), page, 30)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, works)
}

// Moderate POST /api/v1/admin/community/works/:id/moderate {action: approve|reject|hide|feature|unfeature, reason}
func (h *CommunityHandler) Moderate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var in struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.AdminModerate(c.Request.Context(), id, in.Action, in.Reason); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Ban POST /api/v1/admin/community/users/:id/ban {banned}
func (h *CommunityHandler) Ban(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var in struct {
		Banned bool `json:"banned"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.AdminBan(c.Request.Context(), id, in.Banned); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Reports GET /api/v1/admin/community/reports?status=open|resolved|dismissed&page=
func (h *CommunityHandler) Reports(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	list, err := h.svc.AdminReports(c.Request.Context(), c.Query("status"), page)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// SetReport PUT /api/v1/admin/community/reports/:id {status}
func (h *CommunityHandler) SetReport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.AdminSetReport(c.Request.Context(), id, in.Status); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Settings GET|PUT /api/v1/admin/community/settings {review_all}
func (h *CommunityHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.AdminSettings(c.Request.Context()))
}

func (h *CommunityHandler) SaveSettings(c *gin.Context) {
	var in struct {
		ReviewAll bool `json:"review_all"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.AdminSaveSettings(c.Request.Context(), in.ReviewAll); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.svc.AdminSettings(c.Request.Context()))
}
