//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type storedSignupFingerprint struct {
	SignupFingerprint
	at time.Time
}

type signupFingerprintRepoFake struct {
	rows    []storedSignupFingerprint
	now     func() time.Time
	findErr error
	// disabled: banned user IDs; adminShadowed reports admin-made restrictions (auto ones never count).
	disabled      map[int64]bool
	adminShadowed func(userID int64) bool
}

func (r *signupFingerprintRepoFake) FindRecentTrial(_ context.Context, visitorID string, since time.Time) (FingerprintMatch, error) {
	if r.findErr != nil {
		return FingerprintMatch{}, r.findErr
	}
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if row.TrialGranted && !row.at.Before(since) && visitorID != "" && row.VisitorID == visitorID {
			return FingerprintMatch{UserID: row.UserID, ByVisitor: true}, nil
		}
	}
	return FingerprintMatch{}, nil
}

func (r *signupFingerprintRepoFake) FindFlaggedMatch(_ context.Context, visitorID, ipPrefix string, exclude int64, visitorSince, ipSince time.Time) (FingerprintMatch, error) {
	if r.findErr != nil {
		return FingerprintMatch{}, r.findErr
	}
	var byIP FingerprintMatch
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		status := ""
		switch {
		case r.disabled[row.UserID]:
			status = StatusDisabled
		case r.adminShadowed != nil && r.adminShadowed(row.UserID):
			status = "shadowed"
		}
		if status == "" || row.UserID == exclude {
			continue
		}
		if visitorID != "" && row.VisitorID == visitorID && !row.at.Before(visitorSince) {
			return FingerprintMatch{UserID: row.UserID, ByVisitor: true, Status: status}, nil
		}
		if byIP.UserID == 0 && ipPrefix != "" && row.IPPrefix == ipPrefix && !row.at.Before(ipSince) {
			byIP = FingerprintMatch{UserID: row.UserID, Status: status}
		}
	}
	return byIP, nil
}

func (r *signupFingerprintRepoFake) Save(_ context.Context, fp SignupFingerprint) error {
	r.rows = append(r.rows, storedSignupFingerprint{SignupFingerprint: fp, at: r.now()})
	return nil
}

// userShadowRepoFake keeps restrictions, payments and spend in memory.
type userShadowRepoFake struct {
	shadows  map[int64]UserShadow
	paid     map[int64]bool
	spent    map[int64]float64
	filled   map[int64]float64
	spentErr error
	notes    []string
}

func newUserShadowRepoFake() *userShadowRepoFake {
	return &userShadowRepoFake{shadows: map[int64]UserShadow{}, paid: map[int64]bool{}, spent: map[int64]float64{}, filled: map[int64]float64{}}
}

func (r *userShadowRepoFake) ListShadows(context.Context) ([]UserShadow, error) {
	out := []UserShadow{}
	for _, v := range r.shadows {
		out = append(out, v)
	}
	return out, nil
}
func (r *userShadowRepoFake) UpsertShadow(_ context.Context, v UserShadow) error {
	r.shadows[v.UserID] = v
	return nil
}
func (r *userShadowRepoFake) DeleteShadow(_ context.Context, userID int64) error {
	delete(r.shadows, userID)
	return nil
}
func (r *userShadowRepoFake) HasRealPayment(_ context.Context, userID int64) (bool, error) {
	return r.paid[userID], nil
}
func (r *userShadowRepoFake) SpentSince(_ context.Context, userID int64, _ time.Time) (float64, error) {
	return r.spent[userID], r.spentErr
}
func (r *userShadowRepoFake) NoteRiskEvent(_ context.Context, userID int64, rule, detail string) error {
	r.notes = append(r.notes, fmt.Sprintf("%d/%s/%s", userID, rule, detail))
	return nil
}

func (r *userShadowRepoFake) FillSubscriptionDailyUsage(_ context.Context, subID int64, limit float64) error {
	r.filled[subID] = limit
	return nil
}

func growthForTest(mutate func(*GrowthSettings)) *GrowthService {
	settings := parseGrowthSettings(map[string]string{})
	if mutate != nil {
		mutate(settings)
	}
	return &GrowthService{cached: settings, cachedAt: time.Now().Add(time.Hour)}
}

func newShadowServiceForTest(growth *GrowthService) (*UserShadowService, *userShadowRepoFake) {
	repo := newUserShadowRepoFake()
	return NewUserShadowService(repo, growth, nil, nil), repo
}

