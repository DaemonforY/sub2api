//go:build unit

package service

import (
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func quotaResetTestGroup(platform string) *Group {
	d, w, m := 40.0, 350.0, 800.0
	return &Group{ID: 2, Name: "Codex-Pro", Platform: platform, DailyLimitUSD: &d, WeeklyLimitUSD: &w, MonthlyLimitUSD: &m}
}

func quotaResetTestSub(now time.Time, daily, weekly, monthly float64) *UserSubscription {
	today := timezone.StartOfDay(now)
	start := now.Add(-3 * 24 * time.Hour)
	return &UserSubscription{
		ID: 87, UserID: 84, GroupID: 2, Status: SubscriptionStatusActive,
		StartsAt: start, ExpiresAt: now.Add(20 * 24 * time.Hour),
		DailyWindowStart: &today, WeeklyWindowStart: &start, MonthlyWindowStart: &start,
		DailyUsageUSD: daily, WeeklyUsageUSD: weekly, MonthlyUsageUSD: monthly,
	}
}

func TestBuildQuotaResetQuote(t *testing.T) {
	now := time.Now()
	cfg := &PaymentConfig{Enabled: true, BalanceRechargeMultiplier: 1}
	settings := DefaultQuotaResetSettings()
	group := quotaResetTestGroup(PlatformOpenAI)
	until := timezone.StartOfDay(now).AddDate(0, 0, 1)

	t.Run("daily used up can buy at daily limit x rate", func(t *testing.T) {
		q := buildQuotaResetQuote(quotaResetTestSub(now, 40.47, 78, 258), group, settings, cfg, now)
		require.True(t, q.CanBuy)
		require.True(t, q.Blocked)
		require.Equal(t, "", q.Reason)
		require.Equal(t, 20.0, q.Price)
		require.Equal(t, 20.0, q.BalancePrice)
		require.Equal(t, 40.0, q.DailyLimitUSD)
	})
	t.Run("usable subscription is offered later", func(t *testing.T) {
		q := buildQuotaResetQuote(quotaResetTestSub(now, 10, 78, 258), group, settings, cfg, now)
		require.False(t, q.CanBuy)
		require.True(t, q.Available)
		require.Equal(t, QuotaResetReasonNotBlocked, q.Reason)
	})
	t.Run("weekly used up also unblocks with a reset", func(t *testing.T) {
		q := buildQuotaResetQuote(quotaResetTestSub(now, 5, 350, 400), group, settings, cfg, now)
		require.True(t, q.CanBuy)
	})
	t.Run("reset day ignores weekly and monthly limits", func(t *testing.T) {
		sub := quotaResetTestSub(now, 5, 400, 900)
		sub.DailyResetUntil, sub.DailyResetCount = &until, 1
		q := buildQuotaResetQuote(sub, group, settings, cfg, now)
		require.False(t, q.Blocked)
		require.True(t, q.ResetActive)
		require.Equal(t, 1, q.ResetsToday)
		require.Equal(t, QuotaResetReasonMaxReached, q.Reason)
	})
	t.Run("second reset allowed when max per day is 2", func(t *testing.T) {
		sub := quotaResetTestSub(now, 40, 400, 900)
		sub.DailyResetUntil, sub.DailyResetCount = &until, 1
		two := *settings
		two.MaxPerDay = 2
		q := buildQuotaResetQuote(sub, group, &two, cfg, now)
		require.True(t, q.CanBuy)
	})
	t.Run("expired reset day counts as no resets", func(t *testing.T) {
		sub := quotaResetTestSub(now, 40, 100, 100)
		past := now.Add(-time.Hour)
		sub.DailyResetUntil, sub.DailyResetCount = &past, 3
		q := buildQuotaResetQuote(sub, group, settings, cfg, now)
		require.True(t, q.CanBuy)
		require.Equal(t, 0, q.ResetsToday)
		require.False(t, q.ResetActive)
	})
	t.Run("platform not sold", func(t *testing.T) {
		q := buildQuotaResetQuote(quotaResetTestSub(now, 50, 100, 100), quotaResetTestGroup(PlatformAnthropic), settings, cfg, now)
		require.False(t, q.Available)
		require.Equal(t, QuotaResetReasonNotSupported, q.Reason)
	})
	t.Run("disabled", func(t *testing.T) {
		off := *settings
		off.Enabled = false
		q := buildQuotaResetQuote(quotaResetTestSub(now, 50, 100, 100), group, &off, cfg, now)
		require.Equal(t, QuotaResetReasonDisabled, q.Reason)
		require.False(t, q.CanBuy)
	})
	t.Run("one-day trial excluded", func(t *testing.T) {
		sub := quotaResetTestSub(now, 50, 50, 50)
		sub.StartsAt = now.Add(-time.Hour)
		sub.ExpiresAt = now.Add(23 * time.Hour)
		q := buildQuotaResetQuote(sub, group, settings, cfg, now)
		require.Equal(t, QuotaResetReasonOneTime, q.Reason)
	})
	t.Run("yesterday's usage does not block today", func(t *testing.T) {
		sub := quotaResetTestSub(now, 45, 100, 100)
		yesterday := timezone.StartOfDay(now).AddDate(0, 0, -1)
		sub.DailyWindowStart = &yesterday
		q := buildQuotaResetQuote(sub, group, settings, cfg, now)
		require.False(t, q.Blocked)
	})
}

func TestQuotaResetPriceRounding(t *testing.T) {
	require.Equal(t, 12.5, quotaResetPrice(25, 0.5))
	require.Equal(t, 6.67, quotaResetPrice(13.333, 0.5))
	require.Equal(t, "20", formatCNY(20))
	require.Equal(t, "12.5", formatCNY(12.5))
}

func TestSubscriptionLimitsDuringResetDay(t *testing.T) {
	now := time.Now()
	group := quotaResetTestGroup(PlatformOpenAI)
	sub := quotaResetTestSub(now, 10, 400, 900)
	require.False(t, sub.CheckWeeklyLimit(group, 0))
	require.False(t, sub.CheckMonthlyLimit(group, 0))

	until := timezone.StartOfDay(now).AddDate(0, 0, 1)
	sub.DailyResetUntil = &until
	require.True(t, sub.CheckWeeklyLimit(group, 0))
	require.True(t, sub.CheckMonthlyLimit(group, 0))
	require.True(t, sub.CheckDailyLimit(group, 0))
	sub.DailyUsageUSD = 40
	require.False(t, sub.CheckDailyLimit(group, 0), "the daily limit still applies on a reset day")

	cache := &subscriptionCacheData{ResetUntil: until.Unix()}
	require.True(t, cache.dailyResetActive(now))
	require.False(t, cache.dailyResetActive(until.Add(time.Second)))
}

func TestQuotaResetSettingsNormalize(t *testing.T) {
	s := &QuotaResetSettings{MaxPerDay: 0, Platforms: []string{" OpenAI ", "openai", "", "anthropic"}}
	s.normalize()
	require.Equal(t, 1, s.MaxPerDay)
	require.Equal(t, []string{"openai", "anthropic"}, s.Platforms)
}

func TestQuotaResetWeChatText(t *testing.T) {
	now := time.Now()
	group := quotaResetTestGroup(PlatformOpenAI)
	q := QuotaResetQuote{GroupID: 2, Price: 20, DailyLimitUSD: 40}

	text := QuotaResetWeChatText(quotaResetTestSub(now, 41, 100, 100), group, q, now)
	require.Contains(t, text, "今日额度已用完，0 点自动恢复")
	require.Contains(t, text, "¥20")
	require.Contains(t, text, "/purchase?quota_reset=2")
	require.Contains(t, text, "不计入周/月额度")

	weekly := QuotaResetWeChatText(quotaResetTestSub(now, 5, 350, 400), group, q, now)
	require.Contains(t, weekly, "本周额度已用完")
	require.False(t, strings.Contains(weekly, "0 点自动恢复"))
}
