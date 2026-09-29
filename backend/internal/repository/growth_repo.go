package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type growthRepository struct {
	db *sql.DB
}

// NewGrowthRepository creates the growth-program repository (raw SQL).
func NewGrowthRepository(db *sql.DB) service.GrowthRepository {
	return &growthRepository{db: db}
}

// ---------------------------------------------------------------------------
// Education verification
// ---------------------------------------------------------------------------

func (r *growthRepository) GetEduVerification(ctx context.Context, userID int64) (*service.EduVerification, error) {
	var v service.EduVerification
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, email, verified_at FROM user_edu_verifications WHERE user_id = $1`, userID,
	).Scan(&v.UserID, &v.Email, &v.VerifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get edu verification: %w", err)
	}
	return &v, nil
}

func (r *growthRepository) EduEmailOwner(ctx context.Context, email string) (int64, error) {
	var userID int64
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM user_edu_verifications WHERE LOWER(email) = LOWER($1)`, email,
	).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("find edu email owner: %w", err)
	}
	return userID, nil
}

func (r *growthRepository) UpsertEduVerification(ctx context.Context, userID int64, email string) (*service.EduVerification, error) {
	var v service.EduVerification
	err := r.db.QueryRowContext(ctx, `
INSERT INTO user_edu_verifications (user_id, email, verified_at, created_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET email = EXCLUDED.email, verified_at = NOW()
RETURNING user_id, email, verified_at`, userID, email,
	).Scan(&v.UserID, &v.Email, &v.VerifiedAt)
	if isUniqueViolation(err) {
		// The PK conflict is handled above, so this is the email index: another account holds it.
		return nil, service.ErrEduEmailTaken
	}
	if err != nil {
		return nil, fmt.Errorf("upsert edu verification: %w", err)
	}
	return &v, nil
}