func newSignupTrialGuardForTest(mutate func(*GrowthSettings)) (*SignupTrialGuard, *signupFingerprintRepoFake, *UserShadowService, *time.Time) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	repo := &signupFingerprintRepoFake{now: func() time.Time { return now }, disabled: map[int64]bool{}}
	growth := growthForTest(mutate)
	shadow, _ := newShadowServiceForTest(growth)
	repo.adminShadowed = func(id int64) bool {
		v, ok := shadow.Get(id)
		return ok && v.Source == ShadowSourceAdmin
	}
	g := NewSignupTrialGuard(repo, growth, shadow)
	g.now = func() time.Time { return now }
	return g, repo, shadow, &now
}

func signupCtx(visitorID, ip string) context.Context {
	return WithSignupClient(context.Background(), SignupClient{VisitorID: visitorID, IP: ip})
}

func TestSignupTrialGuard_SharedNetworkAloneNeverRestricts(t *testing.T) {
	g, repo, shadow, _ := newSignupTrialGuardForTest(nil)
	// A school lab: one exit IP, a different browser per student, everyone takes the trial.
	for i := int64(1); i <= 30; i++ {
		require.Empty(t, g.Inspect(signupCtx("student-browser-"+strconv.FormatInt(i, 10), "203.0.113.7"), i, true))
		require.False(t, shadow.IsShadowed(i), "student %d", i)
	}
	for _, r := range repo.rows {
		require.True(t, r.TrialGranted)
	}
	require.Equal(t, "203.0.113.0", repo.rows[0].IPPrefix, "only the /24 is stored")
}

func TestSignupTrialGuard_SameBrowserAsTrialIsRestricted(t *testing.T) {
	g, repo, shadow, now := newSignupTrialGuardForTest(nil)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true))
	reason := g.Inspect(signupCtx("visitor-aaaaaaaa", "198.51.100.9"), 2, true)
	require.Contains(t, reason, "同浏览器的账号 #1")
	v, ok := shadow.Get(2)
	require.True(t, ok)
	require.Equal(t, ShadowSourceAuto, v.Source)
	require.True(t, repo.rows[1].TrialGranted, "the restricted account still gets the trial, so nothing looks different")

	*now = now.AddDate(0, 0, 31)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "198.51.100.9"), 3, true), "outside the 30-day window")
}

func TestSignupTrialGuard_AutoRestrictionDoesNotChain(t *testing.T) {
	g, _, shadow, _ := newSignupTrialGuardForTest(nil)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true))
	require.NotEmpty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 2, true))
	require.True(t, shadow.IsShadowed(2))
	// #2 was restricted automatically: later sign-ups from its network are not restricted because of it.
	for i := int64(3); i <= 6; i++ {
		require.Empty(t, g.Inspect(signupCtx("other-browser-"+strconv.FormatInt(i, 10), "203.0.113.99"), i, true))
		require.False(t, shadow.IsShadowed(i))
	}
}

func TestSignupTrialGuard_LinkedToBannedOrAdminRestricted(t *testing.T) {
	ctx := context.Background()
	g, repo, shadow, now := newSignupTrialGuardForTest(nil)
	require.Empty(t, g.Inspect(signupCtx("visitor-banned01", "203.0.113.7"), 48, true))
	require.Empty(t, g.Inspect(signupCtx("visitor-admin001", "198.51.100.7"), 50, true))
	repo.disabled[48] = true
	_, err := shadow.Shadow(ctx, 50, "farm", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)

	*now = now.AddDate(0, 0, 60) // past the trial window, inside the 90-day browser window
	reason := g.Inspect(signupCtx("visitor-banned01", "192.0.2.1"), 61, true)
	require.Contains(t, reason, "已封禁账号 #48 同浏览器")
	require.Contains(t, g.Inspect(signupCtx("visitor-admin001", "192.0.2.2"), 62, true), "管理员静默限制的账号 #50 同浏览器")
	v, ok := shadow.Get(61)
	require.True(t, ok)
	require.Equal(t, int64(48), *v.RelatedUserID)

	// Same /24 as a banned account only counts within 30 days of its sign-up.
	require.Empty(t, g.Inspect(signupCtx("visitor-fresh001", "203.0.113.55"), 63, true), "60 days later: shared IP alone is not enough")
	require.Empty(t, g.Inspect(signupCtx("visitor-fresh002", "203.0.113.56"), 64, true))
	*now = now.AddDate(0, 0, 40) // 100 days: the browser match has expired too
	require.Empty(t, g.Inspect(signupCtx("visitor-banned01", "192.0.2.3"), 65, true))

	g2, repo2, _, _ := newSignupTrialGuardForTest(nil)
	require.Empty(t, g2.Inspect(signupCtx("visitor-xxxxxxxx", "203.0.113.7"), 70, false))
	repo2.disabled[70] = true
	require.Contains(t, g2.Inspect(signupCtx("visitor-yyyyyyyy", "203.0.113.200"), 71, true), "已封禁账号 #70 同 IP 段", "same /24 as a banned account within 30 days")

	// Turned off: no restriction at all.
	off, offRepo, offShadow, _ := newSignupTrialGuardForTest(func(s *GrowthSettings) {
		s.ShadowLinkedFlagged = false
		s.SignupTrialDedupDays = 0
	})
	require.Empty(t, off.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 80, true))
	offRepo.disabled[80] = true
	require.Empty(t, off.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 81, true))
	require.False(t, offShadow.IsShadowed(81))
}

