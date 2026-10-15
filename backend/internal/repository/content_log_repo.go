package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type contentLogRepository struct {
	db *sql.DB
}

// NewContentLogRepository stores 请求内容日志: the target accounts, the captured requests and the
// admin access log (raw SQL).
func NewContentLogRepository(db *sql.DB) service.ContentLogRepository {
	return &contentLogRepository{db: db}
}

func (r *contentLogRepository) ListTargetIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id FROM content_log_targets`)
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

func (r *contentLogRepository) ListTargets(ctx context.Context) ([]service.ContentLogTarget, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT t.user_id, COALESCE(u.email, ''), COALESCE(u.username, ''), t.note, t.created_by,
       COALESCE(a.email, ''), t.created_at, COALESCE(s.cnt, 0), s.last_at
FROM content_log_targets t
LEFT JOIN users u ON u.id = t.user_id
LEFT JOIN users a ON a.id = t.created_by
LEFT JOIN LATERAL (
    SELECT COUNT(*) AS cnt, MAX(l.created_at) AS last_at
    FROM request_content_logs l WHERE l.user_id = t.user_id
) s ON TRUE
ORDER BY t.created_at DESC, t.user_id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentLogTarget{}
	for rows.Next() {
		var v service.ContentLogTarget
		var by sql.NullInt64
		var last sql.NullTime
		if err := rows.Scan(&v.UserID, &v.Email, &v.Username, &v.Note, &by, &v.CreatedByEmail, &v.CreatedAt, &v.LogCount, &last); err != nil {
			return nil, err
		}
		v.CreatedBy = nullInt64ToPtr(by)
		if last.Valid {
			t := last.Time
			v.LastLoggedAt = &t
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *contentLogRepository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`, userID).Scan(&ok)
	return ok, err
}

func (r *contentLogRepository) EnableTarget(ctx context.Context, userID int64, note string, audit service.ContentLogAccess) error {
	var by any
	if audit.AdminID > 0 {
		by = audit.AdminID
	}
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO content_log_targets (user_id, note, created_by) VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE SET note = EXCLUDED.note`, userID, note, by); err != nil {
			return err
		}
		return insertContentLogAccess(ctx, tx, audit)
	})
}

func (r *contentLogRepository) DisableTarget(ctx context.Context, userID int64, audit service.ContentLogAccess) error {
	return r.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM content_log_targets WHERE user_id = $1`, userID); err != nil {
			return err
		}
		return insertContentLogAccess(ctx, tx, audit)
	})
}

func (r *contentLogRepository) inTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertContentLogAccess(ctx context.Context, db sqlExecer, a service.ContentLogAccess) error {
	_, err := db.ExecContext(ctx, `
INSERT INTO content_log_access_logs (admin_id, action, target_user_id, detail, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6)`,
		a.AdminID, a.Action, a.TargetUserID, truncateString(a.Detail, 2000), truncateString(a.IPAddress, 64), truncateString(a.UserAgent, 512))
	return err
}

func (r *contentLogRepository) InsertAccess(ctx context.Context, a service.ContentLogAccess) error {
	return insertContentLogAccess(ctx, r.db, a)
}

