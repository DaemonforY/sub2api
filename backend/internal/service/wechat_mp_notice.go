package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/authidentity"
	"github.com/Wei-Shaw/sub2api/ent/authidentitychannel"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 微信提醒：只用公众号身份注册（邮箱是 *.invalid 占位地址）的用户收不到邮件，四种交易类提醒改用
// 公众号模板消息（cgi-bin/message/template/send）发到他们的 openid。有真实邮箱的用户照旧收邮件，
// 不会两边都发。召回（winback）属于营销，微信规定模板消息不能发，不走这里。

const (
	SettingKeyWeChatMPTemplateNotices = "wechat_mp_template_notices"

	WeChatNoticeBalanceLow           = "balance_low"
	WeChatNoticeTrialExhausted       = "trial_exhausted"
	WeChatNoticeSubscriptionExpiring = "subscription_expiring"
	WeChatNoticeOrderAbandoned       = "order_abandoned"

	weChatNoticeMaxFields       = 10
	weChatNoticeMaxValueRunes   = 200
	weChatNoticeMaxURLPathLen   = 512
	weChatNoticeWarnInterval    = time.Hour
	weChatNoticeTimeLayout      = "2006-01-02 15:04"
	weChatNoticeTemplateSendAPI = "/cgi-bin/message/template/send"
)

// WeChatNoticeEvents lists the template-message events in display order.
var WeChatNoticeEvents = []string{WeChatNoticeBalanceLow, WeChatNoticeTrialExhausted, WeChatNoticeSubscriptionExpiring, WeChatNoticeOrderAbandoned}

// weChatNoticeEventForEmail maps the email events that may go out as a template message.
var weChatNoticeEventForEmail = map[string]string{
	NotificationEmailEventBalanceLow:                 WeChatNoticeBalanceLow,
	NotificationEmailEventGrowthTrialExhausted:       WeChatNoticeTrialExhausted,
	NotificationEmailEventSubscriptionExpiryReminder: WeChatNoticeSubscriptionExpiring,
	NotificationEmailEventPaymentOrderAbandoned:      WeChatNoticeOrderAbandoned,
}

var weChatNoticeCommonPlaceholders = []string{"site_name", "username", "now"}

// WeChatNoticePlaceholders lists the placeholders each event's field values may use.
var WeChatNoticePlaceholders = map[string][]string{
	WeChatNoticeBalanceLow:           {"balance", "threshold"},
	WeChatNoticeTrialExhausted:       {"balance"},
	WeChatNoticeSubscriptionExpiring: {"plan_name", "expires_at", "days_left"},
	WeChatNoticeOrderAbandoned:       {"order_no", "product", "amount_cny", "created_at"},
}

var (
	weChatNoticeTemplateIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	weChatNoticeFieldKeyPattern   = regexp.MustCompile(`^[a-z_]+[0-9]+$`)
	weChatNoticePlaceholderRegexp = regexp.MustCompile(`\{([a-z_]+)\}`)
	weChatNoticeKeyDigits         = regexp.MustCompile(`[0-9]+$`)
)

// WeChatTemplateNoticeConfig is one event's template-message setting.
type WeChatTemplateNoticeConfig struct {
	Enabled    bool              `json:"enabled"`
	TemplateID string            `json:"template_id"`
	URLPath    string            `json:"url_path"`
	Fields     map[string]string `json:"fields"`
}

// ParseWeChatTemplateNotices reads the stored JSON; every known event is present (disabled by default).
func ParseWeChatTemplateNotices(raw string) map[string]WeChatTemplateNoticeConfig {
	stored := map[string]WeChatTemplateNoticeConfig{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &stored)
	}
	out := make(map[string]WeChatTemplateNoticeConfig, len(WeChatNoticeEvents))
	for _, event := range WeChatNoticeEvents {
		cfg := stored[event]
		if cfg.Fields == nil {
			cfg.Fields = map[string]string{}
		}
		out[event] = cfg
	}
	return out
}

