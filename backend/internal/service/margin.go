package service

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 套餐毛利看板: what the upstream costs against what members paid. Usage is valued at standard
// prices (usage_logs.total_cost, USD) times the admin's cost per USD (CNY, e.g. 0.3 when $1 of
// usage costs ¥0.3 upstream). Per subscription, its own usage (usage_logs.subscription_id) is set
// against the subscription orders paid for it; active ones are projected to the end of the term.
const (
	SettingKeyMarginCostPerUSD = "margin_cost_cny_per_usd"

	marginDefaultDays = 30
	marginMaxDays     = 180
	marginMaxCostRate = 10.0
	marginTopUsers    = 20
)

var errMarginBadCost = infraerrors.BadRequest("MARGIN_BAD_COST", "成本系数需在 0 到 10 之间（Cost per USD must be between 0 and 10）")

// MarginGroupRow is one group's usage over the period (members only).
type MarginGroupRow struct {
	GroupID      int64   `json:"group_id"`
	Name         string  `json:"name"`
	Subscription bool    `json:"subscription"`
	Users        int64   `json:"users"`
	UsageUSD     float64 `json:"usage_usd"`  // standard price
	BilledUSD    float64 `json:"billed_usd"` // deducted (pay-as-you-go balance; equals usage for subscriptions)
}

// MarginPlanRow is a plan on sale with its group's limits.
type MarginPlanRow struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	GroupID         int64    `json:"group_id"`
	GroupName       string   `json:"group_name"`
	Price           float64  `json:"price"`
	ValidityDays    int      `json:"validity_days"`
	ValidityUnit    string   `json:"validity_unit"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	Sold            int64    `json:"sold"`    // paid orders in the period
	Revenue         float64  `json:"revenue"` // CNY
}

// MarginSubscriptionRow is a member's subscription that was active during the period.
type MarginSubscriptionRow struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Email           string    `json:"email"`
	GroupID         int64     `json:"group_id"`
	GroupName       string    `json:"group_name"`
	StartsAt        time.Time `json:"starts_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	Paid            float64   `json:"paid"` // CNY, subscription orders for this group within the term
	UsageUSD        float64   `json:"usage_usd"`
	DailyLimitUSD   *float64  `json:"-"`
	WeeklyLimitUSD  *float64  `json:"-"`
	MonthlyLimitUSD *float64  `json:"-"`
}

// MarginUserRow is a member's usage and payments over the period.
type MarginUserRow struct {
	UserID    int64   `json:"user_id"`
	Email     string  `json:"email"`
	UsageUSD  float64 `json:"usage_usd"`
	BilledUSD float64 `json:"billed_usd"` // deducted from balance
	Paid      float64 `json:"paid"`       // CNY paid through a payment gateway in the period
}

// MarginRaw is what the repository reads.
type MarginRaw struct {
	Groups        []MarginGroupRow
	AdminUsageUSD float64
	// Money that came in through a payment gateway (not balance), by order type, CNY.
	RechargeRevenue     float64
	SubscriptionRevenue float64
	OtherRevenue        float64
	Plans               []MarginPlanRow
	Subscriptions       []MarginSubscriptionRow
	Users               []MarginUserRow
}

type MarginRepository interface {
	Load(ctx context.Context, since time.Time, topUsers int) (*MarginRaw, error)
}

// Report types.

type MarginTotals struct {
	Revenue             float64 `json:"revenue"` // CNY through payment gateways
	RechargeRevenue     float64 `json:"recharge_revenue"`
	SubscriptionRevenue float64 `json:"subscription_revenue"`
	OtherRevenue        float64 `json:"other_revenue"`
	PayGoBilled         float64 `json:"paygo_billed"` // CNY of balance deducted for pay-as-you-go usage
	PayGoCost           float64 `json:"paygo_cost"`
	SubscriptionCost    float64 `json:"subscription_cost"`
	AdminCost           float64 `json:"admin_cost"`
	MemberCost          float64 `json:"member_cost"`
	Margin              float64 `json:"margin"` // revenue − member cost
}

type MarginGroup struct {
	MarginGroupRow
	Cost   float64  `json:"cost"`
	Margin *float64 `json:"margin,omitempty"` // pay-as-you-go: billed − cost
}

type MarginPlan struct {
	MarginPlanRow
	Days         int      `json:"days"`
	CapUSD       *float64 `json:"cap_usd"`        // most usage over the plan's length
	MaxCost      *float64 `json:"max_cost"`       // CNY when fully used
	MaxLoss      float64  `json:"max_loss"`       // CNY per fully used plan, 0 when it can't lose
	BreakEvenUSD float64  `json:"break_even_usd"` // usage at which cost equals price
}

