//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fp(v float64) *float64 { return &v }

func TestBuildMarginReport(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	since := now.AddDate(0, 0, -29)
	raw := &MarginRaw{
		Groups: []MarginGroupRow{
			{GroupID: 2, Name: "Codex-Pro", Subscription: true, Users: 3, UsageUSD: 400, BilledUSD: 400},
			{GroupID: 5, Name: "GPT-按量", Users: 6, UsageUSD: 238.33, BilledUSD: 290.99},
		},
		AdminUsageUSD:       700,
		RechargeRevenue:     61.75,
		SubscriptionRevenue: 282,
		Plans: []MarginPlanRow{
			{ID: 1, Name: "月度订阅", Price: 120, ValidityDays: 1, ValidityUnit: "month", DailyLimitUSD: fp(60), WeeklyLimitUSD: fp(350), MonthlyLimitUSD: fp(1200)},
			{ID: 2, Name: "周订阅", Price: 40, ValidityDays: 7, ValidityUnit: "day", DailyLimitUSD: fp(60), WeeklyLimitUSD: fp(350), MonthlyLimitUSD: fp(1200)},
		},
		Subscriptions: []MarginSubscriptionRow{
			// Paid ¥120, $160 in the first 4 days of 30: on pace to the $1,200 cap.
			{ID: 14, UserID: 23, Email: "heavy@example.com", GroupName: "Codex-Pro", StartsAt: now.AddDate(0, 0, -4), ExpiresAt: now.AddDate(0, 0, 26),
				Paid: 120, UsageUSD: 160, DailyLimitUSD: fp(60), WeeklyLimitUSD: fp(350), MonthlyLimitUSD: fp(1200)},
			// Paid ¥108, barely used.
			{ID: 18, UserID: 31, Email: "light@example.com", StartsAt: now.AddDate(0, 0, -2), ExpiresAt: now.AddDate(0, 0, 28), Paid: 108, UsageUSD: 2.5},
			// A 3-day trial nobody paid for, over.
			{ID: 3, UserID: 4, Email: "trial@example.com", StartsAt: now.AddDate(0, 0, -27), ExpiresAt: now.AddDate(0, 0, -24), UsageUSD: 324},
			// Paid ¥54, already past it.
			{ID: 15, UserID: 12, Email: "lite@example.com", StartsAt: now.AddDate(0, 0, -2), ExpiresAt: now.AddDate(0, 0, 28), Paid: 54, UsageUSD: 200, MonthlyLimitUSD: fp(600)},
		},
		Users: []MarginUserRow{{UserID: 23, Email: "heavy@example.com", UsageUSD: 357, Paid: 120}},
	}
	r := BuildMarginReport(raw, 0.3, 30, since, now)
	require.True(t, r.Configured)

	tt := r.Totals
	require.InDelta(t, 343.75, tt.Revenue, 1e-9)
	require.InDelta(t, 120, tt.SubscriptionCost, 1e-9)
	require.InDelta(t, 71.5, tt.PayGoCost, 1e-9)
	require.InDelta(t, 290.99, tt.PayGoBilled, 1e-9)
	require.InDelta(t, 210, tt.AdminCost, 1e-9)
	require.InDelta(t, 191.5, tt.MemberCost, 1e-9)
	require.InDelta(t, 152.25, tt.Margin, 1e-9)
	require.Nil(t, r.Groups[0].Margin)
	require.InDelta(t, 219.49, *r.Groups[1].Margin, 1e-9)

	month, week := r.Plans[0], r.Plans[1]
	require.Equal(t, 30, month.Days)
	require.InDelta(t, 1200, *month.CapUSD, 1e-9)
	require.InDelta(t, 360, *month.MaxCost, 1e-9)
	require.InDelta(t, 240, month.MaxLoss, 1e-9)
	require.InDelta(t, 400, month.BreakEvenUSD, 1e-9)
	require.InDelta(t, 350, *week.CapUSD, 1e-9) // one week's allowance, not a month's
	require.InDelta(t, 65, week.MaxLoss, 1e-9)

	byID := map[int64]MarginSubscription{}
	for _, s := range r.Subscriptions {
		byID[s.ID] = s
	}
	heavy := byID[14]
	require.True(t, heavy.Active)
	require.Equal(t, MarginStatusRisk, heavy.Status)
	require.InDelta(t, 72, heavy.Margin, 1e-9)
	require.InDelta(t, 1200, heavy.ProjectedUSD, 1e-9) // pace 40/day × 30 = 1200, at the cap
	require.InDelta(t, -240, heavy.ProjectedMargin, 1e-9)
	require.NotContains(t, heavy.Email, "heavy")
	require.Equal(t, MarginStatusOK, byID[18].Status)
	require.Equal(t, MarginStatusGift, byID[3].Status)
	require.False(t, byID[3].Active)
	require.InDelta(t, 324, byID[3].ProjectedUSD, 1e-9)
	lite := byID[15]
	require.Equal(t, MarginStatusLoss, lite.Status)
	require.InDelta(t, 600, lite.ProjectedUSD, 1e-9) // pace 100/day, held to the monthly cap
	// Worst projected margin first.
	require.Equal(t, int64(14), r.Subscriptions[0].ID)
	require.InDelta(t, 12.9, r.Users[0].Margin, 1e-9)

	// Without a cost rate the figures stay at zero cost and the page asks for it.
	r0 := BuildMarginReport(raw, 0, 30, since, now)
	require.False(t, r0.Configured)
	require.Zero(t, r0.Totals.MemberCost)
	require.Nil(t, r0.Plans[0].MaxCost)
}
