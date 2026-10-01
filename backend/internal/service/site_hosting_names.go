package service

import (
	"context"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Chosen site names. A name is 3–30 lowercase letters, digits and inner hyphens; names that could
// pass for the platform, a payment service or an authority are refused. After a rename the old
// name redirects to the site and stays reserved for it for siteNameHold, so nobody can take over
// a link that is still being shared.

const (
	siteNameHold       = 30 * 24 * time.Hour
	siteRenameCooldown = 7 * 24 * time.Hour
)

var (
	ErrSiteNameInvalid  = infraerrors.BadRequest("SITE_NAME_INVALID", "站点名需要 3–30 个字符，只能用小写字母、数字和中间的短横线，并以字母开头（Use 3–30 lowercase letters, digits or inner hyphens, starting with a letter）")
	ErrSiteNameReserved = infraerrors.BadRequest("SITE_NAME_RESERVED", "这个站点名不能使用，请换一个（This name is reserved）")
	ErrSiteNameTaken    = infraerrors.Conflict("SITE_NAME_TAKEN", "这个站点名已经被使用，请换一个（This name is taken）")
)

func errSiteRenameTooSoon(at time.Time) error {
	return infraerrors.BadRequest("SITE_RENAME_TOO_SOON", "每 7 天只能改一次站点名，"+at.Format("01-02 15:04")+" 之后可以再改（You can rename once every 7 days）")
}

var siteChosenNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{1,28}[a-z0-9]$`)

// Exact names kept for the platform.
var siteReservedNames = map[string]bool{
	"www": true, "api": true, "app": true, "admin": true, "root": true, "mail": true, "email": true, "smtp": true, "ftp": true,
	"cdn": true, "static": true, "assets": true, "img": true, "files": true, "status": true, "docs": true, "doc": true, "blog": true,
	"help": true, "support": true, "kefu": true, "service": true, "login": true, "signin": true, "signup": true, "register": true,
	"account": true, "accounts": true, "auth": true, "oauth": true, "sso": true, "pay": true, "payment": true, "billing": true,
	"wallet": true, "shop": true, "store": true, "canvas": true, "site": true, "sites": true, "dashboard": true, "console": true,
	"test": true, "demo": true, "dev": true, "staging": true, "null": true, "undefined": true,
}

// Words that make a name look like the platform, a bank or payment service, or an authority.
var siteBlockedNameWords = []string{
	"hivegpt", "xinduanju", "official", "guanfang", "admin",
	"alipay", "zhifubao", "wechat", "weixin", "wxpay", "tenpay", "paypal", "unionpay", "yinlian", "icbc", "bank", "yinhang",
	"taobao", "tmall", "jd-com", "qq-com", "apple-id", "appleid", "icloud", "microsoft", "google", "openai", "anthropic",
	"gov", "zhengfu", "police", "gongan", "jingcha", "court", "fayuan", "tax", "shuiwu",
}

func normalizeSiteName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// validateSiteName checks the form and the reserved / blocked lists (not availability).
func validateSiteName(name string) error {
	if !siteChosenNameRe.MatchString(name) || strings.Contains(name, "--") {
		return ErrSiteNameInvalid
	}
	if siteReservedNames[name] {
		return ErrSiteNameReserved
	}
	for _, word := range siteBlockedNameWords {
		if strings.Contains(name, word) {
			return ErrSiteNameReserved
		}
	}
	return nil
}

// checkNameFree validates name and makes sure no other site uses or holds it.
func (s *SiteHostingService) checkNameFree(ctx context.Context, name string, siteID int64) error {
	if err := validateSiteName(name); err != nil {
		return err
	}
	taken, err := s.repo.SiteNameTaken(ctx, name, siteID, s.now().Add(-siteNameHold))
	if err != nil {
		return err
	}
	if taken {
		return ErrSiteNameTaken
	}
	return nil
}

// SiteNameCheck is the answer to "can I use this name?" (Reason is the user-facing message).
type SiteNameCheck struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

func (s *SiteHostingService) CheckName(ctx context.Context, siteID int64, name string) (SiteNameCheck, error) {
	name = normalizeSiteName(name)
	err := s.checkNameFree(ctx, name, siteID)
	if err == nil {
		return SiteNameCheck{Name: name, Available: true}, nil
	}
	if appErr := infraerrors.FromError(err); appErr != nil && appErr.Code < 500 {
		return SiteNameCheck{Name: name, Reason: appErr.Message}, nil
	}
	return SiteNameCheck{}, err
}

// Rename gives a site a new name; the old one redirects to it for siteNameHold.
func (s *SiteHostingService) Rename(ctx context.Context, userID, siteID int64, name string) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if site.Status == SiteStatusDisabled {
		return nil, ErrSiteDisabledByAdmin
	}
	name = normalizeSiteName(name)
	if name == site.Name {
		return s.mineByID(ctx, userID, siteID)
	}
	if site.RenamedAt != nil {
		if next := site.RenamedAt.Add(siteRenameCooldown); next.After(s.now()) {
			return nil, errSiteRenameTooSoon(next)
		}
	}
	if err := s.checkNameFree(ctx, name, site.ID); err != nil {
		return nil, err
	}
	if err := s.repo.RenameSite(ctx, site.ID, name, s.now()); err != nil {
		return nil, err
	}
	s.forget(site.Name)
	s.forget(name)
	if site.PreviousName != "" {
		s.forget(site.PreviousName)
	}
	return s.mineByID(ctx, userID, siteID)
}

// renamedTo returns the site that used name before a recent rename (nil when none).
func (s *SiteHostingService) renamedTo(ctx context.Context, name string) (*Site, error) {
	key := "\x00prev:" + name
	s.mu.Lock()
	if hit, ok := s.lookups[key]; ok && s.now().Sub(hit.at) < siteLookupTTL {
		s.mu.Unlock()
		return hit.site, nil
	}
	s.mu.Unlock()
	site, err := s.repo.GetSiteByPreviousName(ctx, name, s.now().Add(-siteNameHold))
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.lookups[key] = siteLookup{site: site, at: s.now()}
	s.mu.Unlock()
	return site, nil
}
