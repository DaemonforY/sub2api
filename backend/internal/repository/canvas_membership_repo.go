package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type canvasMembershipRepository struct {
	db *sql.DB
}

// NewCanvasMembershipRepository creates the 创作会员 store (raw SQL, migration 270).
func NewCanvasMembershipRepository(db *sql.DB) service.CanvasMembershipRepository {
	return &canvasMembershipRepository{db: db}
}

func (r *canvasMembershipRepository) Until(ctx context.Context, userID int64) (*time.Time, error) {
	var until time.Time
	err := r.db.QueryRowContext(ctx, `SELECT member_until FROM canvas_memberships WHERE user_id = $1`, userID).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &until, nil
}

func (r *canvasMembershipRepository) Extend(ctx context.Context, userID int64, days int, acceptTerms bool) (time.Time, error) {
	var until time.Time
	err := r.db.QueryRowContext(ctx, `
INSERT INTO canvas_memberships (user_id, member_until, terms_accepted_at)
VALUES ($1, NOW() + make_interval(days => $2), CASE WHEN $3 THEN NOW() END)
ON CONFLICT (user_id) DO UPDATE SET
    member_until = GREATEST(canvas_memberships.member_until, NOW()) + make_interval(days => $2),
    terms_accepted_at = CASE WHEN $3 THEN NOW() ELSE canvas_memberships.terms_accepted_at END,
    updated_at = NOW()
RETURNING member_until`, userID, days, acceptTerms).Scan(&until)
	return until, err
}

func (r *canvasMembershipRepository) Shorten(ctx context.Context, userID int64, days int) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE canvas_memberships
SET member_until = GREATEST(NOW(), member_until - make_interval(days => $2)), updated_at = NOW()
WHERE user_id = $1`, userID, days)
	return err
}

// AcceptTerms stamps the consent; a user who never was a member gets a row that has already ended.
func (r *canvasMembershipRepository) AcceptTerms(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO canvas_memberships (user_id, member_until, terms_accepted_at)
VALUES ($1, NOW(), NOW())
ON CONFLICT (user_id) DO UPDATE SET terms_accepted_at = NOW(), updated_at = NOW()`, userID)
	return err
}

func (r *canvasMembershipRepository) TermsAccepted(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM canvas_memberships WHERE user_id = $1 AND terms_accepted_at IS NOT NULL)`, userID).Scan(&ok)
	return ok, err
}

func (r *canvasMembershipRepository) LogUnmarkedSave(ctx context.Context, save service.CanvasUnmarkedSave) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO canvas_unmarked_saves (user_id, width, height, client_ip, user_agent)
VALUES ($1, $2, $3, $4, $5)`, save.UserID, save.Width, save.Height, save.ClientIP, save.UserAgent)
	return err
}

func (r *canvasMembershipRepository) PruneUnmarkedSaves(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM canvas_unmarked_saves WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *canvasMembershipRepository) ListMembers(ctx context.Context, activeOnly bool, limit int) ([]service.CanvasMember, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT m.user_id, u.email, u.username, m.member_until, m.terms_accepted_at,
       (SELECT COUNT(*) FROM canvas_unmarked_saves s WHERE s.user_id = m.user_id)
FROM canvas_memberships m
JOIN users u ON u.id = m.user_id
WHERE m.member_until > NOW() OR NOT $1
ORDER BY m.member_until DESC
LIMIT $2`, activeOnly, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	members := []service.CanvasMember{}
	for rows.Next() {
		var m service.CanvasMember
		var terms sql.NullTime
		if err := rows.Scan(&m.UserID, &m.Email, &m.Username, &m.MemberUntil, &terms, &m.UnmarkedSaves); err != nil {
			return nil, err
		}
		if terms.Valid {
			m.TermsAcceptedAt = &terms.Time
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
