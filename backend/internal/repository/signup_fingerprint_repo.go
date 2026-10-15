package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type signupFingerprintRepository struct {
	db *sql.DB
}

// NewSignupFingerprintRepository stores each sign-up's browser and IP prefix for 注册试用去重.
func NewSignupFingerprintRepository(db *sql.DB) service.SignupFingerprintRepository {
	return &signupFingerprintRepository{db: db}
}

func (r *signupFingerprintRepository) FindRecentTrial(ctx context.Context, visitorID string, since time.Time) (service.FingerprintMatch, error) {
	if visitorID == "" {
		return service.FingerprintMatch{}, nil
	}
	m := service.FingerprintMatch{ByVisitor: true}
	err := r.db.QueryRowContext(ctx, `
SELECT user_id FROM signup_fingerprints
WHERE visitor_id = $1 AND trial_granted AND created_at >= $2
ORDER BY created_at DESC
LIMIT 1`, visitorID, since).Scan(&m.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.FingerprintMatch{}, nil
	}
	return m, err
}

// FindFlaggedMatch looks up the two fingerprints separately so each side stays on its index.
// Only banned accounts and admin-made restrictions are sources (never automatic ones).
func (r *signupFingerprintRepository) FindFlaggedMatch(ctx context.Context, visitorID, ipPrefix string, excludeUserID int64, visitorSince, ipSince time.Time) (service.FingerprintMatch, error) {
	var m service.FingerprintMatch
	if visitorID == "" && ipPrefix == "" {
		return m, nil
	}
	err := r.db.QueryRowContext(ctx, `
SELECT f.user_id, f.by_visitor, CASE WHEN u.status = 'disabled' THEN 'disabled' ELSE 'shadowed' END
FROM (
    SELECT user_id, TRUE AS by_visitor, created_at FROM signup_fingerprints
    WHERE $1 <> '' AND visitor_id = $1 AND created_at >= $4
    UNION ALL
    SELECT user_id, FALSE, created_at FROM signup_fingerprints
    WHERE $2 <> '' AND ip_prefix = $2 AND created_at >= $5
) f
JOIN users u ON u.id = f.user_id
LEFT JOIN user_shadows s ON s.user_id = f.user_id AND s.source = 'admin'
WHERE f.user_id <> $3 AND (u.status = 'disabled' OR s.user_id IS NOT NULL)
ORDER BY f.by_visitor DESC, (u.status = 'disabled') DESC, f.created_at DESC
LIMIT 1`, visitorID, ipPrefix, excludeUserID, visitorSince, ipSince).Scan(&m.UserID, &m.ByVisitor, &m.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return service.FingerprintMatch{}, nil
	}
	return m, err
}

func (r *signupFingerprintRepository) Save(ctx context.Context, fp service.SignupFingerprint) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO signup_fingerprints (user_id, visitor_id, ip_prefix, trial_granted)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO NOTHING`, fp.UserID, fp.VisitorID, fp.IPPrefix, fp.TrialGranted)
	return err
}
