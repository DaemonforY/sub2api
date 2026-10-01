package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type promptLibraryRepository struct {
	db *sql.DB
}

// NewPromptLibraryRepository creates the prompt library repository (raw SQL).
func NewPromptLibraryRepository(db *sql.DB) service.PromptLibraryRepository {
	return &promptLibraryRepository{db: db}
}

const promptItemSelect = `
SELECT i.id, i.source_id,
       CASE i.source_id WHEN 'user' THEN '社区分享' WHEN 'official' THEN 'HiveGPT 精选' ELSE COALESCE(ps.name, i.source_id) END,
       i.external_id, i.owner_user_id, COALESCE(u.email, ''), i.kind,
       CASE WHEN i.title_zh <> '' THEN i.title_zh ELSE i.title END, i.title, i.title_zh, i.prompt, i.description, i.cover_url,
       i.reference_image_urls, i.source_tags, i.scenes, i.tags, i.model, i.lang, i.needs_reference, i.auto_flags,
       i.author, i.source_url, i.visibility, i.status, i.review_note, i.curated, i.featured, i.quality_score,
       i.use_count, i.favorite_count, i.dedupe_key, i.published_at, i.created_at, i.updated_at
FROM prompt_items i
LEFT JOIN prompt_sources ps ON ps.id = i.source_id
LEFT JOIN users u ON u.id = i.owner_user_id`

func scanPromptItem(row rowScanner) (*service.PromptItem, error) {
	var (
		item      service.PromptItem
		owner     sql.NullInt64
		refs      []byte
		srcTags   []byte
		scenes    pq.StringArray
		tags      pq.StringArray
		flags     pq.StringArray
		published sql.NullTime
	)
	if err := row.Scan(&item.ID, &item.SourceID, &item.SourceName, &item.ExternalID, &owner, &item.OwnerEmail, &item.Kind, &item.Title,
		&item.OriginalTitle, &item.TitleZh, &item.Prompt, &item.Description, &item.CoverURL, &refs, &srcTags, &scenes, &tags, &item.Model, &item.Lang, &item.NeedsReference,
		&flags, &item.Author, &item.SourceURL, &item.Visibility, &item.Status, &item.ReviewNote, &item.Curated, &item.Featured,
		&item.QualityScore, &item.UseCount, &item.FavoriteCount, &item.DedupeKey, &published, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, err
	}
	if owner.Valid {
		id := owner.Int64
		item.OwnerUserID = &id
	}
	if published.Valid {
		t := published.Time
		item.PublishedAt = &t
	}
	if item.OriginalTitle == item.Title {
		item.OriginalTitle = ""
	}
	item.ReferenceImageURLs = decodeStringList(refs)
	item.SourceTags = decodeStringList(srcTags)
	item.Scenes = nonNilStrings(scenes)
	item.Tags = nonNilStrings(tags)
	item.AutoFlags = nonNilStrings(flags)
	return &item, nil
}

