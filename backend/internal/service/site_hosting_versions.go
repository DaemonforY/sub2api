package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// Versions (history, rollback, review decisions), access passwords and signed preview links.

var (
	ErrSiteVersionNotFound   = infraerrors.NotFound("SITE_VERSION_NOT_FOUND", "这个版本不存在或已被清理，只能回到最近保留的几个版本（Version not found）")
	ErrSiteVersionNotServing = infraerrors.BadRequest("SITE_VERSION_NOT_APPROVED", "只能回到审核通过的版本（Version is not approved）")
	ErrSitePasswordInvalid   = infraerrors.BadRequest("SITE_PASSWORD_INVALID", "访问密码需要 4–64 个字符（Password must be 4–64 characters）")
	ErrSiteNothingToReview   = infraerrors.BadRequest("SITE_NOTHING_TO_REVIEW", "这个版本已经审核过了（Already reviewed）")
)

type SiteVersion struct {
	Version      int       `json:"version"`
	SizeBytes    int64     `json:"size_bytes"`
	FileCount    int       `json:"file_count"`
	ReviewStatus string    `json:"review_status"`
	ReviewReason string    `json:"review_reason"`
	ReviewedBy   string    `json:"reviewed_by"`
	CreatedAt    time.Time `json:"created_at"`
	Current      bool      `json:"current"`
	// Available: still on disk (rollback possible when approved).
	Available bool `json:"available"`
}

// SiteReviewItem is one version in the admin review queue.
type SiteReviewItem struct {
	SiteID     int64     `json:"site_id"`
	SiteName   string    `json:"site_name"`
	Title      string    `json:"title"`
	OwnerEmail string    `json:"owner_email"`
	SiteStatus string    `json:"site_status"`
	Version    int       `json:"version"`
	SizeBytes  int64     `json:"size_bytes"`
	FileCount  int       `json:"file_count"`
	Reason     string    `json:"reason"`
	Flags      []string  `json:"flags"`
	Excerpt    string    `json:"excerpt"`
	CreatedAt  time.Time `json:"created_at"`
	PreviewURL string    `json:"preview_url"`
}

func (s *SiteHostingService) Versions(ctx context.Context, userID, siteID int64) ([]SiteVersion, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	return s.versionsOf(ctx, site)
}

func (s *SiteHostingService) versionsOf(ctx context.Context, site *Site) ([]SiteVersion, error) {
	versions, err := s.repo.ListVersions(ctx, site.ID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		v := &versions[i]
		v.Current = v.Version == site.Version
		_, statErr := os.Stat(s.versionDir(site.ID, v.Version))
		v.Available = statErr == nil
	}
	if versions == nil {
		versions = []SiteVersion{}
	}
	return versions, nil
}

// Rollback serves an earlier approved version again.
func (s *SiteHostingService) Rollback(ctx context.Context, userID, siteID int64, version int) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if site.Status == SiteStatusDisabled {
		return nil, ErrSiteDisabledByAdmin
	}
	versions, err := s.versionsOf(ctx, site)
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.Version != version {
			continue
		}
		if !v.Available {
			return nil, ErrSiteVersionNotFound
		}
		if v.ReviewStatus != SiteReviewApproved {
			return nil, ErrSiteVersionNotServing
		}
		if err := s.repo.ServeSiteVersion(ctx, site.ID, version, false); err != nil {
			return nil, err
		}
		s.forget(site.Name)
		return s.mineByID(ctx, userID, siteID)
	}
	return nil, ErrSiteVersionNotFound
}

// SetPassword sets (or with "" removes) the site's access password.
func (s *SiteHostingService) SetPassword(ctx context.Context, userID, siteID int64, password string) (*Site, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	hash := ""
	if password != "" {
		if n := len([]rune(password)); n < 4 || n > 64 {
			return nil, ErrSitePasswordInvalid
		}
		b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hash = string(b)
	}
	if err := s.repo.SetSitePassword(ctx, site.ID, hash); err != nil {
		return nil, err
	}
	s.forget(site.Name)
	return s.mineByID(ctx, userID, siteID)
}

// Review queue ----------------------------------------------------------------------------------

func (s *SiteHostingService) AdminReviews(ctx context.Context, page, pageSize int) ([]SiteReviewItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := s.repo.ListPendingReviews(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i].PreviewURL = s.PreviewURL(items[i].SiteName, items[i].SiteID, items[i].Version)
		if items[i].Flags == nil {
			items[i].Flags = []string{}
		}
	}
	if items == nil {
		items = []SiteReviewItem{}
	}
	return items, total, nil
}

