package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type contestRepository struct {
	db *sql.DB
}

// NewContestRepository creates the contest repository (raw SQL).
func NewContestRepository(db *sql.DB) service.ContestRepository {
	return &contestRepository{db: db}
}

const contestSelectColumns = `c.id, c.title, c.description, c.rules, c.cover_image, c.status,
c.submission_start_at, c.submission_end_at, c.voting_start_at, c.voting_end_at,
c.max_entries_per_user, c.votes_per_user, c.allow_self_vote, c.require_review,
c.min_account_age_hours, c.one_prize_per_user, c.min_votes_for_prize, c.prizes,
c.settled_at, c.created_by, c.created_at, c.updated_at,
(SELECT COUNT(*) FROM contest_entries e WHERE e.contest_id = c.id AND e.status = 'approved'),
(SELECT COUNT(*) FROM contest_votes v WHERE v.contest_id = c.id)`

func scanContest(scan func(dest ...any) error) (*service.Contest, error) {
	var (
		c         service.Contest
		prizes    []byte
		settledAt sql.NullTime
		createdBy sql.NullInt64
	)
	if err := scan(&c.ID, &c.Title, &c.Description, &c.Rules, &c.CoverImage, &c.Status,
		&c.SubmissionStartAt, &c.SubmissionEndAt, &c.VotingStartAt, &c.VotingEndAt,
		&c.MaxEntriesPerUser, &c.VotesPerUser, &c.AllowSelfVote, &c.RequireReview,
		&c.MinAccountAgeHours, &c.OnePrizePerUser, &c.MinVotesForPrize, &prizes,
		&settledAt, &createdBy, &c.CreatedAt, &c.UpdatedAt, &c.EntryCount, &c.VoteCount); err != nil {
		return nil, err
	}
	c.Prizes = []service.ContestPrize{}
	if len(prizes) > 0 {
		if err := json.Unmarshal(prizes, &c.Prizes); err != nil {
			return nil, fmt.Errorf("decode contest prizes: %w", err)
		}
	}
	if settledAt.Valid {
		t := settledAt.Time
		c.SettledAt = &t
	}
	c.CreatedBy = nullInt64ToPtr(createdBy)
	return &c, nil
}

func contestPrizesJSON(p []service.ContestPrize) (string, error) {
	if p == nil {
		p = []service.ContestPrize{}
	}
	b, err := json.Marshal(p)
	return string(b), err
}

func (r *contestRepository) CreateContest(ctx context.Context, c *service.Contest) error {
	prizes, err := contestPrizesJSON(c.Prizes)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO contests (title, description, rules, cover_image, status,
  submission_start_at, submission_end_at, voting_start_at, voting_end_at,
  max_entries_per_user, votes_per_user, allow_self_vote, require_review,
  min_account_age_hours, one_prize_per_user, min_votes_for_prize, prizes, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18)
RETURNING id`,
		c.Title, c.Description, c.Rules, c.CoverImage, c.Status,
		c.SubmissionStartAt.UTC(), c.SubmissionEndAt.UTC(), c.VotingStartAt.UTC(), c.VotingEndAt.UTC(),
		c.MaxEntriesPerUser, c.VotesPerUser, c.AllowSelfVote, c.RequireReview,
		c.MinAccountAgeHours, c.OnePrizePerUser, c.MinVotesForPrize, prizes, nullInt64Ptr(c.CreatedBy),
	).Scan(&c.ID)
}

func (r *contestRepository) UpdateContest(ctx context.Context, c *service.Contest) error {
	prizes, err := contestPrizesJSON(c.Prizes)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE contests SET title=$2, description=$3, rules=$4, cover_image=$5, status=$6,
  submission_start_at=$7, submission_end_at=$8, voting_start_at=$9, voting_end_at=$10,
  max_entries_per_user=$11, votes_per_user=$12, allow_self_vote=$13, require_review=$14,
  min_account_age_hours=$15, one_prize_per_user=$16, min_votes_for_prize=$17, prizes=$18::jsonb,
  updated_at=NOW()
WHERE id=$1 AND status NOT IN ('settled')`,
		c.ID, c.Title, c.Description, c.Rules, c.CoverImage, c.Status,
		c.SubmissionStartAt.UTC(), c.SubmissionEndAt.UTC(), c.VotingStartAt.UTC(), c.VotingEndAt.UTC(),
		c.MaxEntriesPerUser, c.VotesPerUser, c.AllowSelfVote, c.RequireReview,
		c.MinAccountAgeHours, c.OnePrizePerUser, c.MinVotesForPrize, prizes)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrContestNotEditable
	}
	return nil
}

