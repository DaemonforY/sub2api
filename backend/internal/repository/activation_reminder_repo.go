package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type activationReminderRepository struct {
	db *sql.DB
}

// NewActivationReminderRepository finds sign-ups that haven't made a call and records reminders.
func NewActivationReminderRepository(db *sql.DB) service.ActivationReminderRepository {
	return &activationReminderRepository{db: db}
}

func (r *activationReminderRepository) DueUsers(ctx context.Context, from, to time.Time, limit int) ([]service.ActivationReminderUser, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''),
       COALESCE(t.trial_days, 0), COALESCE(t.group_name, ''), COALESCE(t.pending_expires_at, 'epoch'::timestamptz)
FROM users u
LEFT JOIN LATERAL (
    SELECT st.trial_days, st.pending_expires_at, COALESCE(g.name, '') AS group_name
    FROM signup_trial_starts st
    JOIN user_subscriptions s ON s.id = st.subscription_id AND s.deleted_at IS NULL AND s.expires_at = st.pending_expires_at
    LEFT JOIN groups g ON g.id = s.group_id
    WHERE st.user_id = u.id AND st.closed_reason IS NULL
    ORDER BY st.created_at LIMIT 1
) t ON TRUE
WHERE u.role <> 'admin' AND u.status = 'active' AND u.deleted_at IS NULL
  AND u.created_at >= $1 AND u.created_at < $2
  AND NOT EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = u.id)
  AND NOT EXISTS (SELECT 1 FROM activation_reminders a WHERE a.user_id = u.id)
ORDER BY u.created_at LIMIT $3`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.ActivationReminderUser
	for rows.Next() {
		var u service.ActivationReminderUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.TrialDays, &u.TrialGroup, &u.TrialUntil); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *activationReminderRepository) MarkSent(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO activation_reminders (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, userID)
	return err
}

func (r *activationReminderRepository) SentCount(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activation_reminders`).Scan(&n)
	return n, err
}
