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
// utm_source / utm_medium / utm_campaign=<code> (and the partner's invite code, if any), so the
// analytics first touch ties visitors, sign-ups, activation and payments back to the link.

type ChannelLink struct {
	ID         int64     `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Source     string    `json:"source"`
	Medium     string    `json:"medium"`
	TargetPath string    `json:"target_path"`
	AffCode    string    `json:"aff_code"`
	Note       string    `json:"note"`
	Clicks     int64     `json:"clicks"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Funnel of the people whose first visit came through this link (members only).
	Visitors  int64   `json:"visitors"`
	Signups   int64   `json:"signups"`
	Activated int64   `json:"activated"` // made an API call
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`
}

type ChannelLinkInput struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Medium     string `json:"medium"`
	TargetPath string `json:"target_path"`
	AffCode    string `json:"aff_code"`
	Note       string `json:"note"`
}

type ChannelLinkRepository interface {
	List(ctx context.Context) ([]ChannelLink, error)
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
)

var (
	channelLinkCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,31}$`)
	channelLinkTagPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
)

type ChannelLinkService struct {
	repo      ChannelLinkRepository
	affiliate *AffiliateService
}

func NewChannelLinkService(repo ChannelLinkRepository, affiliate *AffiliateService) *ChannelLinkService {
	return &ChannelLinkService{repo: repo, affiliate: affiliate}
}

func (s *ChannelLinkService) List(ctx context.Context) ([]ChannelLink, error) {
	out, err := s.repo.List(ctx)
	if out == nil {
		out = []ChannelLink{}
	}
	return out, err
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
