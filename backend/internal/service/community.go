package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Creative community of HiveGPT 无限画布. The pages live on the canvas site, which calls these
// services through the canvas session (/api/v1/canvas/community/*). A user's public profile is
// created on their first publish; the account's email is never shown.

const (
	WorkVisibilityPublic   = "public"
	WorkVisibilityUnlisted = "unlisted"
	WorkVisibilityPrivate  = "private"

	WorkStatusApproved = "approved"
	WorkStatusPending  = "pending"
	WorkStatusRejected = "rejected"
	WorkStatusHidden   = "hidden"

	ProfileStatusActive = "active"
	ProfileStatusBanned = "banned"

	communityMaxImagesPerWork = 9
	communityMaxTags          = 6
	communityPageSize         = 24
	communityMaxPageSize      = 60
	communityWorksPerDay      = 30
	communityWorksPerDayNew   = 5
	communityNewAccountAge    = 24 * time.Hour
	communityMaxCollections   = 50
	communityMaxWorksPerUser  = 2000
	communityReportsPerIPHour = 10
)

var (
	ErrCommunityProfileRequired = infraerrors.BadRequest("COMMUNITY_PROFILE_REQUIRED", "请先设置你的用户名（Set up your profile first）")
	ErrCommunityHandleInvalid   = infraerrors.BadRequest("COMMUNITY_HANDLE_INVALID", "用户名需要 3–20 个字符：小写字母、数字或下划线，以字母开头（Use 3–20 lowercase letters, digits or underscores, starting with a letter）")
	ErrCommunityHandleReserved  = infraerrors.BadRequest("COMMUNITY_HANDLE_RESERVED", "这个用户名不能使用，请换一个（This name is reserved）")
	ErrCommunityHandleTaken     = infraerrors.Conflict("COMMUNITY_HANDLE_TAKEN", "这个用户名已经被使用，请换一个（This name is taken）")
	ErrCommunityTextInvalid     = infraerrors.BadRequest("COMMUNITY_TEXT_INVALID", "昵称或简介含有不允许的内容，请修改后再保存（Contains disallowed words）")
	ErrCommunityBanned          = infraerrors.Forbidden("COMMUNITY_BANNED", "你的账号已被限制发布作品，如有疑问请联系客服（Publishing is disabled for this account）")
	ErrCommunityWorkNotFound    = infraerrors.NotFound("COMMUNITY_WORK_NOT_FOUND", "作品不存在或已删除（Work not found）")
	ErrCommunityUserNotFound    = infraerrors.NotFound("COMMUNITY_USER_NOT_FOUND", "用户不存在（User not found）")
	ErrCommunityNoImages        = infraerrors.BadRequest("COMMUNITY_NO_IMAGES", "请至少选择 1 张、最多 9 张图片（Pick 1–9 images）")
	ErrCommunityTooMany         = infraerrors.TooManyRequests("COMMUNITY_TOO_MANY", "今天发布的作品已经很多了，明天再来吧（Daily publishing limit reached）")
	ErrCommunityWorksFull       = infraerrors.BadRequest("COMMUNITY_WORKS_FULL", "作品数量已达上限，请先删除一些旧作品（Too many works）")
	ErrCommunitySelfFollow      = infraerrors.BadRequest("COMMUNITY_SELF_FOLLOW", "不能关注自己（You can't follow yourself）")
	ErrCommunityCollectionLimit = infraerrors.BadRequest("COMMUNITY_COLLECTION_LIMIT", "作品集数量已达上限（Too many collections）")
	ErrCommunityCollectionNF    = infraerrors.NotFound("COMMUNITY_COLLECTION_NOT_FOUND", "作品集不存在（Collection not found）")
	ErrCommunityReportInvalid   = infraerrors.BadRequest("COMMUNITY_REPORT_INVALID", "请选择举报原因（Pick a reason）")
	ErrCommunityReportTooMany   = infraerrors.TooManyRequests("COMMUNITY_REPORT_TOO_MANY", "举报太频繁了，请稍后再试（Too many reports）")
)

