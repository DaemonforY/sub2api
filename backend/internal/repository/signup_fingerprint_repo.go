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

func (r *signupFingerprintRepository) FindRecentTrial(ctx context.Context, visitorID, ipPrefix string, since time.Time) (int64, error) {
	if visitorID == "" && ipPrefix == "" {
		return 0, nil
	}
	var userID int64
	err := r.db.QueryRowContext(ctx, `
SELECT user_id FROM signup_fingerprints
WHERE trial_granted AND created_at >= $3
  AND ((visitor_id <> '' AND visitor_id = $1) OR (ip_prefix <> '' AND ip_prefix = $2))
ORDER BY created_at DESC
LIMIT 1`, visitorID, ipPrefix, since).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return userID, err
}

func (r *signupFingerprintRepository) Save(ctx context.Context, fp service.SignupFingerprint) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO signup_fingerprints (user_id, visitor_id, ip_prefix, trial_granted)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO NOTHING`, fp.UserID, fp.VisitorID, fp.IPPrefix, fp.TrialGranted)
	return err
}
