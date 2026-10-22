package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type signupTrialStartRepository struct {
	db *sql.DB
}

// NewSignupTrialStartRepository stores sign-up trials that start on their first call.
func NewSignupTrialStartRepository(db *sql.DB) service.SignupTrialStartRepository {
	return &signupTrialStartRepository{db: db}
}

// Create copies the expiry from the row, so the later equality checks never depend on how the
// caller's time.Time was rounded.
func (r *signupTrialStartRepository) Create(ctx context.Context, subscriptionID, userID int64, trialDays int) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO signup_trial_starts (subscription_id, user_id, trial_days, pending_expires_at)
SELECT s.id, $2, $3, s.expires_at FROM user_subscriptions s WHERE s.id = $1
ON CONFLICT (subscription_id) DO NOTHING`, subscriptionID, userID, trialDays)
	return err
}

// StartUsed: the expiry check sits in the UPDATE's WHERE, so a purchase that extends the
// subscription concurrently wins (PostgreSQL re-checks the condition on the updated row).
func (r *signupTrialStartRepository) StartUsed(ctx context.Context, limit int) ([]service.SignupTrialStarted, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH first_use AS (
    SELECT t.subscription_id, MIN(l.created_at) AS first_at
    FROM signup_trial_starts t
    JOIN usage_logs l ON l.subscription_id = t.subscription_id
    WHERE t.closed_reason IS NULL
    GROUP BY t.subscription_id
    ORDER BY MIN(l.created_at)
    LIMIT $1
), moved AS (
    UPDATE user_subscriptions s
    SET starts_at = f.first_at,
        expires_at = f.first_at + make_interval(days => t.trial_days),
        updated_at = NOW()
    FROM first_use f JOIN signup_trial_starts t ON t.subscription_id = f.subscription_id
    WHERE s.id = f.subscription_id AND s.deleted_at IS NULL AND s.expires_at = t.pending_expires_at
    RETURNING s.id, s.user_id, s.group_id, s.starts_at, s.expires_at
), closed AS (
    UPDATE signup_trial_starts t SET started_at = m.starts_at, closed_reason = 'started'
    FROM moved m WHERE t.subscription_id = m.id
)
SELECT id, user_id, group_id, starts_at, expires_at FROM moved`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.SignupTrialStarted
	for rows.Next() {
		var st service.SignupTrialStarted
		if err := rows.Scan(&st.SubscriptionID, &st.UserID, &st.GroupID, &st.StartsAt, &st.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (r *signupTrialStartRepository) CloseStale(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE signup_trial_starts t
SET closed_reason = CASE WHEN s.deleted_at IS NOT NULL OR s.expires_at <> t.pending_expires_at THEN 'changed' ELSE 'unused' END
FROM user_subscriptions s
WHERE s.id = t.subscription_id AND t.closed_reason IS NULL
  AND (s.deleted_at IS NOT NULL OR s.expires_at <> t.pending_expires_at
       OR (t.pending_expires_at <= $1 AND NOT EXISTS (SELECT 1 FROM usage_logs l WHERE l.subscription_id = t.subscription_id)))`, now)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *signupTrialStartRepository) PendingForUsers(ctx context.Context, userIDs []int64) (map[int64]service.PendingSignupTrial, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT t.subscription_id, t.trial_days, t.pending_expires_at
FROM signup_trial_starts t JOIN user_subscriptions s ON s.id = t.subscription_id
WHERE t.user_id = ANY($1) AND t.closed_reason IS NULL AND s.deleted_at IS NULL AND s.expires_at = t.pending_expires_at`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.PendingSignupTrial{}
	for rows.Next() {
		var p service.PendingSignupTrial
		if err := rows.Scan(&p.SubscriptionID, &p.TrialDays, &p.PendingExpiresAt); err != nil {
			return nil, err
		}
		out[p.SubscriptionID] = p
	}
	return out, rows.Err()
}
