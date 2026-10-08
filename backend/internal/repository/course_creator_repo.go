package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Course creators: applications, sales shares and withdrawals (migration 272).

const creatorColumns = `cc.user_id, cc.status, cc.display_name, cc.bio, cc.contact, cc.plan, cc.commission_percent::float8, cc.admin_note,
       cc.reviewed_at, cc.created_at, COALESCE(u.email, ''), COALESCE(u.username, ''),
       (SELECT COUNT(*) FROM courses c WHERE c.owner_id = cc.user_id),
       (SELECT COUNT(*) FROM courses c WHERE c.owner_id = cc.user_id AND c.status = 'published'),
       (SELECT COUNT(*) FROM courses c WHERE c.owner_id = cc.user_id AND c.review_status = 'pending')`

func scanCreator(row rowScanner) (*service.CourseCreator, error) {
	var c service.CourseCreator
	var commission sql.NullFloat64
	var reviewed sql.NullTime
	if err := row.Scan(&c.UserID, &c.Status, &c.DisplayName, &c.Bio, &c.Contact, &c.Plan, &commission, &c.AdminNote,
		&reviewed, &c.CreatedAt, &c.UserEmail, &c.Username, &c.CourseCount, &c.OnSaleCount, &c.PendingCount); err != nil {
		return nil, err
	}
	if commission.Valid {
		v := commission.Float64
		c.CommissionPercent = &v
	}
	if reviewed.Valid {
		c.ReviewedAt = &reviewed.Time
	}
	return &c, nil
}

func (r *courseRepository) GetCreator(ctx context.Context, userID int64) (*service.CourseCreator, error) {
	c, err := scanCreator(r.db.QueryRowContext(ctx, `SELECT `+creatorColumns+`
FROM course_creators cc LEFT JOIN users u ON u.id = cc.user_id WHERE cc.user_id = $1`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *courseRepository) SaveCreatorApplication(ctx context.Context, c *service.CourseCreator) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO course_creators (user_id, status, display_name, bio, contact, plan) VALUES ($1, 'pending', $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET status = 'pending', display_name = EXCLUDED.display_name, bio = EXCLUDED.bio,
    contact = EXCLUDED.contact, plan = EXCLUDED.plan, admin_note = '', updated_at = NOW()
WHERE course_creators.status IN ('pending', 'rejected')`, c.UserID, c.DisplayName, c.Bio, c.Contact, c.Plan)
	return err
}

func (r *courseRepository) ListCreators(ctx context.Context, status string) ([]service.CourseCreator, error) {
	query := `SELECT ` + creatorColumns + ` FROM course_creators cc LEFT JOIN users u ON u.id = cc.user_id`
	var args []any
	if status != "" {
		query += ` WHERE cc.status = $1`
		args = append(args, status)
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY (cc.status = 'pending') DESC, cc.created_at DESC LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CourseCreator
	for rows.Next() {
		c, err := scanCreator(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *courseRepository) UpdateCreator(ctx context.Context, userID int64, status string, commission *float64, note string) error {
	var pct any
	if commission != nil {
		pct = *commission
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE course_creators SET status = $2::text, commission_percent = $3::numeric, admin_note = $4, updated_at = NOW(),
       reviewed_at = CASE WHEN status <> $2::text THEN NOW() ELSE reviewed_at END
WHERE user_id = $1`, userID, status, pct, note)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrCreatorNotFound
	}
	return nil
}

func (r *courseRepository) ArchiveCreatorCourses(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE courses SET status = 'archived', updated_at = NOW() WHERE owner_id = $1 AND status = 'published'`, userID)
	return err
}

// Sales ------------------------------------------------------------------------------------------

// RecordCreatorSale: the gross is what the buyer paid without the payment fee; the rate is the
// creator's own or the default, fixed at the time of sale.
func (r *courseRepository) RecordCreatorSale(ctx context.Context, orderID int64, defaultPercent float64, settleDays int) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO course_creator_earnings (order_id, creator_id, course_id, gross_cny, commission_percent, available_at)
SELECT po.id, c.owner_id, c.id, ROUND((po.pay_amount / (1 + po.fee_rate / 100.0))::numeric, 2),
       COALESCE(cc.commission_percent, $2::numeric), NOW() + make_interval(days => $3::int)
FROM payment_orders po
JOIN courses c ON c.id = po.course_id
LEFT JOIN course_creators cc ON cc.user_id = c.owner_id
WHERE po.id = $1 AND po.order_type = 'course' AND c.owner_id IS NOT NULL AND po.pay_amount > 0
ON CONFLICT (order_id) DO NOTHING`, orderID, defaultPercent, settleDays)
	return err
}

