package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userRiskRepository struct {
	db *sql.DB
}

// NewUserRiskRepository runs the risk monitor rules and the 用户画像 queries (read-only, admins excluded).
// Usage queries always filter usage_logs by created_at or user_id so they stay on indexes.
func NewUserRiskRepository(db *sql.DB) service.UserRiskRepository {
	return &userRiskRepository{db: db}
}

// realPaidCond: the order was paid through a payment provider (not with balance) and fulfilled.
const realPaidCond = `po.status = 'COMPLETED' AND po.payment_type <> 'balance' AND po.pay_amount > 0`

const signupTrialNote = `'%auto assigned by signup defaults%'`

func (r *userRiskRepository) hits(ctx context.Context, query string, args ...any) ([]service.RiskHit, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.RiskHit{}
	for rows.Next() {
		var h service.RiskHit
		if err := rows.Scan(&h.UserID, &h.Detail); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) LinkedFlaggedSignups(ctx context.Context, since time.Time) ([]service.RiskHit, error) {
	return r.hits(ctx, `
SELECT f.user_id,
       '注册指纹与这些账号相同：' || string_agg(DISTINCT '#' || o.user_id ||
           CASE WHEN ou.status = 'disabled' THEN '（已封禁）' ELSE '（静默）' END ||
           CASE WHEN f.visitor_id <> '' AND o.visitor_id = f.visitor_id THEN '同浏览器' ELSE '同 IP 段' END, '，')
FROM signup_fingerprints f
JOIN users u ON u.id = f.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
JOIN signup_fingerprints o ON o.user_id <> f.user_id
     AND ((f.visitor_id <> '' AND o.visitor_id = f.visitor_id) OR (f.ip_prefix <> '' AND o.ip_prefix = f.ip_prefix))
JOIN users ou ON ou.id = o.user_id
LEFT JOIN user_shadows os ON os.user_id = o.user_id
WHERE f.created_at >= $1 AND (ou.status = 'disabled' OR os.user_id IS NOT NULL)
GROUP BY f.user_id`, since)
}

func (r *userRiskRepository) TrialDay1Exhausted(ctx context.Context, since time.Time) ([]service.RiskHit, error) {
	return r.hits(ctx, `
SELECT s.user_id,
       format('试用「%s」开通当天已用 $%s / $%s', g.name, round(s.daily_usage_usd, 2), round(g.daily_limit_usd::numeric, 2))
FROM user_subscriptions s
JOIN groups g ON g.id = s.group_id
JOIN users u ON u.id = s.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
WHERE s.deleted_at IS NULL AND s.notes LIKE `+signupTrialNote+`
  AND s.created_at >= $1 AND g.daily_limit_usd > 0 AND s.daily_usage_usd >= g.daily_limit_usd
  AND NOT EXISTS (SELECT 1 FROM user_shadows sh WHERE sh.user_id = s.user_id)`, since)
}

func (r *userRiskRepository) RelayFreeSpend(ctx context.Context, since time.Time, uaPattern string) ([]service.RiskHit, error) {
	return r.hits(ctx, `
SELECT l.user_id,
       format('中转工具 %s 次请求，用了 $%s 免费额度，UA：%s', COUNT(*), round(SUM(l.actual_cost), 2), left(MAX(l.user_agent), 120))
FROM usage_logs l
JOIN users u ON u.id = l.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
WHERE l.created_at >= $1 AND l.actual_cost > 0 AND l.user_agent ~* $2
  AND NOT EXISTS (SELECT 1 FROM payment_orders po WHERE po.user_id = l.user_id AND `+realPaidCond+`)
GROUP BY l.user_id`, since, uaPattern)
}

func (r *userRiskRepository) BurstSignups(ctx context.Context, since time.Time, threshold int) ([]service.RiskHit, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH recent AS (
    SELECT f.user_id, f.visitor_id, f.ip_prefix
    FROM signup_fingerprints f
    JOIN users u ON u.id = f.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
    WHERE f.created_at >= $1
)
SELECT 'visitor', visitor_id, array_agg(user_id ORDER BY user_id) FROM recent
WHERE visitor_id <> '' GROUP BY visitor_id HAVING COUNT(*) >= $2
UNION ALL
SELECT 'ip_prefix', ip_prefix, array_agg(user_id ORDER BY user_id) FROM recent
WHERE ip_prefix <> '' GROUP BY ip_prefix HAVING COUNT(*) >= $2`, since, threshold)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	seen := map[int64]bool{}
	out := []service.RiskHit{}
	for rows.Next() {
		var kind, key string
		var ids pq.Int64Array
		if err := rows.Scan(&kind, &key, &ids); err != nil {
			return nil, err
		}
		way := "同浏览器"
		if kind == "ip_prefix" {
			way = "同 IP 段 " + key
		}
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			names = append(names, "#"+strconv.FormatInt(id, 10))
		}
		detail := fmt.Sprintf("24 小时内%s注册了 %d 个账号：%s", way, len(ids), strings.Join(names, " "))
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				out = append(out, service.RiskHit{UserID: id, Detail: detail})
			}
		}
	}
	return out, rows.Err()
}

func (r *userRiskRepository) InviteeSharesInviterFingerprint(ctx context.Context, since time.Time) ([]service.RiskHit, error) {
	return r.hits(ctx, `
SELECT a.user_id,
       '与邀请人 #' || a.inviter_id ||
       CASE WHEN fi.visitor_id <> '' AND fi.visitor_id = fp.visitor_id THEN ' 同浏览器' ELSE ' 同 IP 段' END
FROM user_affiliates a
JOIN signup_fingerprints fi ON fi.user_id = a.user_id
JOIN signup_fingerprints fp ON fp.user_id = a.inviter_id
JOIN users u ON u.id = a.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
WHERE a.inviter_id IS NOT NULL AND fi.created_at >= $1
  AND ((fi.visitor_id <> '' AND fi.visitor_id = fp.visitor_id) OR (fi.ip_prefix <> '' AND fi.ip_prefix = fp.ip_prefix))`, since)
}

func (r *userRiskRepository) InsertRiskEvents(ctx context.Context, rule string, hits []service.RiskHit) ([]service.RiskEvent, error) {
	out := []service.RiskEvent{}
	for _, h := range hits {
		detail := h.Detail
		if len([]rune(detail)) > 500 {
			detail = string([]rune(detail)[:500])
		}
		e := service.RiskEvent{UserID: h.UserID, Rule: rule, Detail: detail}
		err := r.db.QueryRowContext(ctx, `
INSERT INTO risk_events (user_id, rule, detail) VALUES ($1, $2, $3)
ON CONFLICT (user_id, rule) DO NOTHING
RETURNING id, created_at`, h.UserID, rule, detail).Scan(&e.ID, &e.CreatedAt)
		if errors.Is(err, sql.ErrNoRows) {
			continue // already reported
		}
		if err != nil {
			return out, err
		}
		_ = r.db.QueryRowContext(ctx, `SELECT email, COALESCE(username, '') FROM users WHERE id = $1`, h.UserID).Scan(&e.UserEmail, &e.Username)
		out = append(out, e)
	}
	return out, nil
}

func (r *userRiskRepository) ListRiskEvents(ctx context.Context, userID int64, page, pageSize int) ([]service.RiskEvent, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM risk_events WHERE ($1::bigint = 0 OR user_id = $1)`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.user_id, COALESCE(u.email, ''), COALESCE(u.username, ''), e.rule, e.detail, e.created_at
FROM risk_events e LEFT JOIN users u ON u.id = e.user_id
WHERE ($1::bigint = 0 OR e.user_id = $1)
ORDER BY e.created_at DESC, e.id DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.RiskEvent{}
	for rows.Next() {
		var e service.RiskEvent
		if err := rows.Scan(&e.ID, &e.UserID, &e.UserEmail, &e.Username, &e.Rule, &e.Detail, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// ---------- dashboard ----------

func (r *userRiskRepository) InsightsSegments(ctx context.Context, since time.Time) (service.InsightsSegments, []service.InsightsSignupDay, error) {
	var total service.InsightsSegments
	rows, err := r.db.QueryContext(ctx, `
WITH base AS (
    SELECT u.id, u.created_at,
           (SELECT MAX(po.created_at) FROM payment_orders po WHERE po.user_id = u.id AND `+realPaidCond+`) AS last_paid,
           (SELECT l.created_at FROM usage_logs l WHERE l.user_id = u.id ORDER BY l.created_at DESC LIMIT 1) AS last_used
    FROM users u
    WHERE u.role <> 'admin' AND u.deleted_at IS NULL
)
SELECT CASE WHEN last_paid IS NULL AND last_used IS NULL THEN 'inactive'
            WHEN last_paid IS NULL THEN 'trial'
            WHEN last_used >= NOW() - INTERVAL '14 days' OR last_paid >= NOW() - INTERVAL '30 days' THEN 'paying'
            ELSE 'churned' END,
       created_at >= $1,
       to_char((created_at AT TIME ZONE 'Asia/Shanghai')::date, 'YYYY-MM-DD'),
       COUNT(*)
FROM base
GROUP BY 1, 2, 3`, since)
	if err != nil {
		return total, nil, err
	}
	defer func() { _ = rows.Close() }()
	byDay := map[string]*service.InsightsSignupDay{}
	add := func(s *service.InsightsSegments, seg string, n int64) {
		switch seg {
		case "inactive":
			s.Inactive += n
		case "trial":
			s.Trial += n
		case "paying":
			s.Paying += n
		default:
			s.Churned += n
		}
	}
	for rows.Next() {
		var seg, day string
		var inWindow bool
		var n int64
		if err := rows.Scan(&seg, &inWindow, &day, &n); err != nil {
			return total, nil, err
		}
		add(&total, seg, n)
		if inWindow {
			d := byDay[day]
			if d == nil {
				d = &service.InsightsSignupDay{Date: day}
				byDay[day] = d
			}
			add(&d.InsightsSegments, seg, n)
		}
	}
	days := make([]service.InsightsSignupDay, 0, len(byDay))
	for _, d := range byDay {
		days = append(days, *d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	return total, days, rows.Err()
}

func (r *userRiskRepository) InsightsSpend(ctx context.Context, since time.Time) ([]service.InsightsSpendDay, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH payers AS (
    SELECT DISTINCT po.user_id FROM payment_orders po WHERE `+realPaidCond+`
), used AS (
    SELECT (l.created_at AT TIME ZONE 'Asia/Shanghai')::date AS d,
           COALESCE(SUM(l.actual_cost) FILTER (WHERE p.user_id IS NULL), 0) AS free,
           COALESCE(SUM(l.actual_cost), 0) AS total
    FROM usage_logs l
    JOIN users u ON u.id = l.user_id AND u.role <> 'admin'
    LEFT JOIN payers p ON p.user_id = l.user_id
    WHERE l.created_at >= $1
    GROUP BY 1
), paid AS (
    SELECT (COALESCE(po.paid_at, po.created_at) AT TIME ZONE 'Asia/Shanghai')::date AS d, SUM(po.pay_amount) AS cny
    FROM payment_orders po
    JOIN users u ON u.id = po.user_id AND u.role <> 'admin'
    WHERE `+realPaidCond+` AND COALESCE(po.paid_at, po.created_at) >= $1
    GROUP BY 1
)
SELECT to_char(COALESCE(used.d, paid.d), 'YYYY-MM-DD'),
       COALESCE(used.free, 0)::double precision, COALESCE(used.total, 0)::double precision, COALESCE(paid.cny, 0)::double precision
FROM used FULL OUTER JOIN paid ON paid.d = used.d
ORDER BY 1`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsSpendDay{}
	for rows.Next() {
		var d service.InsightsSpendDay
		if err := rows.Scan(&d.Date, &d.FreeUSD, &d.UsedUSD, &d.PaidCNY); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) InsightsUserAgents(ctx context.Context, since time.Time) ([]service.InsightsUARow, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT l.user_id, left(COALESCE(l.user_agent, ''), 200), COUNT(*), COALESCE(SUM(l.actual_cost), 0)::double precision
FROM usage_logs l
JOIN users u ON u.id = l.user_id AND u.role <> 'admin'
WHERE l.created_at >= $1
GROUP BY 1, 2
ORDER BY 3 DESC
LIMIT 20000`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsUARow{}
	for rows.Next() {
		var row service.InsightsUARow
		if err := rows.Scan(&row.UserID, &row.UserAgent, &row.Requests, &row.CostUSD); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) InsightsClusters(ctx context.Context, since time.Time) ([]service.InsightsCluster, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH f AS (
    SELECT f.user_id, f.visitor_id, f.ip_prefix
    FROM signup_fingerprints f
    JOIN users u ON u.id = f.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
    WHERE f.created_at >= $1
)
(SELECT 'visitor', visitor_id, array_agg(user_id ORDER BY user_id) FROM f
 WHERE visitor_id <> '' GROUP BY visitor_id HAVING COUNT(*) >= 2 ORDER BY COUNT(*) DESC LIMIT 100)
UNION ALL
(SELECT 'ip_prefix', ip_prefix, array_agg(user_id ORDER BY user_id) FROM f
 WHERE ip_prefix <> '' GROUP BY ip_prefix HAVING COUNT(*) >= 2 ORDER BY COUNT(*) DESC LIMIT 100)`, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsCluster{}
	for rows.Next() {
		var c service.InsightsCluster
		var ids pq.Int64Array
		if err := rows.Scan(&c.Kind, &c.Key, &ids); err != nil {
			return nil, err
		}
		c.UserIDs = []int64(ids)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) UserRefs(ctx context.Context, ids []int64, since time.Time) (map[int64]service.InsightsUserRef, error) {
	out := map[int64]service.InsightsUserRef{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT u.id, u.email, COALESCE(u.username, ''), u.status, u.created_at,
       COALESCE((SELECT SUM(po.pay_amount) FROM payment_orders po WHERE po.user_id = u.id AND `+realPaidCond+`), 0)::double precision,
       COALESCE((SELECT SUM(l.actual_cost) FROM usage_logs l WHERE l.user_id = u.id AND l.created_at >= $2), 0)::double precision
FROM users u
WHERE u.id = ANY($1) AND u.role <> 'admin'`, pq.Array(ids), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var u service.InsightsUserRef
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.Status, &u.CreatedAt, &u.PaidCNY, &u.SpendUSD); err != nil {
			return nil, err
		}
		out[u.ID] = u
	}
	return out, rows.Err()
}

func (r *userRiskRepository) FindUserID(ctx context.Context, q string) (int64, error) {
	if q == "" {
		return 0, nil
	}
	var id int64
	var err error
	if n, convErr := strconv.ParseInt(strings.TrimPrefix(q, "#"), 10, 64); convErr == nil {
		err = r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL`, n).Scan(&id)
	} else {
		err = r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL`, q).Scan(&id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

func (r *userRiskRepository) counts(ctx context.Context, query string, args ...any) ([]service.InsightsCount, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsCount{}
	for rows.Next() {
		var c service.InsightsCount
		if err := rows.Scan(&c.Key, &c.Count, &c.CostUSD, &c.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) UserProfile(ctx context.Context, userID int64, since time.Time) (*service.UserInsightsProfile, error) {
	p := &service.UserInsightsProfile{}
	err := r.db.QueryRowContext(ctx, `
SELECT id, email, COALESCE(username, ''), status, created_at, role, COALESCE(signup_source, ''), balance::double precision
FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).
		Scan(&p.User.ID, &p.User.Email, &p.User.Username, &p.User.Status, &p.User.CreatedAt, &p.Role, &p.SignupSource, &p.Balance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var a service.InsightsAttribution
	err = r.db.QueryRowContext(ctx, `
SELECT source, medium, campaign, aff_code, referrer_host, landing_path FROM user_attributions WHERE user_id = $1`, userID).
		Scan(&a.Source, &a.Medium, &a.Campaign, &a.AffCode, &a.ReferrerHost, &a.LandingPath)
	switch {
	case err == nil:
		p.Attribution = &a
	case !errors.Is(err, sql.ErrNoRows):
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `SELECT visitor_id, ip_prefix, trial_granted FROM signup_fingerprints WHERE user_id = $1`, userID).
		Scan(&p.VisitorID, &p.SignupIPPrefix, &p.TrialGranted)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if p.WebPrefixes, err = r.counts(ctx, `
SELECT ip_prefix, COUNT(*), 0::double precision, MAX(created_at) FROM analytics_events
WHERE user_id = $1 AND ip_prefix <> '' GROUP BY 1 ORDER BY 4 DESC LIMIT 20`, userID); err != nil {
		return nil, err
	}
	if p.APIIPs, err = r.counts(ctx, `
SELECT ip_address, COUNT(*), COALESCE(SUM(actual_cost), 0)::double precision, MAX(created_at) FROM usage_logs
WHERE user_id = $1 AND created_at >= $2 AND COALESCE(ip_address, '') <> '' GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, userID, since); err != nil {
		return nil, err
	}
	clients, err := r.counts(ctx, `
SELECT left(COALESCE(user_agent, ''), 200), COUNT(*), COALESCE(SUM(actual_cost), 0)::double precision, MAX(created_at) FROM usage_logs
WHERE user_id = $1 AND created_at >= $2 GROUP BY 1 ORDER BY 2 DESC LIMIT 30`, userID, since)
	if err != nil {
		return nil, err
	}
	p.Clients = make([]service.InsightsClientUA, 0, len(clients))
	for _, c := range clients {
		p.Clients = append(p.Clients, service.InsightsClientUA{UserAgent: c.Key, Requests: c.Count, CostUSD: c.CostUSD, LastSeen: c.LastSeen})
	}
	if p.Groups, err = r.counts(ctx, `
SELECT COALESCE(g.name, '#' || l.group_id::text, '-'), COUNT(*), COALESCE(SUM(l.actual_cost), 0)::double precision, MAX(l.created_at)
FROM usage_logs l LEFT JOIN groups g ON g.id = l.group_id
WHERE l.user_id = $1 AND l.created_at >= $2 GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, userID, since); err != nil {
		return nil, err
	}
	if p.Models, err = r.counts(ctx, `
SELECT COALESCE(model, '-'), COUNT(*), COALESCE(SUM(actual_cost), 0)::double precision, MAX(created_at) FROM usage_logs
WHERE user_id = $1 AND created_at >= $2 GROUP BY 1 ORDER BY 2 DESC LIMIT 20`, userID, since); err != nil {
		return nil, err
	}

	err = r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(l.actual_cost), 0)::double precision,
       COALESCE(SUM(l.actual_cost) FILTER (WHERE s.id IS NOT NULL AND s.notes LIKE `+signupTrialNote+`), 0)::double precision,
       COALESCE(SUM(l.actual_cost) FILTER (WHERE s.id IS NOT NULL AND COALESCE(s.notes, '') NOT LIKE `+signupTrialNote+`), 0)::double precision,
       COALESCE(SUM(l.actual_cost) FILTER (WHERE l.subscription_id IS NULL), 0)::double precision
FROM usage_logs l LEFT JOIN user_subscriptions s ON s.id = l.subscription_id
WHERE l.user_id = $1 AND l.created_at >= $2`, userID, since).
		Scan(&p.Spend.TotalUSD, &p.Spend.FreeSubUSD, &p.Spend.PaidSubUSD, &p.Spend.BalanceUSD)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(po.pay_amount), 0)::double precision, COUNT(*) > 0 FROM payment_orders po WHERE po.user_id = $1 AND `+realPaidCond, userID).
		Scan(&p.Spend.PaidCNY, &p.Spend.HasRealPaying)
	if err != nil {
		return nil, err
	}
	p.User.PaidCNY = p.Spend.PaidCNY
	p.User.SpendUSD = p.Spend.TotalUSD

	if p.Payments, err = r.payments(ctx, userID); err != nil {
		return nil, err
	}
	if p.Subscriptions, err = r.subscriptions(ctx, userID); err != nil {
		return nil, err
	}
	if p.InviteUp, err = r.inviteChain(ctx, `
WITH RECURSIVE up(id, depth) AS (
    SELECT inviter_id, 1 FROM user_affiliates WHERE user_id = $1 AND inviter_id IS NOT NULL
    UNION ALL
    SELECT a.inviter_id, up.depth + 1 FROM user_affiliates a JOIN up ON a.user_id = up.id
    WHERE a.inviter_id IS NOT NULL AND up.depth < 5
)
SELECT id FROM up ORDER BY depth`, userID, since); err != nil {
		return nil, err
	}
	if p.InviteDown, err = r.inviteChain(ctx, `
SELECT user_id FROM user_affiliates WHERE inviter_id = $1 ORDER BY created_at DESC LIMIT 100`, userID, since); err != nil {
		return nil, err
	}
	if p.Linked, err = r.linked(ctx, p, since); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *userRiskRepository) payments(ctx context.Context, userID int64) ([]service.InsightsPayment, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, order_type, payment_type, amount::double precision, pay_amount::double precision, status, created_at, paid_at
FROM payment_orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsPayment{}
	for rows.Next() {
		var p service.InsightsPayment
		var paidAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.OrderType, &p.PaymentType, &p.Amount, &p.PayAmount, &p.Status, &p.CreatedAt, &paidAt); err != nil {
			return nil, err
		}
		if paidAt.Valid {
			t := paidAt.Time
			p.PaidAt = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) subscriptions(ctx context.Context, userID int64) ([]service.InsightsSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT s.id, COALESCE(g.name, ''), s.status, COALESCE(s.notes, '') LIKE `+signupTrialNote+`, s.starts_at, s.expires_at,
       s.daily_usage_usd::double precision, g.daily_limit_usd
FROM user_subscriptions s LEFT JOIN groups g ON g.id = s.group_id
WHERE s.user_id = $1 AND s.deleted_at IS NULL ORDER BY s.created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.InsightsSubscription{}
	for rows.Next() {
		var s service.InsightsSubscription
		var limit sql.NullFloat64
		if err := rows.Scan(&s.ID, &s.GroupName, &s.Status, &s.Free, &s.StartsAt, &s.ExpiresAt, &s.DailyUsage, &limit); err != nil {
			return nil, err
		}
		if limit.Valid {
			v := limit.Float64
			s.DailyLimit = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) ids(ctx context.Context, query string, args ...any) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *userRiskRepository) inviteChain(ctx context.Context, query string, userID int64, since time.Time) ([]service.InsightsUserRef, error) {
	ids, err := r.ids(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	refs, err := r.UserRefs(ctx, ids, since)
	if err != nil {
		return nil, err
	}
	out := make([]service.InsightsUserRef, 0, len(ids))
	for _, id := range ids {
		if ref, ok := refs[id]; ok {
			out = append(out, ref)
		}
	}
	return out, nil
}

// linked finds other accounts sharing the sign-up browser or /24, an API IP (public ones), or a relay
// User-Agent with this user.
func (r *userRiskRepository) linked(ctx context.Context, p *service.UserInsightsProfile, since time.Time) ([]service.InsightsLinked, error) {
	userID := p.User.ID
	via := map[int64]map[string]bool{}
	mark := func(ids []int64, how string) {
		for _, id := range ids {
			if id == userID {
				continue
			}
			if via[id] == nil {
				via[id] = map[string]bool{}
			}
			via[id][how] = true
		}
	}
	if p.VisitorID != "" {
		ids, err := r.ids(ctx, `SELECT user_id FROM signup_fingerprints WHERE visitor_id = $1 LIMIT 100`, p.VisitorID)
		if err != nil {
			return nil, err
		}
		mark(ids, "visitor")
	}
	if p.SignupIPPrefix != "" {
		ids, err := r.ids(ctx, `SELECT user_id FROM signup_fingerprints WHERE ip_prefix = $1 LIMIT 100`, p.SignupIPPrefix)
		if err != nil {
			return nil, err
		}
		mark(ids, "ip_prefix")
	}
	var apiIPs []string
	for _, c := range p.APIIPs {
		if parsed := net.ParseIP(c.Key); parsed != nil && !parsed.IsPrivate() && !parsed.IsLoopback() {
			apiIPs = append(apiIPs, c.Key)
		}
	}
	if len(apiIPs) > 0 {
		ids, err := r.ids(ctx, `
SELECT DISTINCT user_id FROM usage_logs WHERE ip_address = ANY($1) AND created_at >= $2 AND user_id <> $3 LIMIT 100`,
			pq.Array(apiIPs), since, userID)
		if err != nil {
			return nil, err
		}
		mark(ids, "api_ip")
	}
	var relayUAs []string
	for _, c := range p.Clients {
		if service.ClassifyClientUA(c.UserAgent) == service.ClientRelay {
			relayUAs = append(relayUAs, c.UserAgent)
		}
	}
	if len(relayUAs) > 0 {
		ids, err := r.ids(ctx, `
SELECT DISTINCT user_id FROM usage_logs WHERE created_at >= $2 AND left(COALESCE(user_agent, ''), 200) = ANY($1) AND user_id <> $3 LIMIT 100`,
			pq.Array(relayUAs), time.Now().AddDate(0, 0, -30), userID)
		if err != nil {
			return nil, err
		}
		mark(ids, "user_agent")
	}
	ids := make([]int64, 0, len(via))
	for id := range via {
		ids = append(ids, id)
	}
	refs, err := r.UserRefs(ctx, ids, since)
	if err != nil {
		return nil, err
	}
	out := make([]service.InsightsLinked, 0, len(refs))
	for id, ref := range refs {
		hows := make([]string, 0, len(via[id]))
		for _, how := range []string{"visitor", "ip_prefix", "api_ip", "user_agent"} {
			if via[id][how] {
				hows = append(hows, how)
			}
		}
		out = append(out, service.InsightsLinked{User: ref, Via: hows})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Via) != len(out[j].Via) {
			return len(out[i].Via) > len(out[j].Via)
		}
		return out[i].User.ID > out[j].User.ID
	})
	return out, nil
}
