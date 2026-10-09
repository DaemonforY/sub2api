//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInvalidatePublicPricingDropsTheCache(t *testing.T) {
	publicPricingCache.Lock()
	publicPricingCache.val, publicPricingCache.at = &PublicPricing{}, time.Now()
	publicPricingCache.Unlock()

	invalidatePublicPricing()

	publicPricingCache.Lock()
	defer publicPricingCache.Unlock()
	require.Nil(t, publicPricingCache.val, "the next request rebuilds from the database")
}
