//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPromptMarketPurchaseAndRefund(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	short := suffix[len(suffix)-6:]
	seller := mustCreateUser(t, integrationEntClient, &service.User{Email: "pm-s-" + suffix + "@prompt.test", Username: "pms" + short})
	buyer := mustCreateUser(t, integrationEntClient, &service.User{Email: "pm-b-" + suffix + "@prompt.test", Username: "pmb" + short, Balance: 1})
	poor := mustCreateUser(t, integrationEntClient, &service.User{Email: "pm-p-" + suffix + "@prompt.test", Username: "pmp" + short, Balance: 0.1})
	other := mustCreateUser(t, integrationEntClient, &service.User{Email: "pm-o-" + suffix + "@prompt.test", Username: "pmo" + short})
	users := []int64{seller.ID, buyer.ID, poor.ID, other.ID}
	t.Cleanup(func() {
		bg := context.Background()
		for _, id := range users {
			_, _ = integrationDB.ExecContext(bg, `DELETE FROM prompt_listings WHERE seller_user_id = $1`, id)
			_, _ = integrationDB.ExecContext(bg, `DELETE FROM prompt_items WHERE owner_user_id = $1`, id)
		}
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM settings WHERE key = $1`, service.SettingKeyPromptMarket)
	})

	prompts := NewPromptLibraryRepository(integrationDB)
	insert := func(owner int64, text string, public bool) int64 {
		o := owner
		item := &service.PromptItem{SourceID: service.PromptSourceUser, ExternalID: fmt.Sprintf("%s-%d-%d", suffix, owner, time.Now().UnixNano()), OwnerUserID: &o, Kind: "image",
			Title: "t", Prompt: text, ReferenceImageURLs: []string{}, SourceTags: []string{}, Scenes: []string{"other"}, Model: "unknown", Lang: "zh",
			Visibility: service.PromptVisibilityPrivate, Status: service.PromptStatusActive, DedupeKey: service.PromptDedupeKey(text)}
		if public {
			item.Visibility = service.PromptVisibilityPublic
		}
		require.NoError(t, prompts.Insert(ctx, item))
		return item.ID
	}
	p1 := insert(seller.ID, "赛博朋克夜景，霓虹灯，雨夜反光，电影感广角 "+suffix, false)
	p2 := insert(seller.ID, "国风水墨山水，远处飞鸟，留白构图 "+suffix, false)

	svc := service.NewPromptMarketService(NewPromptMarketRepository(integrationDB), NewSettingRepository(integrationEntClient), nil, nil)
	in := service.PromptListingInput{Title: "夜景合集", Price: 0.5, PromptIDs: []int64{p1, p2}}
	_, err := svc.CreateListing(ctx, seller.ID, in)
	require.ErrorIs(t, err, service.ErrPromptMarketClosed, "selling is off until the admin opens it")
	_, err = svc.SaveConfig(ctx, service.PromptMarketConfig{Enabled: true, FeeRate: 0.2, MinPrice: 0.1, MaxPrice: 50})
	require.NoError(t, err)

	_, err = svc.CreateListing(ctx, seller.ID, service.PromptListingInput{Title: "x", Price: 0.5, PromptIDs: []int64{p1, insert(buyer.ID, "别人的提示词 "+suffix, false)}})
	require.ErrorIs(t, err, service.ErrPromptListingItems, "only the seller's own prompts")
	listing, err := svc.CreateListing(ctx, seller.ID, in)
	require.NoError(t, err)
	require.Equal(t, service.PromptListingPending, listing.Status)
	require.Equal(t, 2, listing.ItemCount)
	_, err = svc.Get(ctx, buyer.ID, listing.ID)
	require.ErrorIs(t, err, service.ErrPromptListingNotFound, "pending listings are not public")

	_, err = svc.Approve(ctx, listing.ID)
	require.NoError(t, err)
	seen, err := svc.Get(ctx, buyer.ID, listing.ID)
	require.NoError(t, err)
	require.False(t, seen.Unlocked)
	require.NotEmpty(t, seen.Teaser)
	for _, item := range seen.Items {
		require.Empty(t, item.Prompt, "prompts stay hidden until bought")
	}

	balance := func(id int64) float64 {
		var b float64
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance::float8 FROM users WHERE id = $1`, id).Scan(&b))
		return b
	}
	_, err = svc.Purchase(ctx, buyer.ID, listing.ID, 0.3)
	require.ErrorIs(t, err, service.ErrPromptPriceChanged)
	_, err = svc.Purchase(ctx, seller.ID, listing.ID, 0.5)
	require.ErrorIs(t, err, service.ErrPromptBuyOwn)
	_, err = svc.Purchase(ctx, poor.ID, listing.ID, 0.5)
	require.ErrorIs(t, err, service.ErrPromptBalanceLow)
	require.InDelta(t, 0.1, balance(poor.ID), 1e-9, "a failed purchase moves nothing")

	bought, err := svc.Purchase(ctx, buyer.ID, listing.ID, 0.5)
	require.NoError(t, err)
	require.True(t, bought.Unlocked)
	require.NotEmpty(t, bought.Items[0].Prompt)
	require.InDelta(t, 0.5, balance(buyer.ID), 1e-9)
	require.InDelta(t, 0.4, balance(seller.ID), 1e-9, "the seller gets the price minus 20%")
	_, err = svc.Purchase(ctx, buyer.ID, listing.ID, 0.5)
	require.ErrorIs(t, err, service.ErrPromptBoughtAlready)
	library, err := svc.Purchases(ctx, buyer.ID)
	require.NoError(t, err)
	require.Len(t, library, 1)

	// The buyer cannot resell what they bought.
	copied := insert(buyer.ID, "赛博朋克夜景，霓虹灯，雨夜反光，电影感广角 "+suffix, false)
	_, err = svc.CreateListing(ctx, buyer.ID, service.PromptListingInput{Title: "转卖", Price: 0.5, PromptIDs: []int64{copied}})
	require.ErrorIs(t, err, service.ErrPromptListingResale)
	// Nor can anyone sell a prompt that is free in the public library.
	insert(other.ID, "公开的免费提示词 "+suffix, true)
	free := insert(seller.ID, "公开的免费提示词 "+suffix, false)
	_, err = svc.CreateListing(ctx, seller.ID, service.PromptListingInput{Title: "免费的", Price: 0.5, PromptIDs: []int64{free}})
	require.ErrorIs(t, err, service.ErrPromptListingPublicCopy)

	// A change to a live listing waits for review; the approved price keeps selling.
	in.Price = 0.8
	edited, err := svc.UpdateListing(ctx, seller.ID, listing.ID, in)
	require.NoError(t, err)
	require.NotNil(t, edited.Draft)
	require.InDelta(t, 0.5, edited.Price, 1e-9)
	review, total, err := svc.AdminList(ctx, service.PromptMarketListQuery{Status: "review"})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.Contains(t, listingIDs(review), listing.ID)
	approved, err := svc.Approve(ctx, listing.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.8, approved.Price, 1e-9)
	require.Nil(t, approved.Draft)

	// Withdrawn listings are not for sale; buyers keep access.
	_, err = svc.SetListed(ctx, seller.ID, listing.ID, false)
	require.NoError(t, err)
	_, err = svc.Purchase(ctx, poor.ID, listing.ID, 0.8)
	require.ErrorIs(t, err, service.ErrPromptListingNotFound)
	still, err := svc.Get(ctx, buyer.ID, listing.ID)
	require.NoError(t, err)
	require.True(t, still.Unlocked)

	// Refund: the buyer gets the price back, the seller's share is taken back, access ends.
	purchases, _, err := svc.AdminPurchases(ctx, service.PromptPurchaseQuery{Status: service.PromptPurchasePaid})
	require.NoError(t, err)
	var purchaseID int64
	for _, p := range purchases {
		if p.ListingID == listing.ID {
			purchaseID = p.ID
		}
	}
	require.NotZero(t, purchaseID)
	_, err = svc.Refund(ctx, purchaseID)
	require.NoError(t, err)
	require.InDelta(t, 1.0, balance(buyer.ID), 1e-9)
	require.InDelta(t, 0.0, balance(seller.ID), 1e-9)
	_, err = svc.Refund(ctx, purchaseID)
	require.ErrorIs(t, err, service.ErrPromptPurchaseNotFound, "refunds happen once")
	_, err = svc.Get(ctx, buyer.ID, listing.ID)
	require.ErrorIs(t, err, service.ErrPromptListingNotFound, "a withdrawn listing is gone for a refunded buyer")
	summary := mustSellerSummary(t, svc, seller.ID)
	require.Equal(t, int64(0), summary.Sales)
}

func listingIDs(list []service.PromptListing) []int64 {
	out := make([]int64, 0, len(list))
	for _, l := range list {
		out = append(out, l.ID)
	}
	return out
}

func mustSellerSummary(t *testing.T, svc *service.PromptMarketService, sellerID int64) service.PromptSellerSummary {
	t.Helper()
	_, summary, err := svc.Selling(context.Background(), sellerID)
	require.NoError(t, err)
	return summary
}
