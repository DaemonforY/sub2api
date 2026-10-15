package service

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 用户画像 + 风控监控: read-only views over sign-ups, usage and payments for the admin (/admin/user-insights),
// and a job that writes risk events for new suspicious accounts and mails the ops alert recipients.
// Admin accounts are excluded everywhere.

const (
	SettingKeyRiskLinkedSignup       = "growth_risk_linked_signup"       // 新注册账号与已封禁/静默账号同指纹
	SettingKeyRiskTrialDay1          = "growth_risk_trial_day1"          // 试用账号首日用完额度
	SettingKeyRiskRelayFree          = "growth_risk_relay_free"          // 中转工具 UA 消耗免费额度
	SettingKeyRiskBurstSignups       = "growth_risk_burst_signups"       // 同网段/同浏览器 24 小时内多次注册
	SettingKeyRiskBurstThreshold     = "growth_risk_burst_threshold"     // 上一条的注册数阈值
	SettingKeyRiskInviteeFingerprint = "growth_risk_invitee_fingerprint" // 被邀请人与邀请人同指纹

	riskBurstThresholdDefault = 3
	riskBurstThresholdMax     = 100

	RiskRuleLinkedSignup       = "linked_signup"
	RiskRuleTrialDay1          = "trial_day1"
	RiskRuleRelayFree          = "relay_free"
	RiskRuleBurstSignups       = "burst_signups"
	RiskRuleInviteeFingerprint = "invitee_fingerprint"

	riskMonitorInterval = 20 * time.Minute
	riskMonitorLookback = 24 * time.Hour
	riskInviteeLookback = 7 * 24 * time.Hour

	insightsDefaultDays = 30
	insightsMaxDays     = 90
)

// RiskSettings are the monitor rules on the 推广设置 page; all on by default.
type RiskSettings struct {
	RiskLinkedSignup       bool `json:"risk_linked_signup"`
	RiskTrialDay1          bool `json:"risk_trial_day1"`
	RiskRelayFree          bool `json:"risk_relay_free"`
	RiskBurstSignups       bool `json:"risk_burst_signups"`
	RiskBurstThreshold     int  `json:"risk_burst_threshold"`
	RiskInviteeFingerprint bool `json:"risk_invitee_fingerprint"`
}

func parseRiskSettings(vals map[string]string) RiskSettings {
	return RiskSettings{
		RiskLinkedSignup:       vals[SettingKeyRiskLinkedSignup] != "false",
		RiskTrialDay1:          vals[SettingKeyRiskTrialDay1] != "false",
		RiskRelayFree:          vals[SettingKeyRiskRelayFree] != "false",
		RiskBurstSignups:       vals[SettingKeyRiskBurstSignups] != "false",
		RiskBurstThreshold:     int(clampFloat(parseFloatOr(vals[SettingKeyRiskBurstThreshold], riskBurstThresholdDefault), 2, riskBurstThresholdMax)),
		RiskInviteeFingerprint: vals[SettingKeyRiskInviteeFingerprint] != "false",
	}
}

// ---------- API client classes ----------

const (
	ClientCodexDesktop = "codex_desktop"
	ClientCodexCLI     = "codex_cli"
	ClientClaudeCode   = "claude_code"
	ClientCherryStudio = "cherry_studio"
	ClientRelay        = "relay"
	ClientBrowser      = "browser"
	ClientSDK          = "sdk"
	ClientOther        = "other"
)

// relayUAPattern matches relay / reseller tools that resell API access (AxonHub, new-api, …) and the
// bare HTTP libraries they are built on. Kept in sync with ClassifyClientUA.
const relayUAPattern = `axonhub|relaytest|codexplusplus|new-api|one-api|veloera|done-hub|uni-api|gpt-load|cliproxy|claude-relay|go-http-client|fasthttp`

var (
	relayUARe   = regexp.MustCompile(`(?i)` + relayUAPattern)
	browserUARe = regexp.MustCompile(`(?i)mozilla/|chrome/|safari/|firefox/`)
	sdkUARe     = regexp.MustCompile(`(?i)openai/|anthropic/|python|node|axios|okhttp|curl/|httpx|aiohttp|undici|java/|ruby|dart`)
)

