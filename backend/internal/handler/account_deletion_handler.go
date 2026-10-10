package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// AccountDeletionHandler serves self-service account deletion.
type AccountDeletionHandler struct {
	svc *service.AccountDeletionService
}

func NewAccountDeletionHandler(svc *service.AccountDeletionService) *AccountDeletionHandler {
	return &AccountDeletionHandler{svc: svc}
}

// DeleteSelf POST /api/v1/user/delete-account  {"confirm": "注销账号"}
func (h *AccountDeletionHandler) DeleteSelf(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		Confirm string `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.svc.DeleteSelf(c.Request.Context(), subject.UserID, req.Confirm); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
