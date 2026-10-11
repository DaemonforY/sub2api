package service

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Channel links (渠道链接): one short link per place the site is promoted — a 小红书 post, a B 站
// video, a partner's article. hivegpt.cn/go/<code> redirects to the landing page with
// utm_source / utm_medium / utm_campaign=<code> (and the partner's invite code and promo code, if
// any), so the analytics first touch ties visitors, sign-ups, activation and payments back to the
// link.
//
// For a blogger link (one with an invite code) the link's people are those whose first touch was
// the link plus everyone the blogger invited some other way (their own ?aff= link, typing the
// code): together that is the audience the blogger is paid for. The report sets what they paid
// against what their usage cost upstream and the rebates accrued on their orders.

type ChannelLink struct {
	ID         int64     `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Source     string    `json:"source"`
	Medium     string    `json:"medium"`
	TargetPath string    `json:"target_path"`
	AffCode    string    `json:"aff_code"`
	PromoCode  string    `json:"promo_code"`
	Note       string    `json:"note"`
	Clicks     int64     `json:"clicks"` // all time
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Funnel and money of the link's people (members only) within the report period: visitors and
	// sign-ups counted when they happened, payments, usage and rebates when they happened —
	// including from people who signed up before the period.
	Visitors  int64   `json:"visitors"`
	Signups   int64   `json:"signups"`
	Activated int64   `json:"activated"` // of those sign-ups, made an API call
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`   // CNY actually received, payment fee excluded
	UsageUSD  float64 `json:"usage_usd"` // usage at standard prices
	Rebates   float64 `json:"rebates"`   // CNY value of the rebates accrued on their orders
	// Filled in by the report from the 套餐毛利 cost rate; zero when the rate isn't set.
	CostCNY float64 `json:"cost_cny"`
	Profit  float64 `json:"profit"` // Revenue − CostCNY − Rebates
}

// ChannelLinkReport is the 渠道链接 page: every link with its numbers over the last Days days
// (0 = all time).
type ChannelLinkReport struct {
	Days       int           `json:"days"`
	CostPerUSD float64       `json:"cost_per_usd"` // 0 = not set, cost and profit left out
	Links      []ChannelLink `json:"links"`
}

type ChannelLinkInput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Medium     string `json:"medium"`
	TargetPath string `json:"target_path"`
	AffCode    string `json:"aff_code"`
	PromoCode  string `json:"promo_code"`
	Note       string `json:"note"`
}

type ChannelLinkRepository interface {
	// List returns every link, or only those carrying affCode when it isn't empty.
	List(ctx context.Context, since time.Time, affCode string) ([]ChannelLink, error)
	GetByCode(ctx context.Context, code string) (*ChannelLink, error) // nil when unknown
	Create(ctx context.Context, l *ChannelLink) error                 // ErrChannelLinkCodeTaken on a duplicate code
	Update(ctx context.Context, l *ChannelLink) error
	Delete(ctx context.Context, id int64) error
	AddClick(ctx context.Context, id int64) error
}

var (
	ErrChannelLinkNotFound  = infraerrors.NotFound("CHANNEL_LINK_NOT_FOUND", "渠道链接不存在（Channel link not found）")
	ErrChannelLinkCodeTaken = infraerrors.Conflict("CHANNEL_LINK_CODE_TAKEN", "这个短码已经被用了，换一个（This code is already taken）")
	errChannelLinkBadCode   = infraerrors.BadRequest("CHANNEL_LINK_BAD_CODE", "短码只能用小写字母、数字、- 和 _，2–32 个字符（Code: 2–32 lowercase letters, digits, - or _）")
	errChannelLinkBadName   = infraerrors.BadRequest("CHANNEL_LINK_BAD_NAME", "请填写名称，最多 100 字（Name is required, up to 100 characters）")
	errChannelLinkBadSource = infraerrors.BadRequest("CHANNEL_LINK_BAD_SOURCE", "渠道只能用小写字母、数字、- 和 _，最多 32 个字符（Source: up to 32 lowercase letters, digits, - or _）")
	errChannelLinkBadTarget = infraerrors.BadRequest("CHANNEL_LINK_BAD_TARGET", "落地页要填站内路径，以 / 开头，例如 /pricing（The landing page must be a path on this site, like /pricing）")
	errChannelLinkBadAff    = infraerrors.BadRequest("CHANNEL_LINK_BAD_AFF", "邀请码不存在（Unknown invite code）")
	errChannelLinkBadPromo  = infraerrors.BadRequest("CHANNEL_LINK_BAD_PROMO", "优惠码不存在、已停用或已过期，先到「优惠码」里建好（Unknown, disabled or expired promo code）")
)

