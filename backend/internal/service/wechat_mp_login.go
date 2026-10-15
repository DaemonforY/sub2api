package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"golang.org/x/sync/singleflight"
)

// 公众号登录票据（login ticket）。浏览器先建票据，再由公众号消息服务器把 openid 绑上去：
//   - 扫码（PC）：带参二维码 scene_str = "hg_login_" + ticketID，扫码后的 subscribe / SCAN 事件绑定 openid。
//   - 验证码（手机浏览器）：网页显示 6 位验证码，用户把它发给公众号，加密文本消息里带着有效验证码就绑定 openid。
// 浏览器轮询 qr/status；拿到 openid 后原子地取走票据，走与网页授权相同的身份流程。

const (
	WeChatQRLoginScenePrefix = "hg_login_"

	WeChatLoginStatusPending = "pending"
	WeChatLoginStatusScanned = "scanned"
	WeChatLoginStatusExpired = "expired"
	WeChatLoginStatusDone    = "done"
	WeChatLoginStatusError   = "error"

	weChatLoginTicketTTL           = 300 * time.Second
	weChatLoginScannedTTL          = 120 * time.Second
	weChatLoginResultTTL           = 60 * time.Second
	WeChatLoginPollInterval        = 2
	weChatLoginCodeMaxGenTry       = 20
	weChatLoginBadCodeWindow       = 10 * time.Minute
	weChatLoginBadCodeLimit        = 5
	weChatMPMessageReplayTTL       = 10 * time.Minute
	weChatMPMessageMaxSkew         = 300 * time.Second
	weChatMPMessageDedupTTL        = 10 * time.Minute
	weChatMPForwardTimeout         = 3500 * time.Millisecond
	weChatMPAPITimeout             = 10 * time.Second
	weChatMPAccessTokenEarlyExpiry = 5 * time.Minute

	// WeChatMPWhitelistIP is the server's outbound IP that must be on the 公众号 IP whitelist.
	WeChatMPWhitelistIP = "43.133.80.169"

	defaultWeChatMPAPIBaseURL = "https://api.weixin.qq.com"
	weChatMPShowQRCodeURL     = "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket="

	weChatLoginInvalidCodeReply = "验证码无效或已过期，请在网页上重新获取（Invalid or expired code）"
	weChatLoginExpiredQRReply   = "二维码已过期，请回到网页刷新二维码后重新扫码。"
	weChatLoginUnavailableReply = "登录暂时不可用，请稍后在网页上重试。"
)

var weChatLoginTicketIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

var (
	ErrWeChatQRLoginDisabled     = infraerrors.NotFound("WECHAT_QR_LOGIN_DISABLED", "微信扫码登录未开启（WeChat QR login is disabled）")
	ErrWeChatCodeLoginDisabled   = infraerrors.NotFound("WECHAT_CODE_LOGIN_DISABLED", "微信验证码登录未开启（WeChat code login is disabled）")
	ErrWeChatLoginTicketNotFound = infraerrors.NotFound("WECHAT_LOGIN_TICKET_NOT_FOUND", "登录请求不存在或不属于当前浏览器，请刷新后重试（Login ticket not found）")
	ErrWeChatMPIPNotWhitelisted  = infraerrors.ServiceUnavailable("WECHAT_MP_IP_NOT_WHITELISTED",
		"微信接口拒绝了本服务器：请在公众平台「设置与开发 → 基本配置 → IP 白名单」里加入 "+WeChatMPWhitelistIP+"（Server IP is not on the WeChat official account IP whitelist）")
	ErrWeChatMPMessageDisabled = infraerrors.NotFound("WECHAT_MP_MESSAGE_DISABLED", "公众号消息服务器未开启（WeChat message server is disabled）")
	ErrWeChatMPBadSignature    = infraerrors.Forbidden("WECHAT_MP_BAD_SIGNATURE", "签名校验失败（Invalid signature）")
)

