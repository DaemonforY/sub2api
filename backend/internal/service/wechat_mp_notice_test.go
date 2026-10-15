//go:build unit

package service

import (
	"context"
	"database/sql"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestTruncateWeChatNoticeValue(t *testing.T) {
	long := strings.Repeat("长", 25)
	require.Equal(t, strings.Repeat("长", 20), truncateWeChatNoticeValue("thing1", long))
	require.Equal(t, strings.Repeat("长", 20), truncateWeChatNoticeValue("thing12", long))
	require.Equal(t, strings.Repeat("a", 32), truncateWeChatNoticeValue("character_string3", strings.Repeat("a", 40)))
	require.Equal(t, "余额不足提", truncateWeChatNoticeValue("phrase2", "余额不足提醒"))
	for _, key := range []string{"amount4", "time5", "number6", "date7"} {
		require.Equal(t, long, truncateWeChatNoticeValue(key, long), key)
	}
	require.Equal(t, "短", truncateWeChatNoticeValue("thing1", "短"))
}

func TestFillWeChatNoticeValue(t *testing.T) {
	vars := map[string]string{"username": "小明", "balance": "$1.23", "site_name": "HiveGPT"}
	require.Equal(t, "小明，你在 HiveGPT 的余额只剩 $1.23", fillWeChatNoticeValue("{username}，你在 {site_name} 的余额只剩 {balance}", vars))
	require.Equal(t, "{unknown} 不动", fillWeChatNoticeValue("{unknown} 不动", vars))
	require.Equal(t, "无占位符", fillWeChatNoticeValue("无占位符", vars))
}

func TestNormalizeWeChatTemplateNotices(t *testing.T) {
	ok := map[string]WeChatTemplateNoticeConfig{
		WeChatNoticeBalanceLow: {Enabled: true, TemplateID: "AbC-def_1234567890", URLPath: "/purchase", Fields: map[string]string{"thing1": "{username}", "amount2": "{balance}"}},
	}
	out, err := NormalizeWeChatTemplateNotices(ok)
	require.NoError(t, err)
	require.Len(t, out, 4, "every event is present")
	require.False(t, out[WeChatNoticeTrialExhausted].Enabled)
	require.Equal(t, "/purchase", out[WeChatNoticeBalanceLow].URLPath)

	for name, bad := range map[string]map[string]WeChatTemplateNoticeConfig{
		"unknown event":  {"winback": {}},
		"template id":    {WeChatNoticeBalanceLow: {TemplateID: "bad id!"}},
		"external url":   {WeChatNoticeBalanceLow: {URLPath: "https://evil.example/x"}},
		"protocol rel":   {WeChatNoticeBalanceLow: {URLPath: "//evil.example"}},
		"field key":      {WeChatNoticeBalanceLow: {Fields: map[string]string{"Thing1": "x"}}},
		"no digits":      {WeChatNoticeBalanceLow: {Fields: map[string]string{"thing": "x"}}},
		"placeholder":    {WeChatNoticeBalanceLow: {Fields: map[string]string{"thing1": "{plan_name}"}}},
		"enabled, empty": {WeChatNoticeOrderAbandoned: {Enabled: true}},
	} {
		_, err := NormalizeWeChatTemplateNotices(bad)
		require.Error(t, err, name)
		require.Equal(t, 400, infraerrors.Code(err), name)
	}
}

// ---------- routing ----------

type wechatNoticeEnv struct {
	notifications *NotificationEmailService
	notices       *WeChatTemplateNoticeService
	api           *fakeWeChatAPI
	repo          *wechatMPTestRepo
	client        *dbent.Client
	smtpDials     *atomic.Int32
}

func newWeChatNoticeEnv(t *testing.T, notices string) *wechatNoticeEnv {
	t.Helper()
	db, err := sql.Open("sqlite", "file:wechat_mp_notice_"+strconv.FormatInt(time.Now().UnixNano(), 10)+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = client.Close() })

	// A fake SMTP port that only counts connections.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	dials := &atomic.Int32{}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			dials.Add(1)
			_ = conn.Close()
		}
	}()

	svc, _, api, repo := newWeChatMPTestService(t, map[string]string{
		SettingKeyWeChatMPTemplateNotices: notices,
		SettingKeySMTPHost:                "127.0.0.1",
		SettingKeySMTPPort:                strconv.Itoa(ln.Addr().(*net.TCPAddr).Port),
		SettingKeySMTPFrom:                "noreply@hivegpt.cn",
		SettingKeySiteName:                "HiveGPT",
	})
	settingSvc := NewSettingService(repo, &config.Config{})
	noticeSvc := NewWeChatTemplateNoticeService(settingSvc, svc, client)
	noticeSvc.now = func() time.Time { return time.Unix(wechatMPTestNow, 0) }
	notifications := NewNotificationEmailService(repo, NewEmailService(repo, nil))
	notifications.SetWeChatNotices(noticeSvc)
	return &wechatNoticeEnv{notifications: notifications, notices: noticeSvc, api: api, repo: repo, client: client, smtpDials: dials}
}