// NormalizeWeChatTemplateNotices validates the admin's settings before they are saved.
func NormalizeWeChatTemplateNotices(in map[string]WeChatTemplateNoticeConfig) (map[string]WeChatTemplateNoticeConfig, error) {
	out := make(map[string]WeChatTemplateNoticeConfig, len(WeChatNoticeEvents))
	for event := range in {
		if _, ok := WeChatNoticePlaceholders[event]; !ok {
			return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("不支持的提醒类型「%s」（Unknown notice event）", event))
		}
	}
	for _, event := range WeChatNoticeEvents {
		cfg := in[event]
		cfg.TemplateID = strings.TrimSpace(cfg.TemplateID)
		cfg.URLPath = strings.TrimSpace(cfg.URLPath)
		if cfg.TemplateID != "" && !weChatNoticeTemplateIDPattern.MatchString(cfg.TemplateID) {
			return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」的模板 ID 格式不对（Invalid template_id）", event))
		}
		if cfg.URLPath != "" && !weChatNoticeSameSitePath(cfg.URLPath) {
			return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」的跳转页面要填本站路径，比如 /purchase（url_path must be a same-site path）", event))
		}
		if len(cfg.Fields) > weChatNoticeMaxFields {
			return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」最多 %d 个字段（Too many fields）", event, weChatNoticeMaxFields))
		}
		allowed := weChatNoticeAllowedPlaceholders(event)
		fields := make(map[string]string, len(cfg.Fields))
		for key, value := range cfg.Fields {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if !weChatNoticeFieldKeyPattern.MatchString(key) {
				return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」的字段名「%s」不对，应形如 thing1、amount2（Invalid field key）", event, key))
			}
			if len([]rune(value)) > weChatNoticeMaxValueRunes {
				return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」的字段「%s」太长了（Field value too long）", event, key))
			}
			for _, m := range weChatNoticePlaceholderRegexp.FindAllStringSubmatch(value, -1) {
				if _, ok := allowed[m[1]]; !ok {
					return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("「%s」不支持占位符 {%s}（Unknown placeholder）", event, m[1]))
				}
			}
			fields[key] = value
		}
		cfg.Fields = fields
		if cfg.Enabled && (cfg.TemplateID == "" || len(cfg.Fields) == 0) {
			return nil, infraerrors.BadRequest("WECHAT_NOTICE_INVALID", fmt.Sprintf("开启「%s」前请填写模板 ID 和字段（template_id and fields required）", event))
		}
		out[event] = cfg
	}
	return out, nil
}