func (s *SiteHostingService) pendingSite(ctx context.Context, siteID int64, version int) (*Site, error) {
	site, err := s.repo.GetSite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, ErrSiteNotFound
	}
	if site.PendingVersion != version || version == 0 {
		return nil, ErrSiteNothingToReview
	}
	return site, nil
}

// AdminApprove puts a queued version online.
func (s *SiteHostingService) AdminApprove(ctx context.Context, siteID int64, version int) error {
	site, err := s.pendingSite(ctx, siteID, version)
	if err != nil {
		return err
	}
	if err := s.repo.SetVersionReview(ctx, siteID, version, SiteReviewApproved, "", "admin"); err != nil {
		return err
	}
	if err := s.repo.ServeSiteVersion(ctx, siteID, version, true); err != nil {
		return err
	}
	if site.Status == SiteStatusPending {
		if err := s.repo.SetSiteStatus(ctx, siteID, SiteStatusActive, ""); err != nil {
			return err
		}
	} else if site.Status == SiteStatusActive && site.StatusReason != "" {
		_ = s.repo.SetSiteStatus(ctx, siteID, SiteStatusActive, "")
	}
	s.forget(site.Name)
	return nil
}

// AdminReject refuses a queued version (its files are removed). A site that was never online is
// taken down with the reason; otherwise the previous version stays and the owner sees the reason.
func (s *SiteHostingService) AdminReject(ctx context.Context, siteID int64, version int, reason string) error {
	site, err := s.pendingSite(ctx, siteID, version)
	if err != nil {
		return err
	}
	reason = truncateRunes(strings.TrimSpace(reason), 200)
	if reason == "" {
		reason = "内容不符合网站托管规范"
	}
	if err := s.repo.SetVersionReview(ctx, siteID, version, SiteReviewRejected, reason, "admin"); err != nil {
		return err
	}
	if err := s.repo.SetPendingVersion(ctx, siteID, 0); err != nil {
		return err
	}
	_ = os.RemoveAll(s.versionDir(siteID, version))
	if site.Version == 0 {
		err = s.repo.SetSiteStatus(ctx, siteID, SiteStatusDisabled, "审核未通过："+reason)
	} else if site.Status == SiteStatusActive {
		err = s.repo.SetSiteStatus(ctx, siteID, SiteStatusActive, fmt.Sprintf("第 %d 版审核未通过（%s），网站仍显示第 %d 版", version, reason, site.Version))
	}
	s.forget(site.Name)
	return err
}

// Signed cookies and links -----------------------------------------------------------------------

func (s *SiteHostingService) sign(parts ...string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(strings.Join(parts, "|")))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

const sitePreviewTTL = 7 * 24 * time.Hour

// PreviewURL opens a version that is not online yet (review queue, the owner's pending upload).
func (s *SiteHostingService) PreviewURL(name string, siteID int64, version int) string {
	if s.defaults.Domain == "" {
		return ""
	}
	exp := strconv.FormatInt(s.now().Add(sitePreviewTTL).Unix(), 10)
	token := fmt.Sprintf("%d.%s.%s", version, exp, s.sign("preview", strconv.FormatInt(siteID, 10), strconv.Itoa(version), exp))
	return s.siteURL(name) + "/?" + sitePreviewParam + "=" + token
}

const (
	sitePreviewParam  = "__hg_preview"
	sitePreviewCookie = "__hg_preview"
	siteAuthCookie    = "__hg_auth"
	siteUnlockPath    = "/__hivegpt/unlock"
	siteAuthTTL       = 7 * 24 * time.Hour
)

// previewVersion returns the version a valid preview token opens (0 when invalid or expired).
func (s *SiteHostingService) previewVersion(site *Site, token string) int {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return 0
	}
	version, err1 := strconv.Atoi(parts[0])
	exp, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || version <= 0 || s.now().Unix() > exp {
		return 0
	}
	want := s.sign("preview", strconv.FormatInt(site.ID, 10), parts[0], parts[1])
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return 0
	}
	return version
}

// authToken proves a visitor entered the site's password; it changes when the password does.
func (s *SiteHostingService) authToken(site *Site, exp int64) string {
	hashTag := sha256.Sum256([]byte(site.PasswordHash))
	expStr := strconv.FormatInt(exp, 10)
	return expStr + "." + s.sign("auth", strconv.FormatInt(site.ID, 10), hex.EncodeToString(hashTag[:8]), expStr)
}

func (s *SiteHostingService) validAuthToken(site *Site, token string) bool {
	expStr, _, ok := strings.Cut(token, ".")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if !ok || err != nil || s.now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(token), []byte(s.authToken(site, exp)))
}
