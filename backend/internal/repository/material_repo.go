package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type materialRepository struct {
	db *sql.DB
}

// NewMaterialRepository creates the 诗画素材库 store (raw SQL).
func NewMaterialRepository(db *sql.DB) service.MaterialRepository {
	return &materialRepository{db: db}
}

func (r *materialRepository) UpsertAuthor(ctx context.Context, a *service.MaterialAuthor) (int64, error) {
	aliases, err := json.Marshal(nonNilStrings(a.Aliases))
	if err != nil {
		return 0, err
	}
	kind := a.Kind
	if kind == "" {
		kind = "poet"
	}
	var id int64
	// DO UPDATE (a no-op) so RETURNING yields the existing row's id too.
	err = r.db.QueryRowContext(ctx, `
INSERT INTO mat_authors (name, aliases, dynasty, birth_year, death_year, bio, kind)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (name, dynasty) DO UPDATE SET name = EXCLUDED.name
RETURNING id`, a.Name, aliases, a.Dynasty, a.BirthYear, a.DeathYear, a.Bio, kind).Scan(&id)
	return id, err
}

func (r *materialRepository) ImportText(ctx context.Context, t *service.MaterialText, segments []service.MaterialSegment) (bool, error) {
	themes, err := json.Marshal(nonNilStrings(t.Themes))
	if err != nil {
		return false, err
	}
	seasons, err := json.Marshal(nonNilStrings(t.Seasons))
	if err != nil {
		return false, err
	}
	textbook := []byte(t.Textbook)
	if len(textbook) == 0 || string(textbook) == "null" {
		textbook = []byte("[]")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO mat_texts (slug, kind, title, author_id, performer, body, translation, appreciation, license_status,
    license_note, sources, status, verify1_by, verify1_at, verify2_by, verify2_at, external_url, themes, seasons,
    textbook, shihua_work_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
ON CONFLICT DO NOTHING
RETURNING id`,
		t.Slug, t.Kind, t.Title, t.AuthorID, t.Performer, t.Body, t.Translation, t.Appreciation, t.LicenseStatus,
		t.LicenseNote, t.Sources, t.Status, t.Verify1By, t.Verify1At, t.Verify2By, t.Verify2At, t.ExternalURL, themes,
		seasons, textbook, t.ShihuaWorkID, t.CreatedBy).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil // slug or shihua_work_id already there
	}
	if err != nil {
		return false, err
	}
	for _, s := range segments {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO mat_text_segments (text_id, seq, label, content, scene_desc, is_famous) VALUES ($1, $2, $3, $4, $5, $6)`,
			id, s.Seq, s.Label, s.Content, s.SceneDesc, s.IsFamous); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	t.ID = id
	return true, nil
}

const materialTextColumns = `t.id, t.slug, t.kind, t.title, t.author_id, t.performer, t.body, t.translation, t.appreciation,
t.license_status, t.license_note, t.sources, t.status, t.verify1_by, t.verify1_at, t.verify2_by, t.verify2_at,
t.external_url, t.themes, t.seasons, t.textbook, t.shihua_work_id, t.created_by, t.created_at, t.updated_at,
COALESCE(a.name, ''), COALESCE(a.dynasty, '')`

func scanMaterialText(row shihuaScanner, extra ...any) (*service.MaterialText, error) {
	var t service.MaterialText
	var authorID, v1By, v2By, workID, createdBy sql.NullInt64
	var v1At, v2At sql.NullTime
	var themes, seasons, textbook []byte
	dest := []any{&t.ID, &t.Slug, &t.Kind, &t.Title, &authorID, &t.Performer, &t.Body, &t.Translation, &t.Appreciation,
		&t.LicenseStatus, &t.LicenseNote, &t.Sources, &t.Status, &v1By, &v1At, &v2By, &v2At,
		&t.ExternalURL, &themes, &seasons, &textbook, &workID, &createdBy, &t.CreatedAt, &t.UpdatedAt,
		&t.AuthorName, &t.Dynasty}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return nil, err
	}
	t.AuthorID, t.Verify1By, t.Verify2By = shihuaNullInt(authorID), shihuaNullInt(v1By), shihuaNullInt(v2By)
	t.ShihuaWorkID, t.CreatedBy = shihuaNullInt(workID), shihuaNullInt(createdBy)
	t.Verify1At, t.Verify2At = shihuaNullTime(v1At), shihuaNullTime(v2At)
	t.Themes, t.Seasons = jsonStrings(themes), jsonStrings(seasons)
	t.Textbook = json.RawMessage(textbook)
	return &t, nil
}

