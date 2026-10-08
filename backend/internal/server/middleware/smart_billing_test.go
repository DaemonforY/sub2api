//go:build unit

package middleware

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSmartBillingUpgradePath(t *testing.T) {
	for _, p := range []string{"/v1/messages", "/v1/chat/completions", "/v1/responses", "/v1/responses/compact", "/responses", "/backend-api/codex/responses", "/v1beta/models/gemini-3:generateContent"} {
		require.True(t, smartBillingUpgradePath(p), p)
	}
	for _, p := range []string{"/v1/images/generations", "/v1/embeddings", "/v1/videos", "/v1/models", "/v1/messagesx"} {
		require.False(t, smartBillingUpgradePath(p), p)
	}
}

func TestSmartBillingMessages(t *testing.T) {
	limit := 60.0
	reset := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	sub := &service.UserSubscription{DailyUsageUSD: 60.76, DailyWindowStart: &reset}
	msg := subscriptionLimitMessage(service.ErrDailyLimitExceeded, sub, &service.Group{DailyLimitUSD: &limit})
	require.Contains(t, msg, "急用可换一个按量计费分组的 Key")
	smart := smartBillingLimitMessage(msg)
	require.NotContains(t, smart, "换一个按量")
	require.Contains(t, smart, "充值后这个 Key 会自动改用余额继续")
	require.Contains(t, smart, "daily usage limit exceeded")
	require.Contains(t, subscriptionNotFoundMessage(true), "充值后这个 Key 会自动按量计费")
	require.Equal(t, msgSubscriptionNotFound, subscriptionNotFoundMessage(false))
}
