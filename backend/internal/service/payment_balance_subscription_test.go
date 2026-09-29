//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type balanceSubscriptionFixture struct {
	ctx      context.Context
	client   *dbent.Client
	svc      *PaymentService
	settings *paymentConfigSettingRepoStub
	subRepo  *subscriptionUserSubRepoStub
	userID   int64
	planID   int64
}

func newBalanceSubscriptionFixture(t *testing.T, balance, price float64) *balanceSubscriptionFixture {
	t.Helper()
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)

	u, err := client.User.Create().
		SetEmail("balance-sub@example.com").
		SetPasswordHash("hash").
		SetUsername("balance-sub").
		SetBalance(balance).
		Save(ctx)
	require.NoError(t, err)
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(7).
		SetName("Monthly").
		SetPrice(price).
		SetValidityDays(30).
		SetValidityUnit("days").
		SetForSale(true).
		Save(ctx)
	require.NoError(t, err)

	settings := &paymentConfigSettingRepoStub{values: map[string]string{SettingPaymentEnabled: "true"}}
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 7, Status: payment.EntityStatusActive, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	svc := &PaymentService{
		entClient:       client,
		configService:   &PaymentConfigService{entClient: client, settingRepo: settings},
		userRepo:        &userRepoStub{usersByID: map[int64]*User{u.ID: {ID: u.ID, Email: u.Email, Username: u.Username, Status: payment.EntityStatusActive}}},
		groupRepo:       groupRepo,
		subscriptionSvc: NewSubscriptionService(groupRepo, subRepo, nil, nil, nil),
	}
	return &balanceSubscriptionFixture{ctx: ctx, client: client, svc: svc, settings: settings, subRepo: subRepo, userID: u.ID, planID: plan.ID}
}

func (f *balanceSubscriptionFixture) balance(t *testing.T) float64 {
	t.Helper()
	u, err := f.client.User.Get(f.ctx, f.userID)
	require.NoError(t, err)
	return u.Balance
}

func TestPurchaseSubscriptionWithBalance_DeductsAndAssigns(t *testing.T) {
	f := newBalanceSubscriptionFixture(t, 150, 120)

	res, err := f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "127.0.0.1", "hivegpt.cn")
	require.NoError(t, err)
	require.True(t, res.FulfillmentOK)
	require.Equal(t, OrderStatusCompleted, res.Status)
	require.InDelta(t, 120, res.BalanceCost, 1e-9)
	require.InDelta(t, 30, res.BalanceAfter, 1e-9)
	require.InDelta(t, 30, f.balance(t), 1e-9)

	order, err := f.client.PaymentOrder.Get(f.ctx, res.OrderID)
	require.NoError(t, err)
	require.Equal(t, payment.TypeBalance, order.PaymentType)
	require.Equal(t, payment.OrderTypeSubscription, order.OrderType)
	require.Equal(t, OrderStatusCompleted, order.Status)
	require.InDelta(t, 120, order.PayAmount, 1e-9)

	sub, err := f.subRepo.GetByUserIDAndGroupID(f.ctx, f.userID, 7)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, sub.Status)
}

func TestPurchaseSubscriptionWithBalance_InsufficientBalanceChangesNothing(t *testing.T) {
	f := newBalanceSubscriptionFixture(t, 100, 120)

	_, err := f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "", "")
	require.ErrorIs(t, err, ErrBalancePayInsufficient)
	require.InDelta(t, 100, f.balance(t), 1e-9)
	count, err := f.client.PaymentOrder.Query().Where(paymentorder.UserIDEQ(f.userID)).Count(f.ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestPurchaseSubscriptionWithBalance_SecondPurchaseCannotOverdraw(t *testing.T) {
	f := newBalanceSubscriptionFixture(t, 150, 120)

	_, err := f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "", "")
	require.NoError(t, err)
	_, err = f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "", "")
	require.ErrorIs(t, err, ErrBalancePayInsufficient)
	require.InDelta(t, 30, f.balance(t), 1e-9)
}

func TestPurchaseSubscriptionWithBalance_Disabled(t *testing.T) {
	f := newBalanceSubscriptionFixture(t, 150, 120)
	f.settings.values[SettingSubscriptionBalancePayDisabled] = "true"

	_, err := f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "", "")
	require.ErrorIs(t, err, ErrBalancePayDisabled)
	require.InDelta(t, 150, f.balance(t), 1e-9)
}

func TestBalancePaidOrders_NoRebateAndNoGatewayRefund(t *testing.T) {
	f := newBalanceSubscriptionFixture(t, 150, 120)
	res, err := f.svc.PurchaseSubscriptionWithBalance(f.ctx, f.userID, f.planID, "", "")
	require.NoError(t, err)
	order, err := f.client.PaymentOrder.Get(f.ctx, res.OrderID)
	require.NoError(t, err)

	require.Zero(t, affiliateRebateBaseAmount(order))
	_, _, err = f.svc.PrepareRefund(f.ctx, order.ID, 0, "", false, true)
	require.ErrorContains(t, err, "balance-paid")
}

func TestSubscriptionBalancePrice(t *testing.T) {
	// Default: price is paid 1:1 and credited 1:1.
	require.InDelta(t, 120, subscriptionBalancePrice(120, &PaymentConfig{BalanceRechargeMultiplier: 1}), 1e-9)
	// USD plan converted to CNY, then credited at the recharge multiplier: 10 × 7.2 × 0.5.
	require.InDelta(t, 36, subscriptionBalancePrice(10, &PaymentConfig{SubscriptionUSDToCNYRate: 7.2, BalanceRechargeMultiplier: 0.5}), 1e-9)
	require.Zero(t, subscriptionBalancePrice(0, &PaymentConfig{BalanceRechargeMultiplier: 1}))
}

type growthRepoBonusRecorder struct {
	GrowthRepository
	grants []float64
}

func (r *growthRepoBonusRecorder) GrantInviteeBonus(_ context.Context, _, _ int64, amount float64, _ *time.Time) (bool, int64, error) {
	r.grants = append(r.grants, amount)
	return true, 1, nil
}

func TestApplyInviteeFirstOrderBonus_GatewayOrdersOnly(t *testing.T) {
	repo := &growthRepoBonusRecorder{}
	settings := &paymentConfigSettingRepoStub{values: map[string]string{SettingKeyGrowthInviteeBonusRate: "10"}}
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &PaymentService{entClient: client, growthService: NewGrowthService(settings, repo, nil, nil, nil, nil)}

	svc.applyInviteeFirstOrderBonus(ctx, &dbent.PaymentOrder{ID: 1, UserID: 2, Amount: 120, PaymentType: payment.TypeBalance})
	require.Empty(t, repo.grants, "balance-paid orders never earn the invitee bonus")

	svc.applyInviteeFirstOrderBonus(ctx, &dbent.PaymentOrder{ID: 2, UserID: 2, Amount: 120, PaymentType: payment.TypeWxpay})
	require.Equal(t, []float64{12}, repo.grants)
}
