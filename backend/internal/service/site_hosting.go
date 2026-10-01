package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Static-site hosting for subscribers: upload one HTML file or a zip and it is served at
// https://<name>.<domain>. Only users with an active subscription can publish. A few sites per
// subscriber are free; the rest are paid for every 30 days from the balance. When the
// subscription ends the sites stay up for a grace period, then go offline, then are deleted.

const (
	SiteStatusActive   = "active"
	SiteStatusDisabled = "disabled" // taken down by an admin
	SiteStatusUnpaid   = "unpaid"   // renewal could not be charged
	SiteStatusLapsed   = "lapsed"   // the owner's subscription ended (after the grace period)
	SiteStatusPending  = "pending"  // new site waiting for its first review

	sitePeriod        = 30 * 24 * time.Hour
	siteKeepVersions  = 3
	siteSettingsTTL   = 30 * time.Second
	siteLookupTTL     = 30 * time.Second
	siteTitleMaxRunes = 60

	settingSitesEnabled       = "sites_enabled"
	settingSitesMaxPerUser    = "sites_max_per_user"
	settingSitesMaxMB         = "sites_max_mb"
	settingSitesMaxFiles      = "sites_max_files"
	settingSitesFreePerUser   = "sites_free_per_user"
	settingSitesExtraPrice    = "sites_extra_price"
	settingSitesGraceDays     = "sites_grace_days"
	settingSitesRetentionDays = "sites_retention_days"
	settingSitesReviewAll     = "sites_review_all"
	settingSitesReviewBaseURL = "sites_review_base_url"
	settingSitesReviewModel   = "sites_review_model"
	settingSitesReviewAPIKey  = "sites_review_api_key"
)

var (
	ErrSitesDisabled             = infraerrors.ServiceUnavailable("SITES_DISABLED", "网站发布暂未开放（Site hosting is not available）")
	ErrSiteSubscriptionRequired  = infraerrors.Forbidden("SITE_SUBSCRIPTION_REQUIRED", "网站发布只对订阅用户开放：请先在 hivegpt.cn 购买订阅套餐（A subscription is required）")
	ErrSiteNotFound              = infraerrors.NotFound("SITE_NOT_FOUND", "网站不存在（Site not found）")
	ErrSiteDisabledByAdmin       = infraerrors.Forbidden("SITE_DISABLED", "该网站已被管理员下线，不能更新；如有疑问请联系客服（Site was taken down）")
	ErrSiteNotRenewable          = infraerrors.BadRequest("SITE_NOT_RENEWABLE", "该网站不需要续费（Site does not need renewal）")
	ErrSiteSettingsInvalid       = infraerrors.BadRequest("SITE_SETTINGS_INVALID", "设置超出范围：网站数 1–50、大小 1–200MB、文件数 1–5000、免费网站数 0–50、价格 0–1000、天数 0–365（Invalid settings）")
	ErrSiteSettingsUnavailable   = infraerrors.ServiceUnavailable("SITE_SETTINGS_UNAVAILABLE", "设置暂时无法保存，请稍后再试（Settings are unavailable）")
	ErrSiteReportInvalid         = infraerrors.BadRequest("SITE_REPORT_INVALID", "请选择举报原因（Choose a reason）")
	ErrSiteReportTooMany         = infraerrors.TooManyRequests("SITE_REPORT_TOO_MANY", "举报太频繁了，请稍后再试（Too many reports）")
	ErrSiteStatusInvalid         = infraerrors.BadRequest("SITE_STATUS_INVALID", "只能设为上线或下线（Invalid status）")
	errSiteInsufficientForCreate = "余额不足：你已有 %d 个免费网站，再建一个需要 ¥%s / 30 天，请先到 hivegpt.cn 充值（Insufficient balance）"
)

func siteLimitReached(max int) error {
	return infraerrors.Forbidden("SITE_LIMIT_REACHED", fmt.Sprintf("每人最多 %d 个网站，请先删除不用的网站（Site limit reached）", max))
}

// SiteReportReasons are the choices on the report form.
var SiteReportReasons = map[string]bool{"phishing": true, "fraud": true, "gambling": true, "porn": true, "malware": true, "copyright": true, "other": true}

