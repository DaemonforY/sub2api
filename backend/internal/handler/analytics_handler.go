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
	svc *service.AnalyticsService
}

func NewAnalyticsHandler(svc *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
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
