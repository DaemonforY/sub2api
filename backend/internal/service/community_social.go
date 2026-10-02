package service

import (
	"context"
	"strings"
	"time"
)

// Likes, favorites, follows, collections, notifications, reports and moderation of the community.

type Collection struct {
	ID          int64            `json:"id"`
	UserID      int64            `json:"-"`
	Author      *CommunityAuthor `json:"author,omitempty"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Visibility  string           `json:"visibility"`
	WorksCount  int              `json:"works_count"`
	CoverFiles  []string         `json:"-"`
	CoverURLs   []string         `json:"cover_urls"`
	IsMine      bool             `json:"is_mine"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type CommunityNotification struct {
	ID           int64            `json:"id"`
	UserID       int64            `json:"-"`
	Kind         string           `json:"kind"`
	ActorID      int64            `json:"-"`
	Actor        *CommunityAuthor `json:"actor,omitempty"`
	WorkID       int64            `json:"work_id,omitempty"`
	WorkTitle    string           `json:"work_title,omitempty"`
	WorkThumb    string           `json:"-"`
	WorkThumbURL string           `json:"work_thumb_url,omitempty"`
	Detail       string           `json:"detail,omitempty"`
	Read         bool             `json:"read"`
	CreatedAt    time.Time        `json:"created_at"`
}

type WorkReport struct {
	ID         int64     `json:"id"`
	WorkID     int64     `json:"work_id"`
	WorkTitle  string    `json:"work_title"`
	ReporterID int64     `json:"-"`
	Reason     string    `json:"reason"`
	Detail     string    `json:"detail"`
	IP         string    `json:"-"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

var WorkReportReasons = map[string]bool{"porn": true, "violence": true, "politics": true, "copyright": true, "fraud": true, "spam": true, "other": true}

// Likes and favorites ---------------------------------------------------------------------------

func (s *CommunityService) visibleWork(ctx context.Context, workID, viewerID int64) (*Work, error) {
	w, err := s.repo.GetWork(ctx, workID)
	if err != nil {
		return nil, err
	}
	if w == nil || !visibleTo(w, viewerID) {
		return nil, ErrCommunityWorkNotFound
	}
	return w, nil
}

// InteractionState is the work's counters after a like / favorite.
type InteractionState struct {
	LikeCount     int  `json:"like_count"`
	FavoriteCount int  `json:"favorite_count"`
	LikedByMe     bool `json:"liked_by_me"`
	FavoritedByMe bool `json:"favorited_by_me"`
}

func (s *CommunityService) state(ctx context.Context, workID, userID int64) (*InteractionState, error) {
	w, err := s.repo.GetWork(ctx, workID)
	if err != nil || w == nil {
		return nil, err
	}
	list := []Work{*w}
	if err := s.repo.FillViewerState(ctx, userID, list); err != nil {
		return nil, err
	}
	return &InteractionState{LikeCount: list[0].LikeCount, FavoriteCount: list[0].FavoriteCount, LikedByMe: list[0].LikedByMe, FavoritedByMe: list[0].FavoritedByMe}, nil
}

func (s *CommunityService) SetLike(ctx context.Context, userID, workID int64, on bool) (*InteractionState, error) {
	w, err := s.visibleWork(ctx, workID, userID)
	if err != nil {
		return nil, err
	}
	changed, err := s.repo.SetLike(ctx, workID, userID, on)
	if err != nil {
		return nil, err
	}
	if changed && on && w.UserID != userID {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: "like", ActorID: userID, WorkID: workID})
	}
	return s.state(ctx, workID, userID)
}

func (s *CommunityService) SetFavorite(ctx context.Context, userID, workID int64, on bool) (*InteractionState, error) {
	w, err := s.visibleWork(ctx, workID, userID)
	if err != nil {
		return nil, err
	}
	changed, err := s.repo.SetFavorite(ctx, workID, userID, on)
	if err != nil {
		return nil, err
	}
	if changed && on && w.UserID != userID {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: "favorite", ActorID: userID, WorkID: workID})
	}
	return s.state(ctx, workID, userID)
}

// Follows ---------------------------------------------------------------------------------------

func (s *CommunityService) SetFollow(ctx context.Context, userID int64, handle string, on bool) (*CommunityProfile, error) {
	target, err := s.Profile(ctx, handle, userID)
	if err != nil {
		return nil, err
	}
	if target.UserID == userID {
		return nil, ErrCommunitySelfFollow
	}
	changed, err := s.repo.SetFollow(ctx, userID, target.UserID, on)
	if err != nil {
		return nil, err
	}
	if changed && on {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: target.UserID, Kind: "follow", ActorID: userID})
	}
	return s.Profile(ctx, handle, userID)
}

// Follows lists followers (or the users followed) of a profile.
func (s *CommunityService) Follows(ctx context.Context, handle string, followers bool, viewerID int64, offset int) ([]CommunityProfile, error) {
	target, err := s.Profile(ctx, handle, viewerID)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.ListFollows(ctx, target.UserID, followers, communityPageSize, max(0, offset))
	if err != nil {
		return nil, err
	}
	for i := range list {
		s.decorateProfile(ctx, &list[i], viewerID)
	}
	if list == nil {
		list = []CommunityProfile{}
	}
	return list, nil
}

// Collections -----------------------------------------------------------------------------------

type CollectionInput struct {
	Title       string
	Description string
	Visibility  string
}

func (s *CommunityService) decorateCollections(list []Collection, viewerID int64) {
	for i := range list {
		c := &list[i]
		c.IsMine = viewerID != 0 && c.UserID == viewerID
		c.CoverURLs = []string{}
		for _, f := range c.CoverFiles {
			c.CoverURLs = append(c.CoverURLs, CommunityMediaURL(f))
		}
		if c.Author != nil && c.Author.AvatarURL != "" && !strings.HasPrefix(c.Author.AvatarURL, "https://") {
			c.Author.AvatarURL = CommunityMediaURL(c.Author.AvatarURL)
		}
	}
}

func collectionVisibility(v string) string {
	if v == "private" {
		return v
	}
	return "public"
}

func (s *CommunityService) CreateCollection(ctx context.Context, userID int64, in CollectionInput) (*Collection, error) {
	if p, err := s.repo.GetProfileByUser(ctx, userID); err != nil {
		return nil, err
	} else if p == nil {
		return nil, ErrCommunityProfileRequired
	}
	n, err := s.repo.CountCollections(ctx, userID)
	if err != nil {
		return nil, err
	}
	if n >= communityMaxCollections {
		return nil, ErrCommunityCollectionLimit
	}
	c := &Collection{UserID: userID, Title: cleanText(in.Title, 60), Description: cleanText(in.Description, 300), Visibility: collectionVisibility(in.Visibility)}
	if c.Title == "" {
		c.Title = "未命名作品集"
	}
	if len(communityTextFlags(c.Title, c.Description)) > 0 {
		return nil, ErrCommunityTextInvalid
	}
	if err := s.repo.CreateCollection(ctx, c); err != nil {
		return nil, err
	}
	return s.Collection(ctx, c.ID, userID)
}

func (s *CommunityService) ownedCollection(ctx context.Context, userID, id int64) (*Collection, error) {
	c, err := s.repo.GetCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || c.UserID != userID {
		return nil, ErrCommunityCollectionNF
	}
	return c, nil
}

func (s *CommunityService) UpdateCollection(ctx context.Context, userID, id int64, in CollectionInput) (*Collection, error) {
	c, err := s.ownedCollection(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	c.Title, c.Description, c.Visibility = cleanText(in.Title, 60), cleanText(in.Description, 300), collectionVisibility(in.Visibility)
	if c.Title == "" {
		c.Title = "未命名作品集"
	}
	if len(communityTextFlags(c.Title, c.Description)) > 0 {
		return nil, ErrCommunityTextInvalid
	}
	if err := s.repo.UpdateCollection(ctx, c); err != nil {
		return nil, err
	}
	return s.Collection(ctx, id, userID)
}

func (s *CommunityService) DeleteCollection(ctx context.Context, userID, id int64) error {
	if _, err := s.ownedCollection(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteCollection(ctx, id)
}

func (s *CommunityService) Collection(ctx context.Context, id, viewerID int64) (*Collection, error) {
	c, err := s.repo.GetCollection(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil || (c.Visibility != "public" && c.UserID != viewerID) {
		return nil, ErrCommunityCollectionNF
	}
	list := []Collection{*c}
	s.decorateCollections(list, viewerID)
	return &list[0], nil
}

func (s *CommunityService) Collections(ctx context.Context, handle string, viewerID int64) ([]Collection, error) {
	p, err := s.Profile(ctx, handle, viewerID)
	if err != nil {
		return nil, err
	}
	list, err := s.repo.ListCollections(ctx, p.UserID, p.UserID == viewerID)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Collection{}
	}
	s.decorateCollections(list, viewerID)
	return list, nil
}

// SetCollectionItem adds (or removes) one of the viewer's own works to one of their collections.
func (s *CommunityService) SetCollectionItem(ctx context.Context, userID, collectionID, workID int64, on bool) error {
	if _, err := s.ownedCollection(ctx, userID, collectionID); err != nil {
		return err
	}
	if _, err := s.owned(ctx, userID, workID); err != nil {
		return err
	}
	return s.repo.SetCollectionItem(ctx, collectionID, workID, on)
}

// WorkCollections lists which of the owner's collections hold a work.
func (s *CommunityService) WorkCollections(ctx context.Context, userID, workID int64) ([]int64, error) {
	if _, err := s.owned(ctx, userID, workID); err != nil {
		return nil, err
	}
	ids, err := s.repo.WorkCollections(ctx, workID, userID)
	if ids == nil {
		ids = []int64{}
	}
	return ids, err
}

// Notifications ---------------------------------------------------------------------------------

func (s *CommunityService) Notifications(ctx context.Context, userID int64) ([]CommunityNotification, error) {
	list, err := s.repo.ListNotifications(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	for i := range list {
		n := &list[i]
		n.WorkThumbURL = CommunityMediaURL(n.WorkThumb)
		if n.Actor != nil && n.Actor.AvatarURL != "" && !strings.HasPrefix(n.Actor.AvatarURL, "https://") {
			n.Actor.AvatarURL = CommunityMediaURL(n.Actor.AvatarURL)
		}
	}
	if list == nil {
		list = []CommunityNotification{}
	}
	return list, nil
}

func (s *CommunityService) MarkNotificationsRead(ctx context.Context, userID int64) error {
	return s.repo.MarkNotificationsRead(ctx, userID)
}

func (s *CommunityService) UnreadNotifications(ctx context.Context, userID int64) (int, error) {
	return s.repo.UnreadNotifications(ctx, userID)
}

// Reports ---------------------------------------------------------------------------------------

func (s *CommunityService) Report(ctx context.Context, viewerID, workID int64, reason, detail, ip string) error {
	if !WorkReportReasons[reason] {
		return ErrCommunityReportInvalid
	}
	if _, err := s.visibleWork(ctx, workID, viewerID); err != nil {
		return err
	}
	n, err := s.repo.CountWorkReportsSince(ctx, ip, s.now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if n >= communityReportsPerIPHour {
		return ErrCommunityReportTooMany
	}
	return s.repo.CreateWorkReport(ctx, &WorkReport{WorkID: workID, ReporterID: viewerID, Reason: reason, Detail: cleanText(detail, 500), IP: truncateRunes(ip, 64), Status: "open"})
}

// Admin -----------------------------------------------------------------------------------------

// AdminWorks lists works for moderation (status: pending | reported | approved | hidden | rejected | "").
func (s *CommunityService) AdminWorks(ctx context.Context, status string, page, pageSize int) ([]Work, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 30
	}
	works, err := s.repo.ListWorks(ctx, WorkQuery{Feed: "admin", Status: status, Limit: pageSize, Offset: (page - 1) * pageSize})
	if err != nil {
		return nil, err
	}
	if works == nil {
		works = []Work{}
	}
	s.decorateWorks(ctx, works, 0)
	for i := range works {
		// Admins see the prompt and flags regardless of the author's choice.
		if full, err := s.repo.GetWork(ctx, works[i].ID); err == nil && full != nil {
			works[i].Prompt, works[i].ReviewFlags, works[i].Media = full.Prompt, full.ReviewFlags, full.Media
			for j := range works[i].Media {
				works[i].Media[j].URL, works[i].Media[j].ThumbURL = CommunityMediaURL(works[i].Media[j].File), CommunityMediaURL(works[i].Media[j].ThumbFile)
			}
		}
		works[i].OwnerID = works[i].UserID
	}
	return works, nil
}

// AdminModerate: approve | reject | hide | feature | unfeature.
func (s *CommunityService) AdminModerate(ctx context.Context, workID int64, action, reason string) error {
	w, err := s.repo.GetWork(ctx, workID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrCommunityWorkNotFound
	}
	reason = cleanText(reason, 200)
	switch action {
	case "approve":
		err = s.repo.SetWorkStatus(ctx, workID, WorkStatusApproved, "")
	case "reject", "hide":
		status, kind := WorkStatusRejected, "rejected"
		if action == "hide" {
			status, kind = WorkStatusHidden, "hidden"
		}
		if reason == "" {
			reason = "内容不符合社区规范"
		}
		if err = s.repo.SetWorkStatus(ctx, workID, status, reason); err == nil {
			_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: kind, WorkID: workID, Detail: reason})
		}
	case "feature", "unfeature":
		if err = s.repo.SetWorkFeatured(ctx, workID, action == "feature"); err == nil && action == "feature" {
			_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: "featured", WorkID: workID})
		}
	default:
		return ErrCommunityReportInvalid
	}
	if err != nil {
		return err
	}
	return s.repo.RecountProfile(ctx, w.UserID)
}

func (s *CommunityService) AdminBan(ctx context.Context, userID int64, banned bool) error {
	status := ProfileStatusActive
	if banned {
		status = ProfileStatusBanned
	}
	return s.repo.SetProfileStatus(ctx, userID, status)
}

func (s *CommunityService) AdminReports(ctx context.Context, status string, page int) ([]WorkReport, error) {
	if page < 1 {
		page = 1
	}
	list, err := s.repo.ListWorkReports(ctx, status, 50, (page-1)*50)
	if list == nil {
		list = []WorkReport{}
	}
	return list, err
}

func (s *CommunityService) AdminSetReport(ctx context.Context, id int64, status string) error {
	if status != "resolved" && status != "dismissed" && status != "open" {
		return ErrCommunityReportInvalid
	}
	return s.repo.SetWorkReportStatus(ctx, id, status)
}

// CommunityAdminSettings: review every new work by hand; the canvas cloud sync quotas (MB).
type CommunityAdminSettings struct {
	ReviewAll bool `json:"review_all"`
	CanvasCloudQuotaSettings
}

func (s *CommunityService) AdminSettings(ctx context.Context) CommunityAdminSettings {
	return CommunityAdminSettings{ReviewAll: s.reviewAll(ctx), CanvasCloudQuotaSettings: readCanvasCloudQuotas(ctx, s.settings)}
}

// AdminSaveSettings saves review_all and, when given, the cloud quotas.
func (s *CommunityService) AdminSaveSettings(ctx context.Context, reviewAll bool, quotas *CanvasCloudQuotaSettings) error {
	if s.settings == nil {
		return nil
	}
	if quotas != nil {
		if err := saveCanvasCloudQuotas(ctx, s.settings, *quotas); err != nil {
			return err
		}
	}
	v := "false"
	if reviewAll {
		v = "true"
	}
	return s.settings.SetMultiple(ctx, map[string]string{settingCommunityReviewAll: v})
}
