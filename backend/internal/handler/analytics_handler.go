package handler

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const analyticsMaxBody = 32 << 10

// AnalyticsHandler receives browser events (埋点) and serves the admin dashboard.
type AnalyticsHandler struct {
	svc      *service.AnalyticsService
	reminder *service.ActivationReminderService
	links    *service.ChannelLinkService
}

func NewAnalyticsHandler(svc *service.AnalyticsService, reminder *service.ActivationReminderService, links *service.ChannelLinkService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, reminder: reminder, links: links}
}

// Collect POST /api/v1/events — a batch from the main site, /learn or /editor. Signed-in requests
// tie the events to the user. Always answers quickly; bad batches get a 400.
func (h *AnalyticsHandler) Collect(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, analyticsMaxBody)
	var b service.AnalyticsBatch
	if err := c.ShouldBindJSON(&b); err != nil {
		response.ErrorFrom(c, service.ErrAnalyticsBadBatch)
		return
	}
	meta := service.AnalyticsRequestMeta{IP: ip.GetClientIP(c), UserAgent: c.GetHeader("User-Agent")}
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		meta.UserID = subject.UserID
	}
	if _, err := h.svc.Ingest(c.Request.Context(), b, meta); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AdminOverview GET /api/v1/admin/analytics/overview?days=30
func (h *AnalyticsHandler) AdminOverview(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	out, err := h.svc.Overview(c.Request.Context(), days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// AdminReminder GET /api/v1/admin/analytics/reminder — the activation reminder email: on/off,
// how many were sent, how many are due, and a preview.
func (h *AnalyticsHandler) AdminReminder(c *gin.Context) {
	out, err := h.reminder.Status(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// AdminSetReminder PUT /api/v1/admin/analytics/reminder {"enabled": bool}
func (h *AnalyticsHandler) AdminSetReminder(c *gin.Context) {
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.reminder.SetEnabled(c.Request.Context(), in.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.AdminReminder(c)
}

// ChannelRedirect GET /go/:code — a 渠道链接: off to its landing page with the utm tags.
func (h *AnalyticsHandler) ChannelRedirect(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("X-Robots-Tag", "noindex")
	c.Redirect(http.StatusFound, h.links.Resolve(c.Request.Context(), c.Param("code"), c.GetHeader("User-Agent")))
}

// AdminChannelLinks GET /api/v1/admin/analytics/channel-links
func (h *AnalyticsHandler) AdminChannelLinks(c *gin.Context) {
	out, err := h.links.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// AdminCreateChannelLink POST /api/v1/admin/analytics/channel-links
func (h *AnalyticsHandler) AdminCreateChannelLink(c *gin.Context) {
	var in service.ChannelLinkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	out, err := h.links.Create(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// AdminUpdateChannelLink PUT /api/v1/admin/analytics/channel-links/:id (the code never changes)
func (h *AnalyticsHandler) AdminUpdateChannelLink(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return
	}
	var in service.ChannelLinkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	out, err := h.links.Update(c.Request.Context(), id, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// AdminDeleteChannelLink DELETE /api/v1/admin/analytics/channel-links/:id
func (h *AnalyticsHandler) AdminDeleteChannelLink(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid id")
		return
	}
	if err := h.links.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
