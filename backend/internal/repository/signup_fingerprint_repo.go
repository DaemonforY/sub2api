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

func (r *signupFingerprintRepository) FindRecentTrial(ctx context.Context, visitorID, ipPrefix string, since time.Time) (service.FingerprintMatch, error) {
	var m service.FingerprintMatch
	if visitorID == "" && ipPrefix == "" {
		return m, nil
	}
	err := r.db.QueryRowContext(ctx, `
SELECT user_id, (visitor_id <> '' AND visitor_id = $1) FROM signup_fingerprints
WHERE trial_granted AND created_at >= $3
  AND ((visitor_id <> '' AND visitor_id = $1) OR (ip_prefix <> '' AND ip_prefix = $2))
ORDER BY created_at DESC
LIMIT 1`, visitorID, ipPrefix, since).Scan(&m.UserID, &m.ByVisitor)
	if errors.Is(err, sql.ErrNoRows) {
		return service.FingerprintMatch{}, nil
	}
	return m, err
}

func (r *signupFingerprintRepository) FindFlaggedMatch(ctx context.Context, visitorID, ipPrefix string, excludeUserID int64) (service.FingerprintMatch, error) {
	var m service.FingerprintMatch
	if visitorID == "" && ipPrefix == "" {
		return m, nil
	}
	err := r.db.QueryRowContext(ctx, `
SELECT f.user_id, (f.visitor_id <> '' AND f.visitor_id = $1),
       CASE WHEN u.status = 'disabled' THEN 'disabled' ELSE 'shadowed' END
FROM signup_fingerprints f
JOIN users u ON u.id = f.user_id
LEFT JOIN user_shadows s ON s.user_id = f.user_id
WHERE f.user_id <> $3
  AND (u.status = 'disabled' OR s.user_id IS NOT NULL)
  AND ((f.visitor_id <> '' AND f.visitor_id = $1) OR (f.ip_prefix <> '' AND f.ip_prefix = $2))
ORDER BY (u.status = 'disabled') DESC, f.created_at DESC
LIMIT 1`, visitorID, ipPrefix, excludeUserID).Scan(&m.UserID, &m.ByVisitor, &m.Status)
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
