package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// The shipped HiveGPT patch puts gpt-5.6-sol (and its alias gpt-5.6) on OpenAI's promotional price,
// including the >272K tier, over a catalog that still lists the launch price.
func TestHiveGPTOverridesPriceGPT56AtPromotion(t *testing.T) {
	svc := &PricingService{cfg: &config.Config{}}
	svc.cfg.Pricing.OverrideFile = "../../resources/model-pricing/hivegpt_overrides.json"
	data, err := svc.parsePricingData([]byte(`{
		"gpt-5.6-sol": {"litellm_provider": "openai", "mode": "chat",
			"input_cost_per_token": 5e-06, "output_cost_per_token": 3e-05, "cache_read_input_token_cost": 5e-07,
			"input_cost_per_token_above_272k_tokens": 1e-05, "output_cost_per_token_above_272k_tokens": 4.5e-05}
	}`))
	require.NoError(t, err)
	p := data["gpt-5.6-sol"]
	require.NotNil(t, p)
	require.InDelta(t, 4e-6, p.InputCostPerToken, 1e-12)
	require.InDelta(t, 2e-5, p.OutputCostPerToken, 1e-12)
	require.InDelta(t, 4e-7, p.CacheReadInputTokenCost, 1e-12)

	svc.pricingData = data
	billing := NewBillingService(&config.Config{}, svc)
	for _, model := range []string{"gpt-5.6-sol", "gpt-5.6"} {
		cost, err := billing.CalculateCost(model, UsageTokens{InputTokens: 1000, OutputTokens: 1000}, 1)
		require.NoError(t, err)
		require.InDelta(t, 1000*4e-6+1000*2e-5, cost.TotalCost, 1e-10, model)
		long, err := billing.CalculateCost(model, UsageTokens{InputTokens: 300000, OutputTokens: 1000}, 1)
		require.NoError(t, err)
		require.InDelta(t, 300000*8e-6+1000*3e-5, long.TotalCost, 1e-9, model+" >272K")
	}
}