var (
	channelLinkCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,31}$`)
	channelLinkTagPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
)

type ChannelLinkService struct {
	repo      ChannelLinkRepository
	affiliate *AffiliateService
	promo     *PromoService
}

func NewChannelLinkService(repo ChannelLinkRepository, affiliate *AffiliateService, promo *PromoService) *ChannelLinkService {
	return &ChannelLinkService{repo: repo, affiliate: affiliate, promo: promo}
}

// Report lists the links with their numbers over the last days days (0 = all time; anything else
// is clamped to 1–365). costPerUSD is the CNY cost of $1 of usage (套餐毛利); 0 leaves cost and
// profit out.
func (s *ChannelLinkService) Report(ctx context.Context, days int, costPerUSD float64) (*ChannelLinkReport, error) {
	since := time.Time{}
	if days != 0 {
		days = max(1, min(days, 365))
		since = time.Now().AddDate(0, 0, -days)
	}
	links, err := s.repo.List(ctx, since, "")
	if err != nil {
		return nil, err
	}
	if links == nil {
		links = []ChannelLink{}
	}
	if costPerUSD < 0 {
		costPerUSD = 0
	}
	for i := range links {
		applyChannelLinkCost(&links[i], costPerUSD)
	}
	return &ChannelLinkReport{Days: days, CostPerUSD: costPerUSD, Links: links}, nil
}

// PartnerLink is what a blogger sees of a link carrying their invite code on the 邀请返利 page:
// the funnel, what their people paid and the rebates it earned them — not cost or profit.
type PartnerLink struct {
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	PromoCode string  `json:"promo_code"`
	Clicks    int64   `json:"clicks"` // all time
	Visitors  int64   `json:"visitors"`
	Signups   int64   `json:"signups"`
	Activated int64   `json:"activated"`
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`
	Rebates   float64 `json:"rebates"`
}

type PartnerLinkReport struct {
	Days  int           `json:"days"`
	Links []PartnerLink `json:"links"`
}

// ForPartner lists the links that carry the user's invite code with their numbers over the last
// days days (0 = all time). Empty for users who have none.
func (s *ChannelLinkService) ForPartner(ctx context.Context, userID int64, days int) (*PartnerLinkReport, error) {
	out := &PartnerLinkReport{Links: []PartnerLink{}}
	since := time.Time{}
	if days != 0 {
		days = max(1, min(days, 365))
		since = time.Now().AddDate(0, 0, -days)
	}
	out.Days = days
	if s.affiliate == nil || s.affiliate.repo == nil {
		return out, nil
	}
	aff, err := s.affiliate.repo.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		return nil, err
	}
	if aff == nil || aff.AffCode == "" {
		return out, nil
	}
	links, err := s.repo.List(ctx, since, aff.AffCode)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		out.Links = append(out.Links, PartnerLink{
			Code: l.Code, Name: l.Name, PromoCode: l.PromoCode, Clicks: l.Clicks, Visitors: l.Visitors,
			Signups: l.Signups, Activated: l.Activated, PaidUsers: l.PaidUsers,
			Revenue: round2(l.Revenue), Rebates: round2(l.Rebates),
		})
	}
	return out, nil
}

func applyChannelLinkCost(l *ChannelLink, costPerUSD float64) {
	l.Revenue, l.UsageUSD, l.Rebates = round2(l.Revenue), round2(l.UsageUSD), round2(l.Rebates)
	if costPerUSD <= 0 {
		return
	}
	l.CostCNY = round2(l.UsageUSD * costPerUSD)
	l.Profit = round2(l.Revenue - l.CostCNY - l.Rebates)
}

