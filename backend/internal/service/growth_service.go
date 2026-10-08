package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Growth programs built on top of the affiliate system:
//   - invitee first-order bonus: the invited user gets a share of their first paid order back as balance;
//   - education verification: users prove a school email and get a discount on subscription plans;
//   - invite leaderboard: a public, masked ranking plus a detailed admin view for handing out rewards.
// Settings live in the settings table under the keys below and are edited on the admin
// "推广设置" page (GET/PUT /admin/growth/settings).

const (
	SettingKeyGrowthInviteeBonusRate    = "growth_invitee_bonus_rate"         // 被邀请人首单奖励比例（百分比，0=关闭）
	SettingKeyGrowthInviteeBonusCap     = "growth_invitee_bonus_cap"          // 被邀请人首单奖励上限（0=不限）
	SettingKeyGrowthLeaderboardEnabled  = "growth_leaderboard_enabled"        // 是否向用户展示邀请排行榜
	SettingKeyGrowthSignupBonus         = "growth_invitee_signup_bonus"       // 被邀请人注册试用余额（0=关闭）
	SettingKeyGrowthSignupDailyLimit    = "growth_invitee_signup_daily_limit" // 注册试用每天最多发放人数
	SettingKeyGrowthWithdrawEnabled     = "growth_withdraw_enabled"           // 是否开放返利提现（人工打款）
	SettingKeyGrowthWithdrawMinCNY      = "growth_withdraw_min_cny"           // 最低提现金额（元）
	SettingKeyGrowthWithdrawMonthly     = "growth_withdraw_monthly_limit"     // 每人每月最多提现次数（0=不限）
	SettingKeyEduVerifyEnabled          = "edu_verify_enabled"                // 是否开放教育邮箱认证
	SettingKeyEduEmailSuffixes          = "edu_email_suffixes"                // 教育邮箱后缀（JSON 数组，如 ["edu.cn"]）
	SettingKeyEduSubscriptionDiscount   = "edu_subscription_discount_percent" // 认证用户订阅折扣（百分比，0=无折扣）
	growthSettingsCacheTTL              = 5 * time.Second
	growthInviteeBonusRateMax           = 100.0
	growthEduDiscountMax                = 90.0
	growthEduSuffixesMax                = 20
	growthLeaderboardPublicLimitDefault = 20
	growthSignupBonusMax                = 5.0
	growthSignupDailyLimitDefault       = 20
	growthSignupDailyLimitMax           = 1000
	// One inviter gets at most this many trial credits granted to their invitees per day.
	growthSignupPerInviterDaily    = 5
	growthLeaderboardAdminLimitMax = 500
)

var defaultEduEmailSuffixes = []string{"edu.cn"}

var (
	ErrEduVerifyDisabled   = infraerrors.Forbidden("EDU_VERIFY_DISABLED", "education verification is not available")
	ErrEduEmailInvalid     = infraerrors.BadRequest("EDU_EMAIL_INVALID", "invalid email address")
	ErrEduEmailNotSchool   = infraerrors.BadRequest("EDU_EMAIL_NOT_SCHOOL", "this email is not a recognized school email")
	ErrEduEmailTaken       = infraerrors.Conflict("EDU_EMAIL_TAKEN", "this school email is already verified by another account")
	ErrLeaderboardDisabled = infraerrors.Forbidden("LEADERBOARD_DISABLED", "the invite leaderboard is not available")
)

// GrowthSettings are the admin-editable knobs of the growth programs.
type GrowthSettings struct {
	InviteeBonusRatePercent float64 `json:"invitee_bonus_rate_percent"`
	InviteeBonusCap         float64 `json:"invitee_bonus_cap"`
	// Trial balance for someone who signs up with an invite (0 = off), and how many a day at most.
	InviteeSignupBonus      float64  `json:"invitee_signup_bonus"`
	InviteeSignupDailyLimit int      `json:"invitee_signup_daily_limit"`
	LeaderboardEnabled      bool     `json:"leaderboard_enabled"`
	EduVerifyEnabled        bool     `json:"edu_verify_enabled"`
	EduEmailSuffixes        []string `json:"edu_email_suffixes"`
	EduDiscountPercent      float64  `json:"edu_discount_percent"`
	// 返利提现规则（字段平铺在 JSON 里：withdraw_enabled / withdraw_min_cny / withdraw_monthly_limit）
	WithdrawSettings
}