func decodeStringList(raw []byte) []string {
	var out []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func encodeStringList(values []string) string {
	if values == nil {
		values = []string{}
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

// promptWhere builds the WHERE clause of a listing.
type promptWhere struct {
	conds []string
	args  []any
}

func (w *promptWhere) add(cond string, args ...any) {
	for _, arg := range args {
		w.args = append(w.args, arg)
		cond = strings.Replace(cond, "?", "$"+strconv.Itoa(len(w.args)), 1)
	}
	w.conds = append(w.conds, cond)
}

func (w *promptWhere) sql() string {
	if len(w.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.conds, " AND ")
}

func buildPromptWhere(q service.PromptListQuery, withScene bool) *promptWhere {
	w := &promptWhere{}
	switch {
	case q.OwnerUserID > 0:
		w.add("i.owner_user_id = ?", q.OwnerUserID)
	case !q.Admin:
		w.add("i.status = 'active' AND i.visibility = 'public'")
	default:
		if q.Status != "" {
			w.add("i.status = ?", q.Status)
		}
		if q.Curated != nil {
			w.add("i.curated = ?", *q.Curated)
		}
		if q.Featured != nil {
			w.add("i.featured = ?", *q.Featured)
		}
	}
	if q.SourceID != "" {
		w.add("i.source_id = ?", q.SourceID)
	}
	if q.Kind != "" {
		w.add("i.kind = ?", q.Kind)
	}
	switch q.Model {
	case "here":
		w.add("i.model IN ('gpt-image-2', 'unknown')")
	case "other":
		w.add("i.model NOT IN ('gpt-image-2', 'unknown')")
	}
	if q.Tag != "" {
		w.add("(? = ANY(i.tags) OR jsonb_exists(i.source_tags, ?))", q.Tag, q.Tag)
	}
	if q.Keyword != "" {
		cond := "(i.search_text LIKE ?"
		args := []any{"%" + escapeLike(strings.ToLower(q.Keyword)) + "%"}
		if q.Admin {
			cond += " OR i.external_id = ?"
			args = append(args, q.Keyword)
			if id, err := strconv.ParseInt(q.Keyword, 10, 64); err == nil {
				cond += " OR i.id = ?"
				args = append(args, id)
			}
		}
		w.add(cond+")", args...)
	}
	if withScene && q.Scene != "" {
		w.add("? = ANY(i.scenes)", q.Scene)
	}
	return w
}

func promptOrderBy(sort string) string {
	switch sort {
	case service.PromptSortRecommended:
		return " ORDER BY (i.quality_score + CASE WHEN i.featured THEN 6 ELSE 0 END + LEAST(ln(1 + i.use_count) * 2, 10)) DESC, i.id DESC"
	case service.PromptSortLatest:
		return " ORDER BY COALESCE(i.published_at, i.created_at) DESC, i.id DESC"
	default:
		return " ORDER BY i.use_count DESC, i.favorite_count DESC, i.featured DESC, i.quality_score DESC, i.id DESC"
	}
}

func (r *promptLibraryRepository) queryItems(ctx context.Context, query string, args ...any) ([]service.PromptItem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []service.PromptItem{}
	for rows.Next() {
		item, err := scanPromptItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *promptLibraryRepository) List(ctx context.Context, q service.PromptListQuery) ([]service.PromptItem, int64, error) {
	w := buildPromptWhere(q, true)
	totalCh := make(chan error, 1)
	var total int64
	go func() {
		totalCh <- r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM prompt_items i"+w.sql(), w.args...).Scan(&total)
	}()
	args := append(append([]any{}, w.args...), q.PageSize, (q.Page-1)*q.PageSize)
	query := promptItemSelect + w.sql() + promptOrderBy(q.Sort) + fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2)
	items, err := r.queryItems(ctx, query, args...)
	if countErr := <-totalCh; countErr != nil {
		return nil, 0, countErr
	}
	return items, total, err
}

func (r *promptLibraryRepository) SceneCounts(ctx context.Context, q service.PromptListQuery) (map[string]int64, error) {
	w := buildPromptWhere(q, false)
	rows, err := r.db.QueryContext(ctx, "SELECT s, COUNT(*) FROM prompt_items i CROSS JOIN LATERAL unnest(i.scenes) AS s"+w.sql()+" GROUP BY s", w.args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	counts := map[string]int64{}
	for rows.Next() {
		var scene string
		var n int64
		if err := rows.Scan(&scene, &n); err != nil {
			return nil, err
		}
		counts[scene] = n
	}
	return counts, rows.Err()
}

func (r *promptLibraryRepository) Get(ctx context.Context, id int64) (*service.PromptItem, error) {
	item, err := scanPromptItem(r.db.QueryRowContext(ctx, promptItemSelect+" WHERE i.id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func (r *promptLibraryRepository) Insert(ctx context.Context, item *service.PromptItem) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO prompt_items (source_id, external_id, owner_user_id, kind, title, prompt, description, cover_url, reference_image_urls,
    source_tags, scenes, tags, model, lang, needs_reference, auto_flags, author, source_url, visibility, status, review_note,
    curated, featured, quality_score, dedupe_key, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
RETURNING id, created_at, updated_at`,
		item.SourceID, item.ExternalID, item.OwnerUserID, item.Kind, item.Title, item.Prompt, item.Description, item.CoverURL,
		encodeStringList(item.ReferenceImageURLs), encodeStringList(item.SourceTags), pq.Array(nonNilStrings(item.Scenes)),
		pq.Array(nonNilStrings(item.Tags)), item.Model, item.Lang, item.NeedsReference, pq.Array(nonNilStrings(item.AutoFlags)),
		item.Author, item.SourceURL, item.Visibility, item.Status, item.ReviewNote, item.Curated, item.Featured, item.QualityScore,
		item.DedupeKey, item.PublishedAt,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (r *promptLibraryRepository) Update(ctx context.Context, item *service.PromptItem) error {
	return r.db.QueryRowContext(ctx, `
UPDATE prompt_items SET kind = $2, title = $3, prompt = $4, description = $5, cover_url = $6, scenes = $7, tags = $8, model = $9,
    lang = $10, needs_reference = $11, auto_flags = $12, visibility = $13, status = $14, review_note = $15, curated = $16,
    featured = $17, quality_score = $18, dedupe_key = $19, published_at = $20, title_zh = $21, updated_at = NOW()
WHERE id = $1
RETURNING updated_at`,
		item.ID, item.Kind, item.Title, item.Prompt, item.Description, item.CoverURL, pq.Array(nonNilStrings(item.Scenes)),
		pq.Array(nonNilStrings(item.Tags)), item.Model, item.Lang, item.NeedsReference, pq.Array(nonNilStrings(item.AutoFlags)),
		item.Visibility, item.Status, item.ReviewNote, item.Curated, item.Featured, item.QualityScore, item.DedupeKey, item.PublishedAt,
		item.TitleZh,
	).Scan(&item.UpdatedAt)
}

func (r *promptLibraryRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM prompt_items WHERE id = $1`, id)
	return err
}

func (r *promptLibraryRepository) CountOwned(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prompt_items WHERE owner_user_id = $1`, userID).Scan(&n)
	return n, err
}

// Ordered set union / difference of text arrays (first occurrence wins); an empty scene list becomes {other}.
const (
	sqlArrayUnion = `ARRAY(SELECT x FROM unnest(%s || $2::text[]) WITH ORDINALITY AS t(x, o) GROUP BY x ORDER BY MIN(o))`
	sqlArrayMinus = `ARRAY(SELECT x FROM unnest(%s) WITH ORDINALITY AS t(x, o) WHERE x <> ALL($2::text[]) ORDER BY o)`
)

func (r *promptLibraryRepository) Batch(ctx context.Context, ids []int64, op service.PromptBatchOp) (int64, error) {
	var set string
	args := []any{pq.Array(ids)}
	switch op.Action {
	case "add_scenes":
		set = "scenes = COALESCE(NULLIF(" + fmt.Sprintf(sqlArrayUnion, "array_remove(scenes, 'other')") + ", '{}'), '{other}')"
		args = append(args, pq.Array(op.Scenes))
	case "remove_scenes":
		set = "scenes = COALESCE(NULLIF(" + fmt.Sprintf(sqlArrayMinus, "scenes") + ", '{}'), '{other}')"
		args = append(args, pq.Array(op.Scenes))
	case "set_scenes":
		set = "scenes = $2::text[]"
		args = append(args, pq.Array(op.Scenes))
	case "add_tags":
		set = "tags = " + fmt.Sprintf(sqlArrayUnion, "tags")
		args = append(args, pq.Array(op.Tags))
	case "remove_tags":
		set = "tags = " + fmt.Sprintf(sqlArrayMinus, "tags")
		args = append(args, pq.Array(op.Tags))
	case "set_status":
		set = "status = $2::text, review_note = $3::text, published_at = CASE WHEN $2::text = 'active' AND published_at IS NULL THEN NOW() ELSE published_at END"
		args = append(args, op.Status, op.Note)
	case "set_featured":
		set = "featured = $2"
		args = append(args, op.Featured)
	case "mark_reviewed":
		set = "curated = TRUE"
	default:
		return 0, service.ErrPromptBatchInvalid
	}
	res, err := r.db.ExecContext(ctx, "UPDATE prompt_items SET "+set+", curated = TRUE, updated_at = NOW() WHERE id = ANY($1)", args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *promptLibraryRepository) Stats(ctx context.Context) (*service.PromptLibraryStats, error) {
	stats := &service.PromptLibraryStats{StatusCounts: map[string]int64{}}
	rows, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM prompt_items GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		stats.StatusCounts[status] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	err = r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status = 'pending' AND source_id = 'user'),
       COUNT(*) FILTER (WHERE status = 'active' AND NOT curated),
       COALESCE(SUM(use_count), 0)
FROM prompt_items`).Scan(&stats.PendingUser, &stats.Uncurated, &stats.TotalUses)
	return stats, err
}

func (r *promptLibraryRepository) TagCounts(ctx context.Context, limit int) ([]service.PromptTagCount, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t, COUNT(*) FROM prompt_items CROSS JOIN LATERAL unnest(tags) AS t GROUP BY t ORDER BY COUNT(*) DESC, t LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptTagCount{}
	for rows.Next() {
		var tc service.PromptTagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

func (r *promptLibraryRepository) RecordUse(ctx context.Context, userID, itemID int64, window time.Duration) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var last sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT last_used_at FROM prompt_item_uses WHERE user_id = $1 AND item_id = $2 FOR UPDATE`, userID, itemID).Scan(&last)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO prompt_item_uses (user_id, item_id, uses, last_used_at, updated_at) VALUES ($1, $2, 1, NOW(), NOW())
ON CONFLICT (user_id, item_id) DO UPDATE SET uses = prompt_item_uses.uses + 1, last_used_at = NOW(), updated_at = NOW()`, userID, itemID); err != nil {
		return 0, err
	}
	var count int64
	if !last.Valid || time.Since(last.Time) >= window {
		err = tx.QueryRowContext(ctx, `UPDATE prompt_items SET use_count = use_count + 1 WHERE id = $1 RETURNING use_count`, itemID).Scan(&count)
	} else {
		err = tx.QueryRowContext(ctx, `SELECT use_count FROM prompt_items WHERE id = $1`, itemID).Scan(&count)
	}
	if err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

func (r *promptLibraryRepository) SetFavorite(ctx context.Context, userID, itemID int64, favorited bool) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var prev bool
	err = tx.QueryRowContext(ctx, `SELECT favorited FROM prompt_item_uses WHERE user_id = $1 AND item_id = $2 FOR UPDATE`, userID, itemID).Scan(&prev)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO prompt_item_uses (user_id, item_id, favorited, updated_at) VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id, item_id) DO UPDATE SET favorited = EXCLUDED.favorited, updated_at = NOW()`, userID, itemID, favorited); err != nil {
		return 0, err
	}
	delta := 0
	if favorited && !prev {
		delta = 1
	} else if !favorited && prev {
		delta = -1
	}
	var count int64
	err = tx.QueryRowContext(ctx, `UPDATE prompt_items SET favorite_count = GREATEST(0, favorite_count + $2) WHERE id = $1 RETURNING favorite_count`, itemID, delta).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, tx.Commit()
}

func (r *promptLibraryRepository) UserHistory(ctx context.Context, userID int64, limit int) ([]service.PromptUseHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT u.item_id, i.scenes, i.model, i.lang, u.uses, u.favorited, COALESCE(u.last_used_at, u.updated_at)
FROM prompt_item_uses u JOIN prompt_items i ON i.id = u.item_id
WHERE u.user_id = $1
ORDER BY u.updated_at DESC
LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.PromptUseHistory
	for rows.Next() {
		var h service.PromptUseHistory
		var scenes pq.StringArray
		if err := rows.Scan(&h.ItemID, &scenes, &h.Model, &h.Lang, &h.Uses, &h.Favorited, &h.LastUsedAt); err != nil {
			return nil, err
		}
		h.Scenes = nonNilStrings(scenes)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *promptLibraryRepository) Candidates(ctx context.Context, userID int64, kind string, scenes []string, limit int) ([]service.PromptItem, error) {
	return r.queryItems(ctx, promptItemSelect+`
WHERE i.status = 'active' AND i.visibility = 'public' AND i.kind = $2
  AND (cardinality($3::text[]) = 0 OR i.scenes && $3::text[])
  AND (i.owner_user_id IS NULL OR i.owner_user_id <> $1)
  AND NOT EXISTS (SELECT 1 FROM prompt_item_uses pu WHERE pu.user_id = $1 AND pu.item_id = i.id)
ORDER BY i.use_count DESC, i.favorite_count DESC, i.featured DESC, i.quality_score DESC, i.id DESC
LIMIT $4`, userID, kind, pq.Array(nonNilStrings(scenes)), limit)
}

func (r *promptLibraryRepository) ListSources(ctx context.Context) ([]service.PromptSource, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT s.id, s.name, s.format, s.url, s.homepage, s.enabled, s.item_count, s.last_synced_at, s.last_error, COALESCE(c.n, 0)
FROM prompt_sources s
LEFT JOIN (SELECT source_id, COUNT(*) AS n FROM prompt_items WHERE status = 'active' AND visibility = 'public' GROUP BY source_id) c ON c.source_id = s.id
ORDER BY s.created_at, s.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptSource{}
	for rows.Next() {
		src, err := scanPromptSource(rows, true)
		if err != nil {
			return nil, err
		}
		out = append(out, *src)
	}
	return out, rows.Err()
}

func scanPromptSource(row rowScanner, withActive bool) (*service.PromptSource, error) {
	var src service.PromptSource
	var synced sql.NullTime
	dest := []any{&src.ID, &src.Name, &src.Format, &src.URL, &src.Homepage, &src.Enabled, &src.ItemCount, &synced, &src.LastError}
	if withActive {
		dest = append(dest, &src.ActiveCount)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	if synced.Valid {
		t := synced.Time
		src.LastSyncedAt = &t
	}
	return &src, nil
}

func (r *promptLibraryRepository) GetSource(ctx context.Context, id string) (*service.PromptSource, error) {
	src, err := scanPromptSource(r.db.QueryRowContext(ctx, `
SELECT id, name, format, url, homepage, enabled, item_count, last_synced_at, last_error FROM prompt_sources WHERE id = $1`, id), false)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return src, err
}

func (r *promptLibraryRepository) SetSourceEnabled(ctx context.Context, id string, enabled bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE prompt_sources SET enabled = $2, updated_at = NOW() WHERE id = $1`, id, enabled); err != nil {
		return err
	}
	// A disabled source disappears from the library; enabling it again restores what was visible.
	if enabled {
		_, err = tx.ExecContext(ctx, `UPDATE prompt_items SET status = 'active', review_note = '', updated_at = NOW() WHERE source_id = $1 AND status = 'hidden' AND review_note = 'source_disabled'`, id)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE prompt_items SET status = 'hidden', review_note = 'source_disabled', updated_at = NOW() WHERE source_id = $1 AND status = 'active'`, id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *promptLibraryRepository) RecordSourceSync(ctx context.Context, id string, count int, syncErr string) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE prompt_sources SET item_count = $2, last_error = $3::text,
    last_synced_at = CASE WHEN $3::text = '' THEN NOW() ELSE last_synced_at END, updated_at = NOW()
WHERE id = $1`, id, count, syncErr)
	return err
}

func (r *promptLibraryRepository) UpsertSourceItems(ctx context.Context, sourceID string, items []service.PromptItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO prompt_items (source_id, external_id, kind, title, prompt, description, cover_url, reference_image_urls, source_tags,
    scenes, model, lang, needs_reference, auto_flags, author, source_url, visibility, status, quality_score, dedupe_key,
    sync_hash, published_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10, $11, $12, $13, $14, $15, $16, 'public', $17, $18, $19, $20, $21)
ON CONFLICT (source_id, external_id) DO UPDATE SET
    prompt = EXCLUDED.prompt,
    description = EXCLUDED.description,
    cover_url = EXCLUDED.cover_url,
    reference_image_urls = EXCLUDED.reference_image_urls,
    source_tags = EXCLUDED.source_tags,
    lang = EXCLUDED.lang,
    auto_flags = EXCLUDED.auto_flags,
    author = EXCLUDED.author,
    source_url = EXCLUDED.source_url,
    dedupe_key = EXCLUDED.dedupe_key,
    sync_hash = EXCLUDED.sync_hash,
    quality_score = EXCLUDED.quality_score,
    published_at = COALESCE(EXCLUDED.published_at, prompt_items.published_at),
    title = CASE WHEN prompt_items.curated THEN prompt_items.title ELSE EXCLUDED.title END,
    -- A translation belongs to the title it was made from.
    title_zh = CASE WHEN prompt_items.curated OR prompt_items.title = EXCLUDED.title THEN prompt_items.title_zh ELSE '' END,
    kind = CASE WHEN prompt_items.curated THEN prompt_items.kind ELSE EXCLUDED.kind END,
    scenes = CASE WHEN prompt_items.curated THEN prompt_items.scenes ELSE EXCLUDED.scenes END,
    model = CASE WHEN prompt_items.curated THEN prompt_items.model ELSE EXCLUDED.model END,
    needs_reference = CASE WHEN prompt_items.curated THEN prompt_items.needs_reference ELSE EXCLUDED.needs_reference END,
    status = CASE
        WHEN prompt_items.curated OR prompt_items.review_note = 'source_disabled' THEN prompt_items.status
        WHEN EXCLUDED.status = 'hidden' THEN 'hidden'
        WHEN prompt_items.status = 'hidden' THEN 'active'
        ELSE prompt_items.status END,
    updated_at = NOW()
WHERE prompt_items.sync_hash IS DISTINCT FROM EXCLUDED.sync_hash`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()
	for i := range items {
		item := &items[i]
		if _, err := stmt.ExecContext(ctx, sourceID, item.ExternalID, item.Kind, item.Title, item.Prompt, item.Description, item.CoverURL,
			encodeStringList(item.ReferenceImageURLs), encodeStringList(item.SourceTags), pq.Array(nonNilStrings(item.Scenes)), item.Model,
			item.Lang, item.NeedsReference, pq.Array(nonNilStrings(item.AutoFlags)), item.Author, item.SourceURL, item.Status,
			item.QualityScore, item.DedupeKey, item.SyncHash, item.PublishedAt); err != nil {
			return fmt.Errorf("upsert prompt %s: %w", item.ExternalID, err)
		}
	}
	return tx.Commit()
}

func (r *promptLibraryRepository) RefreshDuplicates(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
WITH ranked AS (
    SELECT id, curated,
           ROW_NUMBER() OVER (PARTITION BY dedupe_key ORDER BY curated DESC, (cover_url <> '') DESC, (model = 'gpt-image-2') DESC, use_count DESC, id) AS rn
    FROM prompt_items
    WHERE source_id NOT IN ('user', 'official') AND dedupe_key <> ''
      AND (status IN ('active', 'duplicate') OR (curated AND status = 'hidden'))
)
UPDATE prompt_items p
SET status = CASE WHEN r.rn = 1 THEN 'active' ELSE 'duplicate' END, updated_at = NOW()
FROM ranked r
WHERE p.id = r.id AND NOT r.curated AND p.status <> CASE WHEN r.rn = 1 THEN 'active' ELSE 'duplicate' END`)
	return err
}

func (r *promptLibraryRepository) InsertCover(ctx context.Context, file string, userID, size int64) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO prompt_covers (file, user_id, size_bytes) VALUES ($1, $2, $3)`, file, userID, size)
	return err
}

func (r *promptLibraryRepository) CountCoversSince(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prompt_covers WHERE user_id = $1 AND created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *promptLibraryRepository) CoverOwnedBy(ctx context.Context, file string, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM prompt_covers WHERE file = $1 AND user_id = $2)`, file, userID).Scan(&ok)
	return ok, err
}

