package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 公众号消息服务器。
//   - 登录相关（hg_login_ 二维码事件、验证码、登录关键词）只处理安全模式（encrypt_type=aes）消息：
//     msg_signature 正确、解密后 appid 一致。配置了 EncodingAESKey 时明文消息一律回 success 并记日志。
//   - timestamp 与本机相差超过 300 秒的拒绝；同一 timestamp+nonce 只处理一次（10 分钟），
//     微信自己的重试（同 MsgId / 同事件）拿到缓存的同一条回复。
//   - 同一公众号的「服务器配置」只能填一个 URL；我们不处理的消息可原样转发到
//     wechat_mp_message_forward_url（默认关闭）。

const weChatMPSuccessReply = "success"

// WeChatMPMessageRequest is the raw POST WeChat sent us.
type WeChatMPMessageRequest struct {
	Method      string
	RawQuery    string
	ContentType string
	Body        []byte
}

// WeChatMPMessageResponse is written back to WeChat verbatim.
type WeChatMPMessageResponse struct {
	Status      int
	ContentType string
	Body        []byte
}

func weChatMPSuccessResponse() *WeChatMPMessageResponse {
	return &WeChatMPMessageResponse{Status: http.StatusOK, ContentType: "text/plain; charset=utf-8", Body: []byte(weChatMPSuccessReply)}
}

type weChatMPInbound struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	Event        string   `xml:"Event"`
	EventKey     string   `xml:"EventKey"`
	MsgID        int64    `xml:"MsgId"`
	Encrypt      string   `xml:"Encrypt"`
}

// VerifyServerURL answers WeChat's 服务器配置 check: echo echostr when the signature is right.
func (s *WeChatMPLoginService) VerifyServerURL(ctx context.Context, query url.Values) (string, error) {
	cfg, err := s.Config(ctx)
	if err != nil {
		return "", err
	}
	if !cfg.MessageServerReady() {
		return "", ErrWeChatMPMessageDisabled
	}
	if !WeChatMPSignatureValid(query.Get("signature"), cfg.MessageToken, query.Get("timestamp"), query.Get("nonce")) {
		return "", ErrWeChatMPBadSignature
	}
	return query.Get("echostr"), nil
}

// HandleMessage answers one pushed message or event.
func (s *WeChatMPLoginService) HandleMessage(ctx context.Context, req WeChatMPMessageRequest) (*WeChatMPMessageResponse, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	cfg, err := s.Config(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.MessageServerReady() {
		return nil, ErrWeChatMPMessageDisabled
	}
	query, _ := url.ParseQuery(req.RawQuery)
	timestamp, nonce := query.Get("timestamp"), query.Get("nonce")
	if !WeChatMPSignatureValid(query.Get("signature"), cfg.MessageToken, timestamp, nonce) {
		return nil, ErrWeChatMPBadSignature
	}
	if !s.timestampFresh(timestamp) {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] dropped message with stale timestamp %q", timestamp)
		return weChatMPSuccessResponse(), nil
	}

	var outer weChatMPInbound
	if err := xml.Unmarshal(req.Body, &outer); err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] unreadable message body: %v", err)
		return weChatMPSuccessResponse(), nil
	}
	encrypted := strings.EqualFold(strings.TrimSpace(query.Get("encrypt_type")), "aes")
	aesConfigured := cfg.MessageAESKey != ""

	if !encrypted {
		if aesConfigured {
			// 安全模式下不该有明文消息：可能是伪造的，什么都不做。
			logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] rejected a plain message while EncodingAESKey is set (from %q)", outer.FromUserName)
			return weChatMPSuccessResponse(), nil
		}
		// Plain mode (no AES key): never any login handling; forward or the generic replies only.
		if fresh, cached := s.checkReplay(ctx, timestamp, nonce, outer); !fresh {
			return s.replayResponse(cfg, outer, cached, nil, timestamp, nonce)
		}
		if cfg.ForwardURL != "" {
			return s.forward(ctx, cfg, req), nil
		}
		return s.reply(cfg, outer, s.fallbackReply(cfg, outer), nil, timestamp, nonce)
	}

	if !aesConfigured {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] encrypted message but no EncodingAESKey is set")
		return weChatMPSuccessResponse(), nil
	}
	if !WeChatMPSignatureValid(query.Get("msg_signature"), cfg.MessageToken, timestamp, nonce, outer.Encrypt) {
		return nil, ErrWeChatMPBadSignature
	}
	crypter, err := NewWeChatMPCrypter(cfg.MessageAESKey, cfg.AppID, s.random)
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] EncodingAESKey is not usable: %v", err)
		return weChatMPSuccessResponse(), nil
	}
	plain, err := crypter.Decrypt(outer.Encrypt)
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] decrypt message failed: %v", err)
		return weChatMPSuccessResponse(), nil
	}
	var msg weChatMPInbound
	if err := xml.Unmarshal(plain, &msg); err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] unreadable decrypted message: %v", err)
		return weChatMPSuccessResponse(), nil
	}

	fresh, cached := s.checkReplay(ctx, timestamp, nonce, msg)
	if !fresh {
		return s.replayResponse(cfg, msg, cached, crypter, timestamp, nonce)
	}
	if cached != nil {
		// A WeChat retry that came with a new nonce.
		return s.reply(cfg, msg, *cached, crypter, timestamp, nonce)
	}

	handle, ok := s.ownHandler(cfg, msg)
	if !ok {
		if cfg.ForwardURL != "" {
			return s.forward(ctx, cfg, req), nil
		}
		return s.reply(cfg, msg, s.fallbackReply(cfg, msg), crypter, timestamp, nonce)
	}
	text := handle(ctx)
	if err := s.store.Set(ctx, weChatMPMessageDedupKey(weChatMPMessageID(msg)), text, weChatMPMessageDedupTTL); err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] store message reply failed: %v", err)
	}
	return s.reply(cfg, msg, text, crypter, timestamp, nonce)
}

