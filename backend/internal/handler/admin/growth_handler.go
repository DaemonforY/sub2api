package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GrowthHandler serves the admin "推广设置" and invite leaderboard pages.
type GrowthHandler struct {
	service *service.GrowthService
}

// NewGrowthHandler creates the admin growth handler.
func NewGrowthHandler(svc *service.GrowthService) *GrowthHandler {
	return &GrowthHandler{service: svc}
}

// GetSettings GET /api/v1/admin/growth/settings
func (h *GrowthHandler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

// UpdateSettings PUT /api/v1/admin/growth/settings
func (h *GrowthHandler) UpdateSettings(c *gin.Context) {
	var req service.GrowthSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.service.UpdateSettings(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

// ListEduVerifications GET /api/v1/admin/growth/edu-verifications?page=&page_size=&search=
func (h *GrowthHandler) ListEduVerifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	items, total, err := h.service.AdminListEduVerifications(c.Request.Context(), c.Query("search"), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// RevokeEduVerification DELETE /api/v1/admin/growth/edu-verifications/:user_id
func (h *GrowthHandler) RevokeEduVerification(c *gin.Context) {
	userID, err := strconv.ParseInt(strings.TrimSpace(c.Param("user_id")), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}
	if err := h.service.AdminRevokeEduVerification(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID})
}

// Leaderboard GET /api/v1/admin/growth/leaderboard?start=&end=&limit=
// start/end accept RFC3339 or YYYY-MM-DD (end date is exclusive); omitted = unbounded.
func (h *GrowthHandler) Leaderboard(c *gin.Context) {
	start, ok := parseGrowthTime(c, "start")
	if !ok {
		return
	}
	end, ok := parseGrowthTime(c, "end")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	board, err := h.service.AdminLeaderboard(c.Request.Context(), start, end, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, board)
}

func parseGrowthTime(c *gin.Context, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, true
	}
	if t, err := time.ParseInLocation("2006-01-02", raw, time.Local); err == nil {
		return &t, true
	}
	response.BadRequest(c, "Invalid "+name+": use RFC3339 or YYYY-MM-DD")
	return nil, false
}