// ClassifyClientUA groups a request's User-Agent into a client type.
func ClassifyClientUA(ua string) string {
	l := strings.ToLower(strings.TrimSpace(ua))
	switch {
	case l == "":
		return ClientOther
	case relayUARe.MatchString(l):
		return ClientRelay
	case strings.Contains(l, "codex desktop") || strings.Contains(l, "codex_desktop") || strings.Contains(l, "codex-desktop"):
		return ClientCodexDesktop
	case strings.Contains(l, "codex_cli") || strings.Contains(l, "codex-cli") || strings.HasPrefix(l, "codex_") || strings.HasPrefix(l, "codex/"):
		return ClientCodexCLI
	case strings.Contains(l, "claude-cli") || strings.Contains(l, "claude-code"):
		return ClientClaudeCode
	case strings.Contains(l, "cherrystudio") || strings.Contains(l, "cherry-studio") || strings.Contains(l, "cherry studio"):
		return ClientCherryStudio
	case browserUARe.MatchString(l):
		return ClientBrowser
	case sdkUARe.MatchString(l):
		return ClientSDK
	}
	return ClientOther
}

// ---------- types ----------

type RiskHit struct {
	UserID int64
	Detail string
}

type RiskEvent struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	UserEmail string    `json:"user_email,omitempty"`
	Username  string    `json:"username,omitempty"`
	Rule      string    `json:"rule"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// InsightsUserRef is a user in a list, with what the admin needs to judge it at a glance.
type InsightsUserRef struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Shadowed  bool      `json:"shadowed"`
	PaidCNY   float64   `json:"paid_cny"`  // lifetime, completed provider payments
	SpendUSD  float64   `json:"spend_usd"` // usage cost in the window
}

type InsightsSegments struct {
	Inactive int64 `json:"inactive"` // never used the API
	Trial    int64 `json:"trial"`    // used it, never paid
	Paying   int64 `json:"paying"`   // paid, active in the last 14 days (or paid in the last 30)
	Churned  int64 `json:"churned"`  // paid before, quiet since
}

type InsightsSignupDay struct {
	Date string `json:"date"`
	InsightsSegments
}

type InsightsSpendDay struct {
	Date    string  `json:"date"`
	FreeUSD float64 `json:"free_usd"` // usage by accounts that never paid
	UsedUSD float64 `json:"used_usd"` // all usage
	PaidCNY float64 `json:"paid_cny"` // completed provider payments
}

type InsightsUARow struct {
	UserID    int64
	UserAgent string
	Requests  int64
	CostUSD   float64
}

type InsightsClient struct {
	Class    string  `json:"class"`
	Requests int64   `json:"requests"`
	Users    int     `json:"users"`
	CostUSD  float64 `json:"cost_usd"`
}

type InsightsCluster struct {
	Kind    string            `json:"kind"` // visitor | ip_prefix
	Key     string            `json:"key"`
	UserIDs []int64           `json:"-"`
	Users   []InsightsUserRef `json:"users"`
	FreeUSD float64           `json:"free_usd"` // window spend of the members that never paid
	PaidCNY float64           `json:"paid_cny"`
}

type InsightsRelayUser struct {
	User      InsightsUserRef `json:"user"`
	UserAgent string          `json:"user_agent"`
	Requests  int64           `json:"requests"`
	CostUSD   float64         `json:"cost_usd"`
}

type UserInsightsOverview struct {
	Days       int                 `json:"days"`
	Since      time.Time           `json:"since"`
	Segments   InsightsSegments    `json:"segments"`
	Signups    []InsightsSignupDay `json:"signups"`
	Spend      []InsightsSpendDay  `json:"spend"`
	Clients    []InsightsClient    `json:"clients"`
	Clusters   []InsightsCluster   `json:"clusters"`
	RelayUsers []InsightsRelayUser `json:"relay_users"`
}

type InsightsCount struct {
	Key      string    `json:"key"`
	Count    int64     `json:"count"`
	CostUSD  float64   `json:"cost_usd,omitempty"`
	LastSeen time.Time `json:"last_seen"`
}

type InsightsClientUA struct {
	Class     string    `json:"class"`
	UserAgent string    `json:"user_agent"`
	Requests  int64     `json:"requests"`
	CostUSD   float64   `json:"cost_usd"`
	LastSeen  time.Time `json:"last_seen"`
}

type InsightsPayment struct {
	ID          int64      `json:"id"`
	OrderType   string     `json:"order_type"`
	PaymentType string     `json:"payment_type"`
	Amount      float64    `json:"amount"`
	PayAmount   float64    `json:"pay_amount"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

