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
// looks different to its owner. A sign-up that shares a browser with a banned or admin-restricted
// account or with a recent trial account, or an IP prefix with a banned or admin-restricted account,
// is silently restricted (静默限制, see UserShadowService). Sharing only an IP prefix with a trial
// account is reported by the risk monitor, never restricted: schools and offices share exit IPs.

const (
	SettingKeyGrowthSignupTrialDedupDays = "growth_signup_trial_dedup_days" // 同浏览器多少天内第二次领注册试用即静默限制（0=关闭）
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

// FingerprintMatch is an earlier account with the same browser or IP prefix.
type FingerprintMatch struct {
	UserID    int64
	ByVisitor bool   // else by IP prefix
	Status    string // disabled | shadowed (FindFlaggedMatch only)
}

// Match windows for automatic restriction. Only banned accounts and restrictions an admin made are
// match sources — an automatic restriction never restricts anyone else, so nothing chains.
const (
	shadowFlaggedVisitorWindow = 90 * 24 * time.Hour
	shadowFlaggedIPWindow      = 30 * 24 * time.Hour
)

type SignupFingerprintRepository interface {
	// FindRecentTrial returns an account that got a sign-up trial since `since` with the same visitor
	// ID (empty never matches); UserID 0 when there is none.
	FindRecentTrial(ctx context.Context, visitorID string, since time.Time) (FingerprintMatch, error)
	// FindFlaggedMatch returns a banned or admin-restricted account (other than excludeUserID) that
	// signed up with the same visitor ID since visitorSince, or from the same IP prefix since ipSince;
	// visitor matches first, then banned ones.
	FindFlaggedMatch(ctx context.Context, visitorID, ipPrefix string, excludeUserID int64, visitorSince, ipSince time.Time) (FingerprintMatch, error)
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

// Inspect records userID's sign-up and silently restricts it when
//   - it has the same browser as a banned or admin-restricted account (90 days), or the same IP
//     prefix as one (30 days);
//   - it has the same browser as an account that got the sign-up trial within the window.
//
// A shared IP alone with a trial account (a school lab, an office, a carrier pool) never restricts:
// the risk monitor only reports it. Returns the reason ("" when nothing matched); lookup errors never
// restrict anyone.
func (g *SignupTrialGuard) Inspect(ctx context.Context, userID int64, wantsTrial bool) string {
	if g == nil || g.repo == nil || userID <= 0 {
		return ""
	}
	client := signupClientFrom(ctx)
	fp := SignupFingerprint{UserID: userID, VisitorID: signupVisitorID(client.VisitorID), IPPrefix: signupIPPrefix(client.IP), TrialGranted: wantsTrial}
	reason, related := "", int64(0)
	if fp.VisitorID != "" || fp.IPPrefix != "" {
		days, linkedFlagged := g.settings(ctx)
		now := g.now()
		if linkedFlagged {
			m, err := g.repo.FindFlaggedMatch(ctx, fp.VisitorID, fp.IPPrefix, userID, now.Add(-shadowFlaggedVisitorWindow), now.Add(-shadowFlaggedIPWindow))
			if err != nil {
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] flagged lookup for user %d failed: %v", userID, err)
			} else if m.UserID > 0 {
				what := "已封禁账号"
				if m.Status != StatusDisabled {
					what = "管理员静默限制的账号"
				}
				reason, related = fmt.Sprintf("注册时与%s #%d %s", what, m.UserID, matchWay(m)), m.UserID
			}
		}
		if reason == "" && days > 0 && fp.VisitorID != "" {
			m, err := g.repo.FindRecentTrial(ctx, fp.VisitorID, now.AddDate(0, 0, -days))
			if err != nil {
				logger.LegacyPrintf("service.signup_trial", "[SignupTrial] trial lookup for user %d failed: %v", userID, err)
			} else if m.UserID > 0 {
				reason, related = fmt.Sprintf("%d 天内同浏览器的账号 #%d 已领过注册试用", days, m.UserID), m.UserID
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

// cgnatNet is carrier-grade NAT (100.64.0.0/10): many unrelated customers behind one address.
var cgnatNet = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// signupIPPrefix is the /24 (IPv4) or /64 (IPv6) of a public IP. Private, CGNAT, loopback and
// link-local addresses give "" so a proxy or a carrier NAT never makes sign-ups look like one network.
func signupIPPrefix(raw string) string {
	parsed := net.ParseIP(strings.TrimSpace(raw))
	if parsed == nil || parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsUnspecified() ||
		parsed.IsLinkLocalUnicast() || parsed.IsMulticast() || cgnatNet.Contains(parsed) {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}
