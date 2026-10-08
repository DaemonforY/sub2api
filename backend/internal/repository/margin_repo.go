package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type marginRepository struct {
	db *sql.DB
}

// NewMarginRepository reads usage, payments, plans and subscriptions for the 套餐毛利 dashboard
// (raw SQL, read-only). Admins are kept apart from members everywhere.
func NewMarginRepository(db *sql.DB) service.MarginRepository {
	return &marginRepository{db: db}
}

// When an order brought money in.
const sqlOrderPaidAt = `COALESCE(o.paid_at, o.completed_at, o.created_at)`

func (r *marginRepository) Load(ctx context.Context, since time.Time, topUsers int) (*service.MarginRaw, error) {
	out := &service.MarginRaw{}
	q := func(query string, scan func(*sql.Rows) error, args ...any) error {
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			if err := scan(rows); err != nil {
				return err
			}
		}
		return rows.Err()
	}

	// Usage by group over the period.
	if err := q(`
SELECT COALESCE(g.id, 0), COALESCE(g.name, ''), COALESCE(g.subscription_type = 'subscription', false),
  COUNT(DISTINCT l.user_id), COALESCE(SUM(l.total_cost), 0)::float8, COALESCE(SUM(l.actual_cost), 0)::float8
FROM usage_logs l
JOIN users u ON u.id = l.user_id AND u.role <> 'admin'
LEFT JOIN groups g ON g.id = l.group_id
WHERE l.created_at >= $1
GROUP BY 1, 2, 3 ORDER BY 5 DESC`, func(rows *sql.Rows) error {
		var g service.MarginGroupRow
		if err := rows.Scan(&g.GroupID, &g.Name, &g.Subscription, &g.Users, &g.UsageUSD, &g.BilledUSD); err != nil {
			return err
		}
		out.Groups = append(out.Groups, g)
		return nil
	}, since); err != nil {
		return nil, err
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(l.total_cost), 0)::float8 FROM usage_logs l JOIN users u ON u.id = l.user_id AND u.role = 'admin'
WHERE l.created_at >= $1`, since).Scan(&out.AdminUsageUSD); err != nil {
		return nil, err
	}

	// Money in through payment gateways.
	if err := q(`
SELECT o.order_type, COALESCE(SUM(o.amount), 0)::float8
FROM payment_orders o JOIN users u ON u.id = o.user_id AND u.role <> 'admin'
WHERE o.status = 'COMPLETED' AND o.payment_type <> 'balance' AND `+sqlOrderPaidAt+` >= $1
GROUP BY 1`, func(rows *sql.Rows) error {
		var typ string
		var amount float64
		if err := rows.Scan(&typ, &amount); err != nil {
			return err
		}
		switch typ {
		case "balance":
			out.RechargeRevenue += amount
		case "subscription":
			out.SubscriptionRevenue += amount
		default:
			out.OtherRevenue += amount
		}
		return nil
	}, since); err != nil {
		return nil, err
	}

	// Plans on sale.
	if err := q(`
SELECT p.id, p.name, p.group_id, g.name, p.price, p.validity_days, p.validity_unit,
  g.daily_limit_usd, g.weekly_limit_usd, g.monthly_limit_usd,
  (SELECT COUNT(*) FROM payment_orders o JOIN users u ON u.id = o.user_id AND u.role <> 'admin'
     WHERE o.plan_id = p.id AND o.status = 'COMPLETED' AND `+sqlOrderPaidAt+` >= $1),
  (SELECT COALESCE(SUM(o.amount), 0)::float8 FROM payment_orders o JOIN users u ON u.id = o.user_id AND u.role <> 'admin'
     WHERE o.plan_id = p.id AND o.status = 'COMPLETED' AND `+sqlOrderPaidAt+` >= $1)
FROM subscription_plans p JOIN groups g ON g.id = p.group_id
WHERE p.for_sale
ORDER BY p.sort_order, p.id`, func(rows *sql.Rows) error {
		var p service.MarginPlanRow
		var daily, weekly, monthly sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.Name, &p.GroupID, &p.GroupName, &p.Price, &p.ValidityDays, &p.ValidityUnit,
			&daily, &weekly, &monthly, &p.Sold, &p.Revenue); err != nil {
			return err
		}
		p.DailyLimitUSD, p.WeeklyLimitUSD, p.MonthlyLimitUSD = nullFloat64Ptr(daily), nullFloat64Ptr(weekly), nullFloat64Ptr(monthly)
		out.Plans = append(out.Plans, p)
		return nil
	}, since); err != nil {
		return nil, err
	}

	// Members' subscriptions active at some point in the period: their own usage, and the
	// subscription orders for the same group paid during the term (renewals included).
	if err := q(`
SELECT s.id, s.user_id, u.email, s.group_id, g.name, s.starts_at, s.expires_at,
  (SELECT COALESCE(SUM(o.amount), 0)::float8 FROM payment_orders o
     WHERE o.user_id = s.user_id AND o.order_type = 'subscription' AND o.status = 'COMPLETED'
       AND o.subscription_group_id = s.group_id
       AND `+sqlOrderPaidAt+` >= s.starts_at - INTERVAL '1 hour' AND `+sqlOrderPaidAt+` < s.expires_at),
  (SELECT COALESCE(SUM(l.total_cost), 0)::float8 FROM usage_logs l WHERE l.subscription_id = s.id),
  g.daily_limit_usd, g.weekly_limit_usd, g.monthly_limit_usd
FROM user_subscriptions s
JOIN users u ON u.id = s.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
JOIN groups g ON g.id = s.group_id
WHERE s.deleted_at IS NULL AND s.expires_at >= $1 AND s.starts_at <= NOW()`, func(rows *sql.Rows) error {
		var s service.MarginSubscriptionRow
		var daily, weekly, monthly sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.UserID, &s.Email, &s.GroupID, &s.GroupName, &s.StartsAt, &s.ExpiresAt,
			&s.Paid, &s.UsageUSD, &daily, &weekly, &monthly); err != nil {
			return err
		}
		s.DailyLimitUSD, s.WeeklyLimitUSD, s.MonthlyLimitUSD = nullFloat64Ptr(daily), nullFloat64Ptr(weekly), nullFloat64Ptr(monthly)
		out.Subscriptions = append(out.Subscriptions, s)
		return nil
	}, since); err != nil {
		return nil, err
	}

	// Members who cost the most over the period, with what they paid in it.
	if err := q(`
WITH x AS (
  SELECT l.user_id, SUM(l.total_cost) AS usage,
    SUM(CASE WHEN l.billing_type = 0 THEN l.actual_cost ELSE 0 END) AS billed
  FROM usage_logs l WHERE l.created_at >= $1 GROUP BY l.user_id
)
SELECT u.id, u.email, x.usage::float8, x.billed::float8,
  (SELECT COALESCE(SUM(o.amount), 0)::float8 FROM payment_orders o
     WHERE o.user_id = u.id AND o.status = 'COMPLETED' AND o.payment_type <> 'balance' AND `+sqlOrderPaidAt+` >= $1)
FROM x JOIN users u ON u.id = x.user_id AND u.role <> 'admin'
ORDER BY x.usage DESC, u.id LIMIT $2`, func(rows *sql.Rows) error {
		var m service.MarginUserRow
		if err := rows.Scan(&m.UserID, &m.Email, &m.UsageUSD, &m.BilledUSD, &m.Paid); err != nil {
			return err
		}
		out.Users = append(out.Users, m)
		return nil
	}, since, topUsers); err != nil {
		return nil, err
	}
	return out, nil
}
