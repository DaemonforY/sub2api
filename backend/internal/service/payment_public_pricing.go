package service

import (
	"context"
	"sort"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/usagelog"
	"github.com/Wei-Shaw/sub2api/ent/user"
)

// Public pricing for the /pricing page (no login): the plans on sale with their usage limits,
// the pay-as-you-go groups, how much balance one yuan buys, and the recent average cost of one
// request so visitors can estimate a month. Read live, cached for two minutes; editing a plan or the
// payment settings clears the cache so the home and pricing pages show the change right away.
const publicPricingTTL = 2 * time.Minute

type PublicPricingPlan struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Price           float64  `json:"price"`
	OriginalPrice   *float64 `json:"original_price,omitempty"`
	Currency        string   `json:"currency"`
	ValidityDays    int      `json:"validity_days"`
	ValidityUnit    string   `json:"validity_unit"`
	Features        string   `json:"features"`
	GroupName       string   `json:"group_name"`
	Platform        string   `json:"platform"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	Recommended     bool     `json:"recommended,omitempty"`
}

type PublicPricingGroup struct {
	Name           string  `json:"name"`
	Platform       string  `json:"platform"`
	Description    string  `json:"description"`
	RateMultiplier float64 `json:"rate_multiplier"`
}

type PublicPricing struct {
	PaymentEnabled     bool                 `json:"payment_enabled"`
	RechargeMultiplier float64              `json:"recharge_multiplier"` // balance (USD) per 1 CNY paid
	MinRecharge        float64              `json:"min_recharge"`
	Plans              []PublicPricingPlan  `json:"plans"`
	PayAsYouGo         []PublicPricingGroup `json:"pay_as_you_go"`
	// Average billed cost of one OpenAI-group request by members over the last 30 days (USD), and
	// how many requests that is based on — the page only shows an estimate with enough samples.
	AvgRequestCostUSD float64 `json:"avg_request_cost_usd"`
	AvgSampleRequests int     `json:"avg_sample_requests"`
	// Requests members have made in total (admins excluded); the home page shows it from 10,000 up.
	TotalRequests int       `json:"total_requests"`
	UpdatedAt     time.Time `json:"updated_at"`
}

var publicPricingCache struct {
	sync.Mutex
	at  time.Time
	val *PublicPricing
}

// invalidatePublicPricing drops the cached public pricing (after a plan or payment setting changes).
func invalidatePublicPricing() {
	publicPricingCache.Lock()
	publicPricingCache.val = nil
	publicPricingCache.Unlock()
}

func (s *PaymentConfigService) PublicPricing(ctx context.Context) (*PublicPricing, error) {
	publicPricingCache.Lock()
	defer publicPricingCache.Unlock()
	if publicPricingCache.val != nil && time.Since(publicPricingCache.at) < publicPricingTTL {
		return publicPricingCache.val, nil
	}
	out, err := s.buildPublicPricing(ctx)
	if err != nil {
		return nil, err
	}
	publicPricingCache.val, publicPricingCache.at = out, time.Now()
	return out, nil
}

func (s *PaymentConfigService) buildPublicPricing(ctx context.Context) (*PublicPricing, error) {
	cfg, err := s.GetPaymentConfig(ctx)
	if err != nil {
		return nil, err
	}
	out := &PublicPricing{
		PaymentEnabled:     cfg.Enabled,
		RechargeMultiplier: cfg.BalanceRechargeMultiplier,
		MinRecharge:        cfg.MinAmount,
		Plans:              []PublicPricingPlan{},
		PayAsYouGo:         []PublicPricingGroup{},
		UpdatedAt:          time.Now(),
	}
	if out.RechargeMultiplier <= 0 {
		out.RechargeMultiplier = 1
	}

	plans, err := s.ListPlansForSale(ctx)
	if err != nil {
		return nil, err
	}
	info := s.GetGroupInfoMap(ctx, plans)
	for _, p := range plans {
		gi := info[p.GroupID]
		out.Plans = append(out.Plans, PublicPricingPlan{
			ID: int64(p.ID), Name: p.Name, Description: p.Description, Price: p.Price, OriginalPrice: p.OriginalPrice,
			Currency: p.Currency, ValidityDays: p.ValidityDays, ValidityUnit: p.ValidityUnit, Features: p.Features,
			GroupName: gi.Name, Platform: gi.Platform, Recommended: p.Recommended,
			DailyLimitUSD: gi.DailyLimitUSD, WeeklyLimitUSD: gi.WeeklyLimitUSD, MonthlyLimitUSD: gi.MonthlyLimitUSD,
		})
	}

	groups, err := s.entClient.Group.Query().
		Where(group.StatusEQ(StatusActive), group.SubscriptionTypeEQ(SubscriptionTypeStandard), group.IsExclusiveEQ(false)).
		Order(group.BySortOrder()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		desc := ""
		if g.Description != nil {
			desc = *g.Description
		}
		out.PayAsYouGo = append(out.PayAsYouGo, PublicPricingGroup{Name: g.Name, Platform: g.Platform, Description: desc, RateMultiplier: g.RateMultiplier})
	}
	sort.SliceStable(out.PayAsYouGo, func(i, j int) bool {
		return out.PayAsYouGo[i].Platform == PlatformOpenAI && out.PayAsYouGo[j].Platform != PlatformOpenAI
	})

	var agg []struct {
		Sum   float64 `json:"sum"`
		Count int     `json:"count"`
	}
	err = s.entClient.UsageLog.Query().
		Where(
			usagelog.CreatedAtGTE(time.Now().AddDate(0, 0, -30)),
			usagelog.HasGroupWith(group.PlatformEQ(PlatformOpenAI)),
			usagelog.HasUserWith(user.RoleNEQ(RoleAdmin)),
		).
		Aggregate(dbent.As(dbent.Sum(usagelog.FieldActualCost), "sum"), dbent.As(dbent.Count(), "count")).
		Scan(ctx, &agg)
	if err == nil && len(agg) == 1 && agg[0].Count > 0 {
		out.AvgRequestCostUSD = agg[0].Sum / float64(agg[0].Count)
		out.AvgSampleRequests = agg[0].Count
	}
	if n, err := s.entClient.UsageLog.Query().Where(usagelog.HasUserWith(user.RoleNEQ(RoleAdmin))).Count(ctx); err == nil {
		out.TotalRequests = n
	}
	return out, nil
}
