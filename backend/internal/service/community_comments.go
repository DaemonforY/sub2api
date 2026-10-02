package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Comments on community works: top-level comments, newest first, each with one level of replies
// (oldest first). Comments hitting sensitive words, or all of them when the admin asks, wait for
// review; the work's author and the commenter may delete them; anyone may report them.

const (
	CommentStatusApproved = "approved"
	CommentStatusPending  = "pending"
	CommentStatusHidden   = "hidden"
	CommentStatusDeleted  = "deleted"
	// CommentStatusRemoved is how a hidden / deleted comment that still has replies is shown.
	CommentStatusRemoved = "removed"

	communityCommentMaxRunes     = 500
	communityCommentsPageSize    = 20
	communityCommentRepliesShown = 3
	communityCommentsPerDay      = 200
	communityCommentsPerDayNew   = 20
	communityCommentsPerMinute   = 5
	communityCommentDuplicateAge = time.Hour

	settingCommunityCommentsEnabled   = "community_comments_enabled"
	settingCommunityCommentsReviewAll = "community_comments_review_all"
)

var (
	ErrCommunityCommentsOff        = infraerrors.Forbidden("COMMUNITY_COMMENTS_OFF", "评论功能暂未开放（Comments are turned off）")
	ErrCommunityCommentsClosed     = infraerrors.Forbidden("COMMUNITY_COMMENTS_CLOSED", "作者关闭了这个作品的评论（The author closed comments on this work）")
	ErrCommunityCommentEmpty       = infraerrors.BadRequest("COMMUNITY_COMMENT_EMPTY", "评论需要 1–500 个字（Comments are 1–500 characters）")
	ErrCommunityCommentNotFound    = infraerrors.NotFound("COMMUNITY_COMMENT_NOT_FOUND", "评论不存在或已删除（Comment not found）")
	ErrCommunityCommentTooMany     = infraerrors.TooManyRequests("COMMUNITY_COMMENT_TOO_MANY", "评论太频繁了，请稍后再试（Too many comments, try again later）")
	ErrCommunityCommentDuplicate   = infraerrors.BadRequest("COMMUNITY_COMMENT_DUPLICATE", "你刚刚发过一样的评论了（You just posted the same comment）")
	ErrCommunityCommentBanned      = infraerrors.Forbidden("COMMUNITY_COMMENT_BANNED", "你的账号已被限制发言，如有疑问请联系客服（Commenting is disabled for this account）")
	ErrCommunityCommentNotAllowed  = infraerrors.Forbidden("COMMUNITY_COMMENT_NOT_ALLOWED", "只能删除自己的评论或自己作品下的评论（You can't delete this comment）")
	ErrCommunityCommentWorkNotOpen = infraerrors.BadRequest("COMMUNITY_COMMENT_WORK_NOT_OPEN", "作品公开后才能评论（Comments open once the work is public）")
)

type WorkComment struct {
	ID       int64            `json:"id"`
	WorkID   int64            `json:"work_id"`
	UserID   int64            `json:"-"`
	ParentID int64            `json:"parent_id,omitempty"`
	Author   *CommunityAuthor `json:"author,omitempty"`
	// ReplyTo is set on replies answering another reply ("回复 @…").
	ReplyTo     *CommunityAuthor `json:"reply_to,omitempty"`
	ReplyToID   int64            `json:"-"`
	ReplyToUser int64            `json:"-"`
	Body        string           `json:"body"`
	Status      string           `json:"status"`
	ReviewFlags []string         `json:"review_flags,omitempty"`
	ReplyCount  int              `json:"reply_count"`
	ReportCount int              `json:"report_count,omitempty"`
	Replies     []WorkComment    `json:"replies,omitempty"`
	IsMine      bool             `json:"is_mine"`
	// CanDelete: the commenter or the work's author.
	CanDelete bool      `json:"can_delete"`
	IP        string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	// Admin list only.
	WorkTitle string `json:"work_title,omitempty"`
	OwnerID   int64  `json:"owner_id,omitempty"`
}