func (r *contestRepository) DeleteContest(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM contests WHERE id=$1`, id)
	return err
}

func (r *contestRepository) GetContest(ctx context.Context, id int64) (*service.Contest, error) {
	c, err := scanContest(r.db.QueryRowContext(ctx, `SELECT `+contestSelectColumns+` FROM contests c WHERE c.id=$1`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContestNotFound
	}
	return c, err
}

func (r *contestRepository) ListContests(ctx context.Context, statuses []string) ([]*service.Contest, error) {
	query := `SELECT ` + contestSelectColumns + ` FROM contests c`
	args := []any{}
	if len(statuses) > 0 {
		query += ` WHERE c.status = ANY($1)`
		args = append(args, pq.Array(statuses))
	}
	query += ` ORDER BY c.voting_end_at DESC, c.id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*service.Contest{}
	for rows.Next() {
		c, err := scanContest(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *contestRepository) ListContestIDsDueForSettlement(ctx context.Context, now time.Time) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM contests WHERE status='published' AND voting_end_at <= $1 ORDER BY id`, now.UTC())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ---------------------------------------------------------------------------
// Entries
// ---------------------------------------------------------------------------

const contestEntrySelect = `SELECT e.id, e.contest_id, e.user_id, e.title, e.description, e.prompt, e.image_file,
e.status, e.review_note, e.vote_count, e.final_rank, e.final_votes, e.work_id, e.created_at, e.updated_at,
COALESCE(u.username, ''), COALESCE(u.email, '')
FROM contest_entries e LEFT JOIN users u ON u.id = e.user_id`

func scanContestEntry(scan func(dest ...any) error) (*service.ContestEntry, error) {
	var (
		e                    service.ContestEntry
		finalRank, finalVote sql.NullInt64
		workID               sql.NullInt64
		username, email      string
	)
	if err := scan(&e.ID, &e.ContestID, &e.UserID, &e.Title, &e.Description, &e.Prompt, &e.ImageFile,
		&e.Status, &e.ReviewNote, &e.VoteCount, &finalRank, &finalVote, &workID, &e.CreatedAt, &e.UpdatedAt,
		&username, &email); err != nil {
		return nil, err
	}
	if finalRank.Valid {
		v := int(finalRank.Int64)
		e.FinalRank = &v
	}
	if finalVote.Valid {
		v := int(finalVote.Int64)
		e.FinalVotes = &v
	}
	if workID.Valid {
		e.WorkID = &workID.Int64
	}
	e.AuthorName = service.ContestAuthorName(username, email)
	e.UserEmail = email
	return &e, nil
}

func (r *contestRepository) CreateEntry(ctx context.Context, e *service.ContestEntry) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO contest_entries (contest_id, user_id, title, description, prompt, image_file, status, work_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at, updated_at`,
		e.ContestID, e.UserID, e.Title, e.Description, e.Prompt, e.ImageFile, e.Status, e.WorkID,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "uq_contest_entries_contest_work" {
		return service.ErrContestWorkEntered
	}
	return err
}

func (r *contestRepository) CountActiveUserEntries(ctx context.Context, contestID, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contest_entries
WHERE contest_id=$1 AND user_id=$2 AND status NOT IN ('withdrawn','rejected')`, contestID, userID).Scan(&n)
	return n, err
}

func (r *contestRepository) GetEntry(ctx context.Context, id int64) (*service.ContestEntry, error) {
	e, err := scanContestEntry(r.db.QueryRowContext(ctx, contestEntrySelect+` WHERE e.id=$1`, id).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrContestEntryNotFound
	}
	return e, err
}

func (r *contestRepository) ListEntries(ctx context.Context, f service.ContestEntryFilter) ([]*service.ContestEntry, int, error) {
	clauses := []string{"e.contest_id = $1"}
	args := []any{f.ContestID}
	if len(f.Statuses) > 0 {
		args = append(args, pq.Array(f.Statuses))
		clauses = append(clauses, "e.status = ANY($"+itoa(len(args))+")")
	}
	if f.UserID != nil {
		args = append(args, *f.UserID)
		clauses = append(clauses, "e.user_id = $"+itoa(len(args)))
	}
	where := " WHERE " + strings.Join(clauses, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contest_entries e`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := " ORDER BY e.vote_count DESC, e.created_at ASC, e.id ASC"
	switch f.Sort {
	case "new":
		order = " ORDER BY e.created_at DESC, e.id DESC"
	case "rank":
		order = " ORDER BY e.final_rank ASC NULLS LAST, e.vote_count DESC, e.id ASC"
	}
	page, size := f.Page, f.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 24
	}
	args = append(args, size, (page-1)*size)
	rows, err := r.db.QueryContext(ctx, contestEntrySelect+where+order+
		" LIMIT $"+itoa(len(args)-1)+" OFFSET $"+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []*service.ContestEntry{}
	for rows.Next() {
		e, err := scanContestEntry(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *contestRepository) UpdateEntryStatus(ctx context.Context, id int64, status, note string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `UPDATE contest_entries SET status=$2, review_note=$3, updated_at=NOW() WHERE id=$1`, id, status, note)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrContestEntryNotFound
	}
	if status != service.ContestEntryApproved {
		if _, err := tx.ExecContext(ctx, `DELETE FROM contest_votes WHERE entry_id=$1`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE contest_entries SET vote_count=0 WHERE id=$1`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------------------------------------------------------------------------
// Votes
// ---------------------------------------------------------------------------

func (r *contestRepository) CastVote(ctx context.Context, v *service.ContestVote, votesPerUser int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Serialize this user's votes within the contest so the per-user limit cannot be exceeded by racing requests.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('contest_vote:' || $1::text || ':' || $2::text, 0))`,
		v.ContestID, v.UserID); err != nil {
		return err
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM contest_votes WHERE entry_id=$1 AND user_id=$2)`, v.EntryID, v.UserID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return service.ErrContestAlreadyVoted
	}
	var used int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM contest_votes WHERE contest_id=$1 AND user_id=$2`, v.ContestID, v.UserID).Scan(&used); err != nil {
		return err
	}
	if used >= votesPerUser {
		return service.ErrContestVoteLimit
	}
	res, err := tx.ExecContext(ctx, `UPDATE contest_entries SET vote_count = vote_count + 1 WHERE id=$1 AND contest_id=$2 AND status='approved'`, v.EntryID, v.ContestID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrContestEntryNotOpen
	}
	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if err := tx.QueryRowContext(ctx, `INSERT INTO contest_votes (contest_id, entry_id, user_id, client_ip, created_at)
VALUES ($1,$2,$3,$4,$5) RETURNING id`, v.ContestID, v.EntryID, v.UserID, truncateString(v.ClientIP, 64), createdAt).Scan(&v.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *contestRepository) deleteVoteTx(ctx context.Context, query string, args ...any) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var entryID int64
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE contest_entries SET vote_count = GREATEST(vote_count - 1, 0) WHERE id=$1`, entryID); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (r *contestRepository) RemoveVote(ctx context.Context, entryID, userID int64) (bool, error) {
	return r.deleteVoteTx(ctx, `DELETE FROM contest_votes WHERE entry_id=$1 AND user_id=$2 RETURNING entry_id`, entryID, userID)
}

func (r *contestRepository) VoidVote(ctx context.Context, contestID, voteID int64) (bool, error) {
	return r.deleteVoteTx(ctx, `DELETE FROM contest_votes WHERE id=$1 AND contest_id=$2 RETURNING entry_id`, voteID, contestID)
}

func (r *contestRepository) CountUserVotes(ctx context.Context, contestID, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contest_votes WHERE contest_id=$1 AND user_id=$2`, contestID, userID).Scan(&n)
	return n, err
}