const (
	MarginStatusOK   = "ok"
	MarginStatusRisk = "risk" // on pace to cost more than was paid
	MarginStatusLoss = "loss" // already costs more than was paid
	MarginStatusGift = "gift" // nothing paid
)

type MarginSubscription struct {
	MarginSubscriptionRow
	Active          bool    `json:"active"`
	DaysElapsed     float64 `json:"days_elapsed"`
	DaysTotal       float64 `json:"days_total"`
	Cost            float64 `json:"cost"`
	Margin          float64 `json:"margin"`
	ProjectedUSD    float64 `json:"projected_usd"`
	ProjectedCost   float64 `json:"projected_cost"`
	ProjectedMargin float64 `json:"projected_margin"`
	Status          string  `json:"status"`
}

type MarginUser struct {
	MarginUserRow
	Cost   float64 `json:"cost"`
	Margin float64 `json:"margin"` // paid − cost over the period
}

type MarginReport struct {
	CostPerUSD    float64              `json:"cost_per_usd"`
	Configured    bool                 `json:"configured"`
	Days          int                  `json:"days"`
	Since         time.Time            `json:"since"`
	Totals        MarginTotals         `json:"totals"`
	Groups        []MarginGroup        `json:"groups"`
	Plans         []MarginPlan         `json:"plans"`
	Subscriptions []MarginSubscription `json:"subscriptions"`
	Users         []MarginUser         `json:"users"`
}

type MarginService struct {
	repo     MarginRepository
	settings SettingRepository
	now      func() time.Time
}

func NewMarginService(repo MarginRepository, settings SettingRepository) *MarginService {
	return &MarginService{repo: repo, settings: settings, now: time.Now}
}

// CostPerUSD is the configured CNY cost of $1 of usage (0 when not set).
func (s *MarginService) CostPerUSD(ctx context.Context) float64 {
	v, err := s.settings.GetValue(ctx, SettingKeyMarginCostPerUSD)
	if err != nil {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(f) || f < 0 || f > marginMaxCostRate {
		return 0
	}
	return f
}

func (s *MarginService) SetCostPerUSD(ctx context.Context, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > marginMaxCostRate {
		return errMarginBadCost
	}
	return s.settings.Set(ctx, SettingKeyMarginCostPerUSD, strconv.FormatFloat(v, 'f', -1, 64))
}

func (s *MarginService) Report(ctx context.Context, days int) (*MarginReport, error) {
	if days <= 0 {
		days = marginDefaultDays
	}
	if days > marginMaxDays {
		days = marginMaxDays
	}
	now := s.now()
	local := now.In(analyticsTZ)
	since := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, analyticsTZ).AddDate(0, 0, -(days - 1))
	raw, err := s.repo.Load(ctx, since, marginTopUsers)
	if err != nil {
		return nil, err
	}
	rate := s.CostPerUSD(ctx)
	return BuildMarginReport(raw, rate, days, since, now), nil
}

