package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 注册试用去重 / 自动静默限制: every sign-up still gets the default subscriptions and balance, so nothing
// looks different to its owner. But a sign-up from the same browser or IP /24 as a banned or
// restricted account, or as an account that got the sign-up trial within the window, is silently
// restricted (静默限制, see UserShadowService): its free credit only works up to a small daily allowance.

const (
	SettingKeyGrowthSignupTrialDedupDays = "growth_signup_trial_dedup_days" // 同浏览器 / 同 IP 段多少天内只发一次注册试用订阅（0=不去重）
	growthSignupTrialDedupDaysDefault    = 30
	growthSignupTrialDedupDaysMax        = 365
)

// SignupClient is what the register request knows about the browser: the analytics visitor ID it
// sent and the client IP. Handlers put it on the context; AuthService reads it when granting.
type SignupClient struct {
	VisitorID string
	IP        string
}

type signupClientKey struct{}

// WithSignupClient attaches the sign-up's browser and IP to ctx.
func WithSignupClient(ctx context.Context, c SignupClient) context.Context {
	return context.WithValue(ctx, signupClientKey{}, c)
}

func signupClientFrom(ctx context.Context) SignupClient {
	c, _ := ctx.Value(signupClientKey{}).(SignupClient)
	return c
}

// SignupFingerprint is one stored sign-up.
type SignupFingerprint struct {
	UserID       int64
	VisitorID    string
	IPPrefix     string
	TrialGranted bool
}

// FingerprintMatch is an earlier account with the same browser or /24.
type FingerprintMatch struct {
	UserID    int64
	ByVisitor bool   // else by IP prefix
	Status    string // disabled | shadowed (FindFlaggedMatch only)
}

type SignupFingerprintRepository interface {
	// FindRecentTrial returns an account that got a sign-up trial since `since` with the same visitor
	// ID or IP prefix (empty values never match); UserID 0 when there is none.
	FindRecentTrial(ctx context.Context, visitorID, ipPrefix string, since time.Time) (FingerprintMatch, error)
	// FindFlaggedMatch returns a banned or restricted account (other than excludeUserID) with the
	// same visitor ID or IP prefix; banned ones first.
	FindFlaggedMatch(ctx context.Context, visitorID, ipPrefix string, excludeUserID int64) (FingerprintMatch, error)
	Save(ctx context.Context, fp SignupFingerprint) error
}

// SignupTrialGuard records each sign-up's fingerprint and restricts the suspicious ones.
type SignupTrialGuard struct {
	repo   SignupFingerprintRepository
	growth *GrowthService
	shadow *UserShadowService
	now    func() time.Time
}

func NewSignupTrialGuard(repo SignupFingerprintRepository, growth *GrowthService, shadow *UserShadowService) *SignupTrialGuard {
	return &SignupTrialGuard{repo: repo, growth: growth, shadow: shadow, now: time.Now}
}

func (g *SignupTrialGuard) settings(ctx context.Context) (dedupDays int, linkedFlagged bool) {
	if g.growth == nil {
		return growthSignupTrialDedupDaysDefault, true
	}
	settings, err := g.growth.GetSettings(ctx)
	if err != nil {
		logger.LegacyPrintf("service.signup_trial", "[SignupTrial] load settings failed: %v", err)
		return growthSignupTrialDedupDaysDefault, true
	}
	return settings.SignupTrialDedupDays, settings.ShadowLinkedFlagged
}

func matchWay(m FingerprintMatch) string {
	if m.ByVisitor {
		return "同浏览器"
	}
	return "同 IP 段"
}

// Inspect records userID's sign-up and silently restricts it when it matches a banned / restricted
// account, or an account that got the sign-up trial within the window. It returns the reason ("" when
// nothing matched). Lookup errors never restrict anyone.
func (g *SignupTrialGuard) Inspect(ctx context.Context, userID int64, wantsTrial bool) string {
	if g == nil || g.repo == nil || userID <= 0 {
		return ""
	}
	client := signupClientFrom(ctx)
	fp := SignupFingerprint{UserID: userID, VisitorID: signupVisitorID(client.VisitorID), IPPrefix: signupIPPrefix(client.IP), TrialGranted: wantsTrial}
	reason, related := "", int64(0)
	if fp.VisitorID != "" || fp.IPPrefix != "" {
		days, linkedFlagged := g.settings(ctx)
		if linkedFlagged {
			m, err := g.repo.FindFlaggedMatch(ctx, fp.VisitorID, fp.IPPrefix, userID)
			if err != nil {
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] flagged lookup for user %d failed: %v", userID, err)
			} else if m.UserID > 0 {
				what := "已封禁账号"
				if m.Status != StatusDisabled {
					what = "静默限制账号"
				}
				reason, related = fmt.Sprintf("注册时与%s #%d %s", what, m.UserID, matchWay(m)), m.UserID
			}
		}
		if reason == "" && days > 0 {
			m, err := g.repo.FindRecentTrial(ctx, fp.VisitorID, fp.IPPrefix, g.now().AddDate(0, 0, -days))
			if err != nil {
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] trial lookup for user %d failed: %v", userID, err)
			} else if m.UserID > 0 {
				reason, related = fmt.Sprintf("%d 天内%s的账号 #%d 已领过注册试用", days, matchWay(m), m.UserID), m.UserID
			}
		}
	}
	if err := g.repo.Save(ctx, fp); err != nil {
		logger.LegacyPrintf("service.signup_trial", "[SignupTrial] save fingerprint for user %d failed: %v", userID, err)
	}
	if reason != "" && g.shadow != nil {
		if _, err := g.shadow.Shadow(ctx, userID, reason, ShadowSourceAuto, &related, nil); err != nil {
			logger.LegacyPrintf("service.signup_trial", "[SignupTrial] restrict user %d failed: %v", userID, err)
		}
	}
	return reason
}

func signupVisitorID(v string) string {
	v = strings.TrimSpace(v)
	if !analyticsIDPattern.MatchString(v) {
		return ""
	}
	return v
}

// signupIPPrefix is the /24 (/48) of a public IP. Private, loopback and link-local addresses give ""
// so a proxy that hides client IPs never makes every sign-up look like the same network.
func signupIPPrefix(raw string) string {
	parsed := net.ParseIP(strings.TrimSpace(raw))
	if parsed == nil || parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsUnspecified() ||
		parsed.IsLinkLocalUnicast() || parsed.IsMulticast() {
		return ""
	}
	return analyticsIPPrefix(parsed.String())
}
