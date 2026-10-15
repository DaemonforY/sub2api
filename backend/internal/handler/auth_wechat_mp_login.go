package handler

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 微信扫码登录（PC 浏览器）和公众号验证码登录（手机浏览器）。两者拿到 openid 后都走
// continueWeChatIdentity —— 与网页授权回调完全相同的身份流程，前端随后跳到 /auth/wechat/callback。

const (
	// Keys of the pending session's local flow state that carry sign-up data from a QR / code login
	// to quick-register / create-account.
	oauthSignupVisitorIDStateKey   = "signup_visitor_id"
	oauthSignupAttributionStateKey = "signup_attribution"
	oauthSignupAffCodeStateKey     = "signup_aff_code"

	oauthPendingLocalFlowExtrasCtxKey = "oauth_pending_local_flow_extras"

	wechatMPMessageMaxBody = 64 << 10

	// A WeChat sign-up counts as new when the account was created this recently.
	oauthSignupAttributionWindow = 5 * time.Minute
)

var errWeChatCodeBindUnsupported = infraerrors.BadRequest("WECHAT_CODE_BIND_UNSUPPORTED", "绑定微信请使用扫码，或在微信里打开本页（Use QR scan or open in WeChat to bind）")

var errWeChatMissingUnionID = infraerrors.BadRequest("WECHAT_MISSING_UNIONID", "没有拿到微信身份，请稍后再试（WeChat did not return the user identity）")

// SetWeChatMPLogin enables the 公众号 message server and the QR / code logins.
func (h *AuthHandler) SetWeChatMPLogin(svc *service.WeChatMPLoginService) {
	h.wechatMP = svc
}

type wechatMPLoginRequest struct {
	Redirect    string                        `json:"redirect"`
	Intent      string                        `json:"intent"`
	AffCode     string                        `json:"aff_code"`
	PromoCode   string                        `json:"promo_code"`
	VisitorID   string                        `json:"visitor_id"`
	Attribution *service.AnalyticsAttribution `json:"attribution"`

	TurnstileToken        string `json:"turnstile_token"`
	TencentCaptchaTicket  string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr string `json:"tencent_captcha_randstr"`
}

type wechatLoginStatusResponse struct {
	Status   string `json:"status"`
	Redirect string `json:"redirect,omitempty"`
	Message  string `json:"message,omitempty"`
}

func (h *AuthHandler) wechatMPService() (*service.WeChatMPLoginService, error) {
	if h == nil || h.wechatMP == nil {
		return nil, infraerrors.ServiceUnavailable("WECHAT_MP_NOT_READY", "微信登录服务未就绪（WeChat login service not ready）")
	}
	return h.wechatMP, nil
}

func bindOptionalJSON(c *gin.Context, dst any) error {
	if c.Request.Body == nil {
		return nil
	}
	if err := c.ShouldBindJSON(dst); err != nil && !errors.Is(err, io.EOF) {
		return infraerrors.BadRequest("INVALID_REQUEST", "请求参数不正确（Invalid request）")
	}
	return nil
}