// WeChatLoginTicket is the Redis state of one login (key wechat:qrlogin:<id>), QR or code.
type WeChatLoginTicket struct {
	ID                string
	BrowserSessionKey string
	Redirect          string
	Intent            string
	AffCode           string
	PromoCode         string
	VisitorID         string
	Attribution       json.RawMessage
	BindUser          string // signed bind-user value (as the bind cookie), intent bind only
	Code              string // code login only
	OpenID            string // set once the 公众号 reports the scan / the code
	CreatedAt         int64
}

// WeChatLoginResult is what a claimed ticket answers its browser for 60 s.
type WeChatLoginResult struct {
	Status            string
	Message           string
	Redirect          string
	BrowserSessionKey string
}

// WeChatQRLoginCreated is what the browser needs to show and poll the QR.
type WeChatQRLoginCreated struct {
	TicketID     string `json:"ticket_id"`
	QRURL        string `json:"qr_url"`
	ExpiresIn    int    `json:"expires_in"`
	PollInterval int    `json:"poll_interval"`
}

// WeChatCodeLoginCreated is what the browser needs to show the code and poll.
type WeChatCodeLoginCreated struct {
	TicketID     string `json:"ticket_id"`
	Code         string `json:"code"`
	ExpiresIn    int    `json:"expires_in"`
	PollInterval int    `json:"poll_interval"`
}

// WeChatMPLoginStore is the short-lived state of the logins (Redis in production). Keys are chosen
// by the service.
type WeChatMPLoginStore interface {
	// Incr increments a fixed-window counter, setting ttl when the key is new.
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
	// HSetWithTTL writes a hash and its expiry.
	HSetWithTTL(ctx context.Context, key string, fields map[string]string, ttl time.Duration) error
	// HGetAll returns an empty map when the hash does not exist.
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	// ClaimHashField sets field on an existing hash only when it is unset, atomically. Returns -1 when
	// the hash does not exist, 1 when the field now holds value (set now, or already equal), 0 when it
	// holds another value. When it sets the field it extends the hash's TTL to at least minTTL.
	ClaimHashField(ctx context.Context, key, field, value string, minTTL time.Duration) (int, error)
	// TakeHash atomically deletes the hash at key when requiredField is set in it and returns its
	// fields, writing marker (a hash) at markerKey with markerTTL in the same step. Returns an empty
	// map (and writes nothing) when the hash is missing or requiredField is unset.
	TakeHash(ctx context.Context, key, requiredField, markerKey string, marker map[string]string, markerTTL time.Duration) (map[string]string, error)
}

// WeChatMPLoginService runs the 公众号 message server and the QR / code logins.
type WeChatMPLoginService struct {
	settings   *SettingService
	store      WeChatMPLoginStore
	httpClient *http.Client
	apiBaseURL string
	random     io.Reader
	now        func() time.Time
	tokenSF    singleflight.Group

	forwardTimeout time.Duration
}

// NewWeChatMPLoginService builds the service.
func NewWeChatMPLoginService(settings *SettingService, store WeChatMPLoginStore) *WeChatMPLoginService {
	return &WeChatMPLoginService{
		settings:       settings,
		store:          store,
		httpClient:     &http.Client{Timeout: weChatMPAPITimeout},
		apiBaseURL:     defaultWeChatMPAPIBaseURL,
		random:         rand.Reader,
		now:            time.Now,
		forwardTimeout: weChatMPForwardTimeout,
	}
}

// SetAPIBaseURL points the WeChat API calls elsewhere (tests).
func (s *WeChatMPLoginService) SetAPIBaseURL(base string) {
	s.apiBaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
}

// SetHTTPClient replaces the HTTP client used for WeChat API calls and forwarding (tests).
func (s *WeChatMPLoginService) SetHTTPClient(client *http.Client) {
	if client != nil {
		s.httpClient = client
	}
}

// SetClock replaces the clock used for message timestamps (tests).
func (s *WeChatMPLoginService) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// Config returns the effective 公众号 configuration.
func (s *WeChatMPLoginService) Config(ctx context.Context) (WeChatMPConfig, error) {
	if s == nil || s.settings == nil {
		return WeChatMPConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "设置服务未就绪（settings service not ready）")
	}
	return s.settings.GetWeChatMPConfig(ctx)
}