// CommentPage is a page of a work's top-level comments.
type CommentPage struct {
	Comments   []WorkComment `json:"comments"`
	NextOffset int           `json:"next_offset"`
	HasMore    bool          `json:"has_more"`
	// Total approved comments and replies.
	Total int `json:"total"`
	// Enabled site-wide, and open on this work.
	Enabled bool `json:"enabled"`
	Closed  bool `json:"closed"`
}

// CommentInput: ReplyTo is the comment answered (0: a top-level comment).
type CommentInput struct {
	Body    string
	ReplyTo int64
}

// CommentSettings are the admin's comment switches.
type CommentSettings struct {
	CommentsEnabled   bool `json:"comments_enabled"`
	CommentsReviewAll bool `json:"comments_review_all"`
}

func (s *CommunityService) commentSettings(ctx context.Context) CommentSettings {
	out := CommentSettings{CommentsEnabled: true}
	if s.settings == nil {
		return out
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingCommunityCommentsEnabled, settingCommunityCommentsReviewAll})
	if err != nil {
		return out
	}
	out.CommentsEnabled = values[settingCommunityCommentsEnabled] != "false"
	out.CommentsReviewAll = values[settingCommunityCommentsReviewAll] == "true"
	return out
}

func boolSetting(on bool) string {
	if on {
		return "true"
	}
	return "false"
}

func (s *CommunityService) decorateComments(list []WorkComment, viewerID, workOwnerID int64) {
	for i := range list {
		c := &list[i]
		c.IsMine = viewerID != 0 && c.UserID == viewerID
		visible := c.Status == CommentStatusApproved || (c.Status == CommentStatusPending && c.IsMine)
		if !visible {
			// A removed comment kept as a placeholder for its replies.
			c.Status, c.Body, c.Author, c.ReplyTo, c.IsMine = CommentStatusRemoved, "", nil, nil, false
		}
		c.CanDelete = visible && viewerID != 0 && (c.IsMine || workOwnerID == viewerID)
		for _, a := range []*CommunityAuthor{c.Author, c.ReplyTo} {
			if a != nil && a.AvatarURL != "" && !strings.HasPrefix(a.AvatarURL, "https://") {
				a.AvatarURL = CommunityMediaURL(a.AvatarURL)
			}
		}
		if c.ReplyTo != nil && c.ReplyToID == c.ParentID {
			c.ReplyTo = nil // a direct reply to the top-level comment needs no "回复 @…"
		}
		s.decorateComments(c.Replies, viewerID, workOwnerID)
	}
}

// Comments lists a page of a work's top-level comments, each with its first replies.
func (s *CommunityService) Comments(ctx context.Context, workID, viewerID int64, offset int) (*CommentPage, error) {
	w, err := s.visibleWork(ctx, workID, viewerID)
	if err != nil {
		return nil, err
	}
	settings := s.commentSettings(ctx)
	page := &CommentPage{Comments: []WorkComment{}, Total: w.CommentCount, Enabled: settings.CommentsEnabled, Closed: w.CommentsClosed}
	if !settings.CommentsEnabled {
		return page, nil
	}
	offset = max(offset, 0)
	list, err := s.repo.ListComments(ctx, workID, viewerID, communityCommentsPageSize+1, offset)
	if err != nil {
		return nil, err
	}
	if len(list) > communityCommentsPageSize {
		list, page.HasMore = list[:communityCommentsPageSize], true
	}
	if len(list) > 0 {
		ids := make([]int64, len(list))
		for i := range list {
			ids[i] = list[i].ID
		}
		replies, err := s.repo.FirstReplies(ctx, ids, viewerID, communityCommentRepliesShown)
		if err != nil {
			return nil, err
		}
		for i := range list {
			list[i].Replies = replies[list[i].ID]
		}
	}
	if list == nil {
		list = []WorkComment{}
	}
	s.decorateComments(list, viewerID, w.UserID)
	page.Comments, page.NextOffset = list, offset+len(list)
	return page, nil
}