func (s *WeChatMPLoginService) timestampFresh(timestamp string) bool {
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return false
	}
	skew := s.now().Sub(time.Unix(ts, 0))
	if skew < 0 {
		skew = -skew
	}
	return skew <= weChatMPMessageMaxSkew
}

// checkReplay claims timestamp+nonce. fresh=false means this exact request was seen before. cached
// is the reply already given to this MsgId / event, if any (WeChat retries get it again).
func (s *WeChatMPLoginService) checkReplay(ctx context.Context, timestamp, nonce string, msg weChatMPInbound) (fresh bool, cached *string) {
	if v, ok, err := s.store.Get(ctx, weChatMPMessageDedupKey(weChatMPMessageID(msg))); err == nil && ok {
		cached = &v
	}
	claimed, err := s.store.SetNX(ctx, weChatMPNonceKey(timestamp, nonce), "1", weChatMPMessageReplayTTL)
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] replay check failed: %v", err)
		return false, cached
	}
	return claimed, cached
}

// replayResponse answers a repeated timestamp+nonce: the cached reply when there is one (WeChat's
// own retry), otherwise nothing.
func (s *WeChatMPLoginService) replayResponse(cfg WeChatMPConfig, msg weChatMPInbound, cached *string, crypter *WeChatMPCrypter, timestamp, nonce string) (*WeChatMPMessageResponse, error) {
	if cached == nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] dropped a replayed message (timestamp %s)", timestamp)
		return weChatMPSuccessResponse(), nil
	}
	return s.reply(cfg, msg, *cached, crypter, timestamp, nonce)
}

// ownHandler returns the handler for the messages this site answers itself (安全模式 only).
func (s *WeChatMPLoginService) ownHandler(cfg WeChatMPConfig, msg weChatMPInbound) (func(context.Context) string, bool) {
	openid := strings.TrimSpace(msg.FromUserName)
	if openid == "" {
		return nil, false
	}
	switch strings.ToLower(strings.TrimSpace(msg.MsgType)) {
	case "event":
		key := strings.TrimSpace(msg.EventKey)
		var ticketID string
		switch strings.ToLower(strings.TrimSpace(msg.Event)) {
		case "subscribe":
			if !strings.HasPrefix(key, "qrscene_"+WeChatQRLoginScenePrefix) {
				return nil, false
			}
			ticketID = strings.TrimPrefix(key, "qrscene_"+WeChatQRLoginScenePrefix)
		case "scan":
			if !strings.HasPrefix(key, WeChatQRLoginScenePrefix) {
				return nil, false
			}
			ticketID = strings.TrimPrefix(key, WeChatQRLoginScenePrefix)
		default:
			return nil, false
		}
		return func(ctx context.Context) string {
			bound, err := s.BindLoginTicket(ctx, ticketID, openid)
			if err != nil {
				logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] bind qr ticket failed: %v", err)
				return weChatLoginUnavailableReply
			}
			if !bound {
				return weChatLoginExpiredQRReply
			}
			return cfg.ReplyScanOK
		}, true
	case "text":
		if !cfg.CodeLoginAvailable() {
			return nil, false
		}
		if code := extractWeChatLoginCode(cfg, msg.Content); code != "" {
			return func(ctx context.Context) string {
				return s.bindLoginCode(ctx, cfg, openid, code)
			}, true
		}
		if weChatMPIsLoginKeyword(cfg, msg.Content) {
			return func(context.Context) string {
				return strings.ReplaceAll(cfg.ReplyCode, "{site_url}", cfg.SiteURL)
			}, true
		}
	}
	return nil, false
}

