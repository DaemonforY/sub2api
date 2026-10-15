package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 公众号消息服务器 / 扫码登录 / 验证码登录的设置。每个终端都用同一个已认证服务号（appid 就是
// wechat_connect_mp_app_id），所以同一个人在任何终端拿到的 openid 都一样，进的也是同一个账号。

const (
	defaultWeChatMPLoginKeywords = "登录,验证码,login"
	defaultWeChatMPReplyScanOK   = "已确认登录 HiveGPT，请回到浏览器继续。\n想用微信昵称和头像？{sync_link}\n如果不是你本人在登录，请忽略。"
	defaultWeChatMPReplyCode     = "想登录 HiveGPT？请在网页上点「微信登录」，把页面上的 6 位验证码发到这里即可。\n网站：{site_url}"
	defaultWeChatMPReplyWelcome  = "欢迎关注 HiveGPT！\n在浏览器里登录网站时点「微信登录」，扫码或把页面上的 6 位验证码发到这里即可。\n网站：{site_url}"
	defaultWeChatMPSiteURL       = "https://hivegpt.cn"

	weChatMPAESKeyLength           = 43
	weChatDomainVerifyMaxFiles     = 10
	weChatDomainVerifyMaxContent   = 256
	weChatMPMaxReplyLength         = 600
	weChatMPMaxKeywordsSettingSize = 200
)

var weChatDomainVerifyFilenamePattern = regexp.MustCompile(`^MP_verify_[A-Za-z0-9]{8,32}\.txt$`)

// WeChatDomainVerifyFile is one 公众号「JS 接口安全域名 / 网页授权域名」 verify file served at the site root.
type WeChatDomainVerifyFile struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// IsWeChatDomainVerifyFilename reports whether name looks like a 公众号 domain verify file.
func IsWeChatDomainVerifyFilename(name string) bool {
	return weChatDomainVerifyFilenamePattern.MatchString(name)
}

// ParseWeChatDomainVerifyFiles reads the stored JSON, dropping malformed entries.
func ParseWeChatDomainVerifyFiles(raw string) []WeChatDomainVerifyFile {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []WeChatDomainVerifyFile{}
	}
	var files []WeChatDomainVerifyFile
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return []WeChatDomainVerifyFile{}
	}
	out := make([]WeChatDomainVerifyFile, 0, len(files))
	for _, f := range files {
		name := strings.TrimSpace(f.Filename)
		if !IsWeChatDomainVerifyFilename(name) {
			continue
		}
		out = append(out, WeChatDomainVerifyFile{Filename: name, Content: strings.TrimSpace(f.Content)})
	}
	return out
}

// NormalizeWeChatDomainVerifyFiles validates the admin's list before it is saved.
func NormalizeWeChatDomainVerifyFiles(files []WeChatDomainVerifyFile) ([]WeChatDomainVerifyFile, error) {
	out := make([]WeChatDomainVerifyFile, 0, len(files))
	seen := map[string]struct{}{}
	for _, f := range files {
		name := strings.TrimSpace(f.Filename)
		content := strings.TrimSpace(f.Content)
		if name == "" && content == "" {
			continue
		}
		if !IsWeChatDomainVerifyFilename(name) {
			return nil, infraerrors.BadRequest("WECHAT_DOMAIN_VERIFY_FILE_INVALID",
				fmt.Sprintf("校验文件名「%s」不对，应形如 MP_verify_xxxxxxxx.txt（Invalid verify file name）", name))
		}
		if content == "" || len(content) > weChatDomainVerifyMaxContent {
			return nil, infraerrors.BadRequest("WECHAT_DOMAIN_VERIFY_FILE_INVALID",
				fmt.Sprintf("校验文件「%s」的内容为空或太长（Invalid verify file content）", name))
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, WeChatDomainVerifyFile{Filename: name, Content: content})
	}
	if len(out) > weChatDomainVerifyMaxFiles {
		return nil, infraerrors.BadRequest("WECHAT_DOMAIN_VERIFY_FILE_INVALID",
			fmt.Sprintf("校验文件最多 %d 个（Too many verify files）", weChatDomainVerifyMaxFiles))
	}
	return out, nil
}

