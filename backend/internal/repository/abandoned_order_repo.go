package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type abandonedOrderRepository struct {
	db *sql.DB
}

// NewAbandonedOrderRepository finds expired unpaid orders for the 未付款订单提醒 email.
func NewAbandonedOrderRepository(db *sql.DB) service.AbandonedOrderRepository {
	return &abandonedOrderRepository{db: db}
}

func (r *abandonedOrderRepository) DueAbandonedOrders(ctx context.Context, from, to time.Time, limit int) ([]service.AbandonedOrder, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT ON (o.user_id) o.id, o.user_id, u.email, COALESCE(u.username, ''), o.order_type, o.pay_amount::float8
FROM payment_orders o
JOIN users u ON u.id = o.user_id AND u.deleted_at IS NULL AND u.status = 'active' AND u.role <> 'admin'
WHERE o.status = 'EXPIRED'
  AND o.order_type IN ('balance', 'subscription')
  AND o.payment_type <> 'balance'
  AND o.expires_at >= $1 AND o.expires_at < $2
  AND NOT EXISTS (
      SELECT 1 FROM payment_orders p
      WHERE p.user_id = o.user_id AND p.id <> o.id AND p.created_at > o.created_at
        AND p.status NOT IN ('EXPIRED', 'CANCELLED', 'FAILED')
  )
ORDER BY o.user_id, o.expires_at DESC
LIMIT $3`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.AbandonedOrder
	for rows.Next() {
		var o service.AbandonedOrder
		if err := rows.Scan(&o.OrderID, &o.UserID, &o.Email, &o.Username, &o.OrderType, &o.Amount); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
