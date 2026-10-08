package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type affiliateWithdrawRepository struct {
	client *dbent.Client
}

// NewAffiliateWithdrawRepository stores 返利提现 requests (affiliate_withdrawals) and moves the rebate
// quota in and out of user_affiliates with a ledger row, under the same transaction.
func NewAffiliateWithdrawRepository(client *dbent.Client) service.AffiliateWithdrawRepository {
	return &affiliateWithdrawRepository{client: client}
}

// Cash-backed rebates: matured accruals from completed orders paid in CNY through a payment provider
// (not with balance, not refunded). Each one is worth what its order paid for it, fee excluded:
// rebate × (pay_amount ÷ (1 + fee_rate%)) ÷ amount.
const withdrawEligibilitySQL = `
SELECT ua.aff_quota::double precision,
       cash.quota,
       cash.cny,
       COALESCE(w.quota, 0)::double precision,
       COALESCE(w.cny, 0)::double precision,
       COALESCE(w.month_count, 0)::integer,
       COALESCE(w.pending, 0) > 0
FROM user_affiliates ua
CROSS JOIN LATERAL (
    SELECT COALESCE(SUM(l.amount), 0)::double precision AS quota,
           COALESCE(SUM(l.amount * (po.pay_amount / (1 + po.fee_rate / 100.0)) / po.amount), 0)::double precision AS cny
    FROM user_affiliate_ledger l
    JOIN payment_orders po ON po.id = l.source_order_id
    WHERE l.user_id = ua.user_id
      AND l.action = 'accrue'
      AND (l.frozen_until IS NULL OR l.frozen_until <= NOW())
      AND po.status = 'COMPLETED'
      AND po.payment_type IN ('alipay', 'wxpay', 'alipay_direct', 'wxpay_direct', 'easypay')
      AND po.amount > 0
      AND po.pay_amount > 0
) cash
LEFT JOIN LATERAL (
    SELECT SUM(quota_amount) AS quota,
           SUM(cny_amount) AS cny,
           COUNT(*) FILTER (WHERE created_at >= $2) AS month_count,
           COUNT(*) FILTER (WHERE status = 'pending') AS pending
    FROM affiliate_withdrawals
    WHERE user_id = ua.user_id AND status IN ('pending', 'paid')
) w ON TRUE
WHERE ua.user_id = $1`

func scanWithdrawEligibility(ctx context.Context, q affiliateQueryExecer, userID int64, monthStart time.Time) (*service.WithdrawEligibility, error) {
	rows, err := q.QueryContext(ctx, withdrawEligibilitySQL, userID, monthStart)
	if err != nil {
		return nil, fmt.Errorf("query withdraw eligibility: %w", err)
	}
	defer func() { _ = rows.Close() }()
	e := &service.WithdrawEligibility{}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return e, nil
	}
	if err := rows.Scan(&e.AvailableQuota, &e.CashQuota, &e.CashCNY, &e.WithdrawnQuota, &e.WithdrawnCNY, &e.MonthCount, &e.HasPending); err != nil {
		return nil, err
	}
	return e, rows.Close()
}

func (r *affiliateWithdrawRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin withdraw transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit withdraw transaction: %w", err)
	}
	return nil
}

func (r *affiliateWithdrawRepository) Eligibility(ctx context.Context, userID int64, monthStart time.Time) (*service.WithdrawEligibility, error) {
	var out *service.WithdrawEligibility
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		if _, err := thawFrozenQuotaTx(txCtx, txClient, userID); err != nil {
			return err
		}
		var err error
		out, err = scanWithdrawEligibility(txCtx, txClient, userID, monthStart)
		return err
	})
	return out, err
}

