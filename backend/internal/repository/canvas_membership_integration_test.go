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

func TestCanvasMembershipRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "member-" + suffix + "@member.test", Username: "mem" + suffix[len(suffix)-4:]})
	repo := NewCanvasMembershipRepository(integrationDB)

	until, err := repo.Until(ctx, user.ID)
	require.NoError(t, err)
	require.Nil(t, until)

	ok, err := repo.TermsAccepted(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, repo.AcceptTerms(ctx, user.ID))
	ok, err = repo.TermsAccepted(ctx, user.ID)
	require.NoError(t, err)
	require.True(t, ok)
	// Accepting the terms leaves a row that has already ended.
	until, err = repo.Until(ctx, user.ID)
	require.NoError(t, err)
	require.False(t, until.After(time.Now().Add(time.Minute)))

	first, err := repo.Extend(ctx, user.ID, 30, false)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 30), first, time.Minute)
	second, err := repo.Extend(ctx, user.ID, 30, false)
	require.NoError(t, err)
	require.WithinDuration(t, first.AddDate(0, 0, 30), second, time.Second, "a second order adds to the current end")
	ok, err = repo.TermsAccepted(ctx, user.ID)
	require.NoError(t, err)
	require.True(t, ok, "extending keeps the consent")

	require.NoError(t, repo.Shorten(ctx, user.ID, 30))
	until, err = repo.Until(ctx, user.ID)
	require.NoError(t, err)
	require.WithinDuration(t, first, *until, time.Second)
	require.NoError(t, repo.Shorten(ctx, user.ID, 365))
	until, err = repo.Until(ctx, user.ID)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), *until, time.Minute, "never below now")

	require.NoError(t, repo.LogUnmarkedSave(ctx, service.CanvasUnmarkedSave{UserID: user.ID, Width: 1024, Height: 1536, ClientIP: "1.2.3.4", UserAgent: "test"}))
	members, err := repo.ListMembers(ctx, false, 500)
	require.NoError(t, err)
	var found *service.CanvasMember
	for i := range members {
		if members[i].UserID == user.ID {
			found = &members[i]
		}
	}
	require.NotNil(t, found)
	require.Equal(t, int64(1), found.UnmarkedSaves)
	require.NotNil(t, found.TermsAcceptedAt)

	pruned, err := repo.PruneUnmarkedSaves(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(0), pruned, "recent saves are kept")
}

func TestCanvasSessionKeyHandoff(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "handoff-" + suffix + "@member.test", Username: "hof" + suffix[len(suffix)-4:]})
	repo := NewCanvasSessionRepository(integrationDB)
	now := time.Now()
	session := &service.CanvasSession{UserID: user.ID, UserAgent: "test", IP: "1.2.3.4", CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	require.NoError(t, repo.CreateCanvasSession(ctx, session, "hash-"+suffix))

	id, err := repo.TakeCanvasSessionHandoff(ctx, session.ID)
	require.NoError(t, err)
	require.Zero(t, id)

	require.NoError(t, repo.SetCanvasSessionHandoff(ctx, session.ID, 4242))
	id, err = repo.TakeCanvasSessionHandoff(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, int64(4242), id)
	id, err = repo.TakeCanvasSessionHandoff(ctx, session.ID)
	require.NoError(t, err)
	require.Zero(t, id, "the key is handed over once")
}

func TestCanvasMembershipOrderFulfillment(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	buyer := mustCreateUser(t, integrationEntClient, &service.User{Email: "mb-" + suffix + "@member.test", Username: "mb" + suffix[len(suffix)-6:]})
	membership := service.NewCanvasMembershipService(NewCanvasMembershipRepository(integrationDB), NewSettingRepository(integrationEntClient))
	payments := service.NewPaymentService(integrationEntClient, nil, nil, nil, nil, nil, nil, nil, nil)
	payments.SetCanvasMembershipService(membership)

	order, err := integrationEntClient.PaymentOrder.Create().
		SetUserID(buyer.ID).SetUserEmail(buyer.Email).SetUserName(buyer.Username).
		SetAmount(19.9).SetPayAmount(19.9).SetFeeRate(0).SetRechargeCode("").
		SetOutTradeNo("mb" + suffix).SetPaymentType(payment.TypeWxpay).SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeMembership).SetSubscriptionDays(30).SetStatus(service.OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	require.NoError(t, payments.ExecuteMembershipFulfillment(ctx, order.ID))
	require.NoError(t, payments.ExecuteMembershipFulfillment(ctx, order.ID)) // idempotent
	done, err := integrationEntClient.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, service.OrderStatusCompleted, done.Status)

	until, err := membership.Until(ctx, buyer.ID)
	require.NoError(t, err)
	require.NotNil(t, until)
	require.WithinDuration(t, time.Now().AddDate(0, 0, 30), *until, time.Minute, "the days are added once")
}