func TestSignupIPPrefix(t *testing.T) {
	require.Equal(t, "203.0.113.0", signupIPPrefix("203.0.113.77"))
	require.Equal(t, "2001:db8:1:2::", signupIPPrefix("2001:db8:1:2:aaaa::1"), "IPv6 by /64")
	for _, ip := range []string{"172.18.0.1", "10.1.2.3", "127.0.0.1", "100.64.0.1", "100.127.255.254", "fd00::1", "fe80::1", "", "x"} {
		require.Empty(t, signupIPPrefix(ip), ip)
	}
	require.Equal(t, "100.128.0.0", signupIPPrefix("100.128.0.9"), "just outside CGNAT")
}

func TestSignupTrialGuard_PrivateIPsAndFailOpen(t *testing.T) {
	g, repo, _, _ := newSignupTrialGuardForTest(nil)
	repo.disabled[1] = true
	require.Empty(t, g.Inspect(signupCtx("", "172.18.0.1"), 1, true))
	require.Empty(t, g.Inspect(signupCtx("", "172.18.0.1"), 2, true), "a proxy that hides client IPs is never a match")
	repo.findErr = errors.New("db down")
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 3, true), "lookup errors never restrict")
}

func TestParseGrowthSettings_ShadowAndRiskDefaults(t *testing.T) {
	s := parseGrowthSettings(map[string]string{})
	require.Equal(t, 30, s.SignupTrialDedupDays)
	require.True(t, s.ShadowLinkedFlagged)
	require.Equal(t, 0.3, s.ShadowFreeDailyUSD)
	require.True(t, s.ShadowBlockPayments, "restricted accounts can't pay unless turned off")
	require.Equal(t, RiskSettings{RiskLinkedSignup: true, RiskTrialDay1: true, RiskRelayFree: true, RiskBurstSignups: true,
		RiskBurstThreshold: 3, RiskInviteeFingerprint: true, RiskTrialSameIP: true}, s.RiskSettings)

	s = parseGrowthSettings(map[string]string{SettingKeyGrowthShadowLinkedFlagged: "false", SettingKeyGrowthShadowFreeDailyUSD: "1.5",
		SettingKeyRiskRelayFree: "false", SettingKeyRiskBurstThreshold: "5"})
	require.False(t, s.ShadowLinkedFlagged)
	require.Equal(t, 1.5, s.ShadowFreeDailyUSD)
	require.False(t, s.RiskRelayFree)
	require.Equal(t, 5, s.RiskBurstThreshold)
}

func TestAuthService_Register_RepeatBrowserGetsTrialButIsRestricted(t *testing.T) {
	repo := &userRepoStub{nextID: 42}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultBalance:                      "2",
		SettingKeyDefaultSubscriptions:                `[{"group_id":8,"validity_days":1}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	guard, _, shadow, _ := newSignupTrialGuardForTest(nil)
	svc.SetSignupTrialGuard(guard)

	_, first, err := svc.RegisterWithVerification(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), "a1@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.False(t, shadow.IsShadowed(first.ID))

	repo.nextID = 43
	_, second, err := svc.RegisterWithVerification(signupCtx("visitor-aaaaaaaa", "198.51.100.1"), "a2@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, 2.0, second.Balance, "the repeat sign-up looks normal: default balance")
	require.Len(t, assigner.calls, 2, "and the trial subscription")
	require.Equal(t, int64(43), assigner.calls[1].UserID)
	require.True(t, shadow.IsShadowed(43), "but it is silently restricted")
}
