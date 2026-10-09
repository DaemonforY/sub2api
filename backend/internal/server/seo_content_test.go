//go:build unit

package server

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type fakePlaza []service.PlazaGroup

func (f fakePlaza) PublicGroups(context.Context) ([]service.PlazaGroup, bool) { return f, true }

func TestSEOModelsApplyGroupMultipliers(t *testing.T) {
	in, out, img := 2e-6, 10e-6, 0.134
	plaza := fakePlaza{{
		Name: "G", RateMultiplier: 1.5, ImageRateIndependent: true, ImageRateMultiplier: 3,
		Models: []service.PlazaModel{
			{Name: "gpt-6.1-sol", Pricing: &service.ChannelModelPricing{BillingMode: service.BillingModeToken, InputPrice: &in, OutputPrice: &out}},
			{Name: "gpt-image-2", Pricing: &service.ChannelModelPricing{BillingMode: service.BillingModeImage,
				Intervals: []service.PricingInterval{{TierLabel: "1K", PerRequestPrice: &img}}}},
		},
	}}
	models, ok := newSEOContent(nil, nil, plaza).Models(context.Background())
	require.True(t, ok)
	require.Len(t, models, 2)
	require.InDelta(t, 3.0, *models[0].Input, 1e-9)
	require.InDelta(t, 15.0, *models[0].Output, 1e-9)
	require.Len(t, models[1].PerImage, 1)
	require.InDelta(t, 0.402, models[1].PerImage[0].Price, 1e-9, "the independent image multiplier, not the group's")
}