func (r *affiliateWithdrawRepository) Create(ctx context.Context, userID int64, in service.NewWithdrawal, monthStart time.Time, plan service.WithdrawPlanFunc) (*service.AffiliateWithdrawal, error) {
	var out *service.AffiliateWithdrawal
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		if _, err := ensureUserAffiliateWithClient(txCtx, txClient, userID); err != nil {
			return err
		}
		// Lock the affiliate row first so concurrent requests and transfers line up behind us.
		if _, err := txClient.ExecContext(txCtx, `SELECT 1 FROM user_affiliates WHERE user_id = $1 FOR UPDATE`, userID); err != nil {
			return fmt.Errorf("lock user affiliate: %w", err)
		}
		if _, err := thawFrozenQuotaTx(txCtx, txClient, userID); err != nil {
			return err
		}
		e, err := scanWithdrawEligibility(txCtx, txClient, userID, monthStart)
		if err != nil {
			return err
		}
		quota, err := plan(e)
		if err != nil {
			return err
		}

		res, err := txClient.ExecContext(txCtx, `
UPDATE user_affiliates SET aff_quota = aff_quota - $1, updated_at = NOW()
WHERE user_id = $2 AND aff_quota >= $1`, quota, userID)
		if err != nil {
			return fmt.Errorf("deduct affiliate quota: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return service.ErrWithdrawOverAvailable
		}

		rows, err := txClient.QueryContext(txCtx, `
INSERT INTO affiliate_withdrawals (user_id, quota_amount, cny_amount, method, account_enc, real_name_enc, user_note, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', NOW(), NOW())
RETURNING `+withdrawColumns, userID, quota, in.CNYAmount, in.Method, in.Account, in.RealName, in.UserNote)
		if err != nil {
			if isAffiliateUniqueViolation(err) {
				return service.ErrWithdrawPending
			}
			return fmt.Errorf("insert withdrawal: %w", err)
		}
		w, err := scanOneWithdrawal(rows)
		if err != nil {
			return err
		}
		if err := insertWithdrawLedger(txCtx, txClient, userID, "withdraw", quota, w.ID); err != nil {
			return err
		}
		out = w
		return nil
	})
	return out, err
}

