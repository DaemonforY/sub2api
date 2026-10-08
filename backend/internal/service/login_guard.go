package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Login brute-force protection. Failed password logins are counted per client IP,
// per email and site-wide in fixed 15-minute windows:
//   - after a few failures (IP or email) or a site-wide surge, login asks for the
//     built-in image captcha (unless Turnstile / Tencent / Aliyun captcha is already
//     enabled, which then guards every login);
//   - an IP with many failures is refused for a while.
//
// Redis errors fail open: logging in must not break when Redis hiccups (the route's
// request-rate limit still applies).
const (
	loginGuardWindow          = 15 * time.Minute
	loginGuardBlockFor        = 15 * time.Minute
	loginCaptchaAfterIPFails  = 3
	loginCaptchaAfterMailFail = 3
	loginCaptchaSiteWideFails = 30
	loginBlockAfterIPFails    = 20
	loginCaptchaTTL           = 5 * time.Minute
	loginCaptchaLength        = 4
	// No 0/O, 1/I/L — easy to confuse in a distorted image.
	loginCaptchaAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
)

var (
	ErrLoginCaptchaRequired = infraerrors.BadRequest("LOGIN_CAPTCHA_REQUIRED", "请输入图形验证码（Please enter the captcha）")
	ErrLoginCaptchaInvalid  = infraerrors.BadRequest("LOGIN_CAPTCHA_INVALID", "图形验证码错误或已过期，请重新输入（Invalid or expired captcha）")
)

// LoginGuardCache is the Redis side of the login guard.
type LoginGuardCache interface {
	// Incr increments key; the TTL is set when the key is created (fixed window).
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Get(ctx context.Context, key string) (int64, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Take reads and deletes key; "" when missing.
	Take(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

type LoginGuardService struct {
	cache LoginGuardCache
}

func NewLoginGuardService(cache LoginGuardCache) *LoginGuardService {
	return &LoginGuardService{cache: cache}
}

// LoginCaptcha is a freshly issued image captcha.
type LoginCaptcha struct {
	ID    string `json:"captcha_id"`
	Image string `json:"image"` // data:image/png;base64,…
}

func loginGuardIPKey(ip string) string    { return "login_guard:fail:ip:" + ip }
func loginGuardBlockKey(ip string) string { return "login_guard:block:ip:" + ip }
func loginGuardSiteKey() string           { return "login_guard:fail:all" }
func loginCaptchaKey(id string) string    { return "login_captcha:" + id }

// Emails are hashed so Redis never holds them in clear text.
func loginGuardEmailKey(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return "login_guard:fail:email:" + hex.EncodeToString(sum[:12])
}

func loginTooManyAttempts(retry time.Duration) error {
	minutes := int(math.Ceil(retry.Minutes()))
	if minutes < 1 {
		minutes = 1
	}
	return infraerrors.TooManyRequests("LOGIN_TOO_MANY_ATTEMPTS",
		fmt.Sprintf("登录失败次数过多，请 %d 分钟后再试（Too many failed login attempts, try again in %d minutes）", minutes, minutes)).
		WithMetadata(map[string]string{"retry_after_minutes": strconv.Itoa(minutes)})
}

func withCaptchaRequired(err *infraerrors.ApplicationError) error {
	return err.WithMetadata(map[string]string{"captcha_required": "true"})
}

// CheckBlocked refuses an IP that is serving a block.
func (s *LoginGuardService) CheckBlocked(ctx context.Context, ip string) error {
	if s == nil || ip == "" {
		return nil
	}
	ttl, err := s.cache.TTL(ctx, loginGuardBlockKey(ip))
	if err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] read block failed: %v", err)
		return nil
	}
	if ttl > 0 {
		return loginTooManyAttempts(ttl)
	}
	return nil
}

// CaptchaRequired reports whether this login must pass the built-in captcha.
func (s *LoginGuardService) CaptchaRequired(ctx context.Context, ip, email string) bool {
	if s == nil {
		return false
	}
	over := func(key string, limit int64) bool {
		n, err := s.cache.Get(ctx, key)
		if err != nil {
			logger.LegacyPrintf("service.login_guard", "[LoginGuard] read counter failed: %v", err)
			return false
		}
		return n >= limit
	}
	return (ip != "" && over(loginGuardIPKey(ip), loginCaptchaAfterIPFails)) ||
		over(loginGuardEmailKey(email), loginCaptchaAfterMailFail) ||
		over(loginGuardSiteKey(), loginCaptchaSiteWideFails)
}

// VerifyCaptcha checks (and burns) a captcha answer. A wrong answer counts as a
// failed attempt for the IP, so guessing captchas still ends in a block.
func (s *LoginGuardService) VerifyCaptcha(ctx context.Context, ip, id, answer string) error {
	id, answer = strings.TrimSpace(id), strings.ToUpper(strings.TrimSpace(answer))
	if id == "" || answer == "" {
		return withCaptchaRequired(ErrLoginCaptchaRequired)
	}
	want, err := s.cache.Take(ctx, loginCaptchaKey(id))
	if err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] read captcha failed: %v", err)
		return ErrServiceUnavailable
	}
	if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(answer)) != 1 {
		s.countIP(ctx, ip)
		return withCaptchaRequired(ErrLoginCaptchaInvalid)
	}
	return nil
}

