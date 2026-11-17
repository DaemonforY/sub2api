package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"math"
	"slices"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// 重置今日额度 (paid daily quota reset).
//
// A subscriber whose subscription is blocked by a used-up limit pays daily limit ($) × PricePerUSD
// (CNY) to clear today's usage. From then until the next daily rollover (00:00 in the configured
// timezone) usage counts toward the daily window only — never the weekly or monthly windows — and
// only the daily limit is checked, so the reset also unblocks a used-up week or month for the rest
// of the day. Each reset therefore sells a whole extra day of usage; the price must cover upstream
// cost. Paid online (order type quota_reset, plan_id = group id) or with balance.

// SettingKeySubscriptionQuotaReset stores QuotaResetSettings as JSON.
const SettingKeySubscriptionQuotaReset = "subscription_quota_reset_settings"

// QuotaResetSettings: admin knobs for 重置今日额度.
type QuotaResetSettings struct {
	Enabled bool `json:"enabled"`
	// PricePerUSD: CNY charged per dollar of the group's daily limit.
	PricePerUSD float64 `json:"price_per_usd"`
	// MaxPerDay: resets one subscription may buy per day.
	MaxPerDay int `json:"max_per_day"`
	// Platforms the reset is sold for (group platform). Claude upstream costs ~¥1/$, so it is off by default.
	Platforms []string `json:"platforms"`
	// WeChatNotice: when a subscription first hits a limit each day, send its owner a 公众号 客服消息
	// with a link to buy the reset (only reaches users who talked to the 公众号 in the last 48 h).
	WeChatNotice bool `json:"wechat_notice"`
}

// DefaultQuotaResetSettings: on for Codex (OpenAI) subscriptions at ¥0.5 per daily-limit dollar, once a day.
func DefaultQuotaResetSettings() *QuotaResetSettings {
	return &QuotaResetSettings{Enabled: true, PricePerUSD: 0.5, MaxPerDay: 1, Platforms: []string{PlatformOpenAI}, WeChatNotice: true}
}

func (q *QuotaResetSettings) normalize() {
	if q.MaxPerDay < 1 {
		q.MaxPerDay = 1
	}
	if q.MaxPerDay > 20 {
		q.MaxPerDay = 20
	}
	clean := make([]string, 0, len(q.Platforms))
	for _, p := range q.Platforms {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" && !slices.Contains(clean, p) {
			clean = append(clean, p)
		}
	}
	q.Platforms = clean
}

// GetQuotaResetSettings returns the stored settings, or the defaults when none are stored.
func (s *PaymentConfigService) GetQuotaResetSettings(ctx context.Context) *QuotaResetSettings {
	out := DefaultQuotaResetSettings()
	if s == nil || s.settingRepo == nil {
		return out
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeySubscriptionQuotaReset)
	if err != nil || strings.TrimSpace(raw) == "" {
		return out
	}
	var stored QuotaResetSettings
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return out
	}
	stored.normalize()
	return &stored
}

// SetQuotaResetSettings validates and stores the settings.
func (s *PaymentConfigService) SetQuotaResetSettings(ctx context.Context, in *QuotaResetSettings) (*QuotaResetSettings, error) {
	if in == nil {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "settings required")
	}
	if math.IsNaN(in.PricePerUSD) || math.IsInf(in.PricePerUSD, 0) || in.PricePerUSD <= 0 || in.PricePerUSD > 100 {
		return nil, infraerrors.BadRequest("INVALID_PRICE", "price_per_usd must be between 0 and 100")
	}
	in.normalize()
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeySubscriptionQuotaReset, string(data)); err != nil {
		return nil, fmt.Errorf("save quota reset settings: %w", err)
	}
	return in, nil
}