func marshalWeChatTemplateNotices(cfg map[string]WeChatTemplateNoticeConfig) string {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func weChatNoticeSameSitePath(p string) bool {
	return len(p) <= weChatNoticeMaxURLPathLen && strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") &&
		!strings.Contains(p, "://") && !strings.ContainsAny(p, "\r\n\\ ")
}

func weChatNoticeAllowedPlaceholders(event string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, p := range weChatNoticeCommonPlaceholders {
		out[p] = struct{}{}
	}
	for _, p := range WeChatNoticePlaceholders[event] {
		out[p] = struct{}{}
	}
	return out
}

// fillWeChatNoticeValue replaces {placeholder}s; unknown ones are left as they are.
func fillWeChatNoticeValue(tmpl string, vars map[string]string) string {
	return weChatNoticePlaceholderRegexp.ReplaceAllStringFunc(tmpl, func(m string) string {
		if v, ok := vars[m[1:len(m)-1]]; ok {
			return v
		}
		return m
	})
}

// truncateWeChatNoticeValue applies WeChat's per-type length limits, by field key prefix:
// thing ≤ 20 runes, character_string ≤ 32, phrase ≤ 5; number / amount / time / others unchanged.
func truncateWeChatNoticeValue(key, value string) string {
	limit := 0
	switch weChatNoticeKeyDigits.ReplaceAllString(key, "") {
	case "thing":
		limit = 20
	case "character_string":
		limit = 32
	case "phrase":
		limit = 5
	}
	if limit == 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

// ---------- service ----------

// WeChatNoticeOutcome says what happened to a template-message notice.
type WeChatNoticeOutcome int

const (
	// WeChatNoticeNotApplicable: not enabled for the event, or the user has no 公众号 openid.
	WeChatNoticeNotApplicable WeChatNoticeOutcome = iota
	// WeChatNoticeDelivered: WeChat accepted it.
	WeChatNoticeDelivered
	// WeChatNoticeSkipped: the user doesn't follow the 公众号 or refused its messages; counts as delivered.
	WeChatNoticeSkipped
	// WeChatNoticeDeferred: the template is misconfigured; not delivered, retried on the next run.
	WeChatNoticeDeferred
)

// WeChatTemplateNoticeService sends the transactional reminders as 公众号 template messages.
type WeChatTemplateNoticeService struct {
	settings *SettingService
	mp       *WeChatMPLoginService
	client   *dbent.Client
	now      func() time.Time

	warnMu   sync.Mutex
	lastWarn map[string]time.Time
}

// NewWeChatTemplateNoticeService builds the service.
func NewWeChatTemplateNoticeService(settings *SettingService, mp *WeChatMPLoginService, client *dbent.Client) *WeChatTemplateNoticeService {
	return &WeChatTemplateNoticeService{settings: settings, mp: mp, client: client, now: time.Now, lastWarn: map[string]time.Time{}}
}

func (s *WeChatTemplateNoticeService) loadConfig(ctx context.Context) (WeChatMPConfig, map[string]WeChatTemplateNoticeConfig, error) {
	if s == nil || s.settings == nil || s.settings.settingRepo == nil || s.mp == nil {
		return WeChatMPConfig{}, nil, errors.New("wechat template notices not configured")
	}
	mpCfg, err := s.mp.Config(ctx)
	if err != nil {
		return WeChatMPConfig{}, nil, err
	}
	raw, err := s.settings.settingRepo.GetValue(ctx, SettingKeyWeChatMPTemplateNotices)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return WeChatMPConfig{}, nil, err
	}
	return mpCfg, ParseWeChatTemplateNotices(raw), nil
}

// Enabled reports whether event goes out as a template message for users without a real email.
func (s *WeChatTemplateNoticeService) Enabled(ctx context.Context, event string) bool {
	mpCfg, notices, err := s.loadConfig(ctx)
	if err != nil || !mpCfg.MPReady {
		return false
	}
	n := notices[event]
	return n.Enabled && n.TemplateID != "" && len(n.Fields) > 0
}

// OpenIDForUser returns the user's openid for the configured 公众号 ("" when there is none).
func (s *WeChatTemplateNoticeService) OpenIDForUser(ctx context.Context, appID string, userID int64) (string, error) {
	if s == nil || s.client == nil || userID <= 0 || strings.TrimSpace(appID) == "" {
		return "", nil
	}
	channel, err := s.client.AuthIdentityChannel.Query().
		Where(
			authidentitychannel.ProviderTypeEQ("wechat"),
			authidentitychannel.ChannelEQ("mp"),
			authidentitychannel.ChannelAppIDEQ(strings.TrimSpace(appID)),
			authidentitychannel.HasIdentityWith(authidentity.UserIDEQ(userID)),
		).
		Order(dbent.Desc(authidentitychannel.FieldUpdatedAt)).
		First(ctx)
	if dbent.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(channel.ChannelSubject), nil
}

// Deliver sends event to userID's openid. vars are the event's placeholder values; site_name,
// username and now are added here.
func (s *WeChatTemplateNoticeService) Deliver(ctx context.Context, event string, userID int64, vars map[string]string) (WeChatNoticeOutcome, error) {
	mpCfg, notices, err := s.loadConfig(ctx)
	if err != nil {
		return WeChatNoticeNotApplicable, nil
	}
	n := notices[event]
	if !mpCfg.MPReady || !n.Enabled || n.TemplateID == "" || len(n.Fields) == 0 {
		return WeChatNoticeNotApplicable, nil
	}
	openid, err := s.OpenIDForUser(ctx, mpCfg.AppID, userID)
	if err != nil {
		return WeChatNoticeNotApplicable, err
	}
	if openid == "" {
		return WeChatNoticeNotApplicable, nil
	}
	code, msg, err := s.send(ctx, mpCfg, n, openid, s.placeholderValues(ctx, userID, vars))
	if err != nil {
		return WeChatNoticeNotApplicable, err
	}
	switch code {
	case 0:
		return WeChatNoticeDelivered, nil
	case 43004, 43101, 40003:
		// Not following / refused, or an openid this 公众号 doesn't know (stale binding): this user only.
		logger.LegacyPrintf("service.wechat_notice", "[WeChatNotice] %s for user %d skipped: errcode %d (not following / refused / unknown openid)", event, userID, code)
		return WeChatNoticeSkipped, nil
	case 40037, 41028, 47003:
		s.warnRateLimited(event, fmt.Sprintf("[WeChatNotice] %s template rejected by WeChat: errcode %d %s — check 微信提醒 settings", event, code, msg))
		return WeChatNoticeDeferred, nil
	}
	return WeChatNoticeNotApplicable, weChatMPAPIErrorFor(code, msg)
}

// weChatCustomSendAPI sends a 客服消息. WeChat only delivers it within 48 hours of the user's last
// interaction with the 公众号 (following it, scanning its QR code, sending it a message), which covers
// the QR / 发「登录」 sign-ups a day earlier; 网页授权 sign-ups inside WeChat may have none.
const weChatCustomSendAPI = "/cgi-bin/message/custom/send"

// SendCustomText sends text (may contain <a href> links) to userID's openid as a 客服消息. It needs no
// template, so it is ready as soon as the 公众号 AppID / AppSecret are; it is not gated by 微信提醒.
func (s *WeChatTemplateNoticeService) SendCustomText(ctx context.Context, userID int64, text string) (WeChatNoticeOutcome, error) {
	if s == nil || s.mp == nil {
		return WeChatNoticeNotApplicable, nil
	}
	mpCfg, err := s.mp.Config(ctx)
	if err != nil || !mpCfg.MPReady {
		return WeChatNoticeNotApplicable, nil
	}
	openid, err := s.OpenIDForUser(ctx, mpCfg.AppID, userID)
	if err != nil {
		return WeChatNoticeNotApplicable, err
	}
	if openid == "" {
		return WeChatNoticeNotApplicable, nil
	}
	body := map[string]any{"touser": openid, "msgtype": "text", "text": map[string]string{"content": text}}
	_, err = s.mp.callAPI(ctx, mpCfg, http.MethodPost, weChatCustomSendAPI, nil, body)
	if err == nil {
		return WeChatNoticeDelivered, nil
	}
	code, msg, ok := weChatMPAPIErrorCode(err)
	if !ok {
		return WeChatNoticeNotApplicable, err
	}
	switch code {
	case 45015, 45047, 43004, 43101, 40003, 48001:
		// Outside the 48-hour window, too many messages, not following / refused, unknown openid,
		// or no 客服 permission: this user can't get it now.
		logger.LegacyPrintf("service.wechat_notice", "[WeChatNotice] custom message to user %d skipped: errcode %d %s", userID, code, msg)
		return WeChatNoticeSkipped, nil
	}
	return WeChatNoticeNotApplicable, weChatMPAPIErrorFor(code, msg)
}

// warnRateLimited logs a warning at most once per hour per key.
func (s *WeChatTemplateNoticeService) warnRateLimited(key, msg string) bool {
	s.warnMu.Lock()
	defer s.warnMu.Unlock()
	now := s.now()
	if last, ok := s.lastWarn[key]; ok && now.Sub(last) < weChatNoticeWarnInterval {
		return false
	}
	s.lastWarn[key] = now
	logger.LegacyPrintf("service.wechat_notice", "%s", msg)
	return true
}

func (s *WeChatTemplateNoticeService) placeholderValues(ctx context.Context, userID int64, vars map[string]string) map[string]string {
	out := map[string]string{
		"site_name": "HiveGPT",
		"now":       s.now().Format(weChatNoticeTimeLayout),
	}
	if s.settings != nil && s.settings.settingRepo != nil {
		if name, err := s.settings.settingRepo.GetValue(ctx, SettingKeySiteName); err == nil && strings.TrimSpace(name) != "" {
			out["site_name"] = strings.TrimSpace(name)
		}
	}
	if s.client != nil && userID > 0 {
		if u, err := s.client.User.Get(ctx, userID); err == nil {
			out["username"] = u.Username
			out["balance"] = FormatWeChatNoticeUSD(u.Balance)
		}
	}
	for k, v := range vars {
		out[k] = v
	}
	return out
}

// send posts the template message and returns WeChat's errcode / errmsg (err only for transport
// or token problems).
func (s *WeChatTemplateNoticeService) send(ctx context.Context, mpCfg WeChatMPConfig, n WeChatTemplateNoticeConfig, openid string, vars map[string]string) (int64, string, error) {
	data := make(map[string]map[string]string, len(n.Fields))
	for key, tmpl := range n.Fields {
		data[key] = map[string]string{"value": truncateWeChatNoticeValue(key, fillWeChatNoticeValue(tmpl, vars))}
	}
	body := map[string]any{
		"touser":      openid,
		"template_id": n.TemplateID,
		"data":        data,
	}
	if n.URLPath != "" {
		body["url"] = strings.TrimRight(mpCfg.SiteURL, "/") + n.URLPath
	}
	_, err := s.mp.callAPI(ctx, mpCfg, http.MethodPost, weChatNoticeTemplateSendAPI, nil, body)
	if err == nil {
		return 0, "ok", nil
	}
	if code, msg, ok := weChatMPAPIErrorCode(err); ok {
		return code, msg, nil
	}
	return 0, "", err
}

// WeChatNoticeTestResult is what the admin test shows.
type WeChatNoticeTestResult struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	Message string `json:"message"`
}