// EduVerification is a verified school email of a user.
type EduVerification struct {
	UserID     int64     `json:"user_id"`
	Email      string    `json:"email"`
	Method     string    `json:"method"` // email | manual
	Note       string    `json:"note,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
	UserEmail  string    `json:"user_email,omitempty"`
	Username   string    `json:"username,omitempty"`
}

// InviteLeaderboardEntry is one inviter in a ranking period.
type InviteLeaderboardEntry struct {
	Rank           int     `json:"rank"`
	UserID         int64   `json:"user_id,omitempty"`
	DisplayName    string  `json:"display_name"`
	Email          string  `json:"email,omitempty"`
	Username       string  `json:"username,omitempty"`
	InvitedCount   int     `json:"invited_count"`
	PayingInvitees int     `json:"paying_invitees"`
	InviteePaid    float64 `json:"invitee_paid,omitempty"`
	RebateAccrued  float64 `json:"rebate_accrued,omitempty"`
	// CustomRatePercent is the inviter's exclusive rebate rate, if any (admin view only).
	CustomRatePercent *float64 `json:"custom_rate_percent,omitempty"`
	IsCurrentUser     bool     `json:"is_current_user,omitempty"`
}

// InviteLeaderboard is a ranking over [Start, End).
type InviteLeaderboard struct {
	Period  string                   `json:"period"`
	Start   *time.Time               `json:"start,omitempty"`
	End     *time.Time               `json:"end,omitempty"`
	Entries []InviteLeaderboardEntry `json:"entries"`
	Me      *InviteLeaderboardEntry  `json:"me,omitempty"`
}

// GrowthRepository persists growth-program data (raw SQL).
type GrowthRepository interface {
	GetEduVerification(ctx context.Context, userID int64) (*EduVerification, error)
	// EduEmailOwner returns the user that verified email (case-insensitive), or 0.
	EduEmailOwner(ctx context.Context, email string) (int64, error)
	// UpsertEduVerification stores userID's school email; returns ErrEduEmailTaken when another user holds it.
	UpsertEduVerification(ctx context.Context, userID int64, email string) (*EduVerification, error)
	// GrantEduVerification verifies userID by hand (method manual) with the admin's note.
	GrantEduVerification(ctx context.Context, userID int64, email, note string, adminID int64) (*EduVerification, error)
	// FindEduUser finds an active user by account email (case-insensitive) or ID; 0 when none.
	FindEduUser(ctx context.Context, query string) (int64, string, error)
	DeleteEduVerification(ctx context.Context, userID int64) (bool, error)
	ListEduVerifications(ctx context.Context, search string, page, pageSize int) ([]EduVerification, int64, error)
	// GrantInviteeSignupBonus credits the trial balance once to inviteeID, bound to inviterID, when the
	// invitee registered after registeredAfter, the inviter has used the site (an API call or a paid
	// order), and fewer than dailyLimit grants (perInviterLimit for this inviter) were made since dayStart.
	GrantInviteeSignupBonus(ctx context.Context, inviteeID, inviterID int64, amount float64, registeredAfter, dayStart time.Time, dailyLimit, perInviterLimit int) (bool, error)
	// GrantInviteeBonus credits amount to the invitee's balance for orderID when the invitee has an
	// inviter, no earlier paid gateway order and no previous bonus. Returns the inviter and whether it was granted.
	GrantInviteeBonus(ctx context.Context, inviteeID, orderID int64, amount float64, bindingNotBefore *time.Time) (granted bool, inviterID int64, err error)
	// InviteLeaderboard ranks inviters by invitees who paid (gateway orders) within [start, end).
	InviteLeaderboard(ctx context.Context, start, end *time.Time, limit int) ([]InviteLeaderboardEntry, error)
	InviteLeaderboardEntryFor(ctx context.Context, userID int64, start, end *time.Time) (*InviteLeaderboardEntry, error)
}

// GrowthService implements the growth programs.
type GrowthService struct {
	settingRepo     SettingRepository
	repo            GrowthRepository
	settingService  *SettingService
	emailService    *EmailService
	billingCache    *BillingCacheService
	authInvalidator APIKeyAuthCacheInvalidator

	mu       sync.Mutex
	cached   *GrowthSettings
	cachedAt time.Time
}

// NewGrowthService creates the growth service.
func NewGrowthService(settingRepo SettingRepository, repo GrowthRepository, settingService *SettingService, emailService *EmailService, billingCache *BillingCacheService, authInvalidator APIKeyAuthCacheInvalidator) *GrowthService {
	return &GrowthService{settingRepo: settingRepo, repo: repo, settingService: settingService, emailService: emailService, billingCache: billingCache, authInvalidator: authInvalidator}
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// GetSettings returns the current growth settings (cached for a few seconds).
func (s *GrowthService) GetSettings(ctx context.Context) (*GrowthSettings, error) {
	s.mu.Lock()
	if s.cached != nil && time.Since(s.cachedAt) < growthSettingsCacheTTL {
		out := *s.cached
		s.mu.Unlock()
		return &out, nil
	}
	s.mu.Unlock()

	vals, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyGrowthInviteeBonusRate, SettingKeyGrowthInviteeBonusCap, SettingKeyGrowthLeaderboardEnabled,
		SettingKeyGrowthSignupBonus, SettingKeyGrowthSignupDailyLimit, SettingKeyEduVerifyEnabled, SettingKeyEduEmailSuffixes, SettingKeyEduSubscriptionDiscount,
		SettingKeyGrowthWithdrawEnabled, SettingKeyGrowthWithdrawMinCNY, SettingKeyGrowthWithdrawMonthly,
	})
	if err != nil {
		return nil, fmt.Errorf("get growth settings: %w", err)
	}
	settings := parseGrowthSettings(vals)
	s.mu.Lock()
	s.cached, s.cachedAt = settings, time.Now()
	s.mu.Unlock()
	out := *settings
	return &out, nil
}

func parseGrowthSettings(vals map[string]string) *GrowthSettings {
	settings := &GrowthSettings{
		InviteeBonusRatePercent: clampFloat(parseFloatOr(vals[SettingKeyGrowthInviteeBonusRate], 0), 0, growthInviteeBonusRateMax),
		InviteeBonusCap:         math.Max(0, parseFloatOr(vals[SettingKeyGrowthInviteeBonusCap], 0)),
		InviteeSignupBonus:      clampFloat(parseFloatOr(vals[SettingKeyGrowthSignupBonus], 0), 0, growthSignupBonusMax),
		InviteeSignupDailyLimit: int(clampFloat(parseFloatOr(vals[SettingKeyGrowthSignupDailyLimit], growthSignupDailyLimitDefault), 1, growthSignupDailyLimitMax)),
		LeaderboardEnabled:      vals[SettingKeyGrowthLeaderboardEnabled] != "false", // on unless turned off
		EduVerifyEnabled:        vals[SettingKeyEduVerifyEnabled] == "true",
		EduDiscountPercent:      clampFloat(parseFloatOr(vals[SettingKeyEduSubscriptionDiscount], 0), 0, growthEduDiscountMax),
		EduEmailSuffixes:        defaultEduEmailSuffixes,
		WithdrawSettings: WithdrawSettings{
			Enabled:      vals[SettingKeyGrowthWithdrawEnabled] == "true",
			MinCNY:       clampFloat(parseFloatOr(vals[SettingKeyGrowthWithdrawMinCNY], withdrawMinCNYDefault), withdrawMinCNYFloor, 100000),
			MonthlyLimit: int(clampFloat(parseFloatOr(vals[SettingKeyGrowthWithdrawMonthly], withdrawMonthlyLimitDefault), 0, withdrawMonthlyLimitMax)),
		},
	}
	if raw := strings.TrimSpace(vals[SettingKeyEduEmailSuffixes]); raw != "" {
		var list []string
		if err := json.Unmarshal([]byte(raw), &list); err == nil {
			if normalized := normalizeEduSuffixes(list); len(normalized) > 0 {
				settings.EduEmailSuffixes = normalized
			}
		}
	}
	return settings
}

// UpdateSettings validates and saves growth settings.
func (s *GrowthService) UpdateSettings(ctx context.Context, in GrowthSettings) (*GrowthSettings, error) {
	if !finite(in.InviteeBonusRatePercent) || in.InviteeBonusRatePercent < 0 || in.InviteeBonusRatePercent > growthInviteeBonusRateMax {
		return nil, infraerrors.BadRequest("INVALID_INVITEE_BONUS_RATE", "invitee bonus rate must be between 0 and 100")
	}
	if !finite(in.InviteeBonusCap) || in.InviteeBonusCap < 0 {
		return nil, infraerrors.BadRequest("INVALID_INVITEE_BONUS_CAP", "invitee bonus cap must be >= 0")
	}
	if !finite(in.InviteeSignupBonus) || in.InviteeSignupBonus < 0 || in.InviteeSignupBonus > growthSignupBonusMax {
		return nil, infraerrors.BadRequest("INVALID_INVITEE_SIGNUP_BONUS", "注册试用余额需在 0 到 5 之间（Sign-up trial credit must be between 0 and 5）")
	}
	if in.InviteeSignupDailyLimit <= 0 {
		in.InviteeSignupDailyLimit = growthSignupDailyLimitDefault
	}
	if in.InviteeSignupDailyLimit > growthSignupDailyLimitMax {
		return nil, infraerrors.BadRequest("INVALID_INVITEE_SIGNUP_LIMIT", "每天发放人数最多 1000（At most 1000 trial credits a day）")
	}
	if !finite(in.EduDiscountPercent) || in.EduDiscountPercent < 0 || in.EduDiscountPercent > growthEduDiscountMax {
		return nil, infraerrors.BadRequest("INVALID_EDU_DISCOUNT", "education discount must be between 0 and 90")
	}
	suffixes := normalizeEduSuffixes(in.EduEmailSuffixes)
	if len(suffixes) == 0 {
		return nil, infraerrors.BadRequest("INVALID_EDU_SUFFIXES", "at least one school email suffix is required")
	}
	if len(suffixes) > growthEduSuffixesMax {
		return nil, infraerrors.BadRequest("INVALID_EDU_SUFFIXES", "too many school email suffixes")
	}
	withdraw, err := normalizeWithdrawSettings(in.WithdrawSettings)
	if err != nil {
		return nil, err
	}
	suffixJSON, _ := json.Marshal(suffixes)
	if err := s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingKeyGrowthInviteeBonusRate:   strconv.FormatFloat(in.InviteeBonusRatePercent, 'f', -1, 64),
		SettingKeyGrowthInviteeBonusCap:    strconv.FormatFloat(in.InviteeBonusCap, 'f', -1, 64),
		SettingKeyGrowthLeaderboardEnabled: strconv.FormatBool(in.LeaderboardEnabled),
		SettingKeyGrowthSignupBonus:        strconv.FormatFloat(in.InviteeSignupBonus, 'f', -1, 64),
		SettingKeyGrowthSignupDailyLimit:   strconv.Itoa(in.InviteeSignupDailyLimit),
		SettingKeyEduVerifyEnabled:         strconv.FormatBool(in.EduVerifyEnabled),
		SettingKeyEduEmailSuffixes:         string(suffixJSON),
		SettingKeyEduSubscriptionDiscount:  strconv.FormatFloat(in.EduDiscountPercent, 'f', -1, 64),
		SettingKeyGrowthWithdrawEnabled:    strconv.FormatBool(withdraw.Enabled),
		SettingKeyGrowthWithdrawMinCNY:     strconv.FormatFloat(withdraw.MinCNY, 'f', -1, 64),
		SettingKeyGrowthWithdrawMonthly:    strconv.Itoa(withdraw.MonthlyLimit),
	}); err != nil {
		return nil, fmt.Errorf("save growth settings: %w", err)
	}
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
	return s.GetSettings(ctx)
}

// normalizeEduSuffixes lowercases suffixes and strips "@", "*." and leading dots; drops invalid ones.
func normalizeEduSuffixes(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.ToLower(strings.TrimSpace(raw))
		v = strings.TrimPrefix(v, "@")
		v = strings.TrimPrefix(v, "*.")
		v = strings.TrimLeft(v, ".")
		if v == "" || !strings.Contains(v, ".") || strings.ContainsAny(v, " @/*") || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// isSchoolEmailDomain reports whether domain equals or is a subdomain of one of the suffixes.
func isSchoolEmailDomain(domain string, suffixes []string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, suffix := range suffixes {
		if domain == suffix || strings.HasSuffix(domain, "."+suffix) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Education verification and discount
// ---------------------------------------------------------------------------

// EduStatus is what the user sees on the education verification card.
type EduStatus struct {
	Enabled         bool             `json:"enabled"`
	DiscountPercent float64          `json:"discount_percent"`
	Suffixes        []string         `json:"suffixes"`
	Verification    *EduVerification `json:"verification,omitempty"`
}

// GetEduStatus returns userID's verification state and the program settings.
func (s *GrowthService) GetEduStatus(ctx context.Context, userID int64) (*EduStatus, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	status := &EduStatus{Enabled: settings.EduVerifyEnabled, DiscountPercent: settings.EduDiscountPercent, Suffixes: settings.EduEmailSuffixes}
	v, err := s.repo.GetEduVerification(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.Verification = v
	return status, nil
}

// normalizeSchoolEmail validates email against the configured school suffixes.
func (s *GrowthService) normalizeSchoolEmail(ctx context.Context, raw string) (string, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return "", err
	}
	if !settings.EduVerifyEnabled {
		return "", ErrEduVerifyDisabled
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil || addr.Name != "" || len(addr.Address) > 254 {
		return "", ErrEduEmailInvalid
	}
	email := strings.ToLower(addr.Address)
	at := strings.LastIndex(email, "@")
	if at <= 0 || !isSchoolEmailDomain(email[at+1:], settings.EduEmailSuffixes) {
		return "", ErrEduEmailNotSchool
	}
	return email, nil
}

// SendEduCode emails a verification code to a school email for userID.
func (s *GrowthService) SendEduCode(ctx context.Context, userID int64, rawEmail, locale string) error {
	email, err := s.normalizeSchoolEmail(ctx, rawEmail)
	if err != nil {
		return err
	}
	if s.emailService == nil {
		return infraerrors.ServiceUnavailable("EMAIL_UNAVAILABLE", "email service is not configured")
	}
	// Don't send a code that can never succeed: the address already belongs to an account.
	ownerID, err := s.repo.EduEmailOwner(ctx, email)
	if err != nil {
		return err
	}
	if ownerID == userID {
		return infraerrors.BadRequest("EDU_ALREADY_VERIFIED", "this school email is already verified on your account")
	}
	if ownerID > 0 {
		return ErrEduEmailTaken
	}
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}
	return s.emailService.SendVerifyCode(ctx, eduCodeKey(email), siteName, locale)
}

// VerifyEdu checks the code and records the school email for userID.
func (s *GrowthService) VerifyEdu(ctx context.Context, userID int64, rawEmail, code string) (*EduVerification, error) {
	email, err := s.normalizeSchoolEmail(ctx, rawEmail)
	if err != nil {
		return nil, err
	}
	if s.emailService == nil {
		return nil, infraerrors.ServiceUnavailable("EMAIL_UNAVAILABLE", "email service is not configured")
	}
	if err := s.emailService.VerifyCode(ctx, eduCodeKey(email), strings.TrimSpace(code)); err != nil {
		return nil, err
	}
	return s.repo.UpsertEduVerification(ctx, userID, email)
}

// eduCodeKey is the address the code is sent to and cached under. It is the email itself:
// EmailService keys codes by recipient, and the code only proves the recipient reads that inbox.
func eduCodeKey(email string) string { return email }

// DiscountedPrice returns the price userID pays after the education discount and whether it applied.
func (s *GrowthService) DiscountedPrice(ctx context.Context, userID int64, price float64) (float64, bool) {
	if s == nil || userID <= 0 || price <= 0 {
		return price, false
	}
	settings, err := s.GetSettings(ctx)
	if err != nil || !settings.EduVerifyEnabled || settings.EduDiscountPercent <= 0 {
		return price, false
	}
	v, err := s.repo.GetEduVerification(ctx, userID)
	if err != nil || v == nil {
		return price, false
	}
	discounted := math.Round(price*(100-settings.EduDiscountPercent)) / 100
	if discounted <= 0 || discounted >= price {
		return price, false
	}
	return discounted, true
}

// AdminListEduVerifications lists verified users.
func (s *GrowthService) AdminListEduVerifications(ctx context.Context, search string, page, pageSize int) ([]EduVerification, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	return s.repo.ListEduVerifications(ctx, strings.TrimSpace(search), page, pageSize)
}

const (
	EduMethodEmail  = "email"
	EduMethodManual = "manual"
	eduNoteMax      = 100
)

var (
	ErrEduGrantUserNotFound  = infraerrors.NotFound("EDU_GRANT_USER_NOT_FOUND", "找不到这个用户，请填写账号邮箱或用户 ID（User not found）")
	ErrEduGrantNote          = infraerrors.BadRequest("EDU_GRANT_NOTE", "请填写认证说明（学校、身份、怎么核验的），最多 100 字（A note of 1–100 characters is required）")
	ErrEduGrantEmailVerified = infraerrors.Conflict("EDU_GRANT_EMAIL_VERIFIED", "该用户已通过学校邮箱认证，无需手动认证（Already verified with a school email）")
)

// AdminGrantEduVerification verifies a user by hand — for teachers and students whose school has
// no school email. user is the account email or ID; the note records the school and how it was checked.
// The account email stands in for the school email (an account without one gets a placeholder).
func (s *GrowthService) AdminGrantEduVerification(ctx context.Context, adminID int64, user, note string) (*EduVerification, error) {
	user = strings.TrimPrefix(strings.TrimSpace(user), "#")
	note = strings.TrimSpace(note)
	if n := utf8.RuneCountInString(note); n == 0 || n > eduNoteMax {
		return nil, ErrEduGrantNote
	}
	if user == "" {
		return nil, ErrEduGrantUserNotFound
	}
	userID, email, err := s.repo.FindEduUser(ctx, user)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return nil, ErrEduGrantUserNotFound
	}
	existing, err := s.repo.GetEduVerification(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Method != EduMethodManual {
		return nil, ErrEduGrantEmailVerified
	}
	if email == "" {
		email = fmt.Sprintf("user-%d@manual", userID)
	}
	return s.repo.GrantEduVerification(ctx, userID, email, note, adminID)
}

// AdminRevokeEduVerification removes a user's verification.
func (s *GrowthService) AdminRevokeEduVerification(ctx context.Context, userID int64) error {
	removed, err := s.repo.DeleteEduVerification(ctx, userID)
	if err != nil {
		return err
	}
	if !removed {
		return infraerrors.NotFound("EDU_VERIFICATION_NOT_FOUND", "no education verification for this user")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Invitee sign-up trial credit
// ---------------------------------------------------------------------------

// ProvideGrowthService creates the growth service and has the affiliate service grant the
// sign-up trial credit whenever a user binds an inviter.
func ProvideGrowthService(settingRepo SettingRepository, repo GrowthRepository, settingService *SettingService, emailService *EmailService, billingCache *BillingCacheService, authInvalidator APIKeyAuthCacheInvalidator, affiliate *AffiliateService) *GrowthService {
	s := NewGrowthService(settingRepo, repo, settingService, emailService, billingCache, authInvalidator)
	affiliate.SetInviterBoundHook(func(ctx context.Context, inviteeID, inviterID int64) {
		if _, err := s.GrantInviteeSignupBonus(ctx, inviteeID, inviterID); err != nil {
			logger.LegacyPrintf("service.growth", "[Growth] sign-up trial credit for user %d failed: %v", inviteeID, err)
		}
	})
	return s
}

// GrantInviteeSignupBonus gives someone who just signed up with an invite a little balance to try
// the API with, so the first call doesn't wait for a payment. Once per user; abuse is bounded by a
// daily total, a per-inviter daily cap, and inviters having to be real users themselves.
func (s *GrowthService) GrantInviteeSignupBonus(ctx context.Context, inviteeID, inviterID int64) (float64, error) {
	if s == nil || s.repo == nil || inviteeID <= 0 || inviterID <= 0 || inviteeID == inviterID {
		return 0, nil
	}
	if s.settingService != nil && !s.settingService.IsAffiliateEnabled(ctx) {
		return 0, nil
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	amount := settings.InviteeSignupBonus
	if amount <= 0 {
		return 0, nil
	}
	now := time.Now()
	local := now.In(analyticsTZ)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, analyticsTZ)
	granted, err := s.repo.GrantInviteeSignupBonus(ctx, inviteeID, inviterID, amount,
		now.Add(-AffiliateLateBindWindow), dayStart, settings.InviteeSignupDailyLimit, growthSignupPerInviterDaily)
	if err != nil || !granted {
		return 0, err
	}
	if s.authInvalidator != nil {
		s.authInvalidator.InvalidateAuthCacheByUserID(ctx, inviteeID)
	}
	if s.billingCache != nil {
		_ = s.billingCache.InvalidateUserBalance(ctx, inviteeID)
	}
	return amount, nil
}

// ---------------------------------------------------------------------------
// Invitee first-order bonus
// ---------------------------------------------------------------------------

// GrantInviteeFirstOrderBonus credits the invitee a share of their first paid order.
// Idempotent: repeated calls (fulfillment retries) and concurrent orders grant at most once.
func (s *GrowthService) GrantInviteeFirstOrderBonus(ctx context.Context, inviteeID, orderID int64, orderAmount float64) (float64, error) {
	if s == nil || inviteeID <= 0 || orderID <= 0 || !finite(orderAmount) || orderAmount <= 0 {
		return 0, nil
	}
	if s.settingService != nil && !s.settingService.IsAffiliateEnabled(ctx) {
		return 0, nil
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return 0, err
	}
	if settings.InviteeBonusRatePercent <= 0 {
		return 0, nil
	}
	amount := math.Round(orderAmount*settings.InviteeBonusRatePercent) / 100
	if settings.InviteeBonusCap > 0 && amount > settings.InviteeBonusCap {
		amount = settings.InviteeBonusCap
	}
	if amount <= 0 {
		return 0, nil
	}
	// Same window as the inviter rebate: only invitees bound within the rebate validity period qualify.
	var notBefore *time.Time
	if s.settingService != nil {
		if days := s.settingService.GetAffiliateRebateDurationDays(ctx); days > 0 {
			t := time.Now().AddDate(0, 0, -days)
			notBefore = &t
		}
	}
	granted, _, err := s.repo.GrantInviteeBonus(ctx, inviteeID, orderID, amount, notBefore)
	if err != nil || !granted {
		return 0, err
	}
	if s.authInvalidator != nil {
		s.authInvalidator.InvalidateAuthCacheByUserID(ctx, inviteeID)
	}
	if s.billingCache != nil {
		if err := s.billingCache.InvalidateUserBalance(ctx, inviteeID); err != nil {
			return amount, nil
		}
	}
	return amount, nil
}

// ---------------------------------------------------------------------------
// Invite leaderboard
// ---------------------------------------------------------------------------

// leaderboardPeriod resolves a named period to [start, end). "all" has no bounds.
func leaderboardPeriod(period string, now time.Time) (string, *time.Time, *time.Time) {
	switch period {
	case "all":
		return "all", nil, nil
	case "last_month":
		end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		start := end.AddDate(0, -1, 0)
		return "last_month", &start, &end
	default:
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0)
		return "month", &start, &end
	}
}

// PublicLeaderboard returns the masked ranking shown to users, plus the viewer's own row.
func (s *GrowthService) PublicLeaderboard(ctx context.Context, viewerID int64, period string) (*InviteLeaderboard, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.LeaderboardEnabled || (s.settingService != nil && !s.settingService.IsAffiliateEnabled(ctx)) {
		return nil, ErrLeaderboardDisabled
	}
	name, start, end := leaderboardPeriod(period, time.Now())
	rows, err := s.repo.InviteLeaderboard(ctx, start, end, growthLeaderboardPublicLimitDefault)
	if err != nil {
		return nil, err
	}
	board := &InviteLeaderboard{Period: name, Start: start, End: end, Entries: make([]InviteLeaderboardEntry, 0, len(rows))}
	for _, row := range rows {
		board.Entries = append(board.Entries, publicLeaderboardEntry(row, viewerID))
	}
	if viewerID > 0 {
		me, err := s.repo.InviteLeaderboardEntryFor(ctx, viewerID, start, end)
		if err != nil {
			return nil, err
		}
		if me != nil {
			entry := publicLeaderboardEntry(*me, viewerID)
			board.Me = &entry
		}
	}
	return board, nil
}

// publicLeaderboardEntry keeps only the rank, a masked name and invite counts.
func publicLeaderboardEntry(row InviteLeaderboardEntry, viewerID int64) InviteLeaderboardEntry {
	display := maskSegment(strings.TrimSpace(row.Username))
	if strings.TrimSpace(row.Username) == "" {
		display = maskEmail(row.Email)
	}
	return InviteLeaderboardEntry{
		Rank:           row.Rank,
		DisplayName:    display,
		InvitedCount:   row.InvitedCount,
		PayingInvitees: row.PayingInvitees,
		IsCurrentUser:  viewerID > 0 && row.UserID == viewerID,
	}
}

// AdminLeaderboard returns the full ranking over [start, end) (nil = unbounded).
func (s *GrowthService) AdminLeaderboard(ctx context.Context, start, end *time.Time, limit int) (*InviteLeaderboard, error) {
	if limit <= 0 || limit > growthLeaderboardAdminLimitMax {
		limit = 100
	}
	if start != nil && end != nil && !end.After(*start) {
		return nil, infraerrors.BadRequest("INVALID_RANGE", "end must be after start")
	}
	rows, err := s.repo.InviteLeaderboard(ctx, start, end, limit)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].DisplayName = firstNonEmpty(rows[i].Username, rows[i].Email)
	}
	return &InviteLeaderboard{Period: "custom", Start: start, End: end, Entries: rows}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func parseFloatOr(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || !finite(v) {
		return fallback
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 { return math.Min(hi, math.Max(lo, v)) }

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
