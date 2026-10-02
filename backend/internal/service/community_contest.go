package service

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Entering contests with community works. The chosen image is copied into the contest's image
// store (an entry must not change when the work is edited or deleted later), and the entry links
// back to the work: the work page lists its contests and the contest page links to the work.

// WorkContest is a contest a work is entered in.
type WorkContest struct {
	ContestID int64  `json:"contest_id"`
	Title     string `json:"title"`
	EntryID   int64  `json:"entry_id"`
	Status    string `json:"status"`
	FinalRank *int   `json:"final_rank,omitempty"`
}

// WorkContestInput enters a work in a contest; empty title / description fall back to the work's.
type WorkContestInput struct {
	ContestID   int64
	WorkID      int64
	ImageIndex  int
	Title       string
	Description string
}

var (
	ErrCommunityWorkPrivate       = infraerrors.BadRequest("COMMUNITY_WORK_PRIVATE", "私密作品不能投稿：活动作品是公开展示的，请先把作品改为公开（Make the work public first）")
	ErrCommunityWorkNotApproved   = infraerrors.BadRequest("COMMUNITY_WORK_NOT_APPROVED", "作品审核通过后才能投稿（The work must be approved first）")
	ErrCommunityContestsDisabled  = infraerrors.ServiceUnavailable("COMMUNITY_CONTESTS_DISABLED", "活动功能暂不可用（Contests are unavailable）")
	ErrCommunityWorkImageNotFound = infraerrors.BadRequest("COMMUNITY_WORK_IMAGE_NOT_FOUND", "没有找到要投稿的图片（Image not found）")
)

// SetContests lets works be entered in contests (wired after both services exist).
func (s *CommunityService) SetContests(contests *ContestService) { s.contests = contests }

// EnterContest files one image of the user's own work as a contest entry.
func (s *CommunityService) EnterContest(ctx context.Context, userID int64, in WorkContestInput) (*ContestEntry, error) {
	if s.contests == nil {
		return nil, ErrCommunityContestsDisabled
	}
	w, err := s.repo.GetWork(ctx, in.WorkID)
	if err != nil {
		return nil, err
	}
	if w == nil || w.UserID != userID {
		return nil, ErrCommunityWorkNotFound
	}
	if w.Visibility == WorkVisibilityPrivate {
		return nil, ErrCommunityWorkPrivate
	}
	if w.Status != WorkStatusApproved {
		return nil, ErrCommunityWorkNotApproved
	}
	if in.ImageIndex < 0 || in.ImageIndex >= len(w.Media) {
		return nil, ErrCommunityWorkImageNotFound
	}
	data, err := s.media.ReadForContest(w.Media[in.ImageIndex].File)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = w.Title
	}
	if title == "" {
		title = excerpt(w.Prompt, 40)
	}
	if title == "" {
		title = "AI 作品"
	}
	desc := strings.TrimSpace(in.Description)
	if desc == "" {
		desc = w.Description
	}
	prompt := ""
	if w.ShowPrompt {
		prompt = w.Prompt
	}
	workID := w.ID
	return s.contests.SubmitEntry(ctx, in.ContestID, userID, ContestEntryInput{
		Title: truncateRunes(title, 120), Description: truncateRunes(desc, 2000), Prompt: truncateRunes(prompt, 4000), Image: data, WorkID: &workID,
	})
}

// ReadForContest returns a stored image in a form contests accept: as-is when small enough and of
// an accepted type, otherwise re-encoded as JPEG.
func (s *CommunityMediaStore) ReadForContest(name string) ([]byte, error) {
	p, ok := s.Path(name)
	if !ok {
		return nil, ErrCommunityWorkImageNotFound
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, ErrCommunityWorkImageNotFound
	}
	if _, accepted := contestImageExt[http.DetectContentType(data)]; accepted && len(data) <= ContestImageMaxBytes {
		return data, nil
	}
	img, _, _, err := decodeUpload(data)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := encodeJPEGTo(&out, img, 90); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