type Site struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	UserEmail    string `json:"user_email,omitempty"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	StatusReason string `json:"status_reason"`
	Version      int    `json:"version"`
	// PendingVersion is an uploaded version waiting for review (0: none).
	PendingVersion int        `json:"pending_version"`
	HasPassword    bool       `json:"has_password"`
	PasswordHash   string     `json:"-"`
	Views7d        int64      `json:"views_7d"`
	ViewsTotal     int64      `json:"views_total"`
	SizeBytes      int64      `json:"size_bytes"`
	FileCount      int        `json:"file_count"`
	Paid           bool       `json:"paid"`
	PaidUntil      *time.Time `json:"paid_until,omitempty"`
	LapsedAt       *time.Time `json:"lapsed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	URL            string     `json:"url"`
	// PreviewURL opens the pending version (owner and admins only).
	PreviewURL string `json:"preview_url,omitempty"`
	// PreviousName redirects to this site for siteNameHold after a rename (and stays reserved for it).
	PreviousName string     `json:"previous_name,omitempty"`
	RenamedAt    *time.Time `json:"renamed_at,omitempty"`
	// RenameAfter is when the owner may rename again (absent: now).
	RenameAfter *time.Time `json:"rename_after,omitempty"`
}

type SiteCharge struct {
	ID        int64     `json:"id"`
	SiteID    *int64    `json:"site_id,omitempty"`
	SiteName  string    `json:"site_name"`
	Amount    float64   `json:"amount"`
	PeriodEnd time.Time `json:"period_end"`
	CreatedAt time.Time `json:"created_at"`
}

