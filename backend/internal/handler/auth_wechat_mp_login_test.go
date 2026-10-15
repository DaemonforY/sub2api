//go:build unit

package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/pendingauthsession"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

const (
	wechatMPHandlerTestAppID  = "wx-mp-app"
	wechatMPHandlerTestToken  = "mptoken"
	wechatMPHandlerTestAESKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
)

func wechatMPHandlerTestSettings() map[string]string {
	return map[string]string{
		service.SettingKeyWeChatConnectEnabled:             "true",
		service.SettingKeyWeChatConnectMode:                "mp",
		service.SettingKeyWeChatConnectOpenEnabled:         "false",
		service.SettingKeyWeChatConnectMPEnabled:           "true",
		service.SettingKeyWeChatConnectMPAppID:             wechatMPHandlerTestAppID,
		service.SettingKeyWeChatConnectMPAppSecret:         "wx-mp-secret",
		service.SettingKeyWeChatConnectScopes:              "snsapi_userinfo",
		service.SettingKeyWeChatMPMessageEnabled:           "true",
		service.SettingKeyWeChatMPMessageToken:             wechatMPHandlerTestToken,
		service.SettingKeyWeChatMPMessageAESKey:            wechatMPHandlerTestAESKey,
		service.SettingKeyWeChatQRLoginEnabled:             "true",
		service.SettingKeyWeChatCodeLoginEnabled:           "true",
		service.SettingKeyWeChatConnectAppID:               wechatMPHandlerTestAppID,
		service.SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
		service.SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
	}
}

