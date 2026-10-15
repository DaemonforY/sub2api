package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 同步微信昵称和头像：QR / 验证码登录只拿得到 openid（2021 年起公众号接口不再返回昵称），新账号叫
// 「微信用户_xxxx」。在微信里走一次网页授权（snsapi_userinfo）就能拿到昵称和头像，写到已经绑定这个
// 微信身份的账号上。这个流程只改用户名和头像，从不创建登录会话。

const (
	wechatOAuthIntentSyncProfile = "sync_profile"
	wechatSyncProfileScope       = "snsapi_userinfo"
	// WeChatSyncProfileFrontendPath shows the outcome: ?status=ok|not_bound|error&name=&redirect=
	wechatSyncProfileFrontendPath = "/auth/wechat/synced"

	wechatSyncStatusOK       = "ok"
	wechatSyncStatusNotBound = "not_bound"
	wechatSyncStatusError    = "error"
)

func redirectWeChatSyncResult(c *gin.Context, status, name, redirectTo string) {
	q := url.Values{}
	q.Set("status", status)
	if name != "" {
		q.Set("name", name)
	}
	q.Set("redirect", redirectTo)
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, wechatSyncProfileFrontendPath+"?"+q.Encode())
}

func wechatSyncRedirectTarget(raw string) string {
	redirectTo := sanitizeFrontendRedirectPath(raw)
	if redirectTo == "" {
		redirectTo = wechatOAuthDefaultRedirectTo
	}
	return redirectTo
}

