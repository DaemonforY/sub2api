package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userShadowRepository struct {
	db *sql.DB
}

// NewUserShadowRepository stores 静默限制 flags and answers the free-credit gate's questions.
func NewUserShadowRepository(db *sql.DB) service.UserShadowRepository {
	return &userShadowRepository{db: db}
}

func (r *userShadowRepository) ListShadows(ctx context.Context) ([]service.UserShadow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, reason, source, related_user_id, created_by, created_at FROM user_shadows`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.UserShadow{}
	for rows.Next() {
		var v service.UserShadow
		var related, by sql.NullInt64
		if err := rows.Scan(&v.UserID, &v.Reason, &v.Source, &related, &by, &v.CreatedAt); err != nil {
			return nil, err
		}
		if related.Valid {
			v.RelatedUserID = &related.Int64
		}
		if by.Valid {
			v.CreatedBy = &by.Int64
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *userShadowRepository) UpsertShadow(ctx context.Context, v service.UserShadow) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_shadows (user_id, reason, source, related_user_id, created_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET reason = EXCLUDED.reason, source = EXCLUDED.source,
    related_user_id = EXCLUDED.related_user_id, created_by = EXCLUDED.created_by`,
		v.UserID, v.Reason, v.Source, v.RelatedUserID, v.CreatedBy)
	return err
}

func (r *userShadowRepository) DeleteShadow(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_shadows WHERE user_id = $1`, userID)
	return err
}

func (r *userShadowRepository) HasRealPayment(ctx context.Context, userID int64) (bool, error) {
	var paid bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (SELECT 1 FROM payment_orders
               WHERE user_id = $1 AND paid_at IS NOT NULL AND payment_type <> 'balance' AND pay_amount > 0)`, userID).Scan(&paid)
	return paid, err
}

func (r *userShadowRepository) SpentSince(ctx context.Context, userID int64, since time.Time) (float64, error) {
	var spent float64
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(actual_cost), 0)::double precision FROM usage_logs WHERE user_id = $1 AND created_at >= $2`,
		userID, since).Scan(&spent)
	return spent, err
}

func (r *userShadowRepository) NoteRiskEvent(ctx context.Context, userID int64, rule, detail string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO risk_events (user_id, rule, detail) VALUES ($1, $2, left($3, 500))
ON CONFLICT (user_id, rule) DO UPDATE SET detail = left(EXCLUDED.detail || '；' || risk_events.detail, 500), created_at = NOW()`,
		userID, rule, detail)
	return err
}

func (r *userShadowRepository) FillSubscriptionDailyUsage(ctx context.Context, subscriptionID int64, limit float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_subscriptions SET daily_usage_usd = GREATEST(daily_usage_usd, $2), updated_at = NOW() WHERE id = $1`,
		subscriptionID, limit)
	return err
}