// extractWeChatLoginCode returns the 6-digit code in "123456", "登录 123456" or "登录123456".
func extractWeChatLoginCode(cfg WeChatMPConfig, content string) string {
	content = strings.TrimSpace(content)
	if isSixDigits(content) {
		return content
	}
	lower := strings.ToLower(content)
	for _, kw := range cfg.LoginKeywords {
		kw = strings.ToLower(strings.TrimSpace(kw))
		if kw == "" || !strings.HasPrefix(lower, kw) {
			continue
		}
		rest := strings.TrimLeft(content[len(kw):], " \t　:：")
		if isSixDigits(rest) {
			return rest
		}
	}
	return ""
}

func isSixDigits(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func weChatMPIsLoginKeyword(cfg WeChatMPConfig, content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	for _, kw := range cfg.LoginKeywords {
		if strings.EqualFold(content, kw) {
			return true
		}
	}
	return false
}

func (s *WeChatMPLoginService) fallbackReply(cfg WeChatMPConfig, msg weChatMPInbound) string {
	if strings.EqualFold(msg.MsgType, "event") {
		if strings.EqualFold(msg.Event, "subscribe") {
			return strings.ReplaceAll(cfg.ReplyWelcome, "{site_url}", cfg.SiteURL)
		}
		return ""
	}
	return cfg.ReplyDefault
}

func weChatMPMessageID(msg weChatMPInbound) string {
	if msg.MsgID > 0 {
		return strconv.FormatInt(msg.MsgID, 10)
	}
	return strings.Join([]string{msg.FromUserName, strconv.FormatInt(msg.CreateTime, 10), msg.Event, msg.EventKey}, "|")
}

// forward relays the raw request (method, query, body, content type) and returns the reply verbatim
// when it is 2xx; "success" on a non-2xx, an error or a timeout.
func (s *WeChatMPLoginService) forward(ctx context.Context, cfg WeChatMPConfig, req WeChatMPMessageRequest) *WeChatMPMessageResponse {
	target := cfg.ForwardURL
	if req.RawQuery != "" {
		if strings.Contains(target, "?") {
			target += "&" + req.RawQuery
		} else {
			target += "?" + req.RawQuery
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.forwardTimeout)
	defer cancel()
	method := req.Method
	if method == "" {
		method = http.MethodPost
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(req.Body))
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] build forward request failed: %v", err)
		return weChatMPSuccessResponse()
	}
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	}
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] forward message failed: %v", err)
		return weChatMPSuccessResponse()
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] forward target answered %d", resp.StatusCode)
		return weChatMPSuccessResponse()
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] read forward reply failed: %v", err)
		return weChatMPSuccessResponse()
	}
	return &WeChatMPMessageResponse{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: body}
}

// reply builds a passive text reply, encrypted when the request was; "success" when text is empty.
func (s *WeChatMPLoginService) reply(cfg WeChatMPConfig, msg weChatMPInbound, text string, crypter *WeChatMPCrypter, timestamp, nonce string) (*WeChatMPMessageResponse, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return weChatMPSuccessResponse(), nil
	}
	now := strconv.FormatInt(s.now().Unix(), 10)
	plain := "<xml>" +
		"<ToUserName>" + weChatMPCDATA(msg.FromUserName) + "</ToUserName>" +
		"<FromUserName>" + weChatMPCDATA(msg.ToUserName) + "</FromUserName>" +
		"<CreateTime>" + now + "</CreateTime>" +
		"<MsgType><![CDATA[text]]></MsgType>" +
		"<Content>" + weChatMPCDATA(text) + "</Content>" +
		"</xml>"
	if crypter == nil {
		return &WeChatMPMessageResponse{Status: http.StatusOK, ContentType: "application/xml; charset=utf-8", Body: []byte(plain)}, nil
	}
	encrypted, err := crypter.Encrypt([]byte(plain))
	if err != nil {
		logger.LegacyPrintf("service.wechat_mp", "[WeChatMP] encrypt reply failed: %v", err)
		return weChatMPSuccessResponse(), nil
	}
	signature := WeChatMPSignature(cfg.MessageToken, timestamp, nonce, encrypted)
	out := "<xml>" +
		"<Encrypt>" + weChatMPCDATA(encrypted) + "</Encrypt>" +
		"<MsgSignature>" + weChatMPCDATA(signature) + "</MsgSignature>" +
		"<TimeStamp>" + timestamp + "</TimeStamp>" +
		"<Nonce>" + weChatMPCDATA(nonce) + "</Nonce>" +
		"</xml>"
	return &WeChatMPMessageResponse{Status: http.StatusOK, ContentType: "application/xml; charset=utf-8", Body: []byte(out)}, nil
}

func weChatMPCDATA(s string) string {
	return "<![CDATA[" + strings.ReplaceAll(s, "]]>", "]]]]><![CDATA[>") + "]]>"
}
