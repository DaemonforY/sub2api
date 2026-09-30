//go:build unit

package middleware

import (
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionLimitMessage(t *testing.T) {
	limit := 10.0
	windowStart := time.Date(2026, 9, 30, 0, 0, 0, 0, time.Local)
	sub := &service.UserSubscription{
		StartsAt:         windowStart.AddDate(0, 0, -3),
		ExpiresAt:        windowStart.AddDate(0, 0, 27),
		DailyWindowStart: &windowStart,
		DailyUsageUSD:    10.5,
	}
	group := &service.Group{DailyLimitUSD: &limit}

	msg := subscriptionLimitMessage(service.ErrDailyLimitExceeded, sub, group)
	require.Contains(t, msg, "订阅今日额度已用完")
	require.Contains(t, msg, "已用 $10.50 / 额度 $10.00")
	require.Contains(t, msg, "将于 "+formatUserTime(*sub.DailyResetTime())+" 重置")
	require.Contains(t, msg, "daily usage limit exceeded")

	require.Contains(t, subscriptionLimitMessage(service.ErrSubscriptionExpired, sub, group), "到期")
	require.Contains(t, subscriptionLimitMessage(service.ErrSubscriptionSuspended, sub, group), "暂停")

	// Weekly window not activated and no limit configured: still a readable sentence.
	weekly := subscriptionLimitMessage(service.ErrWeeklyLimitExceeded, sub, group)
	require.Contains(t, weekly, "订阅本周额度已用完，等额度重置后再试")

	other := errors.New("something else")
	require.Equal(t, "something else", subscriptionLimitMessage(other, sub, group))
	require.Equal(t, service.ErrDailyLimitExceeded.Error(), subscriptionLimitMessage(service.ErrDailyLimitExceeded, nil, group))
}

func TestInsufficientBalanceMessage(t *testing.T) {
	msg := insufficientBalanceMessage(0.123)
	require.Contains(t, msg, "当前余额 0.12")
	require.Contains(t, msg, "Insufficient account balance")
}