const wechatNoticeTestSettings = `{"balance_low":{"enabled":true,"template_id":"TPL_balance_low_123","url_path":"/purchase",
 "fields":{"thing1":"{username}，你在{site_name}的余额快用完了，请尽快充值","amount2":"{balance}","amount3":"{threshold}","time4":"{now}"}},
 "subscription_expiring":{"enabled":true,"template_id":"TPL_sub_expiring_1","fields":{"thing1":"{plan_name}","time2":"{expires_at}","number3":"{days_left}"}}}`

func (e *wechatNoticeEnv) user(t *testing.T, email, username string, openid string) int64 {
	t.Helper()
	ctx := context.Background()
	u, err := e.client.User.Create().SetEmail(email).SetUsername(username).SetPasswordHash("x").
		SetRole(RoleUser).SetStatus(StatusActive).SetBalance(0.42).Save(ctx)
	require.NoError(t, err)
	if openid != "" {
		identity, err := e.client.AuthIdentity.Create().SetUserID(u.ID).SetProviderType("wechat").
			SetProviderKey("wechat-main").SetProviderSubject(openid).SetMetadata(map[string]any{}).Save(ctx)
		require.NoError(t, err)
		_, err = e.client.AuthIdentityChannel.Create().SetIdentityID(identity.ID).SetProviderType("wechat").
			SetProviderKey("wechat-main").SetChannel("mp").SetChannelAppID(wechatMPTestAppID).
			SetChannelSubject(openid).SetMetadata(map[string]any{}).Save(ctx)
		require.NoError(t, err)
	}
	return u.ID
}

func (e *wechatNoticeEnv) balanceLow(userID int64, email string) NotificationEmailSendInput {
	return NotificationEmailSendInput{
		Event: NotificationEmailEventBalanceLow, RecipientEmail: email, RecipientName: "x", UserID: userID,
		SourceType: "balance_low", SourceID: strconv.FormatInt(userID, 10), ReminderKey: "2026-10-16",
		Variables:       map[string]string{"current_balance": "0.42", "threshold": "5.00", "recharge_url": "https://hivegpt.cn/purchase"},
		NoticeVariables: map[string]string{"balance": "$0.42", "threshold": "$5.00"},
	}
}

func (e *wechatNoticeEnv) sends() []map[string]any {
	e.api.mu.Lock()
	defer e.api.mu.Unlock()
	return append([]map[string]any(nil), e.api.templateSends...)
}