func (r *growthRepository) DeleteEduVerification(ctx context.Context, userID int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM user_edu_verifications WHERE user_id = $1`, userID)
	if err != nil {
		return false, fmt.Errorf("delete edu verification: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *growthRepository) ListEduVerifications(ctx context.Context, search string, page, pageSize int) ([]service.EduVerification, int64, error) {
	where := ""
	args := []any{}
	if search != "" {
		args = append(args, "%"+escapeLike(search)+"%")
		where = `WHERE v.email ILIKE $1 OR u.email ILIKE $1 OR u.username ILIKE $1`
	}
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_edu_verifications v JOIN users u ON u.id = v.user_id `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count edu verifications: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
SELECT v.user_id, v.email, v.verified_at, u.email, COALESCE(u.username, '')
FROM user_edu_verifications v JOIN users u ON u.id = v.user_id
%s
ORDER BY v.verified_at DESC, v.user_id DESC
LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list edu verifications: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []service.EduVerification{}
	for rows.Next() {
		var v service.EduVerification
		if err := rows.Scan(&v.UserID, &v.Email, &v.VerifiedAt, &v.UserEmail, &v.Username); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

// ---------------------------------------------------------------------------
// Invitee first-order bonus
// ---------------------------------------------------------------------------

func (r *growthRepository) GrantInviteeBonus(ctx context.Context, inviteeID, orderID int64, amount float64, bindingNotBefore *time.Time) (bool, int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, 0, fmt.Errorf("begin invitee bonus tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		inviterID sql.NullInt64
		boundAt   time.Time
	)
	err = tx.QueryRowContext(ctx,
		`SELECT inviter_id, created_at FROM user_affiliates WHERE user_id = $1 FOR UPDATE`, inviteeID,
	).Scan(&inviterID, &boundAt)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!inviterID.Valid || inviterID.Int64 <= 0)) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("load invitee affiliate: %w", err)
	}
	if bindingNotBefore != nil && boundAt.Before(*bindingNotBefore) {
		return false, inviterID.Int64, nil
	}

	// Only the invitee's first paid gateway order qualifies; balance-paid orders never count.
	var (
		orderUserID int64
		orderType   string
		orderPaidAt sql.NullTime
	)
	err = tx.QueryRowContext(ctx,
		`SELECT user_id, payment_type, paid_at FROM payment_orders WHERE id = $1`, orderID,
	).Scan(&orderUserID, &orderType, &orderPaidAt)
	if err != nil {
		return false, inviterID.Int64, fmt.Errorf("load bonus order: %w", err)
	}
	if orderUserID != inviteeID || orderType == payment.TypeBalance || !orderPaidAt.Valid {
		return false, inviterID.Int64, nil
	}
	var hasEarlier bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM payment_orders
    WHERE user_id = $1 AND id <> $2 AND payment_type <> $3 AND paid_at IS NOT NULL
      AND (paid_at < $4 OR (paid_at = $4 AND id < $2))
)`, inviteeID, orderID, payment.TypeBalance, orderPaidAt.Time).Scan(&hasEarlier); err != nil {
		return false, inviterID.Int64, fmt.Errorf("check earlier orders: %w", err)
	}
	if hasEarlier {
		return false, inviterID.Int64, nil
	}

	var ledgerID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, source_user_id, source_order_id, created_at, updated_at)
VALUES ($1, 'invitee_bonus', $2, $3, $4, NOW(), NOW())
ON CONFLICT (user_id) WHERE action = 'invitee_bonus' DO NOTHING
RETURNING id`, inviteeID, amount, inviterID.Int64, orderID).Scan(&ledgerID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, inviterID.Int64, nil // already granted
	}
	if err != nil {
		return false, inviterID.Int64, fmt.Errorf("record invitee bonus: %w", err)
	}
	var balanceAfter float64
	if err := tx.QueryRowContext(ctx,
		`UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL RETURNING balance`,
		amount, inviteeID,
	).Scan(&balanceAfter); err != nil {
		return false, inviterID.Int64, fmt.Errorf("credit invitee bonus: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE user_affiliate_ledger SET balance_after = $1 WHERE id = $2`, balanceAfter, ledgerID,
	); err != nil {
		return false, inviterID.Int64, fmt.Errorf("record bonus balance snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, inviterID.Int64, fmt.Errorf("commit invitee bonus: %w", err)
	}
	return true, inviterID.Int64, nil
}

// ---------------------------------------------------------------------------
// Invite leaderboard
// ---------------------------------------------------------------------------

// inviteLeaderboardCTE ranks inviters over [$1, $2) (NULL = unbounded):
//   - invited: invitees whose affiliate record was created in the period;
//   - paying: distinct invitees with a paid gateway order in the period (balance-paid orders excluded,
//     refunded orders excluded) and the sum of those orders;
//   - rebate: rebate accrued to the inviter in the period.
//
// Ranked by paying invitees, then invitees; ties share a rank.
const inviteLeaderboardCTE = `
WITH invited AS (
    SELECT ua.inviter_id AS uid, COUNT(*) AS invited
    FROM user_affiliates ua
    WHERE ua.inviter_id IS NOT NULL
      AND ($1::timestamptz IS NULL OR ua.created_at >= $1)
      AND ($2::timestamptz IS NULL OR ua.created_at < $2)
    GROUP BY ua.inviter_id
), paying AS (
    SELECT ua.inviter_id AS uid, COUNT(DISTINCT po.user_id) AS paying, COALESCE(SUM(po.amount), 0) AS paid
    FROM payment_orders po
    JOIN user_affiliates ua ON ua.user_id = po.user_id AND ua.inviter_id IS NOT NULL
    WHERE po.payment_type <> 'balance'
      AND po.status IN ('COMPLETED', 'PAID', 'RECHARGING', 'PARTIALLY_REFUNDED')
      AND po.paid_at IS NOT NULL
      AND ($1::timestamptz IS NULL OR po.paid_at >= $1)
      AND ($2::timestamptz IS NULL OR po.paid_at < $2)
    GROUP BY ua.inviter_id
), rebates AS (
    SELECT l.user_id AS uid, COALESCE(SUM(l.amount), 0) AS rebate
    FROM user_affiliate_ledger l
    WHERE l.action = 'accrue'
      AND ($1::timestamptz IS NULL OR l.created_at >= $1)
      AND ($2::timestamptz IS NULL OR l.created_at < $2)
    GROUP BY l.user_id
), combined AS (
    SELECT COALESCE(i.uid, p.uid) AS uid, COALESCE(i.invited, 0) AS invited,
           COALESCE(p.paying, 0) AS paying, COALESCE(p.paid, 0) AS paid
    FROM invited i FULL OUTER JOIN paying p ON p.uid = i.uid
), ranked AS (
    SELECT c.uid, c.invited, c.paying, c.paid,
           RANK() OVER (ORDER BY c.paying DESC, c.invited DESC) AS rnk
    FROM combined c
    JOIN users u ON u.id = c.uid AND u.deleted_at IS NULL
)
SELECT r.rnk, r.uid, u.email, COALESCE(u.username, ''), r.invited, r.paying, r.paid,
       COALESCE(rb.rebate, 0), ua.aff_rebate_rate_percent
FROM ranked r
JOIN users u ON u.id = r.uid
LEFT JOIN rebates rb ON rb.uid = r.uid
LEFT JOIN user_affiliates ua ON ua.user_id = r.uid
`

func (r *growthRepository) InviteLeaderboard(ctx context.Context, start, end *time.Time, limit int) ([]service.InviteLeaderboardEntry, error) {
	rows, err := r.db.QueryContext(ctx, inviteLeaderboardCTE+`ORDER BY r.rnk, r.uid LIMIT $3`,
		nullableTime(start), nullableTime(end), limit)
	if err != nil {
		return nil, fmt.Errorf("invite leaderboard: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []service.InviteLeaderboardEntry{}
	for rows.Next() {
		entry, err := scanLeaderboardEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *entry)
	}
	return out, rows.Err()
}

func (r *growthRepository) InviteLeaderboardEntryFor(ctx context.Context, userID int64, start, end *time.Time) (*service.InviteLeaderboardEntry, error) {
	row := r.db.QueryRowContext(ctx, inviteLeaderboardCTE+`WHERE r.uid = $3`,
		nullableTime(start), nullableTime(end), userID)
	entry, err := scanLeaderboardEntry(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return entry, err
}

func scanLeaderboardEntry(scan func(dest ...any) error) (*service.InviteLeaderboardEntry, error) {
	var (
		e          service.InviteLeaderboardEntry
		customRate sql.NullFloat64
	)
	if err := scan(&e.Rank, &e.UserID, &e.Email, &e.Username, &e.InvitedCount, &e.PayingInvitees,
		&e.InviteePaid, &e.RebateAccrued, &customRate); err != nil {
		return nil, err
	}
	if customRate.Valid {
		v := customRate.Float64
		e.CustomRatePercent = &v
	}
	return &e, nil
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