var (
	ErrWeChatNoticeNoOpenID = infraerrors.BadRequest("WECHAT_NOTICE_NO_OPENID",
		"你的账号还没有绑定公众号的微信，请先在个人资料里绑定微信（Your account has no WeChat bound to the official account）")
	ErrWeChatNoticeNoTemplate = infraerrors.BadRequest("WECHAT_NOTICE_NO_TEMPLATE",
		"请先填写这个提醒的模板 ID 和字段（Fill in template_id and fields first）")
	ErrWeChatNoticeUnknownEvent = infraerrors.BadRequest("WECHAT_NOTICE_INVALID", "不支持的提醒类型（Unknown notice event）")
)

// weChatNoticeSampleVars are the fake values of the admin test.
var weChatNoticeSampleVars = map[string]string{
	"balance":    "$1.23",
	"threshold":  "$5.00",
	"plan_name":  "示例套餐",
	"expires_at": "2026-12-31 23:59",
	"days_left":  "3",
	"order_no":   "TEST20261016001",
	"product":    "余额充值",
	"amount_cny": "¥50.00",
	"created_at": "2026-10-16 10:00",
}

// SendTest sends a sample of event to callerUserID's own openid (the saved template, enabled or not).
func (s *WeChatTemplateNoticeService) SendTest(ctx context.Context, event string, callerUserID int64) (*WeChatNoticeTestResult, error) {
	if _, ok := WeChatNoticePlaceholders[event]; !ok {
		return nil, ErrWeChatNoticeUnknownEvent
	}
	mpCfg, notices, err := s.loadConfig(ctx)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("WECHAT_NOTICE_NOT_READY", "微信提醒未就绪（WeChat notices not ready）").WithCause(err)
	}
	if !mpCfg.MPReady {
		return nil, infraerrors.BadRequest("WECHAT_MP_NOT_CONFIGURED", "公众号的 AppID / AppSecret 还没配置（WeChat official account not configured）")
	}
	n := notices[event]
	if n.TemplateID == "" || len(n.Fields) == 0 {
		return nil, ErrWeChatNoticeNoTemplate
	}
	openid, err := s.OpenIDForUser(ctx, mpCfg.AppID, callerUserID)
	if err != nil {
		return nil, infraerrors.InternalServer("WECHAT_NOTICE_LOOKUP_FAILED", "查找绑定的微信失败（failed to look up WeChat binding）").WithCause(err)
	}
	if openid == "" {
		return nil, ErrWeChatNoticeNoOpenID
	}
	vars := map[string]string{}
	for k, v := range weChatNoticeSampleVars {
		vars[k] = v
	}
	code, msg, err := s.send(ctx, mpCfg, n, openid, s.placeholderValues(ctx, callerUserID, vars))
	if err != nil {
		return nil, err
	}
	return &WeChatNoticeTestResult{ErrCode: code, ErrMsg: msg, Message: weChatNoticeErrMessage(code, msg)}, nil
}

