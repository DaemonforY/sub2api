//go:build unit

package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/stretchr/testify/require"
)

const (
	wechatMPTestAppID  = "wx1234567890abcdef"
	wechatMPTestToken  = "testtoken"
	wechatMPTestAESKey = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
)

type wechatMPTestRepo struct {
	mu     sync.Mutex
	values map[string]string
}

func (r *wechatMPTestRepo) Get(_ context.Context, key string) (*Setting, error) {
	v, err := r.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: v}, nil
}

func (r *wechatMPTestRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (r *wechatMPTestRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = value
	return nil
}

func (r *wechatMPTestRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := r.values[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (r *wechatMPTestRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range settings {
		r.values[k] = v
	}
	return nil
}

func (r *wechatMPTestRepo) GetAll(_ context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]string{}
	for k, v := range r.values {
		out[k] = v
	}
	return out, nil
}

func (r *wechatMPTestRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

// fakeWeChatAPI is an httptest stand-in for api.weixin.qq.com.
type fakeWeChatAPI struct {
	server        *httptest.Server
	tokenCalls    atomic.Int32
	forceRefresh  []bool
	sceneStrs     []string
	userInfoCalls atomic.Int32
	mu            sync.Mutex
	// rejectToken makes user/info answer 40001 for this access token.
	rejectToken string
	tokenErr    int
	tokenDelay  time.Duration
}

func newFakeWeChatAPI(t *testing.T) *fakeWeChatAPI {
	f := &fakeWeChatAPI{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			var body struct {
				AppID        string `json:"appid"`
				ForceRefresh bool   `json:"force_refresh"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.mu.Lock()
			f.forceRefresh = append(f.forceRefresh, body.ForceRefresh)
			tokenErr, delay := f.tokenErr, f.tokenDelay
			f.mu.Unlock()
			time.Sleep(delay)
			if tokenErr != 0 {
				_, _ = w.Write([]byte(`{"errcode":` + strconv.Itoa(tokenErr) + `,"errmsg":"invalid ip"}`))
				return
			}
			n := f.tokenCalls.Add(1)
			_, _ = w.Write([]byte(`{"access_token":"tok-` + strconv.Itoa(int(n)) + `","expires_in":7200}`))
		case "/cgi-bin/qrcode/create":
			var body struct {
				ActionInfo struct {
					Scene struct {
						SceneStr string `json:"scene_str"`
					} `json:"scene"`
				} `json:"action_info"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			f.mu.Lock()
			f.sceneStrs = append(f.sceneStrs, body.ActionInfo.Scene.SceneStr)
			f.mu.Unlock()
			_, _ = w.Write([]byte(`{"ticket":"gQH+ticket/=","expire_seconds":300,"url":"http://weixin.qq.com/q/x"}`))
		case "/cgi-bin/user/info":
			f.userInfoCalls.Add(1)
			f.mu.Lock()
			reject := f.rejectToken
			f.mu.Unlock()
			if reject != "" && r.URL.Query().Get("access_token") == reject {
				_, _ = w.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential"}`))
				return
			}
			_, _ = w.Write([]byte(`{"subscribe":1,"openid":"` + r.URL.Query().Get("openid") + `","unionid":"union-1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func wechatMPTestSettings() map[string]string {
	return map[string]string{
		SettingKeyWeChatConnectEnabled:     "true",
		SettingKeyWeChatConnectMPAppID:     wechatMPTestAppID,
		SettingKeyWeChatConnectMPAppSecret: "mp-secret",
		SettingKeyWeChatConnectMPEnabled:   "true",
		SettingKeyWeChatConnectMode:        "mp",
		SettingKeyWeChatMPMessageEnabled:   "true",
		SettingKeyWeChatMPMessageToken:     wechatMPTestToken,
		SettingKeyWeChatMPMessageAESKey:    wechatMPTestAESKey,
		SettingKeyWeChatQRLoginEnabled:     "true",
		SettingKeyWeChatCodeLoginEnabled:   "true",
		SettingKeyFrontendURL:              "https://hivegpt.cn",
	}
}

// memWeChatMPStore is an in-memory WeChatMPLoginStore with a controllable clock.
type memWeChatMPStore struct {
	mu    sync.Mutex
	now   time.Time
	items map[string]*memWeChatMPItem
}

type memWeChatMPItem struct {
	str  string
	hash map[string]string
	exp  time.Time
}

func newMemWeChatMPStore() *memWeChatMPStore {
	return &memWeChatMPStore{now: time.Unix(1_800_000_000, 0), items: map[string]*memWeChatMPItem{}}
}

func (m *memWeChatMPStore) live(key string) *memWeChatMPItem {
	it, ok := m.items[key]
	if !ok {
		return nil
	}
	if !it.exp.IsZero() && !m.now.Before(it.exp) {
		delete(m.items, key)
		return nil
	}
	return it
}

func (m *memWeChatMPStore) expiry(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return m.now.Add(ttl)
}

func (m *memWeChatMPStore) advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
}

func (m *memWeChatMPStore) peek(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it := m.live(key)
	if it == nil {
		return "", false
	}
	return it.str, true
}

func (m *memWeChatMPStore) ttl(key string) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	it := m.live(key)
	if it == nil || it.exp.IsZero() {
		return 0
	}
	return it.exp.Sub(m.now)
}

func (m *memWeChatMPStore) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it := m.live(key)
	if it == nil {
		it = &memWeChatMPItem{str: "0", exp: m.expiry(ttl)}
		m.items[key] = it
	}
	n, _ := strconv.ParseInt(it.str, 10, 64)
	n++
	it.str = strconv.FormatInt(n, 10)
	return n, nil
}

func (m *memWeChatMPStore) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := m.peek(key)
	return v, ok, nil
}

func (m *memWeChatMPStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = &memWeChatMPItem{str: value, exp: m.expiry(ttl)}
	return nil
}

func (m *memWeChatMPStore) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.live(key) != nil {
		return false, nil
	}
	m.items[key] = &memWeChatMPItem{str: value, exp: m.expiry(ttl)}
	return true, nil
}

func (m *memWeChatMPStore) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.items, k)
	}
	return nil
}

func (m *memWeChatMPStore) HSetWithTTL(_ context.Context, key string, fields map[string]string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := map[string]string{}
	for k, v := range fields {
		h[k] = v
	}
	m.items[key] = &memWeChatMPItem{hash: h, exp: m.expiry(ttl)}
	return nil
}

func (m *memWeChatMPStore) HGetAll(_ context.Context, key string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]string{}
	if it := m.live(key); it != nil {
		for k, v := range it.hash {
			out[k] = v
		}
	}
	return out, nil
}

func (m *memWeChatMPStore) ClaimHashField(_ context.Context, key, field, value string, minTTL time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it := m.live(key)
	if it == nil {
		return -1, nil
	}
	if cur, ok := it.hash[field]; ok {
		if cur == value {
			return 1, nil
		}
		return 0, nil
	}
	it.hash[field] = value
	if it.exp.Sub(m.now) < minTTL {
		it.exp = m.now.Add(minTTL)
	}
	return 1, nil
}

func (m *memWeChatMPStore) TakeHash(_ context.Context, key, requiredField, markerKey string, marker map[string]string, markerTTL time.Duration) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it := m.live(key)
	if it == nil {
		return map[string]string{}, nil
	}
	if _, ok := it.hash[requiredField]; !ok {
		return map[string]string{}, nil
	}
	delete(m.items, key)
	h := map[string]string{}
	for k, v := range marker {
		h[k] = v
	}
	m.items[markerKey] = &memWeChatMPItem{hash: h, exp: m.expiry(markerTTL)}
	out := map[string]string{}
	for k, v := range it.hash {
		out[k] = v
	}
	return out, nil
}

const wechatMPTestNow = int64(1_800_000_000)

func newWeChatMPTestService(t *testing.T, extra map[string]string) (*WeChatMPLoginService, *memWeChatMPStore, *fakeWeChatAPI, *wechatMPTestRepo) {
	t.Helper()
	values := wechatMPTestSettings()
	for k, v := range extra {
		values[k] = v
	}
	repo := &wechatMPTestRepo{values: values}
	settingSvc := NewSettingService(repo, &config.Config{})
	store := newMemWeChatMPStore()
	store.now = time.Unix(wechatMPTestNow, 0)
	api := newFakeWeChatAPI(t)
	svc := NewWeChatMPLoginService(settingSvc, store)
	svc.SetAPIBaseURL(api.server.URL)
	svc.SetClock(func() time.Time { return time.Unix(wechatMPTestNow, 0) })
	return svc, store, api, repo
}

var wechatMPTestNonce atomic.Int64

func nextNonce() string { return "nonce" + strconv.FormatInt(wechatMPTestNonce.Add(1), 10) }

func plainQuery(ts int64, nonce string, extra url.Values) string {
	q := url.Values{}
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("nonce", nonce)
	q.Set("signature", WeChatMPSignature(wechatMPTestToken, strconv.FormatInt(ts, 10), nonce))
	q.Set("openid", "openid-a")
	for k, v := range extra {
		q[k] = v
	}
	return q.Encode()
}

// encryptedRequest wraps inner like WeChat does in 安全模式.
func encryptedRequest(t *testing.T, inner string, ts int64, nonce string) WeChatMPMessageRequest {
	t.Helper()
	crypter, err := NewWeChatMPCrypter(wechatMPTestAESKey, wechatMPTestAppID, rand.Reader)
	require.NoError(t, err)
	encrypted, err := crypter.Encrypt([]byte(inner))
	require.NoError(t, err)
	body := "<xml><ToUserName><![CDATA[gh_test]]></ToUserName><Encrypt><![CDATA[" + encrypted + "]]></Encrypt></xml>"
	query := plainQuery(ts, nonce, url.Values{
		"encrypt_type":  {"aes"},
		"msg_signature": {WeChatMPSignature(wechatMPTestToken, strconv.FormatInt(ts, 10), nonce, encrypted)},
	})
	return WeChatMPMessageRequest{Method: http.MethodPost, RawQuery: query, ContentType: "text/xml", Body: []byte(body)}
}

func eventXML(from, event, key string) string {
	return "<xml><ToUserName><![CDATA[gh_test]]></ToUserName><FromUserName><![CDATA[" + from + "]]></FromUserName>" +
		"<CreateTime>1800000000</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[" + event + "]]></Event>" +
		"<EventKey><![CDATA[" + key + "]]></EventKey></xml>"
}

func textXML(from, content string, msgID int) string {
	return "<xml><ToUserName><![CDATA[gh_test]]></ToUserName><FromUserName><![CDATA[" + from + "]]></FromUserName>" +
		"<CreateTime>1800000000</CreateTime><MsgType><![CDATA[text]]></MsgType><Content><![CDATA[" + content + "]]></Content>" +
		"<MsgId>" + strconv.Itoa(msgID) + "</MsgId></xml>"
}

type replyXML struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	MsgType      string `xml:"MsgType"`
	Content      string `xml:"Content"`
	Encrypt      string `xml:"Encrypt"`
	MsgSignature string `xml:"MsgSignature"`
	TimeStamp    string `xml:"TimeStamp"`
	Nonce        string `xml:"Nonce"`
}

func handle(t *testing.T, svc *WeChatMPLoginService, req WeChatMPMessageRequest) *WeChatMPMessageResponse {
	t.Helper()
	resp, err := svc.HandleMessage(context.Background(), req)
	require.NoError(t, err)
	return resp
}

// decryptedReply returns the text of an encrypted reply, "" for "success".
func decryptedReply(t *testing.T, resp *WeChatMPMessageResponse) string {
	t.Helper()
	if string(resp.Body) == "success" {
		return ""
	}
	var outer replyXML
	require.NoError(t, xml.Unmarshal(resp.Body, &outer), string(resp.Body))
	require.NotEmpty(t, outer.Encrypt, "replies to 安全模式 messages are encrypted")
	require.True(t, WeChatMPSignatureValid(outer.MsgSignature, wechatMPTestToken, outer.TimeStamp, outer.Nonce, outer.Encrypt))
	crypter, err := NewWeChatMPCrypter(wechatMPTestAESKey, wechatMPTestAppID, rand.Reader)
	require.NoError(t, err)
	inner, err := crypter.Decrypt(outer.Encrypt)
	require.NoError(t, err)
	var r replyXML
	require.NoError(t, xml.Unmarshal(inner, &r))
	return r.Content
}

// send posts inner as a fresh 安全模式 message and returns the reply text.
func send(t *testing.T, svc *WeChatMPLoginService, inner string) string {
	t.Helper()
	return decryptedReply(t, handle(t, svc, encryptedRequest(t, inner, wechatMPTestNow, nextNonce())))
}

func createTestQRTicket(t *testing.T, svc *WeChatMPLoginService, api *fakeWeChatAPI) string {
	t.Helper()
	created, err := svc.CreateQRLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "browser-1", Redirect: "/dashboard", Intent: "login"})
	require.NoError(t, err)
	require.Equal(t, 300, created.ExpiresIn)
	require.Equal(t, 2, created.PollInterval)
	require.Equal(t, "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket="+url.QueryEscape("gQH+ticket/="), created.QRURL)
	api.mu.Lock()
	scene := api.sceneStrs[len(api.sceneStrs)-1]
	api.mu.Unlock()
	require.Equal(t, WeChatQRLoginScenePrefix+created.TicketID, scene)
	return created.TicketID
}

func createTestCodeTicket(t *testing.T, svc *WeChatMPLoginService) *WeChatCodeLoginCreated {
	t.Helper()
	created, err := svc.CreateCodeLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "browser-c", Redirect: "/canvas"})
	require.NoError(t, err)
	require.Len(t, created.Code, 6)
	require.Equal(t, 300, created.ExpiresIn)
	require.Equal(t, 2, created.PollInterval)
	return created
}

func ticketOpenID(t *testing.T, svc *WeChatMPLoginService, id string) string {
	t.Helper()
	ticket, err := svc.GetLoginTicket(context.Background(), id)
	require.NoError(t, err)
	if ticket == nil {
		return "<gone>"
	}
	return ticket.OpenID
}

// ---------- signature / crypto ----------

func TestWeChatMPSignatureKnownVector(t *testing.T) {
	// sha1("1409304348nonce123testtoken"), computed independently with Python's hashlib.
	require.Equal(t, "5d56cf11e749eccef30c16ecdfa4f5dccf59a22f", WeChatMPSignature("testtoken", "1409304348", "nonce123"))
	require.Equal(t, "e316b92b8af1e67bd17dcc9156fb594fa5cf7211", WeChatMPSignature("testtoken", "1409304348", "nonce123", "ENCRYPTED"))
	require.True(t, WeChatMPSignatureValid("5D56CF11E749ECCEF30C16ECDFA4F5DCCF59A22F", "testtoken", "1409304348", "nonce123"))
	require.False(t, WeChatMPSignatureValid("5d56cf11e749eccef30c16ecdfa4f5dccf59a22e", "testtoken", "1409304348", "nonce123"))
	require.False(t, WeChatMPSignatureValid("", "testtoken", "1409304348", "nonce123"))
}

func TestWeChatMPCrypterKnownVector(t *testing.T) {
	// Built independently: Python packs random16 + len + msg + appid with PKCS#7 (block 32) and
	// `openssl enc -aes-256-cbc -nopad` encrypts it with key = base64(EncodingAESKey+"="), IV = key[:16].
	const expected = "Q3stYC6hdFzMh9T8HCvyDMnsu8c340VXOKAu7L4L/b1P2o/5A49YbJ2j9ii79bXaOrdISKwixZyWnwUN43wSo5tkRn7w+juueyWa26YiV7asB+0fMLsnEb+alk7/LEr4"
	msg := []byte("<xml><Content><![CDATA[hi]]></Content></xml>")

	crypter, err := NewWeChatMPCrypter(wechatMPTestAESKey, wechatMPTestAppID, bytes.NewReader([]byte("0123456789abcdef")))
	require.NoError(t, err)
	encrypted, err := crypter.Encrypt(msg)
	require.NoError(t, err)
	require.Equal(t, expected, encrypted)

	plain, err := crypter.Decrypt(expected)
	require.NoError(t, err)
	require.Equal(t, msg, plain)

	other, err := NewWeChatMPCrypter(wechatMPTestAESKey, "wxanotherapp00000", rand.Reader)
	require.NoError(t, err)
	_, err = other.Decrypt(expected)
	require.Error(t, err, "a message for another app id is rejected")
}

func TestWeChatMPCrypterRoundTrip(t *testing.T) {
	crypter, err := NewWeChatMPCrypter(wechatMPTestAESKey, wechatMPTestAppID, rand.Reader)
	require.NoError(t, err)
	for _, msg := range []string{"", "a", strings.Repeat("长消息", 100), "<xml>exactly thirty two bytes!</xml>"} {
		encrypted, err := crypter.Encrypt([]byte(msg))
		require.NoError(t, err)
		plain, err := crypter.Decrypt(encrypted)
		require.NoError(t, err)
		require.Equal(t, msg, string(plain))
	}
	_, err = NewWeChatMPCrypter("too-short", wechatMPTestAppID, rand.Reader)
	require.Error(t, err)
	require.Error(t, ValidateWeChatMPAESKey("abc"))
	require.NoError(t, ValidateWeChatMPAESKey(wechatMPTestAESKey))
}

// ---------- message authenticity ----------

func TestWeChatMPVerifyServerURL(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, nil)
	q, _ := url.ParseQuery(plainQuery(wechatMPTestNow, "n1", url.Values{"echostr": {"echo-123"}}))
	echo, err := svc.VerifyServerURL(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, "echo-123", echo)

	q.Set("signature", "bad")
	_, err = svc.VerifyServerURL(context.Background(), q)
	require.ErrorIs(t, err, ErrWeChatMPBadSignature)

	_, err = svc.HandleMessage(context.Background(), WeChatMPMessageRequest{RawQuery: q.Encode(), Body: []byte(textXML("o", "登录", 1))})
	require.ErrorIs(t, err, ErrWeChatMPBadSignature)

	req := encryptedRequest(t, textXML("o", "登录", 1), wechatMPTestNow, "n2")
	q, _ = url.ParseQuery(req.RawQuery)
	q.Set("msg_signature", "bad")
	req.RawQuery = q.Encode()
	_, err = svc.HandleMessage(context.Background(), req)
	require.ErrorIs(t, err, ErrWeChatMPBadSignature)
}

func TestWeChatMPForgedPlainMessageIgnored(t *testing.T) {
	svc, _, api, _ := newWeChatMPTestService(t, nil)
	code := createTestCodeTicket(t, svc)
	qrTicket := createTestQRTicket(t, svc, api)

	// Correctly signed (the token alone) but not encrypted while an EncodingAESKey is set.
	for i, body := range []string{
		textXML("openid-forger", code.Code, 1),
		eventXML("openid-forger", "SCAN", WeChatQRLoginScenePrefix+qrTicket),
	} {
		resp := handle(t, svc, WeChatMPMessageRequest{Method: http.MethodPost, RawQuery: plainQuery(wechatMPTestNow, "forged"+strconv.Itoa(i), nil), Body: []byte(body)})
		require.Equal(t, "success", string(resp.Body))
	}
	require.Empty(t, ticketOpenID(t, svc, code.TicketID))
	require.Empty(t, ticketOpenID(t, svc, qrTicket))

	// Without an AES key there is no QR / code login at all, and plain messages never log anyone in.
	svc2, _, _, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatMPMessageAESKey: ""})
	_, err := svc2.CreateCodeLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "b"})
	require.ErrorIs(t, err, ErrWeChatCodeLoginDisabled)
	_, err = svc2.CreateQRLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "b"})
	require.ErrorIs(t, err, ErrWeChatQRLoginDisabled)
	resp := handle(t, svc2, WeChatMPMessageRequest{Method: http.MethodPost, RawQuery: plainQuery(wechatMPTestNow, "plain1", nil), Body: []byte(textXML("openid-a", code.Code, 2))})
	require.Equal(t, "success", string(resp.Body))
	// Generic replies still work in plain mode.
	resp = handle(t, svc2, WeChatMPMessageRequest{Method: http.MethodPost, RawQuery: plainQuery(wechatMPTestNow, "plain2", nil), Body: []byte(eventXML("openid-a", "subscribe", ""))})
	require.Contains(t, string(resp.Body), "https://hivegpt.cn")
}

func TestWeChatMPStaleTimestampRejected(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, nil)
	code := createTestCodeTicket(t, svc)

	resp := handle(t, svc, encryptedRequest(t, textXML("openid-a", code.Code, 1), wechatMPTestNow-301, "stale"))
	require.Equal(t, "success", string(resp.Body))
	resp = handle(t, svc, encryptedRequest(t, textXML("openid-a", code.Code, 2), wechatMPTestNow+301, "future"))
	require.Equal(t, "success", string(resp.Body))
	require.Empty(t, ticketOpenID(t, svc, code.TicketID))

	resp = handle(t, svc, encryptedRequest(t, textXML("openid-a", code.Code, 3), wechatMPTestNow-299, "fresh"))
	require.Equal(t, wantScanOKReply(), decryptedReply(t, resp))
	require.Equal(t, "openid-a", ticketOpenID(t, svc, code.TicketID))
}

func TestWeChatMPReplayedNonceRejectedRetryGetsSameReply(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, nil)
	code := createTestCodeTicket(t, svc)

	req := encryptedRequest(t, textXML("openid-a", code.Code, 10), wechatMPTestNow, "same-nonce")
	first := decryptedReply(t, handle(t, svc, req))
	require.Equal(t, wantScanOKReply(), first)

	// WeChat's own retry (identical request): the same reply, nothing processed again.
	require.Equal(t, first, decryptedReply(t, handle(t, svc, req)))
	// A retry of the same MsgId that came with a new nonce: also the cached reply.
	require.Equal(t, first, decryptedReply(t, handle(t, svc, encryptedRequest(t, textXML("openid-a", code.Code, 10), wechatMPTestNow, "other-nonce"))))

	// A different message replayed under a used timestamp+nonce is dropped.
	code2 := createTestCodeTicket(t, svc)
	resp := handle(t, svc, encryptedRequest(t, textXML("openid-b", code2.Code, 11), wechatMPTestNow, "same-nonce"))
	require.Equal(t, "success", string(resp.Body))
	require.Empty(t, ticketOpenID(t, svc, code2.TicketID))
}

// ---------- QR and code binding ----------

func TestWeChatMPSubscribeAndScanBindOpenIDToTicket(t *testing.T) {
	svc, mr, api, _ := newWeChatMPTestService(t, nil)

	// Not yet following: subscribe with qrscene_ + our scene.
	ticketID := createTestQRTicket(t, svc, api)
	require.Equal(t, wantScanOKReply(), send(t, svc, eventXML("openid-new", "subscribe", "qrscene_"+WeChatQRLoginScenePrefix+ticketID)))
	require.Equal(t, "openid-new", ticketOpenID(t, svc, ticketID))

	// Someone else scanning the same QR afterwards does not take it over.
	require.Contains(t, send(t, svc, eventXML("openid-other", "SCAN", WeChatQRLoginScenePrefix+ticketID)), "二维码已过期")
	require.Equal(t, "openid-new", ticketOpenID(t, svc, ticketID))

	// Already following: SCAN with the bare scene.
	ticket2 := createTestQRTicket(t, svc, api)
	require.Equal(t, wantScanOKReply(), send(t, svc, eventXML("openid-follower", "SCAN", WeChatQRLoginScenePrefix+ticket2)))
	require.Equal(t, "openid-follower", ticketOpenID(t, svc, ticket2))

	// Tickets live 5 minutes; a scanned one at least 2 more minutes.
	ticket3 := createTestQRTicket(t, svc, api)
	mr.advance(4 * time.Minute)
	_, err := svc.BindLoginTicket(context.Background(), ticket3, "openid-late")
	require.NoError(t, err)
	mr.advance(90 * time.Second)
	require.Equal(t, "openid-late", ticketOpenID(t, svc, ticket3))
	ticket4 := createTestQRTicket(t, svc, api)
	mr.advance(301 * time.Second)
	require.Equal(t, "<gone>", ticketOpenID(t, svc, ticket4))

	require.Contains(t, send(t, svc, eventXML("openid-x", "SCAN", WeChatQRLoginScenePrefix+strings.Repeat("a", 32))), "二维码已过期")
}

func TestWeChatMPCodeLoginBindsTicket(t *testing.T) {
	svc, mr, _, _ := newWeChatMPTestService(t, nil)

	code := createTestCodeTicket(t, svc)
	owner, ok := mr.peek(weChatLoginCodeKey(code.Code))
	require.True(t, ok)
	require.Equal(t, code.TicketID, owner)

	require.Equal(t, wantScanOKReply(), send(t, svc, textXML("openid-c", "登录 "+code.Code, 1)))
	require.Equal(t, "openid-c", ticketOpenID(t, svc, code.TicketID))
	_, ok = mr.peek(weChatLoginCodeKey(code.Code))
	require.False(t, ok, "a code works once")
	require.Equal(t, weChatLoginInvalidCodeReply, send(t, svc, textXML("openid-other", code.Code, 2)))
	require.Equal(t, "openid-c", ticketOpenID(t, svc, code.TicketID))

	cfg := WeChatMPConfig{LoginKeywords: []string{"登录", "login"}}
	for in, want := range map[string]string{
		"123456": "123456", " 123456 ": "123456", "登录123456": "123456", "登录 123456": "123456",
		"登录：123456": "123456", "LOGIN 123456": "123456", "1234567": "", "登录 12345": "", "你好 123456": "",
	} {
		require.Equal(t, want, extractWeChatLoginCode(cfg, in), in)
	}
}

func TestWeChatMPKeywordGetsInstructions(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, nil)
	reply := send(t, svc, textXML("openid-a", "  登录 ", 1))
	require.Contains(t, reply, "微信登录")
	require.Contains(t, reply, "6 位验证码")
	require.Contains(t, send(t, svc, textXML("openid-a", "LOGIN", 2)), "微信登录")
	// Other text without a default reply: "success".
	require.Empty(t, send(t, svc, textXML("openid-a", "你好", 3)))
	// A subscribe with no scene gets the welcome.
	welcome := send(t, svc, eventXML("openid-a", "subscribe", ""))
	require.Contains(t, welcome, "https://hivegpt.cn")
}

func TestWeChatMPWrongCodeLimit(t *testing.T) {
	svc, mr, _, _ := newWeChatMPTestService(t, nil)
	code := createTestCodeTicket(t, svc)
	wrong := "000000"
	if code.Code == wrong {
		wrong = "000001"
	}
	for i := 0; i < 5; i++ {
		require.Equal(t, weChatLoginInvalidCodeReply, send(t, svc, textXML("openid-guess", wrong, 100+i)))
	}
	// Then silence, and no more matches even for a live code.
	require.Empty(t, send(t, svc, textXML("openid-guess", wrong, 200)))
	require.Empty(t, send(t, svc, textXML("openid-guess", code.Code, 201)))
	require.Empty(t, ticketOpenID(t, svc, code.TicketID))
	// Others are not affected.
	require.Equal(t, wantScanOKReply(), send(t, svc, textXML("openid-real", code.Code, 202)))

	tries, ok := mr.peek(weChatLoginBadCodeKey("openid-guess"))
	require.True(t, ok)
	require.Equal(t, "7", tries, "every try is counted up front")
	require.Equal(t, weChatLoginBadCodeWindow, mr.ttl(weChatLoginBadCodeKey("openid-guess")), "TTL set on the first increment")

	mr.advance(11 * time.Minute)
	require.Equal(t, weChatLoginInvalidCodeReply, send(t, svc, textXML("openid-guess", wrong, 300)))
}

func TestWeChatMPCodeUniqueAmongLiveCodes(t *testing.T) {
	svc, mr, _, _ := newWeChatMPTestService(t, nil)
	ctx := context.Background()
	require.NoError(t, mr.Set(ctx, weChatLoginCodeKey("000000"), "someone-else", time.Minute))
	// 16 bytes for the ticket ID, then six 0 digits (taken), then six 1 digits.
	svc.random = io.MultiReader(bytes.NewReader(bytes.Repeat([]byte{0xab}, 16)), bytes.NewReader(make([]byte, 6)), bytes.NewReader(bytes.Repeat([]byte{1}, 6)))
	created, err := svc.CreateCodeLogin(ctx, WeChatLoginTicket{BrowserSessionKey: "b"})
	require.NoError(t, err)
	require.Equal(t, "111111", created.Code)
	owner, _ := mr.peek(weChatLoginCodeKey("000000"))
	require.Equal(t, "someone-else", owner)
	require.Equal(t, weChatLoginTicketTTL, mr.ttl(weChatLoginCodeKey("111111")))
}

func TestWeChatMPClaimLoginTicketOnce(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, nil)
	ctx := context.Background()
	code := createTestCodeTicket(t, svc)

	claimed, err := svc.ClaimLoginTicket(ctx, code.TicketID, "browser-c")
	require.NoError(t, err)
	require.Nil(t, claimed, "nothing to take before the openid arrives")

	bound, err := svc.BindLoginTicket(ctx, code.TicketID, "openid-c")
	require.NoError(t, err)
	require.True(t, bound)
	claimed, err = svc.ClaimLoginTicket(ctx, code.TicketID, "browser-c")
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, "openid-c", claimed.OpenID)
	require.Equal(t, "/canvas", claimed.Redirect)

	again, err := svc.ClaimLoginTicket(ctx, code.TicketID, "browser-c")
	require.NoError(t, err)
	require.Nil(t, again, "a ticket completes once")
	result, err := svc.GetLoginResult(ctx, code.TicketID)
	require.NoError(t, err)
	require.Equal(t, WeChatLoginStatusScanned, result.Status)
	require.Equal(t, "browser-c", result.BrowserSessionKey)

	svc.SetLoginResult(ctx, code.TicketID, WeChatLoginResult{Status: WeChatLoginStatusError, Message: "出错了（failed）", BrowserSessionKey: "browser-c"})
	result, err = svc.GetLoginResult(ctx, code.TicketID)
	require.NoError(t, err)
	require.Equal(t, WeChatLoginStatusError, result.Status)
	require.Equal(t, "出错了（failed）", result.Message)
}

func TestWeChatMPLoginDisabled(t *testing.T) {
	svc, _, _, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatQRLoginEnabled: "false", SettingKeyWeChatCodeLoginEnabled: "false"})
	_, err := svc.CreateQRLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "b"})
	require.ErrorIs(t, err, ErrWeChatQRLoginDisabled)
	_, err = svc.CreateCodeLogin(context.Background(), WeChatLoginTicket{BrowserSessionKey: "b"})
	require.ErrorIs(t, err, ErrWeChatCodeLoginDisabled)
	// And a 6-digit text is not ours then.
	require.Empty(t, send(t, svc, textXML("openid-a", "123456", 5)))
}

// ---------- forwarding ----------

type forwardRecorder struct {
	mu          sync.Mutex
	calls       int
	method      string
	rawQuery    string
	body        []byte
	contentType string
}

func newForwardServer(t *testing.T, delay time.Duration, status int) (*httptest.Server, *forwardRecorder) {
	rec := &forwardRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.calls++
		rec.method = r.Method
		rec.rawQuery = r.URL.RawQuery
		rec.body = body
		rec.contentType = r.Header.Get("Content-Type")
		rec.mu.Unlock()
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("<xml><Content><![CDATA[from the other site]]></Content></xml>"))
	}))
	t.Cleanup(srv.Close)
	return srv, rec
}

func (r *forwardRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func TestWeChatMPForwardsUnhandledMessagesVerbatim(t *testing.T) {
	fwd, rec := newForwardServer(t, 0, http.StatusOK)
	svc, _, api, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatMPMessageForwardURL: fwd.URL + "/wx"})

	req := encryptedRequest(t, textXML("openid-a", "你好", 100), wechatMPTestNow, nextNonce())
	req.ContentType = "text/xml; charset=utf-8"
	resp := handle(t, svc, req)
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "application/xml", resp.ContentType)
	require.Equal(t, "<xml><Content><![CDATA[from the other site]]></Content></xml>", string(resp.Body))
	rec.mu.Lock()
	require.Equal(t, http.MethodPost, rec.method)
	require.Equal(t, req.RawQuery, rec.rawQuery)
	require.Equal(t, req.Body, rec.body)
	require.Equal(t, "text/xml; charset=utf-8", rec.contentType)
	rec.mu.Unlock()

	// A subscribe without a scene, another site's scene and other events go there too.
	for i, inner := range []string{
		eventXML("openid-a", "subscribe", ""),
		eventXML("openid-a", "subscribe", "qrscene_login_123"),
		eventXML("openid-a", "CLICK", "MENU_1"),
	} {
		resp := handle(t, svc, encryptedRequest(t, inner, wechatMPTestNow, "fwd"+strconv.Itoa(i)))
		require.Contains(t, string(resp.Body), "from the other site")
	}
	forwarded := rec.count()
	require.Equal(t, 4, forwarded)

	// Our own keyword, codes and scenes are answered here and never forwarded.
	require.Contains(t, send(t, svc, textXML("openid-a", "登录", 101)), "微信登录")
	code := createTestCodeTicket(t, svc)
	require.Equal(t, wantScanOKReply(), send(t, svc, textXML("openid-a", code.Code, 102)))
	ticketID := createTestQRTicket(t, svc, api)
	require.Equal(t, wantScanOKReply(), send(t, svc, eventXML("openid-a", "subscribe", "qrscene_"+WeChatQRLoginScenePrefix+ticketID)))
	require.Equal(t, forwarded, rec.count())

	// Plain mode (no AES key) forwards plain messages as they are.
	svc2, _, _, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatMPMessageForwardURL: fwd.URL, SettingKeyWeChatMPMessageAESKey: ""})
	resp = handle(t, svc2, WeChatMPMessageRequest{Method: http.MethodPost, RawQuery: plainQuery(wechatMPTestNow, "p1", nil), Body: []byte(textXML("openid-a", "hi", 1))})
	require.Contains(t, string(resp.Body), "from the other site")
}

func TestWeChatMPForwardFailuresAnswerSuccess(t *testing.T) {
	slow, _ := newForwardServer(t, 2*time.Second, http.StatusOK)
	svc, _, _, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatMPMessageForwardURL: slow.URL})
	svc.forwardTimeout = 100 * time.Millisecond
	start := time.Now()
	resp := handle(t, svc, encryptedRequest(t, textXML("openid-a", "随便说点什么", 200), wechatMPTestNow, nextNonce()))
	require.Less(t, time.Since(start), time.Second)
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "success", string(resp.Body))
	require.Equal(t, weChatMPForwardTimeout, NewWeChatMPLoginService(nil, nil).forwardTimeout)

	broken, rec := newForwardServer(t, 0, http.StatusInternalServerError)
	svc2, _, _, _ := newWeChatMPTestService(t, map[string]string{SettingKeyWeChatMPMessageForwardURL: broken.URL})
	resp = handle(t, svc2, encryptedRequest(t, textXML("openid-a", "hi", 201), wechatMPTestNow, nextNonce()))
	require.Equal(t, 1, rec.count())
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "success", string(resp.Body))
}

// ---------- access token ----------

func TestWeChatMPAccessTokenCachedAndRefreshedOn40001(t *testing.T) {
	svc, mr, api, _ := newWeChatMPTestService(t, nil)
	ctx := context.Background()
	api.mu.Lock()
	api.rejectToken = "tok-1"
	api.mu.Unlock()

	unionid, err := svc.FetchUnionID(ctx, "openid-a")
	require.NoError(t, err)
	require.Equal(t, "union-1", unionid)
	require.EqualValues(t, 2, api.tokenCalls.Load(), "40001 forces one refresh")
	api.mu.Lock()
	require.Equal(t, []bool{false, true}, api.forceRefresh)
	api.mu.Unlock()

	cached, ok := mr.peek(weChatMPAccessTokenKey(wechatMPTestAppID))
	require.True(t, ok)
	require.Equal(t, "tok-2", cached)
	require.Equal(t, 7200*time.Second-5*time.Minute, mr.ttl(weChatMPAccessTokenKey(wechatMPTestAppID)), "cached until 5 minutes before expiry")

	_, err = svc.FetchUnionID(ctx, "openid-b")
	require.NoError(t, err)
	require.EqualValues(t, 2, api.tokenCalls.Load(), "the cached token is reused")
	require.EqualValues(t, 3, api.userInfoCalls.Load())
}

func TestWeChatMPAccessTokenRefreshOnlyWhenCacheHoldsTheFailedToken(t *testing.T) {
	svc, mr, api, _ := newWeChatMPTestService(t, nil)
	ctx := context.Background()
	cfg, err := svc.Config(ctx)
	require.NoError(t, err)

	// Another request already replaced the token that failed for us: use the new one, no refresh.
	require.NoError(t, mr.Set(ctx, weChatMPAccessTokenKey(wechatMPTestAppID), "tok-newer", time.Hour))
	token, err := svc.refreshAccessToken(ctx, cfg, "tok-old")
	require.NoError(t, err)
	require.Equal(t, "tok-newer", token)
	require.EqualValues(t, 0, api.tokenCalls.Load())

	// The cache still holds the failed token: force a refresh.
	token, err = svc.refreshAccessToken(ctx, cfg, "tok-newer")
	require.NoError(t, err)
	require.Equal(t, "tok-1", token)
	api.mu.Lock()
	require.Equal(t, []bool{true}, api.forceRefresh)
	api.mu.Unlock()
}

func TestWeChatMPAccessTokenSingleflight(t *testing.T) {
	svc, _, api, _ := newWeChatMPTestService(t, nil)
	api.mu.Lock()
	api.tokenDelay = 100 * time.Millisecond
	api.mu.Unlock()
	cfg, err := svc.Config(context.Background())
	require.NoError(t, err)

	var wg sync.WaitGroup
	tokens := make([]string, 8)
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tok, err := svc.accessToken(context.Background(), cfg)
			require.NoError(t, err)
			tokens[i] = tok
		}(i)
	}
	wg.Wait()
	require.EqualValues(t, 1, api.tokenCalls.Load(), "concurrent misses share one stable_token call")
	for _, tok := range tokens {
		require.Equal(t, "tok-1", tok)
	}
}

func TestWeChatMPAccessTokenIPWhitelistError(t *testing.T) {
	svc, _, api, _ := newWeChatMPTestService(t, nil)
	api.mu.Lock()
	api.tokenErr = 40164
	api.mu.Unlock()
	_, err := svc.FetchUnionID(context.Background(), "openid-a")
	require.ErrorIs(t, err, ErrWeChatMPIPNotWhitelisted)
	require.Contains(t, infraerrors.Message(err), WeChatMPWhitelistIP)
}

// ---------- settings ----------

func TestWeChatDomainVerifyFilesValidationAndCache(t *testing.T) {
	files, err := NormalizeWeChatDomainVerifyFiles([]WeChatDomainVerifyFile{
		{Filename: " MP_verify_AbCd1234.txt ", Content: " AbCd1234 "},
		{Filename: "", Content: ""},
	})
	require.NoError(t, err)
	require.Equal(t, []WeChatDomainVerifyFile{{Filename: "MP_verify_AbCd1234.txt", Content: "AbCd1234"}}, files)

	for _, bad := range []string{"../etc/passwd", "MP_verify_short.txt", "MP_verify_AbCd1234.html", "index.html"} {
		_, err := NormalizeWeChatDomainVerifyFiles([]WeChatDomainVerifyFile{{Filename: bad, Content: "x"}})
		require.Error(t, err, bad)
	}

	_, _, _, repo := newWeChatMPTestService(t, nil)
	repo.values[SettingKeyWeChatDomainVerifyFiles] = marshalWeChatDomainVerifyFiles(files)
	settingSvc := NewSettingService(repo, &config.Config{})
	ctx := context.Background()
	content, ok := settingSvc.GetWeChatDomainVerifyFile(ctx, "MP_verify_AbCd1234.txt")
	require.True(t, ok)
	require.Equal(t, "AbCd1234", content)
	_, ok = settingSvc.GetWeChatDomainVerifyFile(ctx, "MP_verify_Other123.txt")
	require.False(t, ok)

	// Cached: a change in storage shows up only after the cache is dropped (as a settings save does).
	repo.values[SettingKeyWeChatDomainVerifyFiles] = marshalWeChatDomainVerifyFiles([]WeChatDomainVerifyFile{{Filename: "MP_verify_Other123.txt", Content: "Other123"}})
	_, ok = settingSvc.GetWeChatDomainVerifyFile(ctx, "MP_verify_Other123.txt")
	require.False(t, ok)
	settingSvc.invalidateWeChatDomainVerifyFiles()
	content, ok = settingSvc.GetWeChatDomainVerifyFile(ctx, "MP_verify_Other123.txt")
	require.True(t, ok)
	require.Equal(t, "Other123", content)
}

func TestWeChatMPPublicSettings(t *testing.T) {
	_, _, _, repo := newWeChatMPTestService(t, map[string]string{
		SettingKeyWeChatMPAccountName: "HiveGPT",
		SettingKeyWeChatMPOriginalID:  "gh_abc123",
	})
	settingSvc := NewSettingService(repo, &config.Config{})
	pub, err := settingSvc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, pub.WeChatQRLoginEnabled)
	require.True(t, pub.WeChatCodeLoginEnabled)
	require.Equal(t, "HiveGPT", pub.WeChatMPAccountName)
	require.Equal(t, "https://open.weixin.qq.com/qr/code?username=gh_abc123", pub.WeChatMPFollowQRURL)

	repo.values[SettingKeyWeChatMPMessageEnabled] = "false"
	pub, err = settingSvc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, pub.WeChatQRLoginEnabled, "needs the message server")
	require.False(t, pub.WeChatCodeLoginEnabled)

	repo.values[SettingKeyWeChatMPMessageEnabled] = "true"
	repo.values[SettingKeyWeChatMPMessageAESKey] = ""
	pub, err = settingSvc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, pub.WeChatQRLoginEnabled, "needs 安全模式 (an EncodingAESKey)")
	require.False(t, pub.WeChatCodeLoginEnabled)
}

func TestWeChatMPSettingUpdates(t *testing.T) {
	updates := map[string]string{}
	require.NoError(t, writeWeChatMPSettingUpdates(updates, &SystemSettings{
		WeChatMPMessageEnabled:    true,
		WeChatMPMessageAESKey:     wechatMPTestAESKey,
		WeChatMPLoginKeywords:     " 登录，Login, login ,验证码",
		WeChatMPReplyCode:         "在网页上点微信登录拿验证码",
		WeChatMPMessageForwardURL: "https://other.example.com/wx?site=a",
	}, ""))
	_, hasToken := updates[SettingKeyWeChatMPMessageToken]
	require.False(t, hasToken, "an empty token keeps the stored one")
	require.Equal(t, wechatMPTestAESKey, updates[SettingKeyWeChatMPMessageAESKey])
	require.Equal(t, "登录,Login,验证码", updates[SettingKeyWeChatMPLoginKeywords])
	require.Equal(t, "https://other.example.com/wx?site=a", updates[SettingKeyWeChatMPMessageForwardURL])
	require.Equal(t, "[]", updates[SettingKeyWeChatDomainVerifyFiles])

	for _, bad := range []*SystemSettings{
		{WeChatMPMessageAESKey: "short"},
		{WeChatMPMessageToken: "has space"},
		{WeChatMPMessageForwardURL: "ftp://other.example.com"},
		{WeChatDomainVerifyFiles: []WeChatDomainVerifyFile{{Filename: "index.html", Content: "x"}}},
	} {
		require.Error(t, writeWeChatMPSettingUpdates(map[string]string{}, bad, ""))
	}

	// QR / code login need an EncodingAESKey: in the request, or already saved.
	for _, on := range []*SystemSettings{{WeChatQRLoginEnabled: true}, {WeChatCodeLoginEnabled: true}} {
		err := writeWeChatMPSettingUpdates(map[string]string{}, on, "")
		require.ErrorIs(t, err, ErrWeChatLoginNeedsAESKey)
		require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
		require.Contains(t, infraerrors.Message(err), "EncodingAESKey")
		require.NoError(t, writeWeChatMPSettingUpdates(map[string]string{}, on, wechatMPTestAESKey))
	}
	withKey := &SystemSettings{WeChatQRLoginEnabled: true, WeChatCodeLoginEnabled: true, WeChatMPMessageAESKey: wechatMPTestAESKey}
	require.NoError(t, writeWeChatMPSettingUpdates(map[string]string{}, withKey, ""))
}

func wantScanOKReply() string {
	return weChatMPScanOKReply(WeChatMPConfig{ReplyScanOK: defaultWeChatMPReplyScanOK, SiteURL: "https://hivegpt.cn"})
}

func TestWeChatMPScanOKReplyCarriesSyncLink(t *testing.T) {
	const link = `<a href="https://hivegpt.cn/api/v1/auth/oauth/wechat/sync-profile/start?redirect=%2Fauth%2Fwechat%2Fsynced">同步微信昵称和头像</a>`
	require.Equal(t, "已确认登录 HiveGPT，请回到浏览器继续。\n想用微信昵称和头像？"+link+"\n如果不是你本人在登录，请忽略。", wantScanOKReply())

	// A custom template without {sync_link} still gets the link on its own line.
	custom := weChatMPScanOKReply(WeChatMPConfig{ReplyScanOK: "好了，回网页吧。", SiteURL: "https://hivegpt.cn/"})
	require.Equal(t, "好了，回网页吧。\n"+link, custom)

	// Stored copies of earlier defaults count as "unchanged" and move to the new default.
	for _, old := range []string{"", "已确认登录，请回到电脑上的网页继续。", "已确认登录 HiveGPT，请回到浏览器继续。如果不是你本人在登录，请忽略。"} {
		require.Equal(t, defaultWeChatMPReplyScanOK, normalizeWeChatMPReplyScanOK(old))
	}
	require.Equal(t, "自定义", normalizeWeChatMPReplyScanOK(" 自定义 "))

	// What a scan actually answers.
	svc, _, api, repo := newWeChatMPTestService(t, nil)
	repo.values[SettingKeyWeChatMPReplyScanOK] = "已确认登录 HiveGPT，请回到浏览器继续。如果不是你本人在登录，请忽略。"
	ticketID := createTestQRTicket(t, svc, api)
	reply := send(t, svc, eventXML("openid-new", "subscribe", "qrscene_"+WeChatQRLoginScenePrefix+ticketID))
	require.Contains(t, reply, link)
	require.Contains(t, reply, "想用微信昵称和头像？")
}

func TestValidateUserAvatarAcceptsWeChatAvatar(t *testing.T) {
	require.NoError(t, ValidateUserAvatar("https://thirdwx.qlogo.cn/mmopen/vi_32/Q0j4TwGTfTKabc/132"))
	require.NoError(t, ValidateUserAvatar("http://thirdwx.qlogo.cn/mmopen/vi_32/abc/0"))
}
