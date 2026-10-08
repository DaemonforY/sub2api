package service

import (
	"context"
	"errors"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Sign-up protection, sharing the login guard's image captcha, IP failure counter
// and block. With email verification on, every sign-up goes through
// send-verify-code, so that is where the captcha is asked for; without it, at
// register. The captcha is asked for once an IP has
//   - requested 3 codes in 15 minutes,
//   - created 2 accounts in 24 hours,
//   - or failed 3 times (wrong code, email taken, bad captcha …, login included),
//
// or when the whole site sees a surge of code requests. Captcha-verified requests
// are not failures, so a classroom signing up from one IP only types captchas;
// failures still lead to the 20-failure IP block.
const (
	registerCaptchaAfterIPSends   = 3
	registerCaptchaAfterIPSignups = 2
	registerCaptchaSiteWideSends  = 30
	registerSignupWindow          = 24 * time.Hour
)

func registerSendIPKey(ip string) string   { return "register_guard:send:ip:" + ip }
func registerSignupIPKey(ip string) string { return "register_guard:signup:ip:" + ip }
func registerSendSiteKey() string          { return "register_guard:send:all" }

// RegisterCaptchaRequired reports whether a sign-up step from ip must pass the captcha.
func (s *LoginGuardService) RegisterCaptchaRequired(ctx context.Context, ip string) bool {
	if s == nil {
		return false
	}
	over := func(key string, limit int64) bool {
		n, err := s.cache.Get(ctx, key)
		if err != nil {
			logger.LegacyPrintf("service.login_guard", "[RegisterGuard] read counter failed: %v", err)
			return false
		}
		return n >= limit
	}
	if over(registerSendSiteKey(), registerCaptchaSiteWideSends) {
		return true
	}
	if ip == "" {
		return false
	}
	return over(registerSendIPKey(ip), registerCaptchaAfterIPSends) ||
		over(registerSignupIPKey(ip), registerCaptchaAfterIPSignups) ||
		over(loginGuardIPKey(ip), loginCaptchaAfterIPFails)
}

// RecordCodeSent counts a verification email sent for sign-up.
func (s *LoginGuardService) RecordCodeSent(ctx context.Context, ip string) {
	if s == nil {
		return
	}
	if ip != "" {
		if _, err := s.cache.Incr(ctx, registerSendIPKey(ip), loginGuardWindow); err != nil {
			logger.LegacyPrintf("service.login_guard", "[RegisterGuard] count send failed: %v", err)
		}
	}
	if _, err := s.cache.Incr(ctx, registerSendSiteKey(), loginGuardWindow); err != nil {
		logger.LegacyPrintf("service.login_guard", "[RegisterGuard] count site send failed: %v", err)
	}
}

// RecordSignup counts an account created from ip.
func (s *LoginGuardService) RecordSignup(ctx context.Context, ip string) {
	if s == nil || ip == "" {
		return
	}
	if _, err := s.cache.Incr(ctx, registerSignupIPKey(ip), registerSignupWindow); err != nil {
		logger.LegacyPrintf("service.login_guard", "[RegisterGuard] count signup failed: %v", err)
	}
}

// RecordRegisterFailure counts a rejected sign-up step (a 4xx: wrong code, email
// taken, suffix not allowed …) against the IP and returns err, flagged with
// captcha_required when the next try needs the captcha. Server errors don't count.
func (s *LoginGuardService) RecordRegisterFailure(ctx context.Context, ip string, err error) error {
	var appErr *infraerrors.ApplicationError
	if s == nil || !errors.As(err, &appErr) || appErr.Code < 400 || appErr.Code >= 500 {
		return err
	}
	s.countIP(ctx, ip)
	if s.RegisterCaptchaRequired(ctx, ip) {
		return withCaptchaRequired(appErr)
	}
	return err
}
