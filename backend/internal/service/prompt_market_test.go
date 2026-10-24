//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptTeaserShowsAQuarterAtMost40(t *testing.T) {
	require.Equal(t, "", promptTeaser(nil))
	require.Equal(t, "一只…", promptTeaser([]PromptListingItem{{Prompt: "一只橘猫坐在窗台"}}))
	long := strings.Repeat("字", 400)
	require.Equal(t, strings.Repeat("字", 40)+"…", promptTeaser([]PromptListingItem{{Prompt: long}}))
}

func TestPromptListingViewHidesContentUntilBought(t *testing.T) {
	svc := &PromptMarketService{}
	source := int64(9)
	l := PromptListing{SellerUserID: 1, SellerEmail: "s@x", Flags: []string{"nsfw"}, ReviewNote: "note", Draft: &PromptListingDraft{Title: "d"},
		Items: []PromptListingItem{{Title: "a", Prompt: "完整的提示词内容一二三四五六七八", DedupeKey: "k", SourcePromptID: &source}}}

	stranger := svc.view(l, 2, false)
	require.False(t, stranger.Unlocked)
	require.Empty(t, stranger.Items[0].Prompt)
	require.Empty(t, stranger.Items[0].DedupeKey)
	require.Nil(t, stranger.Items[0].SourcePromptID)
	require.Nil(t, stranger.Draft)
	require.Empty(t, stranger.SellerEmail)
	require.Empty(t, stranger.Flags)
	require.NotEmpty(t, stranger.Teaser)

	buyer := svc.view(l, 2, true)
	require.True(t, buyer.Unlocked)
	require.Equal(t, l.Items[0].Prompt, buyer.Items[0].Prompt)
	require.Nil(t, buyer.Draft, "only the seller sees pending changes")

	seller := svc.view(l, 1, false)
	require.True(t, seller.Mine)
	require.True(t, seller.Unlocked)
	require.NotNil(t, seller.Draft)
	require.Equal(t, "note", seller.ReviewNote)
	require.Equal(t, &source, seller.Items[0].SourcePromptID, "the seller edits by their own prompt ids")
	require.Empty(t, seller.Items[0].DedupeKey)
}

func TestPromptMarketConfigValidation(t *testing.T) {
	settings := &memSettings{values: map[string]string{}}
	svc := NewPromptMarketService(nil, settings, nil, nil)
	ctx := context.Background()
	cfg := svc.Config(ctx)
	require.False(t, cfg.Enabled, "off by default")
	require.InDelta(t, 0.2, cfg.FeeRate, 1e-9)

	for _, bad := range []PromptMarketConfig{{FeeRate: 0.95, MinPrice: 0.1, MaxPrice: 1}, {FeeRate: 0.2, MinPrice: 0, MaxPrice: 1}, {FeeRate: 0.2, MinPrice: 2, MaxPrice: 1}, {FeeRate: 0.2, MinPrice: 1, MaxPrice: 5000}} {
		_, err := svc.SaveConfig(ctx, bad)
		require.ErrorIs(t, err, ErrPromptMarketBadConfig)
	}
	saved, err := svc.SaveConfig(ctx, PromptMarketConfig{Enabled: true, FeeRate: 0.15, MinPrice: 0.333, MaxPrice: 20})
	require.NoError(t, err)
	require.InDelta(t, 0.33, saved.MinPrice, 1e-9)
	require.True(t, svc.Config(ctx).Enabled)
}
