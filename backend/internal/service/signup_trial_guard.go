package service

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 注册试用去重: the sign-up default subscriptions (e.g. 3 days of Codex-轻量版) go to one account per
// browser and per IP /24 within a window. A repeat sign-up still gets the default balance.

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

type SignupFingerprintRepository interface {
	// FindRecentTrial returns a user who got a sign-up trial since `since` with the same visitor ID
	// or IP prefix (empty values never match), or 0 when there is none.
	FindRecentTrial(ctx context.Context, visitorID, ipPrefix string, since time.Time) (int64, error)
	Save(ctx context.Context, fp SignupFingerprint) error
}

// SignupTrialGuard decides whether a new account gets the sign-up default subscriptions.
type SignupTrialGuard struct {
	repo   SignupFingerprintRepository
	growth *GrowthService
	now    func() time.Time
}

func NewSignupTrialGuard(repo SignupFingerprintRepository, growth *GrowthService) *SignupTrialGuard {
	return &SignupTrialGuard{repo: repo, growth: growth, now: time.Now}
}

func (g *SignupTrialGuard) dedupDays(ctx context.Context) int {
	if g.growth == nil {
		return growthSignupTrialDedupDaysDefault
	}
	settings, err := g.growth.GetSettings(ctx)
	if err != nil {
		logger.LegacyPrintf("service.signup_trial", "[SignupTrial] load settings failed: %v", err)
		return growthSignupTrialDedupDaysDefault
	}
	return settings.SignupTrialDedupDays
}

// Allow reports whether userID gets the sign-up trial and records the sign-up either way.
// wantsTrial is false when no default subscription is configured (the sign-up is still recorded).
// Lookup errors fail open: a broken check never costs a real user the trial.
func (g *SignupTrialGuard) Allow(ctx context.Context, userID int64, wantsTrial bool) bool {
	if g == nil || g.repo == nil || userID <= 0 {
		return wantsTrial
	}
	client := signupClientFrom(ctx)
	fp := SignupFingerprint{UserID: userID, VisitorID: signupVisitorID(client.VisitorID), IPPrefix: signupIPPrefix(client.IP)}
	granted := wantsTrial
	if wantsTrial && (fp.VisitorID != "" || fp.IPPrefix != "") {
		if days := g.dedupDays(ctx); days > 0 {
			prior, err := g.repo.FindRecentTrial(ctx, fp.VisitorID, fp.IPPrefix, g.now().AddDate(0, 0, -days))
			switch {
			case err != nil:
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] lookup for user %d failed, granting: %v", userID, err)
			case prior > 0:
				granted = false
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] user %d gets no sign-up subscription: same browser or IP /24 as user %d within %d days (prefix=%s)",
					userID, prior, days, fp.IPPrefix)
			}
		}
	}
	fp.TrialGranted = granted
	if err := g.repo.Save(ctx, fp); err != nil {
		logger.LegacyPrintf("service.signup_trial", "[SignupTrial] save fingerprint for user %d failed: %v", userID, err)
	}
	return granted
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
