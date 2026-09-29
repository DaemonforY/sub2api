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

type growthTestEnv struct {
	ctx      context.Context
	repo     service.GrowthRepository
	affRepo  service.AffiliateRepository
	settings service.SettingRepository
	svc      *service.GrowthService
	suffix   string
}

func newGrowthTestEnv(t *testing.T) *growthTestEnv {
	t.Helper()
	settings := NewSettingRepository(integrationEntClient)
	repo := NewGrowthRepository(integrationDB)
	settingService := service.NewSettingService(settings, nil)
	return &growthTestEnv{
		ctx:      context.Background(),
		repo:     repo,
		affRepo:  NewAffiliateRepository(integrationEntClient, integrationDB),
		settings: settings,
		svc:      service.NewGrowthService(settings, repo, settingService, nil, nil, nil),
		suffix:   fmt.Sprintf("%d", time.Now().UnixNano()),
	}
}

func (e *growthTestEnv) user(t *testing.T, name string) *service.User {
	t.Helper()
	return mustCreateUser(t, integrationEntClient, &service.User{Email: name + "-" + e.suffix + "@growth.test", Username: name + e.suffix[len(e.suffix)-4:]})
}

// invite binds invitee to inviter and dates the binding at boundAt.
func (e *growthTestEnv) invite(t *testing.T, inviter, invitee *service.User, boundAt time.Time) {
	t.Helper()
	_, err := e.affRepo.EnsureUserAffiliate(e.ctx, inviter.ID)
	require.NoError(t, err)
	_, err = e.affRepo.EnsureUserAffiliate(e.ctx, invitee.ID)
	require.NoError(t, err)
	bound, err := e.affRepo.BindInviter(e.ctx, invitee.ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, bound)
	_, err = integrationDB.ExecContext(e.ctx, `UPDATE user_affiliates SET created_at = $1 WHERE user_id = $2`, boundAt, invitee.ID)
	require.NoError(t, err)
}

// paidOrder records a paid order (gateway unless paymentType is balance) and returns its id.
func (e *growthTestEnv) paidOrder(t *testing.T, u *service.User, amount float64, paymentType string, paidAt time.Time) int64 {
	t.Helper()
	n := time.Now().UnixNano()
	o, err := integrationEntClient.PaymentOrder.Create().
		SetUserID(u.ID).SetUserEmail(u.Email).SetUserName(u.Username).
		SetAmount(amount).SetPayAmount(amount).SetFeeRate(0).
		SetRechargeCode(fmt.Sprintf("GROWTH-%d", n)).SetOutTradeNo(fmt.Sprintf("growth_%d", n)).
		SetPaymentType(paymentType).SetPaymentTradeNo("t").
		SetOrderType(payment.OrderTypeBalance).SetStatus(service.OrderStatusCompleted).
		SetPaidAt(paidAt).SetExpiresAt(paidAt).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(e.ctx)
	require.NoError(t, err)
	return o.ID
}

func (e *growthTestEnv) balance(t *testing.T, userID int64) float64 {
	t.Helper()
	var b float64
	require.NoError(t, integrationDB.QueryRowContext(e.ctx, `SELECT balance FROM users WHERE id = $1`, userID).Scan(&b))
	return b
}

func (e *growthTestEnv) set(t *testing.T, values map[string]string) {
	t.Helper()
	require.NoError(t, e.settings.SetMultiple(e.ctx, values))
}

