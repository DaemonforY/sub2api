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

// Restricted GET /api/v1/admin/community/restricted — authors who cannot publish (lift with Ban {banned:false}).
func (h *CommunityHandler) Restricted(c *gin.Context) {
	list, err := h.svc.AdminRestricted(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
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

// Comments GET /api/v1/admin/community/comments?status=pending|reported|hidden&page=
func (h *CommunityHandler) Comments(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	list, err := h.svc.AdminComments(c.Request.Context(), c.Query("status"), page)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, list)
}

// ModerateComment POST /api/v1/admin/community/comments/:id/moderate {action: approve|hide}
func (h *CommunityHandler) ModerateComment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var in struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.AdminModerateComment(c.Request.Context(), id, in.Action); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Settings GET|PUT /api/v1/admin/community/settings {review_all, comments_enabled, comments_review_all, cloud_quota_mb, cloud_subscriber_quota_mb}
func (h *CommunityHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.AdminSettings(c.Request.Context()))
}

func (h *CommunityHandler) SaveSettings(c *gin.Context) {
	var in struct {
		ReviewAll         bool   `json:"review_all"`
		QuotaMB           *int64 `json:"cloud_quota_mb"`
		SubscriberQuotaMB *int64 `json:"cloud_subscriber_quota_mb"`
		CommentsEnabled   *bool  `json:"comments_enabled"`
		CommentsReviewAll *bool  `json:"comments_review_all"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	var quotas *service.CanvasCloudQuotaSettings
	if in.QuotaMB != nil && in.SubscriberQuotaMB != nil {
		quotas = &service.CanvasCloudQuotaSettings{QuotaMB: *in.QuotaMB, SubscriberQuotaMB: *in.SubscriberQuotaMB}
	}
	var comments *service.CommentSettings
	if in.CommentsEnabled != nil && in.CommentsReviewAll != nil {
		comments = &service.CommentSettings{CommentsEnabled: *in.CommentsEnabled, CommentsReviewAll: *in.CommentsReviewAll}
	}
	if err := h.svc.AdminSaveSettings(c.Request.Context(), in.ReviewAll, comments, quotas); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.svc.AdminSettings(c.Request.Context()))
}
