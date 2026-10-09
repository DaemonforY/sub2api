//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/stretchr/testify/require"
)

func TestOpenAIEstimateImageCost(t *testing.T) {
	ctx := context.Background()
	svc := &OpenAIGatewayService{billingService: &BillingService{}}
	gid := int64(3)

	// Default gpt-image-2 price at 1536x1024 (2K tier), as billed on hivegpt.cn ($0.2010 a picture).
	price, ok := svc.EstimateImageCost(ctx, &APIKey{UserID: 5, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: 1}}, "gpt-image-2", "1536x1024")
	require.True(t, ok)
	require.InDelta(t, 0.201, price, 0.0001)

	// The group's own image price and its multiplier.
	p2k := 0.05
	price, ok = svc.EstimateImageCost(ctx, &APIKey{UserID: 5, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: 2, ImagePrice2K: &p2k}}, "gpt-image-2", "1536x1024")
	require.True(t, ok)
	require.InDelta(t, 0.10, price, 0.0001)

	// An independent image multiplier replaces the group's.
	price, ok = svc.EstimateImageCost(ctx, &APIKey{UserID: 5, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: 2, ImagePrice2K: &p2k,
		ImageRateIndependent: true, ImageRateMultiplier: 0.5}}, "gpt-image-2", "1536x1024")
	require.True(t, ok)
	require.InDelta(t, 0.025, price, 0.0001)

	_, ok = svc.EstimateImageCost(ctx, &APIKey{UserID: 5}, "gpt-image-2", "1536x1024")
	require.False(t, ok, "no group, no quote")
}

func TestOpenAIEstimateTextCost(t *testing.T) {
	ctx := context.Background()
	svc := &OpenAIGatewayService{billingService: NewBillingService(&config.Config{}, nil)}
	gid := int64(3)
	key := &APIKey{UserID: 5, GroupID: &gid, Group: &Group{ID: gid, RateMultiplier: 1}}
	full, ok := svc.EstimateTextCost(ctx, key, "gpt-5.5", UsageTokens{InputTokens: 10000, OutputTokens: 600})
	require.True(t, ok, "gpt-5.5 has a built-in price")
	cached, ok := svc.EstimateTextCost(ctx, key, "gpt-5.5", UsageTokens{InputTokens: 800, CacheReadTokens: 9200, OutputTokens: 600})
	require.True(t, ok)
	require.Less(t, cached, full, "cached input is cheaper")
	key.Group.RateMultiplier = 2
	double, _ := svc.EstimateTextCost(ctx, key, "gpt-5.5", UsageTokens{InputTokens: 10000, OutputTokens: 600})
	require.InDelta(t, full*2, double, 1e-9, "the group multiplier applies")
	_, ok = svc.EstimateTextCost(ctx, &APIKey{UserID: 5}, "gpt-5.5", UsageTokens{InputTokens: 1})
	require.False(t, ok)
}
