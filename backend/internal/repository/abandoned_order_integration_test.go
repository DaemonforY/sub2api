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

func TestAbandonedOrderRepositoryDue(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewAbandonedOrderRepository(integrationDB)
	mk := func(tag string) *service.User {
		return rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "ab-" + tag + "-" + sfx + "@test.local", Username: "ab" + tag + sfx[len(sfx)-6:]}))
	}
	now := time.Now().UTC()
	order := func(u *service.User, status, orderType string, created, expires time.Time) int64 {
		n := time.Now().UnixNano()
		o, err := integrationEntClient.PaymentOrder.Create().
			SetUserID(u.ID).SetUserEmail(u.Email).SetUserName(u.Username).
			SetAmount(50).SetPayAmount(50).SetFeeRate(0).
			SetRechargeCode(fmt.Sprintf("AB-%d", n)).SetOutTradeNo(fmt.Sprintf("ab_%d", n)).
			SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("t").SetOrderType(orderType).SetStatus(status).
			SetCreatedAt(created).SetExpiresAt(expires).SetClientIP("127.0.0.1").SetSrcHost("test").
			Save(ctx)
		require.NoError(t, err)
		return o.ID
	}

	// Gave up and never came back: due.
	gaveUp := mk("gaveup")
	gaveUpOrder := order(gaveUp, service.OrderStatusExpired, payment.OrderTypeBalance, now.Add(-3*time.Hour), now.Add(-150*time.Minute))
	// Expired, then paid with a second order: not due.
	paidLater := mk("paidlater")
	order(paidLater, service.OrderStatusExpired, payment.OrderTypeBalance, now.Add(-3*time.Hour), now.Add(-150*time.Minute))
	order(paidLater, service.OrderStatusCompleted, payment.OrderTypeBalance, now.Add(-2*time.Hour), now.Add(-90*time.Minute))
	// Cancelled on purpose: not due. Course orders: not due.
	cancelled := mk("cancelled")
	order(cancelled, service.OrderStatusCancelled, payment.OrderTypeBalance, now.Add(-3*time.Hour), now.Add(-150*time.Minute))
	course := mk("course")
	order(course, service.OrderStatusExpired, payment.OrderTypeCourse, now.Add(-3*time.Hour), now.Add(-150*time.Minute))
	// Expired only 10 minutes ago: not yet.
	recent := mk("recent")
	order(recent, service.OrderStatusExpired, payment.OrderTypeSubscription, now.Add(-40*time.Minute), now.Add(-10*time.Minute))

	due, err := repo.DueAbandonedOrders(ctx, now.Add(-24*time.Hour), now.Add(-time.Hour), 500)
	require.NoError(t, err)
	mine := map[int64]service.AbandonedOrder{}
	for _, o := range due {
		mine[o.UserID] = o
	}
	require.Contains(t, mine, gaveUp.ID)
	require.Equal(t, gaveUpOrder, mine[gaveUp.ID].OrderID)
	require.InDelta(t, 50, mine[gaveUp.ID].Amount, 1e-9)
	for _, u := range []*service.User{paidLater, cancelled, course, recent} {
		require.NotContains(t, mine, u.ID)
	}
}
