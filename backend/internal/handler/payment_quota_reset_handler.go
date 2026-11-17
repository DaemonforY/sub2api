package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetQuotaReset quotes 重置今日额度 for each of the user's active subscriptions.
// GET /api/v1/payment/quota-reset
func (h *PaymentHandler) GetQuotaReset(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	out, err := h.paymentService.QuotaResetOverview(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// PurchaseQuotaResetRequest is the body of a balance-paid quota reset.
type PurchaseQuotaResetRequest struct {
	GroupID int64 `json:"group_id" binding:"required"`
}

// PurchaseQuotaResetWithBalance buys 重置今日额度 with the user's account balance.
// POST /api/v1/payment/orders/balance-quota-reset
func (h *PaymentHandler) PurchaseQuotaResetWithBalance(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var req PurchaseQuotaResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.paymentService.PurchaseQuotaResetWithBalance(c.Request.Context(), subject.UserID, req.GroupID, ip.GetClientIP(c), c.Request.Host)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