func TestWeChatNoticeRouting(t *testing.T) {
	env := newWeChatNoticeEnv(t, wechatNoticeTestSettings)
	ctx := context.Background()
	require.True(t, env.notifications.WeChatNoticeEnabled(ctx, NotificationEmailEventBalanceLow))
	require.False(t, env.notifications.WeChatNoticeEnabled(ctx, NotificationEmailEventGrowthTrialExhausted))
	require.False(t, env.notifications.WeChatNoticeEnabled(ctx, NotificationEmailEventGrowthWinback))

	// WeChat-only user with a 公众号 openid: a template message, no SMTP.
	wechatUser := env.user(t, "wechat-openid-a@wechat-connect.invalid", "小蜜蜂", "openid-a")
	input := env.balanceLow(wechatUser, "wechat-openid-a@wechat-connect.invalid")
	require.NoError(t, env.notifications.Send(ctx, input))
	sends := env.sends()
	require.Len(t, sends, 1)
	require.Equal(t, "openid-a", sends[0]["touser"])
	require.Equal(t, "TPL_balance_low_123", sends[0]["template_id"])
	require.Equal(t, "https://hivegpt.cn/purchase", sends[0]["url"])
	data := sends[0]["data"].(map[string]any)
	require.Equal(t, "小蜜蜂，你在HiveGPT的余额快用完了", data["thing1"].(map[string]any)["value"], "thing ≤ 20 runes")
	require.Equal(t, "$0.42", data["amount2"].(map[string]any)["value"])
	require.Equal(t, "$5.00", data["amount3"].(map[string]any)["value"])
	require.Equal(t, time.Unix(wechatMPTestNow, 0).Format("2006-01-02 15:04"), data["time4"].(map[string]any)["value"])
	require.Zero(t, env.smtpDials.Load())

	// Same idempotency key as email: sent once.
	require.NoError(t, env.notifications.Send(ctx, input))
	require.Len(t, env.sends(), 1)

	// A real email: email as before (the SMTP port is dialled), no template.
	realUser := env.user(t, "real@example.com", "real", "openid-real")
	err := env.notifications.Send(ctx, env.balanceLow(realUser, "real@example.com"))
	require.Error(t, err, "the fake SMTP hangs up, which proves the email path ran")
	require.Eventually(t, func() bool { return env.smtpDials.Load() == 1 }, time.Second, 10*time.Millisecond)
	require.Len(t, env.sends(), 1)

	// WeChat-only user without an openid: nothing at all.
	lonely := env.user(t, "wechat-lonely@wechat-connect.invalid", "lonely", "")
	require.NoError(t, env.notifications.Send(ctx, env.balanceLow(lonely, "wechat-lonely@wechat-connect.invalid")))
	require.Len(t, env.sends(), 1)
	time.Sleep(20 * time.Millisecond)
	require.EqualValues(t, 1, env.smtpDials.Load())

	// Winback is never a template message.
	err = env.notifications.Send(ctx, NotificationEmailSendInput{
		Event: NotificationEmailEventGrowthWinback, RecipientEmail: "wechat-openid-a@wechat-connect.invalid", UserID: wechatUser,
		SourceType: "winback", SourceID: strconv.FormatInt(wechatUser, 10), ReminderKey: "once",
	})
	require.NoError(t, err)
	require.Len(t, env.sends(), 1)
}

func TestWeChatNoticeNotFollowingCountsAsDelivered(t *testing.T) {
	env := newWeChatNoticeEnv(t, wechatNoticeTestSettings)
	ctx := context.Background()
	uid := env.user(t, "wechat-gone@wechat-connect.invalid", "gone", "openid-gone")
	env.api.mu.Lock()
	env.api.templateErr = 43004
	env.api.mu.Unlock()

	input := env.balanceLow(uid, "wechat-gone@wechat-connect.invalid")
	require.NoError(t, env.notifications.Send(ctx, input))
	require.NoError(t, env.notifications.Send(ctx, input))
	require.Len(t, env.sends(), 1, "marked delivered: not tried again")
}

func TestWeChatNoticeBadTemplateWarnsOncePerHourAndRetries(t *testing.T) {
	env := newWeChatNoticeEnv(t, wechatNoticeTestSettings)
	ctx := context.Background()
	uid := env.user(t, "wechat-b@wechat-connect.invalid", "b", "openid-b")
	env.api.mu.Lock()
	env.api.templateErr = 40037
	env.api.mu.Unlock()
	now := time.Unix(wechatMPTestNow, 0)
	env.notices.now = func() time.Time { return now }

	input := env.balanceLow(uid, "wechat-b@wechat-connect.invalid")
	require.NoError(t, env.notifications.Send(ctx, input))
	require.NoError(t, env.notifications.Send(ctx, input))
	require.Len(t, env.sends(), 2, "not marked delivered: the next run tries again")

	require.False(t, env.notices.warnRateLimited(WeChatNoticeBalanceLow, "again"), "warned already within the hour")
	now = now.Add(61 * time.Minute)
	require.True(t, env.notices.warnRateLimited(WeChatNoticeBalanceLow, "an hour later"))

	// Fixed template: goes out and is recorded.
	env.api.mu.Lock()
	env.api.templateErr = 0
	env.api.mu.Unlock()
	require.NoError(t, env.notifications.Send(ctx, input))
	require.NoError(t, env.notifications.Send(ctx, input))
	require.Len(t, env.sends(), 3)

	// Any other WeChat error is a delivery failure, like an email failure.
	uid2 := env.user(t, "wechat-c@wechat-connect.invalid", "c", "openid-c")
	env.api.mu.Lock()
	env.api.templateErr = 45009
	env.api.mu.Unlock()
	err := env.notifications.Send(ctx, env.balanceLow(uid2, "wechat-c@wechat-connect.invalid"))
	require.Error(t, err)
	require.True(t, isNotificationEmailDeliveryError(err))
}