func (r *contentLogRepository) ListAccess(ctx context.Context, targetUserID int64, limit int) ([]service.ContentLogAccess, error) {
	where := ""
	args := []any{limit}
	if targetUserID > 0 {
		where = "WHERE l.target_user_id = $2"
		args = append(args, targetUserID)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT l.id, l.admin_id, COALESCE(a.email, ''), l.action, l.target_user_id, COALESCE(u.email, ''),
       l.detail, l.ip_address, l.user_agent, l.created_at
FROM content_log_access_logs l
LEFT JOIN users a ON a.id = l.admin_id
LEFT JOIN users u ON u.id = l.target_user_id
`+where+`
ORDER BY l.id DESC
LIMIT $1`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentLogAccess{}
	for rows.Next() {
		var v service.ContentLogAccess
		if err := rows.Scan(&v.ID, &v.AdminID, &v.AdminEmail, &v.Action, &v.TargetUserID, &v.TargetEmail,
			&v.Detail, &v.IPAddress, &v.UserAgent, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *contentLogRepository) BatchInsertLogs(ctx context.Context, logs []*service.RequestContentLog) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil content log repository")
	}
	const cols = 16
	values := make([]string, 0, len(logs))
	args := make([]any, 0, len(logs)*cols)
	for _, l := range logs {
		if l == nil {
			continue
		}
		calls := l.ToolCalls
		if calls == nil {
			calls = []service.ContentToolCall{}
		}
		toolJSON, err := json.Marshal(calls)
		if err != nil {
			toolJSON = []byte("[]")
		}
		createdAt := l.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		n := len(args)
		ph := make([]string, cols)
		for i := range ph {
			ph[i] = "$" + itoa(n+i+1)
		}
		ph[11] += "::jsonb"
		values = append(values, "("+strings.Join(ph, ", ")+")")
		args = append(args,
			l.UserID, nullInt64Ptr(l.APIKeyID), nullInt64Ptr(l.GroupID), l.RequestID, l.SessionID, l.Model,
			l.InboundEndpoint, l.UserAgent, l.IPAddress, stripNUL(l.UserText), createdAt.UTC(),
			stripNUL(string(toolJSON)), stripNUL(l.ReplyHead), l.InputTokens, l.OutputTokens, l.CacheReadTokens,
		)
	}
	if len(values) == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
INSERT INTO request_content_logs (user_id, api_key_id, group_id, request_id, session_id, model,
    inbound_endpoint, user_agent, ip_address, user_text, created_at, tool_calls, reply_head,
    input_tokens, output_tokens, cache_read_tokens)
VALUES `+strings.Join(values, ", "), args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func stripNUL(s string) string {
	if strings.IndexByte(s, 0) < 0 {
		return s
	}
	return strings.ReplaceAll(s, "\x00", "")
}

func (r *contentLogRepository) ListLogs(ctx context.Context, f service.ContentLogFilter) ([]service.RequestContentLog, error) {
	clauses := []string{"user_id = $1"}
	args := []any{f.UserID}
	if f.SessionID != "" {
		args = append(args, f.SessionID)
		clauses = append(clauses, "session_id = $"+itoa(len(args)))
	}
	if f.BeforeID > 0 {
		args = append(args, f.BeforeID)
		clauses = append(clauses, "id < $"+itoa(len(args)))
	}
	args = append(args, f.Limit)
	rows, err := r.db.QueryContext(ctx, `
SELECT id, created_at, user_id, api_key_id, group_id, request_id, session_id, model, inbound_endpoint,
       user_agent, ip_address, user_text, tool_calls, reply_head, input_tokens, output_tokens, cache_read_tokens
FROM request_content_logs
WHERE `+strings.Join(clauses, " AND ")+`
ORDER BY id DESC
LIMIT $`+itoa(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.RequestContentLog{}
	for rows.Next() {
		var v service.RequestContentLog
		var apiKeyID, groupID sql.NullInt64
		var toolJSON []byte
		if err := rows.Scan(&v.ID, &v.CreatedAt, &v.UserID, &apiKeyID, &groupID, &v.RequestID, &v.SessionID, &v.Model,
			&v.InboundEndpoint, &v.UserAgent, &v.IPAddress, &v.UserText, &toolJSON, &v.ReplyHead,
			&v.InputTokens, &v.OutputTokens, &v.CacheReadTokens); err != nil {
			return nil, err
		}
		v.APIKeyID = nullInt64ToPtr(apiKeyID)
		v.GroupID = nullInt64ToPtr(groupID)
		v.ToolCalls = []service.ContentToolCall{}
		if len(toolJSON) > 0 {
			_ = json.Unmarshal(toolJSON, &v.ToolCalls)
			if v.ToolCalls == nil {
				v.ToolCalls = []service.ContentToolCall{}
			}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *contentLogRepository) DeleteLogsBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	return r.deleteBefore(ctx, "request_content_logs", cutoff, batchSize)
}

func (r *contentLogRepository) DeleteAccessBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	return r.deleteBefore(ctx, "content_log_access_logs", cutoff, batchSize)
}

// deleteBefore removes one batch of rows older than cutoff. table is one of two constants above.
func (r *contentLogRepository) deleteBefore(ctx context.Context, table string, cutoff time.Time, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 5000
	}
	res, err := r.db.ExecContext(ctx, `
WITH batch AS (SELECT id FROM `+table+` WHERE created_at < $1 ORDER BY id LIMIT $2)
DELETE FROM `+table+` WHERE id IN (SELECT id FROM batch)`, cutoff.UTC(), batchSize)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
