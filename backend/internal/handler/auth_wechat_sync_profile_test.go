//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/pendingauthsession"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const wechatSyncTestAvatar = "https://thirdwx.qlogo.cn/mmopen/vi_32/abcDEF123/132"

func TestWeChatFallbackUsername(t *testing.T) {
	require.Equal(t, "微信用户_3abc", wechatFallbackUsername("o0pO06lXyZ-3ABC"))
	require.Equal(t, "微信用户_z9", wechatFallbackUsername(" z_9 "))
	require.Equal(t, "微信用户_qr12", wechatFallbackUsername("union-qr-1_2"))
	require.Equal(t, "微信用户", wechatFallbackUsername(""))
	require.Equal(t, "微信用户", wechatFallbackUsername("—_—"))
}

func TestWeChatSyncProfileStartRedirectsToUserinfoAuthorize(t *testing.T) {
	handler, _ := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: wechatMPHandlerTestSettings()})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/sync-profile/start?redirect=%2Fauth%2Fwechat%2Fsynced", nil)
	c.Request.Host = "api.example.com"
	handler.WeChatSyncProfileStart(c)

	require.Equal(t, http.StatusFound, recorder.Code)
	location, err := url.Parse(recorder.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "open.weixin.qq.com", location.Host)
	require.Equal(t, "/connect/oauth2/authorize", location.Path)
	require.Equal(t, "snsapi_userinfo", location.Query().Get("scope"))
	require.Equal(t, wechatMPHandlerTestAppID, location.Query().Get("appid"))
	cookies := recorder.Result().Cookies()
	require.Equal(t, wechatOAuthIntentSyncProfile, decodeCookieValueForTest(t, findCookie(cookies, wechatOAuthIntentCookieName).Value))
	require.Equal(t, "mp", decodeCookieValueForTest(t, findCookie(cookies, wechatOAuthModeCookieName).Value))
	require.Equal(t, "/auth/wechat/synced", decodeCookieValueForTest(t, findCookie(cookies, wechatOAuthRedirectCookieName).Value))
	require.Equal(t, location.Query().Get("state"), decodeCookieValueForTest(t, findCookie(cookies, wechatOAuthStateCookieName).Value))
}

func wechatSyncFakeOAuth(t *testing.T, unionid, nickname string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/sns/oauth2/access_token":
			_, _ = w.Write([]byte(`{"access_token":"web-access","openid":"openid-sync","unionid":"` + unionid + `","scope":"snsapi_userinfo"}`))
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"openid":"openid-sync","unionid":"` + unionid + `","nickname":"` + nickname + `","headimgurl":"` + wechatSyncTestAvatar + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	originalAccessTokenURL, originalUserInfoURL := wechatOAuthAccessTokenURL, wechatOAuthUserInfoURL
	t.Cleanup(func() { wechatOAuthAccessTokenURL, wechatOAuthUserInfoURL = originalAccessTokenURL, originalUserInfoURL })
	wechatOAuthAccessTokenURL = srv.URL + "/sns/oauth2/access_token"
	wechatOAuthUserInfoURL = srv.URL + "/sns/userinfo"
}

func runWeChatSyncCallback(t *testing.T, handler *AuthHandler, state string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/wechat/callback?code=c&state="+state, nil)
	req.Host = "api.example.com"
	req.AddCookie(encodedCookie(wechatOAuthStateCookieName, "s1"))
	req.AddCookie(encodedCookie(wechatOAuthRedirectCookieName, "/auth/wechat/synced"))
	req.AddCookie(encodedCookie(wechatOAuthIntentCookieName, wechatOAuthIntentSyncProfile))
	req.AddCookie(encodedCookie(wechatOAuthModeCookieName, "mp"))
	c.Request = req
	handler.WeChatOAuthCallback(c)
	return recorder
}

func TestWeChatSyncProfileCallbackUpdatesBoundAccountOnly(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: wechatMPHandlerTestSettings()})
	ctx := context.Background()
	user, err := client.User.Create().
		SetEmail("wechat-union-sync@wechat-connect.invalid").
		SetUsername("微信用户_sync").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.AuthIdentity.Create().
		SetUserID(user.ID).
		SetProviderType("wechat").
		SetProviderKey(wechatOAuthProviderKey).
		SetProviderSubject("union-sync").
		SetMetadata(map[string]any{}).
		Save(ctx)
	require.NoError(t, err)
	wechatSyncFakeOAuth(t, "union-sync", "小蜜蜂 🐝")

	rec := runWeChatSyncCallback(t, handler, "s1")
	require.Equal(t, http.StatusFound, rec.Code)
	location, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/auth/wechat/synced", location.Path)
	require.Equal(t, "ok", location.Query().Get("status"))
	require.Equal(t, "小蜜蜂 🐝", location.Query().Get("name"))
	require.Equal(t, "/auth/wechat/synced", location.Query().Get("redirect"))

	updated, err := client.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "小蜜蜂 🐝", updated.Username)
	require.Equal(t, updated.Email, user.Email, "nothing but username and avatar changes")
	avatar := loadUserAvatarRecord(t, client, user.ID)
	require.NotNil(t, avatar)
	require.Equal(t, wechatSyncTestAvatar, avatar.URL)
	identity, err := client.AuthIdentity.Query().Only(ctx)
	require.NoError(t, err)
	require.Equal(t, "小蜜蜂 🐝", identity.Metadata["display_name"])
	require.Equal(t, wechatSyncTestAvatar, identity.Metadata["avatar_url"])

	// Never a login: no pending session, no session cookie.
	require.Nil(t, findCookie(rec.Result().Cookies(), oauthPendingSessionCookieName))
	count, err := client.PendingAuthSession.Query().Where(pendingauthsession.ProviderTypeEQ("wechat")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)

	// A wrong state is refused.
	rec = runWeChatSyncCallback(t, handler, "forged")
	require.Contains(t, rec.Header().Get("Location"), "status=error")
}

func TestWeChatSyncProfileCallbackNotBound(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{settingValues: wechatMPHandlerTestSettings()})
	wechatSyncFakeOAuth(t, "union-nobody", "Someone")

	rec := runWeChatSyncCallback(t, handler, "s1")
	require.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	require.True(t, strings.HasPrefix(location, "/auth/wechat/synced?"), location)
	require.Contains(t, location, "status=not_bound")
	count, err := client.User.Query().Count(context.Background())
	require.NoError(t, err)
	require.Zero(t, count, "no account is created")
	require.Nil(t, findCookie(rec.Result().Cookies(), oauthPendingSessionCookieName))
}
