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

func TestAnalyticsRepositoryOverview(t *testing.T) {
	ctx := context.Background()
	n := time.Now().UnixNano()
	sfx := fmt.Sprintf("%d", n)
	src := "poster-" + sfx[len(sfx)-8:] // a channel only this test uses
	repo := NewAnalyticsRepository(integrationDB)
	svc := service.NewAnalyticsService(repo)
	mk := func(tag string) *service.User {
		return mustCreateUser(t, integrationEntClient, &service.User{Email: "an-" + tag + "-" + sfx + "@test.local", Username: "an" + tag + sfx[len(sfx)-6:]})
	}
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile"
	attr := service.AnalyticsAttribution{Source: src, Medium: "edu", Landing: "/home"}

	// Two visitors arrive from the poster; one more comes directly.
	visitA, visitB, visitC := "va"+sfx, "vb"+sfx, "vc"+sfx
	for _, v := range []string{visitA, visitB} {
		_, err := svc.Ingest(ctx, service.AnalyticsBatch{VisitorID: v, App: "main", Attr: attr, Events: []service.AnalyticsBatchEvent{
			{Name: "page_view", Path: "/home-" + sfx + "?token=secret"},
			{Name: "signup_view", Path: "/register"},
		}}, service.AnalyticsRequestMeta{IP: "203.0.113.77", UserAgent: ua})
		require.NoError(t, err)
	}
	_, err := svc.Ingest(ctx, service.AnalyticsBatch{VisitorID: visitC, App: "learn", Events: []service.AnalyticsBatchEvent{
		{Name: "page_view", Path: "/learn/a/a1"}, {Name: "learn_run", Props: map[string]any{"lesson": "a1"}},
	}}, service.AnalyticsRequestMeta{IP: "2001:db8:1234:5678::1", UserAgent: "Mozilla/5.0 (Macintosh)"})
	require.NoError(t, err)

	// A signs up, creates a key and calls the API within the hour, then pays; B signs up and stops.
	a, b := mk("a"), mk("b")
	svc.RecordSignup(ctx, a.ID, visitA, attr)
	svc.RecordSignup(ctx, b.ID, visitB, attr)
	svc.RecordSignup(ctx, b.ID, visitB, service.AnalyticsAttribution{Source: "other"}) // first one wins
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: a.ID, Key: "sk-an-" + sfx})
	acc := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "an-acc-" + sfx})
	_, err = integrationEntClient.UsageLog.Create().SetUserID(a.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).
		SetRequestID("an-" + sfx).SetModel("gpt-5").SetCreatedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = integrationEntClient.PaymentOrder.Create().
		SetUserID(a.ID).SetUserEmail(a.Email).SetUserName(a.Username).
		SetAmount(120).SetPayAmount(120).SetFeeRate(0).
		SetRechargeCode("AN-" + sfx).SetOutTradeNo("an_" + sfx).
		SetPaymentType("wxpay").SetPaymentTradeNo("t").
		SetOrderType(payment.OrderTypeBalance).SetStatus(service.OrderStatusCompleted).
		SetPaidAt(time.Now()).SetExpiresAt(time.Now()).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)
	// After sign-up A is active on the site.
	_, err = svc.Ingest(ctx, service.AnalyticsBatch{VisitorID: visitA, App: "main", Attr: attr, Events: []service.AnalyticsBatchEvent{
		{Name: "key_created", Path: "/keys"},
	}}, service.AnalyticsRequestMeta{UserID: a.ID, IP: "203.0.113.77", UserAgent: ua})
	require.NoError(t, err)

	// Stored rows: query stripped, IP truncated, anonymous events tied to the new account.
	var path, prefix, device string
	var uid *int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT path, ip_prefix, device, user_id FROM analytics_events WHERE visitor_id = $1 AND event = 'page_view'`, visitA).
		Scan(&path, &prefix, &device, &uid))
	require.Equal(t, "/home-"+sfx, path)
	require.Equal(t, "203.0.113.0", prefix)
	require.Equal(t, "mobile", device)
	require.NotNil(t, uid)
	require.Equal(t, a.ID, *uid)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT ip_prefix FROM analytics_events WHERE visitor_id = $1 LIMIT 1`, visitC).Scan(&prefix))
	require.Equal(t, "2001:db8:1234::", prefix)
	var gotSrc string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT source FROM user_attributions WHERE user_id = $1`, b.ID).Scan(&gotSrc))
	require.Equal(t, src, gotSrc)

	ov, err := svc.Overview(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 7, ov.Days)
	require.Len(t, ov.Daily, 7)
	today := ov.Daily[6]
	require.GreaterOrEqual(t, today.Visitors, int64(3))
	require.GreaterOrEqual(t, today.Signups, int64(2))
	require.GreaterOrEqual(t, today.Activated, int64(1))
	require.GreaterOrEqual(t, today.Revenue, 120.0)
	require.GreaterOrEqual(t, today.ActiveUsers, int64(1))

	var ch *service.AnalyticsChannel
	for i := range ov.Channels {
		if ov.Channels[i].Source == src {
			ch = &ov.Channels[i]
		}
	}
	require.NotNil(t, ch, "poster channel present")
	require.Equal(t, service.AnalyticsChannel{Source: src, Visitors: 2, Signups: 2, Activated: 1, PaidUsers: 1, Revenue: 120}, *ch)

	found := false
	for _, p := range ov.Pages {
		if p.Path == "/home-"+sfx {
			found = true
			require.Equal(t, int64(2), p.Views)
			require.Equal(t, int64(2), p.Visitors)
		}
	}
	require.True(t, found)
	features := map[string]service.AnalyticsFeature{}
	for _, f := range ov.Features {
		features[f.Event] = f
	}
	require.GreaterOrEqual(t, features["learn_run"].Count, int64(1))
	require.GreaterOrEqual(t, features["signup_view"].Visitors, int64(2))
	require.Len(t, ov.Funnel, 6)
	require.GreaterOrEqual(t, ov.Funnel[2].Count, int64(2))
	require.NotEmpty(t, ov.Retention)
	require.GreaterOrEqual(t, ov.Devices["mobile"], int64(2))
	require.GreaterOrEqual(t, ov.Totals.PaidUsers, int64(1))

	// Old events are cleaned up.
	_, err = integrationDB.ExecContext(ctx, `UPDATE analytics_events SET created_at = NOW() - INTERVAL '100 days' WHERE visitor_id = $1`, visitC)
	require.NoError(t, err)
	deleted, err := repo.DeleteEventsBefore(ctx, time.Now().Add(-90*24*time.Hour))
	require.NoError(t, err)
	require.GreaterOrEqual(t, deleted, int64(2))
}

func TestActivationReminderRepositoryDueUsers(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewActivationReminderRepository(integrationDB)
	mk := func(tag string, age time.Duration) *service.User {
		u := mustCreateUser(t, integrationEntClient, &service.User{Email: "ar-" + tag + "-" + sfx + "@test.local", Username: "ar" + tag + sfx[len(sfx)-5:]})
		_, err := integrationDB.ExecContext(ctx, `UPDATE users SET created_at = NOW() - $1::interval WHERE id = $2`, fmt.Sprintf("%d seconds", int(age.Seconds())), u.ID)
		require.NoError(t, err)
		return u
	}
	due := mk("due", 30*time.Hour)
	fresh := mk("fresh", 2*time.Hour)
	old := mk("old", 100*time.Hour)
	caller := mk("caller", 30*time.Hour)
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: caller.ID, Key: "sk-ar-" + sfx})
	acc := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "ar-acc-" + sfx})
	_, err := integrationEntClient.UsageLog.Create().SetUserID(caller.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).
		SetRequestID("ar-" + sfx).SetModel("gpt-5").SetCreatedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	ids := func() map[int64]bool {
		users, err := repo.DueUsers(ctx, time.Now().Add(-72*time.Hour), time.Now().Add(-24*time.Hour), 500)
		require.NoError(t, err)
		out := map[int64]bool{}
		for _, u := range users {
			out[u.ID] = true
		}
		return out
	}
	got := ids()
	require.True(t, got[due.ID])
	require.False(t, got[fresh.ID], "too new")
	require.False(t, got[old.ID], "too old")
	require.False(t, got[caller.ID], "already made a call")

	before, err := repo.SentCount(ctx)
	require.NoError(t, err)
	require.NoError(t, repo.MarkSent(ctx, due.ID))
	require.NoError(t, repo.MarkSent(ctx, due.ID))
	after, err := repo.SentCount(ctx)
	require.NoError(t, err)
	require.Equal(t, before+1, after)
	require.False(t, ids()[due.ID], "only once")
}

func TestPublicPricing(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	sub := mustCreateGroup(t, integrationEntClient, &service.Group{Name: "pp-sub-" + sfx, Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeSubscription, IsExclusive: true, RateMultiplier: 1})
	daily := 60.0
	require.NoError(t, integrationEntClient.Group.UpdateOneID(sub.ID).SetDailyLimitUsd(daily).Exec(ctx))
	payg := mustCreateGroup(t, integrationEntClient, &service.Group{Name: "pp-payg-" + sfx, Platform: service.PlatformOpenAI, RateMultiplier: 1.2})
	hidden := mustCreateGroup(t, integrationEntClient, &service.Group{Name: "pp-vip-" + sfx, Platform: service.PlatformOpenAI, IsExclusive: true, RateMultiplier: 0.5})
	orig := 299.0
	_, err := integrationEntClient.SubscriptionPlan.Create().SetGroupID(sub.ID).SetName("月度-" + sfx).SetPrice(120).SetOriginalPrice(orig).
		SetValidityDays(30).SetValidityUnit("days").SetForSale(true).Save(ctx)
	require.NoError(t, err)
	_, err = integrationEntClient.SubscriptionPlan.Create().SetGroupID(sub.ID).SetName("下架-" + sfx).SetPrice(1).
		SetValidityDays(30).SetValidityUnit("days").SetForSale(false).Save(ctx)
	require.NoError(t, err)

	member := mustCreateUser(t, integrationEntClient, &service.User{Email: "pp-" + sfx + "@test.local", Username: "pp" + sfx[len(sfx)-6:]})
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: member.ID, Key: "sk-pp-" + sfx})
	acc := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "pp-acc-" + sfx})
	_, err = integrationEntClient.UsageLog.Create().SetUserID(member.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).SetGroupID(payg.ID).
		SetRequestID("pp-" + sfx).SetModel("gpt-5.5").SetActualCost(0.2).SetCreatedAt(time.Now()).Save(ctx)
	require.NoError(t, err)

	svc := service.NewPaymentConfigService(integrationEntClient, NewSettingRepository(integrationEntClient), nil)
	out, err := svc.PublicPricing(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, out.RechargeMultiplier, 0.0001)

	var plan *service.PublicPricingPlan
	for i := range out.Plans {
		require.NotEqual(t, "下架-"+sfx, out.Plans[i].Name, "only plans on sale")
		if out.Plans[i].Name == "月度-"+sfx {
			plan = &out.Plans[i]
		}
	}
	require.NotNil(t, plan)
	require.Equal(t, 120.0, plan.Price)
	require.Equal(t, sub.Name, plan.GroupName)
	require.NotNil(t, plan.DailyLimitUSD)
	require.Equal(t, daily, *plan.DailyLimitUSD)

	names := map[string]bool{}
	for _, g := range out.PayAsYouGo {
		names[g.Name] = true
	}
	require.True(t, names[payg.Name])
	require.False(t, names[hidden.Name], "exclusive groups are not public")
	require.False(t, names[sub.Name], "subscription groups are not pay-as-you-go")
	require.GreaterOrEqual(t, out.AvgSampleRequests, 1)
	require.Greater(t, out.AvgRequestCostUSD, 0.0)
}

func TestChannelLinkRepository(t *testing.T) {
	ctx := context.Background()
	sfx := fmt.Sprintf("%d", time.Now().UnixNano())
	repo := NewChannelLinkRepository(integrationDB)
	links := service.NewChannelLinkService(repo, nil)
	analytics := service.NewAnalyticsService(NewAnalyticsRepository(integrationDB))

	l, err := links.Create(ctx, service.ChannelLinkInput{Name: "测试渠道", Source: "xiaohongshu", Medium: "post", TargetPath: "/pricing"})
	require.NoError(t, err)
	_, err = links.Create(ctx, service.ChannelLinkInput{Code: l.Code, Name: "dup", Source: "x"})
	require.ErrorIs(t, err, service.ErrChannelLinkCodeTaken)

	// Two clicks from browsers; the visitors land with the link's tags.
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Mobile"
	target := links.Resolve(ctx, l.Code, ua)
	require.Contains(t, target, "utm_campaign="+l.Code)
	links.Resolve(ctx, l.Code, ua)
	attr := service.AnalyticsAttribution{Source: "xiaohongshu", Medium: "post", Campaign: l.Code, Landing: "/pricing"}
	va, vb := "cla"+sfx, "clb"+sfx
	for _, v := range []string{va, vb} {
		_, err := analytics.Ingest(ctx, service.AnalyticsBatch{VisitorID: v, App: "main", Attr: attr,
			Events: []service.AnalyticsBatchEvent{{Name: "page_view", Path: "/pricing"}}}, service.AnalyticsRequestMeta{IP: "203.0.113.9", UserAgent: ua})
		require.NoError(t, err)
	}
	// Both sign up; A calls the API and pays ¥60, B stops there.
	mk := func(tag string) *service.User {
		return mustCreateUser(t, integrationEntClient, &service.User{Email: "cl-" + tag + "-" + sfx + "@test.local", Username: "cl" + tag + sfx[len(sfx)-6:]})
	}
	a, b := mk("a"), mk("b")
	analytics.RecordSignup(ctx, a.ID, va, attr)
	analytics.RecordSignup(ctx, b.ID, vb, attr)
	key := mustCreateApiKey(t, integrationEntClient, &service.APIKey{UserID: a.ID, Key: "sk-cl-" + sfx})
	acc := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "cl-acc-" + sfx})
	_, err = integrationEntClient.UsageLog.Create().SetUserID(a.ID).SetAPIKeyID(key.ID).SetAccountID(acc.ID).
		SetRequestID("cl-" + sfx).SetModel("gpt-5").SetCreatedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	_, err = integrationEntClient.PaymentOrder.Create().
		SetUserID(a.ID).SetUserEmail(a.Email).SetUserName(a.Username).
		SetAmount(60).SetPayAmount(60).SetFeeRate(0).
		SetRechargeCode("CL-" + sfx).SetOutTradeNo("cl_" + sfx).
		SetPaymentType("alipay").SetPaymentTradeNo("t").
		SetOrderType(payment.OrderTypeSubscription).SetStatus(service.OrderStatusCompleted).
		SetPaidAt(time.Now()).SetExpiresAt(time.Now()).SetClientIP("127.0.0.1").SetSrcHost("test").
		Save(ctx)
	require.NoError(t, err)

	all, err := links.List(ctx)
	require.NoError(t, err)
	var got *service.ChannelLink
	for i := range all {
		if all[i].Code == l.Code {
			got = &all[i]
		}
	}
	require.NotNil(t, got)
	require.EqualValues(t, 2, got.Clicks)
	require.EqualValues(t, 2, got.Visitors)
	require.EqualValues(t, 2, got.Signups)
	require.EqualValues(t, 1, got.Activated)
	require.EqualValues(t, 1, got.PaidUsers)
	require.InDelta(t, 60, got.Revenue, 1e-9)

	// Editing keeps the code; deleting removes it.
	upd, err := links.Update(ctx, l.ID, service.ChannelLinkInput{Code: "ignored", Name: "改名", Source: "xiaohongshu", TargetPath: "/register"})
	require.NoError(t, err)
	require.Equal(t, l.Code, upd.Code)
	require.Equal(t, "/register", upd.TargetPath)
	require.NoError(t, links.Delete(ctx, l.ID))
	require.ErrorIs(t, links.Delete(ctx, l.ID), service.ErrChannelLinkNotFound)
	require.Equal(t, "/", links.Resolve(ctx, l.Code, ua))
}
