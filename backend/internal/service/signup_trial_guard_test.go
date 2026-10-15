//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
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
	// flagged: user ID → "disabled" | "shadowed"
	flagged map[int64]string
}

func fpMatches(row SignupFingerprint, visitorID, ipPrefix string) (bool, bool) {
	byVisitor := row.VisitorID != "" && row.VisitorID == visitorID
	return byVisitor || (row.IPPrefix != "" && row.IPPrefix == ipPrefix), byVisitor
}

func (r *signupFingerprintRepoFake) FindRecentTrial(_ context.Context, visitorID, ipPrefix string, since time.Time) (FingerprintMatch, error) {
	if r.findErr != nil {
		return FingerprintMatch{}, r.findErr
	}
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if !row.TrialGranted || row.at.Before(since) {
			continue
		}
		if ok, byVisitor := fpMatches(row.SignupFingerprint, visitorID, ipPrefix); ok {
			return FingerprintMatch{UserID: row.UserID, ByVisitor: byVisitor}, nil
		}
	}
	return FingerprintMatch{}, nil
}

func (r *signupFingerprintRepoFake) FindFlaggedMatch(_ context.Context, visitorID, ipPrefix string, exclude int64) (FingerprintMatch, error) {
	if r.findErr != nil {
		return FingerprintMatch{}, r.findErr
	}
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		status, flagged := r.flagged[row.UserID]
		if !flagged || row.UserID == exclude {
			continue
		}
		if ok, byVisitor := fpMatches(row.SignupFingerprint, visitorID, ipPrefix); ok {
			return FingerprintMatch{UserID: row.UserID, ByVisitor: byVisitor, Status: status}, nil
		}
	}
	return FingerprintMatch{}, nil
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
	repo := &signupFingerprintRepoFake{now: func() time.Time { return now }, flagged: map[int64]string{}}
	growth := growthForTest(mutate)
	shadow, _ := newShadowServiceForTest(growth)
	g := NewSignupTrialGuard(repo, growth, shadow)
	g.now = func() time.Time { return now }
	return g, repo, shadow, &now
}

func signupCtx(visitorID, ip string) context.Context {
	return WithSignupClient(context.Background(), SignupClient{VisitorID: visitorID, IP: ip})
}

func TestSignupTrialGuard_RepeatBrowserOrNetworkIsSilentlyRestricted(t *testing.T) {
	g, repo, shadow, _ := newSignupTrialGuardForTest(nil)

	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true), "first sign-up is clean")
	reason := g.Inspect(signupCtx("visitor-aaaaaaaa", "198.51.100.9"), 2, true)
	require.Contains(t, reason, "同浏览器")
	require.Contains(t, reason, "#1")
	require.Contains(t, g.Inspect(signupCtx("visitor-bbbbbbbb", "203.0.113.200"), 3, true), "同 IP 段")
	require.NotEmpty(t, g.Inspect(signupCtx("", "203.0.113.50"), 4, true))

	require.False(t, shadow.IsShadowed(1))
	for _, id := range []int64{2, 3, 4} {
		v, ok := shadow.Get(id)
		require.True(t, ok, "user %d", id)
		require.Equal(t, ShadowSourceAuto, v.Source)
	}
	for _, r := range repo.rows {
		require.True(t, r.TrialGranted, "everyone still gets the trial, so nothing looks different")
	}
	require.Equal(t, "203.0.113.0", repo.rows[0].IPPrefix, "only the /24 is stored")
}

func TestSignupTrialGuard_LinkedToBannedAccount(t *testing.T) {
	g, repo, shadow, now := newSignupTrialGuardForTest(nil)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 48, true))
	repo.flagged[48] = StatusDisabled

	// Long after the trial window: only the ban links them.
	*now = now.AddDate(0, 3, 0)
	reason := g.Inspect(signupCtx("visitor-aaaaaaaa", "198.51.100.9"), 61, true)
	require.Contains(t, reason, "已封禁账号 #48")
	v, ok := shadow.Get(61)
	require.True(t, ok)
	require.Equal(t, int64(48), *v.RelatedUserID)

	// Turned off: no restriction.
	off, offRepo, offShadow, _ := newSignupTrialGuardForTest(func(s *GrowthSettings) {
		s.ShadowLinkedFlagged = false
		s.SignupTrialDedupDays = 0
	})
	require.Empty(t, off.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 48, true))
	offRepo.flagged[48] = StatusDisabled
	require.Empty(t, off.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 49, true))
	require.False(t, offShadow.IsShadowed(49))
}

func TestSignupTrialGuard_WindowPrivateIPsAndFailOpen(t *testing.T) {
	g, repo, _, now := newSignupTrialGuardForTest(nil)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true))
	*now = now.AddDate(0, 0, 31)
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 2, true), "outside the 30-day window")

	// A proxy that hides client IPs must not make every sign-up look like one network.
	require.Empty(t, g.Inspect(signupCtx("", "172.18.0.1"), 3, true))
	require.Empty(t, g.Inspect(signupCtx("x", "172.18.0.1"), 4, true))

	repo.findErr = errors.New("db down")
	require.Empty(t, g.Inspect(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 5, true), "lookup errors never restrict")
}

func TestParseGrowthSettings_ShadowAndRiskDefaults(t *testing.T) {
	s := parseGrowthSettings(map[string]string{})
	require.Equal(t, 30, s.SignupTrialDedupDays)
	require.True(t, s.ShadowLinkedFlagged)
	require.Equal(t, 0.3, s.ShadowFreeDailyUSD)
	require.True(t, s.ShadowBlockPayments, "restricted accounts can't pay unless turned off")
	require.Equal(t, RiskSettings{RiskLinkedSignup: true, RiskTrialDay1: true, RiskRelayFree: true, RiskBurstSignups: true,
		RiskBurstThreshold: 3, RiskInviteeFingerprint: true}, s.RiskSettings)

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
