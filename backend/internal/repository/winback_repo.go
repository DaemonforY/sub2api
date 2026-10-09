package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type winbackRepository struct {
	db *sql.DB
}

// NewWinbackRepository finds users who used the API and then stopped, for the win-back email.
func NewWinbackRepository(db *sql.DB) service.WinbackRepository {
	return &winbackRepository{db: db}
}

func (r *winbackRepository) DueLapsedUsers(ctx context.Context, from, to time.Time, limit int) ([]service.LapsedUser, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''), u.balance::float8, last.at
FROM users u
JOIN LATERAL (SELECT MAX(l.created_at) AS at FROM usage_logs l WHERE l.user_id = u.id) last ON TRUE
WHERE u.role <> 'admin' AND u.status = 'active' AND u.deleted_at IS NULL
  AND last.at >= $1 AND last.at < $2
ORDER BY last.at DESC
LIMIT $3`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.LapsedUser
	for rows.Next() {
		var u service.LapsedUser
		if err := rows.Scan(&u.UserID, &u.Email, &u.Username, &u.Balance, &u.LastUsed); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
