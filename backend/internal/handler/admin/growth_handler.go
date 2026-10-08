package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GrowthHandler serves the admin "推广设置" and invite leaderboard pages.
type GrowthHandler struct {
	service  *service.GrowthService
	withdraw *service.AffiliateWithdrawService
}

// NewGrowthHandler creates the admin growth handler.
func NewGrowthHandler(svc *service.GrowthService, withdraw *service.AffiliateWithdrawService) *GrowthHandler {
	return &GrowthHandler{service: svc, withdraw: withdraw}
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
	// Start from the saved settings so fields an older page doesn't send (e.g. the withdrawal rules) keep their values.
	var req service.GrowthSettings
	if current, err := h.service.GetSettings(c.Request.Context()); err == nil && current != nil {
		req = *current
	}
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

type grantEduRequest struct {
	User string `json:"user"`
	Note string `json:"note"`
}

// GrantEduVerification POST /api/v1/admin/growth/edu-verifications {user: account email or ID, note}
func (h *GrowthHandler) GrantEduVerification(c *gin.Context) {
	var req grantEduRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var adminID int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		adminID = subject.UserID
	}
	v, err := h.service.AdminGrantEduVerification(c.Request.Context(), adminID, req.User, req.Note)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
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

// ListWithdrawals GET /api/v1/admin/growth/withdrawals?status=&search=&page=&page_size=
func (h *GrowthHandler) ListWithdrawals(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	f := service.WithdrawFilter{Status: c.Query("status"), Search: c.Query("search"), Page: page, PageSize: pageSize}
	items, total, err := h.withdraw.AdminList(c.Request.Context(), f)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	response.Paginated(c, items, total, f.Page, f.PageSize)
}

// WithdrawalPendingCount GET /api/v1/admin/growth/withdrawals/pending-count
func (h *GrowthHandler) WithdrawalPendingCount(c *gin.Context) {
	n, err := h.withdraw.PendingCount(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"count": n})
}

type withdrawReviewRequest struct {
	Note string `json:"note"`
}

// MarkWithdrawalPaid POST /api/v1/admin/growth/withdrawals/:id/paid
func (h *GrowthHandler) MarkWithdrawalPaid(c *gin.Context) {
	h.reviewWithdrawal(c, true)
}

// RejectWithdrawal POST /api/v1/admin/growth/withdrawals/:id/reject
func (h *GrowthHandler) RejectWithdrawal(c *gin.Context) {
	h.reviewWithdrawal(c, false)
}

func (h *GrowthHandler) reviewWithdrawal(c *gin.Context, paid bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid withdrawal id")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req withdrawReviewRequest
	_ = c.ShouldBindJSON(&req)
	var w *service.AffiliateWithdrawal
	if paid {
		w, err = h.withdraw.MarkPaid(c.Request.Context(), id, subject.UserID, req.Note)
	} else {
		w, err = h.withdraw.Reject(c.Request.Context(), id, subject.UserID, req.Note)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}