func (s *WeChatMPLoginService) ready() error {
	if s == nil || s.store == nil {
		return infraerrors.ServiceUnavailable("WECHAT_MP_NOT_READY", "微信登录服务未就绪（WeChat login service not ready）")
	}
	return nil
}

func weChatLoginTicketKey(id string) string      { return "wechat:qrlogin:" + id }
func weChatLoginResultKey(id string) string      { return "wechat:qrlogin:" + id + ":result" }
func weChatLoginCodeKey(code string) string      { return "wechat:logincode:" + code }
func weChatLoginBadCodeKey(openid string) string { return "wechat:logincode:tries:" + openid }
func weChatMPAccessTokenKey(appID string) string { return "wechat:mp:access_token:" + appID }
func weChatMPMessageDedupKey(id string) string   { return "wechat:mp:msg:" + id }
func weChatMPNonceKey(timestamp, nonce string) string {
	return "wechat:mpnonce:" + timestamp + ":" + nonce
}

// ValidWeChatLoginTicketID reports whether id has the shape the create calls issue.
func ValidWeChatLoginTicketID(id string) bool {
	return weChatLoginTicketIDPattern.MatchString(id)
}

// ---------- create ----------

// CreateQRLogin creates a temporary 公众号 QR for a new ticket and stores the ticket.
func (s *WeChatMPLoginService) CreateQRLogin(ctx context.Context, in WeChatLoginTicket) (*WeChatQRLoginCreated, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	cfg, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.QRLoginAvailable() {
		return nil, ErrWeChatQRLoginDisabled
	}
	id, err := s.randomHex(16)
	if err != nil {
		return nil, infraerrors.InternalServer("WECHAT_LOGIN_TICKET_GEN_FAILED", "生成二维码失败，请稍后再试（failed to generate ticket）").WithCause(err)
	}

	body := map[string]any{
		"expire_seconds": int(weChatLoginTicketTTL / time.Second),
		"action_name":    "QR_STR_SCENE",
		"action_info": map[string]any{
			"scene": map[string]any{"scene_str": WeChatQRLoginScenePrefix + id},
		},
	}
	raw, err := s.callAPI(ctx, cfg, http.MethodPost, "/cgi-bin/qrcode/create", nil, body)
	if err != nil {
		return nil, err
	}
	var qr struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(raw, &qr); err != nil || strings.TrimSpace(qr.Ticket) == "" {
		return nil, infraerrors.ServiceUnavailable("WECHAT_MP_API_ERROR", "微信没有返回二维码，请稍后再试（WeChat returned no QR ticket）")
	}
	in.ID, in.Code = id, ""
	if err := s.storeTicket(ctx, in); err != nil {
		return nil, err
	}
	return &WeChatQRLoginCreated{
		TicketID:     id,
		QRURL:        weChatMPShowQRCodeURL + url.QueryEscape(qr.Ticket),
		ExpiresIn:    int(weChatLoginTicketTTL / time.Second),
		PollInterval: WeChatLoginPollInterval,
	}, nil
}

// CreateCodeLogin creates a ticket with a 6-digit code (unique among live codes) for the user to
// send to the 公众号.
func (s *WeChatMPLoginService) CreateCodeLogin(ctx context.Context, in WeChatLoginTicket) (*WeChatCodeLoginCreated, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	cfg, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.CodeLoginAvailable() {
		return nil, ErrWeChatCodeLoginDisabled
	}
	id, err := s.randomHex(16)
	if err != nil {
		return nil, infraerrors.InternalServer("WECHAT_LOGIN_TICKET_GEN_FAILED", "生成验证码失败，请稍后再试（failed to generate ticket）").WithCause(err)
	}
	code := ""
	for i := 0; i < weChatLoginCodeMaxGenTry && code == ""; i++ {
		candidate, err := s.randomDigits(6)
		if err != nil {
			return nil, infraerrors.InternalServer("WECHAT_LOGIN_TICKET_GEN_FAILED", "生成验证码失败，请稍后再试（failed to generate code）").WithCause(err)
		}
		ok, err := s.store.SetNX(ctx, weChatLoginCodeKey(candidate), id, weChatLoginTicketTTL)
		if err != nil {
			return nil, infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_STORE_FAILED", "生成验证码失败，请稍后再试（failed to store code）").WithCause(err)
		}
		if ok {
			code = candidate
		}
	}
	if code == "" {
		return nil, infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_STORE_FAILED", "生成验证码失败，请稍后再试（no free code）")
	}
	in.ID, in.Code = id, code
	if err := s.storeTicket(ctx, in); err != nil {
		_ = s.store.Del(ctx, weChatLoginCodeKey(code))
		return nil, err
	}
	return &WeChatCodeLoginCreated{
		TicketID:     id,
		Code:         code,
		ExpiresIn:    int(weChatLoginTicketTTL / time.Second),
		PollInterval: WeChatLoginPollInterval,
	}, nil
}