// Reasons a subscription can't buy a reset right now (QuotaResetQuote.Reason).
const (
	QuotaResetReasonDisabled     = "disabled"       // feature off
	QuotaResetReasonNotSupported = "not_supported"  // group platform not sold
	QuotaResetReasonOneTime      = "one_time"       // one-day trial subscriptions
	QuotaResetReasonNoDailyLimit = "no_daily_limit" // nothing to price or reset
	QuotaResetReasonNotBlocked   = "not_blocked"    // usable now; offered once a limit is used up
	QuotaResetReasonMaxReached   = "max_reached"    // today's resets used up
)

// QuotaResetQuote describes whether and for how much one subscription can buy a reset.
type QuotaResetQuote struct {
	SubscriptionID int64      `json:"subscription_id"`
	GroupID        int64      `json:"group_id"`
	GroupName      string     `json:"group_name"`
	Available      bool       `json:"available"`
	Blocked        bool       `json:"blocked"`
	CanBuy         bool       `json:"can_buy"`
	Reason         string     `json:"reason"`
	Price          float64    `json:"price"`
	BalancePrice   float64    `json:"balance_price"`
	DailyLimitUSD  float64    `json:"daily_limit_usd"`
	MaxPerDay      int        `json:"max_per_day"`
	ResetsToday    int        `json:"resets_today"`
	ResetActive    bool       `json:"boost_active"`
	ResetUntil     *time.Time `json:"boost_until"`
}

// quotaResetPrice: daily limit × rate, rounded to the fen.
func quotaResetPrice(dailyLimit, perUSD float64) float64 {
	return math.Round(dailyLimit*perUSD*100) / 100
}

// subscriptionBlockedAt: a limit that applies right now is used up (the auth check would reject).
func subscriptionBlockedAt(sub *UserSubscription, group *Group, now time.Time) bool {
	daily := sub.DailyUsageUSD
	if sub.canAutomaticallyResetDailyAt(now) {
		daily = 0
	}
	if group.HasDailyLimit() && daily >= *group.DailyLimitUSD {
		return true
	}
	if sub.DailyResetActiveAt(now) {
		return false
	}
	weekly, monthly := sub.WeeklyUsageUSD, sub.MonthlyUsageUSD
	if sub.canAutomaticallyResetWeeklyAt(now) {
		weekly = 0
	}
	if sub.canAutomaticallyResetMonthlyAt(now) {
		monthly = 0
	}
	return (group.HasWeeklyLimit() && weekly >= *group.WeeklyLimitUSD) ||
		(group.HasMonthlyLimit() && monthly >= *group.MonthlyLimitUSD)
}

// buildQuotaResetQuote decides eligibility and price for one active subscription.
func buildQuotaResetQuote(sub *UserSubscription, group *Group, settings *QuotaResetSettings, cfg *PaymentConfig, now time.Time) QuotaResetQuote {
	q := QuotaResetQuote{SubscriptionID: sub.ID, GroupID: sub.GroupID, MaxPerDay: settings.MaxPerDay}
	if group == nil {
		q.Reason = QuotaResetReasonNotSupported
		return q
	}
	q.GroupName = group.Name
	q.ResetActive = sub.DailyResetActiveAt(now)
	if q.ResetActive {
		until := *sub.DailyResetUntil
		q.ResetUntil = &until
	}
	q.ResetsToday = sub.DailyResetsUsedAt(now)
	q.Blocked = subscriptionBlockedAt(sub, group, now)
	if group.HasDailyLimit() {
		q.DailyLimitUSD = *group.DailyLimitUSD
		q.Price = quotaResetPrice(q.DailyLimitUSD, settings.PricePerUSD)
		if cfg != nil {
			q.BalancePrice = calculateCreditedBalance(q.Price, cfg.BalanceRechargeMultiplier)
		}
	}
	switch {
	case !settings.Enabled:
		q.Reason = QuotaResetReasonDisabled
	case !slices.Contains(settings.Platforms, strings.ToLower(group.Platform)):
		q.Reason = QuotaResetReasonNotSupported
	case !group.HasDailyLimit() || q.Price <= 0:
		q.Reason = QuotaResetReasonNoDailyLimit
	case sub.HasOneTimeDailyQuota():
		q.Reason = QuotaResetReasonOneTime
	}
	if q.Reason != "" {
		return q
	}
	q.Available = true
	switch {
	case q.ResetsToday >= settings.MaxPerDay:
		q.Reason = QuotaResetReasonMaxReached
	case !q.Blocked:
		q.Reason = QuotaResetReasonNotBlocked
	default:
		q.CanBuy = true
	}
	return q
}

