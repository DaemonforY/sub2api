package service

import (
	"context"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Contest lifecycle status (persisted).
const (
	ContestStatusDraft     = "draft"
	ContestStatusPublished = "published"
	ContestStatusSettled   = "settled"
	ContestStatusCancelled = "cancelled"
)

// Contest phase (derived from status + time; never persisted).
const (
	ContestPhaseDraft      = "draft"
	ContestPhaseUpcoming   = "upcoming"
	ContestPhaseSubmitting = "submitting"
	ContestPhaseVoting     = "voting"
	// ContestPhaseSubmitAndVote: submission and voting windows overlap.
	ContestPhaseSubmitAndVote = "submitting_voting"
	// ContestPhaseWaitingVote: submissions closed, voting not started yet.
	ContestPhaseWaitingVote = "waiting_vote"
	// ContestPhaseTallying: voting closed, settlement not run yet.
	ContestPhaseTallying  = "tallying"
	ContestPhaseSettled   = "settled"
	ContestPhaseCancelled = "cancelled"
)

// Contest entry status.
const (
	ContestEntryPending      = "pending"
	ContestEntryApproved     = "approved"
	ContestEntryRejected     = "rejected"
	ContestEntryWithdrawn    = "withdrawn"
	ContestEntryDisqualified = "disqualified"
)

// Prize types and award statuses.
const (
	ContestPrizeBalance = "balance" // credited to the user's account balance on grant
	ContestPrizeCustom  = "custom"  // physical / off-platform prize, marked delivered manually

	ContestAwardPending  = "pending"
	ContestAwardGranting = "granting"
	ContestAwardGranted  = "granted"
	ContestAwardFailed   = "failed"
)

var (
	ErrContestNotFound       = infraerrors.NotFound("CONTEST_NOT_FOUND", "活动不存在或已下线")
	ErrContestEntryNotFound  = infraerrors.NotFound("CONTEST_ENTRY_NOT_FOUND", "作品不存在或已被撤回")
	ErrContestAwardNotFound  = infraerrors.NotFound("CONTEST_AWARD_NOT_FOUND", "contest award not found")
	ErrContestVoteNotFound   = infraerrors.NotFound("CONTEST_VOTE_NOT_FOUND", "contest vote not found")
	ErrContestInvalid        = infraerrors.BadRequest("CONTEST_INVALID", "invalid contest settings")
	ErrContestNotEditable    = infraerrors.Conflict("CONTEST_NOT_EDITABLE", "settled or cancelled contests cannot be edited")
	ErrContestNotDeletable   = infraerrors.Conflict("CONTEST_NOT_DELETABLE", "only draft or cancelled contests can be deleted")
	ErrContestNotSubmitting  = infraerrors.Forbidden("CONTEST_NOT_ACCEPTING_ENTRIES", "活动现在不在投稿时间内，请在活动页查看投稿开始和截止时间")
	ErrContestWorkEntered    = infraerrors.Conflict("CONTEST_WORK_ENTERED", "这个作品已经投过这个活动了（This work is already entered）")
	ErrContestEntryLimit     = infraerrors.Conflict("CONTEST_ENTRY_LIMIT", "你在这个活动的投稿数已达上限；如需更换作品，可以先在活动页撤回旧作品再投")
	ErrContestNotVoting      = infraerrors.Forbidden("CONTEST_NOT_VOTING", "活动现在不在投票时间内，请在活动页查看投票时间")
	ErrContestVoteLimit      = infraerrors.Conflict("CONTEST_VOTE_LIMIT", "你的票已经投完了；可以先撤回一票，再投给其他作品")
	ErrContestAlreadyVoted   = infraerrors.Conflict("CONTEST_ALREADY_VOTED", "你已经给这幅作品投过票了")
	ErrContestNotVoted       = infraerrors.Conflict("CONTEST_NOT_VOTED", "你还没有给这幅作品投票")
	ErrContestSelfVote       = infraerrors.Forbidden("CONTEST_SELF_VOTE", "不能给自己的作品投票")
	ErrContestAccountTooNew  = infraerrors.Forbidden("CONTEST_ACCOUNT_TOO_NEW", "账号注册时间太短，暂时不能投票（本活动要求注册满一定时长），请稍后再来")
	ErrContestEntryNotOpen   = infraerrors.Forbidden("CONTEST_ENTRY_NOT_VOTABLE", "这幅作品还在审核中或已下架，暂时不能投票")
	ErrContestNotSettleable  = infraerrors.Conflict("CONTEST_NOT_SETTLEABLE", "the contest can only be settled after voting has ended")
	ErrContestAwardNotGrant  = infraerrors.Conflict("CONTEST_AWARD_NOT_GRANTABLE", "this award has already been granted or is being granted")
	ErrContestImageInvalid   = infraerrors.BadRequest("CONTEST_IMAGE_INVALID", "图片格式不支持，请上传 PNG、JPEG、WebP 或 GIF")
	ErrContestImageTooLarge  = infraerrors.BadRequest("CONTEST_IMAGE_TOO_LARGE", "图片太大（上限 10MB），请压缩后再投稿")
	ErrContestWithdrawClosed = infraerrors.Forbidden("CONTEST_WITHDRAW_CLOSED", "投稿已截止，作品不能再撤回")
)

// ContestPrize is one prize tier covering places rank_from..rank_to (inclusive, 1-based).
type ContestPrize struct {
	RankFrom int     `json:"rank_from"`
	RankTo   int     `json:"rank_to"`
	Type     string  `json:"type"`
	Amount   float64 `json:"amount"`
	Label    string  `json:"label"`
}

// Contest is an activity such as a drawing contest.
type Contest struct {
	ID                 int64          `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	Rules              string         `json:"rules"`
	CoverImage         string         `json:"cover_image"`
	Status             string         `json:"status"`
	SubmissionStartAt  time.Time      `json:"submission_start_at"`
	SubmissionEndAt    time.Time      `json:"submission_end_at"`
	VotingStartAt      time.Time      `json:"voting_start_at"`
	VotingEndAt        time.Time      `json:"voting_end_at"`
	MaxEntriesPerUser  int            `json:"max_entries_per_user"`
	VotesPerUser       int            `json:"votes_per_user"`
	AllowSelfVote      bool           `json:"allow_self_vote"`
	RequireReview      bool           `json:"require_review"`
	MinAccountAgeHours int            `json:"min_account_age_hours"`
	OnePrizePerUser    bool           `json:"one_prize_per_user"`
	MinVotesForPrize   int            `json:"min_votes_for_prize"`
	Prizes             []ContestPrize `json:"prizes"`
	SettledAt          *time.Time     `json:"settled_at,omitempty"`
	CreatedBy          *int64         `json:"created_by,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`

	// Read-side fields.
	EntryCount int    `json:"entry_count"`
	VoteCount  int    `json:"vote_count"`
	Phase      string `json:"phase"`
}

// ContestEntry is one submitted artwork.
type ContestEntry struct {
	ID          int64     `json:"id"`
	ContestID   int64     `json:"contest_id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	ImageFile   string    `json:"-"`
	Status      string    `json:"status"`
	ReviewNote  string    `json:"review_note"`
	VoteCount   int       `json:"vote_count"`
	FinalRank   *int      `json:"final_rank,omitempty"`
	FinalVotes  *int      `json:"final_votes,omitempty"`
	WorkID      *int64    `json:"work_id,omitempty"` // the canvas community work it was made from
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Read-side fields.
	ImageURL   string `json:"image_url"`
	AuthorName string `json:"author_name"`
	UserEmail  string `json:"user_email,omitempty"` // admin views only
	VotedByMe  bool   `json:"voted_by_me"`
	IsMine     bool   `json:"is_mine"`
}

// ContestVote is one vote (admin views).
type ContestVote struct {
	ID        int64     `json:"id"`
	ContestID int64     `json:"contest_id"`
	EntryID   int64     `json:"entry_id"`
	UserID    int64     `json:"user_id"`
	ClientIP  string    `json:"client_ip"`
	CreatedAt time.Time `json:"created_at"`
	UserEmail string    `json:"user_email,omitempty"`
	// UserCreatedAt helps admins spot freshly-registered voter accounts.
	UserCreatedAt *time.Time `json:"user_created_at,omitempty"`
}

// ContestAward is a prize assigned at settlement.
type ContestAward struct {
	ID        int64      `json:"id"`
	ContestID int64      `json:"contest_id"`
	EntryID   int64      `json:"entry_id"`
	UserID    int64      `json:"user_id"`
	Place     int        `json:"place"`
	PrizeType string     `json:"prize_type"`
	Amount    float64    `json:"amount"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	Note      string     `json:"note"`
	GrantedAt *time.Time `json:"granted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	EntryTitle string `json:"entry_title,omitempty"`
	UserEmail  string `json:"user_email,omitempty"`
	AuthorName string `json:"author_name,omitempty"`
}

// ContestRankedEntry is an approved entry with its vote tally at a cutoff.
type ContestRankedEntry struct {
	EntryID    int64
	UserID     int64
	Votes      int
	LastVoteAt *time.Time
	CreatedAt  time.Time
	Rank       int
}

// ContestEntryFilter selects entries.
type ContestEntryFilter struct {
	ContestID int64
	// Statuses restricts the result; empty = all.
	Statuses []string
	UserID   *int64
	// Sort: "votes" (default) or "new".
	Sort     string
	Page     int
	PageSize int
}

// ContestRepository persists contests, entries, votes and awards.
type ContestRepository interface {
	CreateContest(ctx context.Context, c *Contest) error
	UpdateContest(ctx context.Context, c *Contest) error
	DeleteContest(ctx context.Context, id int64) error
	GetContest(ctx context.Context, id int64) (*Contest, error)
	ListContests(ctx context.Context, statuses []string) ([]*Contest, error)
	ListContestIDsDueForSettlement(ctx context.Context, now time.Time) ([]int64, error)

	CreateEntry(ctx context.Context, e *ContestEntry) error
	CountActiveUserEntries(ctx context.Context, contestID, userID int64) (int, error)
	GetEntry(ctx context.Context, id int64) (*ContestEntry, error)
	ListEntries(ctx context.Context, filter ContestEntryFilter) ([]*ContestEntry, int, error)
	// UpdateEntryStatus changes status; leaving "approved" also deletes the entry's votes (refunding voters).
	UpdateEntryStatus(ctx context.Context, id int64, status, note string) error

	// CastVote inserts a vote atomically, enforcing the per-user limit under a lock.
	CastVote(ctx context.Context, v *ContestVote, votesPerUser int) error
	RemoveVote(ctx context.Context, entryID, userID int64) (bool, error)
	VoidVote(ctx context.Context, contestID, voteID int64) (bool, error)
	CountUserVotes(ctx context.Context, contestID, userID int64) (int, error)
	ListUserVotedEntryIDs(ctx context.Context, contestID, userID int64) ([]int64, error)
	ListEntryVotes(ctx context.Context, entryID int64) ([]*ContestVote, error)

	// RankEntries tallies approved entries using only votes cast at or before cutoff.
	RankEntries(ctx context.Context, contestID int64, cutoff time.Time) ([]ContestRankedEntry, error)
	// SettleContest atomically writes final ranks and awards and marks the contest settled.
	// Returns false when the contest was no longer in "published" status (already settled).
	SettleContest(ctx context.Context, contestID int64, ranked []ContestRankedEntry, awards []*ContestAward, settledAt time.Time) (bool, error)
	ListAwards(ctx context.Context, contestID int64) ([]*ContestAward, error)
	// ClaimAwardForGrant moves pending/failed → granting and returns the award; ok=false if not claimable.
	ClaimAwardForGrant(ctx context.Context, contestID, awardID int64) (*ContestAward, bool, error)
	FinishAwardGrant(ctx context.Context, awardID int64, status, note string, grantedAt *time.Time) error
}

// ContestPhaseAt derives the phase of a contest at time now.
func ContestPhaseAt(c *Contest, now time.Time) string {
	if c == nil {
		return ""
	}
	switch c.Status {
	case ContestStatusDraft:
		return ContestPhaseDraft
	case ContestStatusSettled:
		return ContestPhaseSettled
	case ContestStatusCancelled:
		return ContestPhaseCancelled
	}
	submitting := !now.Before(c.SubmissionStartAt) && now.Before(c.SubmissionEndAt)
	voting := !now.Before(c.VotingStartAt) && now.Before(c.VotingEndAt)
	switch {
	case submitting && voting:
		return ContestPhaseSubmitAndVote
	case submitting:
		return ContestPhaseSubmitting
	case voting:
		return ContestPhaseVoting
	case now.Before(c.SubmissionStartAt):
		return ContestPhaseUpcoming
	case !now.Before(c.VotingEndAt):
		return ContestPhaseTallying
	default:
		return ContestPhaseWaitingVote
	}
}

// ContestAcceptsEntries reports whether the submission window is open.
func ContestAcceptsEntries(c *Contest, now time.Time) bool {
	p := ContestPhaseAt(c, now)
	return p == ContestPhaseSubmitting || p == ContestPhaseSubmitAndVote
}

// ContestAcceptsVotes reports whether the voting window is open.
func ContestAcceptsVotes(c *Contest, now time.Time) bool {
	p := ContestPhaseAt(c, now)
	return p == ContestPhaseVoting || p == ContestPhaseSubmitAndVote
}

func contestInvalid(msg string) error {
	return infraerrors.BadRequest("CONTEST_INVALID", msg)
}

// NormalizeAndValidateContest trims fields, applies defaults and validates settings.
func NormalizeAndValidateContest(c *Contest) error {
	if c == nil {
		return ErrContestInvalid
	}
	c.Title = strings.TrimSpace(c.Title)
	c.Description = strings.TrimSpace(c.Description)
	c.Rules = strings.TrimSpace(c.Rules)
	c.CoverImage = strings.TrimSpace(c.CoverImage)
	if c.Title == "" || len([]rune(c.Title)) > 200 {
		return contestInvalid("title is required (max 200 characters)")
	}
	if c.CoverImage != "" && !strings.HasPrefix(c.CoverImage, "https://") && !strings.HasPrefix(c.CoverImage, "/") {
		return contestInvalid("cover image must be an https URL or a site-relative path")
	}
	if c.SubmissionStartAt.IsZero() || c.SubmissionEndAt.IsZero() || c.VotingStartAt.IsZero() || c.VotingEndAt.IsZero() {
		return contestInvalid("all four schedule times are required")
	}
	if !c.SubmissionStartAt.Before(c.SubmissionEndAt) {
		return contestInvalid("submission end must be after submission start")
	}
	if !c.VotingStartAt.Before(c.VotingEndAt) {
		return contestInvalid("voting end must be after voting start")
	}
	if c.VotingStartAt.Before(c.SubmissionStartAt) {
		return contestInvalid("voting cannot start before submissions open")
	}
	if c.VotingEndAt.Before(c.SubmissionEndAt) {
		return contestInvalid("voting must not end before submissions close")
	}
	if c.MaxEntriesPerUser < 1 || c.MaxEntriesPerUser > 50 {
		return contestInvalid("max entries per user must be between 1 and 50")
	}
	if c.VotesPerUser < 1 || c.VotesPerUser > 1000 {
		return contestInvalid("votes per user must be between 1 and 1000")
	}
	if c.MinAccountAgeHours < 0 || c.MinAccountAgeHours > 24*365 {
		return contestInvalid("minimum account age must be between 0 and 8760 hours")
	}
	if c.MinVotesForPrize < 0 {
		return contestInvalid("minimum votes for a prize cannot be negative")
	}
	if c.Prizes == nil {
		c.Prizes = []ContestPrize{}
	}
	sort.SliceStable(c.Prizes, func(i, j int) bool { return c.Prizes[i].RankFrom < c.Prizes[j].RankFrom })
	lastTo := 0
	for i := range c.Prizes {
		p := &c.Prizes[i]
		p.Type = strings.TrimSpace(p.Type)
		p.Label = strings.TrimSpace(p.Label)
		if p.RankFrom < 1 || p.RankTo < p.RankFrom || p.RankTo > 1000 {
			return contestInvalid("prize ranks must satisfy 1 <= rank_from <= rank_to <= 1000")
		}
		if p.RankFrom <= lastTo {
			return contestInvalid("prize rank ranges must not overlap")
		}
		lastTo = p.RankTo
		switch p.Type {
		case ContestPrizeBalance:
			if p.Amount <= 0 || p.Amount > 100000 {
				return contestInvalid("balance prize amount must be between 0 and 100000")
			}
		case ContestPrizeCustom:
			if p.Label == "" {
				return contestInvalid("custom prizes need a label describing the prize")
			}
			p.Amount = 0
		default:
			return contestInvalid("prize type must be balance or custom")
		}
		if len([]rune(p.Label)) > 200 {
			return contestInvalid("prize label max 200 characters")
		}
	}
	return nil
}

// AssignContestRanks sorts ranked entries (votes desc, earliest final vote, earliest submission)
// and writes 1-based ranks in place.
func AssignContestRanks(ranked []ContestRankedEntry) {
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Votes != b.Votes {
			return a.Votes > b.Votes
		}
		switch {
		case a.LastVoteAt != nil && b.LastVoteAt != nil && !a.LastVoteAt.Equal(*b.LastVoteAt):
			return a.LastVoteAt.Before(*b.LastVoteAt)
		case a.LastVoteAt != nil && b.LastVoteAt == nil:
			return true
		case a.LastVoteAt == nil && b.LastVoteAt != nil:
			return false
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.EntryID < b.EntryID
	})
	for i := range ranked {
		ranked[i].Rank = i + 1
	}
}

// BuildContestAwards walks ranked entries (already ranked) and assigns prize tiers by award place.
// With onePrizePerUser, a user's lower-ranked entries are skipped and places shift down.
func BuildContestAwards(contestID int64, ranked []ContestRankedEntry, prizes []ContestPrize, onePrizePerUser bool, minVotes int) []*ContestAward {
	if len(prizes) == 0 {
		return nil
	}
	prizeFor := func(place int) *ContestPrize {
		for i := range prizes {
			if place >= prizes[i].RankFrom && place <= prizes[i].RankTo {
				return &prizes[i]
			}
		}
		return nil
	}
	maxPlace := 0
	for _, p := range prizes {
		if p.RankTo > maxPlace {
			maxPlace = p.RankTo
		}
	}
	awarded := make(map[int64]bool)
	awards := make([]*ContestAward, 0, maxPlace)
	place := 0
	for _, r := range ranked {
		if place >= maxPlace {
			break
		}
		if r.Votes < minVotes {
			break // ranked by votes desc, nothing below qualifies
		}
		if onePrizePerUser && awarded[r.UserID] {
			continue
		}
		place++
		prize := prizeFor(place)
		if prize == nil {
			continue // gap between tiers: place counts but carries no prize
		}
		awarded[r.UserID] = true
		awards = append(awards, &ContestAward{
			ContestID: contestID,
			EntryID:   r.EntryID,
			UserID:    r.UserID,
			Place:     place,
			PrizeType: prize.Type,
			Amount:    prize.Amount,
			Label:     prize.Label,
			Status:    ContestAwardPending,
		})
	}
	return awards
}

// ContestAuthorName returns a public display name: username, else a masked email.
func ContestAuthorName(username, email string) string {
	if u := strings.TrimSpace(username); u != "" {
		return u
	}
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local, domain := []rune(email[:at]), email[at:]
	if len(local) <= 2 {
		return string(local[:1]) + "***" + domain
	}
	return string(local[:2]) + "***" + domain
}