func weChatNoticeErrMessage(code int64, msg string) string {
	switch code {
	case 0:
		return "发送成功，请在微信里查看（Sent）"
	case 43004:
		return "你还没关注公众号，或已取消关注（Not following the official account）"
	case 43101:
		return "你拒绝接收这个公众号的消息（Messages refused by the user）"
	case 40037:
		return "模板 ID 不对，请在公众平台「模板消息」里复制（Invalid template_id）"
	case 41028, 47003:
		return "模板字段或取值不符合要求，请对照模板详情检查字段名和长度（Invalid template fields）"
	case 40003:
		return "openid 不属于这个公众号（Invalid openid）"
	case 45009:
		return "今天的接口调用次数用完了（API daily quota reached）"
	}
	return fmt.Sprintf("微信返回错误 %d：%s（WeChat error %d: %s）", code, msg, code, msg)
}

// FormatWeChatNoticeUSD formats a balance for a notice, e.g. "$1.23".
func FormatWeChatNoticeUSD(v float64) string {
	return "$" + strconv.FormatFloat(v, 'f', 2, 64)
}

// FormatWeChatNoticeCNY formats an order amount, e.g. "¥50.00".
func FormatWeChatNoticeCNY(v float64) string {
	return "¥" + strconv.FormatFloat(v, 'f', 2, 64)
}

// FormatWeChatNoticeTime formats a time for a notice.
func FormatWeChatNoticeTime(t time.Time) string {
	return t.Format(weChatNoticeTimeLayout)
}