// verifyWeChatMPLoginCaptcha applies the same captcha gating as POST /auth/oauth/wechat/start;
// binding to the signed-in account skips it, as /bind/start does.
func (h *AuthHandler) verifyWeChatMPLoginCaptcha(c *gin.Context, req wechatMPLoginRequest) bool {
	if normalizeWeChatOAuthIntent(req.Intent) == wechatOAuthIntentBind {
		return true
	}
	if err := h.authService.VerifyActionCaptchaIfEnabled(c.Request.Context(), service.CaptchaProof{
		TurnstileToken: req.TurnstileToken,
		TencentTicket:  req.TencentCaptchaTicket,
		TencentRandstr: req.TencentCaptchaRandstr,
	}, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	return true
}

func wechatMPLoginRedirect(raw string) string {
	redirectTo := sanitizeFrontendRedirectPath(raw)
	if redirectTo == "" {
		redirectTo = wechatOAuthDefaultRedirectTo
	}
	return redirectTo
}

func encodeAttribution(attr *service.AnalyticsAttribution) json.RawMessage {
	if attr == nil {
		return nil
	}
	raw, err := json.Marshal(attr)
	if err != nil {
		return nil
	}
	return raw
}

// prepareWeChatLoginTicket reads a create request (qr/create, code/create), applies the captcha
// gating and returns the ticket to store plus the browser session key. It writes the error response
// and returns false on failure.
func (h *AuthHandler) prepareWeChatLoginTicket(c *gin.Context, allowBind bool) (service.WeChatLoginTicket, string, bool) {
	var req wechatMPLoginRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		response.ErrorFrom(c, err)
		return service.WeChatLoginTicket{}, "", false
	}
	if !allowBind && normalizeWeChatOAuthIntent(req.Intent) == wechatOAuthIntentBind {
		response.ErrorFrom(c, errWeChatCodeBindUnsupported)
		return service.WeChatLoginTicket{}, "", false
	}
	if !h.verifyWeChatMPLoginCaptcha(c, req) {
		return service.WeChatLoginTicket{}, "", false
	}
	if _, err := h.getWeChatOAuthConfig(c.Request.Context(), "mp", c); err != nil {
		response.ErrorFrom(c, err)
		return service.WeChatLoginTicket{}, "", false
	}
	intent := normalizeWeChatOAuthIntent(req.Intent)
	bindUser := ""
	if intent == wechatOAuthIntentBind {
		value, err := h.buildOAuthBindUserCookieFromContext(c)
		if err != nil {
			response.ErrorFrom(c, err)
			return service.WeChatLoginTicket{}, "", false
		}
		bindUser = value
	}
	browserSessionKey, _ := readOAuthPendingBrowserCookie(c)
	if strings.TrimSpace(browserSessionKey) == "" {
		key, err := generateOAuthPendingBrowserSession()
		if err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BROWSER_SESSION_GEN_FAILED", "failed to generate oauth browser session").WithCause(err))
			return service.WeChatLoginTicket{}, "", false
		}
		browserSessionKey = key
	}
	return service.WeChatLoginTicket{
		BrowserSessionKey: browserSessionKey,
		Redirect:          wechatMPLoginRedirect(req.Redirect),
		Intent:            intent,
		AffCode:           strings.TrimSpace(req.AffCode),
		PromoCode:         strings.TrimSpace(req.PromoCode),
		VisitorID:         strings.TrimSpace(req.VisitorID),
		Attribution:       encodeAttribution(req.Attribution),
		BindUser:          bindUser,
	}, browserSessionKey, true
}

func respondWeChatLoginTicketCreated(c *gin.Context, browserSessionKey string, payload any) {
	secureCookie := isRequestHTTPS(c)
	setOAuthPendingBrowserCookie(c, browserSessionKey, secureCookie)
	clearOAuthPendingSessionCookie(c, secureCookie)
	c.Header("Cache-Control", "no-store")
	response.Success(c, payload)
}

// WeChatQRLoginCreate creates a 公众号 QR for PC browsers.
// POST /api/v1/auth/oauth/wechat/qr/create
func (h *AuthHandler) WeChatQRLoginCreate(c *gin.Context) {
	svc, err := h.wechatMPService()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	ticket, browserSessionKey, ok := h.prepareWeChatLoginTicket(c, true)
	if !ok {
		return
	}
	created, err := svc.CreateQRLogin(c.Request.Context(), ticket)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	respondWeChatLoginTicketCreated(c, browserSessionKey, created)
}

