package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// HiveGPT 桌面助手 sign-in, device-authorization style:
//
//  1. The app calls Start with its device id and name. It gets a public ticket id, shown as a QR of
//     <site>/desktop-login?t=<ticket>, and a secret device code that never leaves the app.
//  2. The user scans the QR with WeChat. The phone opens the site, signs in with the normal web
//     login (WeChat in-app sign-in, sign-up, invites, attribution: all reused), and confirms.
//  3. The app polls with ticket id + device code. Once approved, one poll claims the ticket and gets
//     a device key: 「桌面助手 · <device> #<id>」, reused on later sign-ins from the same device,
//     placed in the user's OpenAI subscription group (trial or paid) or else the pay-as-you-go group.
//
// Someone who only sees the QR can approve it for their own account, but cannot poll for the key.
//
// Phishing (someone sends the victim *their own* computer's QR) is made harder: the page compares the
// network the computer asked from with the phone's, and approving a computer on another network needs
// an explicit 「这台电脑就在我面前」 (checked here, not just in the page). Device names that pose as
// HiveGPT or an official check are replaced.

const (
	DesktopLoginStatusPending  = "pending"
	DesktopLoginStatusApproved = "approved"
	DesktopLoginStatusRejected = "rejected"
	DesktopLoginStatusExpired  = "expired"
	DesktopLoginStatusDone     = "done"

	DesktopKeyNamePrefix = "桌面助手 · "

	desktopLoginTicketTTL   = 10 * time.Minute
	desktopLoginClaimTTL    = time.Minute
	DesktopLoginPollSeconds = 2
	desktopDeviceNameMax    = 40
	desktopKeyTagLen        = 8
	desktopLoginKeyPrefix   = "desktop_login:"
	desktopLoginClaimPrefix = "desktop_login_claim:"
	desktopLoginPath        = "/desktop-login?t="
)

var (
	desktopTicketIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)
	desktopDeviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

	ErrDesktopLoginNotFound   = infraerrors.NotFound("DESKTOP_LOGIN_NOT_FOUND", "登录请求不存在或已过期，请回到电脑上刷新二维码（Desktop sign-in request not found or expired）")
	ErrDesktopLoginBadDevice  = infraerrors.BadRequest("DESKTOP_LOGIN_BAD_DEVICE", "设备信息不完整，请重启桌面助手后重试（Invalid device id）")
	ErrDesktopLoginBadCode    = infraerrors.NotFound("DESKTOP_LOGIN_BAD_CODE", "登录请求不存在或不属于这台电脑，请刷新二维码（Device code does not match）")
	ErrDesktopLoginNotPending = infraerrors.Conflict("DESKTOP_LOGIN_NOT_PENDING", "这个登录请求已经处理过了，请回到电脑上查看（Sign-in request already handled）")
	ErrDesktopLoginNotNearby  = infraerrors.BadRequest("DESKTOP_LOGIN_CONFIRM_NEARBY", "这台电脑和你的手机不在同一个网络，请确认电脑就在你面前、是你自己在登录（Confirm the computer is in front of you）")

	// Names that pose as HiveGPT or an official check, to make a phishing QR look trustworthy.
	desktopNameImpersonation = regexp.MustCompile(`(?i)hive\s*gpt|官方|客服|验证|认证|审核|安全中心|管理员|领取|福利|奖励|红包|中奖|退款`)
)

// DesktopLoginStore is the Redis subset the flow needs (the 公众号 login store provides it).
type DesktopLoginStore interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

type desktopKeyStore interface {
	List(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error)
	Create(ctx context.Context, userID int64, req CreateAPIKeyRequest) (*APIKey, error)
	Update(ctx context.Context, id int64, userID int64, req UpdateAPIKeyRequest) (*APIKey, error)
}

type desktopSubscriptions interface {
	ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error)
}

type desktopUsers interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type desktopRouting interface {
	DefaultPayAsYouGoGroup(ctx context.Context, user *User, platform string) *Group
}

type DesktopLoginService struct {
	store   DesktopLoginStore
	keys    desktopKeyStore
	subs    desktopSubscriptions
	users   desktopUsers
	routing desktopRouting
	now     func() time.Time
}

