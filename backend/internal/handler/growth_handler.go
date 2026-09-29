package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GrowthHandler serves user-facing growth programs: education verification and the invite leaderboard.
type GrowthHandler struct {
	service        *service.GrowthService
	settingService *service.SettingService
}

// NewGrowthHandler creates the user growth handler.
func NewGrowthHandler(svc *service.GrowthService, settingService *service.SettingService) *GrowthHandler {
	return &GrowthHandler{service: svc, settingService: settingService}
}

type growthPublicConfig struct {
	AffiliateEnabled        bool     `json:"affiliate_enabled"`
	InviteeBonusRatePercent float64  `json:"invitee_bonus_rate_percent"`
	InviteeBonusCap         float64  `json:"invitee_bonus_cap"`
	LeaderboardEnabled      bool     `json:"leaderboard_enabled"`
	EduVerifyEnabled        bool     `json:"edu_verify_enabled"`
	EduDiscountPercent      float64  `json:"edu_discount_percent"`
	EduEmailSuffixes        []string `json:"edu_email_suffixes"`
}

// PublicConfig GET /api/v1/growth/config — what the register, referral and purchase pages advertise.
func (h *GrowthHandler) PublicConfig(c *gin.Context) {
	ctx := c.Request.Context()
	settings, err := h.service.GetSettings(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	affiliateEnabled := h.settingService != nil && h.settingService.IsAffiliateEnabled(ctx)
	cfg := growthPublicConfig{
		AffiliateEnabled:   affiliateEnabled,
		LeaderboardEnabled: affiliateEnabled && settings.LeaderboardEnabled,
		EduVerifyEnabled:   settings.EduVerifyEnabled,
		EduEmailSuffixes:   settings.EduEmailSuffixes,
	}
	if affiliateEnabled {
		cfg.InviteeBonusRatePercent = settings.InviteeBonusRatePercent
		cfg.InviteeBonusCap = settings.InviteeBonusCap
	}
	if settings.EduVerifyEnabled {
		cfg.EduDiscountPercent = settings.EduDiscountPercent
	}
	response.Success(c, cfg)
}

// GetEdu GET /api/v1/user/edu
func (h *GrowthHandler) GetEdu(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.service.GetEduStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

type eduEmailRequest struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code"`
}

// SendEduCode POST /api/v1/user/edu/send-code
func (h *GrowthHandler) SendEduCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req eduEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.service.SendEduCode(c.Request.Context(), subject.UserID, req.Email, c.GetHeader("Accept-Language")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}

// VerifyEdu POST /api/v1/user/edu/verify
func (h *GrowthHandler) VerifyEdu(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req eduEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		response.BadRequest(c, "Invalid request: email and code are required")
		return
	}
	v, err := h.service.VerifyEdu(c.Request.Context(), subject.UserID, req.Email, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, v)
}

// Leaderboard GET /api/v1/user/aff/leaderboard?period=month|last_month|all
func (h *GrowthHandler) Leaderboard(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	board, err := h.service.PublicLeaderboard(c.Request.Context(), subject.UserID, c.Query("period"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, board)
}