// QuotaResetOverview is what the subscriptions page and checkout read.
type QuotaResetOverview struct {
	Enabled           bool              `json:"enabled"`
	Balance           float64           `json:"balance"`
	BalancePayEnabled bool              `json:"balance_pay_enabled"`
	Items             []QuotaResetQuote `json:"items"`
}

// QuotaResetOverview quotes every active subscription of userID.
func (s *PaymentService) QuotaResetOverview(ctx context.Context, userID int64) (*QuotaResetOverview, error) {
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get payment config: %w", err)
	}
	settings := s.configService.GetQuotaResetSettings(ctx)
	out := &QuotaResetOverview{Enabled: settings.Enabled && cfg.Enabled, BalancePayEnabled: !cfg.SubscriptionBalancePayDisabled, Items: []QuotaResetQuote{}}
	if user, err := s.userRepo.GetByID(ctx, userID); err == nil {
		out.Balance = user.Balance
	}
	subs, err := s.subscriptionSvc.ListActiveUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	seen := map[int64]bool{}
	for i := range subs {
		groupID := subs[i].GroupID
		if seen[groupID] {
			continue
		}
		seen[groupID] = true
		// Quote the subscription a purchase would reset (one per group), like quoteForPurchase does.
		sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, userID, groupID)
		if err != nil || sub == nil {
			continue
		}
		group, gerr := s.groupRepo.GetByID(ctx, groupID)
		if gerr != nil {
			group = nil
		}
		out.Items = append(out.Items, buildQuotaResetQuote(sub, group, settings, cfg, now))
	}
	return out, nil
}

var (
	ErrQuotaResetUnavailable = infraerrors.BadRequest("QUOTA_RESET_UNAVAILABLE", "这个订阅现在不能重置今日额度")
	ErrQuotaResetNotBlocked  = infraerrors.BadRequest("QUOTA_RESET_NOT_NEEDED", "订阅额度还没用完，用完后再重置今日额度")
	ErrQuotaResetMaxReached  = infraerrors.BadRequest("QUOTA_RESET_MAX_REACHED", "今天的重置次数已用完，0 点后额度会自动恢复")
)

