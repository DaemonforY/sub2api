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
