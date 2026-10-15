//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func hitFor(hits []service.RiskHit, userID int64) (service.RiskHit, bool) {
	for _, h := range hits {
		if h.UserID == userID {
			return h, true
		}
	}
	return service.RiskHit{}, false
}

// TestUserRiskRepository runs every 静默限制 / 风控 / 用户画像 query against Postgres.
func TestUserRiskRepository(t *testing.T) {
	ctx := context.Background()
	rows := trackCommitted(t)
	n := time.Now().UnixNano()
	sfx := fmt.Sprintf("%d", n)
	mk := func(tag string) *service.User {
		return rows.user(mustCreateUser(t, integrationEntClient, &service.User{Email: "risk-" + tag + "-" + sfx + "@test.local", Username: "rk" + tag + sfx[len(sfx)-6:]}))
	}
	banned, fresh, trial, relay, payer, inviter, invitee := mk("ban"), mk("new"), mk("trial"), mk("relay"), mk("payer"), mk("inviter"), mk("invitee")
	autoA, autoB, labA, labB := mk("autoa"), mk("autob"), mk("laba"), mk("labb")
	ids := []int64{banned.ID, fresh.ID, trial.ID, relay.ID, payer.ID, inviter.ID, invitee.ID, autoA.ID, autoB.ID, labA.ID, labB.ID}
	t.Cleanup(func() {
		for _, q := range []string{`DELETE FROM signup_fingerprints WHERE user_id = ANY($1)`, `DELETE FROM risk_events WHERE user_id = ANY($1)`} {
			_, err := integrationDB.ExecContext(context.Background(), q, pq.Array(ids))
			require.NoError(t, err)
		}
	})
	require.NoError(t, integrationEntClient.User.UpdateOneID(banned.ID).SetStatus(service.StatusDisabled).Exec(ctx))

	fpRepo := NewSignupFingerprintRepository(integrationDB)
	visitor := "rk-visitor-" + sfx
	prefixA := fmt.Sprintf("198.18.%d.0", n%200)
	prefixB := fmt.Sprintf("198.19.%d.0", n%200)
	prefixC := fmt.Sprintf("198.20.%d.0", n%200) // a school lab
	prefixD := fmt.Sprintf("198.21.%d.0", n%200) // network of an automatically restricted account
	for _, fp := range []service.SignupFingerprint{
		{UserID: banned.ID, VisitorID: visitor, IPPrefix: prefixA, TrialGranted: true},
		{UserID: fresh.ID, VisitorID: visitor, TrialGranted: true},
		{UserID: trial.ID, TrialGranted: true},
		{UserID: relay.ID, TrialGranted: true},
		{UserID: payer.ID},
		{UserID: inviter.ID, IPPrefix: prefixB},
		{UserID: invitee.ID, IPPrefix: prefixB, TrialGranted: true},
		{UserID: autoA.ID, VisitorID: "rk-auto-" + sfx, IPPrefix: prefixD, TrialGranted: true},
		{UserID: labA.ID, VisitorID: "rk-lab-a-" + sfx, IPPrefix: prefixC, TrialGranted: true},
	} {
		require.NoError(t, fpRepo.Save(ctx, fp))
	}
	visitorSince, ipSince := time.Now().AddDate(0, 0, -90), time.Now().AddDate(0, 0, -30)
	m, err := fpRepo.FindFlaggedMatch(ctx, visitor, "", fresh.ID, visitorSince, ipSince)
	require.NoError(t, err)
	require.Equal(t, service.FingerprintMatch{UserID: banned.ID, ByVisitor: true, Status: service.StatusDisabled}, m)
	m, err = fpRepo.FindFlaggedMatch(ctx, "", prefixA, fresh.ID, visitorSince, ipSince)
	require.NoError(t, err)
	require.Equal(t, service.FingerprintMatch{UserID: banned.ID, Status: service.StatusDisabled}, m, "same /24 as a banned account")
	m, err = fpRepo.FindFlaggedMatch(ctx, visitor, prefixA, fresh.ID, time.Now().Add(time.Hour), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, m.UserID, "outside both windows")
	m, err = fpRepo.FindFlaggedMatch(ctx, "", prefixB, invitee.ID, visitorSince, ipSince)
	require.NoError(t, err)
	require.Zero(t, m.UserID, "the inviter is neither banned nor restricted")

	// 静默限制 storage and the free-credit gate's lookups.
	shadowRepo := NewUserShadowRepository(integrationDB)
	related := banned.ID
	require.NoError(t, shadowRepo.UpsertShadow(ctx, service.UserShadow{UserID: inviter.ID, Reason: "first", Source: service.ShadowSourceAuto, RelatedUserID: &related}))
	require.NoError(t, shadowRepo.UpsertShadow(ctx, service.UserShadow{UserID: inviter.ID, Reason: "farm", Source: service.ShadowSourceAdmin}))
	all, err := shadowRepo.ListShadows(ctx)
	require.NoError(t, err)
	var got *service.UserShadow
	for i := range all {
		if all[i].UserID == inviter.ID {
			got = &all[i]
		}
	}
	require.NotNil(t, got)
	require.Equal(t, "farm", got.Reason)
	require.Equal(t, service.ShadowSourceAdmin, got.Source)
	m, err = fpRepo.FindFlaggedMatch(ctx, "", prefixB, invitee.ID, visitorSince, ipSince)
	require.NoError(t, err)
	require.Equal(t, "shadowed", m.Status, "an admin-made restriction is a source")

	// An automatic restriction is never a source: nothing chains.
	require.NoError(t, shadowRepo.UpsertShadow(ctx, service.UserShadow{UserID: autoA.ID, Reason: "auto", Source: service.ShadowSourceAuto}))
	require.NoError(t, fpRepo.Save(ctx, service.SignupFingerprint{UserID: autoB.ID, VisitorID: "rk-auto-b-" + sfx, IPPrefix: prefixD, TrialGranted: true}))
	m, err = fpRepo.FindFlaggedMatch(ctx, "rk-auto-"+sfx, prefixD, autoB.ID, visitorSince, ipSince)
	require.NoError(t, err)
	require.Zero(t, m.UserID)

	// Another student in the lab, other browser: reported by the monitor, not a restriction source.
	require.NoError(t, fpRepo.Save(ctx, service.SignupFingerprint{UserID: labB.ID, VisitorID: "rk-lab-b-" + sfx, IPPrefix: prefixC, TrialGranted: true}))

	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: relay.ID, Key: "sk-rk-" + sfx})
	acc := rows.account(mustCreateAccount(t, integrationEntClient, &service.Account{Name: "rk-acc-" + sfx}))
	limit := 35.0
	grp := rows.group(mustCreateGroup(t, integrationEntClient, &service.Group{Name: "rk-trial-" + sfx, Platform: service.PlatformOpenAI,
		SubscriptionType: service.SubscriptionTypeSubscription, DailyLimitUSD: &limit}))
	_, err = integrationEntClient.UsageLog.Create().SetUserID(relay.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).SetGroupID(grp.ID).
		SetRequestID("rk-" + sfx).SetModel("gpt-5").SetUserAgent("axonhub/1.0").SetIPAddress("203.0.113.9").
		SetTotalCost(0.4).SetActualCost(0.4).SetCreatedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	spent, err := shadowRepo.SpentSince(ctx, relay.ID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.InDelta(t, 0.4, spent, 1e-9)

	_, err = integrationEntClient.PaymentOrder.Create().
		SetUserID(payer.ID).SetUserEmail(payer.Email).SetUserName(payer.Username).
		SetAmount(10).SetPayAmount(72).SetFeeRate(0).
		SetRechargeCode("RK-" + sfx).SetOutTradeNo("rk_" + sfx).
		SetPaymentType("alipay").SetPaymentTradeNo("t").
		SetOrderType(payment.OrderTypeBalance).SetStatus(service.OrderStatusCompleted).
		SetPaidAt(time.Now()).SetExpiresAt(time.Now()).SetClientIP("203.0.113.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	paid, err := shadowRepo.HasRealPayment(ctx, payer.ID)
	require.NoError(t, err)
	require.True(t, paid)
	paid, err = shadowRepo.HasRealPayment(ctx, relay.ID)
	require.NoError(t, err)
	require.False(t, paid)
	// Refunded since, but the money did arrive: still a paying customer.
	_, err = integrationEntClient.PaymentOrder.Create().
		SetUserID(labA.ID).SetUserEmail(labA.Email).SetUserName(labA.Username).
		SetAmount(10).SetPayAmount(72).SetFeeRate(0).
		SetRechargeCode("RK2-" + sfx).SetOutTradeNo("rk2_" + sfx).
		SetPaymentType("wxpay").SetPaymentTradeNo("t2").
		SetOrderType(payment.OrderTypeBalance).SetStatus("REFUNDED").
		SetPaidAt(time.Now()).SetExpiresAt(time.Now()).SetClientIP("203.0.113.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	paid, err = shadowRepo.HasRealPayment(ctx, labA.ID)
	require.NoError(t, err)
	require.True(t, paid)

	now := time.Now()
	sub := mustCreateSubscription(t, integrationEntClient, &service.UserSubscription{UserID: trial.ID, GroupID: grp.ID,
		StartsAt: now, ExpiresAt: now.Add(24 * time.Hour), Status: service.SubscriptionStatusActive, AssignedAt: now,
		Notes: "auto assigned by signup defaults", DailyUsageUSD: 1})
	require.NoError(t, shadowRepo.FillSubscriptionDailyUsage(ctx, sub.ID, limit))
	require.InDelta(t, limit, querySingleFloat(t, ctx, integrationEntClient, `SELECT daily_usage_usd FROM user_subscriptions WHERE id = $1`, sub.ID), 1e-9)

	affRepo := NewAffiliateRepository(integrationEntClient, integrationDB)
	for _, id := range []int64{inviter.ID, invitee.ID} {
		_, err = affRepo.EnsureUserAffiliate(ctx, id)
		require.NoError(t, err)
	}
	bound, err := affRepo.BindInviter(ctx, invitee.ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, bound)

	// Monitor rules.
	repo := NewUserRiskRepository(integrationDB)
	since := time.Now().Add(-time.Hour)
	hits, err := repo.LinkedFlaggedSignups(ctx, since)
	require.NoError(t, err)
	h, ok := hitFor(hits, fresh.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, fmt.Sprintf("#%d（已封禁）同浏览器", banned.ID))
	h, ok = hitFor(hits, invitee.ID)
	require.True(t, ok, "shares a /24 with an admin-restricted inviter")
	require.Contains(t, h.Detail, "（静默）同 IP 段")
	_, ok = hitFor(hits, autoB.ID)
	require.False(t, ok, "an automatic restriction is not a source")

	hits, err = repo.TrialSameIPSignups(ctx, since, 30*24*time.Hour)
	require.NoError(t, err)
	h, ok = hitFor(hits, labB.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, fmt.Sprintf("#%d", labA.ID))
	_, ok = hitFor(hits, labA.ID)
	require.False(t, ok, "the first one in the lab is fine")

	hits, err = repo.TrialDay1Exhausted(ctx, since)
	require.NoError(t, err)
	h, ok = hitFor(hits, trial.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, "$35.00 / $35.00")

	hits, err = repo.RelayFreeSpend(ctx, since, `axonhub|go-http-client`)
	require.NoError(t, err)
	h, ok = hitFor(hits, relay.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, "axonhub/1.0")

	hits, err = repo.BurstSignups(ctx, since, 2)
	require.NoError(t, err)
	_, ok = hitFor(hits, inviter.ID)
	require.True(t, ok)
	h, ok = hitFor(hits, fresh.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, "同浏览器")

	hits, err = repo.InviteeSharesInviterFingerprint(ctx, since)
	require.NoError(t, err)
	h, ok = hitFor(hits, invitee.ID)
	require.True(t, ok)
	require.Contains(t, h.Detail, fmt.Sprintf("#%d 同 IP 段", inviter.ID))

	events, err := repo.InsertRiskEvents(ctx, service.RiskRuleRelayFree, []service.RiskHit{{UserID: relay.ID, Detail: "x"}})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, relay.Email, events[0].UserEmail)
	events, err = repo.InsertRiskEvents(ctx, service.RiskRuleRelayFree, []service.RiskHit{{UserID: relay.ID, Detail: "again"}})
	require.NoError(t, err)
	require.Empty(t, events, "once per user and rule")
	listed, total, err := repo.ListRiskEvents(ctx, relay.ID, 1, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "x", listed[0].Detail)

	// Dashboard queries.
	segments, days, err := repo.InsightsSegments(ctx, since)
	require.NoError(t, err)
	require.Positive(t, segments.Inactive+segments.Trial+segments.Paying+segments.Churned)
	require.NotEmpty(t, days)
	spend, err := repo.InsightsSpend(ctx, since)
	require.NoError(t, err)
	require.NotEmpty(t, spend)
	uaRows, err := repo.InsightsUserAgents(ctx, since)
	require.NoError(t, err)
	found := false
	for _, r := range uaRows {
		found = found || (r.UserID == relay.ID && r.UserAgent == "axonhub/1.0")
	}
	require.True(t, found)
	clusters, err := repo.InsightsClusters(ctx, since)
	require.NoError(t, err)
	found = false
	for _, c := range clusters {
		found = found || (c.Kind == "visitor" && c.Key == visitor && len(c.UserIDs) == 2)
	}
	require.True(t, found)
	refs, err := repo.UserRefs(ctx, []int64{payer.ID, relay.ID}, since)
	require.NoError(t, err)
	require.InDelta(t, 72, refs[payer.ID].PaidCNY, 1e-9)
	require.InDelta(t, 0.4, refs[relay.ID].SpendUSD, 1e-9)

	id, err := repo.FindUserID(ctx, fresh.Email)
	require.NoError(t, err)
	require.Equal(t, fresh.ID, id)
	id, err = repo.FindUserID(ctx, fmt.Sprintf("#%d", fresh.ID))
	require.NoError(t, err)
	require.Equal(t, fresh.ID, id)
	id, err = repo.FindUserID(ctx, "nobody-"+sfx+"@test.local")
	require.NoError(t, err)
	require.Zero(t, id)

	p, err := repo.UserProfile(ctx, fresh.ID, time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Equal(t, visitor, p.VisitorID)
	require.Len(t, p.Linked, 1)
	require.Equal(t, banned.ID, p.Linked[0].User.ID)
	require.Equal(t, []string{"visitor"}, p.Linked[0].Via)

	p, err = repo.UserProfile(ctx, relay.ID, time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Len(t, p.Clients, 1)
	require.Equal(t, "axonhub/1.0", p.Clients[0].UserAgent)
	require.InDelta(t, 0.4, p.Spend.BalanceUSD, 1e-9)
	require.False(t, p.Spend.HasRealPaying)
	require.Len(t, p.APIIPs, 1)
	require.Len(t, p.Groups, 1)

	p, err = repo.UserProfile(ctx, invitee.ID, time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Len(t, p.InviteUp, 1)
	require.Equal(t, inviter.ID, p.InviteUp[0].ID)
	p, err = repo.UserProfile(ctx, inviter.ID, time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Len(t, p.InviteDown, 1)

	p, err = repo.UserProfile(ctx, trial.ID, time.Now().AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Len(t, p.Subscriptions, 1)
	require.True(t, p.Subscriptions[0].Free)

	// A late payment is noted on one event per user, newest first.
	require.NoError(t, shadowRepo.NoteRiskEvent(ctx, payer.ID, service.RiskRuleShadowPaid, "order #1"))
	require.NoError(t, shadowRepo.NoteRiskEvent(ctx, payer.ID, service.RiskRuleShadowPaid, "order #2"))
	listed, total, err = repo.ListRiskEvents(ctx, payer.ID, 1, 10)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "order #2；order #1", listed[0].Detail)

	require.NoError(t, shadowRepo.DeleteShadow(ctx, inviter.ID))
}