func NewDesktopLoginService(store WeChatMPLoginStore, keys *APIKeyService, subs *SubscriptionService, users UserRepository, routing *BillingRouteService) *DesktopLoginService {
	return &DesktopLoginService{store: store, keys: keys, subs: subs, users: users, routing: routing, now: time.Now}
}

type desktopTicket struct {
	DeviceCodeHash string `json:"device_code_hash"`
	DeviceID       string `json:"device_id"`
	DeviceName     string `json:"device_name"`
	Status         string `json:"status"`
	UserID         int64  `json:"user_id,omitempty"`
	CreatedAt      int64  `json:"created_at"`
	// RequestIP is where the computer asked from (for the same-network check; never shown in full).
	RequestIP string `json:"request_ip,omitempty"`
}

type DesktopLoginStarted struct {
	TicketID     string `json:"ticket_id"`
	DeviceCode   string `json:"device_code"`
	LoginPath    string `json:"login_path"`
	ExpiresIn    int    `json:"expires_in"`
	PollInterval int    `json:"poll_interval"`
}

type DesktopLoginInfo struct {
	DeviceName string `json:"device_name"`
	Status     string `json:"status"`
	ExpiresIn  int    `json:"expires_in"`
	// SameNetwork: the computer asked from the phone's network (same IP, IPv4 /24 or IPv6 /48).
	// When false, approving needs confirm_nearby.
	SameNetwork bool `json:"same_network"`
	// ComputerNetwork is the computer's address with the last part hidden, e.g. 203.0.113.*.
	ComputerNetwork string `json:"computer_network"`
	// StartedAgo is how many seconds ago the computer asked.
	StartedAgo int `json:"started_ago"`
}