func (r *materialRepository) ListTexts(ctx context.Context, q service.MaterialTextQuery) ([]service.MaterialText, int, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if q.Kind != "" {
		where = append(where, "t.kind = "+arg(q.Kind))
	}
	if q.Status != "" {
		where = append(where, "t.status = "+arg(q.Status))
	}
	if q.License != "" {
		where = append(where, "t.license_status = "+arg(q.License))
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		p := arg("%" + s + "%")
		where = append(where, "(t.title ILIKE "+p+" OR t.slug ILIKE "+p+" OR a.name ILIKE "+p+" OR t.body ILIKE "+p+")")
	}
	from := ` FROM mat_texts t LEFT JOIN mat_authors a ON a.id = t.author_id`
	if len(where) > 0 {
		from += " WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + materialTextColumns + `,
       (SELECT COUNT(*) FROM mat_text_segments s WHERE s.text_id = t.id)` + from +
		` ORDER BY t.id LIMIT ` + arg(q.Limit) + ` OFFSET ` + arg(q.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	texts := []service.MaterialText{}
	for rows.Next() {
		var n int
		t, err := scanMaterialText(rows, &n)
		if err != nil {
			return nil, 0, err
		}
		t.SegmentCount = n
		texts = append(texts, *t)
	}
	return texts, total, rows.Err()
}

func (r *materialRepository) GetText(ctx context.Context, id int64) (*service.MaterialText, error) {
	t, err := scanMaterialText(r.db.QueryRowContext(ctx, `SELECT `+materialTextColumns+`
FROM mat_texts t LEFT JOIN mat_authors a ON a.id = t.author_id WHERE t.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, text_id, seq, label, content, scene_desc, is_famous FROM mat_text_segments WHERE text_id = $1 ORDER BY seq`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	t.Segments = []service.MaterialSegment{}
	for rows.Next() {
		var s service.MaterialSegment
		if err := rows.Scan(&s.ID, &s.TextID, &s.Seq, &s.Label, &s.Content, &s.SceneDesc, &s.IsFamous); err != nil {
			return nil, err
		}
		t.Segments = append(t.Segments, s)
	}
	t.SegmentCount = len(t.Segments)
	return t, rows.Err()
}

func (r *materialRepository) ListStyles(ctx context.Context, includeDisabled bool) ([]service.MaterialStyle, error) {
	query := `
SELECT id, name, category, desc_zh, prompt_en, negative_en, suits_emotions, suits_categories, options, default_size,
       default_quality, enabled, sort
FROM mat_styles`
	if !includeDisabled {
		query += ` WHERE enabled`
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	styles := []service.MaterialStyle{}
	for rows.Next() {
		var s service.MaterialStyle
		var emotions, categories, options []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Category, &s.DescZh, &s.PromptEn, &s.NegativeEn, &emotions, &categories,
			&options, &s.DefaultSize, &s.DefaultQuality, &s.Enabled, &s.Sort); err != nil {
			return nil, err
		}
		s.SuitsEmotions, s.SuitsCategories, s.Options = jsonStrings(emotions), jsonStrings(categories), json.RawMessage(options)
		styles = append(styles, s)
	}
	return styles, rows.Err()
}

func (r *materialRepository) ListBlocklist(ctx context.Context, enabledOnly bool) ([]service.MaterialBlockTerm, error) {
	query := `SELECT id, term, kind, match, action, applies_to, reason, enabled FROM mat_blocklist`
	if enabledOnly {
		query += ` WHERE enabled`
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY kind, id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	terms := []service.MaterialBlockTerm{}
	for rows.Next() {
		var t service.MaterialBlockTerm
		if err := rows.Scan(&t.ID, &t.Term, &t.Kind, &t.Match, &t.Action, &t.AppliesTo, &t.Reason, &t.Enabled); err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}
	return terms, rows.Err()
}

func jsonStrings(raw []byte) []string {
	var v []string
	if err := json.Unmarshal(raw, &v); err != nil || v == nil {
		return []string{}
	}
	return v
}

// ---- imagery ----

const materialImageryColumns = `i.id, i.name, i.aliases, i.category, i.meaning, i.visual_zh, i.visual_en, i.emotions, i.seasons,
i.times, i.weathers, i.eras, i.style_ids, i.status, i.created_by, i.created_at, i.updated_at`

func scanMaterialImagery(row shihuaScanner, extra ...any) (*service.MaterialImagery, error) {
	var im service.MaterialImagery
	var aliases, emotions, seasons, times, weathers, eras, styleIDs []byte
	var createdBy sql.NullInt64
	dest := []any{&im.ID, &im.Name, &aliases, &im.Category, &im.Meaning, &im.VisualZh, &im.VisualEn, &emotions, &seasons,
		&times, &weathers, &eras, &styleIDs, &im.Status, &createdBy, &im.CreatedAt, &im.UpdatedAt}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return nil, err
	}
	im.Aliases, im.Emotions, im.Seasons = jsonStrings(aliases), jsonStrings(emotions), jsonStrings(seasons)
	im.Times, im.Weathers, im.Eras = jsonStrings(times), jsonStrings(weathers), jsonStrings(eras)
	if err := json.Unmarshal(styleIDs, &im.StyleIDs); err != nil || im.StyleIDs == nil {
		im.StyleIDs = []int64{}
	}
	im.CreatedBy = shihuaNullInt(createdBy)
	return &im, nil
}

func materialImageryLists(im *service.MaterialImagery) ([]any, error) {
	out := []any{}
	for _, v := range [][]string{im.Aliases, im.Emotions, im.Seasons, im.Times, im.Weathers, im.Eras} {
		b, err := json.Marshal(nonNilStrings(v))
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	ids := im.StyleIDs
	if ids == nil {
		ids = []int64{}
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	return append(out, b), nil
}

func (r *materialRepository) ListImagery(ctx context.Context, q service.MaterialImageryQuery) ([]service.MaterialImagery, int, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if q.Status != "" {
		where = append(where, "i.status = "+arg(q.Status))
	}
	if q.Category != "" {
		where = append(where, "i.category = "+arg(q.Category))
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		p := arg("%" + s + "%")
		where = append(where, "(i.name ILIKE "+p+" OR i.aliases::text ILIKE "+p+" OR i.meaning ILIKE "+p+")")
	}
	from := ` FROM mat_imagery i`
	if len(where) > 0 {
		from += " WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+materialImageryColumns+`,
       (SELECT COUNT(*) FROM mat_segment_imagery si WHERE si.imagery_id = i.id)`+from+
		` ORDER BY i.id LIMIT `+arg(q.Limit)+` OFFSET `+arg(q.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []service.MaterialImagery{}
	for rows.Next() {
		var n int
		im, err := scanMaterialImagery(rows, &n)
		if err != nil {
			return nil, 0, err
		}
		im.SegmentCount = n
		items = append(items, *im)
	}
	return items, total, rows.Err()
}

func (r *materialRepository) AllImagery(ctx context.Context) ([]service.MaterialImagery, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+materialImageryColumns+` FROM mat_imagery i ORDER BY i.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []service.MaterialImagery{}
	for rows.Next() {
		im, err := scanMaterialImagery(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *im)
	}
	return items, rows.Err()
}

func (r *materialRepository) CreateImagery(ctx context.Context, im *service.MaterialImagery) (bool, error) {
	lists, err := materialImageryLists(im)
	if err != nil {
		return false, err
	}
	args := append([]any{im.Name, im.Category, im.Meaning, im.VisualZh, im.VisualEn, im.Status, im.CreatedBy}, lists...)
	err = r.db.QueryRowContext(ctx, `
INSERT INTO mat_imagery (name, category, meaning, visual_zh, visual_en, status, created_by, aliases, emotions, seasons, times, weathers, eras, style_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (name) DO NOTHING
RETURNING id`, args...).Scan(&im.ID)
	if errors.Is(err, sql.ErrNoRows) {
		im.ID = 0
		return false, nil
	}
	return err == nil, err
}

func (r *materialRepository) UpdateImagery(ctx context.Context, im *service.MaterialImagery) (bool, error) {
	lists, err := materialImageryLists(im)
	if err != nil {
		return false, err
	}
	args := append([]any{im.ID, im.Name, im.Category, im.Meaning, im.VisualZh, im.VisualEn, im.Status}, lists...)
	res, err := r.db.ExecContext(ctx, `
UPDATE mat_imagery SET name = $2, category = $3, meaning = $4, visual_zh = $5, visual_en = $6, status = $7, aliases = $8,
       emotions = $9, seasons = $10, times = $11, weathers = $12, eras = $13, style_ids = $14, updated_at = NOW()
WHERE id = $1`, args...)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *materialRepository) SetImageryStatus(ctx context.Context, ids []int64, status string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE mat_imagery SET status = $2, updated_at = NOW() WHERE id = ANY($1)`, pq.Array(ids), status)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ---- segments ----

func (r *materialRepository) GetSegment(ctx context.Context, id int64) (*service.MaterialSegmentContext, error) {
	var s service.MaterialSegmentContext
	err := r.db.QueryRowContext(ctx, `
SELECT s.id, s.text_id, s.seq, s.label, s.content, s.scene_desc, s.is_famous, t.kind, t.title, COALESCE(a.name, ''),
       COALESCE(a.dynasty, ''), t.license_status, t.body
FROM mat_text_segments s
JOIN mat_texts t ON t.id = s.text_id
LEFT JOIN mat_authors a ON a.id = t.author_id
WHERE s.id = $1`, id).Scan(&s.ID, &s.TextID, &s.Seq, &s.Label, &s.Content, &s.SceneDesc, &s.IsFamous, &s.Kind, &s.Title,
		&s.AuthorName, &s.Dynasty, &s.LicenseStatus, &s.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *materialRepository) UpdateSegmentScene(ctx context.Context, id int64, scene string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE mat_text_segments SET scene_desc = $2, updated_at = NOW() WHERE id = $1`, id, scene)
	return err
}

