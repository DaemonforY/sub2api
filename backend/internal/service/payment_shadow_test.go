//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder_RestrictedAccountCannotPay(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	config := NewPaymentConfigService(client, &settingRepoStub{values: map[string]string{SettingPaymentEnabled: "true"}}, nil)
	shadow, _ := newShadowServiceForTest(growthForTest(nil)) // 不能在线支付 is on by default
	svc := &PaymentService{entClient: client, configService: config}
	svc.SetShadowGuard(shadow)
	_, err := shadow.Shadow(ctx, 7, "same browser as #48", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)

	for _, orderType := range []string{payment.OrderTypeBalance, payment.OrderTypeSubscription, payment.OrderTypeCourse, payment.OrderTypeMembership} {
		_, err = svc.CreateOrder(ctx, CreateOrderRequest{UserID: 7, Amount: 50, PaymentType: payment.TypeAlipay, OrderType: orderType})
		require.ErrorIs(t, err, ErrOnlinePaymentUnavailable, orderType)
		require.Contains(t, err.Error(), "该账号暂不支持在线支付，请联系客服")
		require.NotContains(t, err.Error(), "限制")
		require.NotContains(t, err.Error(), "风控")
	}

	// Other accounts, and every account once the 推广设置 switch is off, go on to the usual checks
	// (CreateOrder only asks the guard).
	require.False(t, shadow.BlocksPayment(ctx, 8))
	off, _ := newShadowServiceForTest(growthForTest(func(s *GrowthSettings) { s.ShadowBlockPayments = false }))
	_, err = off.Shadow(ctx, 7, "x", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)
	require.False(t, off.BlocksPayment(ctx, 7))
}

func TestToPaid_OrderPaidBeforeRestrictionIsStillFulfilled(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	user, err := client.User.Create().SetEmail("late-payer@example.com").SetPasswordHash("hash").SetUsername("late-payer").Save(ctx)
	require.NoError(t, err)
	// Created before the restriction, paid after it.
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).
		SetAmount(199).SetPayAmount(199).SetFeeRate(0).SetRechargeCode("").
		SetOutTradeNo("shadow_paid_1").SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeCourse).SetCourseID(5).SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("203.0.113.7").SetSrcHost("hivegpt.cn").
		Save(ctx)
	require.NoError(t, err)

	shadow, shadowRepo := newShadowServiceForTest(growthForTest(nil))
	_, err = shadow.Shadow(ctx, user.ID, "same browser as #48", ShadowSourceAdmin, nil, nil)
	require.NoError(t, err)
	repo := &courseEnrollRepoStub{}
	courses := NewCourseService(repo, nil, nil)
	courses.SetSettings(&courseSettingsStub{values: map[string]string{settingCourseAffiliateRate: "0"}})
	svc := &PaymentService{entClient: client, courses: courses}
	svc.SetShadowGuard(shadow)

	require.NoError(t, svc.toPaid(ctx, order, "wx-trade-1", 199, payment.TypeWxpay))

	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, reloaded.Status, "the money was taken, so the order is fulfilled")
	require.Len(t, repo.enrolled, 1)
	n, err := client.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ("PAID_WHILE_SHADOWED")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n, "the admin can see it in the order's audit log")
	require.Len(t, shadowRepo.notes, 1)
	require.Contains(t, shadowRepo.notes[0], RiskRuleShadowPaid)
	require.Contains(t, shadowRepo.notes[0], "¥199.00")
}