func (s *WeChatMPLoginService) storeTicket(ctx context.Context, in WeChatLoginTicket) error {
	fields := map[string]string{
		"browser":    in.BrowserSessionKey,
		"redirect":   in.Redirect,
		"intent":     in.Intent,
		"aff":        in.AffCode,
		"promo":      in.PromoCode,
		"visitor":    in.VisitorID,
		"attr":       string(in.Attribution),
		"bind_user":  in.BindUser,
		"code":       in.Code,
		"created_at": strconv.FormatInt(s.now().Unix(), 10),
	}
	if err := s.store.HSetWithTTL(ctx, weChatLoginTicketKey(in.ID), fields, weChatLoginTicketTTL); err != nil {
		return infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_STORE_FAILED", "生成登录请求失败，请稍后再试（failed to store ticket）").WithCause(err)
	}
	return nil
}

func weChatLoginTicketFromFields(id string, values map[string]string) *WeChatLoginTicket {
	if len(values) == 0 {
		return nil
	}
	createdAt, _ := strconv.ParseInt(values["created_at"], 10, 64)
	t := &WeChatLoginTicket{
		ID:                id,
		BrowserSessionKey: values["browser"],
		Redirect:          values["redirect"],
		Intent:            values["intent"],
		AffCode:           values["aff"],
		PromoCode:         values["promo"],
		VisitorID:         values["visitor"],
		BindUser:          values["bind_user"],
		Code:              values["code"],
		OpenID:            values["openid"],
		CreatedAt:         createdAt,
	}
	if attr := strings.TrimSpace(values["attr"]); attr != "" && json.Valid([]byte(attr)) {
		t.Attribution = json.RawMessage(attr)
	}
	return t
}

// ---------- status ----------

// GetLoginTicket returns the ticket, or nil when it does not exist (expired or taken).
func (s *WeChatMPLoginService) GetLoginTicket(ctx context.Context, id string) (*WeChatLoginTicket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !ValidWeChatLoginTicketID(id) {
		return nil, nil
	}
	values, err := s.store.HGetAll(ctx, weChatLoginTicketKey(id))
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_LOAD_FAILED", "读取登录状态失败，请稍后再试（failed to load ticket）").WithCause(err)
	}
	return weChatLoginTicketFromFields(id, values), nil
}

// ClaimLoginTicket atomically takes a ticket whose openid has arrived, so only one poll can finish
// it, and leaves a "scanned" result for the other polls in the meantime. Nil when there is nothing
// to take (gone, not scanned yet, or another poll took it).
func (s *WeChatMPLoginService) ClaimLoginTicket(ctx context.Context, id, browserSessionKey string) (*WeChatLoginTicket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !ValidWeChatLoginTicketID(id) {
		return nil, nil
	}
	marker := weChatLoginResultFields(WeChatLoginResult{Status: WeChatLoginStatusScanned, BrowserSessionKey: browserSessionKey})
	values, err := s.store.TakeHash(ctx, weChatLoginTicketKey(id), "openid", weChatLoginResultKey(id), marker, weChatLoginResultTTL)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_LOAD_FAILED", "读取登录状态失败，请稍后再试（failed to claim ticket）").WithCause(err)
	}
	ticket := weChatLoginTicketFromFields(id, values)
	if ticket != nil && ticket.Code != "" {
		_ = s.store.Del(ctx, weChatLoginCodeKey(ticket.Code))
	}
	return ticket, nil
}

