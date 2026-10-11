//go:build unit

package service

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestStarterUnitCost(t *testing.T) {
	// 131 images at $26.33 standard, billed at 3× now: about $0.60 an image.
	require.InDelta(t, 0.603, starterUnitCost(26.331, 131, starterMinImages, 3), 0.001)
	require.Zero(t, starterUnitCost(2.0, 10, starterMinImages, 3), "too few samples to quote")
	require.Zero(t, starterUnitCost(0, 100, starterMinImages, 3))
	require.Zero(t, starterUnitCost(5, 100, starterMinImages, 0), "free groups quote nothing")
}

func TestGroupImageMultiplier(t *testing.T) {
	require.Equal(t, 0.6, groupImageMultiplier(&dbent.Group{RateMultiplier: 0.6}))
	require.Equal(t, 3.0, groupImageMultiplier(&dbent.Group{RateMultiplier: 0.6, ImageRateIndependent: true, ImageRateMultiplier: 3}))
	require.Zero(t, groupImageMultiplier(&dbent.Group{RateMultiplier: 0.6, ImageRateIndependent: true, ImageRateMultiplier: -1}))
}
