package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type analyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository stores browser events and sign-up attribution, and builds the admin
// dashboard from them plus users, usage_logs, api_keys and payment_orders (raw SQL).
func NewAnalyticsRepository(db *sql.DB) service.AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) InsertEvents(ctx context.Context, rows []service.AnalyticsEventRow) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO analytics_events (event, app, path, visitor_id, session_id, user_id, source, medium, campaign,
  referrer_host, device, ip_prefix, props)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, 0), $7, $8, $9, $10, $11, $12, $13)`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for _, e := range rows {
		var props any
		if len(e.Props) > 0 {
			b, err := json.Marshal(e.Props)
			if err != nil {
				return err
			}
			props = string(b)
		}
		if _, err := stmt.ExecContext(ctx, e.Event, e.App, e.Path, e.VisitorID, e.SessionID, e.UserID, e.Source, e.Medium,
			e.Campaign, e.ReferrerHost, e.Device, e.IPPrefix, props); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *analyticsRepository) SaveAttribution(ctx context.Context, a service.UserAttribution) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_attributions (user_id, visitor_id, source, medium, campaign, aff_code, referrer_host, landing_path, first_seen_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO NOTHING`,
		a.UserID, a.VisitorID, a.Source, a.Medium, a.Campaign, a.AffCode, a.ReferrerHost, a.LandingPath, a.FirstSeenAt)
	if err != nil {
		return err
	}
	// Tie the visitor's earlier anonymous events to the new account.
	if a.VisitorID != "" {
		_, err = r.db.ExecContext(ctx, `UPDATE analytics_events SET user_id = $1 WHERE visitor_id = $2 AND user_id IS NULL`, a.UserID, a.VisitorID)
	}
	return err
}

func (r *analyticsRepository) SaveLateAttribution(ctx context.Context, a service.UserAttribution, createdAfter time.Time) error {
	res, err := r.db.ExecContext(ctx, `
INSERT INTO user_attributions (user_id, visitor_id, source, medium, campaign, aff_code, referrer_host, landing_path, first_seen_at)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
WHERE EXISTS (SELECT 1 FROM users WHERE id = $1 AND created_at >= $10)
ON CONFLICT (user_id) DO NOTHING`,
		a.UserID, a.VisitorID, a.Source, a.Medium, a.Campaign, a.AffCode, a.ReferrerHost, a.LandingPath, a.FirstSeenAt, createdAfter)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 && a.VisitorID != "" {
		_, err = r.db.ExecContext(ctx, `UPDATE analytics_events SET user_id = $1 WHERE visitor_id = $2 AND user_id IS NULL`, a.UserID, a.VisitorID)
	}
	return err
}

func (r *analyticsRepository) DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM analytics_events WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Day boundaries are Beijing time; admins are left out of every figure.
const sqlAnalyticsActivity = `
act AS (
  SELECT e.user_id, (e.created_at AT TIME ZONE 'Asia/Shanghai')::date AS d
  FROM analytics_events e WHERE e.user_id IS NOT NULL AND e.created_at >= $1
  UNION
  SELECT l.user_id, (l.created_at AT TIME ZONE 'Asia/Shanghai')::date
  FROM usage_logs l WHERE l.created_at >= $1
),
members AS (SELECT id FROM users WHERE role <> 'admin'),
act_m AS (SELECT act.* FROM act JOIN members m ON m.id = act.user_id)`

const sqlAnalyticsSignups = `
su AS (
  SELECT u.id, u.created_at, (u.created_at AT TIME ZONE 'Asia/Shanghai')::date AS d0,
    EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = u.id AND l.created_at < u.created_at + INTERVAL '24 hours') AS activated
  FROM users u WHERE u.role <> 'admin' AND u.deleted_at IS NULL AND u.created_at >= $1
)`

// Orders that brought in money (not paid from balance), by members.
const sqlAnalyticsOrders = `
paid AS (
  SELECT o.user_id, o.amount, (COALESCE(o.paid_at, o.completed_at, o.created_at) AT TIME ZONE 'Asia/Shanghai')::date AS d
  FROM payment_orders o JOIN users u ON u.id = o.user_id AND u.role <> 'admin'
  WHERE o.status = 'COMPLETED' AND o.payment_type <> 'balance' AND COALESCE(o.paid_at, o.completed_at, o.created_at) >= $1
)`