// WeChatSyncProfileStart starts 网页授权 with snsapi_userinfo to copy the WeChat nickname and avatar.
// It only works inside WeChat; elsewhere WeChat shows its own 「请在微信客户端打开」 page.
// GET /api/v1/auth/oauth/wechat/sync-profile/start?redirect=<same-site path>
func (h *AuthHandler) WeChatSyncProfileStart(c *gin.Context) {
	redirectTo := wechatSyncRedirectTarget(c.Query("redirect"))
	cfg, err := h.getWeChatOAuthConfig(c.Request.Context(), "mp", c)
	if err != nil {
		slog.Warn("wechat profile sync start: oauth config unavailable", "error", err.Error())
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	state, err := oauth.GenerateState()
	if err != nil {
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	secureCookie := isRequestHTTPS(c)
	wechatSetCookie(c, wechatOAuthStateCookieName, encodeCookieValue(state), wechatOAuthCookieMaxAgeSec, secureCookie)
	wechatSetCookie(c, wechatOAuthRedirectCookieName, encodeCookieValue(redirectTo), wechatOAuthCookieMaxAgeSec, secureCookie)
	wechatSetCookie(c, wechatOAuthIntentCookieName, encodeCookieValue(wechatOAuthIntentSyncProfile), wechatOAuthCookieMaxAgeSec, secureCookie)
	wechatSetCookie(c, wechatOAuthModeCookieName, encodeCookieValue("mp"), wechatOAuthCookieMaxAgeSec, secureCookie)
	wechatClearCookie(c, wechatOAuthBindUserCookieName, secureCookie)

	cfg.scope = wechatSyncProfileScope
	authURL, err := buildWeChatAuthorizeURL(cfg, state)
	if err != nil {
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, authURL)
}

// wechatSyncProfileCallback is WeChatOAuthCallback for intent sync_profile. Same state-cookie check
// as the login flow; it never creates a pending session or logs anyone in.
func (h *AuthHandler) wechatSyncProfileCallback(c *gin.Context) {
	secureCookie := isRequestHTTPS(c)
	defer func() {
		wechatClearCookie(c, wechatOAuthStateCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthRedirectCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthIntentCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthModeCookieName, secureCookie)
		wechatClearCookie(c, wechatOAuthBindUserCookieName, secureCookie)
	}()
	redirectCookie, _ := readCookieDecoded(c, wechatOAuthRedirectCookieName)
	redirectTo := wechatSyncRedirectTarget(redirectCookie)

	if strings.TrimSpace(c.Query("error")) != "" {
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	expectedState, err := readCookieDecoded(c, wechatOAuthStateCookieName)
	if code == "" || state == "" || err != nil || expectedState == "" || state != expectedState {
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	cfg, err := h.getWeChatOAuthConfig(c.Request.Context(), "mp", c)
	if err != nil {
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	tokenResp, userInfo, err := fetchWeChatOAuthIdentity(c.Request.Context(), cfg, code)
	if err != nil {
		slog.Warn("wechat profile sync: identity fetch failed", "error", singleLine(err.Error()))
		redirectWeChatSyncResult(c, wechatSyncStatusError, "", redirectTo)
		return
	}
	status, name := h.syncWeChatProfile(c.Request.Context(), cfg,
		firstNonEmpty(userInfo.UnionID, tokenResp.UnionID),
		firstNonEmpty(userInfo.OpenID, tokenResp.OpenID),
		userInfo.Nickname, userInfo.HeadImgURL)
	redirectWeChatSyncResult(c, status, name, redirectTo)
}

// syncWeChatProfile writes the WeChat nickname and avatar onto the active account bound to this
// WeChat identity (and nothing else). Returns the status and the name the account now has.
func (h *AuthHandler) syncWeChatProfile(ctx context.Context, cfg wechatOAuthConfig, unionid, openid, nickname, avatarURL string) (string, string) {
	unionid, openid = strings.TrimSpace(unionid), strings.TrimSpace(openid)
	providerSubject := unionid
	if providerSubject == "" {
		if cfg.requiresUnionID() {
			return wechatSyncStatusError, ""
		}
		providerSubject = openid
	}
	if providerSubject == "" {
		return wechatSyncStatusError, ""
	}
	identityRef := service.PendingAuthIdentityKey{
		ProviderType:    "wechat",
		ProviderKey:     wechatOAuthProviderKey,
		ProviderSubject: providerSubject,
	}
	user, err := h.findOAuthIdentityUser(ctx, identityRef)
	if err == nil && user == nil {
		user, err = h.findWeChatUserByLegacyOpenID(ctx, identityRef, cfg, openid)
	}
	if err != nil {
		slog.Warn("wechat profile sync: identity lookup failed", "error", err.Error())
		return wechatSyncStatusError, ""
	}
	if user == nil {
		return wechatSyncStatusNotBound, ""
	}
	client := h.entClient()
	if client == nil {
		return wechatSyncStatusError, ""
	}

	name := normalizeAdoptedOAuthDisplayName(nickname)
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL != "" && service.ValidateUserAvatar(avatarURL) != nil {
		avatarURL = ""
	}
	claims := map[string]any{
		"subject":         providerSubject,
		"openid":          openid,
		"unionid":         unionid,
		"mode":            cfg.mode,
		"channel":         cfg.mode,
		"channel_app_id":  strings.TrimSpace(cfg.appID),
		"channel_subject": openid,
	}
	if name != "" {
		claims["display_name"] = name
		claims["suggested_display_name"] = name
	}
	if avatarURL != "" {
		claims["avatar_url"] = avatarURL
		claims["suggested_avatar_url"] = avatarURL
	}
	if err := h.ensureWeChatRuntimeIdentityBinding(ctx, user.ID, identityRef, claims); err != nil {
		slog.Warn("wechat profile sync: identity metadata update failed", "user_id", user.ID, "error", err.Error())
		return wechatSyncStatusError, ""
	}
	if name != "" {
		if err := client.User.UpdateOneID(user.ID).SetUsername(name).Exec(ctx); err != nil {
			slog.Warn("wechat profile sync: username update failed", "user_id", user.ID, "error", err.Error())
			return wechatSyncStatusError, ""
		}
	}
	if avatarURL != "" && h.userService != nil {
		if _, err := h.userService.SetAvatar(ctx, user.ID, avatarURL); err != nil {
			slog.Warn("wechat profile sync: avatar update failed", "user_id", user.ID, "error", err.Error())
			return wechatSyncStatusError, ""
		}
	}
	h.userService.InvalidateAuthCache(ctx, user.ID)
	if name == "" {
		name = user.Username
	}
	return wechatSyncStatusOK, name
}
