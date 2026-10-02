package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Web-page works: a work presenting one of the author's hosted sites. The cover images are
// screenshots; the work page links to (and previews) the live site. It is public only while the
// site is up — a site taken down, unpaid or deleted hides its work from everyone but the author.

const (
	WorkKindImage = "image"
	WorkKindSite  = "site"
)

// WorkSite is the site behind a web-page work.
type WorkSite struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Title  string `json:"title"`
	Status string `json:"status"`
	// URL is set while the site is live (and always for its author).
	URL string `json:"url,omitempty"`
}

// Live: the site exists and is being served.
func (s *WorkSite) Live() bool { return s != nil && s.ID > 0 && s.Status == SiteStatusActive }

// communitySiteReader is the part of the site hosting repository the community needs.
type communitySiteReader interface {
	GetSite(ctx context.Context, id int64) (*Site, error)
	ListSitesByUser(ctx context.Context, userID int64) ([]Site, error)
}

// MySite is one of the user's sites in the canvas' "publish a web page" picker.
type MySite struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
	URL   string `json:"url"`
	// Publishable: live, no password and not presented by a work yet (WorkID says which).
	Publishable bool   `json:"publishable"`
	Reason      string `json:"reason,omitempty"`
	WorkID      int64  `json:"work_id,omitempty"`
}

var (
	ErrCommunitySiteNotFound         = infraerrors.NotFound("COMMUNITY_SITE_NOT_FOUND", "没有找到这个网站（Site not found）")
	ErrCommunitySiteNotLive          = infraerrors.BadRequest("COMMUNITY_SITE_NOT_LIVE", "网站通过审核、正常访问后才能发布到社区（The site must be live）")
	ErrCommunitySiteAlreadyPublished = infraerrors.Conflict("COMMUNITY_SITE_PUBLISHED", "这个网站已经发布过作品了，可以在作品页修改（This site already has a work）")
	ErrCommunitySitePassword         = infraerrors.BadRequest("COMMUNITY_SITE_PASSWORD", "设置了访问密码的网站不能发布到社区，请先取消密码（Remove the site password first）")
	ErrCommunitySitesDisabled        = infraerrors.ServiceUnavailable("COMMUNITY_SITES_DISABLED", "网站托管暂不可用（Site hosting is unavailable）")
)

// SetSites lets works present hosted sites; domain is the sites' domain (e.g. s.example.com).
func (s *CommunityService) SetSites(sites communitySiteReader, domain string) {
	s.sites, s.siteDomain = sites, domain
}

// workSiteFor checks the site a new web-page work presents: the author's own, and live.
func (s *CommunityService) workSiteFor(ctx context.Context, userID, siteID int64) (*WorkSite, error) {
	if s.sites == nil || s.siteDomain == "" {
		return nil, ErrCommunitySitesDisabled
	}
	site, err := s.sites.GetSite(ctx, siteID)
	if err != nil || site == nil || site.UserID != userID {
		return nil, ErrCommunitySiteNotFound
	}
	if site.Status != SiteStatusActive || site.Version <= 0 {
		return nil, ErrCommunitySiteNotLive
	}
	if site.HasPassword || site.PasswordHash != "" {
		return nil, ErrCommunitySitePassword
	}
	return &WorkSite{ID: site.ID, Name: site.Name, Title: site.Title, Status: site.Status}, nil
}

// decorateSite fills the site URL for viewers who may follow it.
func (s *CommunityService) decorateSite(w *Work, viewerID int64) {
	if w.Site == nil {
		return
	}
	if s.siteDomain != "" && w.Site.Name != "" && (w.Site.Live() || w.UserID == viewerID) {
		w.Site.URL = "https://" + w.Site.Name + "." + s.siteDomain
	}
}

// MySites lists the user's sites for the publish picker, saying which can become a work.
func (s *CommunityService) MySites(ctx context.Context, userID int64) ([]MySite, error) {
	if s.sites == nil || s.siteDomain == "" {
		return []MySite{}, nil
	}
	sites, err := s.sites.ListSitesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	published, err := s.repo.SiteWorkIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]MySite, 0, len(sites))
	for _, site := range sites {
		m := MySite{ID: site.ID, Name: site.Name, Title: site.Title, URL: "https://" + site.Name + "." + s.siteDomain, WorkID: published[site.ID]}
		switch {
		case site.Status != SiteStatusActive || site.Version <= 0:
			m.Reason = "not_live"
		case site.HasPassword || site.PasswordHash != "":
			m.Reason = "password"
		case m.WorkID > 0:
			m.Reason = "published"
		default:
			m.Publishable = true
		}
		out = append(out, m)
	}
	return out, nil
}
