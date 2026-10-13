package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type contentEngineRepository struct {
	db *sql.DB
}

// NewContentEngineRepository creates the 内容引擎 store (raw SQL: content_owner_settings,
// content_topics, content_items, content_variants, content_rules).
func NewContentEngineRepository(db *sql.DB) service.ContentEngineRepository {
	return &contentEngineRepository{db: db}
}

// --- settings ----------------------------------------------------------------------------------

func (r *contentEngineRepository) GetSettings(ctx context.Context, owner int64) (*service.ContentOwnerRow, error) {
	row := service.ContentOwnerRow{OwnerUserID: owner}
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data, notify_key_enc, enabled, last_run_day FROM content_owner_settings WHERE owner_user_id = $1`, owner).
		Scan(&data, &row.NotifyKeyEnc, &row.Enabled, &row.LastRunDay)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &row.Settings); err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *contentEngineRepository) SaveSettings(ctx context.Context, row *service.ContentOwnerRow) error {
	data, err := json.Marshal(row.Settings)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO content_owner_settings (owner_user_id, data, notify_key_enc, enabled, updated_at)
VALUES ($1, $2::jsonb, $3, $4, NOW())
ON CONFLICT (owner_user_id) DO UPDATE SET data = EXCLUDED.data, notify_key_enc = EXCLUDED.notify_key_enc, enabled = EXCLUDED.enabled, updated_at = NOW()`,
		row.OwnerUserID, string(data), row.NotifyKeyEnc, row.Enabled)
	return err
}

func (r *contentEngineRepository) ListEnabledOwners(ctx context.Context) ([]service.ContentOwnerRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT owner_user_id, data, notify_key_enc, enabled, last_run_day FROM content_owner_settings WHERE enabled`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentOwnerRow{}
	for rows.Next() {
		var row service.ContentOwnerRow
		var data []byte
		if err := rows.Scan(&row.OwnerUserID, &data, &row.NotifyKeyEnc, &row.Enabled, &row.LastRunDay); err != nil {
			return nil, err
		}
		if json.Unmarshal(data, &row.Settings) != nil {
			continue
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) MarkRun(ctx context.Context, owner int64, day string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE content_owner_settings SET last_run_day = $2 WHERE owner_user_id = $1 AND last_run_day <> $2`, owner, day)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// --- topics ------------------------------------------------------------------------------------

const contentTopicColumns = `id, owner_user_id, title, angle, notes, source, reason, score, status, item_id, created_at, updated_at`