// CommentReplies lists replies to a top-level comment, oldest first.
func (s *CommunityService) CommentReplies(ctx context.Context, commentID, viewerID int64, offset int) ([]WorkComment, bool, error) {
	parent, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return nil, false, err
	}
	if parent == nil || parent.ParentID != 0 {
		return nil, false, ErrCommunityCommentNotFound
	}
	w, err := s.visibleWork(ctx, parent.WorkID, viewerID)
	if err != nil {
		return nil, false, err
	}
	if !s.commentSettings(ctx).CommentsEnabled {
		return []WorkComment{}, false, nil
	}
	list, err := s.repo.ListReplies(ctx, commentID, viewerID, communityCommentsPageSize+1, max(offset, 0))
	if err != nil {
		return nil, false, err
	}
	more := len(list) > communityCommentsPageSize
	if more {
		list = list[:communityCommentsPageSize]
	}
	if list == nil {
		list = []WorkComment{}
	}
	s.decorateComments(list, viewerID, w.UserID)
	return list, more, nil
}

// AddComment posts a comment or a reply; flagged ones (or all, when the admin asks) wait for review.
func (s *CommunityService) AddComment(ctx context.Context, userID, workID int64, in CommentInput, ip string) (*WorkComment, error) {
	settings := s.commentSettings(ctx)
	if !settings.CommentsEnabled {
		return nil, ErrCommunityCommentsOff
	}
	profile, err := s.repo.GetProfileByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrCommunityProfileRequired
	}
	if profile.Status == ProfileStatusBanned {
		return nil, ErrCommunityCommentBanned
	}
	w, err := s.visibleWork(ctx, workID, userID)
	if err != nil {
		return nil, err
	}
	if w.Status != WorkStatusApproved || w.Visibility == WorkVisibilityPrivate {
		return nil, ErrCommunityCommentWorkNotOpen
	}
	if w.CommentsClosed {
		return nil, ErrCommunityCommentsClosed
	}
	body := cleanCommentBody(in.Body)
	if body == "" {
		return nil, ErrCommunityCommentEmpty
	}

	c := &WorkComment{WorkID: workID, UserID: userID, Body: body, Status: CommentStatusApproved, IP: truncateRunes(ip, 64)}
	var target *WorkComment
	if in.ReplyTo > 0 {
		if target, err = s.repo.GetComment(ctx, in.ReplyTo); err != nil {
			return nil, err
		}
		if target == nil || target.WorkID != workID || target.Status != CommentStatusApproved {
			return nil, ErrCommunityCommentNotFound
		}
		c.ParentID, c.ReplyToID, c.ReplyToUser = target.ID, target.ID, target.UserID
		if target.ParentID != 0 {
			c.ParentID = target.ParentID
		}
	}

	limit := communityCommentsPerDay
	if created, err := s.repo.UserCreatedAt(ctx, userID); err == nil && s.now().Sub(created) < communityNewAccountAge {
		limit = communityCommentsPerDayNew
	}
	if n, err := s.repo.CountComments(ctx, userID, s.now().Add(-24*time.Hour)); err != nil {
		return nil, err
	} else if n >= limit {
		return nil, ErrCommunityCommentTooMany
	}
	if n, err := s.repo.CountComments(ctx, userID, s.now().Add(-time.Minute)); err != nil {
		return nil, err
	} else if n >= communityCommentsPerMinute {
		return nil, ErrCommunityCommentTooMany
	}
	if dup, err := s.repo.HasRecentComment(ctx, userID, workID, body, s.now().Add(-communityCommentDuplicateAge)); err != nil {
		return nil, err
	} else if dup {
		return nil, ErrCommunityCommentDuplicate
	}

	if flags := communityTextFlags(body); len(flags) > 0 {
		c.Status, c.ReviewFlags = CommentStatusPending, flags
	} else if settings.CommentsReviewAll {
		c.Status = CommentStatusPending
	}
	if err := s.repo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	if c.Status == CommentStatusApproved {
		s.notifyComment(ctx, c, w)
	}
	saved, err := s.repo.GetComment(ctx, c.ID)
	if err != nil || saved == nil {
		return c, err
	}
	list := []WorkComment{*saved}
	s.decorateComments(list, userID, w.UserID)
	return &list[0], nil
}

func cleanCommentBody(body string) string {
	body = strings.TrimSpace(strings.ToValidUTF8(body, ""))
	// At most two blank lines in a row.
	for strings.Contains(body, "\n\n\n") {
		body = strings.ReplaceAll(body, "\n\n\n", "\n\n")
	}
	if utf8.RuneCountInString(body) > communityCommentMaxRunes {
		return ""
	}
	return body
}