type InsightsSubscription struct {
	ID         int64     `json:"id"`
	GroupName  string    `json:"group_name"`
	Status     string    `json:"status"`
	Free       bool      `json:"free"` // the sign-up default subscription
	StartsAt   time.Time `json:"starts_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	DailyUsage float64   `json:"daily_usage_usd"`
	DailyLimit *float64  `json:"daily_limit_usd,omitempty"`
}

type InsightsSpend struct {
	TotalUSD      float64 `json:"total_usd"`
	FreeSubUSD    float64 `json:"free_sub_usd"`    // through the sign-up default subscription
	PaidSubUSD    float64 `json:"paid_sub_usd"`    // through other subscriptions
	BalanceUSD    float64 `json:"balance_usd"`     // billed to balance
	PaidCNY       float64 `json:"paid_cny"`        // lifetime provider payments
	HasRealPaying bool    `json:"has_real_paying"` // ever paid through a provider
}

type InsightsLinked struct {
	User InsightsUserRef `json:"user"`
	Via  []string        `json:"via"` // visitor | ip_prefix | api_ip | user_agent | inviter | invitee
}

type InsightsAttribution struct {
	Source       string `json:"source"`
	Medium       string `json:"medium"`
	Campaign     string `json:"campaign"`
	AffCode      string `json:"aff_code"`
	ReferrerHost string `json:"referrer_host"`
	LandingPath  string `json:"landing_path"`
}

type UserInsightsProfile struct {
	User           InsightsUserRef        `json:"user"`
	Role           string                 `json:"role"`
	SignupSource   string                 `json:"signup_source"`
	Balance        float64                `json:"balance"`
	Attribution    *InsightsAttribution   `json:"attribution,omitempty"`
	VisitorID      string                 `json:"visitor_id"`
	SignupIPPrefix string                 `json:"signup_ip_prefix"`
	TrialGranted   bool                   `json:"trial_granted"`
	WebPrefixes    []InsightsCount        `json:"web_prefixes"`
	APIIPs         []InsightsCount        `json:"api_ips"`
	Clients        []InsightsClientUA     `json:"clients"`
	Groups         []InsightsCount        `json:"groups"`
	Models         []InsightsCount        `json:"models"`
	Spend          InsightsSpend          `json:"spend"`
	Payments       []InsightsPayment      `json:"payments"`
	Subscriptions  []InsightsSubscription `json:"subscriptions"`
	InviteUp       []InsightsUserRef      `json:"invite_up"`
	InviteDown     []InsightsUserRef      `json:"invite_down"`
	Linked         []InsightsLinked       `json:"linked"`
	Shadow         *UserShadow            `json:"shadow,omitempty"`
	Events         []RiskEvent            `json:"events"`
	RiskScore      int                    `json:"risk_score"`
	RiskReasons    []string               `json:"risk_reasons"`
}

type UserRiskRepository interface {
	// Monitor rules: hits since `since`, admins excluded.
	LinkedFlaggedSignups(ctx context.Context, since time.Time) ([]RiskHit, error)
	TrialDay1Exhausted(ctx context.Context, since time.Time) ([]RiskHit, error)
	RelayFreeSpend(ctx context.Context, since time.Time, uaPattern string) ([]RiskHit, error)
	BurstSignups(ctx context.Context, since time.Time, threshold int) ([]RiskHit, error)
	InviteeSharesInviterFingerprint(ctx context.Context, since time.Time) ([]RiskHit, error)
	// InsertRiskEvents stores hits and returns only the ones that are new for (user, rule).
	InsertRiskEvents(ctx context.Context, rule string, hits []RiskHit) ([]RiskEvent, error)
	ListRiskEvents(ctx context.Context, userID int64, page, pageSize int) ([]RiskEvent, int64, error)

	// Dashboard.
	InsightsSegments(ctx context.Context, since time.Time) (InsightsSegments, []InsightsSignupDay, error)
	InsightsSpend(ctx context.Context, since time.Time) ([]InsightsSpendDay, error)
	InsightsUserAgents(ctx context.Context, since time.Time) ([]InsightsUARow, error)
	InsightsClusters(ctx context.Context, since time.Time) ([]InsightsCluster, error)
	UserRefs(ctx context.Context, ids []int64, since time.Time) (map[int64]InsightsUserRef, error)
	FindUserID(ctx context.Context, q string) (int64, error)
	UserProfile(ctx context.Context, userID int64, since time.Time) (*UserInsightsProfile, error)
}

var ErrInsightsUserNotFound = infraerrors.NotFound("INSIGHTS_USER_NOT_FOUND", "没有找到这个用户（User not found）")

// UserRiskService runs the dashboard and the monitor.
type UserRiskService struct {
	repo   UserRiskRepository
	shadow *UserShadowService
	growth *GrowthService
	ops    *OpsService
	email  *EmailService
	now    func() time.Time
	// mail overrides the ops recipients in tests.
	mail     func(subject, body string) bool
	runMu    sync.Mutex
	stopOnce sync.Once
	stop     chan struct{}
}

func NewUserRiskService(repo UserRiskRepository, shadow *UserShadowService, growth *GrowthService, ops *OpsService, email *EmailService) *UserRiskService {
	return &UserRiskService{repo: repo, shadow: shadow, growth: growth, ops: ops, email: email, now: time.Now, stop: make(chan struct{})}
}

// ProvideUserRiskService starts the monitor (every 20 minutes).
func ProvideUserRiskService(repo UserRiskRepository, shadow *UserShadowService, growth *GrowthService, ops *OpsService, email *EmailService) *UserRiskService {
	s := NewUserRiskService(repo, shadow, growth, ops, email)
	go func() {
		select {
		case <-s.stop:
			return
		case <-time.After(2 * time.Minute):
		}
		t := time.NewTicker(riskMonitorInterval)
		defer t.Stop()
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if _, err := s.RunMonitor(ctx); err != nil {
				logger.LegacyPrintf("service.user_risk", "[Risk] monitor run failed: %v", err)
			}
			cancel()
			select {
			case <-s.stop:
				return
			case <-t.C:
			}
		}
	}()
	return s
}

func (s *UserRiskService) Stop() { s.stopOnce.Do(func() { close(s.stop) }) }

// RiskRuleLabels are the Chinese names used in mails (the dashboard has its own i18n).
var RiskRuleLabels = map[string]string{
	RiskRuleLinkedSignup:       "新注册关联到已封禁/静默账号",
	RiskRuleTrialDay1:          "试用账号首日用完额度",
	RiskRuleRelayFree:          "中转工具消耗免费额度",
	RiskRuleBurstSignups:       "同网段/同浏览器短时间多次注册",
	RiskRuleInviteeFingerprint: "被邀请人与邀请人同指纹",
}

// RunMonitor evaluates the enabled rules once, stores new events and mails them.
func (s *UserRiskService) RunMonitor(ctx context.Context) ([]RiskEvent, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	settings := RiskSettings{RiskLinkedSignup: true, RiskTrialDay1: true, RiskRelayFree: true, RiskBurstSignups: true,
		RiskBurstThreshold: riskBurstThresholdDefault, RiskInviteeFingerprint: true}
	if s.growth != nil {
		gs, err := s.growth.GetSettings(ctx)
		if err != nil {
			return nil, err
		}
		settings = gs.RiskSettings
	}
	now := s.now()
	since := now.Add(-riskMonitorLookback)
	type rule struct {
		name string
		on   bool
		run  func() ([]RiskHit, error)
	}
	rules := []rule{
		{RiskRuleLinkedSignup, settings.RiskLinkedSignup, func() ([]RiskHit, error) { return s.repo.LinkedFlaggedSignups(ctx, since) }},
		{RiskRuleTrialDay1, settings.RiskTrialDay1, func() ([]RiskHit, error) { return s.repo.TrialDay1Exhausted(ctx, since) }},
		{RiskRuleRelayFree, settings.RiskRelayFree, func() ([]RiskHit, error) { return s.repo.RelayFreeSpend(ctx, since, relayUAPattern) }},
		{RiskRuleBurstSignups, settings.RiskBurstSignups, func() ([]RiskHit, error) {
			return s.repo.BurstSignups(ctx, since, settings.RiskBurstThreshold)
		}},
		{RiskRuleInviteeFingerprint, settings.RiskInviteeFingerprint, func() ([]RiskHit, error) {
			return s.repo.InviteeSharesInviterFingerprint(ctx, now.Add(-riskInviteeLookback))
		}},
	}
	var fresh []RiskEvent
	for _, r := range rules {
		if !r.on {
			continue
		}
		hits, err := r.run()
		if err != nil {
			logger.LegacyPrintf("service.user_risk", "[Risk] rule %s failed: %v", r.name, err)
			continue
		}
		if len(hits) == 0 {
			continue
		}
		events, err := s.repo.InsertRiskEvents(ctx, r.name, hits)
		if err != nil {
			logger.LegacyPrintf("service.user_risk", "[Risk] store %s events failed: %v", r.name, err)
			continue
		}
		fresh = append(fresh, events...)
	}
	if len(fresh) > 0 {
		s.mailEvents(ctx, fresh)
	}
	return fresh, nil
}

func (s *UserRiskService) mailEvents(ctx context.Context, events []RiskEvent) {
	subject := fmt.Sprintf("[风控] %d 条新事件", len(events))
	body := riskMailBody(events)
	if s.mail != nil {
		s.mail(subject, body)
		return
	}
	if s.email == nil || s.ops == nil {
		return
	}
	cfg, err := s.ops.GetEmailNotificationConfig(ctx)
	if err != nil || cfg == nil || !cfg.Alert.Enabled || len(cfg.Alert.Recipients) == 0 {
		return
	}
	for _, to := range cfg.Alert.Recipients {
		if addr := strings.TrimSpace(to); addr != "" {
			if err := s.email.SendEmail(ctx, addr, subject, body); err != nil {
				logger.LegacyPrintf("service.user_risk", "[Risk] mail to %s failed: %v", addr, err)
			}
		}
	}
}

func riskMailBody(events []RiskEvent) string {
	var b strings.Builder
	_, _ = b.WriteString(`<h3>风控监控发现新事件</h3><table border="1" cellpadding="6" style="border-collapse:collapse"><tr><th>用户</th><th>规则</th><th>详情</th></tr>`)
	for _, e := range events {
		who := "#" + strconv.FormatInt(e.UserID, 10)
		if e.UserEmail != "" {
			who += " " + e.UserEmail
		}
		label := RiskRuleLabels[e.Rule]
		if label == "" {
			label = e.Rule
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%s</td><td>%s</td></tr>", html.EscapeString(who), html.EscapeString(label), html.EscapeString(e.Detail))
	}
	_, _ = b.WriteString(`</table><p>在后台「用户画像」里按用户 ID 查看详情，可在「用户管理」里静默限制或封禁。</p>` +
		`<p style="color:#888;font-size:12px">同一用户的同一条规则只提醒一次。规则开关在「推广设置 → 风控监控」。</p>`)
	return b.String()
}

// ListEvents pages through risk events (userID 0 = all).
func (s *UserRiskService) ListEvents(ctx context.Context, userID int64, page, pageSize int) ([]RiskEvent, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListRiskEvents(ctx, userID, page, pageSize)
}

func insightsWindow(days int) int {
	if days <= 0 {
		return insightsDefaultDays
	}
	if days > insightsMaxDays {
		return insightsMaxDays
	}
	return days
}

// Overview is the 用户画像 overview for the last `days` days.
func (s *UserRiskService) Overview(ctx context.Context, days int) (*UserInsightsOverview, error) {
	days = insightsWindow(days)
	since := s.now().AddDate(0, 0, -days)
	out := &UserInsightsOverview{Days: days, Since: since}
	var err error
	if out.Segments, out.Signups, err = s.repo.InsightsSegments(ctx, since); err != nil {
		return nil, err
	}
	if out.Spend, err = s.repo.InsightsSpend(ctx, since); err != nil {
		return nil, err
	}
	rows, err := s.repo.InsightsUserAgents(ctx, since)
	if err != nil {
		return nil, err
	}
	var relayIDs []int64
	out.Clients, out.RelayUsers, relayIDs = summarizeUserAgents(rows)
	clusters, err := s.repo.InsightsClusters(ctx, since)
	if err != nil {
		return nil, err
	}
	ids := append([]int64(nil), relayIDs...)
	for _, c := range clusters {
		ids = append(ids, c.UserIDs...)
	}
	refs, err := s.repo.UserRefs(ctx, ids, since)
	if err != nil {
		return nil, err
	}
	s.markShadowed(refs)
	for i := range out.RelayUsers {
		if ref, ok := refs[out.RelayUsers[i].User.ID]; ok {
			out.RelayUsers[i].User = ref
		}
	}
	for i := range clusters {
		c := &clusters[i]
		c.Users = make([]InsightsUserRef, 0, len(c.UserIDs))
		for _, id := range c.UserIDs {
			ref, ok := refs[id]
			if !ok {
				continue
			}
			c.Users = append(c.Users, ref)
			c.PaidCNY += ref.PaidCNY
			if ref.PaidCNY <= 0 {
				c.FreeUSD += ref.SpendUSD
			}
		}
		c.FreeUSD, c.PaidCNY = roundTo(c.FreeUSD, 4), roundTo(c.PaidCNY, 2)
	}
	sort.SliceStable(clusters, func(i, j int) bool {
		if len(clusters[i].Users) != len(clusters[j].Users) {
			return len(clusters[i].Users) > len(clusters[j].Users)
		}
		return clusters[i].FreeUSD > clusters[j].FreeUSD
	})
	out.Clusters = clusters
	return out, nil
}

func (s *UserRiskService) markShadowed(refs map[int64]InsightsUserRef) {
	if s.shadow == nil {
		return
	}
	for id, ref := range refs {
		ref.Shadowed = s.shadow.IsShadowed(id)
		refs[id] = ref
	}
}

// summarizeUserAgents turns per-(user, UA) usage into the client mix and the relay-UA user list
// (one row per user: their busiest relay UA, with all their relay usage).
func summarizeUserAgents(rows []InsightsUARow) ([]InsightsClient, []InsightsRelayUser, []int64) {
	byClass := map[string]*InsightsClient{}
	usersByClass := map[string]map[int64]bool{}
	relay := map[int64]*InsightsRelayUser{}
	relayTop := map[int64]int64{}
	for _, r := range rows {
		class := ClassifyClientUA(r.UserAgent)
		c := byClass[class]
		if c == nil {
			c = &InsightsClient{Class: class}
			byClass[class] = c
			usersByClass[class] = map[int64]bool{}
		}
		c.Requests += r.Requests
		c.CostUSD += r.CostUSD
		usersByClass[class][r.UserID] = true
		if class == ClientRelay {
			u := relay[r.UserID]
			if u == nil {
				u = &InsightsRelayUser{User: InsightsUserRef{ID: r.UserID}}
				relay[r.UserID] = u
			}
			u.Requests += r.Requests
			u.CostUSD += r.CostUSD
			if r.Requests > relayTop[r.UserID] {
				relayTop[r.UserID] = r.Requests
				u.UserAgent = r.UserAgent
			}
		}
	}
	clients := make([]InsightsClient, 0, len(byClass))
	for class, c := range byClass {
		c.Users = len(usersByClass[class])
		c.CostUSD = roundTo(c.CostUSD, 4)
		clients = append(clients, *c)
	}
	sort.Slice(clients, func(i, j int) bool { return clients[i].Requests > clients[j].Requests })
	users := make([]InsightsRelayUser, 0, len(relay))
	ids := make([]int64, 0, len(relay))
	for id, u := range relay {
		u.CostUSD = roundTo(u.CostUSD, 4)
		users = append(users, *u)
		ids = append(ids, id)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].CostUSD > users[j].CostUSD })
	return clients, users, ids
}

// Profile is one user's 用户画像; q is a user ID or an email.
func (s *UserRiskService) Profile(ctx context.Context, q string) (*UserInsightsProfile, error) {
	userID, err := s.repo.FindUserID(ctx, strings.TrimSpace(q))
	if err != nil {
		return nil, err
	}
	if userID <= 0 {
		return nil, ErrInsightsUserNotFound
	}
	p, err := s.repo.UserProfile(ctx, userID, s.now().AddDate(0, 0, -insightsMaxDays))
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrInsightsUserNotFound
	}
	if s.shadow != nil {
		if v, ok := s.shadow.Get(userID); ok {
			p.Shadow = &v
			p.User.Shadowed = true
		}
		for i := range p.Linked {
			p.Linked[i].User.Shadowed = s.shadow.IsShadowed(p.Linked[i].User.ID)
		}
		for i := range p.InviteUp {
			p.InviteUp[i].Shadowed = s.shadow.IsShadowed(p.InviteUp[i].ID)
		}
		for i := range p.InviteDown {
			p.InviteDown[i].Shadowed = s.shadow.IsShadowed(p.InviteDown[i].ID)
		}
	}
	for i := range p.Clients {
		p.Clients[i].Class = ClassifyClientUA(p.Clients[i].UserAgent)
	}
	if p.Events, _, err = s.repo.ListRiskEvents(ctx, userID, 1, 50); err != nil {
		return nil, err
	}
	p.RiskScore, p.RiskReasons = scoreUserRisk(p)
	return p, nil
}

// scoreUserRisk is a 0–100 heuristic with the reasons behind it.
func scoreUserRisk(p *UserInsightsProfile) (int, []string) {
	score := 0
	reasons := []string{}
	add := func(points int, reason string) {
		score += points
		reasons = append(reasons, fmt.Sprintf("+%d %s", points, reason))
	}
	if p.User.Status == StatusDisabled {
		add(30, "账号已封禁")
	}
	if p.Shadow != nil {
		add(20, "已静默限制："+p.Shadow.Reason)
	}
	banned, shadowed, linked := 0, 0, 0
	for _, l := range p.Linked {
		fp := false
		for _, v := range l.Via {
			if v == "visitor" || v == "ip_prefix" || v == "api_ip" || v == "user_agent" {
				fp = true
			}
		}
		if !fp {
			continue
		}
		linked++
		if l.User.Status == StatusDisabled {
			banned++
		} else if l.User.Shadowed {
			shadowed++
		}
	}
	if banned > 0 {
		add(min(40, 25*banned), fmt.Sprintf("与 %d 个已封禁账号同浏览器/网段/IP/UA", banned))
	}
	if shadowed > 0 {
		add(min(30, 15*shadowed), fmt.Sprintf("与 %d 个静默账号同浏览器/网段/IP/UA", shadowed))
	}
	if linked >= 2 {
		add(min(30, 5*linked), fmt.Sprintf("关联账号 %d 个", linked))
	}
	for _, c := range p.Clients {
		if c.Class == ClientRelay {
			add(25, "用中转工具调用："+c.UserAgent)
			break
		}
	}
	if !p.Spend.HasRealPaying && p.Spend.TotalUSD >= 5 {
		add(10, fmt.Sprintf("从未付款，已用免费额度 $%.2f", p.Spend.TotalUSD))
	}
	for _, e := range p.Events {
		switch e.Rule {
		case RiskRuleTrialDay1:
			add(15, "试用首日用完额度")
		case RiskRuleInviteeFingerprint:
			add(20, "与邀请人同指纹")
		case RiskRuleBurstSignups:
			add(15, "同网段/同浏览器短时间多次注册")
		}
	}
	if p.Spend.HasRealPaying {
		score -= 20
		reasons = append(reasons, "-20 付过款")
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score, reasons
}