func (r *materialRepository) ListSegmentImagery(ctx context.Context, segmentID int64) ([]service.MaterialSegmentImagery, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT si.segment_id, si.imagery_id, si.weight, si.source, si.confirmed, i.name, i.category, i.status, i.visual_zh
FROM mat_segment_imagery si JOIN mat_imagery i ON i.id = si.imagery_id
WHERE si.segment_id = $1
ORDER BY si.weight DESC, i.id`, segmentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	links := []service.MaterialSegmentImagery{}
	for rows.Next() {
		var l service.MaterialSegmentImagery
		if err := rows.Scan(&l.SegmentID, &l.ImageryID, &l.Weight, &l.Source, &l.Confirmed, &l.Name, &l.Category, &l.Status, &l.VisualZh); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (r *materialRepository) writeSegmentImagery(ctx context.Context, segmentID int64, links []service.MaterialSegmentImagery, clear string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM mat_segment_imagery WHERE segment_id = $1`+clear, segmentID); err != nil {
		return err
	}
	for _, l := range links {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO mat_segment_imagery (segment_id, imagery_id, weight, source, confirmed) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (segment_id, imagery_id) DO NOTHING`, segmentID, l.ImageryID, l.Weight, l.Source, l.Confirmed); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *materialRepository) ReplaceAISegmentImagery(ctx context.Context, segmentID int64, links []service.MaterialSegmentImagery) error {
	return r.writeSegmentImagery(ctx, segmentID, links, ` AND source = 'ai' AND NOT confirmed`)
}

func (r *materialRepository) SetSegmentImagery(ctx context.Context, segmentID int64, links []service.MaterialSegmentImagery) error {
	return r.writeSegmentImagery(ctx, segmentID, links, "")
}

// ---- artworks ----

const materialArtworkColumns = `a.id, a.segment_id, a.style_id, a.options, a.prompt_final, a.prompt_parts, a.model, a.size, a.quality,
a.status, a.attempts, a.not_before, a.error, a.file, a.width, a.height, a.cost_usd, a.duration_ms, a.parent_id, a.source,
a.rating, a.review_status, a.reject_reason, a.text_overlay_allowed, a.created_by, a.created_at, a.started_at, a.finished_at,
COALESCE((SELECT array_agg(ai.imagery_id ORDER BY ai.imagery_id) FROM mat_artwork_imagery ai WHERE ai.artwork_id = a.id), '{}')`

// materialArtworkJoined adds the style, segment and text a drawing belongs to.
const materialArtworkJoined = `, COALESCE(st.name, ''), s.text_id, COALESCE(t.title, ''), COALESCE(s.label, '')
FROM mat_artworks a
LEFT JOIN mat_styles st ON st.id = a.style_id
LEFT JOIN mat_text_segments s ON s.id = a.segment_id
LEFT JOIN mat_texts t ON t.id = s.text_id`

func scanMaterialArtwork(row shihuaScanner, joined bool) (*service.MaterialArtwork, error) {
	var a service.MaterialArtwork
	var segmentID, styleID, parentID, createdBy, textID sql.NullInt64
	var notBefore, startedAt, finishedAt sql.NullTime
	var options, parts []byte
	var imagery pq.Int64Array
	dest := []any{&a.ID, &segmentID, &styleID, &options, &a.PromptFinal, &parts, &a.Model, &a.Size, &a.Quality,
		&a.Status, &a.Attempts, &notBefore, &a.Error, &a.File, &a.Width, &a.Height, &a.CostUSD, &a.DurationMs, &parentID, &a.Source,
		&a.Rating, &a.ReviewStatus, &a.RejectReason, &a.TextOverlayAllowed, &createdBy, &a.CreatedAt, &startedAt, &finishedAt,
		&imagery}
	if joined {
		dest = append(dest, &a.StyleName, &textID, &a.TextTitle, &a.SegmentLabel)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	a.SegmentID, a.StyleID, a.ParentID = shihuaNullInt(segmentID), shihuaNullInt(styleID), shihuaNullInt(parentID)
	a.CreatedBy, a.TextID = shihuaNullInt(createdBy), shihuaNullInt(textID)
	a.NotBefore, a.StartedAt, a.FinishedAt = shihuaNullTime(notBefore), shihuaNullTime(startedAt), shihuaNullTime(finishedAt)
	a.Options, a.PromptParts = json.RawMessage(options), json.RawMessage(parts)
	a.ImageryIDs = []int64(imagery)
	if a.ImageryIDs == nil {
		a.ImageryIDs = []int64{}
	}
	return &a, nil
}

func materialJSONOr(raw json.RawMessage, def string) []byte {
	if len(raw) == 0 || string(raw) == "null" {
		return []byte(def)
	}
	return raw
}

func (r *materialRepository) EnqueueArtworks(ctx context.Context, arts []*service.MaterialArtwork) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, a := range arts {
		err := tx.QueryRowContext(ctx, `
INSERT INTO mat_artworks (segment_id, style_id, options, prompt_final, prompt_parts, model, size, quality, status, parent_id,
    source, review_status, text_overlay_allowed, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, created_at`,
			a.SegmentID, a.StyleID, materialJSONOr(a.Options, "{}"), a.PromptFinal, materialJSONOr(a.PromptParts, "{}"), a.Model, a.Size,
			a.Quality, a.Status, a.ParentID, a.Source, a.ReviewStatus, a.TextOverlayAllowed, a.CreatedBy).Scan(&a.ID, &a.CreatedAt)
		if err != nil {
			return err
		}
		if len(a.ImageryIDs) > 0 {
			// Imagery deleted since the prompt was built is skipped.
			if _, err := tx.ExecContext(ctx, `
INSERT INTO mat_artwork_imagery (artwork_id, imagery_id)
SELECT $1, i.id FROM mat_imagery i WHERE i.id = ANY($2)
ON CONFLICT DO NOTHING`, a.ID, pq.Array(a.ImageryIDs)); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *materialRepository) ClaimNextArtwork(ctx context.Context) (*service.MaterialArtwork, error) {
	a, err := scanMaterialArtwork(r.db.QueryRowContext(ctx, `
WITH next AS (
    SELECT id FROM mat_artworks
    WHERE status = 'queued' AND (not_before IS NULL OR not_before <= NOW())
    ORDER BY id
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE mat_artworks a SET status = 'running', attempts = a.attempts + 1, started_at = NOW(), error = ''
FROM next WHERE a.id = next.id
RETURNING `+materialArtworkColumns), false)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *materialRepository) FinishArtwork(ctx context.Context, a *service.MaterialArtwork) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE mat_artworks SET status = $2, not_before = $3, error = $4, file = $5, width = $6, height = $7, cost_usd = $8,
       duration_ms = $9, finished_at = $10
WHERE id = $1 AND status = 'running'`,
		a.ID, a.Status, a.NotBefore, a.Error, a.File, a.Width, a.Height, a.CostUSD, a.DurationMs, a.FinishedAt)
	return err
}

func (r *materialRepository) RequeueRunningArtworks(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE mat_artworks SET status = 'queued', not_before = NULL WHERE status = 'running'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *materialRepository) ArtworkQueueStats(ctx context.Context, dayStart time.Time) (service.MaterialQueueStats, error) {
	var s service.MaterialQueueStats
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status = 'queued'), COUNT(*) FILTER (WHERE status = 'running'),
       COUNT(*) FILTER (WHERE started_at >= $1)
FROM mat_artworks`, dayStart).Scan(&s.Queued, &s.Running, &s.StartedToday)
	return s, err
}

func (r *materialRepository) ListArtworks(ctx context.Context, q service.MaterialArtworkQuery) ([]service.MaterialArtwork, int, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if q.SegmentID > 0 {
		where = append(where, "a.segment_id = "+arg(q.SegmentID))
	}
	if q.TextID > 0 {
		where = append(where, "s.text_id = "+arg(q.TextID))
	}
	if q.Status != "" {
		where = append(where, "a.status = "+arg(q.Status))
	}
	if q.ReviewStatus != "" {
		where = append(where, "a.review_status = "+arg(q.ReviewStatus))
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mat_artworks a LEFT JOIN mat_text_segments s ON s.id = a.segment_id`+cond,
		args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+materialArtworkColumns+materialArtworkJoined+cond+
		` ORDER BY a.id DESC LIMIT `+arg(q.Limit)+` OFFSET `+arg(q.Offset), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := []service.MaterialArtwork{}
	for rows.Next() {
		a, err := scanMaterialArtwork(rows, true)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *a)
	}
	return items, total, rows.Err()
}

