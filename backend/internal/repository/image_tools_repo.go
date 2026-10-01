package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type imageToolsRepository struct {
	db *sql.DB
}

// NewImageToolsRepository creates the image tools usage / billing repository (raw SQL).
func NewImageToolsRepository(db *sql.DB) service.ImageToolsRepository {
	return &imageToolsRepository{db: db}
}

func (r *imageToolsRepository) CountFreeSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_tool_uses WHERE user_id = $1 AND free AND created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *imageToolsRepository) Balance(ctx context.Context, userID int64) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return balance, err
}

func (r *imageToolsRepository) Charge(ctx context.Context, use service.ImageToolUse, freeDaily int, since time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize a user's charges so two parallel runs cannot both take the last free one.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('image_tool_uses'), $1::int)`, use.UserID); err != nil {
		return false, err
	}
	free := false
	if use.Subscribed && freeDaily > 0 {
		var used int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM image_tool_uses WHERE user_id = $1 AND free AND created_at >= $2`, use.UserID, since).Scan(&used); err != nil {
			return false, err
		}
		free = used < freeDaily
	}
	cost := 0.0
	if !free && use.Price > 0 {
		res, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, use.Price, use.UserID)
		if err != nil {
			return false, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return false, service.ErrInsufficientBalance
		}
		cost = use.Price
	}
	var apiKeyID any
	if use.APIKeyID > 0 {
		apiKeyID = use.APIKeyID
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO image_tool_uses (user_id, api_key_id, tool, free, cost, input_bytes, output_bytes, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, use.UserID, apiKeyID, use.Tool, free, cost, use.InputBytes, use.OutputBytes, use.DurationMs); err != nil {
		return false, err
	}
	return free, tx.Commit()
}

func (r *imageToolsRepository) ListUses(ctx context.Context, q service.ImageToolUseQuery) ([]service.ImageToolUseRecord, int64, error) {
	where := []string{"TRUE"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q.UserID > 0 {
		add("t.user_id = ?", q.UserID)
	}
	if q.Keyword != "" {
		add("u.email ILIKE ?", "%"+escapeLike(q.Keyword)+"%")
	}
	if q.Tool != "" {
		add("t.tool = ?", q.Tool)
	}
	if q.From != nil {
		add("t.created_at >= ?", *q.From)
	}
	if q.To != nil {
		add("t.created_at < ?", *q.To)
	}
	from := ` FROM image_tool_uses t JOIN users u ON u.id = t.user_id LEFT JOIN api_keys k ON k.id = t.api_key_id WHERE ` + strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.user_id, u.email, t.api_key_id, COALESCE(k.name, ''), t.tool, t.free, t.cost, t.input_bytes, t.output_bytes, t.duration_ms, t.created_at`+from+
		` ORDER BY t.created_at DESC, t.id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.ImageToolUseRecord
	for rows.Next() {
		var rec service.ImageToolUseRecord
		var keyID sql.NullInt64
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.UserEmail, &keyID, &rec.APIKeyName, &rec.Tool, &rec.Free, &rec.Cost, &rec.InputBytes, &rec.OutputBytes, &rec.DurationMs, &rec.CreatedAt); err != nil {
			return nil, 0, err
		}
		if keyID.Valid {
			rec.APIKeyID = &keyID.Int64
		}
		out = append(out, rec)
	}
	return out, total, rows.Err()
}

func (r *imageToolsRepository) Stats(ctx context.Context, userID int64, since time.Time) ([]service.ImageToolStat, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT tool, COUNT(*), COUNT(*) FILTER (WHERE free), COALESCE(SUM(cost), 0), COUNT(DISTINCT user_id)
FROM image_tool_uses WHERE created_at >= $1 AND ($2::bigint = 0 OR user_id = $2::bigint)
GROUP BY tool ORDER BY tool`, since, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ImageToolStat{}
	for rows.Next() {
		var st service.ImageToolStat
		if err := rows.Scan(&st.Tool, &st.Runs, &st.FreeRuns, &st.Cost, &st.Users); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}
