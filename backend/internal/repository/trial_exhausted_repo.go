package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type trialExhaustedRepository struct {
	db *sql.DB
}

// NewTrialExhaustedRepository finds recent sign-ups who used up their trial credit without paying.
func NewTrialExhaustedRepository(db *sql.DB) service.TrialExhaustedRepository {
	return &trialExhaustedRepository{db: db}
}

func (r *trialExhaustedRepository) DueTrialUsers(ctx context.Context, signedUpAfter time.Time, maxBalance float64, limit int) ([]service.TrialUser, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, '')
FROM users u
WHERE u.role <> 'admin' AND u.status = 'active' AND u.deleted_at IS NULL
  AND u.created_at >= $1
  AND u.balance < $2
  AND EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = u.id)
  AND NOT EXISTS (
      SELECT 1 FROM payment_orders o
      WHERE o.user_id = u.id AND o.payment_type <> 'balance' AND o.paid_at IS NOT NULL)
  AND NOT EXISTS (
      SELECT 1 FROM user_subscriptions s
      WHERE s.user_id = u.id AND s.deleted_at IS NULL AND s.expires_at > NOW())
ORDER BY u.id
LIMIT $3`, signedUpAfter, maxBalance, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.TrialUser
	for rows.Next() {
		var u service.TrialUser
		if err := rows.Scan(&u.UserID, &u.Email, &u.Username); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