func insertWithdrawLedger(ctx context.Context, q affiliateQueryExecer, userID int64, action string, quota float64, withdrawalID int64) error {
	snapshot, err := queryAffiliateTransferSnapshot(ctx, q, userID)
	if err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `
INSERT INTO user_affiliate_ledger (user_id, action, amount, withdrawal_id, balance_after, aff_quota_after, aff_frozen_quota_after, aff_history_quota_after, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		userID, action, quota, withdrawalID,
		snapshot.BalanceAfter, snapshot.AvailableQuotaAfter, snapshot.FrozenQuotaAfter, snapshot.HistoryQuotaAfter); err != nil {
		return fmt.Errorf("insert %s ledger: %w", action, err)
	}
	return nil
}

const withdrawColumns = `id, user_id, quota_amount::double precision, cny_amount::double precision, method, account_enc, real_name_enc, user_note, status, admin_note, reviewed_at, created_at`

func scanWithdrawal(row interface{ Scan(...any) error }, w *service.AffiliateWithdrawal, extra ...any) error {
	var reviewedAt sql.NullTime
	if err := row.Scan(append([]any{&w.ID, &w.UserID, &w.QuotaAmount, &w.CNYAmount, &w.Method, &w.Account, &w.RealName,
		&w.UserNote, &w.Status, &w.AdminNote, &reviewedAt, &w.CreatedAt}, extra...)...); err != nil {
		return err
	}
	if reviewedAt.Valid {
		t := reviewedAt.Time
		w.ReviewedAt = &t
	}
	return nil
}

func scanOneWithdrawal(rows *sql.Rows) (*service.AffiliateWithdrawal, error) {
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrWithdrawNotFound
	}
	var w service.AffiliateWithdrawal
	if err := scanWithdrawal(rows, &w); err != nil {
		return nil, err
	}
	return &w, rows.Close()
}

func (r *affiliateWithdrawRepository) ListByUser(ctx context.Context, userID int64, limit int) ([]service.AffiliateWithdrawal, error) {
	rows, err := r.client.QueryContext(ctx, `SELECT `+withdrawColumns+` FROM affiliate_withdrawals WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list user withdrawals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []service.AffiliateWithdrawal{}
	for rows.Next() {
		var w service.AffiliateWithdrawal
		if err := scanWithdrawal(rows, &w); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *affiliateWithdrawRepository) List(ctx context.Context, f service.WithdrawFilter) ([]service.AffiliateWithdrawal, int64, error) {
	where := []string{"TRUE"}
	args := []any{}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, fmt.Sprintf("w.status = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+escapeLike(f.Search)+"%")
		where = append(where, fmt.Sprintf("(u.email ILIKE $%[1]d OR u.username ILIKE $%[1]d OR w.user_id::text = $%[2]d)", len(args), len(args)+1))
		args = append(args, f.Search)
	}
	cond := strings.Join(where, " AND ")

	var total int64
	if err := scanSingleRow(ctx, r.client, `SELECT COUNT(*) FROM affiliate_withdrawals w JOIN users u ON u.id = w.user_id WHERE `+cond, args, &total); err != nil {
		return nil, 0, fmt.Errorf("count withdrawals: %w", err)
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	cols := strings.ReplaceAll("w."+withdrawColumns, ", ", ", w.")
	rows, err := r.client.QueryContext(ctx, `
SELECT `+cols+`, COALESCE(u.email, ''), COALESCE(u.username, '')
FROM affiliate_withdrawals w JOIN users u ON u.id = w.user_id
WHERE `+cond+`
ORDER BY (w.status = 'pending') DESC, w.created_at DESC, w.id DESC
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list withdrawals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := []service.AffiliateWithdrawal{}
	for rows.Next() {
		var w service.AffiliateWithdrawal
		if err := scanWithdrawal(rows, &w, &w.UserEmail, &w.Username); err != nil {
			return nil, 0, err
		}
		out = append(out, w)
	}
	return out, total, rows.Err()
}

func (r *affiliateWithdrawRepository) CountPending(ctx context.Context) (int64, error) {
	var n int64
	err := scanSingleRow(ctx, r.client, `SELECT COUNT(*) FROM affiliate_withdrawals WHERE status = 'pending'`, nil, &n)
	return n, err
}

func (r *affiliateWithdrawRepository) MarkPaid(ctx context.Context, id int64, adminNote string, reviewerID int64) (*service.AffiliateWithdrawal, error) {
	rows, err := r.client.QueryContext(ctx, `
UPDATE affiliate_withdrawals
SET status = 'paid', admin_note = $2, reviewed_by = $3, reviewed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING `+withdrawColumns, id, adminNote, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("mark withdrawal paid: %w", err)
	}
	w, err := scanOneWithdrawal(rows)
	if errors.Is(err, service.ErrWithdrawNotFound) {
		return nil, r.notPendingOrMissing(ctx, id, 0)
	}
	return w, err
}

func (r *affiliateWithdrawRepository) Return(ctx context.Context, id int64, status, adminNote string, reviewerID, ownerID int64) (*service.AffiliateWithdrawal, error) {
	var out *service.AffiliateWithdrawal
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		rows, err := txClient.QueryContext(txCtx, `
UPDATE affiliate_withdrawals
SET status = $2, admin_note = $3, reviewed_by = NULLIF($4::bigint, 0), reviewed_at = NOW(), updated_at = NOW()
WHERE id = $1 AND status = 'pending' AND ($5::bigint = 0 OR user_id = $5::bigint)
RETURNING `+withdrawColumns, id, status, adminNote, reviewerID, ownerID)
		if err != nil {
			return fmt.Errorf("return withdrawal: %w", err)
		}
		w, err := scanOneWithdrawal(rows)
		if errors.Is(err, service.ErrWithdrawNotFound) {
			return r.notPendingOrMissing(txCtx, id, ownerID)
		}
		if err != nil {
			return err
		}
		if _, err := txClient.ExecContext(txCtx, `UPDATE user_affiliates SET aff_quota = aff_quota + $1, updated_at = NOW() WHERE user_id = $2`, w.QuotaAmount, w.UserID); err != nil {
			return fmt.Errorf("return affiliate quota: %w", err)
		}
		if err := insertWithdrawLedger(txCtx, txClient, w.UserID, "withdraw_return", w.QuotaAmount, w.ID); err != nil {
			return err
		}
		out = w
		return nil
	})
	return out, err
}

// notPendingOrMissing tells "already handled" apart from "no such withdrawal (for this user)".
func (r *affiliateWithdrawRepository) notPendingOrMissing(ctx context.Context, id, ownerID int64) error {
	var n int64
	if err := scanSingleRow(ctx, clientFromContext(ctx, r.client),
		`SELECT COUNT(*) FROM affiliate_withdrawals WHERE id = $1 AND ($2::bigint = 0 OR user_id = $2::bigint)`, []any{id, ownerID}, &n); err != nil {
		return err
	}
	if n == 0 {
		return service.ErrWithdrawNotFound
	}
	return service.ErrWithdrawNotPending
}