// RecordFailure counts a wrong password and returns the error to send back:
// ErrInvalidCredentials, flagged when the next try needs a captcha.
func (s *LoginGuardService) RecordFailure(ctx context.Context, ip, email string) error {
	if s == nil {
		return ErrInvalidCredentials
	}
	ipFails := s.countIP(ctx, ip)
	mailFails, err := s.cache.Incr(ctx, loginGuardEmailKey(email), loginGuardWindow)
	if err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] count email failure failed: %v", err)
	}
	siteFails, err := s.cache.Incr(ctx, loginGuardSiteKey(), loginGuardWindow)
	if err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] count site failure failed: %v", err)
	}
	if ipFails >= loginCaptchaAfterIPFails || mailFails >= loginCaptchaAfterMailFail || siteFails >= loginCaptchaSiteWideFails {
		return withCaptchaRequired(ErrInvalidCredentials)
	}
	return ErrInvalidCredentials
}

// RecordSuccess forgets the email's failures. The IP's stay: many people can share
// one IP (a school, an office), and an attacker may own one of the accounts.
func (s *LoginGuardService) RecordSuccess(ctx context.Context, email string) {
	if s == nil {
		return
	}
	if err := s.cache.Del(ctx, loginGuardEmailKey(email)); err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] reset email failures failed: %v", err)
	}
}

func (s *LoginGuardService) countIP(ctx context.Context, ip string) int64 {
	if ip == "" {
		return 0
	}
	n, err := s.cache.Incr(ctx, loginGuardIPKey(ip), loginGuardWindow)
	if err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] count IP failure failed: %v", err)
		return 0
	}
	if n == loginBlockAfterIPFails {
		if err := s.cache.Set(ctx, loginGuardBlockKey(ip), "1", loginGuardBlockFor); err != nil {
			logger.LegacyPrintf("service.login_guard", "[LoginGuard] set block failed: %v", err)
		} else {
			logger.LegacyPrintf("service.login_guard", "[LoginGuard] IP %s blocked for %s after %d failed logins", ip, loginGuardBlockFor, n)
		}
	}
	return n
}

// NewCaptcha issues an image captcha valid for one try within five minutes.
func (s *LoginGuardService) NewCaptcha(ctx context.Context) (*LoginCaptcha, error) {
	if s == nil {
		return nil, ErrServiceUnavailable
	}
	code, err := randomLoginCaptchaCode()
	if err != nil {
		return nil, err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("captcha id: %w", err)
	}
	id := hex.EncodeToString(idBytes)
	img, err := renderLoginCaptcha(code)
	if err != nil {
		return nil, err
	}
	if err := s.cache.Set(ctx, loginCaptchaKey(id), code, loginCaptchaTTL); err != nil {
		logger.LegacyPrintf("service.login_guard", "[LoginGuard] store captcha failed: %v", err)
		return nil, ErrServiceUnavailable
	}
	return &LoginCaptcha{ID: id, Image: img}, nil
}

func randomLoginCaptchaCode() (string, error) {
	buf := make([]byte, loginCaptchaLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("captcha code: %w", err)
	}
	out := make([]byte, loginCaptchaLength)
	for i, b := range buf {
		out[i] = loginCaptchaAlphabet[int(b)%len(loginCaptchaAlphabet)]
	}
	return string(out), nil
}
