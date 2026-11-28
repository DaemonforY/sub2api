//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// affiliateRepoRateStub: user 100 invited everyone else; inviterRate is user 100's exclusive rate.
type affiliateRepoRateStub struct {
	AffiliateRepository
	inviterRate *float64
	last        float64
}

func (r *affiliateRepoRateStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	if userID == 100 {
		return &AffiliateSummary{UserID: 100, AffRebateRatePercent: r.inviterRate, CreatedAt: time.Now()}, nil
	}
	inviter := int64(100)
	return &AffiliateSummary{UserID: userID, InviterID: &inviter, CreatedAt: time.Now()}, nil
}

func (r *affiliateRepoRateStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}

func (r *affiliateRepoRateStub) AccrueQuota(_ context.Context, _ int64, _ int64, amount float64, _ int, _ *int64) (bool, error) {
	r.last = amount
	return true, nil
}

func newSubscriptionRebateService(values map[string]string, inviterRate *float64) (*AffiliateService, *affiliateRepoRateStub) {
	vals := map[string]string{SettingKeyAffiliateEnabled: "true", SettingKeyAffiliateRebateRate: "20"}
	for k, v := range values {
		vals[k] = v
	}
	settings := NewSettingService(&settingRepoStub{values: vals}, &config.Config{})
	repo := &affiliateRepoRateStub{inviterRate: inviterRate}
	return NewAffiliateService(repo, settings, nil, nil), repo
}

// TestSubscriptionRebate: with 订阅订单返利比例 set to 10, a subscription order earns 10% while a
// top-up keeps the 20% global rate; unset, subscriptions earn the global rate too.
func TestSubscriptionRebate_UsesSubscriptionRate(t *testing.T) {
	ctx := context.Background()

	svc, repo := newSubscriptionRebateService(map[string]string{SettingKeyGrowthSubscriptionRebateRate: "10"}, nil)
	got, err := svc.AccrueInviteRebateForSubscription(ctx, 61, 100, nil)
	require.NoError(t, err)
	require.InDelta(t, 10.0, got, 1e-9)
	require.InDelta(t, 10.0, repo.last, 1e-9)

	got, err = svc.AccrueInviteRebateForOrder(ctx, 61, 100, nil)
	require.NoError(t, err)
	require.InDelta(t, 20.0, got, 1e-9, "top-ups keep the normal rate")

	unset, _ := newSubscriptionRebateService(nil, nil)
	got, err = unset.AccrueInviteRebateForSubscription(ctx, 61, 100, nil)
	require.NoError(t, err)
	require.InDelta(t, 20.0, got, 1e-9, "unset falls back to the global rate")
}

// An inviter with an exclusive rate (an agreed blogger deal) keeps it on subscription orders.
func TestSubscriptionRebate_ExclusiveRateWins(t *testing.T) {
	ctx := context.Background()
	exclusive := 30.0
	svc, _ := newSubscriptionRebateService(map[string]string{SettingKeyGrowthSubscriptionRebateRate: "10"}, &exclusive)
	got, err := svc.AccrueInviteRebateForSubscription(ctx, 61, 100, nil)
	require.NoError(t, err)
	require.InDelta(t, 30.0, got, 1e-9)

	detail := svc.resolveSubscriptionRebateRatePercent(ctx, &AffiliateSummary{AffRebateRatePercent: &exclusive})
	require.InDelta(t, 30.0, detail, 1e-9)
}

func TestParseSubscriptionRebateRate(t *testing.T) {
	for raw, want := range map[string]struct {
		rate float64
		ok   bool
	}{
		"":     {0, false},
		"  ":   {0, false},
		"abc":  {0, false},
		"-1":   {0, false},
		"10":   {10, true},
		"0":    {0, true},
		"150":  {AffiliateRebateRateMax, true},
		"12.5": {12.5, true},
	} {
		rate, ok := parseSubscriptionRebateRate(raw)
		require.Equal(t, want.ok, ok, raw)
		require.InDelta(t, want.rate, rate, 1e-9, raw)
	}
}

// The admin PUT decodes JSON on top of GetSettings' result; that must not change the cache.
func TestGrowthSettingsClone_DoesNotShareSubscriptionRate(t *testing.T) {
	rate := 10.0
	orig := &GrowthSettings{SubscriptionRebateRatePercent: &rate, EduEmailSuffixes: []string{"edu.cn"}}
	cp := orig.clone()
	*cp.SubscriptionRebateRatePercent = 50
	cp.EduEmailSuffixes[0] = "x"
	require.InDelta(t, 10.0, *orig.SubscriptionRebateRatePercent, 1e-9)
	require.Equal(t, "edu.cn", orig.EduEmailSuffixes[0])
}