// CommunityAuthor is the public face of a user on cards and pages.
type CommunityAuthor struct {
	UserID       int64  `json:"-"`
	Handle       string `json:"handle"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url"`
	FollowedByMe bool   `json:"followed_by_me"`
}

type CommunityProfile struct {
	CommunityAuthor
	AvatarFile     string    `json:"-"`
	Bio            string    `json:"bio"`
	Status         string    `json:"-"`
	WorksCount     int       `json:"works_count"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	LikesReceived  int       `json:"likes_received"`
	CreatedAt      time.Time `json:"created_at"`
	IsMe           bool      `json:"is_me"`
}

type WorkMedia struct {
	Position  int    `json:"position"`
	File      string `json:"-"`
	ThumbFile string `json:"-"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumb_url"`
	MimeType  string `json:"mime_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
}

type Work struct {
	ID            int64           `json:"id"`
	UserID        int64           `json:"-"`
	Author        CommunityAuthor `json:"author"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	Prompt        string          `json:"prompt"`
	ShowPrompt    bool            `json:"show_prompt"`
	Model         string          `json:"model"`
	Params        json.RawMessage `json:"params"`
	Source        string          `json:"source"`
	Tags          []string        `json:"tags"`
	Visibility    string          `json:"visibility"`
	Status        string          `json:"status"`
	ReviewReason  string          `json:"review_reason,omitempty"`
	ReviewFlags   []string        `json:"review_flags,omitempty"`
	Featured      bool            `json:"featured"`
	CoverFile     string          `json:"-"`
	CoverThumb    string          `json:"-"`
	CoverURL      string          `json:"cover_url"`
	CoverThumbURL string          `json:"cover_thumb_url"`
	CoverWidth    int             `json:"cover_width"`
	CoverHeight   int             `json:"cover_height"`
	ImageCount    int             `json:"image_count"`
	LikeCount     int             `json:"like_count"`
	FavoriteCount int             `json:"favorite_count"`
	RemixCount    int             `json:"remix_count"`
	ViewCount     int             `json:"view_count"`
	ReportCount   int             `json:"report_count,omitempty"`
	// OwnerID is shown to admins only (to restrict the author).
	OwnerID       int64         `json:"owner_id,omitempty"`
	LikedByMe     bool          `json:"liked_by_me"`
	FavoritedByMe bool          `json:"favorited_by_me"`
	IsMine        bool          `json:"is_mine"`
	Media         []WorkMedia   `json:"media,omitempty"`
	Contests      []WorkContest `json:"contests,omitempty"` // work page only
	Kind          string        `json:"kind"`               // image | site
	// CommentCount: approved comments and replies; the author may close comments.
	CommentCount   int       `json:"comment_count"`
	CommentsClosed bool      `json:"comments_closed"`
	Site           *WorkSite `json:"site,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// WorkQuery selects works for a feed, a profile, a collection, favorites or the admin queue.
type WorkQuery struct {
	// latest | recommended | following | user | favorites | collection | admin
	Feed         string
	ViewerID     int64
	UserID       int64
	CollectionID int64
	Tag          string
	// Owner views include unlisted / private / pending works.
	IncludeAll bool
	// Admin filter: pending | reported | approved | hidden | rejected | "" (all)
	Status string
	// Kind: image | site | "" (all)
	Kind string
	// At pins the recommended / latest feeds to one moment while paging: works published later are
	// left out and scores age from it, so later pages neither repeat nor skip works.
	At     time.Time
	Limit  int
	Offset int
}

// feedSnapshotMaxAge bounds how old a client-sent feed time may be.
const feedSnapshotMaxAge = 6 * time.Hour

// FeedSnapshot is the feed time for a page: the client's (unix seconds) when recent, else now.
func FeedSnapshot(unix int64, now time.Time) time.Time {
	if unix <= 0 {
		return now
	}
	return clampFeedTime(time.Unix(unix, 0), now)
}

// FeedSnapshotUnix is the "at" a client sends back for later pages: at rounded up to the second,
// so works published in its last fraction of a second stay in.
func FeedSnapshotUnix(at time.Time) int64 {
	return at.Add(time.Second - time.Nanosecond).Unix()
}

func clampFeedTime(at, now time.Time) time.Time {
	if at.IsZero() || at.After(now) || now.Sub(at) > feedSnapshotMaxAge {
		return now
	}
	return at
}

type CommunityRepository interface {
	GetProfileByUser(ctx context.Context, userID int64) (*CommunityProfile, error)
	GetProfileByHandle(ctx context.Context, handle string) (*CommunityProfile, error)
	CreateProfile(ctx context.Context, p *CommunityProfile) error
	UpdateProfile(ctx context.Context, p *CommunityProfile) error
	SetProfileStatus(ctx context.Context, userID int64, status string) error
	ListRestrictedProfiles(ctx context.Context, limit int) ([]RestrictedAuthor, error)
	UserCreatedAt(ctx context.Context, userID int64) (time.Time, error)
	AccountAvatarURL(ctx context.Context, userID int64) (string, error)

	CreateWork(ctx context.Context, w *Work, media []WorkMedia) error
	GetWork(ctx context.Context, id int64) (*Work, error)
	UpdateWork(ctx context.Context, w *Work) error
	DeleteWork(ctx context.Context, id int64) error
	ListWorks(ctx context.Context, q WorkQuery) ([]Work, error)
	// RelatedWorks lists other authors' public works like w (shared tags, same model / kind), popular first.
	RelatedWorks(ctx context.Context, w *Work, limit int) ([]Work, error)
	CountWorks(ctx context.Context, userID int64, since time.Time) (int, error)
	SetWorkStatus(ctx context.Context, id int64, status, reason string) error
	SetWorkFeatured(ctx context.Context, id int64, featured bool) error
	AddWorkView(ctx context.Context, id int64) error
	AddWorkRemix(ctx context.Context, id int64) error
	// FillViewerState sets LikedByMe / FavoritedByMe / Author.FollowedByMe / IsMine for the viewer.
	FillViewerState(ctx context.Context, viewerID int64, works []Work) error
	RecountProfile(ctx context.Context, userID int64) error

	SetLike(ctx context.Context, workID, userID int64, on bool) (bool, error)
	SetFavorite(ctx context.Context, workID, userID int64, on bool) (bool, error)
	SetFollow(ctx context.Context, followerID, followeeID int64, on bool) (bool, error)
	IsFollowing(ctx context.Context, followerID, followeeID int64) (bool, error)
	ListFollows(ctx context.Context, userID int64, followers bool, limit, offset int) ([]CommunityProfile, error)

	CreateCollection(ctx context.Context, c *Collection) error
	UpdateCollection(ctx context.Context, c *Collection) error
	DeleteCollection(ctx context.Context, id int64) error
	GetCollection(ctx context.Context, id int64) (*Collection, error)
	ListCollections(ctx context.Context, userID int64, includePrivate bool) ([]Collection, error)
	CountCollections(ctx context.Context, userID int64) (int, error)
	SetCollectionItem(ctx context.Context, collectionID, workID int64, on bool) error
	WorkCollections(ctx context.Context, workID, ownerID int64) ([]int64, error)
	// Comments (viewerID also sees their own pending ones).
	ListComments(ctx context.Context, workID, viewerID int64, limit, offset int) ([]WorkComment, error)
	FirstReplies(ctx context.Context, parentIDs []int64, viewerID int64, n int) (map[int64][]WorkComment, error)
	ListReplies(ctx context.Context, parentID, viewerID int64, limit, offset int) ([]WorkComment, error)
	GetComment(ctx context.Context, id int64) (*WorkComment, error)
	CreateComment(ctx context.Context, c *WorkComment) error
	// SetCommentStatus also recounts the work's comments and the parent's replies.
	SetCommentStatus(ctx context.Context, id int64, status string) error
	CountComments(ctx context.Context, userID int64, since time.Time) (int, error)
	HasRecentComment(ctx context.Context, userID, workID int64, body string, since time.Time) (bool, error)
	AdminListComments(ctx context.Context, status string, limit, offset int) ([]WorkComment, error)
	// Creator stats (days are China Standard Time dates, inclusive).
	CreatorSeries(ctx context.Context, userID int64, from, to time.Time) ([]CreatorDay, error)
	CreatorTotals(ctx context.Context, userID int64) (CreatorTotals, error)
	CreatorTopWorks(ctx context.Context, userID int64, since time.Time, limit int) ([]CreatorWork, error)
	CreatorTrackedSince(ctx context.Context) (string, error)
	// SiteWorkIDs maps the user's site ids to the works presenting them.
	SiteWorkIDs(ctx context.Context, userID int64) (map[int64]int64, error)
	// WorkContests lists the live contest entries made from a work (pending ones too when includePending).
	WorkContests(ctx context.Context, workID int64, includePending bool) ([]WorkContest, error)

	AddNotification(ctx context.Context, n *CommunityNotification) error
	ListNotifications(ctx context.Context, userID int64, limit int) ([]CommunityNotification, error)
	MarkNotificationsRead(ctx context.Context, userID int64) error
	UnreadNotifications(ctx context.Context, userID int64) (int, error)

	CreateWorkReport(ctx context.Context, r *WorkReport) error
	CountWorkReportsSince(ctx context.Context, ip string, since time.Time) (int, error)
	ListWorkReports(ctx context.Context, status string, limit, offset int) ([]WorkReport, error)
	SetWorkReportStatus(ctx context.Context, id int64, status string) error
}

type CommunityService struct {
	repo       CommunityRepository
	media      *CommunityMediaStore
	settings   imageToolSettings
	now        func() time.Time
	shareCache shareMetaCache
	contests   *ContestService
	sites      communitySiteReader
	siteDomain string
}

func NewCommunityService(repo CommunityRepository, media *CommunityMediaStore, settings imageToolSettings) *CommunityService {
	return &CommunityService{repo: repo, media: media, settings: settings, now: time.Now}
}

func (s *CommunityService) Media() *CommunityMediaStore { return s.media }

// Handles ---------------------------------------------------------------------------------------

var communityHandleRe = regexp.MustCompile(`^[a-z][a-z0-9_]{2,19}$`)

var communityReservedHandles = map[string]bool{
	"admin": true, "root": true, "system": true, "support": true, "help": true, "kefu": true, "official": true,
	"explore": true, "discover": true, "me": true, "settings": true, "login": true, "signup": true, "register": true,
	"api": true, "canvas": true, "tools": true, "assets": true, "prompts": true, "image": true, "video": true,
	"notifications": true, "u": true, "w": true, "c": true, "null": true, "undefined": true, "anonymous": true,
}

func validateCommunityHandle(handle string) error {
	if !communityHandleRe.MatchString(handle) || strings.Contains(handle, "__") {
		return ErrCommunityHandleInvalid
	}
	if communityReservedHandles[handle] {
		return ErrCommunityHandleReserved
	}
	for _, word := range siteBlockedNameWords {
		if strings.Contains(handle, strings.ReplaceAll(word, "-", "")) {
			return ErrCommunityHandleReserved
		}
	}
	return nil
}

// Text checks -----------------------------------------------------------------------------------

var communityExtraRiskWords = map[string][]string{
	"adult":    {"裸体", "露点", "nsfw", "nude", "naked"},
	"violence": {"血腥", "虐杀", "斩首", "gore"},
}

// communityTextFlags lists risky words found in user text (empty: nothing found).
func communityTextFlags(texts ...string) []string {
	lower := strings.ToLower(strings.Join(texts, "\n"))
	var flags []string
	check := func(lists map[string][]string) {
		for kind, words := range lists {
			for _, w := range words {
				if strings.Contains(lower, strings.ToLower(w)) {
					flags = append(flags, kind+":"+w)
					break
				}
			}
		}
	}
	check(siteRiskWords)
	check(communityExtraRiskWords)
	return flags
}

func cleanText(text string, max int) string {
	text = strings.TrimSpace(strings.ToValidUTF8(text, ""))
	if utf8.RuneCountInString(text) > max {
		text = truncateRunes(text, max)
	}
	return text
}

// Profiles --------------------------------------------------------------------------------------

func (s *CommunityService) decorateProfile(ctx context.Context, p *CommunityProfile, viewerID int64) {
	if p == nil {
		return
	}
	if p.AvatarFile != "" {
		p.AvatarURL = CommunityMediaURL(p.AvatarFile)
	} else if url, err := s.repo.AccountAvatarURL(ctx, p.UserID); err == nil && strings.HasPrefix(url, "https://") {
		// The account's own avatar (http(s) only: inline data URLs are not sent to every visitor).
		p.AvatarURL = url
	}
	p.IsMe = viewerID != 0 && viewerID == p.UserID
	if viewerID != 0 && !p.IsMe {
		p.FollowedByMe, _ = s.repo.IsFollowing(ctx, viewerID, p.UserID)
	}
}

// MyProfile returns the viewer's profile, or nil when they have not created one.
func (s *CommunityService) MyProfile(ctx context.Context, userID int64) (*CommunityProfile, error) {
	p, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil || p == nil {
		return nil, err
	}
	s.decorateProfile(ctx, p, userID)
	return p, nil
}

// ProfileInput updates (or, the first time, creates) the viewer's profile. Avatar: new image bytes.
type ProfileInput struct {
	Handle      string
	DisplayName string
	Bio         string
	Avatar      []byte
	ClearAvatar bool
}

func (s *CommunityService) SaveProfile(ctx context.Context, userID int64, in ProfileInput) (*CommunityProfile, error) {
	handle := strings.ToLower(strings.TrimSpace(in.Handle))
	if err := validateCommunityHandle(handle); err != nil {
		return nil, err
	}
	display := cleanText(in.DisplayName, 20)
	bio := cleanText(in.Bio, 200)
	if len(communityTextFlags(handle, display, bio)) > 0 {
		return nil, ErrCommunityTextInvalid
	}
	existing, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if other, err := s.repo.GetProfileByHandle(ctx, handle); err != nil {
		return nil, err
	} else if other != nil && other.UserID != userID {
		return nil, ErrCommunityHandleTaken
	}
	p := existing
	if p == nil {
		p = &CommunityProfile{Status: ProfileStatusActive}
		p.UserID = userID
	}
	p.Handle, p.DisplayName, p.Bio = handle, display, bio
	oldAvatar := p.AvatarFile
	if len(in.Avatar) > 0 {
		name, err := s.media.SaveAvatar(in.Avatar)
		if err != nil {
			return nil, err
		}
		p.AvatarFile = name
	} else if in.ClearAvatar {
		p.AvatarFile = ""
	}
	if existing == nil {
		err = s.repo.CreateProfile(ctx, p)
	} else {
		err = s.repo.UpdateProfile(ctx, p)
	}
	if err != nil {
		if p.AvatarFile != oldAvatar {
			s.media.Remove(p.AvatarFile)
		}
		return nil, err
	}
	if oldAvatar != "" && oldAvatar != p.AvatarFile {
		s.media.Remove(oldAvatar)
	}
	return s.MyProfile(ctx, userID)
}

// Profile is a public profile page (nil error and nil profile never: not found is an error).
func (s *CommunityService) Profile(ctx context.Context, handle string, viewerID int64) (*CommunityProfile, error) {
	p, err := s.repo.GetProfileByHandle(ctx, strings.ToLower(strings.TrimSpace(handle)))
	if err != nil {
		return nil, err
	}
	if p == nil || (p.Status == ProfileStatusBanned && p.UserID != viewerID) {
		return nil, ErrCommunityUserNotFound
	}
	s.decorateProfile(ctx, p, viewerID)
	return p, nil
}

// Works -----------------------------------------------------------------------------------------

// PublishInput is a new work: 1–9 images and their metadata.
type PublishInput struct {
	Images       [][]byte
	Title        string
	Description  string
	Prompt       string
	ShowPrompt   bool
	Model        string
	Params       json.RawMessage
	Source       string
	Tags         []string
	Visibility   string
	CollectionID int64
	RemixOf      int64
	// SiteID makes a web-page work presenting one of the author's live sites (Images are its screenshots).
	SiteID int64
}

func normalizeVisibility(v string) string {
	switch v {
	case WorkVisibilityUnlisted, WorkVisibilityPrivate:
		return v
	default:
		return WorkVisibilityPublic
	}
}

func normalizeSource(v string) string {
	switch v {
	case "image_workbench", "tools", "contest", "canvas", "site":
		return v
	default:
		return "canvas"
	}
}

func normalizeTags(tags []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = cleanText(strings.TrimPrefix(strings.TrimSpace(tag), "#"), 12)
		if tag == "" || seen[strings.ToLower(tag)] {
			continue
		}
		seen[strings.ToLower(tag)] = true
		out = append(out, tag)
		if len(out) == communityMaxTags {
			break
		}
	}
	return out
}

func normalizeParams(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || len(raw) > 4000 {
		return json.RawMessage(`{}`)
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

// reviewAll: the admin wants every new work checked by hand.
func (s *CommunityService) reviewAll(ctx context.Context) bool {
	if s.settings == nil {
		return false
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingCommunityReviewAll})
	return err == nil && values[settingCommunityReviewAll] == "true"
}

const settingCommunityReviewAll = "community_review_all"

func (s *CommunityService) Publish(ctx context.Context, userID int64, in PublishInput) (*Work, error) {
	profile, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrCommunityProfileRequired
	}
	if profile.Status == ProfileStatusBanned {
		return nil, ErrCommunityBanned
	}
	if len(in.Images) == 0 || len(in.Images) > communityMaxImagesPerWork {
		return nil, ErrCommunityNoImages
	}
	var site *WorkSite
	if in.SiteID > 0 {
		if site, err = s.workSiteFor(ctx, userID, in.SiteID); err != nil {
			return nil, err
		}
	}
	total, err := s.repo.CountWorks(ctx, userID, time.Time{})
	if err != nil {
		return nil, err
	}
	if total >= communityMaxWorksPerUser {
		return nil, ErrCommunityWorksFull
	}
	limit := communityWorksPerDay
	if created, err := s.repo.UserCreatedAt(ctx, userID); err == nil && s.now().Sub(created) < communityNewAccountAge {
		limit = communityWorksPerDayNew
	}
	today, err := s.repo.CountWorks(ctx, userID, s.now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	if today >= limit {
		return nil, ErrCommunityTooMany
	}

	w := &Work{
		UserID:      userID,
		Title:       cleanText(in.Title, 80),
		Description: cleanText(in.Description, 1000),
		Prompt:      cleanText(in.Prompt, 8000),
		ShowPrompt:  in.ShowPrompt,
		Model:       cleanText(in.Model, 100),
		Params:      normalizeParams(in.Params),
		Source:      normalizeSource(in.Source),
		Tags:        normalizeTags(in.Tags),
		Visibility:  normalizeVisibility(in.Visibility),
		Status:      WorkStatusApproved,
		Kind:        WorkKindImage,
	}
	if site != nil {
		w.Kind, w.Site = WorkKindSite, site
	}
	if flags := communityTextFlags(w.Title, w.Description, w.Prompt, strings.Join(w.Tags, " ")); len(flags) > 0 {
		w.Status, w.ReviewFlags, w.ReviewReason = WorkStatusPending, flags, "内容命中敏感词，人工审核后公开"
	} else if s.reviewAll(ctx) {
		w.Status, w.ReviewReason = WorkStatusPending, "新作品需要人工审核后公开"
	}
	if w.ReviewFlags == nil {
		w.ReviewFlags = []string{}
	}

	media := make([]WorkMedia, 0, len(in.Images))
	cleanup := func() {
		for _, m := range media {
			s.media.Remove(m.File, m.ThumbFile)
		}
	}
	for i, data := range in.Images {
		stored, err := s.media.SaveImage(data)
		if err != nil {
			cleanup()
			return nil, err
		}
		media = append(media, WorkMedia{Position: i, File: stored.File, ThumbFile: stored.ThumbFile, MimeType: stored.MimeType, Width: stored.Width, Height: stored.Height, SizeBytes: stored.Size})
	}
	w.CoverWidth, w.CoverHeight = media[0].Width, media[0].Height
	if err := s.repo.CreateWork(ctx, w, media); err != nil {
		cleanup()
		return nil, err
	}
	if in.CollectionID > 0 {
		if c, err := s.repo.GetCollection(ctx, in.CollectionID); err == nil && c != nil && c.UserID == userID {
			_ = s.repo.SetCollectionItem(ctx, c.ID, w.ID, true)
		}
	}
	if in.RemixOf > 0 {
		_ = s.Remix(ctx, userID, in.RemixOf)
	}
	_ = s.repo.RecountProfile(ctx, userID)
	return s.Work(ctx, w.ID, userID)
}

// visibleTo: who may open a work page.
func visibleTo(w *Work, viewerID int64) bool {
	if w.UserID == viewerID && viewerID != 0 {
		return true
	}
	return w.Status == WorkStatusApproved && w.Visibility != WorkVisibilityPrivate && (w.Kind != WorkKindSite || w.Site.Live())
}

func (s *CommunityService) decorateWorks(ctx context.Context, works []Work, viewerID int64) {
	for i := range works {
		w := &works[i]
		w.CoverURL, w.CoverThumbURL = CommunityMediaURL(w.CoverFile), CommunityMediaURL(w.CoverThumb)
		s.decorateSite(w, viewerID)
		if w.Author.AvatarURL != "" && !strings.HasPrefix(w.Author.AvatarURL, "https://") && !strings.HasPrefix(w.Author.AvatarURL, CommunityMediaPublicPrefix) {
			w.Author.AvatarURL = CommunityMediaURL(w.Author.AvatarURL)
		}
		for j := range w.Media {
			w.Media[j].URL, w.Media[j].ThumbURL = CommunityMediaURL(w.Media[j].File), CommunityMediaURL(w.Media[j].ThumbFile)
		}
		if !w.ShowPrompt && w.UserID != viewerID {
			w.Prompt = ""
		}
		if w.Tags == nil {
			w.Tags = []string{}
		}
		if len(w.Params) == 0 {
			w.Params = json.RawMessage(`{}`)
		}
		if w.UserID != viewerID {
			w.ReviewFlags = nil
		}
	}
	if viewerID != 0 && len(works) > 0 {
		_ = s.repo.FillViewerState(ctx, viewerID, works)
	}
}

// Work is a work page; counts a view for visitors other than the author.
func (s *CommunityService) Work(ctx context.Context, id, viewerID int64) (*Work, error) {
	w, err := s.repo.GetWork(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil || !visibleTo(w, viewerID) {
		return nil, ErrCommunityWorkNotFound
	}
	if w.UserID != viewerID {
		_ = s.repo.AddWorkView(ctx, id)
	}
	list := []Work{*w}
	s.decorateWorks(ctx, list, viewerID)
	if s.contests != nil {
		// The author also sees entries waiting for review.
		if contests, err := s.repo.WorkContests(ctx, id, w.UserID == viewerID); err == nil {
			list[0].Contests = contests
		}
	}
	return &list[0], nil
}

// Works lists a feed or a page of works for the viewer.
func (s *CommunityService) Works(ctx context.Context, q WorkQuery) ([]Work, error) {
	if q.Limit <= 0 || q.Limit > communityMaxPageSize {
		q.Limit = communityPageSize
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	switch q.Feed {
	case "latest", "recommended", "following", "user", "favorites", "collection":
	default:
		q.Feed = "recommended"
	}
	if q.Kind != WorkKindImage && q.Kind != WorkKindSite {
		q.Kind = ""
	}
	if (q.Feed == "following" || q.Feed == "favorites") && q.ViewerID == 0 {
		return []Work{}, nil
	}
	if q.Feed == "user" {
		q.IncludeAll = q.ViewerID != 0 && q.ViewerID == q.UserID
	}
	if q.Feed == "collection" {
		c, err := s.repo.GetCollection(ctx, q.CollectionID)
		if err != nil {
			return nil, err
		}
		if c == nil || (c.Visibility != "public" && c.UserID != q.ViewerID) {
			return nil, ErrCommunityCollectionNF
		}
		q.IncludeAll = c.UserID == q.ViewerID
	}
	q.Tag = cleanText(q.Tag, 12)
	q.At = clampFeedTime(q.At, time.Now())
	works, err := s.repo.ListWorks(ctx, q)
	if err != nil {
		return nil, err
	}
	if works == nil {
		works = []Work{}
	}
	s.decorateWorks(ctx, works, q.ViewerID)
	return works, nil
}

// RelatedWorks lists works like the given one for its work page ("相似作品").
func (s *CommunityService) RelatedWorks(ctx context.Context, id, viewerID int64, limit int) ([]Work, error) {
	w, err := s.repo.GetWork(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil || !visibleTo(w, viewerID) {
		return nil, ErrCommunityWorkNotFound
	}
	if limit <= 0 || limit > 24 {
		limit = 12
	}
	works, err := s.repo.RelatedWorks(ctx, w, limit)
	if err != nil {
		return nil, err
	}
	if works == nil {
		works = []Work{}
	}
	s.decorateWorks(ctx, works, viewerID)
	return works, nil
}

// UpdateWorkInput edits a work's text and visibility (images stay).
type UpdateWorkInput struct {
	Title       string
	Description string
	ShowPrompt  bool
	Tags        []string
	Visibility  string
	// CommentsClosed, when set, opens or closes the work's comments.
	CommentsClosed *bool
}

func (s *CommunityService) owned(ctx context.Context, userID, workID int64) (*Work, error) {
	w, err := s.repo.GetWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	if w == nil || w.UserID != userID {
		return nil, ErrCommunityWorkNotFound
	}
	return w, nil
}

func (s *CommunityService) UpdateWork(ctx context.Context, userID, workID int64, in UpdateWorkInput) (*Work, error) {
	w, err := s.owned(ctx, userID, workID)
	if err != nil {
		return nil, err
	}
	w.Title, w.Description, w.ShowPrompt = cleanText(in.Title, 80), cleanText(in.Description, 1000), in.ShowPrompt
	w.Tags, w.Visibility = normalizeTags(in.Tags), normalizeVisibility(in.Visibility)
	if in.CommentsClosed != nil {
		w.CommentsClosed = *in.CommentsClosed
	}
	if flags := communityTextFlags(w.Title, w.Description, strings.Join(w.Tags, " ")); len(flags) > 0 && w.Status == WorkStatusApproved {
		w.Status, w.ReviewFlags, w.ReviewReason = WorkStatusPending, flags, "修改后的内容命中敏感词，人工审核后公开"
	}
	if err := s.repo.UpdateWork(ctx, w); err != nil {
		return nil, err
	}
	_ = s.repo.RecountProfile(ctx, userID)
	return s.Work(ctx, workID, userID)
}

func (s *CommunityService) DeleteWork(ctx context.Context, userID, workID int64) error {
	w, err := s.owned(ctx, userID, workID)
	if err != nil {
		return err
	}
	full, err := s.repo.GetWork(ctx, w.ID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteWork(ctx, w.ID); err != nil {
		return err
	}
	if full != nil {
		for _, m := range full.Media {
			s.media.Remove(m.File, m.ThumbFile)
		}
	}
	return s.repo.RecountProfile(ctx, userID)
}

// Remix counts a "做同款" and tells the author.
func (s *CommunityService) Remix(ctx context.Context, userID, workID int64) error {
	w, err := s.repo.GetWork(ctx, workID)
	if err != nil {
		return err
	}
	if w == nil || !visibleTo(w, userID) {
		return ErrCommunityWorkNotFound
	}
	if err := s.repo.AddWorkRemix(ctx, workID); err != nil {
		return err
	}
	if userID != 0 && userID != w.UserID {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: "remix", ActorID: userID, WorkID: workID})
	}
	return nil
}