func scanContentTopic(sc interface{ Scan(...any) error }) (*service.ContentTopic, error) {
	var t service.ContentTopic
	var item sql.NullInt64
	if err := sc.Scan(&t.ID, &t.OwnerUserID, &t.Title, &t.Angle, &t.Notes, &t.Source, &t.Reason, &t.Score, &t.Status, &item, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	if item.Valid {
		t.ItemID = &item.Int64
	}
	return &t, nil
}

func (r *contentEngineRepository) ListTopics(ctx context.Context, owner int64, status string, limit int) ([]service.ContentTopic, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+contentTopicColumns+` FROM content_topics
WHERE owner_user_id = $1 AND ($2 = '' OR status = $2)
ORDER BY CASE WHEN status = 'pending' THEN 0 ELSE 1 END, score DESC, id DESC LIMIT $3`, owner, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentTopic{}
	for rows.Next() {
		t, err := scanContentTopic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) GetTopic(ctx context.Context, owner, id int64) (*service.ContentTopic, error) {
	t, err := scanContentTopic(r.db.QueryRowContext(ctx, `SELECT `+contentTopicColumns+` FROM content_topics WHERE id = $1 AND owner_user_id = $2`, id, owner))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *contentEngineRepository) CreateTopic(ctx context.Context, t *service.ContentTopic) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO content_topics (owner_user_id, title, angle, notes, source, reason, score, status, item_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, created_at, updated_at`, t.OwnerUserID, t.Title, t.Angle, t.Notes, t.Source, t.Reason, t.Score, t.Status, nullInt64(t.ItemID)).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *contentEngineRepository) UpdateTopic(ctx context.Context, t *service.ContentTopic) error {
	return r.db.QueryRowContext(ctx, `
UPDATE content_topics SET title = $3, angle = $4, notes = $5, score = $6, status = $7, item_id = $8, updated_at = NOW()
WHERE id = $1 AND owner_user_id = $2
RETURNING updated_at`, t.ID, t.OwnerUserID, t.Title, t.Angle, t.Notes, t.Score, t.Status, nullInt64(t.ItemID)).Scan(&t.UpdatedAt)
}

func (r *contentEngineRepository) DeleteTopic(ctx context.Context, owner, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM content_topics WHERE id = $1 AND owner_user_id = $2`, id, owner)
	return err
}

// RecentTitles: the latest topics and articles, so suggestions and checks can avoid repeats.
func (r *contentEngineRepository) RecentTitles(ctx context.Context, owner int64, limit int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT title FROM (
  SELECT title, created_at FROM content_topics WHERE owner_user_id = $1 AND status <> 'dismissed'
  UNION ALL
  SELECT title, created_at FROM content_items WHERE owner_user_id = $1 AND title <> ''
) t ORDER BY created_at DESC LIMIT $2`, owner, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	seen := map[string]bool{}
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		if !seen[title] {
			seen[title] = true
			out = append(out, title)
		}
	}
	return out, rows.Err()
}

// --- items -------------------------------------------------------------------------------------

const contentItemColumns = `id, owner_user_id, topic_id, api_key_id, status, title, error, created_at, updated_at`

func (r *contentEngineRepository) CreateItem(ctx context.Context, it *service.ContentItem) error {
	data, err := json.Marshal(it.ContentItemData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO content_items (owner_user_id, topic_id, api_key_id, status, title, data, error)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
RETURNING id, created_at, updated_at`, it.OwnerUserID, nullInt64(it.TopicID), it.KeyID, it.Status, it.Title, string(data), it.Error).
		Scan(&it.ID, &it.CreatedAt, &it.UpdatedAt)
}