func (r *analyticsRepository) Overview(ctx context.Context, since time.Time, days int) (*service.AnalyticsOverview, error) {
	out := &service.AnalyticsOverview{Since: since.Format("2006-01-02"), Days: days, Devices: map[string]int64{}}
	byDay := map[string]*service.AnalyticsDay{}
	for i := 0; i < days; i++ {
		d := since.AddDate(0, 0, i).Format("2006-01-02")
		byDay[d] = &service.AnalyticsDay{Day: d}
	}

	q := func(query string, scan func(*sql.Rows) error, args ...any) error {
		rows, err := r.db.QueryContext(ctx, query, append([]any{since}, args...)...)
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
	dayOf := func(d string) *service.AnalyticsDay {
		if v, ok := byDay[d]; ok {
			return v
		}
		return &service.AnalyticsDay{} // outside the window (clock skew): ignore
	}

	// Daily visitors and page views.
	if err := q(`
SELECT to_char((created_at AT TIME ZONE 'Asia/Shanghai')::date, 'YYYY-MM-DD'), COUNT(DISTINCT visitor_id),
  COUNT(*) FILTER (WHERE event = 'page_view')
FROM analytics_events WHERE created_at >= $1 GROUP BY 1`, func(rows *sql.Rows) error {
		var d string
		var v, pv int64
		if err := rows.Scan(&d, &v, &pv); err != nil {
			return err
		}
		dayOf(d).Visitors, dayOf(d).PageViews = v, pv
		return nil
	}); err != nil {
		return nil, err
	}

	// Daily active users (signed-in events or API calls) and API users.
	if err := q(`WITH `+sqlAnalyticsActivity+`,
api AS (SELECT DISTINCT l.user_id, (l.created_at AT TIME ZONE 'Asia/Shanghai')::date AS d FROM usage_logs l JOIN members m ON m.id = l.user_id WHERE l.created_at >= $1)
SELECT to_char(d, 'YYYY-MM-DD'), COUNT(DISTINCT user_id), (SELECT COUNT(*) FROM api WHERE api.d = act_m.d)
FROM act_m GROUP BY d`, func(rows *sql.Rows) error {
		var d string
		var a, api int64
		if err := rows.Scan(&d, &a, &api); err != nil {
			return err
		}
		dayOf(d).ActiveUsers, dayOf(d).APIUsers = a, api
		return nil
	}); err != nil {
		return nil, err
	}

	// Daily sign-ups and activations.
	if err := q(`WITH `+sqlAnalyticsSignups+`
SELECT to_char(d0, 'YYYY-MM-DD'), COUNT(*), COUNT(*) FILTER (WHERE activated) FROM su GROUP BY d0`, func(rows *sql.Rows) error {
		var d string
		var n, act int64
		if err := rows.Scan(&d, &n, &act); err != nil {
			return err
		}
		dayOf(d).Signups, dayOf(d).Activated = n, act
		return nil
	}); err != nil {
		return nil, err
	}

	// Daily orders and revenue.
	if err := q(`WITH `+sqlAnalyticsOrders+`
SELECT to_char(d, 'YYYY-MM-DD'), COUNT(*), COALESCE(SUM(amount), 0)::float8 FROM paid GROUP BY d`, func(rows *sql.Rows) error {
		var d string
		var n int64
		var rev float64
		if err := rows.Scan(&d, &n, &rev); err != nil {
			return err
		}
		dayOf(d).Orders, dayOf(d).Revenue = n, rev
		return nil
	}); err != nil {
		return nil, err
	}
	for i := 0; i < days; i++ {
		out.Daily = append(out.Daily, *byDay[since.AddDate(0, 0, i).Format("2006-01-02")])
	}

	// Totals.
	t := &out.Totals
	if err := r.db.QueryRowContext(ctx, `WITH `+sqlAnalyticsActivity+`, `+sqlAnalyticsSignups+`, `+sqlAnalyticsOrders+`
SELECT
  (SELECT COUNT(DISTINCT visitor_id) FROM analytics_events WHERE created_at >= $1),
  (SELECT COUNT(DISTINCT user_id) FROM act_m),
  (SELECT COUNT(DISTINCT user_id) FROM act_m WHERE d > (NOW() AT TIME ZONE 'Asia/Shanghai')::date - 7),
  (SELECT COUNT(*) FROM su), (SELECT COUNT(*) FROM su WHERE activated),
  (SELECT COUNT(DISTINCT user_id) FROM paid), (SELECT COALESCE(SUM(amount), 0)::float8 FROM paid)`, since).
		Scan(&t.Visitors, &t.ActiveUsers, &t.WAU, &t.Signups, &t.Activated, &t.PaidUsers, &t.Revenue); err != nil {
		return nil, err
	}

	// Funnel: visitors → opened sign-up → signed up → created a key → first API call → paid.
	var f [6]int64
	if err := r.db.QueryRowContext(ctx, `WITH `+sqlAnalyticsSignups+`
SELECT
  (SELECT COUNT(DISTINCT visitor_id) FROM analytics_events WHERE created_at >= $1),
  (SELECT COUNT(DISTINCT visitor_id) FROM analytics_events WHERE created_at >= $1 AND event = 'signup_view'),
  (SELECT COUNT(*) FROM su),
  (SELECT COUNT(*) FROM su WHERE EXISTS (SELECT 1 FROM api_keys k WHERE k.user_id = su.id)),
  (SELECT COUNT(*) FROM su WHERE EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = su.id)),
  (SELECT COUNT(*) FROM su WHERE EXISTS (SELECT 1 FROM payment_orders o WHERE o.user_id = su.id AND o.status = 'COMPLETED'))`, since).
		Scan(&f[0], &f[1], &f[2], &f[3], &f[4], &f[5]); err != nil {
		return nil, err
	}
	for i, step := range []string{"visitors", "signup_view", "signups", "key_created", "first_call", "paid"} {
		out.Funnel = append(out.Funnel, service.AnalyticsFunnelStep{Step: step, Count: f[i]})
	}

	// Channels: visitors from events; sign-ups, activation and payments by the user's first touch.
	ch := map[string]*service.AnalyticsChannel{}
	get := func(src string) *service.AnalyticsChannel {
		if ch[src] == nil {
			ch[src] = &service.AnalyticsChannel{Source: src}
		}
		return ch[src]
	}
	if err := q(`SELECT source, COUNT(DISTINCT visitor_id) FROM analytics_events WHERE created_at >= $1 GROUP BY source`, func(rows *sql.Rows) error {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			return err
		}
		get(s).Visitors = n
		return nil
	}); err != nil {
		return nil, err
	}
	if err := q(`WITH `+sqlAnalyticsSignups+`
SELECT COALESCE(a.source, 'unknown'), COUNT(*), COUNT(*) FILTER (WHERE su.activated)
FROM su LEFT JOIN user_attributions a ON a.user_id = su.id GROUP BY 1`, func(rows *sql.Rows) error {
		var s string
		var n, act int64
		if err := rows.Scan(&s, &n, &act); err != nil {
			return err
		}
		get(s).Signups, get(s).Activated = n, act
		return nil
	}); err != nil {
		return nil, err
	}
	if err := q(`WITH `+sqlAnalyticsOrders+`
SELECT COALESCE(a.source, 'unknown'), COUNT(DISTINCT paid.user_id), COALESCE(SUM(paid.amount), 0)::float8
FROM paid LEFT JOIN user_attributions a ON a.user_id = paid.user_id GROUP BY 1`, func(rows *sql.Rows) error {
		var s string
		var n int64
		var rev float64
		if err := rows.Scan(&s, &n, &rev); err != nil {
			return err
		}
		get(s).PaidUsers, get(s).Revenue = n, rev
		return nil
	}); err != nil {
		return nil, err
	}
	for _, c := range ch {
		out.Channels = append(out.Channels, *c)
	}
	sort.Slice(out.Channels, func(i, j int) bool {
		a, b := out.Channels[i], out.Channels[j]
		if a.Signups != b.Signups {
			return a.Signups > b.Signups
		}
		if a.Visitors != b.Visitors {
			return a.Visitors > b.Visitors
		}
		return a.Source < b.Source
	})

	// Top pages.
	if err := q(`
SELECT app, path, COUNT(*), COUNT(DISTINCT visitor_id) FROM analytics_events
WHERE created_at >= $1 AND event = 'page_view' GROUP BY app, path ORDER BY 3 DESC, 4 DESC LIMIT 30`, func(rows *sql.Rows) error {
		var p service.AnalyticsPage
		if err := rows.Scan(&p.App, &p.Path, &p.Views, &p.Visitors); err != nil {
			return err
		}
		out.Pages = append(out.Pages, p)
		return nil
	}); err != nil {
		return nil, err
	}

	// Feature events.
	if err := q(`
SELECT event, COUNT(*), COUNT(DISTINCT visitor_id), COUNT(DISTINCT user_id) FROM analytics_events
WHERE created_at >= $1 AND event <> 'page_view' GROUP BY event ORDER BY 2 DESC`, func(rows *sql.Rows) error {
		var f service.AnalyticsFeature
		if err := rows.Scan(&f.Event, &f.Count, &f.Visitors, &f.Users); err != nil {
			return err
		}
		out.Features = append(out.Features, f)
		return nil
	}); err != nil {
		return nil, err
	}

	// Weekly sign-up cohorts: did they come back?
	if err := q(`WITH `+sqlAnalyticsSignups+`,
act AS (
  SELECT user_id, (created_at AT TIME ZONE 'Asia/Shanghai')::date AS d FROM analytics_events WHERE user_id IN (SELECT id FROM su)
  UNION
  SELECT user_id, (created_at AT TIME ZONE 'Asia/Shanghai')::date FROM usage_logs WHERE user_id IN (SELECT id FROM su)
)
SELECT to_char(date_trunc('week', su.d0)::date, 'YYYY-MM-DD'), COUNT(*),
  COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM act WHERE act.user_id = su.id AND act.d = su.d0 + 1)),
  COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM act WHERE act.user_id = su.id AND act.d BETWEEN su.d0 + 2 AND su.d0 + 7)),
  COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM act WHERE act.user_id = su.id AND act.d BETWEEN su.d0 + 8 AND su.d0 + 30)),
  COUNT(*) FILTER (WHERE su.activated)
FROM su GROUP BY 1 ORDER BY 1`, func(rows *sql.Rows) error {
		var c service.AnalyticsCohort
		if err := rows.Scan(&c.Week, &c.Signups, &c.Day1, &c.Day2to7, &c.Day8to30, &c.Activated); err != nil {
			return err
		}
		out.Retention = append(out.Retention, c)
		return nil
	}); err != nil {
		return nil, err
	}

	// Devices (distinct visitors).
	if err := q(`SELECT device, COUNT(DISTINCT visitor_id) FROM analytics_events WHERE created_at >= $1 GROUP BY device`, func(rows *sql.Rows) error {
		var d string
		var n int64
		if err := rows.Scan(&d, &n); err != nil {
			return err
		}
		out.Devices[d] = n
		return nil
	}); err != nil {
		return nil, err
	}

	// Why sign-ups stop, and which home-page buttons send people to sign up.
	breakdown := func(event, prop string, dst *[]service.AnalyticsBreakdown) error {
		return q(`SELECT COALESCE(NULLIF(props->>$3::text, ''), '-'), COUNT(*), COUNT(DISTINCT visitor_id)
FROM analytics_events WHERE created_at >= $1 AND event = $2 GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, func(rows *sql.Rows) error {
			var b service.AnalyticsBreakdown
			if err := rows.Scan(&b.Key, &b.Count, &b.Visitors); err != nil {
				return err
			}
			*dst = append(*dst, b)
			return nil
		}, event, prop)
	}
	if err := breakdown("signup_error", "reason", &out.SignupErrors); err != nil {
		return nil, err
	}
	if err := breakdown("cta_click", "where", &out.CTAClicks); err != nil {
		return nil, err
	}
	return out, nil
}
