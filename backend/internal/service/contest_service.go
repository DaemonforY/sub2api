package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	// ContestImageMaxBytes caps an uploaded entry image.
	ContestImageMaxBytes = 10 << 20

	contestSettleInterval = time.Minute
	// ContestImagePublicPrefix is the public route serving entry images.
	ContestImagePublicPrefix = "/api/v1/contest-images/"
)

var contestImageFileRe = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg|webp|gif)$`)

var contestImageExt = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// ContestImageStore stores entry images on local disk (the data volume).
type ContestImageStore struct {
	dir string
}

// NewContestImageStore creates a store rooted at dir.
func NewContestImageStore(dir string) *ContestImageStore {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/contest-images"
	}
	return &ContestImageStore{dir: filepath.Clean(dir)}
}

// Save validates the image by content sniffing and writes it under a random name.
func (s *ContestImageStore) Save(data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrContestImageInvalid
	}
	if len(data) > ContestImageMaxBytes {
		return "", ErrContestImageTooLarge
	}
	ext, ok := contestImageExt[http.DetectContentType(data)]
	if !ok {
		return "", ErrContestImageInvalid
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate image name: %w", err)
	}
	name := hex.EncodeToString(buf[:]) + "." + ext
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("create contest image dir: %w", err)
	}
	tmp := filepath.Join(s.dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("write contest image: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("store contest image: %w", err)
	}
	return name, nil
}

// Path resolves a stored image name to a file path; ok=false for invalid names.
func (s *ContestImageStore) Path(name string) (string, bool) {
	if !contestImageFileRe.MatchString(name) {
		return "", false
	}
	return filepath.Join(s.dir, name), true
}

// Remove deletes a stored image (best effort).
func (s *ContestImageStore) Remove(name string) {
	if p, ok := s.Path(name); ok {
		_ = os.Remove(p)
	}
}

// ContestImageURL returns the public URL path for a stored image name.
func ContestImageURL(name string) string {
	if name == "" {
		return ""
	}
	return ContestImagePublicPrefix + name
}

// contestBalanceGranter credits balance prizes. It mirrors the side effects of an admin
// balance "add" (atomic update, cache invalidation, adjustment record) but deliberately
// skips affiliate rebates: contest prizes are not recharges.
type contestBalanceGranter struct {
	userRepo             UserRepository
	redeemCodeRepo       RedeemCodeRepository
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func (g *contestBalanceGranter) grant(ctx context.Context, userID int64, amount float64, notes string) error {
	if g == nil || g.userRepo == nil {
		return fmt.Errorf("balance granter unavailable")
	}
	if _, err := g.userRepo.AdjustBalance(ctx, userID, amount); err != nil {
		return err
	}
	if g.authCacheInvalidator != nil {
		g.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if g.billingCacheService != nil {
		if err := g.billingCacheService.InvalidateUserBalance(ctx, userID); err != nil {
			logger.LegacyPrintf("service.contest", "invalidate user balance cache failed: user_id=%d err=%v", userID, err)
		}
	}
	if g.redeemCodeRepo != nil {
		if code, err := GenerateRedeemCode(); err == nil {
			now := time.Now()
			uid := userID
			rec := &RedeemCode{Code: code, Type: AdjustmentTypeAdminBalance, Value: amount, Status: StatusUsed, UsedBy: &uid, UsedAt: &now, Notes: notes}
			if err := g.redeemCodeRepo.Create(ctx, rec); err != nil {
				logger.LegacyPrintf("service.contest", "record contest prize adjustment failed: user_id=%d err=%v", userID, err)
			}
		}
	}
	return nil
}

// ContestService implements contest management, submissions, voting and settlement.
type ContestService struct {
	repo     ContestRepository
	userRepo UserRepository
	images   *ContestImageStore
	granter  *contestBalanceGranter
	now      func() time.Time

	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewContestService creates the service.
func NewContestService(repo ContestRepository, userRepo UserRepository, redeemCodeRepo RedeemCodeRepository,
	billingCache *BillingCacheService, authInvalidator APIKeyAuthCacheInvalidator, images *ContestImageStore) *ContestService {
	return &ContestService{
		repo:     repo,
		userRepo: userRepo,
		images:   images,
		granter: &contestBalanceGranter{
			userRepo: userRepo, redeemCodeRepo: redeemCodeRepo,
			billingCacheService: billingCache, authCacheInvalidator: authInvalidator,
		},
		now:    time.Now,
		stopCh: make(chan struct{}),
	}
}

// Images exposes the image store (for the image-serving handler).
func (s *ContestService) Images() *ContestImageStore { return s.images }

// Start launches the auto-settlement loop.
func (s *ContestService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(contestSettleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.SettleDue(context.Background())
			}
		}
	}()
}

// Stop stops the background loop.
func (s *ContestService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.wg.Wait()
}

func (s *ContestService) decorate(c *Contest) *Contest {
	if c != nil {
		c.Phase = ContestPhaseAt(c, s.now())
	}
	return c
}

// ---------------------------------------------------------------------------
// Admin: contest CRUD
// ---------------------------------------------------------------------------

// AdminListContests lists all contests.
func (s *ContestService) AdminListContests(ctx context.Context) ([]*Contest, error) {
	list, err := s.repo.ListContests(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		s.decorate(c)
	}
	return list, nil
}

// AdminGetContest returns one contest.
func (s *ContestService) AdminGetContest(ctx context.Context, id int64) (*Contest, error) {
	c, err := s.repo.GetContest(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.decorate(c), nil
}

// CreateContest creates a contest (draft or published).
func (s *ContestService) CreateContest(ctx context.Context, c *Contest, adminID int64) (*Contest, error) {
	if c.Status == "" {
		c.Status = ContestStatusDraft
	}
	if c.Status != ContestStatusDraft && c.Status != ContestStatusPublished {
		return nil, contestInvalid("new contests must be draft or published")
	}
	if err := NormalizeAndValidateContest(c); err != nil {
		return nil, err
	}
	if adminID > 0 {
		c.CreatedBy = &adminID
	}
	if err := s.repo.CreateContest(ctx, c); err != nil {
		return nil, err
	}
	return s.AdminGetContest(ctx, c.ID)
}

// UpdateContest replaces editable fields. Settled/cancelled contests are frozen.
func (s *ContestService) UpdateContest(ctx context.Context, id int64, in *Contest) (*Contest, error) {
	cur, err := s.repo.GetContest(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == ContestStatusSettled || cur.Status == ContestStatusCancelled {
		return nil, ErrContestNotEditable
	}
	if in.Status == "" {
		in.Status = cur.Status
	}
	if in.Status != ContestStatusDraft && in.Status != ContestStatusPublished {
		return nil, contestInvalid("status can only be switched between draft and published here")
	}
	in.ID = id
	if err := NormalizeAndValidateContest(in); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateContest(ctx, in); err != nil {
		return nil, err
	}
	return s.AdminGetContest(ctx, id)
}

// CancelContest cancels an unsettled contest.
func (s *ContestService) CancelContest(ctx context.Context, id int64) (*Contest, error) {
	cur, err := s.repo.GetContest(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur.Status == ContestStatusSettled {
		return nil, ErrContestNotEditable
	}
	cur.Status = ContestStatusCancelled
	if err := s.repo.UpdateContest(ctx, cur); err != nil {
		return nil, err
	}
	return s.AdminGetContest(ctx, id)
}

// DeleteContest removes a draft or cancelled contest and its images.
func (s *ContestService) DeleteContest(ctx context.Context, id int64) error {
	cur, err := s.repo.GetContest(ctx, id)
	if err != nil {
		return err
	}
	if cur.Status != ContestStatusDraft && cur.Status != ContestStatusCancelled {
		return ErrContestNotDeletable
	}
	entries, _, err := s.repo.ListEntries(ctx, ContestEntryFilter{ContestID: id, PageSize: 100000})
	if err != nil {
		return err
	}
	if err := s.repo.DeleteContest(ctx, id); err != nil {
		return err
	}
	for _, e := range entries {
		s.images.Remove(e.ImageFile)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Public views
// ---------------------------------------------------------------------------

// ContestViewerState describes the logged-in viewer's standing in a contest.
type ContestViewerState struct {
	LoggedIn       bool            `json:"logged_in"`
	VotesUsed      int             `json:"votes_used"`
	VotesLeft      int             `json:"votes_left"`
	VotedEntryIDs  []int64         `json:"voted_entry_ids"`
	MyEntries      []*ContestEntry `json:"my_entries"`
	EntriesLeft    int             `json:"entries_left"`
	CanSubmit      bool            `json:"can_submit"`
	CanVote        bool            `json:"can_vote"`
	VoteBlockedWhy string          `json:"vote_blocked_reason,omitempty"`
}

// ContestDetail is the public detail payload.
type ContestDetail struct {
	Contest *Contest            `json:"contest"`
	Viewer  *ContestViewerState `json:"viewer"`
	Awards  []*ContestAward     `json:"awards,omitempty"`
}

func isPublicContestStatus(status string) bool {
	return status == ContestStatusPublished || status == ContestStatusSettled
}

// ListPublicContests lists published and settled contests, newest first.
func (s *ContestService) ListPublicContests(ctx context.Context) ([]*Contest, error) {
	list, err := s.repo.ListContests(ctx, []string{ContestStatusPublished, ContestStatusSettled})
	if err != nil {
		return nil, err
	}
	for _, c := range list {
		s.decorate(c)
	}
	return list, nil
}

func (s *ContestService) getPublicContest(ctx context.Context, id int64) (*Contest, error) {
	c, err := s.repo.GetContest(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isPublicContestStatus(c.Status) {
		return nil, ErrContestNotFound
	}
	return s.decorate(c), nil
}

// GetPublicContest returns a contest with the viewer's state (viewerID 0 = anonymous).
func (s *ContestService) GetPublicContest(ctx context.Context, id, viewerID int64) (*ContestDetail, error) {
	c, err := s.getPublicContest(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &ContestDetail{Contest: c, Viewer: &ContestViewerState{VotedEntryIDs: []int64{}, MyEntries: []*ContestEntry{}}}
	if c.Status == ContestStatusSettled {
		awards, err := s.repo.ListAwards(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, a := range awards {
			a.UserEmail = "" // public payload never exposes emails
		}
		detail.Awards = awards
	}
	if viewerID <= 0 {
		return detail, nil
	}
	v := detail.Viewer
	v.LoggedIn = true
	if v.VotesUsed, err = s.repo.CountUserVotes(ctx, id, viewerID); err != nil {
		return nil, err
	}
	v.VotesLeft = max(c.VotesPerUser-v.VotesUsed, 0)
	if v.VotedEntryIDs, err = s.repo.ListUserVotedEntryIDs(ctx, id, viewerID); err != nil {
		return nil, err
	}
	uid := viewerID
	mine, _, err := s.repo.ListEntries(ctx, ContestEntryFilter{ContestID: id, UserID: &uid, Sort: "new", PageSize: 100})
	if err != nil {
		return nil, err
	}
	active := 0
	for _, e := range mine {
		s.decorateEntry(e, viewerID, nil)
		if e.Status != ContestEntryWithdrawn && e.Status != ContestEntryRejected {
			active++
		}
	}
	v.MyEntries = mine
	v.EntriesLeft = max(c.MaxEntriesPerUser-active, 0)
	now := s.now()
	v.CanSubmit = ContestAcceptsEntries(c, now) && v.EntriesLeft > 0
	v.CanVote = ContestAcceptsVotes(c, now) && v.VotesLeft > 0
	if ContestAcceptsVotes(c, now) && c.MinAccountAgeHours > 0 {
		if u, err := s.userRepo.GetByID(ctx, viewerID); err == nil && u != nil &&
			now.Sub(u.CreatedAt) < time.Duration(c.MinAccountAgeHours)*time.Hour {
			v.CanVote = false
			v.VoteBlockedWhy = "account_too_new"
		}
	}
	return detail, nil
}

func (s *ContestService) decorateEntry(e *ContestEntry, viewerID int64, voted map[int64]bool) {
	e.ImageURL = ContestImageURL(e.ImageFile)
	e.IsMine = viewerID > 0 && e.UserID == viewerID
	e.VotedByMe = voted[e.ID]
}

// ListPublicEntries lists approved entries; sort "votes" or "new".
func (s *ContestService) ListPublicEntries(ctx context.Context, contestID, viewerID int64, sortBy string, page, pageSize int) ([]*ContestEntry, int, error) {
	if _, err := s.getPublicContest(ctx, contestID); err != nil {
		return nil, 0, err
	}
	entries, total, err := s.repo.ListEntries(ctx, ContestEntryFilter{
		ContestID: contestID, Statuses: []string{ContestEntryApproved}, Sort: sortBy, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	voted := map[int64]bool{}
	if viewerID > 0 {
		ids, err := s.repo.ListUserVotedEntryIDs(ctx, contestID, viewerID)
		if err != nil {
			return nil, 0, err
		}
		for _, id := range ids {
			voted[id] = true
		}
	}
	for _, e := range entries {
		s.decorateEntry(e, viewerID, voted)
		e.UserEmail = ""
	}
	return entries, total, nil
}

// ContestLeaderboardRow is one leaderboard line.
type ContestLeaderboardRow struct {
	Rank       int    `json:"rank"`
	EntryID    int64  `json:"entry_id"`
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	ImageURL   string `json:"image_url"`
	Votes      int    `json:"votes"`
	Final      bool   `json:"final"`
}

// Leaderboard returns the top entries: final ranks once settled, otherwise a live tally.
func (s *ContestService) Leaderboard(ctx context.Context, contestID int64, limit int) ([]ContestLeaderboardRow, error) {
	c, err := s.getPublicContest(ctx, contestID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	entries, _, err := s.repo.ListEntries(ctx, ContestEntryFilter{
		ContestID: contestID, Statuses: []string{ContestEntryApproved}, Sort: "votes", PageSize: 100000,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]*ContestEntry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}

	var ranked []ContestRankedEntry
	final := c.Status == ContestStatusSettled
	if final {
		for _, e := range entries {
			if e.FinalRank != nil {
				votes := 0
				if e.FinalVotes != nil {
					votes = *e.FinalVotes
				}
				ranked = append(ranked, ContestRankedEntry{EntryID: e.ID, Votes: votes, Rank: *e.FinalRank})
			}
		}
		sortRankedByRank(ranked)
	} else {
		cutoff := s.now()
		if cutoff.After(c.VotingEndAt) {
			cutoff = c.VotingEndAt
		}
		if ranked, err = s.repo.RankEntries(ctx, contestID, cutoff); err != nil {
			return nil, err
		}
		AssignContestRanks(ranked)
	}

	rows := make([]ContestLeaderboardRow, 0, min(limit, len(ranked)))
	for _, r := range ranked {
		if len(rows) >= limit {
			break
		}
		e := byID[r.EntryID]
		if e == nil {
			continue
		}
		rows = append(rows, ContestLeaderboardRow{
			Rank: r.Rank, EntryID: e.ID, Title: e.Title, AuthorName: e.AuthorName,
			ImageURL: ContestImageURL(e.ImageFile), Votes: r.Votes, Final: final,
		})
	}
	return rows, nil
}

func sortRankedByRank(r []ContestRankedEntry) {
	sort.Slice(r, func(i, j int) bool { return r[i].Rank < r[j].Rank })
}

// ---------------------------------------------------------------------------
// Submissions
// ---------------------------------------------------------------------------

// ContestEntryInput is a new submission.
type ContestEntryInput struct {
	Title       string
	Description string
	Prompt      string
	Image       []byte
}

// SubmitEntry stores the image and creates an entry for the user.
func (s *ContestService) SubmitEntry(ctx context.Context, contestID, userID int64, in ContestEntryInput) (*ContestEntry, error) {
	c, err := s.getPublicContest(ctx, contestID)
	if err != nil {
		return nil, err
	}
	if !ContestAcceptsEntries(c, s.now()) {
		return nil, ErrContestNotSubmitting
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || len([]rune(title)) > 120 {
		return nil, contestInvalid("title is required (max 120 characters)")
	}
	desc := strings.TrimSpace(in.Description)
	prompt := strings.TrimSpace(in.Prompt)
	if len([]rune(desc)) > 2000 || len([]rune(prompt)) > 4000 {
		return nil, contestInvalid("description max 2000 characters, prompt max 4000 characters")
	}
	count, err := s.repo.CountActiveUserEntries(ctx, contestID, userID)
	if err != nil {
		return nil, err
	}
	if count >= c.MaxEntriesPerUser {
		return nil, ErrContestEntryLimit
	}
	name, err := s.images.Save(in.Image)
	if err != nil {
		return nil, err
	}
	status := ContestEntryApproved
	if c.RequireReview {
		status = ContestEntryPending
	}
	e := &ContestEntry{ContestID: contestID, UserID: userID, Title: title, Description: desc, Prompt: prompt, ImageFile: name, Status: status}
	if err := s.repo.CreateEntry(ctx, e); err != nil {
		s.images.Remove(name)
		return nil, err
	}
	s.decorateEntry(e, userID, nil)
	return e, nil
}

// WithdrawEntry lets an author withdraw their own entry before voting ends.
func (s *ContestService) WithdrawEntry(ctx context.Context, contestID, entryID, userID int64) error {
	c, err := s.getPublicContest(ctx, contestID)
	if err != nil {
		return err
	}
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return err
	}
	if e.ContestID != contestID || e.UserID != userID {
		return ErrContestEntryNotFound
	}
	if c.Status == ContestStatusSettled || !s.now().Before(c.VotingEndAt) {
		return ErrContestWithdrawClosed
	}
	if e.Status == ContestEntryWithdrawn {
		return nil
	}
	return s.repo.UpdateEntryStatus(ctx, entryID, ContestEntryWithdrawn, "")
}

// ---------------------------------------------------------------------------
// Voting
// ---------------------------------------------------------------------------

func (s *ContestService) loadVotable(ctx context.Context, contestID, entryID int64) (*Contest, *ContestEntry, error) {
	c, err := s.getPublicContest(ctx, contestID)
	if err != nil {
		return nil, nil, err
	}
	if !ContestAcceptsVotes(c, s.now()) {
		return nil, nil, ErrContestNotVoting
	}
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return nil, nil, err
	}
	if e.ContestID != contestID {
		return nil, nil, ErrContestEntryNotFound
	}
	return c, e, nil
}

// Vote casts one vote for an entry.
func (s *ContestService) Vote(ctx context.Context, contestID, entryID, userID int64, clientIP string) error {
	c, e, err := s.loadVotable(ctx, contestID, entryID)
	if err != nil {
		return err
	}
	if e.Status != ContestEntryApproved {
		return ErrContestEntryNotOpen
	}
	if !c.AllowSelfVote && e.UserID == userID {
		return ErrContestSelfVote
	}
	if c.MinAccountAgeHours > 0 {
		u, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return err
		}
		if s.now().Sub(u.CreatedAt) < time.Duration(c.MinAccountAgeHours)*time.Hour {
			return ErrContestAccountTooNew
		}
	}
	return s.repo.CastVote(ctx, &ContestVote{ContestID: contestID, EntryID: entryID, UserID: userID, ClientIP: clientIP, CreatedAt: s.now().UTC()}, c.VotesPerUser)
}

// Unvote retracts the user's vote while voting is still open.
func (s *ContestService) Unvote(ctx context.Context, contestID, entryID, userID int64) error {
	if _, _, err := s.loadVotable(ctx, contestID, entryID); err != nil {
		return err
	}
	removed, err := s.repo.RemoveVote(ctx, entryID, userID)
	if err != nil {
		return err
	}
	if !removed {
		return ErrContestNotVoted
	}
	return nil
}

// ---------------------------------------------------------------------------
// Admin: moderation
// ---------------------------------------------------------------------------

// AdminListEntries lists entries of any status.
func (s *ContestService) AdminListEntries(ctx context.Context, contestID int64, statuses []string, sortBy string, page, pageSize int) ([]*ContestEntry, int, error) {
	if _, err := s.repo.GetContest(ctx, contestID); err != nil {
		return nil, 0, err
	}
	entries, total, err := s.repo.ListEntries(ctx, ContestEntryFilter{ContestID: contestID, Statuses: statuses, Sort: sortBy, Page: page, PageSize: pageSize})
	if err != nil {
		return nil, 0, err
	}
	for _, e := range entries {
		s.decorateEntry(e, 0, nil)
	}
	return entries, total, nil
}

// ReviewEntry sets an entry to approved / rejected / disqualified.
// Leaving "approved" removes the entry's votes so voters get them back.
func (s *ContestService) ReviewEntry(ctx context.Context, contestID, entryID int64, status, note string) error {
	switch status {
	case ContestEntryApproved, ContestEntryRejected, ContestEntryDisqualified:
	default:
		return contestInvalid("status must be approved, rejected or disqualified")
	}
	c, err := s.repo.GetContest(ctx, contestID)
	if err != nil {
		return err
	}
	if c.Status == ContestStatusSettled {
		return ErrContestNotEditable
	}
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return err
	}
	if e.ContestID != contestID {
		return ErrContestEntryNotFound
	}
	note = strings.TrimSpace(note)
	if len([]rune(note)) > 500 {
		note = string([]rune(note)[:500])
	}
	return s.repo.UpdateEntryStatus(ctx, entryID, status, note)
}

// AdminListEntryVotes lists votes on one entry (with voter emails / IPs for fraud review).
func (s *ContestService) AdminListEntryVotes(ctx context.Context, contestID, entryID int64) ([]*ContestVote, error) {
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if e.ContestID != contestID {
		return nil, ErrContestEntryNotFound
	}
	return s.repo.ListEntryVotes(ctx, entryID)
}

// VoidVote deletes a fraudulent vote (before settlement).
func (s *ContestService) VoidVote(ctx context.Context, contestID, voteID int64) error {
	c, err := s.repo.GetContest(ctx, contestID)
	if err != nil {
		return err
	}
	if c.Status == ContestStatusSettled {
		return ErrContestNotEditable
	}
	ok, err := s.repo.VoidVote(ctx, contestID, voteID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrContestVoteNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Settlement & prizes
// ---------------------------------------------------------------------------

// SettleContest freezes the ranking at voting_end_at and creates awards. Idempotent.
func (s *ContestService) SettleContest(ctx context.Context, contestID int64) (*Contest, error) {
	c, err := s.repo.GetContest(ctx, contestID)
	if err != nil {
		return nil, err
	}
	if c.Status == ContestStatusSettled {
		return s.decorate(c), nil
	}
	if c.Status != ContestStatusPublished || s.now().Before(c.VotingEndAt) {
		return nil, ErrContestNotSettleable
	}
	ranked, err := s.repo.RankEntries(ctx, contestID, c.VotingEndAt)
	if err != nil {
		return nil, err
	}
	AssignContestRanks(ranked)
	awards := BuildContestAwards(contestID, ranked, c.Prizes, c.OnePrizePerUser, c.MinVotesForPrize)
	if _, err := s.repo.SettleContest(ctx, contestID, ranked, awards, s.now().UTC()); err != nil {
		return nil, err
	}
	return s.AdminGetContest(ctx, contestID)
}

// SettleDue settles every published contest whose voting has ended.
func (s *ContestService) SettleDue(ctx context.Context) {
	ids, err := s.repo.ListContestIDsDueForSettlement(ctx, s.now())
	if err != nil {
		logger.LegacyPrintf("service.contest", "list contests due for settlement failed: %v", err)
		return
	}
	for _, id := range ids {
		if _, err := s.SettleContest(ctx, id); err != nil {
			logger.LegacyPrintf("service.contest", "auto-settle contest %d failed: %v", id, err)
		} else {
			logger.LegacyPrintf("service.contest", "contest %d settled automatically", id)
		}
	}
}

// ListAwards lists a contest's awards (admin).
func (s *ContestService) ListAwards(ctx context.Context, contestID int64) ([]*ContestAward, error) {
	return s.repo.ListAwards(ctx, contestID)
}

// GrantAward delivers one award: balance prizes are credited, custom prizes are marked delivered.
func (s *ContestService) GrantAward(ctx context.Context, contestID, awardID int64, note string) (*ContestAward, error) {
	c, err := s.repo.GetContest(ctx, contestID)
	if err != nil {
		return nil, err
	}
	award, ok, err := s.repo.ClaimAwardForGrant(ctx, contestID, awardID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrContestAwardNotGrant
	}
	note = strings.TrimSpace(note)
	if award.PrizeType == ContestPrizeBalance {
		notes := fmt.Sprintf("活动奖励: %s 第%d名", c.Title, award.Place)
		if err := s.granter.grant(ctx, award.UserID, award.Amount, notes); err != nil {
			_ = s.repo.FinishAwardGrant(ctx, awardID, ContestAwardFailed, truncateRunes(err.Error(), 500), nil)
			return nil, err
		}
	}
	now := s.now().UTC()
	if err := s.repo.FinishAwardGrant(ctx, awardID, ContestAwardGranted, truncateRunes(note, 500), &now); err != nil {
		return nil, err
	}
	award.Status, award.Note, award.GrantedAt = ContestAwardGranted, note, &now
	return award, nil
}

// GrantAllBalanceAwards grants every pending/failed balance award of a contest.
func (s *ContestService) GrantAllBalanceAwards(ctx context.Context, contestID int64) (granted int, failed int, err error) {
	awards, err := s.repo.ListAwards(ctx, contestID)
	if err != nil {
		return 0, 0, err
	}
	for _, a := range awards {
		if a.PrizeType != ContestPrizeBalance || (a.Status != ContestAwardPending && a.Status != ContestAwardFailed) {
			continue
		}
		if _, err := s.GrantAward(ctx, contestID, a.ID, ""); err != nil {
			failed++
			continue
		}
		granted++
	}
	return granted, failed, nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