func marshalWeChatDomainVerifyFiles(files []WeChatDomainVerifyFile) string {
	if files == nil {
		files = []WeChatDomainVerifyFile{}
	}
	b, err := json.Marshal(files)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// ValidateWeChatMPAESKey checks an EncodingAESKey: 43 characters that decode (with a trailing "=") to 32 bytes.
func ValidateWeChatMPAESKey(key string) error {
	key = strings.TrimSpace(key)
	if _, err := decodeWeChatMPAESKey(key); err != nil {
		return infraerrors.BadRequest("WECHAT_MP_AES_KEY_INVALID", "消息加解密密钥（EncodingAESKey）应是 43 位字母和数字（Invalid EncodingAESKey）")
	}
	return nil
}

func decodeWeChatMPAESKey(key string) ([]byte, error) {
	if len(key) != weChatMPAESKeyLength {
		return nil, fmt.Errorf("encoding aes key must be %d chars", weChatMPAESKeyLength)
	}
	raw, err := base64.StdEncoding.DecodeString(key + "=")
	if err != nil {
		return nil, fmt.Errorf("decode encoding aes key: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("encoding aes key decodes to %d bytes", len(raw))
	}
	return raw, nil
}

// NormalizeWeChatMPLoginKeywords trims and de-duplicates the comma-separated keyword list.
func NormalizeWeChatMPLoginKeywords(raw string) string {
	return strings.Join(parseWeChatMPLoginKeywords(raw), ",")
}

func parseWeChatMPLoginKeywords(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '、' || r == ';' || r == '；'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

// WeChatMPFollowQRURL is the 公众号 follow QR image for an original ID (gh_…).
func WeChatMPFollowQRURL(originalID string) string {
	originalID = strings.TrimSpace(originalID)
	if originalID == "" {
		return ""
	}
	return "https://open.weixin.qq.com/qr/code?username=" + url.QueryEscape(originalID)
}

// WeChatMPConfig is the effective 公众号 message-server and login configuration.
type WeChatMPConfig struct {
	AppID     string
	AppSecret string
	// MPReady: WeChat Connect is on, the 公众号 channel is on, and its app id / secret are set.
	MPReady bool

	MessageEnabled   bool
	MessageToken     string
	MessageAESKey    string
	LoginKeywords    []string
	ReplyWelcome     string
	ReplyDefault     string
	ReplyScanOK      string
	ReplyCode        string
	AccountName      string
	OriginalID       string
	QRLoginEnabled   bool
	CodeLoginEnabled bool
	SiteURL          string
	// ForwardURL receives every message this site does not handle (empty = none).
	ForwardURL string
}

// MessageServerReady: the message endpoint answers WeChat.
func (c WeChatMPConfig) MessageServerReady() bool {
	return c.MessageEnabled && strings.TrimSpace(c.MessageToken) != ""
}

// QRLoginAvailable: PC browsers can log in by scanning the 公众号 QR.
func (c WeChatMPConfig) QRLoginAvailable() bool {
	return c.MessageServerReady() && c.MessageAESKey != "" && c.MPReady && c.QRLoginEnabled
}

// CodeLoginAvailable: browsers can log in with the code the 公众号 sends.
func (c WeChatMPConfig) CodeLoginAvailable() bool {
	return c.MessageServerReady() && c.MessageAESKey != "" && c.MPReady && c.CodeLoginEnabled
}

var weChatMPSettingKeys = []string{
	SettingKeyWeChatMPMessageEnabled,
	SettingKeyWeChatMPMessageToken,
	SettingKeyWeChatMPMessageAESKey,
	SettingKeyWeChatMPLoginKeywords,
	SettingKeyWeChatMPReplyWelcome,
	SettingKeyWeChatMPReplyDefault,
	SettingKeyWeChatMPReplyScanOK,
	SettingKeyWeChatMPReplyCode,
	SettingKeyWeChatMPAccountName,
	SettingKeyWeChatMPOriginalID,
	SettingKeyWeChatQRLoginEnabled,
	SettingKeyWeChatCodeLoginEnabled,
	SettingKeyWeChatMPMessageForwardURL,
	SettingKeyFrontendURL,
	SettingKeyAPIBaseURL,
}

var weChatConnectSettingKeys = []string{
	SettingKeyWeChatConnectEnabled,
	SettingKeyWeChatConnectAppID,
	SettingKeyWeChatConnectAppSecret,
	SettingKeyWeChatConnectOpenAppID,
	SettingKeyWeChatConnectOpenAppSecret,
	SettingKeyWeChatConnectMPAppID,
	SettingKeyWeChatConnectMPAppSecret,
	SettingKeyWeChatConnectMobileAppID,
	SettingKeyWeChatConnectMobileAppSecret,
	SettingKeyWeChatConnectOpenEnabled,
	SettingKeyWeChatConnectMPEnabled,
	SettingKeyWeChatConnectMobileEnabled,
	SettingKeyWeChatConnectMode,
	SettingKeyWeChatConnectScopes,
	SettingKeyWeChatConnectRedirectURL,
	SettingKeyWeChatConnectFrontendRedirectURL,
}

// GetWeChatMPConfig loads the effective 公众号 configuration.
func (s *SettingService) GetWeChatMPConfig(ctx context.Context) (WeChatMPConfig, error) {
	if s == nil || s.settingRepo == nil {
		return WeChatMPConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "设置服务未就绪（settings service not ready）")
	}
	keys := append(append([]string{}, weChatMPSettingKeys...), weChatConnectSettingKeys...)
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return WeChatMPConfig{}, fmt.Errorf("get wechat mp settings: %w", err)
	}
	cfg := s.weChatMPConfigFromSettings(settings)
	if cfg.SiteURL == "" && s.cfg != nil {
		cfg.SiteURL = strings.TrimSpace(s.cfg.Server.FrontendURL)
	}
	if cfg.SiteURL == "" {
		cfg.SiteURL = defaultWeChatMPSiteURL
	}
	return cfg, nil
}

func (s *SettingService) weChatMPConfigFromSettings(settings map[string]string) WeChatMPConfig {
	connect := s.effectiveWeChatConnectOAuthConfig(settings)
	appID := connect.AppIDForMode("mp")
	appSecret := connect.AppSecretForMode("mp")
	keywords := parseWeChatMPLoginKeywords(settings[SettingKeyWeChatMPLoginKeywords])
	if len(keywords) == 0 {
		keywords = parseWeChatMPLoginKeywords(defaultWeChatMPLoginKeywords)
	}
	siteURL := strings.TrimRight(strings.TrimSpace(firstNonEmpty(settings[SettingKeyFrontendURL], settings[SettingKeyAPIBaseURL])), "/")
	siteURL = strings.TrimSuffix(siteURL, "/api/v1")
	return WeChatMPConfig{
		AppID:            appID,
		AppSecret:        appSecret,
		MPReady:          connect.Enabled && connect.MPEnabled && appID != "" && appSecret != "",
		MessageEnabled:   settings[SettingKeyWeChatMPMessageEnabled] == "true",
		MessageToken:     strings.TrimSpace(settings[SettingKeyWeChatMPMessageToken]),
		MessageAESKey:    strings.TrimSpace(settings[SettingKeyWeChatMPMessageAESKey]),
		LoginKeywords:    keywords,
		ReplyWelcome:     firstNonEmpty(strings.TrimSpace(settings[SettingKeyWeChatMPReplyWelcome]), defaultWeChatMPReplyWelcome),
		ReplyDefault:     strings.TrimSpace(settings[SettingKeyWeChatMPReplyDefault]),
		ReplyScanOK:      normalizeWeChatMPReplyScanOK(settings[SettingKeyWeChatMPReplyScanOK]),
		ReplyCode:        firstNonEmpty(strings.TrimSpace(settings[SettingKeyWeChatMPReplyCode]), defaultWeChatMPReplyCode),
		AccountName:      strings.TrimSpace(settings[SettingKeyWeChatMPAccountName]),
		OriginalID:       strings.TrimSpace(settings[SettingKeyWeChatMPOriginalID]),
		QRLoginEnabled:   settings[SettingKeyWeChatQRLoginEnabled] == "true",
		CodeLoginEnabled: settings[SettingKeyWeChatCodeLoginEnabled] == "true",
		SiteURL:          siteURL,
		ForwardURL:       strings.TrimSpace(settings[SettingKeyWeChatMPMessageForwardURL]),
	}
}

const weChatDomainVerifyCacheTTL = 60 * time.Second

type cachedWeChatDomainVerifyFiles struct {
	files     map[string]string
	expiresAt time.Time
}

// GetWeChatDomainVerifyFile returns the content of a configured verify file. The list is cached in
// memory for 60 s (and dropped when the admin saves settings), so this is no DB read per request.
func (s *SettingService) GetWeChatDomainVerifyFile(ctx context.Context, filename string) (string, bool) {
	if s == nil || s.settingRepo == nil || !IsWeChatDomainVerifyFilename(filename) {
		return "", false
	}
	cached, _ := s.wechatVerifyFilesCache.Load().(*cachedWeChatDomainVerifyFiles)
	if cached == nil || time.Now().After(cached.expiresAt) {
		raw, err := s.settingRepo.GetValue(ctx, SettingKeyWeChatDomainVerifyFiles)
		if err != nil && !errors.Is(err, ErrSettingNotFound) {
			return "", false
		}
		files := map[string]string{}
		for _, f := range ParseWeChatDomainVerifyFiles(raw) {
			files[f.Filename] = f.Content
		}
		cached = &cachedWeChatDomainVerifyFiles{files: files, expiresAt: time.Now().Add(weChatDomainVerifyCacheTTL)}
		s.wechatVerifyFilesCache.Store(cached)
	}
	content, ok := cached.files[filename]
	return content, ok
}

// invalidateWeChatDomainVerifyFiles drops the cached verify files after a settings save.
func (s *SettingService) invalidateWeChatDomainVerifyFiles() {
	s.wechatVerifyFilesCache.Store((*cachedWeChatDomainVerifyFiles)(nil))
}

// weChatMPSystemSettings fills the admin view of the 公众号 settings (secrets only as *Configured).
func (s *SettingService) applyWeChatMPSystemSettings(result *SystemSettings, settings map[string]string) {
	mp := s.weChatMPConfigFromSettings(settings)
	result.WeChatMPMessageEnabled = mp.MessageEnabled
	result.WeChatMPMessageToken = mp.MessageToken
	result.WeChatMPMessageTokenConfigured = mp.MessageToken != ""
	result.WeChatMPMessageAESKey = mp.MessageAESKey
	result.WeChatMPMessageAESKeyConfigured = mp.MessageAESKey != ""
	result.WeChatMPLoginKeywords = strings.Join(mp.LoginKeywords, ",")
	result.WeChatMPReplyWelcome = mp.ReplyWelcome
	result.WeChatMPReplyDefault = mp.ReplyDefault
	result.WeChatMPReplyScanOK = mp.ReplyScanOK
	result.WeChatMPReplyCode = mp.ReplyCode
	result.WeChatMPAccountName = mp.AccountName
	result.WeChatMPOriginalID = mp.OriginalID
	result.WeChatQRLoginEnabled = mp.QRLoginEnabled
	result.WeChatCodeLoginEnabled = mp.CodeLoginEnabled
	result.WeChatDomainVerifyFiles = ParseWeChatDomainVerifyFiles(settings[SettingKeyWeChatDomainVerifyFiles])
	result.WeChatMPMessageForwardURL = mp.ForwardURL
	result.WeChatMPTemplateNotices = ParseWeChatTemplateNotices(settings[SettingKeyWeChatMPTemplateNotices])
}

// ErrWeChatLoginNeedsAESKey: QR / code login only trust 安全模式 messages.
var ErrWeChatLoginNeedsAESKey = infraerrors.BadRequest("WECHAT_MP_AES_KEY_REQUIRED", "开启扫码或验证码登录前，请先填写 EncodingAESKey 并在公众号选择安全模式（AES key required）")

// writeWeChatMPSettingUpdates validates and writes the 公众号 settings. storedAESKey is the key already
// saved (an empty key in the request keeps it).
func writeWeChatMPSettingUpdates(updates map[string]string, settings *SystemSettings, storedAESKey string) error {
	if (settings.WeChatQRLoginEnabled || settings.WeChatCodeLoginEnabled) &&
		strings.TrimSpace(settings.WeChatMPMessageAESKey) == "" && strings.TrimSpace(storedAESKey) == "" {
		return ErrWeChatLoginNeedsAESKey
	}
	updates[SettingKeyWeChatMPMessageEnabled] = strconv.FormatBool(settings.WeChatMPMessageEnabled)
	if token := strings.TrimSpace(settings.WeChatMPMessageToken); token != "" {
		if len(token) < 3 || len(token) > 32 || !isAlnum(token) {
			return infraerrors.BadRequest("WECHAT_MP_TOKEN_INVALID", "消息服务器令牌（Token）应是 3–32 位字母和数字（Invalid token）")
		}
		updates[SettingKeyWeChatMPMessageToken] = token
	}
	if key := strings.TrimSpace(settings.WeChatMPMessageAESKey); key != "" {
		if err := ValidateWeChatMPAESKey(key); err != nil {
			return err
		}
		updates[SettingKeyWeChatMPMessageAESKey] = key
	}
	keywords := NormalizeWeChatMPLoginKeywords(settings.WeChatMPLoginKeywords)
	if len(keywords) > weChatMPMaxKeywordsSettingSize {
		return infraerrors.BadRequest("WECHAT_MP_KEYWORDS_INVALID", "登录关键词太长了（Too many keywords）")
	}
	updates[SettingKeyWeChatMPLoginKeywords] = keywords
	for key, value := range map[string]string{
		SettingKeyWeChatMPReplyWelcome: settings.WeChatMPReplyWelcome,
		SettingKeyWeChatMPReplyDefault: settings.WeChatMPReplyDefault,
		SettingKeyWeChatMPReplyScanOK:  settings.WeChatMPReplyScanOK,
		SettingKeyWeChatMPReplyCode:    settings.WeChatMPReplyCode,
	} {
		value = strings.TrimSpace(value)
		if len([]rune(value)) > weChatMPMaxReplyLength {
			return infraerrors.BadRequest("WECHAT_MP_REPLY_TOO_LONG", fmt.Sprintf("自动回复最多 %d 个字（Reply too long）", weChatMPMaxReplyLength))
		}
		updates[key] = value
	}
	updates[SettingKeyWeChatMPAccountName] = strings.TrimSpace(settings.WeChatMPAccountName)
	updates[SettingKeyWeChatMPOriginalID] = strings.TrimSpace(settings.WeChatMPOriginalID)
	forwardURL := strings.TrimSpace(settings.WeChatMPMessageForwardURL)
	if forwardURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(forwardURL); err != nil {
			return infraerrors.BadRequest("WECHAT_MP_FORWARD_URL_INVALID", "消息转发地址应是完整的 http(s) 网址（Invalid forward URL）")
		}
	}
	updates[SettingKeyWeChatMPMessageForwardURL] = forwardURL
	notices, err := NormalizeWeChatTemplateNotices(settings.WeChatMPTemplateNotices)
	if err != nil {
		return err
	}
	updates[SettingKeyWeChatMPTemplateNotices] = marshalWeChatTemplateNotices(notices)
	updates[SettingKeyWeChatQRLoginEnabled] = strconv.FormatBool(settings.WeChatQRLoginEnabled)
	updates[SettingKeyWeChatCodeLoginEnabled] = strconv.FormatBool(settings.WeChatCodeLoginEnabled)
	files, err := NormalizeWeChatDomainVerifyFiles(settings.WeChatDomainVerifyFiles)
	if err != nil {
		return err
	}
	updates[SettingKeyWeChatDomainVerifyFiles] = marshalWeChatDomainVerifyFiles(files)
	return nil
}