func (r *contestRepository) ListUserVotedEntryIDs(ctx context.Context, contestID, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT entry_id FROM contest_votes WHERE contest_id=$1 AND user_id=$2 ORDER BY id`, contestID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *contestRepository) ListEntryVotes(ctx context.Context, entryID int64) ([]*service.ContestVote, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT v.id, v.contest_id, v.entry_id, v.user_id, v.client_ip, v.created_at,
COALESCE(u.email, ''), u.created_at
FROM contest_votes v LEFT JOIN users u ON u.id = v.user_id
WHERE v.entry_id=$1 ORDER BY v.created_at DESC, v.id DESC`, entryID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*service.ContestVote{}
	for rows.Next() {
		var (
			v             service.ContestVote
			userCreatedAt sql.NullTime
		)
		if err := rows.Scan(&v.ID, &v.ContestID, &v.EntryID, &v.UserID, &v.ClientIP, &v.CreatedAt, &v.UserEmail, &userCreatedAt); err != nil {
			return nil, err
		}
		if userCreatedAt.Valid {
			t := userCreatedAt.Time
			v.UserCreatedAt = &t
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Settlement & awards
// ---------------------------------------------------------------------------

func (r *contestRepository) RankEntries(ctx context.Context, contestID int64, cutoff time.Time) ([]service.ContestRankedEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.user_id, COUNT(v.id)::int, MAX(v.created_at), e.created_at
FROM contest_entries e
LEFT JOIN contest_votes v ON v.entry_id = e.id AND v.created_at <= $2
WHERE e.contest_id = $1 AND e.status = 'approved'
GROUP BY e.id, e.user_id, e.created_at`, contestID, cutoff.UTC())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContestRankedEntry{}
	for rows.Next() {
		var (
			r        service.ContestRankedEntry
			lastVote sql.NullTime
		)
		if err := rows.Scan(&r.EntryID, &r.UserID, &r.Votes, &lastVote, &r.CreatedAt); err != nil {
			return nil, err
		}
		if lastVote.Valid {
			t := lastVote.Time
			r.LastVoteAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *contestRepository) SettleContest(ctx context.Context, contestID int64, ranked []service.ContestRankedEntry, awards []*service.ContestAward, settledAt time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `UPDATE contests SET status='settled', settled_at=$2, updated_at=NOW() WHERE id=$1 AND status='published'`, contestID, settledAt.UTC())
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE contest_entries SET final_rank=NULL, final_votes=NULL WHERE contest_id=$1`, contestID); err != nil {
		return false, err
	}
	if len(ranked) > 0 {
		ids := make([]int64, len(ranked))
		ranks := make([]int64, len(ranked))
		votes := make([]int64, len(ranked))
		for i, re := range ranked {
			ids[i], ranks[i], votes[i] = re.EntryID, int64(re.Rank), int64(re.Votes)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE contest_entries e SET final_rank = d.rank, final_votes = d.votes, updated_at = NOW()
FROM (SELECT unnest($1::bigint[]) AS id, unnest($2::int[]) AS rank, unnest($3::int[]) AS votes) d
WHERE e.id = d.id AND e.contest_id = $4`, pq.Array(ids), pq.Array(ranks), pq.Array(votes), contestID); err != nil {
			return false, err
		}
	}
	for _, a := range awards {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO contest_awards (contest_id, entry_id, user_id, place, prize_type, amount, label, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,'pending') ON CONFLICT (contest_id, entry_id) DO NOTHING`,
			contestID, a.EntryID, a.UserID, a.Place, a.PrizeType, a.Amount, truncateString(a.Label, 200)); err != nil {
			return false, err
		}
	}
	return true, tx.Commit()
}

const contestAwardColumns = `a.id, a.contest_id, a.entry_id, a.user_id, a.place, a.prize_type, a.amount, a.label,
a.status, a.note, a.granted_at, a.created_at`

// contestAwardDest returns scan targets matching contestAwardColumns, followed by extra targets.
func contestAwardDest(a *service.ContestAward, grantedAt *sql.NullTime, extra ...any) []any {
	return append([]any{&a.ID, &a.ContestID, &a.EntryID, &a.UserID, &a.Place, &a.PrizeType,
		&a.Amount, &a.Label, &a.Status, &a.Note, grantedAt, &a.CreatedAt}, extra...)
}

func (r *contestRepository) ListAwards(ctx context.Context, contestID int64) ([]*service.ContestAward, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+contestAwardColumns+`, COALESCE(e.title,''), COALESCE(u.username,''), COALESCE(u.email,'')
FROM contest_awards a
LEFT JOIN contest_entries e ON e.id = a.entry_id
LEFT JOIN users u ON u.id = a.user_id
WHERE a.contest_id=$1 ORDER BY a.place ASC, a.id ASC`, contestID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*service.ContestAward{}
	for rows.Next() {
		var (
			a         service.ContestAward
			grantedAt sql.NullTime
			username  string
		)
		if err := rows.Scan(contestAwardDest(&a, &grantedAt, &a.EntryTitle, &username, &a.UserEmail)...); err != nil {
			return nil, err
		}
		if grantedAt.Valid {
			t := grantedAt.Time
			a.GrantedAt = &t
		}
		a.AuthorName = service.ContestAuthorName(username, a.UserEmail)
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *contestRepository) ClaimAwardForGrant(ctx context.Context, contestID, awardID int64) (*service.ContestAward, bool, error) {
	var (
		a         service.ContestAward
		grantedAt sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, `UPDATE contest_awards a SET status='granting'
WHERE a.id=$1 AND a.contest_id=$2 AND a.status IN ('pending','failed')
RETURNING `+contestAwardColumns, awardID, contestID).Scan(contestAwardDest(&a, &grantedAt)...)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if qerr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM contest_awards WHERE id=$1 AND contest_id=$2)`, awardID, contestID).Scan(&exists); qerr != nil {
			return nil, false, qerr
		}
		if !exists {
			return nil, false, service.ErrContestAwardNotFound
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if grantedAt.Valid {
		t := grantedAt.Time
		a.GrantedAt = &t
	}
	return &a, true, nil
}

func (r *contestRepository) FinishAwardGrant(ctx context.Context, awardID int64, status, note string, grantedAt *time.Time) error {
	var ga any
	if grantedAt != nil {
		ga = grantedAt.UTC()
	}
	_, err := r.db.ExecContext(ctx, `UPDATE contest_awards SET status=$2, note=$3, granted_at=$4 WHERE id=$1`, awardID, status, note, ga)
	return err
}