func (r *contentEngineRepository) GetItem(ctx context.Context, owner, id int64) (*service.ContentItem, error) {
	var it service.ContentItem
	var topic sql.NullInt64
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT `+contentItemColumns+`, data FROM content_items WHERE id = $1 AND owner_user_id = $2`, id, owner).
		Scan(&it.ID, &it.OwnerUserID, &topic, &it.KeyID, &it.Status, &it.Title, &it.Error, &it.CreatedAt, &it.UpdatedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if topic.Valid {
		it.TopicID = &topic.Int64
	}
	if err := json.Unmarshal(data, &it.ContentItemData); err != nil {
		return nil, err
	}
	if it.Sources == nil {
		it.Sources = []service.ArticleSource{}
	}
	if it.Events == nil {
		it.Events = []service.ArticleEvent{}
	}
	return &it, nil
}

// ListItems leaves out the heavy parts (drafts, Markdown, events) and adds each version's status.
func (r *contentEngineRepository) ListItems(ctx context.Context, owner int64, status string, limit int) ([]service.ContentItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+contentItemColumns+`, data->'brief', COALESCE(data->'checks', '[]'::jsonb),
       COALESCE((data->>'prompt_tokens')::bigint, 0), COALESCE((data->>'completion_tokens')::bigint, 0), COALESCE(data->>'landing_path', ''),
       COALESCE((SELECT json_agg(json_build_object('platform', v.platform, 'status', v.status) ORDER BY v.id) FROM content_variants v WHERE v.item_id = i.id), '[]'::json),
       COALESCE(data#>>'{desk,stage}', ''), COALESCE(data#>'{desk,job}', 'null'::jsonb)
FROM content_items i WHERE owner_user_id = $1 AND ($2 = '' OR status = $2)
ORDER BY created_at DESC, id DESC LIMIT $3`, owner, status, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentItem{}
	for rows.Next() {
		var it service.ContentItem
		var topic sql.NullInt64
		var brief, checks, variants, job []byte
		var prompt, completion int64
		var stage string
		if err := rows.Scan(&it.ID, &it.OwnerUserID, &topic, &it.KeyID, &it.Status, &it.Title, &it.Error, &it.CreatedAt, &it.UpdatedAt,
			&brief, &checks, &prompt, &completion, &it.LandingPath, &variants, &stage, &job); err != nil {
			return nil, err
		}
		if stage != "" {
			// Lists only carry the desk's step and its running AI step.
			it.Desk = &service.ContentDesk{Stage: stage}
			_ = json.Unmarshal(job, &it.Desk.Job)
		}
		if topic.Valid {
			it.TopicID = &topic.Int64
		}
		_ = json.Unmarshal(brief, &it.Brief)
		_ = json.Unmarshal(checks, &it.Checks)
		_ = json.Unmarshal(variants, &it.Variants)
		it.PromptTokens, it.CompletionTokens = int(prompt), int(completion)
		it.Sources, it.Events = []service.ArticleSource{}, []service.ArticleEvent{}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) SaveItem(ctx context.Context, it *service.ContentItem) error {
	data, err := json.Marshal(it.ContentItemData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
UPDATE content_items SET status = $3, title = $4, data = $5::jsonb, error = $6, updated_at = NOW()
WHERE id = $1 AND owner_user_id = $2
RETURNING updated_at`, it.ID, it.OwnerUserID, it.Status, it.Title, string(data), it.Error).Scan(&it.UpdatedAt)
}

func (r *contentEngineRepository) DeleteItem(ctx context.Context, owner, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// The topic goes back to the pool.
	if _, err := tx.ExecContext(ctx, `UPDATE content_topics SET status = 'pending', item_id = NULL, updated_at = NOW() WHERE owner_user_id = $1 AND item_id = $2`, owner, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM content_items WHERE id = $1 AND owner_user_id = $2`, id, owner); err != nil {
		return err
	}
	return tx.Commit()
}

const contentActiveStatuses = `('outlining', 'drafting', 'reviewing', 'humanizing', 'adapting')`

func (r *contentEngineRepository) CountActive(ctx context.Context, owner int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM content_items WHERE owner_user_id = $1 AND status IN `+contentActiveStatuses, owner).Scan(&n)
	return n, err
}

func (r *contentEngineRepository) FailActive(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE content_items SET status = 'failed', error = $1, updated_at = NOW(),
       data = jsonb_set(data, '{events}', COALESCE(data->'events', '[]'::jsonb) || jsonb_build_array(jsonb_build_object('at', NOW(), 'kind', 'error', 'text', $1::text)))
WHERE status IN `+contentActiveStatuses, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

// CountByStatus: articles by status, plus "topics" (pending topics) and "published" (versions).
func (r *contentEngineRepository) CountByStatus(ctx context.Context, owner int64) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT status, COUNT(*) FROM content_items WHERE owner_user_id = $1 GROUP BY status
UNION ALL SELECT 'topics', COUNT(*) FROM content_topics WHERE owner_user_id = $1 AND status = 'pending'
UNION ALL SELECT 'published', COUNT(*) FROM content_variants WHERE owner_user_id = $1 AND status = 'published'`, owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]int{}
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] += n
	}
	return out, rows.Err()
}

// --- versions ----------------------------------------------------------------------------------

const contentVariantColumns = `id, item_id, owner_user_id, platform, kind, status, title, body, data, link_code, promo_code, scheduled_at, post_url, published_at, created_at, updated_at`

