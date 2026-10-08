//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateWithdrawRepository_Lifecycle(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	client := integrationEntClient
	affRepo := NewAffiliateRepository(client, integrationDB)
	repo := NewAffiliateWithdrawRepository(client)

	inviter := mustCreateUser(t, client, &service.User{Email: "wd-inviter-" + sfx + "@test.local", Username: "wdi" + sfx[len(sfx)-6:]})
	invitee := mustCreateUser(t, client, &service.User{Email: "wd-invitee-" + sfx + "@test.local", Username: "wde" + sfx[len(sfx)-6:]})
	_, err := affRepo.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)

	order := func(tag, paymentType, status string, amount, pay, fee float64) int64 {
		o, err := client.PaymentOrder.Create().
			SetUserID(invitee.ID).SetUserEmail(invitee.Email).SetUserName(invitee.Username).
			SetAmount(amount).SetPayAmount(pay).SetFeeRate(fee).
			SetRechargeCode("WD-" + tag + sfx).SetOutTradeNo("wd_" + tag + sfx).
			SetPaymentType(paymentType).SetPaymentTradeNo("t").
			SetOrderType(payment.OrderTypeBalance).SetStatus(status).
			SetPaidAt(time.Now()).SetExpiresAt(time.Now()).SetClientIP("127.0.0.1").SetSrcHost("test").
			Save(ctx)
		require.NoError(t, err)
		return o.ID
	}
	// ¥72 + 2% fee bought $10 → rebate $2 is worth ¥14.40.
	cashOrder := order("a", "alipay", service.OrderStatusCompleted, 10, 73.44, 2)
	_, err = affRepo.AccrueQuota(ctx, inviter.ID, invitee.ID, 2, 0, &cashOrder)
	require.NoError(t, err)
	// Not cash: refunded order, balance-paid order, still frozen, and no order at all (admin top-up).
	refunded := order("b", "wxpay", service.OrderStatusRefunded, 10, 72, 0)
	_, err = affRepo.AccrueQuota(ctx, inviter.ID, invitee.ID, 2, 0, &refunded)
	require.NoError(t, err)
	byBalance := order("c", "balance", service.OrderStatusCompleted, 10, 10, 0)
	_, err = affRepo.AccrueQuota(ctx, inviter.ID, invitee.ID, 2, 0, &byBalance)
	require.NoError(t, err)
	frozen := order("d", "wxpay", service.OrderStatusCompleted, 10, 72, 0)
	_, err = affRepo.AccrueQuota(ctx, inviter.ID, invitee.ID, 2, 24, &frozen)
	require.NoError(t, err)
	_, err = affRepo.AccrueQuota(ctx, inviter.ID, invitee.ID, 1, 0, nil)
	require.NoError(t, err)

	month := time.Now().AddDate(0, 0, -1)
	e, err := repo.Eligibility(ctx, inviter.ID, month)
	require.NoError(t, err)
	require.InDelta(t, 7, e.AvailableQuota, 1e-6) // 2 + 2 + 2 + 1 (the frozen 2 isn't available yet)
	require.InDelta(t, 2, e.CashQuota, 1e-6)
	require.InDelta(t, 14.4, e.CashCNY, 1e-6)
	require.False(t, e.HasPending)

	payee := service.NewWithdrawal{CNYAmount: 7.2, Method: "alipay", Account: "enc-acc", RealName: "enc-name"}
	w, err := repo.Create(ctx, inviter.ID, payee, month, func(e *service.WithdrawEligibility) (float64, error) { return 1, nil })
	require.NoError(t, err)
	require.Equal(t, service.WithdrawStatusPending, w.Status)
	require.InDelta(t, 6, querySingleFloat(t, ctx, client, `SELECT aff_quota FROM user_affiliates WHERE user_id = $1`, inviter.ID), 1e-6)
	require.Equal(t, 1, querySingleInt(t, ctx, client, `SELECT COUNT(*) FROM user_affiliate_ledger WHERE withdrawal_id = $1 AND action = 'withdraw'`, w.ID))

	e, err = repo.Eligibility(ctx, inviter.ID, month)
	require.NoError(t, err)
	require.True(t, e.HasPending)
	require.Equal(t, 1, e.MonthCount)
	require.InDelta(t, 7.2, e.WithdrawnCNY, 1e-6)

	// The plan sees the pending one; the unique index is the backstop.
	_, err = repo.Create(ctx, inviter.ID, payee, month, func(e *service.WithdrawEligibility) (float64, error) {
		require.True(t, e.HasPending)
		return 0, service.ErrWithdrawPending
	})
	require.ErrorIs(t, err, service.ErrWithdrawPending)
	_, err = repo.Create(ctx, inviter.ID, payee, month, func(*service.WithdrawEligibility) (float64, error) { return 1, nil })
	require.ErrorIs(t, err, service.ErrWithdrawPending)
	require.InDelta(t, 6, querySingleFloat(t, ctx, client, `SELECT aff_quota FROM user_affiliates WHERE user_id = $1`, inviter.ID), 1e-6, "rolled back")

	// Someone else can't cancel it; the owner can, and the quota comes back.
	_, err = repo.Return(ctx, w.ID, service.WithdrawStatusCancelled, "", 0, invitee.ID)
	require.ErrorIs(t, err, service.ErrWithdrawNotFound)
	back, err := repo.Return(ctx, w.ID, service.WithdrawStatusCancelled, "", 0, inviter.ID)
	require.NoError(t, err)
	require.Equal(t, service.WithdrawStatusCancelled, back.Status)
	require.InDelta(t, 7, querySingleFloat(t, ctx, client, `SELECT aff_quota FROM user_affiliates WHERE user_id = $1`, inviter.ID), 1e-6)
	_, err = repo.Return(ctx, w.ID, service.WithdrawStatusCancelled, "", 0, inviter.ID)
	require.ErrorIs(t, err, service.ErrWithdrawNotPending)

	// Can't deduct more than is available.
	_, err = repo.Create(ctx, inviter.ID, payee, month, func(*service.WithdrawEligibility) (float64, error) { return 100, nil })
	require.ErrorIs(t, err, service.ErrWithdrawOverAvailable)

	// A new one gets paid.
	w2, err := repo.Create(ctx, inviter.ID, payee, month, func(*service.WithdrawEligibility) (float64, error) { return 1, nil })
	require.NoError(t, err)
	paid, err := repo.MarkPaid(ctx, w2.ID, "支付宝流水 123", inviter.ID)
	require.NoError(t, err)
	require.Equal(t, service.WithdrawStatusPaid, paid.Status)
	require.NotNil(t, paid.ReviewedAt)
	_, err = repo.MarkPaid(ctx, w2.ID, "", inviter.ID)
	require.ErrorIs(t, err, service.ErrWithdrawNotPending)
	_, err = repo.MarkPaid(ctx, -1, "", inviter.ID)
	require.ErrorIs(t, err, service.ErrWithdrawNotFound)

	e, err = repo.Eligibility(ctx, inviter.ID, month)
	require.NoError(t, err)
	require.False(t, e.HasPending)
	require.Equal(t, 1, e.MonthCount, "cancelled ones don't count")
	require.InDelta(t, 7.2, e.WithdrawnCNY, 1e-6)

	list, total, err := repo.List(ctx, service.WithdrawFilter{Search: inviter.Email, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Equal(t, inviter.Email, list[0].UserEmail)
	list, total, err = repo.List(ctx, service.WithdrawFilter{Status: service.WithdrawStatusPaid, Search: fmt.Sprint(inviter.ID), Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, w2.ID, list[0].ID)
	mine, err := repo.ListByUser(ctx, inviter.ID, 10)
	require.NoError(t, err)
	require.Len(t, mine, 2)
	require.Equal(t, w2.ID, mine[0].ID)
}
