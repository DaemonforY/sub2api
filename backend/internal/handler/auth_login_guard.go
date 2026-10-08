package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SetLoginGuard enables failed-login counting, the built-in captcha and IP blocks.
func (h *AuthHandler) SetLoginGuard(guard *service.LoginGuardService) {
	h.loginGuard = guard
}

// checkLoginCaptcha asks for the built-in image captcha once the IP, the email or
// the whole site has had several failed logins. Skipped when a third-party captcha
// (Turnstile / Tencent / Aliyun) is enabled — that one already guards every login.
func (h *AuthHandler) checkLoginCaptcha(c *gin.Context, clientIP string, req *LoginRequest) error {
	if h.loginGuard == nil || h.thirdPartyCaptchaEnabled(c) {
		return nil
	}
	if !h.loginGuard.CaptchaRequired(c.Request.Context(), clientIP, req.Email) {
		return nil
	}
	return h.loginGuard.VerifyCaptcha(c.Request.Context(), clientIP, req.CaptchaID, req.CaptchaCode)
}

// checkRegisterCaptcha asks for the image captcha on a sign-up step (sending the
// email code, or registering when email verification is off) once this IP has
// requested several codes, created accounts or failed — see register_guard.go.
func (h *AuthHandler) checkRegisterCaptcha(c *gin.Context, clientIP, captchaID, captchaCode string) error {
	if h.loginGuard == nil || h.thirdPartyCaptchaEnabled(c) {
		return nil
	}
	if !h.loginGuard.RegisterCaptchaRequired(c.Request.Context(), clientIP) {
		return nil
	}
	return h.loginGuard.VerifyCaptcha(c.Request.Context(), clientIP, captchaID, captchaCode)
}

func (h *AuthHandler) thirdPartyCaptchaEnabled(c *gin.Context) bool {
	if h.settingSvc == nil {
		return false
	}
	cfg, err := h.settingSvc.GetCaptchaProviderConfig(c.Request.Context())
	if err != nil {
		return false
	}
	return cfg.TurnstileEnabled || cfg.Tencent.Enabled || cfg.Aliyun.Enabled
}

// LoginCaptcha issues a built-in image captcha for the login and sign-up forms.
// GET /api/v1/auth/login-captcha
func (h *AuthHandler) LoginCaptcha(c *gin.Context) {
	if err := h.loginGuard.CheckBlocked(c.Request.Context(), ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	captcha, err := h.loginGuard.NewCaptcha(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, captcha)
}