func TestWeChatNoticeSubscriptionExpiringPlaceholders(t *testing.T) {
	env := newWeChatNoticeEnv(t, wechatNoticeTestSettings)
	ctx := context.Background()
	uid := env.user(t, "wechat-s@wechat-connect.invalid", "s", "openid-s")
	require.NoError(t, env.notifications.Send(ctx, NotificationEmailSendInput{
		Event: NotificationEmailEventSubscriptionExpiryReminder, RecipientEmail: "wechat-s@wechat-connect.invalid", UserID: uid,
		SourceType: "user_subscription", SourceID: "9", ReminderKey: "3d",
		Variables:       map[string]string{"subscription_group": "Claude 包月", "expiry_time": "2026-10-19 12:00", "days_remaining": "3", "renew_url": "x"},
		NoticeVariables: map[string]string{"plan_name": "Claude 包月", "expires_at": "2026-10-19 12:00", "days_left": "3"},
	}))
	sends := env.sends()
	require.Len(t, sends, 1)
	_, hasURL := sends[0]["url"]
	require.False(t, hasURL, "no url_path, no link")
	data := sends[0]["data"].(map[string]any)
	require.Equal(t, "Claude 包月", data["thing1"].(map[string]any)["value"])
	require.Equal(t, "2026-10-19 12:00", data["time2"].(map[string]any)["value"])
	require.Equal(t, "3", data["number3"].(map[string]any)["value"])
}

func TestWeChatNoticeSendTestOnlyToCaller(t *testing.T) {
	env := newWeChatNoticeEnv(t, wechatNoticeTestSettings)
	ctx := context.Background()
	admin := env.user(t, "admin@example.com", "admin", "openid-admin")
	_ = env.user(t, "wechat-other@wechat-connect.invalid", "other", "openid-other")
	noWeChat := env.user(t, "plain@example.com", "plain", "")

	res, err := env.notices.SendTest(ctx, WeChatNoticeBalanceLow, admin)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.ErrCode)
	require.Contains(t, res.Message, "发送成功")
	sends := env.sends()
	require.Len(t, sends, 1)
	require.Equal(t, "openid-admin", sends[0]["touser"])
	require.Equal(t, "$1.23", sends[0]["data"].(map[string]any)["amount2"].(map[string]any)["value"], "sample values")

	_, err = env.notices.SendTest(ctx, WeChatNoticeBalanceLow, noWeChat)
	require.ErrorIs(t, err, ErrWeChatNoticeNoOpenID)
	_, err = env.notices.SendTest(ctx, WeChatNoticeTrialExhausted, admin)
	require.ErrorIs(t, err, ErrWeChatNoticeNoTemplate)
	_, err = env.notices.SendTest(ctx, "winback", admin)
	require.ErrorIs(t, err, ErrWeChatNoticeUnknownEvent)

	env.api.mu.Lock()
	env.api.templateErr = 43004
	env.api.mu.Unlock()
	res, err = env.notices.SendTest(ctx, WeChatNoticeBalanceLow, admin)
	require.NoError(t, err)
	require.EqualValues(t, 43004, res.ErrCode)
	require.Contains(t, res.Message, "没关注公众号")
	require.Len(t, env.sends(), 2)
	for _, s := range env.sends() {
		require.Equal(t, "openid-admin", s["touser"])
	}
}