// quoteForPurchase re-checks eligibility server-side; the order price always comes from here.
func (s *PaymentService) quoteForPurchase(ctx context.Context, userID, groupID int64, cfg *PaymentConfig) (*QuotaResetQuote, *UserSubscription, error) {
	if groupID <= 0 {
		return nil, nil, ErrQuotaResetUnavailable
	}
	sub, err := s.subscriptionSvc.GetActiveSubscription(ctx, userID, groupID)
	if err != nil || sub == nil {
		return nil, nil, ErrQuotaResetUnavailable
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil || group == nil || !group.IsSubscriptionType() {
		return nil, nil, ErrQuotaResetUnavailable
	}
	q := buildQuotaResetQuote(sub, group, s.configService.GetQuotaResetSettings(ctx), cfg, time.Now())
	if !q.CanBuy {
		switch q.Reason {
		case QuotaResetReasonNotBlocked:
			return nil, nil, ErrQuotaResetNotBlocked
		case QuotaResetReasonMaxReached:
			return nil, nil, ErrQuotaResetMaxReached
		}
		return nil, nil, ErrQuotaResetUnavailable
	}
	return &q, sub, nil
}

// SubscriptionLimitHit runs when the gateway rejects sub for a used-up limit. It returns the price
// (CNY) of a reset the user can buy now, 0 when they can't, and the first time each day it also sends
// the owner a WeChat message linking to the checkout.
func (s *PaymentService) SubscriptionLimitHit(ctx context.Context, sub *UserSubscription, group *Group) float64 {
	if s == nil || sub == nil || group == nil {
		return 0
	}
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil || !cfg.Enabled {
		return 0
	}
	settings := s.configService.GetQuotaResetSettings(ctx)
	now := time.Now()
	q := buildQuotaResetQuote(sub, group, settings, cfg, now)
	if !q.Available || q.ResetsToday >= q.MaxPerDay || q.Price <= 0 {
		return 0
	}
	if settings.WeChatNotice {
		s.notifyLimitHitOnce(sub, group, q, now)
	}
	return q.Price
}

// QuotaResetHintText is the Chinese sentence added to a limit error when a reset costs price ("" when price <= 0).
func QuotaResetHintText(price float64) string {
	if price <= 0 {
		return ""
	}
	return fmt.Sprintf("也可以到 %s 的「我的订阅」花 ¥%s 重置今日额度，重置后今天的用量不计入周/月额度", QuotaResetSiteHost, formatCNY(price))
}

// QuotaResetPromoText is the ASCII offer Codex shows in its usage-limit message (header values must be ASCII).
func QuotaResetPromoText(price float64) string {
	return fmt.Sprintf("Reset today's quota for CNY %s at %s/subscriptions", formatCNY(price), QuotaResetSiteHost)
}

// quotaResetCustomTexter is WeChatTemplateNoticeService's 客服消息 sender.
type quotaResetCustomTexter interface {
	SendCustomText(ctx context.Context, userID int64, text string) (WeChatNoticeOutcome, error)
}

// notifyLimitHitOnce sends the limit-hit WeChat message at most once per subscription per day
// (remembered in memory, so a restart may repeat it once).
func (s *PaymentService) notifyLimitHitOnce(sub *UserSubscription, group *Group, q QuotaResetQuote, now time.Time) {
	if s.notificationEmailService == nil {
		return
	}
	sender, ok := s.notificationEmailService.wechatNotices.(quotaResetCustomTexter)
	if !ok || sender == nil {
		return
	}
	day := timezone.StartOfDay(now).Format("2006-01-02")
	key := fmt.Sprintf("%d:%d", sub.UserID, sub.GroupID)
	s.limitNoticeMu.Lock()
	if s.limitNoticeDay != day || s.limitNoticeSent == nil {
		s.limitNoticeDay = day
		s.limitNoticeSent = map[string]struct{}{}
	}
	if _, done := s.limitNoticeSent[key]; done {
		s.limitNoticeMu.Unlock()
		return
	}
	s.limitNoticeSent[key] = struct{}{}
	s.limitNoticeMu.Unlock()

	text := QuotaResetWeChatText(sub, group, q, now)
	userID := sub.UserID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		outcome, err := sender.SendCustomText(ctx, userID, text)
		if err != nil {
			slog.Info("quota reset wechat notice not sent", "userID", userID, "error", err)
			return
		}
		if outcome == WeChatNoticeDelivered {
			slog.Info("quota reset wechat notice sent", "userID", userID, "groupID", q.GroupID)
		}
	}()
}

// QuotaResetWeChatText is the 客服消息 sent when a subscription first hits a limit in a day.
func QuotaResetWeChatText(sub *UserSubscription, group *Group, q QuotaResetQuote, now time.Time) string {
	period := "今日"
	daily := sub.DailyUsageUSD
	if sub.canAutomaticallyResetDailyAt(now) {
		daily = 0
	}
	if !(group.HasDailyLimit() && daily >= *group.DailyLimitUSD) {
		weekly := sub.WeeklyUsageUSD
		if sub.canAutomaticallyResetWeeklyAt(now) {
			weekly = 0
		}
		period = "本月"
		if group.HasWeeklyLimit() && weekly >= *group.WeeklyLimitUSD {
			period = "本周"
		}
	}
	recover := "0 点自动恢复"
	if period != "今日" {
		recover = "要等" + period + "额度重置后才能继续用"
	}
	link := fmt.Sprintf("https://%s/purchase?quota_reset=%d&utm_source=wechat&utm_medium=quota_reset", QuotaResetSiteHost, q.GroupID)
	return fmt.Sprintf("你的「%s」%s额度已用完，%s。\n急用的话，可以花 ¥%s 重置今日额度：今天的用量从 0 重新算（日额度 $%s），而且不计入周/月额度。\n<a href=\"%s\">立即重置今日额度</a>",
		html.EscapeString(group.Name), period, recover, formatCNY(q.Price), formatCNY(q.DailyLimitUSD), html.EscapeString(link))
}

