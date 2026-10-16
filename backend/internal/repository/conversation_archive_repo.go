package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type conversationArchiveRepository struct {
	db *sql.DB
}

// NewConversationArchiveRepository stores the 对话归档 index (turns + images), its settings row
// and its access-trail rows (raw SQL). The archived content itself lives in the bucket.
func NewConversationArchiveRepository(db *sql.DB) service.ConversationArchiveRepository {
	return &conversationArchiveRepository{db: db}
}

func (r *conversationArchiveRepository) GetConfigRaw(ctx context.Context) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, service.SettingKeyConversationArchiveConfig).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func (r *conversationArchiveRepository) SaveConfig(ctx context.Context, raw string, audit service.ContentLogAccess) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
		service.SettingKeyConversationArchiveConfig, raw); err != nil {
		return err
	}
	if err := insertContentLogAccess(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *conversationArchiveRepository) InsertAccess(ctx context.Context, a service.ContentLogAccess) error {
	return insertContentLogAccess(ctx, r.db, a)
}

func (r *conversationArchiveRepository) InsertTurn(ctx context.Context, t *service.ArchiveTurn, images []service.ArchiveImage) (int64, error) {
	if t == nil {
		return 0, errors.New("nil turn")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO conversation_archive_turns (user_id, api_key_id, group_id, session_key, request_id, created_at, model,
    inbound_endpoint, user_agent, ip_address, input_tokens, output_tokens, cache_read_tokens, object_key, bytes,
    new_item_count, reset, image_count, preview)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING id`,
		t.UserID, nullableInt64(t.APIKeyID), nullableInt64(t.GroupID), stripNUL(t.SessionKey), stripNUL(t.RequestID), t.CreatedAt,
		stripNUL(t.Model), stripNUL(t.InboundEndpoint), stripNUL(t.UserAgent), stripNUL(t.IPAddress), t.InputTokens, t.OutputTokens,
		t.CacheReadTokens, t.ObjectKey, t.Bytes, t.NewItemCount, t.Reset, t.ImageCount, stripNUL(t.Preview)).Scan(&id)
	if err != nil {
		return 0, err
	}
	for _, img := range images {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO conversation_archive_images (turn_id, user_id, object_key, mime, bytes, created_at)
VALUES ($1, $2, $3, $4, $5, $6)`, id, img.UserID, img.ObjectKey, img.Mime, img.Bytes, img.CreatedAt); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	t.ID = id
	return id, nil
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// ListSessions finds the page of sessions by last activity first (created_at range, optional
// user), then aggregates each page session over its (user_id, session_key, id) index.
func (r *conversationArchiveRepository) ListSessions(ctx context.Context, f service.ArchiveSessionFilter) ([]service.ArchiveSession, error) {
	conds := []string{"TRUE"}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.UserID > 0 {
		add("user_id = ?", f.UserID)
	}
	if f.From != nil {
		add("created_at >= ?", *f.From)
	}
	if f.To != nil {
		add("created_at < ?", *f.To)
	}
	having := ""
	if f.Before != nil {
		args = append(args, *f.Before)
		having = "HAVING MAX(created_at) < $" + strconv.Itoa(len(args))
	}
	args = append(args, f.Limit)
	limitArg := "$" + strconv.Itoa(len(args))
	query := `
WITH page AS (
    SELECT user_id, session_key, MAX(created_at) AS last_at
    FROM conversation_archive_turns
    WHERE ` + strings.Join(conds, " AND ") + `
    GROUP BY user_id, session_key
    ` + having + `
    ORDER BY last_at DESC, user_id DESC, session_key
    LIMIT ` + limitArg + `
)
SELECT p.user_id, p.session_key, p.last_at, COALESCE(u.email, ''),
       s.first_at, s.turns, s.models, s.endpoints, s.image_count, s.input_tokens, s.output_tokens,
       COALESCE((SELECT t.user_agent FROM conversation_archive_turns t
                 WHERE t.user_id = p.user_id AND t.session_key = p.session_key ORDER BY t.created_at DESC, t.id DESC LIMIT 1), ''),
       COALESCE((SELECT t.preview FROM conversation_archive_turns t
                 WHERE t.user_id = p.user_id AND t.session_key = p.session_key AND t.preview <> '' ORDER BY t.created_at, t.id LIMIT 1), '')
FROM page p
LEFT JOIN users u ON u.id = p.user_id
CROSS JOIN LATERAL (
    SELECT MIN(t.created_at) AS first_at, COUNT(*) AS turns,
           COALESCE(array_agg(DISTINCT t.model) FILTER (WHERE t.model <> ''), '{}') AS models,
           COALESCE(array_agg(DISTINCT t.inbound_endpoint) FILTER (WHERE t.inbound_endpoint <> ''), '{}') AS endpoints,
           COALESCE(SUM(t.image_count), 0) AS image_count,
           COALESCE(SUM(t.input_tokens), 0) AS input_tokens,
           COALESCE(SUM(t.output_tokens), 0) AS output_tokens
    FROM conversation_archive_turns t
    WHERE t.user_id = p.user_id AND t.session_key = p.session_key
) s
ORDER BY p.last_at DESC, p.user_id DESC, p.session_key`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArchiveSession{}
	for rows.Next() {
		var v service.ArchiveSession
		var models, endpoints pq.StringArray
		if err := rows.Scan(&v.UserID, &v.SessionKey, &v.LastAt, &v.Email, &v.FirstAt, &v.Turns, &models, &endpoints,
			&v.ImageCount, &v.InputTokens, &v.OutputTokens, &v.UserAgent, &v.Preview); err != nil {
			return nil, err
		}
		v.Models = []string(models)
		v.Endpoints = []string(endpoints)
		out = append(out, v)
	}
	return out, rows.Err()
}

const archiveTurnColumns = `t.id, t.user_id, t.api_key_id, t.group_id, t.session_key, t.request_id, t.created_at, t.model,
    t.inbound_endpoint, t.user_agent, t.ip_address, t.input_tokens, t.output_tokens, t.cache_read_tokens, t.object_key,
    t.bytes, t.new_item_count, t.reset, t.image_count, t.preview`

func scanArchiveTurn(scan func(dest ...any) error, extra ...any) (service.ArchiveTurn, error) {
	var t service.ArchiveTurn
	var apiKey, group sql.NullInt64
	dest := []any{&t.ID, &t.UserID, &apiKey, &group, &t.SessionKey, &t.RequestID, &t.CreatedAt, &t.Model,
		&t.InboundEndpoint, &t.UserAgent, &t.IPAddress, &t.InputTokens, &t.OutputTokens, &t.CacheReadTokens, &t.ObjectKey,
		&t.Bytes, &t.NewItemCount, &t.Reset, &t.ImageCount, &t.Preview}
	if err := scan(append(dest, extra...)...); err != nil {
		return t, err
	}
	t.APIKeyID = nullInt64ToPtr(apiKey)
	t.GroupID = nullInt64ToPtr(group)
	return t, nil
}

// ListTurns returns a session's turns newest first by (created_at, id). before_id is a cursor:
// when that row is in the session, the page starts strictly before it; otherwise (e.g. the UI's
// id + 1 after a jump from a usage row) it starts at the newest row with a smaller id, inclusive.
func (r *conversationArchiveRepository) ListTurns(ctx context.Context, userID int64, sessionKey string, beforeID int64, limit int) ([]service.ArchiveTurn, error) {
	query := `
SELECT ` + archiveTurnColumns + `
FROM conversation_archive_turns t
WHERE t.user_id = $1 AND t.session_key = $2
ORDER BY t.created_at DESC, t.id DESC
LIMIT $3`
	args := []any{userID, sessionKey, limit}
	if beforeID > 0 {
		query = `
WITH cur AS (
    SELECT c.created_at, c.id, c.id = $4 AS exact
    FROM conversation_archive_turns c
    WHERE c.user_id = $1 AND c.session_key = $2 AND c.id <= $4
    ORDER BY c.id DESC
    LIMIT 1
)
SELECT ` + archiveTurnColumns + `
FROM conversation_archive_turns t, cur
WHERE t.user_id = $1 AND t.session_key = $2
  AND ((cur.exact AND (t.created_at, t.id) < (cur.created_at, cur.id))
    OR (NOT cur.exact AND (t.created_at, t.id) <= (cur.created_at, cur.id)))
ORDER BY t.created_at DESC, t.id DESC
LIMIT $3`
		args = append(args, beforeID)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArchiveTurn{}
	for rows.Next() {
		t, err := scanArchiveTurn(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *conversationArchiveRepository) GetTurnByRequestID(ctx context.Context, requestID string) (*service.ArchiveTurn, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT `+archiveTurnColumns+`, COALESCE(u.email, '')
FROM conversation_archive_turns t
LEFT JOIN users u ON u.id = t.user_id
WHERE t.request_id = $1
ORDER BY t.id DESC
LIMIT 1`, requestID)
	var email string
	t, err := scanArchiveTurn(row.Scan, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Email = email
	return &t, nil
}

func (r *conversationArchiveRepository) ListTurnImages(ctx context.Context, turnIDs []int64) (map[int64][]service.ArchiveImage, error) {
	out := map[int64][]service.ArchiveImage{}
	if len(turnIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, turn_id, user_id, object_key, mime, bytes, created_at
FROM conversation_archive_images
WHERE turn_id = ANY($1)
ORDER BY turn_id, id`, pq.Array(turnIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var img service.ArchiveImage
		if err := rows.Scan(&img.ID, &img.TurnID, &img.UserID, &img.ObjectKey, &img.Mime, &img.Bytes, &img.CreatedAt); err != nil {
			return nil, err
		}
		out[img.TurnID] = append(out[img.TurnID], img)
	}
	return out, rows.Err()
}

func (r *conversationArchiveRepository) GetImage(ctx context.Context, id int64) (*service.ArchiveImage, error) {
	var img service.ArchiveImage
	err := r.db.QueryRowContext(ctx, `
SELECT id, turn_id, user_id, object_key, mime, bytes, created_at
FROM conversation_archive_images WHERE id = $1`, id).
		Scan(&img.ID, &img.TurnID, &img.UserID, &img.ObjectKey, &img.Mime, &img.Bytes, &img.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *conversationArchiveRepository) DailyStats(ctx context.Context, since time.Time, tz string) ([]service.ArchiveDayStats, error) {
	if strings.TrimSpace(tz) == "" {
		tz = "UTC"
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT to_char((created_at AT TIME ZONE $2)::date, 'YYYY-MM-DD') AS day,
       COUNT(*), COUNT(DISTINCT user_id), COALESCE(SUM(bytes), 0), COALESCE(SUM(image_count), 0)
FROM conversation_archive_turns
WHERE created_at >= $1
GROUP BY 1
ORDER BY 1`, since, tz)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArchiveDayStats{}
	for rows.Next() {
		var d service.ArchiveDayStats
		if err := rows.Scan(&d.Day, &d.Turns, &d.Users, &d.Bytes, &d.Images); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *conversationArchiveRepository) ListExpired(ctx context.Context, cutoff time.Time, limit int) ([]service.ArchiveExpiredTurn, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT t.id, t.object_key,
       COALESCE((SELECT array_agg(i.object_key) FROM conversation_archive_images i WHERE i.turn_id = t.id), '{}')
FROM conversation_archive_turns t
WHERE t.created_at < $1
ORDER BY t.created_at
LIMIT $2`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArchiveExpiredTurn{}
	for rows.Next() {
		var t service.ArchiveExpiredTurn
		var keys pq.StringArray
		if err := rows.Scan(&t.ID, &t.ObjectKey, &keys); err != nil {
			return nil, err
		}
		t.ImageKeys = []string(keys)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *conversationArchiveRepository) DeleteTurns(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM conversation_archive_turns WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *conversationArchiveRepository) ResolveUserID(ctx context.Context, q string) (int64, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return 0, nil
	}
	if id, err := strconv.ParseInt(q, 10, 64); err == nil && id > 0 {
		var found int64
		err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1`, id).Scan(&found)
		if err == nil {
			return found, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `
SELECT id FROM users
WHERE LOWER(email) = LOWER($1) OR (username <> '' AND LOWER(username) = LOWER($1))
ORDER BY (deleted_at IS NULL) DESC, id
LIMIT 1`, q).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return id, err
}

func (r *conversationArchiveRepository) QueuePurge(ctx context.Context, userID int64, prefixes []string) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO conversation_archive_purges (user_id, prefixes) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET
    prefixes = ARRAY(SELECT DISTINCT unnest(conversation_archive_purges.prefixes || EXCLUDED.prefixes)),
    updated_at = NOW()`, userID, pq.Array(prefixes))
	return err
}

func (r *conversationArchiveRepository) ListPurges(ctx context.Context, limit int) ([]service.ArchivePurge, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT p.user_id, p.prefixes, p.created_at, p.attempts, COALESCE(u.deleted_at IS NOT NULL, TRUE)
FROM conversation_archive_purges p
LEFT JOIN users u ON u.id = p.user_id
ORDER BY p.updated_at, p.user_id
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArchivePurge{}
	for rows.Next() {
		var p service.ArchivePurge
		var prefixes pq.StringArray
		if err := rows.Scan(&p.UserID, &prefixes, &p.CreatedAt, &p.Attempts, &p.UserDeleted); err != nil {
			return nil, err
		}
		p.Prefixes = []string(prefixes)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *conversationArchiveRepository) FinishPurge(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM conversation_archive_purges WHERE user_id = $1`, userID)
	return err
}

func (r *conversationArchiveRepository) FailPurge(ctx context.Context, userID int64, errText string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE conversation_archive_purges
SET attempts = attempts + 1, last_error = $2, updated_at = NOW()
WHERE user_id = $1`, userID, stripNUL(errText))
	return err
}
