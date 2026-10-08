package handler

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// CanvasMembershipHandler serves 创作会员 to the purchase page and the admin (see service.CanvasMembershipService).
type CanvasMembershipHandler struct {
	membership *service.CanvasMembershipService
	users      service.UserRepository
}

func NewCanvasMembershipHandler(membership *service.CanvasMembershipService, users service.UserRepository) *CanvasMembershipHandler {
	return &CanvasMembershipHandler{membership: membership, users: users}
}

// Status GET /api/v1/canvas-membership — plans on sale and the user's membership end.
func (h *CanvasMembershipHandler) Status(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "请先登录")
		return
	}
	status, err := h.membership.Status(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

// AdminGet GET /api/v1/admin/canvas-membership — the plans setting and the members.
func (h *CanvasMembershipHandler) AdminGet(c *gin.Context) {
	ctx := c.Request.Context()
	members, err := h.membership.ListMembers(ctx, c.Query("all") != "1")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"config": h.membership.Config(ctx), "members": members})
}

// AdminSave PUT /api/v1/admin/canvas-membership — replaces the plans setting.
func (h *CanvasMembershipHandler) AdminSave(c *gin.Context) {
	var cfg service.CanvasMembershipConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if _, err := h.membership.SaveConfig(c.Request.Context(), cfg); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.AdminGet(c)
}

var errMembershipGrantUser = infraerrors.NotFound("USER_NOT_FOUND", "找不到这个用户，请检查用户 ID 或邮箱（User not found）")

// AdminGrant POST /api/v1/admin/canvas-membership/grant — gives (days > 0) or takes back days.
func (h *CanvasMembershipHandler) AdminGrant(c *gin.Context) {
	var in struct {
		UserID int64  `json:"user_id"`
		Email  string `json:"email"`
		Days   int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := c.Request.Context()
	userID := in.UserID
	if userID <= 0 {
		email := strings.TrimSpace(in.Email)
		if email == "" {
			response.ErrorFrom(c, errMembershipGrantUser)
			return
		}
		user, err := h.users.GetByEmail(ctx, email)
		if err != nil || user == nil {
			response.ErrorFrom(c, errMembershipGrantUser)
			return
		}
		userID = user.ID
	} else if user, err := h.users.GetByID(ctx, userID); err != nil || user == nil {
		response.ErrorFrom(c, errMembershipGrantUser)
		return
	}
	until, err := h.membership.Grant(ctx, userID, in.Days)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "until": until})
}
