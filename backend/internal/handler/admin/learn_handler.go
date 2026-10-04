package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// LearnHandler: AI 学习 run settings (learning key, model, free runs) and stats.
type LearnHandler struct {
	svc *service.LearnService
}

func NewLearnHandler(svc *service.LearnService) *LearnHandler {
	return &LearnHandler{svc: svc}
}

// Settings GET /api/v1/admin/learn/settings
func (h *LearnHandler) Settings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// SaveSettings PUT /api/v1/admin/learn/settings {run_enabled, model, free_runs_per_day, daily_cap, api_key?}
func (h *LearnHandler) SaveSettings(c *gin.Context) {
	var in service.LearnSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	out, err := h.svc.SaveSettings(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

// Stats GET /api/v1/admin/learn/stats
func (h *LearnHandler) Stats(c *gin.Context) {
	stats, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

// Certificates GET /api/v1/admin/learn/certificates?page=&page_size=
func (h *LearnHandler) Certificates(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	items, total, err := h.svc.AdminCertificates(c.Request.Context(), page, size)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total})
}

// RevokeCertificate POST /api/v1/admin/learn/certificates/:code/revoke {revoked}
func (h *LearnHandler) RevokeCertificate(c *gin.Context) {
	var in struct {
		Revoked bool `json:"revoked"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "参数格式不正确")
		return
	}
	if err := h.svc.SetCertificateRevoked(c.Request.Context(), c.Param("code"), in.Revoked); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
