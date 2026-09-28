package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// gatewayRequestLogRepository stores every API gateway request (raw SQL, append-only).
type gatewayRequestLogRepository struct {
	db *sql.DB
}

// NewGatewayRequestLogRepository creates the repository.
func NewGatewayRequestLogRepository(db *sql.DB) service.GatewayRequestLogRepository {
	return &gatewayRequestLogRepository{db: db}
}

func (r *gatewayRequestLogRepository) BatchInsert(ctx context.Context, logs []*service.GatewayRequestLog) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil gateway request log repository")
	}
	if len(logs) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"gateway_request_logs",
		"created_at", "request_id", "method", "url", "path", "status_code", "success", "error_code",
		"duration_ms", "api_key", "api_key_id", "user_id", "group_id", "account_id", "model",
		"client_ip", "user_agent",
	))
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	var inserted int64
	for _, l := range logs {
		if l == nil {
			continue
		}
		createdAt := l.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx,
			createdAt.UTC(),
			truncateString(l.RequestID, 64),
			truncateString(l.Method, 16),
			truncateString(l.URL, 4096),
			truncateString(l.Path, 512),
			l.StatusCode,
			l.Success,
			truncateString(l.ErrorCode, 64),
			l.DurationMs,
			truncateString(l.APIKey, 512),
			nullInt64Ptr(l.APIKeyID),
			nullInt64Ptr(l.UserID),
			nullInt64Ptr(l.GroupID),
			nullInt64Ptr(l.AccountID),
			truncateString(l.Model, 128),
			truncateString(l.ClientIP, 64),
			truncateString(l.UserAgent, 512),
		); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return inserted, err
		}
		inserted++
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		_ = tx.Rollback()
		return inserted, err
	}
	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return inserted, err
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func buildGatewayRequestLogsWhere(filter *service.GatewayRequestLogFilter) (string, []any) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, strings.ReplaceAll(clause, "?", "$"+itoa(len(args))))
	}

	if filter.StartTime != nil {
		add("l.created_at >= ?", filter.StartTime.UTC())
	}
	if filter.EndTime != nil {
		add("l.created_at <= ?", filter.EndTime.UTC())
	}
	if filter.UserID != nil {
		add("l.user_id = ?", *filter.UserID)
	}
	if filter.APIKeyID != nil {
		add("l.api_key_id = ?", *filter.APIKeyID)
	}
	if filter.Success != nil {
		add("l.success = ?", *filter.Success)
	}
	if filter.StatusCode != nil {
		add("l.status_code = ?", *filter.StatusCode)
	}
	if m := strings.TrimSpace(filter.Method); m != "" {
		add("l.method = ?", strings.ToUpper(m))
	}
	if m := strings.TrimSpace(filter.Model); m != "" {
		add("l.model = ?", m)
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + escapeLikePattern(q) + "%"
		args = append(args, like)
		n := "$" + itoa(len(args))
		clauses = append(clauses, "(l.url ILIKE "+n+" OR l.api_key ILIKE "+n+" OR l.model ILIKE "+n+
			" OR l.client_ip ILIKE "+n+" OR u.email ILIKE "+n+" OR k.name ILIKE "+n+")")
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

const gatewayRequestLogsFrom = `
FROM gateway_request_logs l
LEFT JOIN users u ON u.id = l.user_id
LEFT JOIN api_keys k ON k.id = l.api_key_id
`

func (r *gatewayRequestLogRepository) List(ctx context.Context, filter *service.GatewayRequestLogFilter) (*service.GatewayRequestLogList, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil gateway request log repository")
	}
	if filter == nil {
		filter = &service.GatewayRequestLogFilter{}
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	where, args := buildGatewayRequestLogsWhere(filter)
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+gatewayRequestLogsFrom+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	query := `SELECT l.id, l.created_at, l.request_id, l.method, l.url, l.path, l.status_code, l.success,
l.error_code, l.duration_ms, l.api_key, l.api_key_id, l.user_id, l.group_id, l.account_id, l.model,
l.client_ip, l.user_agent, COALESCE(u.email, ''), COALESCE(k.name, '')` + gatewayRequestLogsFrom + where + `
ORDER BY l.created_at DESC, l.id DESC
LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	rows, err := r.db.QueryContext(ctx, query, append(args, pageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	logs := make([]*service.GatewayRequestLog, 0, pageSize)
	for rows.Next() {
		var (
			item                                 service.GatewayRequestLog
			apiKeyID, userID, groupID, accountID sql.NullInt64
		)
		if err := rows.Scan(
			&item.ID, &item.CreatedAt, &item.RequestID, &item.Method, &item.URL, &item.Path,
			&item.StatusCode, &item.Success, &item.ErrorCode, &item.DurationMs, &item.APIKey,
			&apiKeyID, &userID, &groupID, &accountID, &item.Model, &item.ClientIP, &item.UserAgent,
			&item.UserEmail, &item.APIKeyName,
		); err != nil {
			return nil, err
		}
		item.APIKeyID = nullInt64ToPtr(apiKeyID)
		item.UserID = nullInt64ToPtr(userID)
		item.GroupID = nullInt64ToPtr(groupID)
		item.AccountID = nullInt64ToPtr(accountID)
		logs = append(logs, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.GatewayRequestLogList{Logs: logs, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *gatewayRequestLogRepository) DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil gateway request log repository")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}
	res, err := r.db.ExecContext(ctx, `
WITH batch AS (
  SELECT id FROM gateway_request_logs WHERE created_at < $1 ORDER BY id LIMIT $2
)
DELETE FROM gateway_request_logs WHERE id IN (SELECT id FROM batch)`, cutoff.UTC(), batchSize)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func nullInt64ToPtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}