func (r *promptLibraryRepository) ApplyTitleTranslations(ctx context.Context, translations map[string]string) (int64, error) {
	en := make([]string, 0, len(translations))
	zh := make([]string, 0, len(translations))
	for k, v := range translations {
		if k != "" && v != "" {
			en = append(en, k)
			zh = append(zh, v)
		}
	}
	if len(en) == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE prompt_items p SET title_zh = t.zh, updated_at = NOW()
FROM unnest($1::text[], $2::text[]) AS t(en, zh)
WHERE p.title = t.en AND p.title_zh = '' AND p.source_id NOT IN ('user', 'official')`, pq.Array(en), pq.Array(zh))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// promptUntranslated selects listed source items whose title has no Chinese and no translation yet.
const promptUntranslated = `FROM prompt_items WHERE title_zh = '' AND source_id NOT IN ('user', 'official') AND status = 'active' AND title !~ '[一-鿿]'`

func (r *promptLibraryRepository) UntranslatedTitles(ctx context.Context, limit int) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT title `+promptUntranslated+` ORDER BY title LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		out = append(out, title)
	}
	return out, rows.Err()
}

func (r *promptLibraryRepository) CountUntranslated(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT title) `+promptUntranslated).Scan(&n)
	return n, err
}

func (r *promptLibraryRepository) ApplySceneOverrides(ctx context.Context, overrides map[string][]string) (int64, error) {
	keys := make([]string, 0, len(overrides))
	values := make([]string, 0, len(overrides))
	for key, scenes := range overrides {
		if key != "" && len(scenes) > 0 {
			keys = append(keys, key)
			values = append(values, strings.Join(scenes, ","))
		}
	}
	if len(keys) == 0 {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE prompt_items p
SET scenes = string_to_array(t.sc, ','),
    kind = CASE WHEN split_part(t.sc, ',', 1) = 'video' THEN 'video' ELSE 'image' END,
    updated_at = NOW()
FROM unnest($1::text[], $2::text[]) AS t(k, sc)
WHERE p.source_id || ':' || p.external_id = t.k
  AND NOT p.curated AND p.source_id NOT IN ('user', 'official')
  AND p.scenes IS DISTINCT FROM string_to_array(t.sc, ',')`, pq.Array(keys), pq.Array(values))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