type SiteReport struct {
	ID         int64      `json:"id"`
	SiteID     *int64     `json:"site_id,omitempty"`
	SiteName   string     `json:"site_name"`
	SiteStatus string     `json:"site_status"`
	OwnerEmail string     `json:"owner_email"`
	Reason     string     `json:"reason"`
	Detail     string     `json:"detail"`
	Contact    string     `json:"contact"`
	ReporterIP string     `json:"reporter_ip"`
	Status     string     `json:"status"`
	HandledAt  *time.Time `json:"handled_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type SiteListQuery struct {
	Keyword  string // name, title or owner email
	Status   string
	Page     int
	PageSize int
}

type SiteHostingRepository interface {
	CreateSite(ctx context.Context, site *Site) error
	GetSite(ctx context.Context, id int64) (*Site, error)
	GetSiteByName(ctx context.Context, name string) (*Site, error)
	ListSitesByUser(ctx context.Context, userID int64) ([]Site, error)
	ListSites(ctx context.Context, q SiteListQuery) ([]Site, int64, error)
	ListSiteOwners(ctx context.Context) ([]int64, error)
	LatestVersion(ctx context.Context, siteID int64) (int, error)
	// AddSiteVersion records an uploaded version and its review.
	AddSiteVersion(ctx context.Context, siteID int64, version int, size int64, files int, review SiteReview) error
	// ServeSiteVersion switches the site to a version (size and file count follow); clearPending drops the queued one.
	ServeSiteVersion(ctx context.Context, siteID int64, version int, clearPending bool) error
	// SetPendingVersion queues a version for review (an older queued one is superseded).
	SetPendingVersion(ctx context.Context, siteID int64, version int) error
	SetVersionReview(ctx context.Context, siteID int64, version int, status, reason, by string) error
	ListVersions(ctx context.Context, siteID int64) ([]SiteVersion, error)
	ListPendingReviews(ctx context.Context, page, pageSize int) ([]SiteReviewItem, int64, error)
	SetSitePassword(ctx context.Context, id int64, hash string) error
	AddDailyStats(ctx context.Context, stats []SiteDailyStat) error
	ListDailyStats(ctx context.Context, siteID int64, since time.Time) ([]SiteDailyStat, error)
	SetSiteTitle(ctx context.Context, id int64, title string) error
	// SiteNameTaken: another site uses the name, or held it as its previous name since holdSince.
	SiteNameTaken(ctx context.Context, name string, exceptSiteID int64, holdSince time.Time) (bool, error)
	// RenameSite keeps the old name as previous_name; ErrSiteNameTaken on a unique violation.
	RenameSite(ctx context.Context, id int64, name string, at time.Time) error
	GetSiteByPreviousName(ctx context.Context, name string, holdSince time.Time) (*Site, error)
	SetSiteStatus(ctx context.Context, id int64, status, reason string) error
	SetSiteFree(ctx context.Context, id int64) error
	SetLapsedAt(ctx context.Context, userID int64, at *time.Time) error
	DeleteSite(ctx context.Context, id int64) error
	// ChargeSite takes amount from the balance (never below zero) and extends paid_until; ErrInsufficientBalance otherwise.
	ChargeSite(ctx context.Context, site *Site, amount float64, periodEnd time.Time) error
	ListCharges(ctx context.Context, userID int64, limit int) ([]SiteCharge, error)
	CreateReport(ctx context.Context, r *SiteReport) error
	CountReportsSince(ctx context.Context, ip string, since time.Time) (int, error)
	ListReports(ctx context.Context, status string, page, pageSize int) ([]SiteReport, int64, error)
	SetReportStatus(ctx context.Context, id int64, status string) error
	Balance(ctx context.Context, userID int64) (float64, error)
}

// SiteHostingConfig is the configuration with the admin's overrides on top.
type SiteHostingConfig struct {
	Domain        string  `json:"domain"`
	Enabled       bool    `json:"enabled"`
	MaxPerUser    int     `json:"max_per_user"`
	MaxMB         int     `json:"max_mb"`
	MaxFiles      int     `json:"max_files"`
	FreePerUser   int     `json:"free_per_user"`
	ExtraPrice    float64 `json:"extra_price"`
	GraceDays     int     `json:"grace_days"`
	RetentionDays int     `json:"retention_days"`
	// ReviewAll sends every upload to the admin review queue.
	ReviewAll bool `json:"review_all"`
	// The model that reviews uploads (OpenAI-compatible); the key is write-only.
	ReviewBaseURL          string `json:"review_base_url"`
	ReviewModel            string `json:"review_model"`
	ReviewAPIKeyConfigured bool   `json:"review_api_key_configured"`
	ReviewAPIKey           string `json:"review_api_key,omitempty"`
	ClearReviewAPIKey      bool   `json:"clear_review_api_key,omitempty"`
}

func (c SiteHostingConfig) maxBytes() int64 { return int64(c.MaxMB) << 20 }
func (c SiteHostingConfig) available() bool { return c.Domain != "" && c.Enabled }

// DefaultSiteHostingConfig holds the defaults the admin can change.
func DefaultSiteHostingConfig(domain string) SiteHostingConfig {
	return SiteHostingConfig{Domain: domain, Enabled: true, MaxPerUser: 3, MaxMB: 20, MaxFiles: 500, FreePerUser: 1, ExtraPrice: 5, GraceDays: 7, RetentionDays: 30}
}

type SiteHostingService struct {
	repo        SiteHostingRepository
	subs        imageToolSubscriptions
	cache       imageToolBalanceCache
	settings    imageToolSettings
	defaults    SiteHostingConfig
	dir         string
	mainSiteURL string
	secret      []byte
	httpClient  *http.Client
	stats       *siteStatsCollector
	now         func() time.Time

	mu          sync.Mutex
	cfg         *SiteHostingConfig
	cfgAt       time.Time
	lookups     map[string]siteLookup
	unlockTries map[string][]time.Time
	stopOnce    sync.Once
	stop        chan struct{}
}

type siteLookup struct {
	site *Site
	at   time.Time
}

func NewSiteHostingService(repo SiteHostingRepository, subs imageToolSubscriptions, cache imageToolBalanceCache, settings imageToolSettings, defaults SiteHostingConfig, dir, mainSiteURL string) *SiteHostingService {
	defaults.Domain = strings.Trim(strings.ToLower(strings.TrimSpace(defaults.Domain)), ".")
	svc := &SiteHostingService{repo: repo, subs: subs, cache: cache, settings: settings, defaults: defaults, dir: dir,
		mainSiteURL: strings.TrimRight(mainSiteURL, "/"), httpClient: &http.Client{Timeout: time.Minute}, now: time.Now,
		lookups: map[string]siteLookup{}, stop: make(chan struct{})}
	svc.stats = newSiteStatsCollector(svc.now)
	svc.secret = []byte("hivegpt-sites:" + dir)
	return svc
}

// WithSecret sets the key that signs access-password and preview cookies (use a server secret so
// they survive restarts).
func (s *SiteHostingService) WithSecret(secret string) *SiteHostingService {
	if secret != "" {
		s.secret = []byte("hivegpt-sites:" + secret)
	}
	return s
}

// Domain is the hosting domain ("" when hosting is off).
func (s *SiteHostingService) Domain() string { return s.defaults.Domain }

// Config returns the effective settings (cached briefly).
func (s *SiteHostingService) Config(ctx context.Context) SiteHostingConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg != nil && s.now().Sub(s.cfgAt) < siteSettingsTTL {
		return *s.cfg
	}
	c := s.defaults
	if s.settings != nil {
		keys := []string{settingSitesEnabled, settingSitesMaxPerUser, settingSitesMaxMB, settingSitesMaxFiles, settingSitesFreePerUser, settingSitesExtraPrice,
			settingSitesGraceDays, settingSitesRetentionDays, settingSitesReviewAll, settingSitesReviewBaseURL, settingSitesReviewModel, settingSitesReviewAPIKey}
		if values, err := s.settings.GetMultiple(ctx, keys); err == nil {
			if values[settingSitesEnabled] == "false" {
				c.Enabled = false
			}
			c.ReviewAll = values[settingSitesReviewAll] == "true"
			c.ReviewBaseURL, c.ReviewModel = values[settingSitesReviewBaseURL], values[settingSitesReviewModel]
			c.ReviewAPIKeyConfigured = values[settingSitesReviewAPIKey] != ""
			for key, target := range map[string]*int{settingSitesMaxPerUser: &c.MaxPerUser, settingSitesMaxMB: &c.MaxMB, settingSitesMaxFiles: &c.MaxFiles,
				settingSitesFreePerUser: &c.FreePerUser, settingSitesGraceDays: &c.GraceDays, settingSitesRetentionDays: &c.RetentionDays} {
				if v, err := strconv.Atoi(values[key]); err == nil {
					*target = v
				}
			}
			if v, err := strconv.ParseFloat(values[settingSitesExtraPrice], 64); err == nil {
				c.ExtraPrice = v
			}
		}
	}
	s.cfg, s.cfgAt = &c, s.now()
	return c
}

func (s *SiteHostingService) SaveConfig(ctx context.Context, in SiteHostingConfig) (SiteHostingConfig, error) {
	if s.settings == nil {
		return SiteHostingConfig{}, ErrSiteSettingsUnavailable
	}
	if in.MaxPerUser < 1 || in.MaxPerUser > 50 || in.MaxMB < 1 || in.MaxMB > 200 || in.MaxFiles < 1 || in.MaxFiles > 5000 ||
		in.FreePerUser < 0 || in.FreePerUser > 50 || in.ExtraPrice < 0 || in.ExtraPrice > 1000 || in.GraceDays < 0 || in.GraceDays > 365 ||
		in.RetentionDays < 0 || in.RetentionDays > 365 {
		return SiteHostingConfig{}, ErrSiteSettingsInvalid
	}
	base := strings.TrimRight(strings.TrimSpace(in.ReviewBaseURL), "/")
	if base != "" && !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return SiteHostingConfig{}, ErrPromptTranslateInvalidURL
	}
	values := map[string]string{
		settingSitesEnabled: strconv.FormatBool(in.Enabled), settingSitesMaxPerUser: strconv.Itoa(in.MaxPerUser), settingSitesMaxMB: strconv.Itoa(in.MaxMB),
		settingSitesMaxFiles: strconv.Itoa(in.MaxFiles), settingSitesFreePerUser: strconv.Itoa(in.FreePerUser),
		settingSitesExtraPrice: strconv.FormatFloat(in.ExtraPrice, 'f', -1, 64), settingSitesGraceDays: strconv.Itoa(in.GraceDays),
		settingSitesRetentionDays: strconv.Itoa(in.RetentionDays), settingSitesReviewAll: strconv.FormatBool(in.ReviewAll),
		settingSitesReviewBaseURL: base, settingSitesReviewModel: strings.TrimSpace(in.ReviewModel),
	}
	if key := strings.TrimSpace(in.ReviewAPIKey); key != "" {
		values[settingSitesReviewAPIKey] = key
	} else if in.ClearReviewAPIKey {
		values[settingSitesReviewAPIKey] = ""
	}
	err := s.settings.SetMultiple(ctx, values)
	if err != nil {
		return SiteHostingConfig{}, err
	}
	s.mu.Lock()
	s.cfg = nil
	s.mu.Unlock()
	return s.Config(ctx), nil
}

func (s *SiteHostingService) siteURL(name string) string {
	if s.defaults.Domain == "" {
		return ""
	}
	return "https://" + name + "." + s.defaults.Domain
}

func (s *SiteHostingService) decorate(sites []Site) []Site {
	for i := range sites {
		s.decorateOne(&sites[i])
	}
	return sites
}

func (s *SiteHostingService) decorateOne(site *Site) {
	site.URL = s.siteURL(site.Name)
	if site.RenamedAt != nil {
		if next := site.RenamedAt.Add(siteRenameCooldown); next.After(s.now()) {
			site.RenameAfter = &next
		}
	}
	if site.PendingVersion > 0 {
		site.PreviewURL = s.PreviewURL(site.Name, site.ID, site.PendingVersion)
	}
}

func (s *SiteHostingService) subscribed(ctx context.Context, userID int64) (bool, error) {
	if s.subs == nil {
		return false, nil
	}
	subs, err := s.subs.ListActiveByUserID(ctx, userID)
	return len(subs) > 0, err
}

// MySitesQuota tells the user what they can publish.
type MySitesQuota struct {
	Available  bool    `json:"available"`
	Subscribed bool    `json:"subscribed"`
	Domain     string  `json:"domain"`
	MaxSites   int     `json:"max_sites"`
	Used       int     `json:"used"`
	FreeSites  int     `json:"free_sites"`
	FreeUsed   int     `json:"free_used"`
	ExtraPrice float64 `json:"extra_price"`
	MaxMB      int     `json:"max_mb"`
	MaxFiles   int     `json:"max_files"`
	GraceDays  int     `json:"grace_days"`
	Balance    float64 `json:"balance"`
}

type MySites struct {
	Sites   []Site       `json:"sites"`
	Quota   MySitesQuota `json:"quota"`
	Charges []SiteCharge `json:"charges"`
}

func (s *SiteHostingService) Mine(ctx context.Context, userID int64) (*MySites, error) {
	c := s.Config(ctx)
	sites, err := s.repo.ListSitesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	subscribed, err := s.subscribed(ctx, userID)
	if err != nil {
		return nil, err
	}
	balance, err := s.repo.Balance(ctx, userID)
	if err != nil {
		return nil, err
	}
	charges, err := s.repo.ListCharges(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	if sites == nil {
		sites = []Site{}
	}
	if charges == nil {
		charges = []SiteCharge{}
	}
	q := MySitesQuota{Available: c.available(), Subscribed: subscribed, Domain: c.Domain, MaxSites: c.MaxPerUser, Used: len(sites),
		FreeSites: c.FreePerUser, ExtraPrice: c.ExtraPrice, MaxMB: c.MaxMB, MaxFiles: c.MaxFiles, GraceDays: c.GraceDays, Balance: balance}
	for _, site := range sites {
		if !site.Paid {
			q.FreeUsed++
		}
	}
	return &MySites{Sites: s.decorate(sites), Quota: q, Charges: charges}, nil
}

// SiteUpload is a new version: the uploaded file (optional when only the title changes) and a title.
type SiteUpload struct {
	Title string
	// Name is the chosen site name ("" picks a random one); only used when creating.
	Name     string
	FileName string
	Data     []byte
}

func (s *SiteHostingService) checkPublisher(ctx context.Context, userID int64) (SiteHostingConfig, error) {
	c := s.Config(ctx)
	if !c.available() {
		return c, ErrSitesDisabled
	}
	subscribed, err := s.subscribed(ctx, userID)
	if err != nil {
		return c, err
	}
	if !subscribed {
		return c, ErrSiteSubscriptionRequired
	}
	return c, nil
}

// Create publishes a new site.
func (s *SiteHostingService) Create(ctx context.Context, userID int64, up SiteUpload) (*Site, error) {
	c, err := s.checkPublisher(ctx, userID)
	if err != nil {
		return nil, err
	}
	sites, err := s.repo.ListSitesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(sites) >= c.MaxPerUser {
		return nil, siteLimitReached(c.MaxPerUser)
	}
	freeUsed := 0
	for _, site := range sites {
		if !site.Paid {
			freeUsed++
		}
	}
	paid := freeUsed >= c.FreePerUser && c.ExtraPrice > 0
	if paid {
		balance, err := s.repo.Balance(ctx, userID)
		if err != nil {
			return nil, err
		}
		if balance < c.ExtraPrice {
			return nil, infraerrors.Forbidden("SITE_INSUFFICIENT_BALANCE", fmt.Sprintf(errSiteInsufficientForCreate, freeUsed, strconv.FormatFloat(c.ExtraPrice, 'f', -1, 64)))
		}
	}
	files, err := unpackSiteUpload(up.FileName, up.Data, c.maxBytes(), c.MaxFiles)
	if err != nil {
		return nil, err
	}
	site := &Site{UserID: userID, Title: cleanSiteTitle(up.Title), Status: SiteStatusActive, Paid: paid}
	if up.Name != "" {
		site.Name = normalizeSiteName(up.Name)
		if err := s.checkNameFree(ctx, site.Name, 0); err != nil {
			return nil, err
		}
		if err := s.repo.CreateSite(ctx, site); err != nil {
			return nil, err
		}
	} else {
		for attempt := 0; ; attempt++ {
			site.Name = randomSiteName()
			if err = s.repo.CreateSite(ctx, site); err == nil || attempt >= 4 {
				break
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if err := s.publish(ctx, site, files); err != nil {
		_ = s.remove(ctx, site)
		return nil, err
	}
	if paid {
		if err := s.repo.ChargeSite(ctx, site, c.ExtraPrice, s.now().Add(sitePeriod)); err != nil {
			_ = s.remove(ctx, site)
			if errors.Is(err, ErrInsufficientBalance) {
				return nil, infraerrors.Forbidden("SITE_INSUFFICIENT_BALANCE", fmt.Sprintf(errSiteInsufficientForCreate, freeUsed, strconv.FormatFloat(c.ExtraPrice, 'f', -1, 64)))
			}
			return nil, err
		}
		s.invalidateBalance(ctx, userID)
	}
	return s.mineByID(ctx, userID, site.ID)
}

// Update publishes a new version and/or renames the site.
func (s *SiteHostingService) Update(ctx context.Context, userID, siteID int64, up SiteUpload) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if site.Status == SiteStatusDisabled {
		return nil, ErrSiteDisabledByAdmin
	}
	c, err := s.checkPublisher(ctx, userID)
	if err != nil {
		return nil, err
	}
	if title := cleanSiteTitle(up.Title); title != site.Title && strings.TrimSpace(up.Title) != "" {
		if err := s.repo.SetSiteTitle(ctx, site.ID, title); err != nil {
			return nil, err
		}
	}
	if len(up.Data) > 0 {
		files, err := unpackSiteUpload(up.FileName, up.Data, c.maxBytes(), c.MaxFiles)
		if err != nil {
			return nil, err
		}
		if err := s.publish(ctx, site, files); err != nil {
			return nil, err
		}
	}
	s.forget(site.Name)
	return s.mineByID(ctx, userID, site.ID)
}

func (s *SiteHostingService) Delete(ctx context.Context, userID, siteID int64) error {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return err
	}
	return s.remove(ctx, site)
}

// Renew pays an unpaid site again and brings it back online.
func (s *SiteHostingService) Renew(ctx context.Context, userID, siteID int64) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if site.Status != SiteStatusUnpaid {
		return nil, ErrSiteNotRenewable
	}
	c, err := s.checkPublisher(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.renew(ctx, site, c); err != nil {
		return nil, err
	}
	return s.mineByID(ctx, userID, site.ID)
}

func (s *SiteHostingService) renew(ctx context.Context, site *Site, c SiteHostingConfig) error {
	if err := s.repo.ChargeSite(ctx, site, c.ExtraPrice, s.now().Add(sitePeriod)); err != nil {
		if errors.Is(err, ErrInsufficientBalance) {
			return infraerrors.Forbidden("SITE_INSUFFICIENT_BALANCE", fmt.Sprintf("余额不足：续费需要 ¥%s / 30 天，请先到 hivegpt.cn 充值（Insufficient balance）", strconv.FormatFloat(c.ExtraPrice, 'f', -1, 64)))
		}
		return err
	}
	s.invalidateBalance(ctx, site.UserID)
	if site.Status == SiteStatusUnpaid {
		if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusActive, ""); err != nil {
			return err
		}
	}
	s.forget(site.Name)
	return nil
}

func (s *SiteHostingService) invalidateBalance(ctx context.Context, userID int64) {
	if s.cache != nil {
		_ = s.cache.InvalidateUserBalance(ctx, userID)
	}
}

func (s *SiteHostingService) owned(ctx context.Context, userID, siteID int64) (*Site, error) {
	site, err := s.repo.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if site == nil || site.UserID != userID {
		return nil, ErrSiteNotFound
	}
	return site, nil
}

func (s *SiteHostingService) mineByID(ctx context.Context, userID, siteID int64) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	s.decorateOne(site)
	return site, nil
}

func (s *SiteHostingService) siteDir(id int64) string {
	return filepath.Join(s.dir, strconv.FormatInt(id, 10))
}

func (s *SiteHostingService) versionDir(id int64, version int) string {
	return filepath.Join(s.siteDir(id), "v"+strconv.Itoa(version))
}

// publish writes the files as a new version and reviews it: approved versions are served at once,
// others wait in the admin queue (the previous version stays online meanwhile).
func (s *SiteHostingService) publish(ctx context.Context, site *Site, files []siteFile) error {
	latest, err := s.repo.LatestVersion(ctx, site.ID)
	if err != nil {
		return err
	}
	version := latest + 1
	dir := s.versionDir(site.ID, version)
	_ = os.RemoveAll(dir)
	if err := writeSiteFiles(dir, files); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	var size int64
	for _, f := range files {
		size += int64(len(f.Data))
	}
	review := s.reviewSite(ctx, s.Config(ctx), files)
	if err := s.repo.AddSiteVersion(ctx, site.ID, version, size, len(files), review); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	if review.Status == SiteReviewApproved {
		if err := s.repo.ServeSiteVersion(ctx, site.ID, version, true); err != nil {
			return err
		}
		site.Version, site.PendingVersion, site.SizeBytes, site.FileCount = version, 0, size, len(files)
		if site.Status == SiteStatusPending {
			if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusActive, ""); err != nil {
				return err
			}
		} else if site.StatusReason != "" && site.Status == SiteStatusActive {
			_ = s.repo.SetSiteStatus(ctx, site.ID, SiteStatusActive, "")
		}
	} else {
		if err := s.repo.SetPendingVersion(ctx, site.ID, version); err != nil {
			return err
		}
		site.PendingVersion = version
		if site.Version == 0 {
			if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusPending, review.Reason); err != nil {
				return err
			}
			site.Status = SiteStatusPending
		}
		slog.Info("sites: version waits for review", "site", site.Name, "version", version, "flags", review.Flags, "by", review.By)
	}
	s.pruneVersions(site.ID, version, site.Version, site.PendingVersion)
	s.forget(site.Name)
	return nil
}

// pruneVersions keeps the newest few versions plus the served and pending ones on disk.
func (s *SiteHostingService) pruneVersions(siteID int64, latest int, keep ...int) {
	entries, err := os.ReadDir(s.siteDir(siteID))
	if err != nil {
		return
	}
	for _, e := range entries {
		v, err := strconv.Atoi(strings.TrimPrefix(e.Name(), "v"))
		if err != nil || !e.IsDir() || v > latest-siteKeepVersions {
			continue
		}
		kept := false
		for _, k := range keep {
			kept = kept || k == v
		}
		if !kept {
			_ = os.RemoveAll(s.versionDir(siteID, v))
		}
	}
}

func (s *SiteHostingService) remove(ctx context.Context, site *Site) error {
	if err := s.repo.DeleteSite(ctx, site.ID); err != nil {
		return err
	}
	_ = os.RemoveAll(s.siteDir(site.ID))
	s.forget(site.Name)
	return nil
}

func cleanSiteTitle(title string) string {
	return truncateRunes(strings.TrimSpace(title), siteTitleMaxRunes)
}

var siteNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

const siteNameAlphabet = "abcdefghijkmnpqrstuvwxyz23456789"

// randomSiteName makes a 7-character name starting with a letter (no 0/1/l/o to avoid confusion).
func randomSiteName() string {
	b := make([]byte, 7)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(siteNameAlphabet))))
		b[i] = siteNameAlphabet[n.Int64()]
		if i == 0 && b[i] >= '2' && b[i] <= '9' {
			b[i] = 's'
		}
	}
	return string(b)
}

// Admin -----------------------------------------------------------------------------------------

func (s *SiteHostingService) AdminList(ctx context.Context, q SiteListQuery) ([]Site, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	sites, total, err := s.repo.ListSites(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	if sites == nil {
		sites = []Site{}
	}
	return s.decorate(sites), total, nil
}

// AdminSetStatus takes a site down (disabled, with a reason shown to the owner) or puts it back.
func (s *SiteHostingService) AdminSetStatus(ctx context.Context, siteID int64, status, reason string) error {
	if status != SiteStatusActive && status != SiteStatusDisabled {
		return ErrSiteStatusInvalid
	}
	site, err := s.repo.GetSite(ctx, siteID)
	if err != nil {
		return err
	}
	if site == nil {
		return ErrSiteNotFound
	}
	if status == SiteStatusActive {
		reason = ""
	}
	if err := s.repo.SetSiteStatus(ctx, siteID, status, truncateRunes(strings.TrimSpace(reason), 300)); err != nil {
		return err
	}
	s.forget(site.Name)
	return nil
}

func (s *SiteHostingService) AdminDelete(ctx context.Context, siteID int64) error {
	site, err := s.repo.GetSite(ctx, siteID)
	if err != nil {
		return err
	}
	if site == nil {
		return ErrSiteNotFound
	}
	return s.remove(ctx, site)
}

// Report records a visitor's abuse report.
func (s *SiteHostingService) Report(ctx context.Context, name, reason, detail, contact, ip string) error {
	if !SiteReportReasons[reason] {
		return ErrSiteReportInvalid
	}
	name = strings.ToLower(strings.TrimSpace(name))
	site, err := s.repo.GetSiteByName(ctx, name)
	if err != nil {
		return err
	}
	if site == nil {
		return ErrSiteNotFound
	}
	if n, err := s.repo.CountReportsSince(ctx, ip, s.now().Add(-time.Hour)); err != nil {
		return err
	} else if n >= 5 {
		return ErrSiteReportTooMany
	}
	return s.repo.CreateReport(ctx, &SiteReport{SiteID: &site.ID, SiteName: site.Name, Reason: reason, Detail: truncateRunes(strings.TrimSpace(detail), 1000),
		Contact: truncateRunes(strings.TrimSpace(contact), 200), ReporterIP: ip})
}

func (s *SiteHostingService) AdminReports(ctx context.Context, status string, page, pageSize int) ([]SiteReport, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	reports, total, err := s.repo.ListReports(ctx, status, page, pageSize)
	if reports == nil {
		reports = []SiteReport{}
	}
	return reports, total, err
}

func (s *SiteHostingService) AdminSetReportStatus(ctx context.Context, id int64, status string) error {
	if status != "open" && status != "resolved" && status != "dismissed" {
		return ErrSiteStatusInvalid
	}
	return s.repo.SetReportStatus(ctx, id, status)
}

// Maintenance -----------------------------------------------------------------------------------

// Start runs Maintain every hour (first run a minute after boot) and saves visit stats every minute.
func (s *SiteHostingService) Start() {
	s.startStatsFlusher()
	go func() {
		timer := time.NewTimer(time.Minute)
		defer timer.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-timer.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				if err := s.Maintain(ctx); err != nil {
					slog.Warn("sites: maintenance failed", "error", err)
				}
				cancel()
				timer.Reset(time.Hour)
			}
		}
	}()
}

func (s *SiteHostingService) Stop() {
	s.stopOnce.Do(func() { close(s.stop) })
}

// Maintain applies subscription lapses (grace, offline, deletion) and 30-day renewals of paid sites.
func (s *SiteHostingService) Maintain(ctx context.Context) error {
	c := s.Config(ctx)
	owners, err := s.repo.ListSiteOwners(ctx)
	if err != nil {
		return err
	}
	for _, userID := range owners {
		if err := s.maintainOwner(ctx, userID, c); err != nil {
			slog.Warn("sites: maintenance failed for user", "user_id", userID, "error", err)
		}
	}
	return nil
}

func (s *SiteHostingService) maintainOwner(ctx context.Context, userID int64, c SiteHostingConfig) error {
	sites, err := s.repo.ListSitesByUser(ctx, userID)
	if err != nil || len(sites) == 0 {
		return err
	}
	subscribed, err := s.subscribed(ctx, userID)
	if err != nil {
		return err
	}
	now := s.now()
	grace := time.Duration(c.GraceDays) * 24 * time.Hour
	retention := time.Duration(c.RetentionDays) * 24 * time.Hour
	if !subscribed {
		lapsedAt := sites[0].LapsedAt
		if lapsedAt == nil {
			if err := s.repo.SetLapsedAt(ctx, userID, &now); err != nil {
				return err
			}
			lapsedAt = &now
		}
		for i := range sites {
			site := &sites[i]
			switch elapsed := now.Sub(*lapsedAt); {
			case elapsed > grace+retention:
				slog.Info("sites: deleting site of a lapsed subscription", "site", site.Name, "user_id", userID)
				if err := s.remove(ctx, site); err != nil {
					return err
				}
			case elapsed > grace && (site.Status == SiteStatusActive || site.Status == SiteStatusUnpaid || site.Status == SiteStatusPending):
				if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusLapsed, fmt.Sprintf("订阅已到期超过 %d 天，网站已暂停；续订后自动恢复", c.GraceDays)); err != nil {
					return err
				}
				s.forget(site.Name)
			}
		}
		return nil
	}
	if sites[0].LapsedAt != nil {
		if err := s.repo.SetLapsedAt(ctx, userID, nil); err != nil {
			return err
		}
	}
	freeUsed := 0
	for _, site := range sites {
		if !site.Paid {
			freeUsed++
		}
	}
	for i := range sites {
		site := &sites[i]
		if site.Status == SiteStatusLapsed {
			if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusActive, ""); err != nil {
				return err
			}
			site.Status = SiteStatusActive
			s.forget(site.Name)
		}
		if !site.Paid || site.PaidUntil == nil || site.PaidUntil.After(now) || site.Status == SiteStatusDisabled {
			continue
		}
		// A free slot opened up (a free site was deleted, or the admin raised the allowance).
		if freeUsed < c.FreePerUser || c.ExtraPrice <= 0 {
			if err := s.repo.SetSiteFree(ctx, site.ID); err != nil {
				return err
			}
			freeUsed++
			if site.Status == SiteStatusUnpaid {
				if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusActive, ""); err != nil {
					return err
				}
			}
			s.forget(site.Name)
			continue
		}
		if err := s.renew(ctx, site, c); err == nil {
			continue
		}
		if now.Sub(*site.PaidUntil) > retention {
			slog.Info("sites: deleting unpaid site", "site", site.Name, "user_id", userID)
			if err := s.remove(ctx, site); err != nil {
				return err
			}
			continue
		}
		if site.Status == SiteStatusActive {
			if err := s.repo.SetSiteStatus(ctx, site.ID, SiteStatusUnpaid, fmt.Sprintf("余额不足，续费 ¥%s / 30 天失败，网站已暂停；充值后点「续费」或等待自动续费", strconv.FormatFloat(c.ExtraPrice, 'f', -1, 64))); err != nil {
				return err
			}
			s.forget(site.Name)
		}
	}
	return nil
}
