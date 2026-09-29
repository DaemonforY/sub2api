package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Buying a subscription plan with account balance.
//
// Balance-paid orders go through the regular subscription fulfillment (assignment,
// idempotency, notification) but never touch a payment gateway:
//   - the balance is deducted atomically in the same transaction that records the order,
//     and only if it covers the full price (no overdraft);
//   - they earn no affiliate rebate — the balance was already rebated when it was recharged;
//   - they are excluded from revenue statistics, for the same reason;
//   - they have no provider instance, so the gateway refund flow rejects them.

// ErrBalancePayDisabled is returned when buying subscriptions with balance is turned off.
var ErrBalancePayDisabled = infraerrors.Forbidden("BALANCE_SUBSCRIPTION_DISABLED", "buying subscriptions with balance is disabled")

// ErrBalancePayInsufficient is returned when the user's balance does not cover the plan.
var ErrBalancePayInsufficient = infraerrors.BadRequest("BALANCE_INSUFFICIENT_FOR_PLAN", "balance is insufficient for this plan")

// BalanceSubscriptionResult is returned after a successful balance purchase.
type BalanceSubscriptionResult struct {
	OrderID       int64   `json:"order_id"`
	BalanceCost   float64 `json:"balance_cost"`
	BalanceAfter  float64 `json:"balance_after"`
	Status        string  `json:"status"`
	GroupID       int64   `json:"group_id"`
	ValidityDays  int     `json:"validity_days"`
	FulfillmentOK bool    `json:"fulfillment_ok"`
}

// subscriptionBalancePrice converts a plan price into balance: the amount the gateway
// would charge before fees, credited at the recharge multiplier — i.e. exactly the
// balance a user would receive by recharging that amount.
func subscriptionBalancePrice(price float64, cfg *PaymentConfig) float64 {
	if cfg == nil || price <= 0 {
		return 0
	}
	base := calculateSubscriptionGatewayBaseAmount(price, cfg.SubscriptionUSDToCNYRate, payment.DefaultPaymentCurrency)
	return calculateCreditedBalance(base, cfg.BalanceRechargeMultiplier)
}

// SubscriptionBalancePrice exposes the balance price of a plan for display.
func (s *PaymentService) SubscriptionBalancePrice(price float64, cfg *PaymentConfig) float64 {
	return subscriptionBalancePrice(price, cfg)
}

// PurchaseSubscriptionWithBalance buys a plan for userID, paying with account balance.
func (s *PaymentService) PurchaseSubscriptionWithBalance(ctx context.Context, userID, planID int64, clientIP, srcHost string) (*BalanceSubscriptionResult, error) {
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
	plan, err := s.validateSubOrder(ctx, CreateOrderRequest{UserID: userID, OrderType: payment.OrderTypeSubscription, PlanID: planID})
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

	price := s.planPriceForUser(ctx, plan, userID)
	cost := subscriptionBalancePrice(price, cfg)
	if cost <= 0 {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", "plan price must be positive")
	}

	order, balanceAfter, err := s.createBalancePaidOrder(ctx, user, plan, price, cost, clientIP, srcHost)
	if err != nil {
		return nil, err
	}
	if s.redeemService != nil {
		s.redeemService.invalidateRedeemCaches(ctx, userID, &RedeemCode{Type: RedeemTypeBalance})
	}

	result := &BalanceSubscriptionResult{
		OrderID:      order.ID,
		BalanceCost:  cost,
		BalanceAfter: balanceAfter,
		GroupID:      plan.GroupID,
		ValidityDays: psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit),
	}
	// The balance is already paid; a fulfillment failure leaves the order FAILED so the
	// admin "retry fulfillment" action can finish it, exactly like a gateway order.
	if err := s.ExecuteSubscriptionFulfillment(ctx, order.ID); err != nil {
		slog.Error("balance subscription fulfillment failed", "orderID", order.ID, "userID", userID, "error", err)
		result.Status = OrderStatusFailed
		return result, nil
	}
	result.Status = OrderStatusCompleted
	result.FulfillmentOK = true
	return result, nil
}

// createBalancePaidOrder deducts the balance and records a PAID order in one transaction.
func (s *PaymentService) createBalancePaidOrder(ctx context.Context, user *User, plan *dbent.SubscriptionPlan, price, cost float64, clientIP, srcHost string) (*dbent.PaymentOrder, float64, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Conditional update: never overdraws, and serializes concurrent purchases on the row lock.
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
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusPaid).
		SetPaidAt(now).
		SetExpiresAt(now).
		SetClientIP(clientIP).
		SetSrcHost(srcHost).
		SetPlanID(plan.ID).
		SetSubscriptionGroupID(plan.GroupID).
		SetSubscriptionDays(psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit)).
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
		SetDetail(fmt.Sprintf(`{"balanceCost":%v,"balanceAfter":%v,"planID":%d}`, cost, after.Balance, plan.ID)).
		SetOperator(fmt.Sprintf("user:%d", user.ID)).
		Save(ctx); err != nil {
		return nil, 0, fmt.Errorf("record balance deduction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, fmt.Errorf("commit balance order: %w", err)
	}
	return order, after.Balance, nil
}

// isBalancePaidOrder reports whether an order was paid from account balance.
func isBalancePaidOrder(o *dbent.PaymentOrder) bool {
	return o != nil && o.PaymentType == payment.TypeBalance
}

// planPriceForUser returns the price userID pays for plan (education discount applied).
func (s *PaymentService) planPriceForUser(ctx context.Context, plan *dbent.SubscriptionPlan, userID int64) float64 {
	price, _ := s.PlanPriceForUser(ctx, plan, userID)
	return price
}

// PlanPriceForUser returns the price userID pays for plan and whether the education
// discount was applied. userID 0 (anonymous) always gets the list price.
func (s *PaymentService) PlanPriceForUser(ctx context.Context, plan *dbent.SubscriptionPlan, userID int64) (float64, bool) {
	if plan == nil {
		return 0, false
	}
	if s.eduService == nil || userID <= 0 {
		return plan.Price, false
	}
	return s.eduService.DiscountedPrice(ctx, userID, plan.Price)
}
