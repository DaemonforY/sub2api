package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetQuotaResetSettings returns the 重置今日额度 settings.
// GET /api/v1/admin/payment/quota-reset
func (h *PaymentHandler) GetQuotaResetSettings(c *gin.Context) {
	response.Success(c, h.configService.GetQuotaResetSettings(c.Request.Context()))
}

// UpdateQuotaResetSettings stores the 重置今日额度 settings.
// PUT /api/v1/admin/payment/quota-reset
func (h *PaymentHandler) UpdateQuotaResetSettings(c *gin.Context) {
	var req service.QuotaResetSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	out, err := h.configService.SetQuotaResetSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