// wechatMPFakeAPI answers stable_token, qrcode/create and user/info like api.weixin.qq.com, and the
// 网页授权 endpoints for the callback comparison.
func wechatMPFakeAPI(t *testing.T, unionid string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			_, _ = w.Write([]byte(`{"access_token":"mp-token","expires_in":7200}`))
		case "/cgi-bin/qrcode/create":
			_, _ = w.Write([]byte(`{"ticket":"qr-ticket","expire_seconds":300}`))
		case "/cgi-bin/user/info":
			if strings.HasPrefix(r.URL.Query().Get("openid"), "fail") {
				_, _ = w.Write([]byte(`{"errcode":45009,"errmsg":"api freq out of limit"}`))
				return
			}
			_, _ = w.Write([]byte(`{"subscribe":1,"openid":"` + r.URL.Query().Get("openid") + `","unionid":"` + unionid + `"}`))
		case "/sns/oauth2/access_token":
			_, _ = w.Write([]byte(`{"access_token":"web-access","openid":"openid-web","unionid":"` + unionid + `-web","scope":"snsapi_userinfo"}`))
		case "/sns/userinfo":
			_, _ = w.Write([]byte(`{"openid":"openid-web","unionid":"` + unionid + `-web","nickname":"Web Nick","headimgurl":"https://cdn.example/a.png"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

type wechatMPHandlerEnv struct {
	handler *AuthHandler
	client  *dbent.Client
	svc     *service.WeChatMPLoginService
	mr      *miniredis.Miniredis
	api     *httptest.Server
}

func newWeChatMPHandlerEnv(t *testing.T, unionid string) *wechatMPHandlerEnv {
	t.Helper()
	handler, client := newWeChatOAuthTestHandlerWithSettings(t, false, wechatMPHandlerTestSettings())
	t.Cleanup(func() { _ = client.Close() })
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	api := wechatMPFakeAPI(t, unionid)
	svc := service.NewWeChatMPLoginService(handler.settingSvc, repository.NewWeChatMPLoginStore(rdb))
	svc.SetAPIBaseURL(api.URL)
	handler.SetWeChatMPLogin(svc)
	return &wechatMPHandlerEnv{handler: handler, client: client, svc: svc, mr: mr, api: api}
}

func (e *wechatMPHandlerEnv) do(t *testing.T, method, target, body string, cookies []*http.Cookie, fn func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.Host = "api.example.com"
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	c.Request = req
	fn(c)
	return recorder
}

func envelopeData(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Reason  string         `json:"reason"`
		Data    map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &env), recorder.Body.String())
	return env.Data
}

func pendingSessionFromRecorder(t *testing.T, client *dbent.Client, recorder *httptest.ResponseRecorder) *dbent.PendingAuthSession {
	t.Helper()
	var cookie *http.Cookie
	for _, ck := range recorder.Result().Cookies() {
		if ck.Name == oauthPendingSessionCookieName && ck.Value != "" {
			cookie = ck // the last one wins, as in a browser
		}
	}
	require.NotNil(t, cookie, "the pending session cookie is set")
	session, err := client.PendingAuthSession.Query().
		Where(pendingauthsession.SessionTokenEQ(decodeCookieValueForTest(t, cookie.Value))).
		Only(context.Background())
	require.NoError(t, err)
	return session
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func createQRTicketViaHandler(t *testing.T, env *wechatMPHandlerEnv, body string, cookies []*http.Cookie) (string, *http.Cookie) {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/v1/auth/oauth/wechat/qr/create", body, cookies, env.handler.WeChatQRLoginCreate)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := envelopeData(t, rec)
	ticketID, _ := data["ticket_id"].(string)
	require.Len(t, ticketID, 32)
	require.Equal(t, "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=qr-ticket", data["qr_url"])
	require.EqualValues(t, 300, data["expires_in"])
	require.EqualValues(t, 2, data["poll_interval"])
	browser := findCookie(rec.Result().Cookies(), oauthPendingBrowserCookieName)
	require.NotNil(t, browser)
	return ticketID, browser
}

func TestWeChatQRLoginStatusFlowCreatesCallbackShapedPendingSession(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "union-qr")
	ctx := context.Background()

	ticketID, browser := createQRTicketViaHandler(t, env,
		`{"redirect":"/image","aff_code":"AFF1","promo_code":"PROMO1","visitor_id":"v_abcdefgh12345678","attribution":{"source":"xhs","landing":"/"}}`, nil)
	browserCookie := &http.Cookie{Name: oauthPendingBrowserCookieName, Value: browser.Value}
	statusURL := "/api/v1/auth/oauth/wechat/qr/status?ticket_id=" + ticketID

	// Not scanned yet.
	rec := env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "pending", envelopeData(t, rec)["status"])

	// Another browser can't see or finish it.
	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{encodedCookie(oauthPendingBrowserCookieName, "someone-else")}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = env.do(t, http.MethodGet, statusURL, "", nil, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusNotFound, rec.Code)

	// The 公众号 reports the scan.
	bound, err := env.svc.BindLoginTicket(ctx, ticketID, "openid-qr")
	require.NoError(t, err)
	require.True(t, bound)

	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := envelopeData(t, rec)
	require.Equal(t, "done", data["status"])
	require.Equal(t, "/auth/wechat/callback", data["redirect"])
	qrSession := pendingSessionFromRecorder(t, env.client, rec)

	// The ticket is used up: later polls answer done from the result, without another session.
	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, "done", envelopeData(t, rec)["status"])
	require.Nil(t, findCookie(rec.Result().Cookies(), oauthPendingSessionCookieName))
	ticket, err := env.svc.GetLoginTicket(ctx, ticketID)
	require.NoError(t, err)
	require.Nil(t, ticket)

	require.Equal(t, "wechat", qrSession.ProviderType)
	require.Equal(t, wechatOAuthProviderKey, qrSession.ProviderKey)
	require.Equal(t, "union-qr", qrSession.ProviderSubject)
	require.Equal(t, "wechat-union-qr@wechat-connect.invalid", qrSession.ResolvedEmail)
	require.Equal(t, decodeCookieValueForTest(t, browser.Value), qrSession.BrowserSessionKey)
	require.Equal(t, "/image", qrSession.RedirectTo)
	claims := qrSession.UpstreamIdentityClaims
	require.Equal(t, "openid-qr", claims["openid"])
	require.Equal(t, "openid-qr", claims["channel_subject"])
	require.Equal(t, "mp", claims["mode"])
	require.Equal(t, "mp", claims["channel"])
	require.Equal(t, wechatMPHandlerTestAppID, claims["channel_app_id"])
	require.Equal(t, wechatFallbackUsername("union-qr"), claims["username"])
	require.Equal(t, "PROMO1", qrSession.LocalFlowState[oauthPromoCodeStateKey])
	require.Equal(t, "AFF1", qrSession.LocalFlowState[oauthSignupAffCodeStateKey])
	require.Equal(t, "v_abcdefgh12345678", qrSession.LocalFlowState[oauthSignupVisitorIDStateKey])

	// The same identity through 网页授权 (callback, mp mode) gives a session of the same shape.
	originalAccessTokenURL, originalUserInfoURL := wechatOAuthAccessTokenURL, wechatOAuthUserInfoURL
	t.Cleanup(func() {
		wechatOAuthAccessTokenURL, wechatOAuthUserInfoURL = originalAccessTokenURL, originalUserInfoURL
	})
	wechatOAuthAccessTokenURL = env.api.URL + "/sns/oauth2/access_token"
	wechatOAuthUserInfoURL = env.api.URL + "/sns/userinfo"
	rec = env.do(t, http.MethodGet, "/api/v1/auth/oauth/wechat/callback?code=c&state=s", "", []*http.Cookie{
		encodedCookie(wechatOAuthStateCookieName, "s"),
		encodedCookie(wechatOAuthRedirectCookieName, "/image"),
		encodedCookie(wechatOAuthModeCookieName, "mp"),
		encodedCookie(oauthPendingBrowserCookieName, "browser-web"),
	}, env.handler.WeChatOAuthCallback)
	require.Equal(t, http.StatusFound, rec.Code)
	require.Equal(t, "/auth/wechat/callback", rec.Header().Get("Location"))
	webSession := pendingSessionFromRecorder(t, env.client, rec)

	require.Equal(t, webSession.ProviderType, qrSession.ProviderType)
	require.Equal(t, webSession.ProviderKey, qrSession.ProviderKey)
	require.Equal(t, webSession.Intent, qrSession.Intent)
	require.Equal(t, sortedKeys(webSession.UpstreamIdentityClaims), sortedKeys(qrSession.UpstreamIdentityClaims))
	require.Equal(t, webSession.UpstreamIdentityClaims["channel_app_id"], qrSession.UpstreamIdentityClaims["channel_app_id"])
	require.Equal(t, webSession.UpstreamIdentityClaims["mode"], qrSession.UpstreamIdentityClaims["mode"])
	webCompletion, _ := readCompletionResponse(webSession.LocalFlowState)
	qrCompletion, _ := readCompletionResponse(qrSession.LocalFlowState)
	require.Equal(t, sortedKeys(webCompletion), sortedKeys(qrCompletion))
	require.Equal(t, webCompletion["step"], qrCompletion["step"])
	require.Equal(t, oauthPendingChoiceStep, qrCompletion["step"])
}

func TestWeChatQRLoginStatusLogsInExistingIdentity(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "union-existing")
	ctx := context.Background()
	user, err := env.client.User.Create().
		SetEmail("existing@example.com").
		SetUsername("existing").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	_, err = env.client.AuthIdentity.Create().
		SetUserID(user.ID).
		SetProviderType("wechat").
		SetProviderKey(wechatOAuthProviderKey).
		SetProviderSubject("union-existing").
		SetMetadata(map[string]any{}).
		Save(ctx)
	require.NoError(t, err)

	ticketID, browser := createQRTicketViaHandler(t, env, `{}`, nil)
	_, err = env.svc.BindLoginTicket(ctx, ticketID, "openid-existing")
	require.NoError(t, err)
	rec := env.do(t, http.MethodGet, "/api/v1/auth/oauth/wechat/qr/status?ticket_id="+ticketID, "",
		[]*http.Cookie{{Name: oauthPendingBrowserCookieName, Value: browser.Value}}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "done", envelopeData(t, rec)["status"])
	session := pendingSessionFromRecorder(t, env.client, rec)
	require.NotNil(t, session.TargetUserID)
	require.Equal(t, user.ID, *session.TargetUserID)
	require.Equal(t, "/dashboard", session.RedirectTo)
}

// postEncryptedMessage sends inner to the message endpoint the way WeChat does in 安全模式.
func postEncryptedMessage(t *testing.T, env *wechatMPHandlerEnv, inner, nonce string) *httptest.ResponseRecorder {
	t.Helper()
	crypter, err := service.NewWeChatMPCrypter(wechatMPHandlerTestAESKey, wechatMPHandlerTestAppID, rand.Reader)
	require.NoError(t, err)
	encrypted, err := crypter.Encrypt([]byte(inner))
	require.NoError(t, err)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	q := url.Values{}
	q.Set("timestamp", ts)
	q.Set("nonce", nonce)
	q.Set("signature", service.WeChatMPSignature(wechatMPHandlerTestToken, ts, nonce))
	q.Set("encrypt_type", "aes")
	q.Set("msg_signature", service.WeChatMPSignature(wechatMPHandlerTestToken, ts, nonce, encrypted))
	body := "<xml><ToUserName><![CDATA[gh]]></ToUserName><Encrypt><![CDATA[" + encrypted + "]]></Encrypt></xml>"
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/wechat/mp/message?"+q.Encode(), strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "text/xml")
	env.handler.WeChatMPMessage(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	return recorder
}

func decryptReplyContent(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var outer struct {
		Encrypt string `xml:"Encrypt"`
	}
	require.NoError(t, xml.Unmarshal(recorder.Body.Bytes(), &outer), recorder.Body.String())
	crypter, err := service.NewWeChatMPCrypter(wechatMPHandlerTestAESKey, wechatMPHandlerTestAppID, rand.Reader)
	require.NoError(t, err)
	inner, err := crypter.Decrypt(outer.Encrypt)
	require.NoError(t, err)
	var reply struct {
		Content string `xml:"Content"`
	}
	require.NoError(t, xml.Unmarshal(inner, &reply))
	return reply.Content
}

func textMessage(from, content string, msgID int) string {
	return "<xml><ToUserName><![CDATA[gh]]></ToUserName><FromUserName><![CDATA[" + from + "]]></FromUserName><CreateTime>1</CreateTime>" +
		"<MsgType><![CDATA[text]]></MsgType><Content><![CDATA[" + content + "]]></Content><MsgId>" + strconv.Itoa(msgID) + "</MsgId></xml>"
}

func TestWeChatCodeLoginCreateThenEncryptedTextCompletes(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "")

	rec := env.do(t, http.MethodPost, "/api/v1/auth/oauth/wechat/code/create",
		`{"redirect":"/canvas","visitor_id":"v_code123456789012"}`, nil, env.handler.WeChatCodeLoginCreate)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := envelopeData(t, rec)
	ticketID, _ := data["ticket_id"].(string)
	code, _ := data["code"].(string)
	require.Len(t, ticketID, 32)
	require.Regexp(t, `^\d{6}$`, code)
	require.EqualValues(t, 300, data["expires_in"])
	require.EqualValues(t, 2, data["poll_interval"])
	browser := findCookie(rec.Result().Cookies(), oauthPendingBrowserCookieName)
	require.NotNil(t, browser)
	browserCookie := &http.Cookie{Name: oauthPendingBrowserCookieName, Value: browser.Value}
	statusURL := "/api/v1/auth/oauth/wechat/qr/status?ticket_id=" + ticketID

	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, "pending", envelopeData(t, rec)["status"])

	// The user sends 「登录 123456」 to the 公众号.
	reply := decryptReplyContent(t, postEncryptedMessage(t, env, textMessage("openid-code", "登录 "+code, 1), "n-code-1"))
	require.Contains(t, reply, "已确认")

	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data = envelopeData(t, rec)
	require.Equal(t, "done", data["status"])
	require.Equal(t, "/auth/wechat/callback", data["redirect"])
	session := pendingSessionFromRecorder(t, env.client, rec)
	require.Equal(t, decodeCookieValueForTest(t, browser.Value), session.BrowserSessionKey)
	// No unionid from user/info: the openid is the subject (single-channel 公众号).
	require.Equal(t, "openid-code", session.ProviderSubject)
	require.Equal(t, "/canvas", session.RedirectTo)
	require.Equal(t, "mp", session.UpstreamIdentityClaims["mode"])
	require.Equal(t, "v_code123456789012", session.LocalFlowState[oauthSignupVisitorIDStateKey])

	// The code is used up.
	reply = decryptReplyContent(t, postEncryptedMessage(t, env, textMessage("openid-other", code, 2), "n-code-2"))
	require.Contains(t, reply, "验证码无效或已过期")
}

func TestWeChatCodeLoginCreateRejectsBind(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "")
	rec := env.do(t, http.MethodPost, "/api/v1/auth/oauth/wechat/code/create", `{"intent":"bind_current_user"}`, nil, env.handler.WeChatCodeLoginCreate)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "WECHAT_CODE_BIND_UNSUPPORTED")
	require.Contains(t, rec.Body.String(), "绑定微信请使用扫码")
}

func TestWeChatLoginStatusDoublePollCompletesOnce(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "union-double")
	ctx := context.Background()
	ticketID, browser := createQRTicketViaHandler(t, env, `{}`, nil)
	browserCookie := &http.Cookie{Name: oauthPendingBrowserCookieName, Value: browser.Value}
	_, err := env.svc.BindLoginTicket(ctx, ticketID, "openid-double")
	require.NoError(t, err)
	statusURL := "/api/v1/auth/oauth/wechat/qr/status?ticket_id=" + ticketID

	var wg sync.WaitGroup
	statuses := make([]string, 4)
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
			statuses[i], _ = envelopeData(t, rec)["status"].(string)
		}(i)
	}
	wg.Wait()
	for _, st := range statuses {
		require.Contains(t, []string{"scanned", "done"}, st)
	}
	require.Contains(t, statuses, "done")

	count, err := env.client.PendingAuthSession.Query().Where(pendingauthsession.ProviderSubjectEQ("union-double")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count, "the ticket completes once")

	// Later polls keep answering done for a minute.
	rec := env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, "done", envelopeData(t, rec)["status"])
	require.Nil(t, findCookie(rec.Result().Cookies(), oauthPendingSessionCookieName), "no second pending session")
	// Not for another browser.
	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{encodedCookie(oauthPendingBrowserCookieName, "other")}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestWeChatLoginStatusSurfacesError(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "")
	ctx := context.Background()
	ticketID, browser := createQRTicketViaHandler(t, env, `{}`, nil)
	browserCookie := &http.Cookie{Name: oauthPendingBrowserCookieName, Value: browser.Value}
	_, err := env.svc.BindLoginTicket(ctx, ticketID, "fail-openid") // user/info fails for this one
	require.NoError(t, err)
	statusURL := "/api/v1/auth/oauth/wechat/qr/status?ticket_id=" + ticketID

	rec := env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := envelopeData(t, rec)
	require.Equal(t, "error", data["status"])
	require.Contains(t, data["message"], "微信接口暂时不可用")

	// The browser keeps seeing the error, not "expired".
	rec = env.do(t, http.MethodGet, statusURL, "", []*http.Cookie{browserCookie}, env.handler.WeChatQRLoginStatus)
	data = envelopeData(t, rec)
	require.Equal(t, "error", data["status"])
	require.Contains(t, data["message"], "微信接口暂时不可用")
}

func TestWeChatMPMessageEndpoint(t *testing.T) {
	env := newWeChatMPHandlerEnv(t, "")
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sig := service.WeChatMPSignature(wechatMPHandlerTestToken, ts, "n")
	rec := env.do(t, http.MethodGet, "/api/v1/wechat/mp/message?signature="+sig+"&timestamp="+ts+"&nonce=n&echostr=hello", "", nil, env.handler.WeChatMPMessage)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "hello", rec.Body.String())

	rec = env.do(t, http.MethodGet, "/api/v1/wechat/mp/message?signature=bad&timestamp="+ts+"&nonce=n&echostr=hello", "", nil, env.handler.WeChatMPMessage)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Empty(t, rec.Body.String())

	// A plain message while the EncodingAESKey is set: ignored.
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/wechat/mp/message?signature="+sig+"&timestamp="+ts+"&nonce=n&openid=openid-m", strings.NewReader(textMessage("openid-m", "登录", 9)))
	c.Request.Header.Set("Content-Type", "text/xml")
	env.handler.WeChatMPMessage(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "success", recorder.Body.String())

	// The keyword in 安全模式 gets the instructions.
	reply := decryptReplyContent(t, postEncryptedMessage(t, env, textMessage("openid-m", "登录", 10), "n-kw"))
	require.Contains(t, reply, "微信登录")
}

// ---------- sign-up attribution ----------

type wechatAttributionRepo struct {
	mu    sync.Mutex
	saved []service.UserAttribution
}

func (r *wechatAttributionRepo) InsertEvents(context.Context, []service.AnalyticsEventRow) error {
	return nil
}
func (r *wechatAttributionRepo) SaveAttribution(_ context.Context, a service.UserAttribution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, a)
	return nil
}
func (r *wechatAttributionRepo) SaveLateAttribution(context.Context, service.UserAttribution, time.Time) error {
	return nil
}
func (r *wechatAttributionRepo) DeleteEventsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *wechatAttributionRepo) Overview(context.Context, time.Time, int) (*service.AnalyticsOverview, error) {
	return &service.AnalyticsOverview{}, nil
}

type wechatFingerprintRepo struct {
	mu    sync.Mutex
	saved []service.SignupFingerprint
}

func (r *wechatFingerprintRepo) FindRecentTrial(context.Context, string, time.Time) (service.FingerprintMatch, error) {
	return service.FingerprintMatch{}, nil
}
func (r *wechatFingerprintRepo) FindFlaggedMatch(context.Context, string, string, int64, time.Time, time.Time) (service.FingerprintMatch, error) {
	return service.FingerprintMatch{}, nil
}
func (r *wechatFingerprintRepo) Save(_ context.Context, fp service.SignupFingerprint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, fp)
	return nil
}

func createWeChatChoiceSession(t *testing.T, client *dbent.Client, token, browser, subject string, localState map[string]any) *dbent.PendingAuthSession {
	t.Helper()
	state := map[string]any{
		oauthCompletionResponseKey: map[string]any{
			"step":                  oauthPendingChoiceStep,
			"redirect":              "/dashboard",
			"force_email_on_signup": false,
		},
	}
	for k, v := range localState {
		state[k] = v
	}
	session, err := client.PendingAuthSession.Create().
		SetSessionToken(token).
		SetIntent("login").
		SetProviderType("wechat").
		SetProviderKey(wechatOAuthProviderKey).
		SetProviderSubject(subject).
		SetResolvedEmail("wechat-" + subject + "@wechat-connect.invalid").
		SetBrowserSessionKey(browser).
		SetUpstreamIdentityClaims(map[string]any{"username": "wechat_" + subject}).
		SetLocalFlowState(state).
		SetExpiresAt(time.Now().UTC().Add(10 * time.Minute)).
		Save(context.Background())
	require.NoError(t, err)
	return session
}

func quickRegister(t *testing.T, handler *AuthHandler, session *dbent.PendingAuthSession, browser, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/wechat/quick-register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: oauthPendingSessionCookieName, Value: encodeCookieValue(session.SessionToken)})
	req.AddCookie(&http.Cookie{Name: oauthPendingBrowserCookieName, Value: encodeCookieValue(browser)})
	c.Request = req
	handler.QuickWeChatOAuthRegistration(c)
	return recorder
}

func TestQuickWeChatOAuthRegistrationRecordsSignupAttribution(t *testing.T) {
	handler, client := newOAuthPendingFlowTestHandler(t, false)
	repo := &wechatAttributionRepo{}
	handler.analytics = service.NewAnalyticsService(repo)
	fingerprints := &wechatFingerprintRepo{}
	handler.authService.SetSignupTrialGuard(service.NewSignupTrialGuard(fingerprints, nil, nil))

	// 网页授权: the browser sends visitor_id + attribution with quick-register.
	session := createWeChatChoiceSession(t, client, "quick-attr-1", "browser-attr-1", "subject-attr-1", nil)
	rec := quickRegister(t, handler, session, "browser-attr-1",
		`{"visitor_id":"v_web1234567890abc","attribution":{"source":"xhs","campaign":"oct","landing":"/learn"},"aff_code":"FRIEND1"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	identity, err := client.AuthIdentity.Query().Where(authidentity.ProviderSubjectEQ("subject-attr-1")).Only(context.Background())
	require.NoError(t, err)

	repo.mu.Lock()
	require.Len(t, repo.saved, 1)
	got := repo.saved[0]
	repo.mu.Unlock()
	require.Equal(t, identity.UserID, got.UserID)
	require.Equal(t, "v_web1234567890abc", got.VisitorID)
	require.Equal(t, "xhs", got.Source)
	require.Equal(t, "oct", got.Campaign)
	require.Equal(t, "FRIEND1", got.AffCode)
	fingerprints.mu.Lock()
	require.Len(t, fingerprints.saved, 1, "the sign-up trial guard saw the sign-up")
	require.Equal(t, "v_web1234567890abc", fingerprints.saved[0].VisitorID)
	fingerprints.mu.Unlock()

	// QR / code login: the data saved with the pending session is used when the request has none.
	session2 := createWeChatChoiceSession(t, client, "quick-attr-2", "browser-attr-2", "subject-attr-2", map[string]any{
		oauthSignupVisitorIDStateKey:   "v_qr12345678901234",
		oauthSignupAttributionStateKey: map[string]any{"source": "douyin"},
	})
	rec = quickRegister(t, handler, session2, "browser-attr-2", `{}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	repo.mu.Lock()
	require.Len(t, repo.saved, 2)
	got = repo.saved[1]
	repo.mu.Unlock()
	require.Equal(t, "v_qr12345678901234", got.VisitorID)
	require.Equal(t, "douyin", got.Source)
	fingerprints.mu.Lock()
	require.Len(t, fingerprints.saved, 2)
	require.Equal(t, "v_qr12345678901234", fingerprints.saved[1].VisitorID)
	fingerprints.mu.Unlock()
}