// QuotaResetSiteHost: where users manage subscriptions (named in gateway errors).
const QuotaResetSiteHost = "hivegpt.cn"

func formatCNY(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s
}

// ApplyDailyQuotaReset clears today's usage of userID's active subscription in groupID and starts
// a reset day ending at the next daily rollover. Called by fulfillment, after payment.
func (s *SubscriptionService) ApplyDailyQuotaReset(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	today := timezone.StartOfDay(now)
	until := today.AddDate(0, 0, 1)
	res, err := s.entClient.ExecContext(ctx, `
		UPDATE user_subscriptions
		SET daily_usage_usd = 0,
			daily_window_start = $1,
			daily_reset_count = CASE WHEN daily_reset_until > NOW() THEN daily_reset_count + 1 ELSE 1 END,
			daily_reset_until = $2,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL`, today, until, sub.ID)
	if err != nil {
		return nil, fmt.Errorf("apply daily quota reset: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrSubscriptionNotFound
	}
	if err := s.invalidateSubscriptionCaches(userID, groupID); err != nil {
		slog.Warn("quota reset: cache invalidation failed", "userID", userID, "groupID", groupID, "error", err)
	}
	return s.userSubRepo.GetByID(ctx, sub.ID)
}

// ExecuteQuotaResetFulfillment applies a paid 重置今日额度 order. If the subscription is gone the
// order fails (paid) for the admin to refund.
func (s *PaymentService) ExecuteQuotaResetFulfillment(ctx context.Context, oid int64) error {
	o, err := s.entClient.PaymentOrder.Get(ctx, oid)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if o.Status == OrderStatusCompleted {
		return nil
	}
	if psIsRefundStatus(o.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusFailed && o.Status != OrderStatusRecharging {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+o.Status)
	}
	if o.SubscriptionGroupID == nil || *o.SubscriptionGroupID <= 0 {
		return infraerrors.BadRequest("INVALID_STATUS", "missing subscription group")
	}
	lease, err := s.acquirePaymentFulfillmentLease(ctx, o)
	if err != nil {
		return err
	}
	if lease == nil {
		return nil
	}
	if !s.hasAuditLog(ctx, o.ID, "QUOTA_RESET_APPLIED") {
		sub, err := s.subscriptionSvc.ApplyDailyQuotaReset(ctx, o.UserID, *o.SubscriptionGroupID)
		if err != nil {
			s.markFailed(ctx, oid, lease, err)
			return err
		}
		detail := map[string]any{"groupID": *o.SubscriptionGroupID, "subscriptionID": sub.ID}
		if sub.DailyResetUntil != nil {
			detail["until"] = sub.DailyResetUntil.Format(time.RFC3339)
		}
		s.writeAuditLog(ctx, o.ID, "QUOTA_RESET_APPLIED", "system", detail)
	}
	return s.markCompleted(ctx, o, lease, "QUOTA_RESET_SUCCESS")
}

// QuotaResetBalanceResult is returned after paying a reset with balance.
type QuotaResetBalanceResult struct {
	OrderID       int64   `json:"order_id"`
	BalanceCost   float64 `json:"balance_cost"`
	BalanceAfter  float64 `json:"balance_after"`
	Status        string  `json:"status"`
	FulfillmentOK bool    `json:"fulfillment_ok"`
}

// PurchaseQuotaResetWithBalance buys a reset for userID's subscription in groupID with balance.
func (s *PaymentService) PurchaseQuotaResetWithBalance(ctx context.Context, userID, groupID int64, clientIP, srcHost string) (*QuotaResetBalanceResult, error) {
	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get payment config: %w", err)
	}
	if !cfg.Enabled {
		return nil, infraerrors.Forbidden("PAYMENT_DISABLED", "payment system is disabled")
	}
	if cfg.SubscriptionBalancePayDisabled {
		return nil, ErrBalancePayDisabled
	}
	q, _, err := s.quoteForPurchase(ctx, userID, groupID, cfg)
	if err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Status != payment.EntityStatusActive {
		return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
	}
	if q.BalancePrice <= 0 {
		return nil, ErrQuotaResetUnavailable
	}
	order, after, err := s.createQuotaResetBalanceOrder(ctx, user, groupID, q.Price, q.BalancePrice, clientIP, srcHost)
	if err != nil {
		return nil, err
	}
	if s.redeemService != nil {
		s.redeemService.invalidateRedeemCaches(ctx, userID, &RedeemCode{Type: RedeemTypeBalance})
	}
	result := &QuotaResetBalanceResult{OrderID: order.ID, BalanceCost: q.BalancePrice, BalanceAfter: after}
	if err := s.ExecuteQuotaResetFulfillment(ctx, order.ID); err != nil {
		slog.Error("balance quota reset fulfillment failed", "orderID", order.ID, "userID", userID, "error", err)
		result.Status = OrderStatusFailed
		return result, nil
	}
	result.Status = OrderStatusCompleted
	result.FulfillmentOK = true
	return result, nil
}