// BuildMarginReport values the raw figures at rate (CNY per USD of usage).
func BuildMarginReport(raw *MarginRaw, rate float64, days int, since, now time.Time) *MarginReport {
	out := &MarginReport{
		CostPerUSD: rate, Configured: rate > 0, Days: days, Since: since,
		Groups: []MarginGroup{}, Plans: []MarginPlan{}, Subscriptions: []MarginSubscription{}, Users: []MarginUser{},
	}
	t := &out.Totals
	t.RechargeRevenue, t.SubscriptionRevenue, t.OtherRevenue = round2(raw.RechargeRevenue), round2(raw.SubscriptionRevenue), round2(raw.OtherRevenue)
	t.Revenue = round2(raw.RechargeRevenue + raw.SubscriptionRevenue + raw.OtherRevenue)
	t.AdminCost = round2(raw.AdminUsageUSD * rate)

	for _, g := range raw.Groups {
		mg := MarginGroup{MarginGroupRow: g, Cost: round2(g.UsageUSD * rate)}
		if g.Subscription {
			t.SubscriptionCost += g.UsageUSD * rate
		} else {
			t.PayGoCost += g.UsageUSD * rate
			t.PayGoBilled += g.BilledUSD
			m := round2(g.BilledUSD - g.UsageUSD*rate)
			mg.Margin = &m
		}
		out.Groups = append(out.Groups, mg)
	}
	t.PayGoCost, t.SubscriptionCost, t.PayGoBilled = round2(t.PayGoCost), round2(t.SubscriptionCost), round2(t.PayGoBilled)
	t.MemberCost = round2(t.PayGoCost + t.SubscriptionCost)
	t.Margin = round2(t.Revenue - t.MemberCost)

	for _, p := range raw.Plans {
		mp := MarginPlan{MarginPlanRow: p, Days: psComputeValidityDays(p.ValidityDays, p.ValidityUnit)}
		mp.CapUSD = periodCapUSD(mp.Days, p.DailyLimitUSD, p.WeeklyLimitUSD, p.MonthlyLimitUSD)
		if rate > 0 {
			mp.BreakEvenUSD = round2(p.Price / rate)
			if mp.CapUSD != nil {
				c := round2(*mp.CapUSD * rate)
				mp.MaxCost = &c
				mp.MaxLoss = round2(math.Max(0, c-p.Price))
			}
		}
		out.Plans = append(out.Plans, mp)
	}

	for _, sub := range raw.Subscriptions {
		ms := MarginSubscription{MarginSubscriptionRow: sub}
		ms.DaysTotal = round2(math.Max(0, sub.ExpiresAt.Sub(sub.StartsAt).Hours()/24))
		end := sub.ExpiresAt
		if now.Before(end) {
			ms.Active = true
			end = now
		}
		ms.DaysElapsed = round2(math.Max(0, end.Sub(sub.StartsAt).Hours()/24))
		ms.Cost = round2(sub.UsageUSD * rate)
		ms.Margin = round2(sub.Paid - ms.Cost)
		ms.ProjectedUSD = sub.UsageUSD
		if ms.Active && ms.DaysElapsed > 0 {
			// Pace so far, a day at least, up to what the group's limits allow over the term.
			pace := sub.UsageUSD / math.Max(1, ms.DaysElapsed)
			ms.ProjectedUSD = math.Max(sub.UsageUSD, pace*ms.DaysTotal)
			if c := periodCapUSD(int(math.Round(ms.DaysTotal)), sub.DailyLimitUSD, sub.WeeklyLimitUSD, sub.MonthlyLimitUSD); c != nil {
				ms.ProjectedUSD = math.Max(sub.UsageUSD, math.Min(ms.ProjectedUSD, *c))
			}
		}
		ms.ProjectedUSD = round2(ms.ProjectedUSD)
		ms.ProjectedCost = round2(ms.ProjectedUSD * rate)
		ms.ProjectedMargin = round2(sub.Paid - ms.ProjectedCost)
		switch {
		case sub.Paid <= 0:
			ms.Status = MarginStatusGift
		case ms.Margin < 0:
			ms.Status = MarginStatusLoss
		case ms.ProjectedMargin < 0:
			ms.Status = MarginStatusRisk
		default:
			ms.Status = MarginStatusOK
		}
		ms.Email = MaskEmail(sub.Email)
		out.Subscriptions = append(out.Subscriptions, ms)
	}
	// Worst first: the most money lost (or projected to be) on top.
	sort.SliceStable(out.Subscriptions, func(i, j int) bool {
		return out.Subscriptions[i].ProjectedMargin < out.Subscriptions[j].ProjectedMargin
	})

	for _, u := range raw.Users {
		mu := MarginUser{MarginUserRow: u, Cost: round2(u.UsageUSD * rate)}
		mu.Margin = round2(u.Paid - mu.Cost)
		mu.Email = MaskEmail(u.Email)
		out.Users = append(out.Users, mu)
	}
	return out
}

// periodCapUSD is the most usage the limits allow over days: the tightest of daily × days,
// weekly × weeks and monthly × months (at least one week / month), nil without limits.
func periodCapUSD(days int, daily, weekly, monthly *float64) *float64 {
	d := math.Max(1, float64(days))
	var caps []float64
	if daily != nil && *daily > 0 {
		caps = append(caps, *daily*d)
	}
	if weekly != nil && *weekly > 0 {
		caps = append(caps, *weekly*math.Max(1, d/7))
	}
	if monthly != nil && *monthly > 0 {
		caps = append(caps, *monthly*math.Max(1, d/30))
	}
	if len(caps) == 0 {
		return nil
	}
	c := caps[0]
	for _, v := range caps[1:] {
		c = math.Min(c, v)
	}
	c = round2(c)
	return &c
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