// notifyComment tells the work's author and the comment answered (once each, never oneself).
func (s *CommunityService) notifyComment(ctx context.Context, c *WorkComment, w *Work) {
	detail := truncateRunes(strings.Join(strings.Fields(c.Body), " "), 60)
	if c.ReplyToUser != 0 && c.ReplyToUser != c.UserID {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: c.ReplyToUser, Kind: "reply", ActorID: c.UserID, WorkID: c.WorkID, Detail: detail})
	}
	if w.UserID != c.UserID && w.UserID != c.ReplyToUser {
		_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: w.UserID, Kind: "comment", ActorID: c.UserID, WorkID: c.WorkID, Detail: detail})
	}
}

// DeleteComment: by its author or the work's author.
func (s *CommunityService) DeleteComment(ctx context.Context, userID, commentID int64) error {
	c, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil || c.Status == CommentStatusDeleted || c.Status == CommentStatusHidden {
		return ErrCommunityCommentNotFound
	}
	if c.UserID != userID {
		w, err := s.repo.GetWork(ctx, c.WorkID)
		if err != nil {
			return err
		}
		if w == nil || w.UserID != userID {
			return ErrCommunityCommentNotAllowed
		}
	}
	return s.repo.SetCommentStatus(ctx, commentID, CommentStatusDeleted)
}

// ReportComment files a report about a comment (same queue and limits as works).
func (s *CommunityService) ReportComment(ctx context.Context, viewerID, commentID int64, reason, detail, ip string) error {
	if !WorkReportReasons[reason] {
		return ErrCommunityReportInvalid
	}
	c, err := s.repo.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil || c.Status != CommentStatusApproved {
		return ErrCommunityCommentNotFound
	}
	if _, err := s.visibleWork(ctx, c.WorkID, viewerID); err != nil {
		return err
	}
	n, err := s.repo.CountWorkReportsSince(ctx, ip, s.now().Add(-time.Hour))
	if err != nil {
		return err
	}
	if n >= communityReportsPerIPHour {
		return ErrCommunityReportTooMany
	}
	return s.repo.CreateWorkReport(ctx, &WorkReport{WorkID: c.WorkID, CommentID: commentID, ReporterID: viewerID, Reason: reason, Detail: cleanText(detail, 500), IP: truncateRunes(ip, 64), Status: "open"})
}

// Admin -----------------------------------------------------------------------------------------

// AdminComments lists comments for moderation (status: pending | reported | hidden | "" all).
func (s *CommunityService) AdminComments(ctx context.Context, status string, page int) ([]WorkComment, error) {
	if page < 1 {
		page = 1
	}
	list, err := s.repo.AdminListComments(ctx, status, 50, (page-1)*50)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []WorkComment{}
	}
	for i := range list {
		c := &list[i]
		c.OwnerID = c.UserID
		if c.Author != nil && c.Author.AvatarURL != "" && !strings.HasPrefix(c.Author.AvatarURL, "https://") {
			c.Author.AvatarURL = CommunityMediaURL(c.Author.AvatarURL)
		}
	}
	return list, nil
}

// AdminModerateComment: approve (a pending or hidden comment) | hide.
func (s *CommunityService) AdminModerateComment(ctx context.Context, id int64, action string) error {
	c, err := s.repo.GetComment(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrCommunityCommentNotFound
	}
	switch action {
	case "approve":
		if err := s.repo.SetCommentStatus(ctx, id, CommentStatusApproved); err != nil {
			return err
		}
		if c.Status == CommentStatusPending {
			if w, err := s.repo.GetWork(ctx, c.WorkID); err == nil && w != nil {
				s.notifyComment(ctx, c, w)
			}
		}
		return nil
	case "hide":
		if err := s.repo.SetCommentStatus(ctx, id, CommentStatusHidden); err != nil {
			return err
		}
		if c.Status == CommentStatusApproved {
			_ = s.repo.AddNotification(ctx, &CommunityNotification{UserID: c.UserID, Kind: "comment_hidden", WorkID: c.WorkID, Detail: truncateRunes(c.Body, 60)})
		}
		return nil
	}
	return ErrCommunityReportInvalid
}
