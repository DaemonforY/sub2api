package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SiteHostingHandler manages hosted sites: settings, take-downs and abuse reports.
type SiteHostingHandler struct {
	service *service.SiteHostingService
}

func NewSiteHostingHandler(svc *service.SiteHostingService) *SiteHostingHandler {
	return &SiteHostingHandler{service: svc}
}

// Settings GET /api/v1/admin/sites/settings
func (h *SiteHostingHandler) Settings(c *gin.Context) {
	response.Success(c, h.service.Config(c.Request.Context()))
}

// SaveSettings PUT /api/v1/admin/sites/settings
func (h *SiteHostingHandler) SaveSettings(c *gin.Context) {
	var in service.SiteHostingConfig
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	out, err := h.service.SaveConfig(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// List GET /api/v1/admin/sites?q=&status=&page=&page_size=
func (h *SiteHostingHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	sites, total, err := h.service.AdminList(c.Request.Context(), service.SiteListQuery{Keyword: c.Query("q"), Status: c.Query("status"), Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": sites, "total": total})
}

func adminSiteID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrSiteNotFound)
		return 0, false
	}
	return id, true
}

// SetStatus PUT /api/v1/admin/sites/:id/status {status: active|disabled, reason}
func (h *SiteHostingHandler) SetStatus(c *gin.Context) {
	id, ok := adminSiteID(c)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteStatusInvalid)
		return
	}
	if err := h.service.AdminSetStatus(c.Request.Context(), id, in.Status, in.Reason); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": in.Status})
}

// Delete DELETE /api/v1/admin/sites/:id
func (h *SiteHostingHandler) Delete(c *gin.Context) {
	id, ok := adminSiteID(c)
	if !ok {
		return
	}
	if err := h.service.AdminDelete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Reports GET /api/v1/admin/sites/reports?status=&page=&page_size=
func (h *SiteHostingHandler) Reports(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	reports, total, err := h.service.AdminReports(c.Request.Context(), c.Query("status"), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": reports, "total": total})
}

// SetReportStatus PUT /api/v1/admin/sites/reports/:id {status: open|resolved|dismissed}
func (h *SiteHostingHandler) SetReportStatus(c *gin.Context) {
	id, ok := adminSiteID(c)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteStatusInvalid)
		return
	}
	if err := h.service.AdminSetReportStatus(c.Request.Context(), id, in.Status); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": in.Status})
}

// Reviews GET /api/v1/admin/sites/reviews — versions waiting for review.
func (h *SiteHostingHandler) Reviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	items, total, err := h.service.AdminReviews(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// Review POST /api/v1/admin/sites/:id/review {version, action: approve|reject, reason}
func (h *SiteHostingHandler) Review(c *gin.Context) {
	id, ok := adminSiteID(c)
	if !ok {
		return
	}
	var in struct {
		Version int    `json:"version"`
		Action  string `json:"action"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.ErrorFrom(c, service.ErrSiteNothingToReview)
		return
	}
	var err error
	switch in.Action {
	case "approve":
		err = h.service.AdminApprove(c.Request.Context(), id, in.Version)
	case "reject":
		err = h.service.AdminReject(c.Request.Context(), id, in.Version, in.Reason)
	default:
		err = service.ErrSiteStatusInvalid
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"action": in.Action})
}
