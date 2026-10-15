package handler

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserRiskHandler serves 用户画像, 风控事件 and 静默限制 (admin only).
type UserRiskHandler struct {
	risk   *service.UserRiskService
	shadow *service.UserShadowService
}

func NewUserRiskHandler(risk *service.UserRiskService, shadow *service.UserShadowService) *UserRiskHandler {
	return &UserRiskHandler{risk: risk, shadow: shadow}
}

// Overview GET /api/v1/admin/user-insights/overview?days=30
func (h *UserRiskHandler) Overview(c *gin.Context) {
	days, _ := strconv.Atoi(c.Query("days"))
	out, err := h.risk.Overview(c.Request.Context(), days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Profile GET /api/v1/admin/user-insights/profile?q=<user id or email>
func (h *UserRiskHandler) Profile(c *gin.Context) {
	out, err := h.risk.Profile(c.Request.Context(), c.Query("q"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Events GET /api/v1/admin/user-insights/events?user_id=&page=&page_size=
func (h *UserRiskHandler) Events(c *gin.Context) {
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	items, total, err := h.risk.ListEvents(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// RunMonitor POST /api/v1/admin/user-insights/monitor/run — evaluate the rules now.
func (h *UserRiskHandler) RunMonitor(c *gin.Context) {
	fresh, err := h.risk.RunMonitor(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"new_events": fresh})
}

// Shadows GET /api/v1/admin/user-insights/shadows?user_ids=1,2,3 — the restrictions of these users.
func (h *UserRiskHandler) Shadows(c *gin.Context) {
	var ids []int64
	for _, part := range strings.Split(c.Query("user_ids"), ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 && len(ids) < 500 {
			ids = append(ids, id)
		}
	}
	found := h.shadow.Lookup(ids)
	out := make(map[string]service.UserShadow, len(found))
	for id, v := range found {
		out[strconv.FormatInt(id, 10)] = v
	}
	response.Success(c, out)
}

type shadowRequest struct {
	Reason string `json:"reason"`
}

// Shadow PUT /api/v1/admin/user-insights/shadows/:id {reason}
func (h *UserRiskHandler) Shadow(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req shadowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	var adminID *int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		id := subject.UserID
		adminID = &id
	}
	out, err := h.shadow.Shadow(c.Request.Context(), userID, req.Reason, service.ShadowSourceAdmin, nil, adminID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Unshadow DELETE /api/v1/admin/user-insights/shadows/:id
func (h *UserRiskHandler) Unshadow(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.shadow.Unshadow(c.Request.Context(), userID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID})
}