func weChatLoginResultFields(r WeChatLoginResult) map[string]string {
	return map[string]string{
		"status":   r.Status,
		"message":  r.Message,
		"redirect": r.Redirect,
		"browser":  r.BrowserSessionKey,
	}
}

// SetLoginResult records how a claimed ticket ended (done / error) for its browser, for 60 s.
func (s *WeChatMPLoginService) SetLoginResult(ctx context.Context, id string, r WeChatLoginResult) {
	if s.ready() != nil || !ValidWeChatLoginTicketID(id) {
		return
	}
	if err := s.store.HSetWithTTL(ctx, weChatLoginResultKey(id), weChatLoginResultFields(r), weChatLoginResultTTL); err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] store login result failed: %v", err)
	}
}

// GetLoginResult returns the result of a claimed ticket, or nil.
func (s *WeChatMPLoginService) GetLoginResult(ctx context.Context, id string) (*WeChatLoginResult, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if !ValidWeChatLoginTicketID(id) {
		return nil, nil
	}
	values, err := s.store.HGetAll(ctx, weChatLoginResultKey(id))
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("WECHAT_LOGIN_TICKET_LOAD_FAILED", "读取登录状态失败，请稍后再试（failed to load result）").WithCause(err)
	}
	if len(values) == 0 || values["status"] == "" {
		return nil, nil
	}
	return &WeChatLoginResult{
		Status:            values["status"],
		Message:           values["message"],
		Redirect:          values["redirect"],
		BrowserSessionKey: values["browser"],
	}, nil
}

// ---------- binding (message server) ----------

// BindLoginTicket attaches openid to a live ticket; false when the ticket is gone or someone else
// got there first.
func (s *WeChatMPLoginService) BindLoginTicket(ctx context.Context, id, openid string) (bool, error) {
	if err := s.ready(); err != nil {
		return false, err
	}
	openid = strings.TrimSpace(openid)
	if !ValidWeChatLoginTicketID(id) || openid == "" {
		return false, nil
	}
	res, err := s.store.ClaimHashField(ctx, weChatLoginTicketKey(id), "openid", openid, weChatLoginScannedTTL)
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// bindLoginCode handles a code sent to the 公众号 and returns the reply ("" = stay silent).
// Every code a sender tries counts first (one atomic INCR, TTL set on the first): past 5 in 10
// minutes it gets no answer and no lookup, so guessing in parallel gains nothing.
func (s *WeChatMPLoginService) bindLoginCode(ctx context.Context, cfg WeChatMPConfig, openid, code string) string {
	n, err := s.store.Incr(ctx, weChatLoginBadCodeKey(openid), weChatLoginBadCodeWindow)
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] count login code attempt failed: %v", err)
		return weChatLoginUnavailableReply
	}
	if n > weChatLoginBadCodeLimit {
		return ""
	}
	ticketID, ok, err := s.store.Get(ctx, weChatLoginCodeKey(code))
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] look up login code failed: %v", err)
		return weChatLoginUnavailableReply
	}
	if ok {
		bound, err := s.BindLoginTicket(ctx, ticketID, openid)
		if err != nil {
			logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] bind login code failed: %v", err)
			return weChatLoginUnavailableReply
		}
		if bound {
			_ = s.store.Del(ctx, weChatLoginCodeKey(code))
			return weChatMPScanOKReply(cfg)
		}
	}
	return weChatLoginInvalidCodeReply
}

// ---------- helpers ----------

func (s *WeChatMPLoginService) randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(s.random, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *WeChatMPLoginService) randomDigits(n int) (string, error) {
	var b strings.Builder
	for i := 0; i < n; i++ {
		d, err := rand.Int(s.random, big.NewInt(10))
		if err != nil {
			return "", err
		}
		_ = b.WriteByte(byte('0' + d.Int64()))
	}
	return b.String(), nil
}
