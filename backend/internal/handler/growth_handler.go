package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GrowthHandler serves user-facing growth programs: education verification and the invite leaderboard.
type GrowthHandler struct {
	service        *service.GrowthService
	settingService *service.SettingService
	withdraw       *service.AffiliateWithdrawService
	billingRoute   *service.BillingRouteService
}

// ProvideGrowthHandler also tells the pages whether one key serves both subscriptions and balance.
func ProvideGrowthHandler(svc *service.GrowthService, settingService *service.SettingService, withdraw *service.AffiliateWithdrawService, billingRoute *service.BillingRouteService) *GrowthHandler {
	h := NewGrowthHandler(svc, settingService, withdraw)
	h.billingRoute = billingRoute
	return h
}

// NewGrowthHandler creates the user growth handler.
func NewGrowthHandler(svc *service.GrowthService, settingService *service.SettingService, withdraw *service.AffiliateWithdrawService) *GrowthHandler {
	return &GrowthHandler{service: svc, settingService: settingService, withdraw: withdraw}
}

type growthPublicConfig struct {
	AffiliateEnabled        bool    `json:"affiliate_enabled"`
	InviteeBonusRatePercent float64 `json:"invitee_bonus_rate_percent"`
	InviteeBonusCap         float64 `json:"invitee_bonus_cap"`
	InviteeSignupBonus      float64 `json:"invitee_signup_bonus"`
	PriceLockEnabled        bool    `json:"price_lock_enabled"`
	PriceLockGraceDays      int     `json:"price_lock_grace_days"`
	// One key serves both the user's subscriptions and balance (smart billing).
	SmartBilling       bool     `json:"smart_billing"`
	LeaderboardEnabled bool     `json:"leaderboard_enabled"`
	EduVerifyEnabled   bool     `json:"edu_verify_enabled"`
	EduDiscountPercent float64  `json:"edu_discount_percent"`
	EduEmailSuffixes   []string `json:"edu_email_suffixes"`
	WithdrawEnabled    bool     `json:"withdraw_enabled"`
	WithdrawMinCNY     float64  `json:"withdraw_min_cny,omitempty"`
	// 首充奖励 (0 = off); whether a user still qualifies comes with the checkout info.
	FirstTopupBonusPercent float64 `json:"first_topup_bonus_percent"`
	FirstTopupBonusCap     float64 `json:"first_topup_bonus_cap"`
	FirstTopupMinAmount    float64 `json:"first_topup_min_amount"`
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
		PriceLockEnabled:   settings.PriceLockEnabled,
		PriceLockGraceDays: settings.PriceLockGraceDays,
		SmartBilling:       h.billingRoute.Enabled(ctx),
	}
	if settings.FirstTopupBonusPercent > 0 {
		cfg.FirstTopupBonusPercent = settings.FirstTopupBonusPercent
		cfg.FirstTopupBonusCap = settings.FirstTopupBonusCap
		cfg.FirstTopupMinAmount = settings.FirstTopupMinAmount
	}
	if affiliateEnabled {
		cfg.InviteeBonusRatePercent = settings.InviteeBonusRatePercent
		cfg.InviteeBonusCap = settings.InviteeBonusCap
		cfg.InviteeSignupBonus = settings.InviteeSignupBonus
		if settings.Enabled {
			cfg.WithdrawEnabled = true
			cfg.WithdrawMinCNY = settings.MinCNY
		}
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

// WithdrawStatus GET /api/v1/user/aff/withdraw — the withdrawal box on 邀请返利.
func (h *GrowthHandler) WithdrawStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.withdraw.Status(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

// RequestWithdraw POST /api/v1/user/aff/withdraw
func (h *GrowthHandler) RequestWithdraw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req service.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	w, err := h.withdraw.Request(c.Request.Context(), subject.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}

// CancelWithdraw POST /api/v1/user/aff/withdraw/:id/cancel
func (h *GrowthHandler) CancelWithdraw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid withdrawal id")
		return
	}
	w, err := h.withdraw.Cancel(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, w)
}
