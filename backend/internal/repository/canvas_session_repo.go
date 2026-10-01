package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type canvasSessionRepository struct {
	db *sql.DB
}

// NewCanvasSessionRepository stores the canvas sign-in sessions (raw SQL).
func NewCanvasSessionRepository(db *sql.DB) service.CanvasSessionRepository {
	return &canvasSessionRepository{db: db}
}

const canvasSessionColumns = `id, user_id, user_agent, ip, created_at, last_used_at, expires_at, revoked_at`

func scanCanvasSession(row rowScanner) (*service.CanvasSession, error) {
	var s service.CanvasSession
	var revoked sql.NullTime
	if err := row.Scan(&s.ID, &s.UserID, &s.UserAgent, &s.IP, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &revoked); err != nil {
		return nil, err
	}
	if revoked.Valid {
		s.RevokedAt = &revoked.Time
	}
	return &s, nil
}

func (r *canvasSessionRepository) CreateCanvasSession(ctx context.Context, s *service.CanvasSession, tokenHash string) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO canvas_sessions (user_id, token_hash, user_agent, ip, created_at, last_used_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $5, $6) RETURNING id`,
		s.UserID, tokenHash, s.UserAgent, s.IP, s.CreatedAt, s.ExpiresAt).Scan(&s.ID)
}

func (r *canvasSessionRepository) GetCanvasSessionByHash(ctx context.Context, tokenHash string) (*service.CanvasSession, error) {
	s, err := scanCanvasSession(r.db.QueryRowContext(ctx, `SELECT `+canvasSessionColumns+` FROM canvas_sessions WHERE token_hash = $1`, tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (r *canvasSessionRepository) TouchCanvasSession(ctx context.Context, id int64, at, expires time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE canvas_sessions SET last_used_at = $2, expires_at = $3 WHERE id = $1 AND revoked_at IS NULL`, id, at, expires)
	return err
}

func (r *canvasSessionRepository) RevokeCanvasSession(ctx context.Context, userID, id int64, at time.Time) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE canvas_sessions SET revoked_at = $3 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, id, userID, at)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *canvasSessionRepository) ListCanvasSessions(ctx context.Context, userID int64, now time.Time) ([]service.CanvasSession, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+canvasSessionColumns+` FROM canvas_sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > $2 ORDER BY last_used_at DESC`, userID, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.CanvasSession
	for rows.Next() {
		s, err := scanCanvasSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *canvasSessionRepository) TrimCanvasSessions(ctx context.Context, userID int64, keep int, at time.Time) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE canvas_sessions SET revoked_at = $3
WHERE user_id = $1 AND revoked_at IS NULL AND id NOT IN (
    SELECT id FROM canvas_sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC, id DESC LIMIT $2
)`, userID, keep, at)
	return err
}