func (r *materialRepository) GetArtwork(ctx context.Context, id int64) (*service.MaterialArtwork, error) {
	a, err := scanMaterialArtwork(r.db.QueryRowContext(ctx, `SELECT `+materialArtworkColumns+materialArtworkJoined+` WHERE a.id = $1`, id), true)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return a, err
}

func (r *materialRepository) ReviewArtwork(ctx context.Context, id int64, status string, rating int, reason string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE mat_artworks SET review_status = $2, rating = $3, reject_reason = $4 WHERE id = $1`,
		id, status, rating, reason)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *materialRepository) CancelQueuedArtworks(ctx context.Context, ids []int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE mat_artworks SET status = 'canceled', not_before = NULL, finished_at = NOW()
WHERE id = ANY($1) AND status = 'queued'`, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *materialRepository) RetryArtwork(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE mat_artworks SET status = 'queued', attempts = 0, not_before = NULL, error = '', finished_at = NULL
WHERE id = $1 AND status IN ('failed', 'canceled')`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ---- posters ----

func (r *materialRepository) SaveArtworkRender(ctx context.Context, m *service.MaterialRender) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO mat_artwork_renders (artwork_id, aspect, file, width, height, layout)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (artwork_id, aspect) DO UPDATE SET file = EXCLUDED.file, width = EXCLUDED.width, height = EXCLUDED.height,
    layout = EXCLUDED.layout, created_at = NOW()
RETURNING id, created_at`, m.ArtworkID, m.Aspect, m.File, m.Width, m.Height, materialJSONOr(m.Layout, "{}")).Scan(&m.ID, &m.CreatedAt)
}

func (r *materialRepository) ListArtworkRenders(ctx context.Context, artworkIDs []int64) ([]service.MaterialRender, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, artwork_id, aspect, file, width, height, layout, created_at FROM mat_artwork_renders
WHERE artwork_id = ANY($1) ORDER BY artwork_id, aspect`, pq.Array(artworkIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.MaterialRender{}
	for rows.Next() {
		var m service.MaterialRender
		var layout []byte
		if err := rows.Scan(&m.ID, &m.ArtworkID, &m.Aspect, &m.File, &m.Width, &m.Height, &layout, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Layout = json.RawMessage(layout)
		out = append(out, m)
	}
	return out, rows.Err()
}