// creatorEarningsCTE values each sale: refunds come off by the order's refunded share, and only
// completed (or finished refund) orders past the settlement date are settled.
const creatorEarningsCTE = `
WITH e AS (
    SELECT ce.*, po.status AS order_status,
           CASE WHEN po.status = 'REFUNDED' THEN 0
                WHEN po.status = 'PARTIALLY_REFUNDED' AND po.amount > 0 THEN GREATEST(0, 1 - COALESCE(po.refund_amount, 0) / po.amount)
                ELSE 1 END AS factor,
           (ce.available_at <= NOW() AND po.status IN ('COMPLETED', 'PARTIALLY_REFUNDED', 'REFUNDED')) AS settled
    FROM course_creator_earnings ce JOIN payment_orders po ON po.id = ce.order_id
    WHERE ce.creator_id = $1
)`

func (r *courseRepository) CreatorBalance(ctx context.Context, userID int64) (*service.CreatorBalance, error) {
	return creatorBalance(ctx, r.db, userID)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func creatorBalance(ctx context.Context, q queryRower, userID int64) (*service.CreatorBalance, error) {
	var b service.CreatorBalance
	var settled float64
	err := q.QueryRowContext(ctx, creatorEarningsCTE+`
SELECT COUNT(*) FILTER (WHERE factor > 0),
       COALESCE(SUM(gross_cny * factor), 0)::float8,
       COALESCE(SUM(gross_cny * factor * (1 - commission_percent / 100)), 0)::float8,
       COALESCE(SUM(gross_cny * factor * (1 - commission_percent / 100)) FILTER (WHERE settled), 0)::float8,
       COALESCE((SELECT SUM(cny_amount) FROM course_creator_withdrawals WHERE user_id = $1 AND status = 'paid'), 0)::float8,
       COALESCE((SELECT SUM(cny_amount) FROM course_creator_withdrawals WHERE user_id = $1 AND status = 'pending'), 0)::float8
FROM e`, userID).Scan(&b.Orders, &b.Gross, &b.Net, &settled, &b.Paid, &b.Pending)
	if err != nil {
		return nil, err
	}
	b.Gross, b.Net = roundCNY(b.Gross), roundCNY(b.Net)
	b.Frozen = roundCNY(b.Net - settled)
	b.Available = roundCNY(settled - b.Paid - b.Pending)
	return &b, nil
}

func roundCNY(v float64) float64 {
	if v < 0 {
		return -roundCNY(-v)
	}
	return float64(int64(v*100+0.5)) / 100
}

func (r *courseRepository) CreatorSales(ctx context.Context, userID int64, limit, offset int) ([]service.CreatorSale, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM course_creator_earnings WHERE creator_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, creatorEarningsCTE+`
SELECT e.order_id, e.course_id, COALESCE(c.title, ''), COALESCE(u.email, ''),
       (e.gross_cny * e.factor)::float8, e.commission_percent::float8,
       (e.gross_cny * e.factor * (1 - e.commission_percent / 100))::float8,
       CASE WHEN e.order_status = 'REFUNDED' THEN 'refunded'
            WHEN e.order_status IN ('REFUND_REQUESTED', 'REFUNDING', 'REFUND_PENDING', 'REFUND_FAILED') THEN 'refunding'
            WHEN e.order_status = 'PARTIALLY_REFUNDED' AND NOT e.settled THEN 'partially_refunded'
            WHEN e.settled THEN 'available'
            ELSE 'frozen' END,
       e.available_at, e.created_at
FROM e
LEFT JOIN courses c ON c.id = e.course_id
LEFT JOIN payment_orders po ON po.id = e.order_id
LEFT JOIN users u ON u.id = po.user_id
ORDER BY e.created_at DESC
LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CreatorSale
	for rows.Next() {
		var s service.CreatorSale
		if err := rows.Scan(&s.OrderID, &s.CourseID, &s.CourseTitle, &s.Buyer, &s.Gross, &s.CommissionPercent, &s.Net, &s.Status,
			&s.AvailableAt, &s.CreatedAt); err != nil {
			return nil, 0, err
		}
		s.Gross, s.Net = roundCNY(s.Gross), roundCNY(s.Net)
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// Withdrawals ------------------------------------------------------------------------------------

const creatorWithdrawalColumns = `w.id, w.user_id, COALESCE(u.email, ''), COALESCE(cc.display_name, ''), w.cny_amount::float8, w.method,
       w.account_enc, w.real_name_enc, w.user_note, w.status, w.admin_note, w.reviewed_at, w.created_at`

const creatorWithdrawalFrom = ` FROM course_creator_withdrawals w
LEFT JOIN users u ON u.id = w.user_id
LEFT JOIN course_creators cc ON cc.user_id = w.user_id`

func scanCreatorWithdrawal(row rowScanner) (*service.CreatorWithdrawal, error) {
	var w service.CreatorWithdrawal
	var reviewed sql.NullTime
	if err := row.Scan(&w.ID, &w.UserID, &w.UserEmail, &w.DisplayName, &w.CNYAmount, &w.Method, &w.Account, &w.RealName,
		&w.UserNote, &w.Status, &w.AdminNote, &reviewed, &w.CreatedAt); err != nil {
		return nil, err
	}
	if reviewed.Valid {
		w.ReviewedAt = &reviewed.Time
	}
	return &w, nil
}

func (r *courseRepository) CreateCreatorWithdrawal(ctx context.Context, w *service.CreatorWithdrawal, check func(*service.CreatorBalance) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// One request at a time per creator: the creator row is the lock.
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM course_creators WHERE user_id = $1 FOR UPDATE`, w.UserID); err != nil {
		return err
	}
	b, err := creatorBalance(ctx, tx, w.UserID)
	if err != nil {
		return err
	}
	if err := check(b); err != nil {
		return err
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO course_creator_withdrawals (user_id, cny_amount, method, account_enc, real_name_enc, user_note)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, status, created_at`,
		w.UserID, w.CNYAmount, w.Method, w.Account, w.RealName, w.UserNote).Scan(&w.ID, &w.Status, &w.CreatedAt)
	if isUniqueViolation(err) {
		return service.ErrWithdrawPending
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *courseRepository) ListCreatorWithdrawals(ctx context.Context, f service.CreatorWithdrawFilter) ([]service.CreatorWithdrawal, int, error) {
	var where []string
	var args []any
	if f.UserID > 0 {
		args = append(args, f.UserID)
		where = append(where, fmt.Sprintf("w.user_id = $%d", len(args)))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("w.status = $%d", len(args)))
	}
	cond := ""
	if len(where) > 0 {
		cond = ` WHERE ` + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+creatorWithdrawalFrom+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.db.QueryContext(ctx, `SELECT `+creatorWithdrawalColumns+creatorWithdrawalFrom+cond+
		fmt.Sprintf(` ORDER BY (w.status = 'pending') DESC, w.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CreatorWithdrawal
	for rows.Next() {
		w, err := scanCreatorWithdrawal(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *w)
	}
	if out == nil {
		out = []service.CreatorWithdrawal{}
	}
	return out, total, rows.Err()
}

func (r *courseRepository) ResolveCreatorWithdrawal(ctx context.Context, id int64, status, note string, reviewerID, ownerID int64) (*service.CreatorWithdrawal, error) {
	var reviewer any
	if reviewerID > 0 {
		reviewer = reviewerID
	}
	query := `UPDATE course_creator_withdrawals SET status = $2, admin_note = $3, reviewed_by = $4, reviewed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'pending'`
	args := []any{id, status, note, reviewer}
	if ownerID > 0 {
		query += ` AND user_id = $5`
		args = append(args, ownerID)
	}
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var current string
		err := r.db.QueryRowContext(ctx, `SELECT status FROM course_creator_withdrawals WHERE id = $1 AND ($2 = 0 OR user_id = $2)`, id, ownerID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrWithdrawNotFound
		}
		if err != nil {
			return nil, err
		}
		return nil, service.ErrWithdrawNotPending
	}
	return scanCreatorWithdrawal(r.db.QueryRowContext(ctx, `SELECT `+creatorWithdrawalColumns+creatorWithdrawalFrom+` WHERE w.id = $1`, id))
}