func scanContentVariant(sc interface{ Scan(...any) error }) (*service.ContentVariant, error) {
	var v service.ContentVariant
	var data []byte
	var scheduled, published sql.NullTime
	if err := sc.Scan(&v.ID, &v.ItemID, &v.OwnerUserID, &v.Platform, &v.Kind, &v.Status, &v.Title, &v.Body, &data, &v.LinkCode, &v.PromoCode,
		&scheduled, &v.PostURL, &published, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &v.ContentVariantData); err != nil {
		return nil, err
	}
	if scheduled.Valid {
		v.ScheduledAt = &scheduled.Time
	}
	if published.Valid {
		v.PublishedAt = &published.Time
	}
	return &v, nil
}

func (r *contentEngineRepository) ListVariants(ctx context.Context, owner, itemID int64) ([]service.ContentVariant, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+contentVariantColumns+` FROM content_variants WHERE owner_user_id = $1 AND item_id = $2 ORDER BY id`, owner, itemID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentVariant{}
	for rows.Next() {
		v, err := scanContentVariant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) GetVariant(ctx context.Context, owner, id int64) (*service.ContentVariant, error) {
	v, err := scanContentVariant(r.db.QueryRowContext(ctx, `SELECT `+contentVariantColumns+` FROM content_variants WHERE id = $1 AND owner_user_id = $2`, id, owner))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

func (r *contentEngineRepository) UpsertVariant(ctx context.Context, v *service.ContentVariant) error {
	data, err := json.Marshal(v.ContentVariantData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO content_variants (item_id, owner_user_id, platform, kind, status, title, body, data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
ON CONFLICT (item_id, platform) DO UPDATE SET kind = EXCLUDED.kind, status = EXCLUDED.status, title = EXCLUDED.title, body = EXCLUDED.body,
       data = EXCLUDED.data, updated_at = NOW()
RETURNING id, created_at, updated_at`, v.ItemID, v.OwnerUserID, v.Platform, v.Kind, v.Status, v.Title, v.Body, string(data)).
		Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (r *contentEngineRepository) SaveVariant(ctx context.Context, v *service.ContentVariant) error {
	data, err := json.Marshal(v.ContentVariantData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
UPDATE content_variants SET kind = $3, status = $4, title = $5, body = $6, data = $7::jsonb, link_code = $8, promo_code = $9,
       scheduled_at = $10, post_url = $11, published_at = $12, updated_at = NOW()
WHERE id = $1 AND owner_user_id = $2
RETURNING updated_at`, v.ID, v.OwnerUserID, v.Kind, v.Status, v.Title, v.Body, string(data), v.LinkCode, v.PromoCode,
		nullTime(v.ScheduledAt), v.PostURL, nullTime(v.PublishedAt)).Scan(&v.UpdatedAt)
}

func (r *contentEngineRepository) ListPublished(ctx context.Context, owner int64, since time.Time) ([]service.ContentVariant, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+prefixColumns("v", contentVariantColumns)+`, i.title FROM content_variants v JOIN content_items i ON i.id = v.item_id
WHERE v.owner_user_id = $1 AND v.published_at IS NOT NULL AND v.published_at >= $2
ORDER BY v.published_at DESC LIMIT 500`, owner, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentVariant{}
	for rows.Next() {
		var v service.ContentVariant
		var data []byte
		var scheduled, published sql.NullTime
		if err := rows.Scan(&v.ID, &v.ItemID, &v.OwnerUserID, &v.Platform, &v.Kind, &v.Status, &v.Title, &v.Body, &data, &v.LinkCode, &v.PromoCode,
			&scheduled, &v.PostURL, &published, &v.CreatedAt, &v.UpdatedAt, &v.ItemTitle); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(data, &v.ContentVariantData)
		if scheduled.Valid {
			v.ScheduledAt = &scheduled.Time
		}
		if published.Valid {
			v.PublishedAt = &published.Time
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// FailDeskJobs marks 写作台 steps a previous process left running as failed.
func (r *contentEngineRepository) FailDeskJobs(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE content_items
SET data = jsonb_set(jsonb_set(data, '{desk,job,status}', '"failed"'), '{desk,job,error}', to_jsonb($1::text)), updated_at = NOW()
WHERE status = 'desk' AND data#>>'{desk,job,status}' = 'running'`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

// --- persona & versions ------------------------------------------------------------------------

func (r *contentEngineRepository) GetPersona(ctx context.Context, owner int64) (*service.ContentPersona, error) {
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT data FROM content_personas WHERE owner_user_id = $1`, owner).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p service.ContentPersona
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *contentEngineRepository) SavePersona(ctx context.Context, owner int64, p *service.ContentPersona) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO content_personas (owner_user_id, data, updated_at) VALUES ($1, $2::jsonb, NOW())
ON CONFLICT (owner_user_id) DO UPDATE SET data = EXCLUDED.data, updated_at = NOW()`, owner, string(data))
	return err
}

func (r *contentEngineRepository) CreateVersion(ctx context.Context, v *service.ContentVersion) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO content_item_versions (item_id, owner_user_id, kind, label, markdown) VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at`, v.ItemID, v.OwnerUserID, v.Kind, v.Label, v.Markdown).Scan(&v.ID, &v.CreatedAt)
}

func (r *contentEngineRepository) ListVersions(ctx context.Context, owner, itemID int64) ([]service.ContentVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, item_id, kind, label, char_length(markdown), created_at FROM content_item_versions
WHERE owner_user_id = $1 AND item_id = $2 ORDER BY id DESC LIMIT 200`, owner, itemID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentVersion{}
	for rows.Next() {
		var v service.ContentVersion
		if err := rows.Scan(&v.ID, &v.ItemID, &v.Kind, &v.Label, &v.Chars, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) GetVersion(ctx context.Context, owner, id int64) (*service.ContentVersion, error) {
	var v service.ContentVersion
	err := r.db.QueryRowContext(ctx, `
SELECT id, item_id, owner_user_id, kind, label, markdown, char_length(markdown), created_at FROM content_item_versions
WHERE id = $1 AND owner_user_id = $2`, id, owner).Scan(&v.ID, &v.ItemID, &v.OwnerUserID, &v.Kind, &v.Label, &v.Markdown, &v.Chars, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *contentEngineRepository) FailGenerating(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE content_variants SET status = 'failed', data = jsonb_set(data, '{error}', to_jsonb($1::text)), updated_at = NOW()
WHERE status = 'generating'`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

// --- rules -------------------------------------------------------------------------------------

func (r *contentEngineRepository) ListRules(ctx context.Context, owner int64) ([]service.ContentRule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, owner_user_id, rule, origin, enabled, created_at FROM content_rules WHERE owner_user_id = $1 ORDER BY id`, owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ContentRule{}
	for rows.Next() {
		var x service.ContentRule
		if err := rows.Scan(&x.ID, &x.OwnerUserID, &x.Rule, &x.Origin, &x.Enabled, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *contentEngineRepository) CreateRule(ctx context.Context, x *service.ContentRule) error {
	return r.db.QueryRowContext(ctx, `INSERT INTO content_rules (owner_user_id, rule, origin, enabled) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		x.OwnerUserID, x.Rule, x.Origin, x.Enabled).Scan(&x.ID, &x.CreatedAt)
}

func (r *contentEngineRepository) SetRuleEnabled(ctx context.Context, owner, id int64, enabled bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE content_rules SET enabled = $3 WHERE id = $1 AND owner_user_id = $2`, id, owner, enabled)
	return err
}

func (r *contentEngineRepository) DeleteRule(ctx context.Context, owner, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM content_rules WHERE id = $1 AND owner_user_id = $2`, id, owner)
	return err
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// prefixColumns qualifies a comma-separated column list with a table alias.
func prefixColumns(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