func (s *ChannelLinkService) Create(ctx context.Context, in ChannelLinkInput) (*ChannelLink, error) {
	l, err := s.validate(ctx, in)
	if err != nil {
		return nil, err
	}
	if l.Code == "" {
		// source-xxxx, retried on the rare clash.
		for i := 0; i < 5; i++ {
			l.Code = truncateRunes(l.Source, 26) + "-" + randomChannelSuffix()
			if err = s.repo.Create(ctx, l); !errors.Is(err, ErrChannelLinkCodeTaken) {
				break
			}
		}
	} else {
		err = s.repo.Create(ctx, l)
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Update changes everything but the code — printed QR codes and posted links keep working.
func (s *ChannelLinkService) Update(ctx context.Context, id int64, in ChannelLinkInput) (*ChannelLink, error) {
	in.Code = ""
	l, err := s.validate(ctx, in)
	if err != nil {
		return nil, err
	}
	l.ID = id
	if err := s.repo.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (s *ChannelLinkService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Resolve returns where /go/<code> should send the visitor ("/" for an unknown code) and counts
// the click unless it came from a bot.
func (s *ChannelLinkService) Resolve(ctx context.Context, code, userAgent string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if !channelLinkCodePattern.MatchString(code) {
		return "/"
	}
	l, err := s.repo.GetByCode(ctx, code)
	if err != nil || l == nil {
		return "/"
	}
	if userAgent != "" && !analyticsBotPattern.MatchString(userAgent) {
		_ = s.repo.AddClick(ctx, l.ID)
	}
	return ChannelLinkTarget(l)
}

// ChannelLinkTarget is the landing URL (path + query) of a link.
func ChannelLinkTarget(l *ChannelLink) string {
	u, err := url.Parse(l.TargetPath)
	if err != nil {
		u = &url.URL{Path: "/"}
	}
	q := u.Query()
	q.Set("utm_source", l.Source)
	if l.Medium != "" {
		q.Set("utm_medium", l.Medium)
	}
	q.Set("utm_campaign", l.Code)
	if l.AffCode != "" {
		q.Set("aff", l.AffCode)
	}
	if l.PromoCode != "" {
		q.Set("promo", l.PromoCode)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *ChannelLinkService) validate(ctx context.Context, in ChannelLinkInput) (*ChannelLink, error) {
	l := &ChannelLink{
		Code:       strings.ToLower(strings.TrimSpace(in.Code)),
		Name:       strings.TrimSpace(in.Name),
		Source:     strings.ToLower(strings.TrimSpace(in.Source)),
		Medium:     strings.ToLower(strings.TrimSpace(in.Medium)),
		TargetPath: strings.TrimSpace(in.TargetPath),
		AffCode:    strings.ToUpper(strings.TrimSpace(in.AffCode)),
		PromoCode:  strings.ToUpper(strings.TrimSpace(in.PromoCode)),
		Note:       truncateRunes(strings.TrimSpace(in.Note), 300),
	}
	if l.Code != "" && !channelLinkCodePattern.MatchString(l.Code) {
		return nil, errChannelLinkBadCode
	}
	if l.Name == "" || utf8.RuneCountInString(l.Name) > 100 {
		return nil, errChannelLinkBadName
	}
	if !channelLinkTagPattern.MatchString(l.Source) || (l.Medium != "" && !channelLinkTagPattern.MatchString(l.Medium)) {
		return nil, errChannelLinkBadSource
	}
	if l.TargetPath == "" {
		l.TargetPath = "/"
	}
	if !channelLinkPathOK(l.TargetPath) {
		return nil, errChannelLinkBadTarget
	}
	if l.AffCode != "" {
		if s.affiliate == nil || s.affiliate.CheckCode(ctx, l.AffCode) != nil {
			return nil, errChannelLinkBadAff
		}
	}
	if l.PromoCode != "" {
		if len(l.PromoCode) > 32 || s.promo == nil {
			return nil, errChannelLinkBadPromo
		}
		if _, err := s.promo.ValidatePromoCode(ctx, l.PromoCode); err != nil {
			return nil, errChannelLinkBadPromo
		}
	}
	return l, nil
}

// Only paths on this site: no scheme, host, protocol-relative or backslash tricks.
func channelLinkPathOK(p string) bool {
	if len(p) > 200 || !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") || strings.ContainsAny(p, "\\ \t\r\n") {
		return false
	}
	u, err := url.Parse(p)
	return err == nil && u.Scheme == "" && u.Host == "" && u.Fragment == ""
}

func randomChannelSuffix() string {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