func isAlnum(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func (s *SettingService) storedWeChatMPAESKey(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return ""
	}
	key, err := s.settingRepo.GetValue(ctx, SettingKeyWeChatMPMessageAESKey)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(key)
}

// Earlier built-in defaults of wechat_mp_reply_scan_ok. A stored copy of one of them counts as
// "never customised", so it moves to the current default (which carries {sync_link}).
var legacyWeChatMPReplyScanOKDefaults = map[string]struct{}{
	"已确认登录，请回到电脑上的网页继续。":                     {},
	"已确认登录 HiveGPT，请回到浏览器继续。如果不是你本人在登录，请忽略。": {},
}

func normalizeWeChatMPReplyScanOK(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultWeChatMPReplyScanOK
	}
	if _, legacy := legacyWeChatMPReplyScanOKDefaults[raw]; legacy {
		return defaultWeChatMPReplyScanOK
	}
	return raw
}

// WeChatSyncProfileStartPath starts 网页授权 (snsapi_userinfo) to copy the WeChat nickname and avatar
// onto the account bound to that WeChat.
const WeChatSyncProfileStartPath = "/api/v1/auth/oauth/wechat/sync-profile/start"

// weChatMPSyncProfileLink is the 「同步微信昵称和头像」 link for 公众号 text replies.
func weChatMPSyncProfileLink(siteURL string) string {
	href := strings.TrimRight(strings.TrimSpace(siteURL), "/") + WeChatSyncProfileStartPath + "?redirect=%2Fauth%2Fwechat%2Fsynced"
	return `<a href="` + href + `">同步微信昵称和头像</a>`
}

// weChatMPScanOKReply is the reply after a successful QR scan or code: the template with {sync_link}
// and {site_url} filled in; a template without {sync_link} still gets the link on its own line.
func weChatMPScanOKReply(cfg WeChatMPConfig) string {
	link := weChatMPSyncProfileLink(cfg.SiteURL)
	text := cfg.ReplyScanOK
	if strings.Contains(text, "{sync_link}") {
		text = strings.ReplaceAll(text, "{sync_link}", link)
	} else {
		text = strings.TrimRight(text, "\n") + "\n" + link
	}
	return strings.ReplaceAll(text, "{site_url}", cfg.SiteURL)
}
