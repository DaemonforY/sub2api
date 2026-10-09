package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type opsAgentRepository struct {
	db *sql.DB
}

// NewOpsAgentRepository: the ops assistant's read-only queries (raw SQL over ops_error_logs,
// usage_logs, accounts and ops_alert_events).
func NewOpsAgentRepository(db *sql.DB) service.OpsAgentRepository {
	return &opsAgentRepository{db: db}
}

// opsAgentMessage: the upstream's message when there is one, else the gateway's.
const opsAgentMessage = `left(coalesce(nullif(e.upstream_error_message, ''), e.error_message, ''), 200)`

func (r *opsAgentRepository) ErrorSummary(ctx context.Context, start, end time.Time) (*service.OpsAgentErrorSummary, error) {
	out := &service.OpsAgentErrorSummary{Groups: []service.OpsAgentErrorGroup{}}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM usage_logs WHERE created_at >= $1 AND created_at < $2`, start, end).Scan(&out.Success); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, `
SELECT count(*), count(*) FILTER (WHERE is_business_limited)
FROM ops_error_logs WHERE created_at >= $1 AND created_at < $2`, start, end).Scan(&out.Errors, &out.BusinessLimited); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT coalesce(e.status_code, 0), e.upstream_status_code, e.error_phase, e.error_type, `+opsAgentMessage+`, e.is_business_limited,
       count(*), count(DISTINCT e.user_id),
       (array_agg(DISTINCT e.model) FILTER (WHERE coalesce(e.model, '') <> ''))[1:5],
       (array_agg(DISTINCT '#' || e.account_id || ' ' || coalesce(a.name, '')) FILTER (WHERE e.account_id IS NOT NULL))[1:5],
       min(e.created_at), max(e.created_at)
FROM ops_error_logs e LEFT JOIN accounts a ON a.id = e.account_id
WHERE e.created_at >= $1 AND e.created_at < $2
GROUP BY 1, 2, 3, 4, 5, 6
ORDER BY 7 DESC
LIMIT 15`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var g service.OpsAgentErrorGroup
		var upstream sql.NullInt64
		var models, accounts []string
		if err := rows.Scan(&g.StatusCode, &upstream, &g.Phase, &g.Type, &g.Message, &g.BusinessLimited, &g.Count, &g.Users,
			pq.Array(&models), pq.Array(&accounts), &g.First, &g.Last); err != nil {
			return nil, err
		}
		if upstream.Valid {
			v := int(upstream.Int64)
			g.UpstreamStatus = &v
		}
		g.Models, g.Accounts = nonNilStrings(models), nonNilStrings(accounts)
		out.Groups = append(out.Groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var errs error
	if out.ByAccount, errs = r.counts(ctx, start, end,
		`SELECT '#' || u.account_id || ' ' || coalesce(a.name, ''), count(*) FROM usage_logs u LEFT JOIN accounts a ON a.id = u.account_id
		 WHERE u.created_at >= $1 AND u.created_at < $2 GROUP BY 1`,
		`SELECT '#' || e.account_id || ' ' || coalesce(a.name, ''), count(*) FROM ops_error_logs e LEFT JOIN accounts a ON a.id = e.account_id
		 WHERE e.created_at >= $1 AND e.created_at < $2 AND e.account_id IS NOT NULL GROUP BY 1`, 10); errs != nil {
		return nil, errs
	}
	if out.ByModel, errs = r.counts(ctx, start, end,
		`SELECT model, count(*) FROM usage_logs WHERE created_at >= $1 AND created_at < $2 GROUP BY 1`,
		`SELECT model, count(*) FROM ops_error_logs WHERE created_at >= $1 AND created_at < $2 AND coalesce(model, '') <> '' GROUP BY 1`, 12); errs != nil {
		return nil, errs
	}
	if out.ByUser, errs = r.counts(ctx, start, end,
		`SELECT '#' || u.user_id || ' ' || coalesce(us.email, ''), count(*) FROM usage_logs u LEFT JOIN users us ON us.id = u.user_id
		 WHERE u.created_at >= $1 AND u.created_at < $2 GROUP BY 1`,
		`SELECT '#' || e.user_id || ' ' || coalesce(us.email, ''), count(*) FROM ops_error_logs e LEFT JOIN users us ON us.id = e.user_id
		 WHERE e.created_at >= $1 AND e.created_at < $2 AND e.user_id IS NOT NULL GROUP BY 1`, 8); errs != nil {
		return nil, errs
	}
	return out, nil
}

// counts merges a success query and an error query (key, count) and keeps the keys with the most
// errors first, then the busiest.
func (r *opsAgentRepository) counts(ctx context.Context, start, end time.Time, successSQL, errorSQL string, limit int) ([]service.OpsAgentCount, error) {
	byKey := map[string]*service.OpsAgentCount{}
	var order []string
	read := func(query string, errors bool) error {
		rows, err := r.db.QueryContext(ctx, query, start, end)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var key sql.NullString
			var n int
			if err := rows.Scan(&key, &n); err != nil {
				return err
			}
			k := strings.TrimSpace(key.String)
			if k == "" {
				k = "（无）"
			}
			c := byKey[k]
			if c == nil {
				c = &service.OpsAgentCount{Key: k}
				byKey[k] = c
				order = append(order, k)
			}
			if errors {
				c.Errors += n
			} else {
				c.Success += n
			}
		}
		return rows.Err()
	}
	if err := read(successSQL, false); err != nil {
		return nil, err
	}
	if err := read(errorSQL, true); err != nil {
		return nil, err
	}
	out := make([]service.OpsAgentCount, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sortOpsAgentCounts(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func sortOpsAgentCounts(list []service.OpsAgentCount) {
	less := func(a, b service.OpsAgentCount) bool {
		if a.Errors != b.Errors {
			return a.Errors > b.Errors
		}
		if a.Success != b.Success {
			return a.Success > b.Success
		}
		return a.Key < b.Key
	}
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && less(list[j], list[j-1]); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

func (r *opsAgentRepository) ListErrors(ctx context.Context, f service.OpsAgentErrorFilter) ([]service.OpsAgentErrorRow, error) {
	where := []string{"e.created_at >= $1", "e.created_at < $2"}
	args := []any{f.Start, f.End}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if m := strings.TrimSpace(f.Model); m != "" {
		add("e.model ILIKE $%d", "%"+m+"%")
	}
	if f.AccountID > 0 {
		add("e.account_id = $%d", f.AccountID)
	}
	if f.UserID > 0 {
		add("e.user_id = $%d", f.UserID)
	}
	if f.StatusCode > 0 {
		add("(e.status_code = $%[1]d OR e.upstream_status_code = $%[1]d)", f.StatusCode)
	}
	limit := f.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.created_at, e.user_id, coalesce(us.email, ''), coalesce(g.name, ''),
       CASE WHEN e.account_id IS NULL THEN '' ELSE '#' || e.account_id || ' ' || coalesce(a.name, '') END,
       coalesce(e.model, ''), coalesce(e.request_path, ''), e.stream, coalesce(e.status_code, 0), e.upstream_status_code,
       e.error_phase, e.error_type, `+opsAgentMessage+`, left(coalesce(e.upstream_error_detail, ''), 300),
       coalesce(e.upstream_errors::text, ''), e.duration_ms
FROM ops_error_logs e
LEFT JOIN users us ON us.id = e.user_id
LEFT JOIN groups g ON g.id = e.group_id
LEFT JOIN accounts a ON a.id = e.account_id
WHERE `+strings.Join(where, " AND ")+`
ORDER BY e.created_at DESC
LIMIT `+fmt.Sprint(limit), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.OpsAgentErrorRow{}
	for rows.Next() {
		var e service.OpsAgentErrorRow
		var userID, upstream, duration sql.NullInt64
		var chain string
		if err := rows.Scan(&e.ID, &e.Time, &userID, &e.UserEmail, &e.Group, &e.Account, &e.Model, &e.Path, &e.Stream, &e.StatusCode,
			&upstream, &e.Phase, &e.Type, &e.Message, &e.UpstreamDetail, &chain, &duration); err != nil {
			return nil, err
		}
		if userID.Valid {
			v := userID.Int64
			e.UserID = &v
		}
		if upstream.Valid {
			v := int(upstream.Int64)
			e.UpstreamStatus = &v
		}
		if duration.Valid {
			v := int(duration.Int64)
			e.DurationMs = &v
		}
		e.Tried = opsAgentTried(chain)
		out = append(out, e)
	}
	return out, rows.Err()
}

// opsAgentTried reads the accounts tried from upstream_errors ([{account_id, ...}, ...]).
func opsAgentTried(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var chain []struct {
		AccountID   *int64 `json:"account_id"`
		AccountName string `json:"account_name"`
	}
	if json.Unmarshal([]byte(raw), &chain) != nil {
		return nil
	}
	var out []string
	for _, c := range chain {
		if c.AccountID == nil {
			continue
		}
		out = append(out, strings.TrimSpace(fmt.Sprintf("#%d %s", *c.AccountID, c.AccountName)))
	}
	return out
}

func (r *opsAgentRepository) Timeline(ctx context.Context, start, end time.Time, bucket time.Duration) ([]service.OpsAgentTimelineBucket, error) {
	if bucket < time.Minute {
		bucket = time.Minute
	}
	secs := int64(bucket / time.Second)
	rows, err := r.db.QueryContext(ctx, `
WITH s AS (SELECT to_timestamp(floor(extract(epoch FROM created_at) / $3) * $3) b, count(*) n
           FROM usage_logs WHERE created_at >= $1 AND created_at < $2 GROUP BY 1),
     e AS (SELECT to_timestamp(floor(extract(epoch FROM created_at) / $3) * $3) b, count(*) n
           FROM ops_error_logs WHERE created_at >= $1 AND created_at < $2 GROUP BY 1)
SELECT coalesce(s.b, e.b), coalesce(s.n, 0), coalesce(e.n, 0)
FROM s FULL JOIN e ON s.b = e.b
ORDER BY 1`, start, end, secs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.OpsAgentTimelineBucket{}
	for rows.Next() {
		var b service.OpsAgentTimelineBucket
		if err := rows.Scan(&b.Start, &b.Success, &b.Errors); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *opsAgentRepository) Accounts(ctx context.Context) ([]service.OpsAgentAccount, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT a.id, a.name, a.platform, a.type, a.status, a.schedulable, a.concurrency, a.priority,
       coalesce(a.credentials->>'base_url', ''), coalesce(a.error_message, ''),
       a.rate_limited_at, a.rate_limit_reset_at, a.overload_until, a.temp_unschedulable_until,
       coalesce(a.temp_unschedulable_reason, ''), a.last_used_at,
       coalesce((SELECT array_agg(g.name ORDER BY g.id) FROM account_groups ag JOIN groups g ON g.id = ag.group_id
                 WHERE ag.account_id = a.id AND g.deleted_at IS NULL), '{}')
FROM accounts a
WHERE a.deleted_at IS NULL
ORDER BY a.id
LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.OpsAgentAccount{}
	for rows.Next() {
		var a service.OpsAgentAccount
		var base string
		var rl, rlReset, overload, temp, lastUsed sql.NullTime
		var groups []string
		if err := rows.Scan(&a.ID, &a.Name, &a.Platform, &a.Type, &a.Status, &a.Schedulable, &a.Concurrency, &a.Priority,
			&base, &a.ErrorMessage, &rl, &rlReset, &overload, &temp, &a.TempUnschedulableWhy, &lastUsed, pq.Array(&groups)); err != nil {
			return nil, err
		}
		if u, err := url.Parse(strings.TrimSpace(base)); err == nil && u.Host != "" {
			a.BaseHost = u.Host
		}
		a.RateLimitedAt, a.RateLimitResetAt, a.OverloadUntil = nullTimePtr(rl), nullTimePtr(rlReset), nullTimePtr(overload)
		a.TempUnschedulableUntil, a.LastUsedAt = nullTimePtr(temp), nullTimePtr(lastUsed)
		a.Groups = nonNilStrings(groups)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *opsAgentRepository) AlertEvents(ctx context.Context, since time.Time, limit int) ([]service.OpsAgentAlert, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, coalesce(title, ''), coalesce(description, ''), coalesce(severity, ''), coalesce(status, ''), fired_at, resolved_at
FROM ops_alert_events WHERE fired_at >= $1 ORDER BY fired_at DESC LIMIT $2`, since, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.OpsAgentAlert{}
	for rows.Next() {
		var a service.OpsAgentAlert
		var resolved sql.NullTime
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.Severity, &a.Status, &a.FiredAt, &resolved); err != nil {
			return nil, err
		}
		a.ResolvedAt = nullTimePtr(resolved)
		out = append(out, a)
	}
	return out, rows.Err()
}

func nullTimePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
