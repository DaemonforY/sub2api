//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestUserShadow_FreeCreditCapLooksLikeANormalLimit(t *testing.T) {
	ctx := context.Background()
	svc, repo := newShadowServiceForTest(growthForTest(nil)) // $0.3 a day

	// Not restricted: never capped.
	repo.spent[7] = 50
	require.NoError(t, svc.CheckFreeCredit(ctx, 7, false, nil, nil))

	_, err := svc.Shadow(ctx, 7, "same browser as #48", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)

	repo.spent[7] = 0.29
	require.NoError(t, svc.CheckFreeCredit(ctx, 7, false, nil, nil), "under the daily allowance")

	// Balance: exactly the error a normal user with no balance gets.
	repo.spent[7] = 0.3
	normalCache := &balanceEligibilityCacheStub{balance: 0}
	normal := NewBillingCacheService(normalCache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(normal.Stop)
	normalErr := normal.CheckBillingEligibility(ctx, &User{ID: 99}, nil, nil, nil, "")
	require.Error(t, normalErr)

	richCache := &balanceEligibilityCacheStub{balance: 1.7} // the gifted $2, partly used
	gated := NewBillingCacheService(richCache, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(gated.Stop)
	require.NoError(t, gated.CheckBillingEligibility(ctx, &User{ID: 7}, nil, nil, nil, ""), "no gate installed yet")
	gated.SetFreeCreditGate(svc)
	shadowErr := gated.CheckBillingEligibility(ctx, &User{ID: 7}, nil, nil, nil, "")
	require.Equal(t, normalErr, shadowErr, "same error value, code and text as a normal user's 余额不足")
	require.NoError(t, gated.CheckBillingEligibility(ctx, &User{ID: 8}, nil, nil, nil, ""), "other users are untouched")

	// Subscription: the normal daily-limit error, and the day's usage is filled to the limit so the
	// usage page and the error message agree.
	limit := 35.0
	group := &Group{ID: 8, DailyLimitUSD: &limit}
	sub := &UserSubscription{ID: 501, UserID: 7, GroupID: 8, DailyUsageUSD: 0.3}
	err = svc.CheckFreeCredit(ctx, 7, true, group, sub)
	require.Equal(t, ErrDailyLimitExceeded, err)
	require.True(t, strings.Contains(err.Error(), "今日额度已用完"))
	require.Equal(t, 35.0, sub.DailyUsageUSD)
	require.Equal(t, 35.0, repo.filled[501])

	// Paid money is not capped.
	repo.paid[7] = true
	svc.paid.Delete(int64(7))
	require.NoError(t, svc.CheckFreeCredit(ctx, 7, false, nil, nil))

	// Lifting the restriction lifts the cap.
	repo.paid[7] = false
	svc.paid.Delete(int64(7))
	require.Error(t, svc.CheckFreeCredit(ctx, 7, false, nil, nil))
	require.NoError(t, svc.Unshadow(ctx, 7))
	require.NoError(t, svc.CheckFreeCredit(ctx, 7, false, nil, nil))
}

func TestUserShadow_ReasonRequiredAndReload(t *testing.T) {
	ctx := context.Background()
	svc, repo := newShadowServiceForTest(nil)
	_, err := svc.Shadow(ctx, 7, "  ", ShadowSourceAdmin, nil, nil)
	require.ErrorIs(t, err, ErrShadowReasonRequired)

	repo.shadows[9] = UserShadow{UserID: 9, Reason: "set elsewhere", Source: ShadowSourceAdmin, CreatedAt: time.Now()}
	require.False(t, svc.IsShadowed(9))
	svc.reload(ctx)
	require.True(t, svc.IsShadowed(9))
	require.Len(t, svc.Lookup([]int64{9, 10}), 1)
}

type growthRepoShadowStub struct {
	GrowthRepository
	signupGrants int
}

func (r *growthRepoShadowStub) GrantInviteeSignupBonus(context.Context, int64, int64, float64, time.Time, time.Time, int, int) (bool, error) {
	r.signupGrants++
	return true, nil
}

type affiliateRepoShadowStub struct {
	AffiliateRepository
	accrued int
}

func (r *affiliateRepoShadowStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	inviter := int64(100)
	if userID == 100 {
		return &AffiliateSummary{UserID: 100, CreatedAt: time.Now()}, nil
	}
	return &AffiliateSummary{UserID: userID, InviterID: &inviter, CreatedAt: time.Now()}, nil
}

func (r *affiliateRepoShadowStub) GetAccruedRebateFromInvitee(context.Context, int64, int64) (float64, error) {
	return 0, nil
}

func (r *affiliateRepoShadowStub) AccrueQuota(context.Context, int64, int64, float64, int, *int64) (bool, error) {
	r.accrued++
	return true, nil
}

func TestUserShadow_WithholdsInviteBonusesAndRebates(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingService(&settingRepoStub{values: map[string]string{SettingKeyAffiliateEnabled: "true"}}, &config.Config{})
	shadow, _ := newShadowServiceForTest(nil)

	growthRepo := &growthRepoShadowStub{}
	growth := growthForTest(func(s *GrowthSettings) { s.InviteeSignupBonus = 1; s.InviteeBonusRatePercent = 20 })
	growth.repo, growth.settingService = growthRepo, settings
	growth.SetShadowChecker(shadow)

	affRepo := &affiliateRepoShadowStub{}
	aff := NewAffiliateService(affRepo, settings, nil, nil)
	aff.SetShadowChecker(shadow)

	// Normal accounts get both.
	amount, err := growth.GrantInviteeSignupBonus(ctx, 61, 100)
	require.NoError(t, err)
	require.Equal(t, 1.0, amount)
	rebate, err := aff.AccrueInviteRebate(ctx, 61, 10)
	require.NoError(t, err)
	require.Positive(t, rebate)
	require.Equal(t, 1, growthRepo.signupGrants)
	require.Equal(t, 1, affRepo.accrued)

	// A restricted invitee: silently nothing (no error either).
	_, err = shadow.Shadow(ctx, 62, "auto", ShadowSourceAuto, nil, nil)
	require.NoError(t, err)
	amount, err = growth.GrantInviteeSignupBonus(ctx, 62, 100)
	require.NoError(t, err)
	require.Zero(t, amount)
	bonus, err := growth.GrantInviteeFirstOrderBonus(ctx, 62, 9001, 50)
	require.NoError(t, err)
	require.Zero(t, bonus)
	rebate, err = aff.AccrueInviteRebate(ctx, 62, 10)
	require.NoError(t, err)
	require.Zero(t, rebate)

	// A restricted inviter: its invitees' bonus and rebates are withheld too.
	_, err = shadow.Shadow(ctx, 100, "farm", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)
	amount, err = growth.GrantInviteeSignupBonus(ctx, 61, 100)
	require.NoError(t, err)
	require.Zero(t, amount)
	rebate, err = aff.AccrueInviteRebate(ctx, 61, 10)
	require.NoError(t, err)
	require.Zero(t, rebate)

	require.Equal(t, 1, growthRepo.signupGrants, "nothing else reached the repository")
	require.Equal(t, 1, affRepo.accrued)
}