// createQuotaResetBalanceOrder deducts the balance and records a PAID quota_reset order in one transaction.
func (s *PaymentService) createQuotaResetBalanceOrder(ctx context.Context, user *User, groupID int64, price, cost float64, clientIP, srcHost string) (*dbent.PaymentOrder, float64, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	n, err := tx.User.Update().
		Where(dbuser.IDEQ(user.ID), dbuser.BalanceGTE(cost)).
		AddBalance(-cost).
		Save(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("deduct balance: %w", err)
	}
	if n == 0 {
		return nil, 0, ErrBalancePayInsufficient
	}
	after, err := tx.User.Query().Where(dbuser.IDEQ(user.ID)).Only(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("reload balance: %w", err)
	}
	outTradeNo, err := s.allocateOutTradeNo(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	order, err := tx.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetNillableUserNotes(psNilIfEmpty(user.Notes)).
		SetAmount(price).
		SetPayAmount(cost).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo(outTradeNo).
		SetPaymentType(payment.TypeBalance).
		SetPaymentTradeNo(outTradeNo).
		SetOrderType(payment.OrderTypeQuotaReset).
		SetStatus(OrderStatusPaid).
		SetPaidAt(now).
		SetExpiresAt(now).
		SetClientIP(clientIP).
		SetSrcHost(srcHost).
		SetSubscriptionGroupID(groupID).
		Save(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("create order: %w", err)
	}
	order, err = tx.PaymentOrder.UpdateOneID(order.ID).SetRechargeCode(fmt.Sprintf("BAL-%d", order.ID)).Save(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("set recharge code: %w", err)
	}
	if _, err := tx.PaymentAuditLog.Create().
		SetOrderID(fmt.Sprintf("%d", order.ID)).
		SetAction("BALANCE_DEDUCTED").
		SetDetail(fmt.Sprintf(`{"balanceCost":%v,"balanceAfter":%v,"quotaResetGroupID":%d}`, cost, after.Balance, groupID)).
		SetOperator(fmt.Sprintf("user:%d", user.ID)).
		Save(ctx); err != nil {
		return nil, 0, fmt.Errorf("record balance deduction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit balance order: %w", err)
	}
	return order, after.Balance, nil
}