func TestGrowthEduVerification(t *testing.T) {
	e := newGrowthTestEnv(t)
	alice, bob := e.user(t, "alice"), e.user(t, "bob")
	email := "s" + e.suffix + "@pku.edu.cn"

	v, err := e.repo.UpsertEduVerification(e.ctx, alice.ID, email)
	require.NoError(t, err)
	require.Equal(t, email, v.Email)

	owner, err := e.repo.EduEmailOwner(e.ctx, "S"+e.suffix+"@PKU.edu.cn")
	require.NoError(t, err)
	require.Equal(t, alice.ID, owner, "owner lookup is case-insensitive")

	_, err = e.repo.UpsertEduVerification(e.ctx, bob.ID, email)
	require.ErrorIs(t, err, service.ErrEduEmailTaken, "one school email verifies one account")

	// Re-verifying with another address replaces the user's row.
	other := "t" + e.suffix + "@mail.tsinghua.edu.cn"
	_, err = e.repo.UpsertEduVerification(e.ctx, alice.ID, other)
	require.NoError(t, err)
	got, err := e.repo.GetEduVerification(e.ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, other, got.Email)

	items, total, err := e.repo.ListEduVerifications(e.ctx, "t"+e.suffix, 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, alice.ID, items[0].UserID)
	require.Equal(t, alice.Email, items[0].UserEmail)

	removed, err := e.repo.DeleteEduVerification(e.ctx, alice.ID)
	require.NoError(t, err)
	require.True(t, removed)
	got, err = e.repo.GetEduVerification(e.ctx, alice.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestGrowthEduDiscountAppliesOnlyToVerifiedUsers(t *testing.T) {
	e := newGrowthTestEnv(t)
	e.set(t, map[string]string{
		service.SettingKeyEduVerifyEnabled:        "true",
		service.SettingKeyEduSubscriptionDiscount: "20",
	})
	t.Cleanup(func() { e.set(t, map[string]string{service.SettingKeyEduVerifyEnabled: "false"}) })
	student, other := e.user(t, "student"), e.user(t, "other")
	_, err := e.repo.UpsertEduVerification(e.ctx, student.ID, "d"+e.suffix+"@fudan.edu.cn")
	require.NoError(t, err)

	price, ok := e.svc.DiscountedPrice(e.ctx, student.ID, 120)
	require.True(t, ok)
	require.InDelta(t, 96, price, 1e-9)
	price, ok = e.svc.DiscountedPrice(e.ctx, other.ID, 120)
	require.False(t, ok)
	require.InDelta(t, 120, price, 1e-9)
}

func TestGrowthInviteeFirstOrderBonus(t *testing.T) {
	e := newGrowthTestEnv(t)
	e.set(t, map[string]string{
		service.SettingKeyAffiliateEnabled:            "true",
		service.SettingKeyAffiliateRebateDurationDays: "0",
		service.SettingKeyGrowthInviteeBonusRate:      "10",
		service.SettingKeyGrowthInviteeBonusCap:       "5",
	})
	t.Cleanup(func() {
		e.set(t, map[string]string{service.SettingKeyAffiliateEnabled: "false", service.SettingKeyGrowthInviteeBonusRate: "0"})
	})
	now := time.Now().UTC()
	inviter, invitee, loner := e.user(t, "inviter"), e.user(t, "invitee"), e.user(t, "loner")
	e.invite(t, inviter, invitee, now.Add(-time.Hour))

	// A balance-paid order is not a "first order" and earns nothing.
	balanceOrder := e.paidOrder(t, invitee, 100, payment.TypeBalance, now.Add(-30*time.Minute))
	amount, err := e.svc.GrantInviteeFirstOrderBonus(e.ctx, invitee.ID, balanceOrder, 100)
	require.NoError(t, err)
	require.Zero(t, amount)

	first := e.paidOrder(t, invitee, 100, payment.TypeWxpay, now.Add(-20*time.Minute))
	before := e.balance(t, invitee.ID)
	amount, err = e.svc.GrantInviteeFirstOrderBonus(e.ctx, invitee.ID, first, 100)
	require.NoError(t, err)
	require.InDelta(t, 5, amount, 1e-9, "10% of 100, capped at 5")
	require.InDelta(t, before+5, e.balance(t, invitee.ID), 1e-9)

	// Retries of the same order and later orders never grant again.
	amount, err = e.svc.GrantInviteeFirstOrderBonus(e.ctx, invitee.ID, first, 100)
	require.NoError(t, err)
	require.Zero(t, amount)
	second := e.paidOrder(t, invitee, 100, payment.TypeWxpay, now.Add(-10*time.Minute))
	amount, err = e.svc.GrantInviteeFirstOrderBonus(e.ctx, invitee.ID, second, 100)
	require.NoError(t, err)
	require.Zero(t, amount)
	require.InDelta(t, before+5, e.balance(t, invitee.ID), 1e-9)

	// Users without an inviter get nothing.
	lonerOrder := e.paidOrder(t, loner, 100, payment.TypeWxpay, now)
	amount, err = e.svc.GrantInviteeFirstOrderBonus(e.ctx, loner.ID, lonerOrder, 100)
	require.NoError(t, err)
	require.Zero(t, amount)
}

func TestGrowthInviteeBonusRespectsRebateWindow(t *testing.T) {
	e := newGrowthTestEnv(t)
	e.set(t, map[string]string{
		service.SettingKeyAffiliateEnabled:            "true",
		service.SettingKeyAffiliateRebateDurationDays: "30",
		service.SettingKeyGrowthInviteeBonusRate:      "10",
		service.SettingKeyGrowthInviteeBonusCap:       "0",
	})
	t.Cleanup(func() {
		e.set(t, map[string]string{service.SettingKeyAffiliateEnabled: "false", service.SettingKeyGrowthInviteeBonusRate: "0", service.SettingKeyAffiliateRebateDurationDays: "0"})
	})
	inviter, invitee := e.user(t, "old-inviter"), e.user(t, "old-invitee")
	e.invite(t, inviter, invitee, time.Now().AddDate(0, 0, -40))
	order := e.paidOrder(t, invitee, 50, payment.TypeWxpay, time.Now())
	amount, err := e.svc.GrantInviteeFirstOrderBonus(e.ctx, invitee.ID, order, 50)
	require.NoError(t, err)
	require.Zero(t, amount, "bound 40 days ago, outside the 30-day window")
}

func TestGrowthInviteLeaderboard(t *testing.T) {
	e := newGrowthTestEnv(t)
	// A private window far from other tests' data.
	start := time.Date(2031, 3, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(time.Now().UnixNano()%1_000_000) * time.Second)
	end := start.Add(24 * time.Hour)
	in := start.Add(time.Hour)

	top, mid, quiet := e.user(t, "top"), e.user(t, "mid"), e.user(t, "quiet")
	for i := 0; i < 3; i++ {
		invitee := e.user(t, fmt.Sprintf("top-inv%d", i))
		e.invite(t, top, invitee, in)
		if i < 2 {
			e.paidOrder(t, invitee, 30, payment.TypeWxpay, in)
			e.paidOrder(t, invitee, 10, payment.TypeWxpay, in) // same invitee counted once
		}
	}
	midInvitee, midBalanceOnly := e.user(t, "mid-inv"), e.user(t, "mid-bal")
	e.invite(t, mid, midInvitee, in)
	e.invite(t, mid, midBalanceOnly, in)
	e.paidOrder(t, midInvitee, 50, payment.TypeWxpay, in)
	e.paidOrder(t, midBalanceOnly, 50, payment.TypeBalance, in) // balance-paid does not make a paying invitee
	e.paidOrder(t, midInvitee, 999, payment.TypeWxpay, end.Add(time.Hour))

	rows, err := e.repo.InviteLeaderboard(e.ctx, &start, &end, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, top.ID, rows[0].UserID)
	require.Equal(t, 1, rows[0].Rank)
	require.Equal(t, 2, rows[0].PayingInvitees)
	require.Equal(t, 3, rows[0].InvitedCount)
	require.InDelta(t, 80, rows[0].InviteePaid, 1e-9)
	require.Equal(t, mid.ID, rows[1].UserID)
	require.Equal(t, 1, rows[1].PayingInvitees)
	require.Equal(t, 2, rows[1].InvitedCount)
	require.InDelta(t, 50, rows[1].InviteePaid, 1e-9, "order outside the window is excluded")

	me, err := e.repo.InviteLeaderboardEntryFor(e.ctx, mid.ID, &start, &end)
	require.NoError(t, err)
	require.Equal(t, 2, me.Rank)
	none, err := e.repo.InviteLeaderboardEntryFor(e.ctx, quiet.ID, &start, &end)
	require.NoError(t, err)
	require.Nil(t, none)
}
