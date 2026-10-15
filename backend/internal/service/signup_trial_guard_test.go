//go:build unit

package service

import (
	"context"
	"errors"
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
}

func (r *signupFingerprintRepoFake) FindRecentTrial(_ context.Context, visitorID, ipPrefix string, since time.Time) (int64, error) {
	if r.findErr != nil {
		return 0, r.findErr
	}
	for i := len(r.rows) - 1; i >= 0; i-- {
		row := r.rows[i]
		if !row.TrialGranted || row.at.Before(since) {
			continue
		}
		if (row.VisitorID != "" && row.VisitorID == visitorID) || (row.IPPrefix != "" && row.IPPrefix == ipPrefix) {
			return row.UserID, nil
		}
	}
	return 0, nil
}

func (r *signupFingerprintRepoFake) Save(_ context.Context, fp SignupFingerprint) error {
	r.rows = append(r.rows, storedSignupFingerprint{SignupFingerprint: fp, at: r.now()})
	return nil
}

func newSignupTrialGuardForTest(days int) (*SignupTrialGuard, *signupFingerprintRepoFake, *time.Time) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	repo := &signupFingerprintRepoFake{now: func() time.Time { return now }}
	growth := &GrowthService{cached: &GrowthSettings{SignupTrialDedupDays: days}, cachedAt: time.Now()}
	g := NewSignupTrialGuard(repo, growth)
	g.now = func() time.Time { return now }
	return g, repo, &now
}

func signupCtx(visitorID, ip string) context.Context {
	return WithSignupClient(context.Background(), SignupClient{VisitorID: visitorID, IP: ip})
}

func TestSignupTrialGuard_OneTrialPerBrowserAndIPPrefix(t *testing.T) {
	g, repo, _ := newSignupTrialGuardForTest(30)

	require.True(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true), "first sign-up gets the trial")
	require.False(t, g.Allow(signupCtx("visitor-aaaaaaaa", "198.51.100.9"), 2, true), "same browser, other network")
	require.False(t, g.Allow(signupCtx("visitor-bbbbbbbb", "203.0.113.200"), 3, true), "other browser, same /24")
	require.False(t, g.Allow(signupCtx("", "203.0.113.50"), 4, true), "no visitor ID, same /24")
	require.True(t, g.Allow(signupCtx("visitor-cccccccc", "198.51.100.9"), 5, true),
		"user 2 never got a trial, so its network does not block others")

	require.Len(t, repo.rows, 5)
	require.Equal(t, "203.0.113.0", repo.rows[0].IPPrefix, "only the /24 is stored")
	require.Equal(t, "visitor-aaaaaaaa", repo.rows[0].VisitorID)
	granted := []bool{}
	for _, r := range repo.rows {
		granted = append(granted, r.TrialGranted)
	}
	require.Equal(t, []bool{true, false, false, false, true}, granted)
}

func TestSignupTrialGuard_WindowAndOff(t *testing.T) {
	g, _, now := newSignupTrialGuardForTest(30)
	require.True(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true))

	*now = now.AddDate(0, 0, 29)
	require.False(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 2, true), "29 days later is still inside the window")

	*now = now.AddDate(0, 0, 2)
	require.True(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 3, true), "31 days after the trial it can be granted again")

	off, repo, _ := newSignupTrialGuardForTest(0)
	require.True(t, off.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, true))
	require.True(t, off.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 2, true), "0 days turns the dedup off")
	require.Len(t, repo.rows, 2, "sign-ups are still recorded while off")
}

func TestSignupTrialGuard_PrivateIPsAndBadVisitorIDsNeverMatch(t *testing.T) {
	g, repo, _ := newSignupTrialGuardForTest(30)
	// A proxy that hides the client IP must not make every sign-up look like one network.
	require.True(t, g.Allow(signupCtx("", "172.18.0.1"), 1, true))
	require.True(t, g.Allow(signupCtx("", "172.18.0.1"), 2, true))
	require.True(t, g.Allow(signupCtx("", "127.0.0.1"), 3, true))
	require.True(t, g.Allow(signupCtx("x", "10.0.0.8"), 4, true), "visitor IDs that aren't analytics IDs are ignored")
	require.True(t, g.Allow(signupCtx("x", "10.0.0.9"), 5, true))
	for _, r := range repo.rows {
		require.Empty(t, r.IPPrefix)
		require.Empty(t, r.VisitorID)
	}

	// IPv6 is grouped by /48.
	require.True(t, g.Allow(signupCtx("", "2001:db8:1:2::1"), 6, true))
	require.False(t, g.Allow(signupCtx("", "2001:db8:1:ffff::9"), 7, true))
}

func TestSignupTrialGuard_NoTrialConfiguredAndFailOpen(t *testing.T) {
	g, repo, _ := newSignupTrialGuardForTest(30)
	require.False(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 1, false), "nothing to grant")
	require.Len(t, repo.rows, 1)
	require.False(t, repo.rows[0].TrialGranted)
	require.True(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 2, true),
		"a sign-up that got no trial doesn't block the next one")

	repo.findErr = errors.New("db down")
	require.True(t, g.Allow(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), 3, true), "lookup errors fail open")
}

func TestParseGrowthSettings_SignupTrialDedupDays(t *testing.T) {
	require.Equal(t, 30, parseGrowthSettings(map[string]string{}).SignupTrialDedupDays, "on by default")
	require.Equal(t, 0, parseGrowthSettings(map[string]string{SettingKeyGrowthSignupTrialDedupDays: "0"}).SignupTrialDedupDays)
	require.Equal(t, 7, parseGrowthSettings(map[string]string{SettingKeyGrowthSignupTrialDedupDays: "7"}).SignupTrialDedupDays)
	require.Equal(t, 365, parseGrowthSettings(map[string]string{SettingKeyGrowthSignupTrialDedupDays: "9999"}).SignupTrialDedupDays)
}

func TestAuthService_Register_SkipsSignupSubscriptionForRepeatBrowser(t *testing.T) {
	repo := &userRepoStub{nextID: 42}
	assigner := &defaultSubscriptionAssignerStub{}
	svc := newAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled:                 "true",
		SettingKeyDefaultBalance:                      "2",
		SettingKeyDefaultSubscriptions:                `[{"group_id":8,"validity_days":3}]`,
		SettingKeyAuthSourceDefaultEmailGrantOnSignup: "false",
	}, nil, nil)
	svc.defaultSubAssigner = assigner
	guard, fps, _ := newSignupTrialGuardForTest(30)
	svc.SetSignupTrialGuard(guard)

	_, first, err := svc.RegisterWithVerification(signupCtx("visitor-aaaaaaaa", "203.0.113.7"), "a1@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, 2.0, first.Balance)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(8), assigner.calls[0].GroupID)

	repo.nextID = 43
	_, second, err := svc.RegisterWithVerification(signupCtx("visitor-aaaaaaaa", "198.51.100.1"), "a2@test.com", "password", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, 2.0, second.Balance, "the repeat sign-up still gets the default balance")
	require.Len(t, assigner.calls, 1, "but not another trial subscription")
	require.Len(t, fps.rows, 2)
	require.Equal(t, int64(43), fps.rows[1].UserID)
	require.False(t, fps.rows[1].TrialGranted)
}