type DesktopLoginPollResult struct {
	Status    string `json:"status"`
	APIKey    string `json:"api_key,omitempty"`
	KeyName   string `json:"key_name,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	UserName  string `json:"user_name,omitempty"`
}

func desktopRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashDeviceCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// cleanDeviceName keeps printable characters and caps the length; empty becomes 「我的电脑」.
func cleanDeviceName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if unicode.IsPrint(r) && r != '#' {
			_, _ = b.WriteRune(r)
		}
	}
	out := []rune(strings.TrimSpace(b.String()))
	if len(out) > desktopDeviceNameMax {
		out = out[:desktopDeviceNameMax]
	}
	if len(out) == 0 {
		return "我的电脑"
	}
	if desktopNameImpersonation.MatchString(string(out)) {
		return "未命名电脑"
	}
	return string(out)
}

// desktopNetwork is the part of an address that identifies the network: IPv4 /24, IPv6 /48.
func desktopNetwork(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String()
}

func desktopSameNetwork(a, b string) bool {
	na, nb := desktopNetwork(a), desktopNetwork(b)
	return na != "" && na == nb
}

// desktopMaskedNetwork shows where a computer is without its full address: 203.0.113.* / 2001:db8:1::*.
func desktopMaskedNetwork(ip string) string {
	n := desktopNetwork(ip)
	if n == "" {
		return ""
	}
	if strings.Contains(n, ":") {
		return strings.TrimSuffix(n, "::") + "::*"
	}
	return strings.TrimSuffix(n, ".0") + ".*"
}

func desktopKeyTag(deviceID string) string {
	id := strings.ToLower(strings.ReplaceAll(deviceID, "-", ""))
	if len(id) > desktopKeyTagLen {
		id = id[:desktopKeyTagLen]
	}
	return " #" + id
}

func (s *DesktopLoginService) Start(ctx context.Context, deviceID, deviceName, requestIP string) (*DesktopLoginStarted, error) {
	if !desktopDeviceIDPattern.MatchString(deviceID) {
		return nil, ErrDesktopLoginBadDevice
	}
	ticketID, err := desktopRandomHex(16)
	if err != nil {
		return nil, err
	}
	code, err := desktopRandomHex(32)
	if err != nil {
		return nil, err
	}
	t := desktopTicket{
		DeviceCodeHash: hashDeviceCode(code),
		DeviceID:       deviceID,
		DeviceName:     cleanDeviceName(deviceName),
		Status:         DesktopLoginStatusPending,
		CreatedAt:      s.now().Unix(),
		RequestIP:      strings.TrimSpace(requestIP),
	}
	if err := s.save(ctx, ticketID, t); err != nil {
		return nil, err
	}
	return &DesktopLoginStarted{
		TicketID:     ticketID,
		DeviceCode:   code,
		LoginPath:    desktopLoginPath + ticketID,
		ExpiresIn:    int(desktopLoginTicketTTL.Seconds()),
		PollInterval: DesktopLoginPollSeconds,
	}, nil
}

func (s *DesktopLoginService) save(ctx context.Context, ticketID string, t desktopTicket) error {
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	ttl := desktopLoginTicketTTL - s.now().Sub(time.Unix(t.CreatedAt, 0))
	if ttl <= 0 {
		return ErrDesktopLoginNotFound
	}
	return s.store.Set(ctx, desktopLoginKeyPrefix+ticketID, string(raw), ttl)
}

func (s *DesktopLoginService) load(ctx context.Context, ticketID string) (*desktopTicket, error) {
	if !desktopTicketIDPattern.MatchString(ticketID) {
		return nil, ErrDesktopLoginNotFound
	}
	raw, ok, err := s.store.Get(ctx, desktopLoginKeyPrefix+ticketID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var t desktopTicket
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, nil
	}
	return &t, nil
}

func (s *DesktopLoginService) expiresIn(t *desktopTicket) int {
	left := desktopLoginTicketTTL - s.now().Sub(time.Unix(t.CreatedAt, 0))
	if left < 0 {
		return 0
	}
	return int(left.Seconds())
}

// Info is what the signed-in phone page shows before the user confirms; viewerIP is the phone's.
func (s *DesktopLoginService) Info(ctx context.Context, ticketID, viewerIP string) (*DesktopLoginInfo, error) {
	t, err := s.load(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrDesktopLoginNotFound
	}
	ago := int(s.now().Sub(time.Unix(t.CreatedAt, 0)).Seconds())
	if ago < 0 {
		ago = 0
	}
	return &DesktopLoginInfo{
		DeviceName: t.DeviceName, Status: t.Status, ExpiresIn: s.expiresIn(t),
		SameNetwork: desktopSameNetwork(t.RequestIP, viewerIP), ComputerNetwork: desktopMaskedNetwork(t.RequestIP),
		StartedAgo: ago,
	}, nil
}

// Decide records the signed-in user's answer for a pending ticket. Approving a computer that asked
// from another network than the phone's (viewerIP) needs confirmedNearby.
func (s *DesktopLoginService) Decide(ctx context.Context, ticketID string, userID int64, approve bool, viewerIP string, confirmedNearby bool) (string, error) {
	t, err := s.load(ctx, ticketID)
	if err != nil {
		return "", err
	}
	if t == nil {
		return "", ErrDesktopLoginNotFound
	}
	if t.Status != DesktopLoginStatusPending {
		return "", ErrDesktopLoginNotPending
	}
	if approve && !confirmedNearby && !desktopSameNetwork(t.RequestIP, viewerIP) {
		return "", ErrDesktopLoginNotNearby
	}
	if approve {
		t.Status = DesktopLoginStatusApproved
		t.UserID = userID
	} else {
		t.Status = DesktopLoginStatusRejected
	}
	if err := s.save(ctx, ticketID, *t); err != nil {
		return "", err
	}
	return t.Status, nil
}

// Poll is called by the app every few seconds. An approved ticket is claimed by exactly one poll,
// which issues the device key and deletes the ticket.
func (s *DesktopLoginService) Poll(ctx context.Context, ticketID, deviceCode string) (*DesktopLoginPollResult, error) {
	t, err := s.load(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return &DesktopLoginPollResult{Status: DesktopLoginStatusExpired}, nil
	}
	if subtle.ConstantTimeCompare([]byte(hashDeviceCode(deviceCode)), []byte(t.DeviceCodeHash)) != 1 {
		return nil, ErrDesktopLoginBadCode
	}
	switch t.Status {
	case DesktopLoginStatusPending:
		return &DesktopLoginPollResult{Status: DesktopLoginStatusPending}, nil
	case DesktopLoginStatusRejected:
		_ = s.store.Del(ctx, desktopLoginKeyPrefix+ticketID)
		return &DesktopLoginPollResult{Status: DesktopLoginStatusRejected}, nil
	case DesktopLoginStatusApproved:
	default:
		return &DesktopLoginPollResult{Status: DesktopLoginStatusExpired}, nil
	}

	claimed, err := s.store.SetNX(ctx, desktopLoginClaimPrefix+ticketID, "1", desktopLoginClaimTTL)
	if err != nil {
		return nil, err
	}
	if !claimed {
		// Another poll is issuing the key right now.
		return &DesktopLoginPollResult{Status: DesktopLoginStatusApproved}, nil
	}
	key, groupName, err := s.issueKey(ctx, t.UserID, t.DeviceID, t.DeviceName)
	if err != nil {
		_ = s.store.Del(ctx, desktopLoginClaimPrefix+ticketID)
		return nil, err
	}
	_ = s.store.Del(ctx, desktopLoginKeyPrefix+ticketID)

	res := &DesktopLoginPollResult{Status: DesktopLoginStatusDone, APIKey: key.Key, KeyName: key.Name, GroupName: groupName}
	if u, err := s.users.GetByID(ctx, t.UserID); err == nil && u != nil {
		res.UserName = desktopUserName(u)
	}
	return res, nil
}

func desktopUserName(u *User) string {
	if name := strings.TrimSpace(u.Username); name != "" {
		return name
	}
	email := strings.TrimSpace(u.Email)
	if at := strings.IndexByte(email, '@'); at > 1 {
		return email[:1] + "***" + email[at:]
	}
	return ""
}

// desktopGroup picks where the device key lives: the user's OpenAI subscription that lasts longest
// (the sign-up Codex trial counts, including one that starts at the first call), else the
// pay-as-you-go OpenAI group. Nil means no group (the key service then applies its default).
func (s *DesktopLoginService) desktopGroup(ctx context.Context, userID int64) (*int64, string, error) {
	subs, err := s.subs.ListActiveUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	var openai []UserSubscription
	for _, sub := range subs {
		if sub.Group != nil && sub.Group.Platform == PlatformOpenAI {
			openai = append(openai, sub)
		}
	}
	if len(openai) > 0 {
		sort.SliceStable(openai, func(i, j int) bool { return openai[i].ExpiresAt.After(openai[j].ExpiresAt) })
		id := openai[0].GroupID
		return &id, openai[0].Group.Name, nil
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if g := s.routing.DefaultPayAsYouGoGroup(ctx, user, PlatformOpenAI); g != nil {
		id := g.ID
		return &id, g.Name, nil
	}
	return nil, "", nil
}

// issueKey reuses this device's key (moving it to the current group and re-enabling it: signing in
// again is the user's consent) or creates one.
func (s *DesktopLoginService) issueKey(ctx context.Context, userID int64, deviceID, deviceName string) (*APIKey, string, error) {
	groupID, groupName, err := s.desktopGroup(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	tag := desktopKeyTag(deviceID)
	existing, _, err := s.keys.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{Search: strings.TrimSpace(tag)})
	if err != nil {
		return nil, "", err
	}
	name := DesktopKeyNamePrefix + deviceName + tag
	for i := range existing {
		k := existing[i]
		if !strings.HasPrefix(k.Name, DesktopKeyNamePrefix) || !strings.HasSuffix(k.Name, tag) {
			continue
		}
		sameGroup := (k.GroupID == nil && groupID == nil) || (k.GroupID != nil && groupID != nil && *k.GroupID == *groupID)
		if sameGroup && k.Status == StatusAPIKeyActive && k.Name == name {
			return &k, groupName, nil
		}
		active := StatusAPIKeyActive
		req := UpdateAPIKeyRequest{Status: &active, Name: &name}
		if groupID != nil {
			req.GroupID = groupID
		}
		updated, err := s.keys.Update(ctx, k.ID, userID, req)
		if err != nil {
			return nil, "", err
		}
		return updated, groupName, nil
	}
	created, err := s.keys.Create(ctx, userID, CreateAPIKeyRequest{Name: name, GroupID: groupID})
	if err != nil {
		return nil, "", err
	}
	return created, groupName, nil
}