// WeChatCodeLoginCreate gives the browser a 6-digit code to send to the 公众号 (phone browsers,
// and the PC fallback). Poll qr/status with the ticket_id.
// POST /api/v1/auth/oauth/wechat/code/create
func (h *AuthHandler) WeChatCodeLoginCreate(c *gin.Context) {
	svc, err := h.wechatMPService()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// Binding goes only through the 128-bit QR ticket or 网页授权, never a 6-digit code.
	ticket, browserSessionKey, ok := h.prepareWeChatLoginTicket(c, false)
	if !ok {
		return
	}
	created, err := svc.CreateCodeLogin(c.Request.Context(), ticket)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	respondWeChatLoginTicketCreated(c, browserSessionKey, created)
}

func sameBrowser(cookieKey, ticketKey string) bool {
	return strings.TrimSpace(cookieKey) != "" &&
		subtle.ConstantTimeCompare([]byte(cookieKey), []byte(ticketKey)) == 1
}

// WeChatQRLoginStatus is polled by the browser that created a login ticket (QR or code). Once the
// 公众号 reports the openid, one poll atomically takes the ticket and runs the identity continuation.
// GET /api/v1/auth/oauth/wechat/qr/status?ticket_id=
func (h *AuthHandler) WeChatQRLoginStatus(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	svc, err := h.wechatMPService()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	ticketID := strings.TrimSpace(c.Query("ticket_id"))
	if !service.ValidWeChatLoginTicketID(ticketID) {
		response.ErrorFrom(c, service.ErrWeChatLoginTicketNotFound)
		return
	}
	ctx := c.Request.Context()
	browserSessionKey, _ := readOAuthPendingBrowserCookie(c)

	// A claimed ticket answers from its result (scanned while finishing, then done / error).
	result, err := svc.GetLoginResult(ctx, ticketID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if result != nil {
		if !sameBrowser(browserSessionKey, result.BrowserSessionKey) {
			response.ErrorFrom(c, service.ErrWeChatLoginTicketNotFound)
			return
		}
		response.Success(c, wechatLoginStatusResponse{Status: result.Status, Redirect: result.Redirect, Message: result.Message})
		return
	}

	ticket, err := svc.GetLoginTicket(ctx, ticketID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if ticket == nil {
		response.Success(c, wechatLoginStatusResponse{Status: service.WeChatLoginStatusExpired})
		return
	}
	if !sameBrowser(browserSessionKey, ticket.BrowserSessionKey) {
		response.ErrorFrom(c, service.ErrWeChatLoginTicketNotFound)
		return
	}
	if ticket.OpenID == "" {
		response.Success(c, wechatLoginStatusResponse{Status: service.WeChatLoginStatusPending})
		return
	}
	claimed, err := svc.ClaimLoginTicket(ctx, ticketID, ticket.BrowserSessionKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if claimed == nil {
		// Another poll took it and is finishing it right now.
		response.Success(c, wechatLoginStatusResponse{Status: service.WeChatLoginStatusScanned})
		return
	}

	if err := h.continueWeChatMPLogin(c, svc, claimed.OpenID, wechatMPLoginState{
		intent:            claimed.Intent,
		redirectTo:        claimed.Redirect,
		browserSessionKey: claimed.BrowserSessionKey,
		bindUserValue:     claimed.BindUser,
		promoCode:         claimed.PromoCode,
		affCode:           claimed.AffCode,
		visitorID:         claimed.VisitorID,
		attribution:       claimed.Attribution,
	}); err != nil {
		message := wechatLoginErrorMessage(err)
		svc.SetLoginResult(ctx, ticketID, service.WeChatLoginResult{
			Status:            service.WeChatLoginStatusError,
			Message:           message,
			BrowserSessionKey: claimed.BrowserSessionKey,
		})
		response.Success(c, wechatLoginStatusResponse{Status: service.WeChatLoginStatusError, Message: message})
		return
	}
	redirect := h.wechatOAuthFrontendCallback(ctx)
	svc.SetLoginResult(ctx, ticketID, service.WeChatLoginResult{
		Status:            service.WeChatLoginStatusDone,
		Redirect:          redirect,
		BrowserSessionKey: claimed.BrowserSessionKey,
	})
	response.Success(c, wechatLoginStatusResponse{Status: service.WeChatLoginStatusDone, Redirect: redirect})
}

// wechatLoginErrorMessage is the Chinese (English) text the polling page shows for a failed login.
func wechatLoginErrorMessage(err error) string {
	switch infraerrors.Code(err) {
	case http.StatusConflict:
		return "这个微信已经绑定了另一个账号，请换一个微信或先登录那个账号（This WeChat account is already bound to another account）"
	case http.StatusUnauthorized, http.StatusForbidden:
		if infraerrors.Reason(err) == "BACKEND_MODE_ADMIN_ONLY" {
			return infraerrors.Message(err)
		}
		return "登录状态已失效，请重新登录后再绑定微信（Please sign in again）"
	}
	if msg := infraerrors.Message(err); wechatMessageHasChinese(msg) {
		return msg
	}
	return "微信登录没能完成，请刷新后重试（WeChat login failed, please retry）"
}

func wechatMessageHasChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

type wechatMPLoginState struct {
	intent            string
	redirectTo        string
	browserSessionKey string
	bindUserValue     string
	promoCode         string
	affCode           string
	visitorID         string
	attribution       json.RawMessage
}

// continueWeChatMPLogin turns a 公众号 openid into the same pending session the 网页授权 callback makes.
func (h *AuthHandler) continueWeChatMPLogin(c *gin.Context, svc *service.WeChatMPLoginService, openid string, st wechatMPLoginState) error {
	ctx := c.Request.Context()
	cfg, err := h.getWeChatOAuthConfig(ctx, "mp", c)
	if err != nil {
		return err
	}
	unionid, err := svc.FetchUnionID(ctx, openid)
	if err != nil {
		return err
	}

	extras := map[string]any{}
	if st.promoCode != "" {
		extras[oauthPromoCodeStateKey] = st.promoCode
	}
	if st.affCode != "" {
		extras[oauthSignupAffCodeStateKey] = st.affCode
	}
	if st.visitorID != "" {
		extras[oauthSignupVisitorIDStateKey] = st.visitorID
	}
	if len(st.attribution) > 0 {
		var attr map[string]any
		if json.Unmarshal(st.attribution, &attr) == nil && len(attr) > 0 {
			extras[oauthSignupAttributionStateKey] = attr
		}
	}
	if len(extras) > 0 {
		c.Set(oauthPendingLocalFlowExtrasCtxKey, extras)
	}

	if cerr := h.continueWeChatIdentity(c, wechatIdentityInput{
		cfg:               cfg,
		unionid:           unionid,
		openid:            openid,
		intent:            st.intent,
		redirectTo:        st.redirectTo,
		browserSessionKey: st.browserSessionKey,
		bindUserValue:     st.bindUserValue,
	}); cerr != nil {
		if cerr.err != nil && infraerrors.Code(cerr.err) >= http.StatusBadRequest && infraerrors.Code(cerr.err) < http.StatusInternalServerError {
			return cerr.err
		}
		return infraerrors.InternalServer("WECHAT_LOGIN_CONTINUE_FAILED", "微信登录没能完成，请重试（failed to continue WeChat login）").WithCause(cerr.err)
	}
	return nil
}

// oauthPendingLocalFlowExtras returns extra local-flow state set for this request (QR / code login).
func oauthPendingLocalFlowExtras(c *gin.Context) map[string]any {
	if c == nil {
		return nil
	}
	v, ok := c.Get(oauthPendingLocalFlowExtrasCtxKey)
	if !ok {
		return nil
	}
	extras, _ := v.(map[string]any)
	return extras
}

// pendingOAuthSignupData returns the visitor ID, attribution and invite code a QR / code login saved
// with the pending session, for when the sign-up request itself doesn't carry them.
func pendingOAuthSignupData(session *dbent.PendingAuthSession) (visitorID string, attr *service.AnalyticsAttribution, affCode string) {
	if session == nil {
		return "", nil, ""
	}
	state := session.LocalFlowState
	visitorID = pendingSessionStringValue(state, oauthSignupVisitorIDStateKey)
	affCode = pendingSessionStringValue(state, oauthSignupAffCodeStateKey)
	if raw, ok := state[oauthSignupAttributionStateKey]; ok && raw != nil {
		if b, err := json.Marshal(raw); err == nil {
			var a service.AnalyticsAttribution
			if json.Unmarshal(b, &a) == nil {
				attr = &a
			}
		}
	}
	return visitorID, attr, affCode
}

// oauthSignupData is the browser data a WeChat sign-up carries (request first, then pending session).
type oauthSignupData struct {
	visitorID   string
	attribution *service.AnalyticsAttribution
	affCode     string
}

func resolveOAuthSignupData(session *dbent.PendingAuthSession, visitorID string, attr *service.AnalyticsAttribution, affCode string) oauthSignupData {
	storedVisitor, storedAttr, storedAff := pendingOAuthSignupData(session)
	out := oauthSignupData{
		visitorID:   strings.TrimSpace(firstNonEmpty(visitorID, storedVisitor)),
		attribution: attr,
		affCode:     strings.TrimSpace(firstNonEmpty(affCode, storedAff)),
	}
	if out.attribution == nil {
		out.attribution = storedAttr
	}
	return out
}

// withOAuthSignupClient puts the sign-up's browser and IP on the request context so the sign-up
// trial guard (注册试用去重 / 静默限制) sees them, as the email register does.
func withOAuthSignupClient(c *gin.Context, visitorID string) {
	c.Request = c.Request.WithContext(service.WithSignupClient(c.Request.Context(), service.SignupClient{
		VisitorID: strings.TrimSpace(visitorID),
		IP:        ip.GetClientIP(c),
	}))
}

// recordOAuthSignupAttribution records where a just-created OAuth account came from. An existing
// account that merely logged in is left alone.
func (h *AuthHandler) recordOAuthSignupAttribution(c *gin.Context, userID int64, createdAt time.Time, data oauthSignupData) {
	if userID <= 0 || createdAt.IsZero() || time.Since(createdAt) > oauthSignupAttributionWindow {
		return
	}
	h.recordSignupAttribution(c, &service.User{ID: userID}, data.visitorID, data.attribution, data.affCode)
}

// WeChatMPMessage is the 公众号 消息服务器 endpoint (服务器配置 URL).
// GET  /api/v1/wechat/mp/message — URL verification (echostr)
// POST /api/v1/wechat/mp/message — messages and events
func (h *AuthHandler) WeChatMPMessage(c *gin.Context) {
	svc, err := h.wechatMPService()
	if err != nil {
		c.String(http.StatusServiceUnavailable, "")
		return
	}
	if c.Request.Method == http.MethodGet {
		echo, err := svc.VerifyServerURL(c.Request.Context(), c.Request.URL.Query())
		if err != nil {
			c.String(infraerrors.Code(err), "")
			return
		}
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(echo))
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, wechatMPMessageMaxBody+1))
	if err != nil || len(body) > wechatMPMessageMaxBody {
		c.String(http.StatusRequestEntityTooLarge, "")
		return
	}
	resp, err := svc.HandleMessage(c.Request.Context(), service.WeChatMPMessageRequest{
		Method:      c.Request.Method,
		RawQuery:    c.Request.URL.RawQuery,
		ContentType: c.GetHeader("Content-Type"),
		Body:        body,
	})
	if err != nil {
		c.String(infraerrors.Code(err), "")
		return
	}
	contentType := resp.ContentType
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	c.Data(resp.Status, contentType, resp.Body)
}
