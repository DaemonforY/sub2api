package repository

import (
	"context"
	"database/sql"
	"errors"
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
