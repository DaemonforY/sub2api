//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionLimitChecksTreatExactLimitAsUsedUp(t *testing.T) {
	limit := 10.0
	group := &Group{DailyLimitUSD: &limit, WeeklyLimitUSD: &limit, MonthlyLimitUSD: &limit}
	cases := []struct {
		name       string
		usage, add float64
		want       bool
	}{
		{"below limit", 9.99, 0, true},
		{"exactly at limit", 10, 0, false},
		{"over limit", 10.01, 0, false},
		{"cost fills the rest exactly", 9, 1, true},
		{"cost overshoots", 9.5, 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := &UserSubscription{DailyUsageUSD: tc.usage, WeeklyUsageUSD: tc.usage, MonthlyUsageUSD: tc.usage}
			require.Equal(t, tc.want, sub.CheckDailyLimit(group, tc.add), "daily")
			require.Equal(t, tc.want, sub.CheckWeeklyLimit(group, tc.add), "weekly")
			require.Equal(t, tc.want, sub.CheckMonthlyLimit(group, tc.add), "monthly")
		})
	}

	unlimited := &UserSubscription{DailyUsageUSD: 1e9, WeeklyUsageUSD: 1e9, MonthlyUsageUSD: 1e9}
	require.True(t, unlimited.CheckDailyLimit(&Group{}, 1))
	require.True(t, unlimited.CheckWeeklyLimit(&Group{}, 1))
	require.True(t, unlimited.CheckMonthlyLimit(&Group{}, 1))
}

type atCapSubRepo struct {
	userSubRepoNoop
	subs []UserSubscription
}

func (r atCapSubRepo) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	for i := range r.subs {
		if r.subs[i].UserID == userID && r.subs[i].GroupID == groupID {
			cp := r.subs[i]
			return &cp, nil
		}
	}
	return nil, ErrSubscriptionNotFound
}

func (r atCapSubRepo) ListActiveByUserID(_ context.Context, userID int64) ([]UserSubscription, error) {
	var out []UserSubscription
	for _, s := range r.subs {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

type atCapGroupRepo struct {
	groupRepoNoop
	groups []Group
}

func (r atCapGroupRepo) ListActive(context.Context) ([]Group, error) { return r.groups, nil }

// A subscription whose usage sits exactly on a cap is rejected by billing (>=), so smart billing
// must not keep the request there: it moves to the user's other subscription, or to
// pay-as-you-go when there is none. Before the fix the request stayed and got a 403.
func TestBillingRouteMovesOnWhenOwnSubscriptionIsExactlyAtLimit(t *testing.T) {
	daily, weekly, monthly := 40.0, 350.0, 800.0
	oldGroup := Group{ID: 2, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &monthly, SortOrder: 10}
	newGroup := Group{ID: 12, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly, MonthlyLimitUSD: &monthly, SortOrder: 10}
	payg := Group{ID: 5, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, SortOrder: 20}

	now := time.Now()
	// Start of today: an earlier start would cross midnight and trigger a daily window reset.
	windowStart := startOfDay(now)
	sub := func(groupID int64, dailyUsed, weeklyUsed, monthlyUsed float64) UserSubscription {
		return UserSubscription{
			UserID: 1, GroupID: groupID, Status: SubscriptionStatusActive,
			StartsAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(10 * 24 * time.Hour),
			DailyWindowStart: &windowStart, WeeklyWindowStart: &windowStart, MonthlyWindowStart: &windowStart,
			DailyUsageUSD: dailyUsed, WeeklyUsageUSD: weeklyUsed, MonthlyUsageUSD: monthlyUsed,
		}
	}
	atCap := map[string]UserSubscription{
		"daily":   sub(oldGroup.ID, daily, daily, daily),
		"weekly":  sub(oldGroup.ID, 0, weekly, weekly),
		"monthly": sub(oldGroup.ID, 0, 0, monthly),
	}

	route := func(subs []UserSubscription, balanceOK bool) *Group {
		subSvc := NewSubscriptionService(groupRepoNoop{}, atCapSubRepo{subs: subs}, nil, nil, nil)
		svc := NewBillingRouteService(subSvc, atCapGroupRepo{groups: []Group{oldGroup, newGroup, payg}}, &settingRepoStub{})
		key := &APIKey{Group: &oldGroup, User: &User{ID: 1}}
		return svc.Route(context.Background(), key, false, balanceOK)
	}

	for kind, own := range atCap {
		t.Run(kind+" cap, another subscription", func(t *testing.T) {
			g := route([]UserSubscription{own, sub(newGroup.ID, 0, 0, 0)}, true)
			require.NotNil(t, g)
			require.Equal(t, newGroup.ID, g.ID)
		})
		t.Run(kind+" cap, pay-as-you-go fallback", func(t *testing.T) {
			g := route([]UserSubscription{own}, true)
			require.NotNil(t, g)
			require.Equal(t, payg.ID, g.ID)
		})
	}

	t.Run("just under the cap keeps the key's own group", func(t *testing.T) {
		require.Nil(t, route([]UserSubscription{sub(oldGroup.ID, daily-0.01, weekly-0.01, monthly-0.01)}, true))
	})
}
