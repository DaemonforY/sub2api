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

type shihuaRepository struct {
	db *sql.DB
}

// NewShihuaRepository creates the 课本诗画 store (raw SQL).
func NewShihuaRepository(db *sql.DB) service.ShihuaRepository {
	return &shihuaRepository{db: db}
}

type shihuaScanner interface {
	Scan(dest ...any) error
}

// ---- works ----

const shihuaWorkColumns = `w.id, w.slug, w.title, w.author, w.dynasty, w.genre, w.stage, w.grade, w.textbooks, w.full_text,
w.display_text, w.excerpt_note, w.famous_lines, w.sources, w.variant_note, w.text_status, w.verify1_by, w.verify1_at,
w.verify2_by, w.verify2_at, w.theme, w.picked_image_id, w.sort, w.created_at, w.updated_at`

func scanShihuaWork(row shihuaScanner, extra ...any) (*service.ShihuaWork, error) {
	var w service.ShihuaWork
	var textbooks, theme []byte
	var v1By, v2By, picked sql.NullInt64
	var v1At, v2At sql.NullTime
	dest := []any{&w.ID, &w.Slug, &w.Title, &w.Author, &w.Dynasty, &w.Genre, &w.Stage, &w.Grade, &textbooks, &w.FullText,
		&w.DisplayText, &w.ExcerptNote, &w.FamousLines, &w.Sources, &w.VariantNote, &w.TextStatus, &v1By, &v1At,
		&v2By, &v2At, &theme, &picked, &w.Sort, &w.CreatedAt, &w.UpdatedAt}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(textbooks, &w.Textbooks); err != nil || w.Textbooks == nil {
		w.Textbooks = []service.ShihuaTextbook{}
	}
	_ = json.Unmarshal(theme, &w.Theme)
	w.Verify1By, w.Verify2By, w.PickedImageID = shihuaNullInt(v1By), shihuaNullInt(v2By), shihuaNullInt(picked)
	w.Verify1At, w.Verify2At = shihuaNullTime(v1At), shihuaNullTime(v2At)
	return &w, nil
}

func (r *shihuaRepository) ListWorks(ctx context.Context, q service.ShihuaWorkQuery) ([]service.ShihuaWork, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if q.Stage != "" {
		where = append(where, "w.stage = "+arg(q.Stage))
	}
	if q.Grade != "" {
		where = append(where, "w.grade = "+arg(q.Grade))
	}
	if q.Status != "" {
		where = append(where, "w.text_status = "+arg(q.Status))
	}
	if s := strings.TrimSpace(q.Q); s != "" {
		p := arg("%" + s + "%")
		where = append(where, "(w.title ILIKE "+p+" OR w.author ILIKE "+p+" OR w.slug ILIKE "+p+" OR w.full_text ILIKE "+p+")")
	}
	if q.IDs != nil {
		where = append(where, "w.id = ANY("+arg(pq.Array(q.IDs))+")")
	}
	query := `
SELECT ` + shihuaWorkColumns + `,
       COALESCE(i.total, 0), COALESCE(i.queued, 0), COALESCE(i.running, 0), COALESCE(i.succeeded, 0), COALESCE(i.failed, 0),
       COALESCE(rr.n, 0)
FROM shihua_works w
LEFT JOIN (
    SELECT work_id, COUNT(*) FILTER (WHERE status <> 'canceled') AS total,
           COUNT(*) FILTER (WHERE status = 'queued') AS queued, COUNT(*) FILTER (WHERE status = 'running') AS running,
           COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded, COUNT(*) FILTER (WHERE status = 'failed') AS failed
    FROM shihua_images GROUP BY work_id
) i ON i.work_id = w.id
LEFT JOIN (
    SELECT r.work_id, COUNT(*) AS n FROM shihua_renders r
    JOIN shihua_works pw ON pw.id = r.work_id AND pw.picked_image_id = r.image_id
    GROUP BY r.work_id
) rr ON rr.work_id = w.id`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY w.sort, w.id"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	works := []service.ShihuaWork{}
	for rows.Next() {
		var c service.ShihuaImageCounts
		var renders int
		w, err := scanShihuaWork(rows, &c.Total, &c.Queued, &c.Running, &c.Succeeded, &c.Failed, &renders)
		if err != nil {
			return nil, err
		}
		w.Images, w.Renders = c, renders
		works = append(works, *w)
	}
	return works, rows.Err()
}

func (r *shihuaRepository) getWork(ctx context.Context, where string, arg any) (*service.ShihuaWork, error) {
	w, err := scanShihuaWork(r.db.QueryRowContext(ctx, `SELECT `+shihuaWorkColumns+` FROM shihua_works w WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return w, err
}

func (r *shihuaRepository) GetWork(ctx context.Context, id int64) (*service.ShihuaWork, error) {
	return r.getWork(ctx, "w.id = $1", id)
}

func (r *shihuaRepository) GetWorkBySlug(ctx context.Context, slug string) (*service.ShihuaWork, error) {
	return r.getWork(ctx, "w.slug = $1", slug)
}

func shihuaWorkArgs(w *service.ShihuaWork) ([]any, error) {
	textbooks, err := json.Marshal(w.Textbooks)
	if err != nil {
		return nil, err
	}
	theme, err := json.Marshal(w.Theme)
	if err != nil {
		return nil, err
	}
	return []any{w.Slug, w.Title, w.Author, w.Dynasty, w.Genre, w.Stage, w.Grade, string(textbooks), w.FullText,
		w.DisplayText, w.ExcerptNote, w.FamousLines, w.Sources, w.VariantNote, w.TextStatus, w.Verify1By, w.Verify1At,
		w.Verify2By, w.Verify2At, string(theme), w.Sort}, nil
}

func (r *shihuaRepository) CreateWork(ctx context.Context, w *service.ShihuaWork) error {
	args, err := shihuaWorkArgs(w)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO shihua_works (slug, title, author, dynasty, genre, stage, grade, textbooks, full_text, display_text,
    excerpt_note, famous_lines, sources, variant_note, text_status, verify1_by, verify1_at, verify2_by, verify2_at, theme, sort)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20::jsonb, $21)
RETURNING id, created_at, updated_at`, args...).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *shihuaRepository) UpdateWork(ctx context.Context, w *service.ShihuaWork) error {
	args, err := shihuaWorkArgs(w)
	if err != nil {
		return err
	}
	args = append(args, w.ID)
	_, err = r.db.ExecContext(ctx, `
UPDATE shihua_works SET slug = $1, title = $2, author = $3, dynasty = $4, genre = $5, stage = $6, grade = $7,
    textbooks = $8::jsonb, full_text = $9, display_text = $10, excerpt_note = $11, famous_lines = $12, sources = $13,
    variant_note = $14, text_status = $15, verify1_by = $16, verify1_at = $17, verify2_by = $18, verify2_at = $19,
    theme = $20::jsonb, sort = $21, updated_at = NOW()
WHERE id = $22`, args...)
	return err
}

func (r *shihuaRepository) DeleteWork(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM shihua_works WHERE id = $1`, id)
	return err
}

func (r *shihuaRepository) SetPicked(ctx context.Context, workID int64, imageID *int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE shihua_works SET picked_image_id = $2, updated_at = NOW() WHERE id = $1`, workID, imageID)
	return err
}

// ---- jobs ----

func (r *shihuaRepository) CreateJob(ctx context.Context, name string, candidates int, createdBy int64, images []service.ShihuaImage) (*service.ShihuaJob, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	job := &service.ShihuaJob{Name: name, Candidates: candidates, CreatedBy: &createdBy}
	if err := tx.QueryRowContext(ctx, `INSERT INTO shihua_jobs (name, candidates, created_by) VALUES ($1, $2, $3) RETURNING id, created_at`,
		name, candidates, createdBy).Scan(&job.ID, &job.CreatedAt); err != nil {
		return nil, err
	}
	for i := range images {
		images[i].JobID = &job.ID
	}
	if err := insertShihuaImages(ctx, tx, images); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	works := map[int64]bool{}
	for _, img := range images {
		works[img.WorkID] = true
	}
	job.Works, job.Counts.Total, job.Counts.Queued = len(works), len(images), len(images)
	return job, nil
}

type shihuaExecer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func insertShihuaImages(ctx context.Context, db shihuaExecer, images []service.ShihuaImage) error {
	for i := range images {
		img := &images[i]
		if err := db.QueryRowContext(ctx, `
INSERT INTO shihua_images (work_id, job_id, status, prompt, size, quality) VALUES ($1, $2, 'queued', $3, $4, $5)
RETURNING id, created_at`, img.WorkID, img.JobID, img.Prompt, img.Size, img.Quality).Scan(&img.ID, &img.CreatedAt); err != nil {
			return err
		}
		img.Status = service.ShihuaImageQueued
	}
	return nil
}

func (r *shihuaRepository) EnqueueImages(ctx context.Context, images []service.ShihuaImage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertShihuaImages(ctx, tx, images); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *shihuaRepository) ListJobs(ctx context.Context, limit int) ([]service.ShihuaJob, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT j.id, j.name, j.candidates, j.created_by, j.created_at,
       COUNT(DISTINCT i.work_id), COUNT(i.id) FILTER (WHERE i.status <> 'canceled'),
       COUNT(i.id) FILTER (WHERE i.status = 'queued'), COUNT(i.id) FILTER (WHERE i.status = 'running'),
       COUNT(i.id) FILTER (WHERE i.status = 'succeeded'), COUNT(i.id) FILTER (WHERE i.status = 'failed'),
       COUNT(i.id) FILTER (WHERE i.status = 'canceled'), COALESCE(SUM(i.cost_usd), 0)
FROM shihua_jobs j LEFT JOIN shihua_images i ON i.job_id = j.id
GROUP BY j.id ORDER BY j.id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	jobs := []service.ShihuaJob{}
	for rows.Next() {
		var j service.ShihuaJob
		var by sql.NullInt64
		if err := rows.Scan(&j.ID, &j.Name, &j.Candidates, &by, &j.CreatedAt, &j.Works, &j.Counts.Total, &j.Counts.Queued,
			&j.Counts.Running, &j.Counts.Succeeded, &j.Counts.Failed, &j.Canceled, &j.CostUSD); err != nil {
			return nil, err
		}
		j.CreatedBy = shihuaNullInt(by)
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *shihuaRepository) exec(ctx context.Context, query string, args ...any) (int64, error) {
	out, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

func (r *shihuaRepository) CancelQueued(ctx context.Context, jobID int64) (int64, error) {
	return r.exec(ctx, `UPDATE shihua_images SET status = 'canceled', finished_at = NOW() WHERE status = 'queued' AND ($1 = 0 OR job_id = $1)`, jobID)
}

func (r *shihuaRepository) RetryFailed(ctx context.Context, jobID int64) (int64, error) {
	return r.exec(ctx, `
UPDATE shihua_images SET status = 'queued', attempts = 0, not_before = NULL, error = '', finished_at = NULL
WHERE status IN ('failed', 'canceled') AND ($1 = 0 OR job_id = $1)`, jobID)
}

const shihuaImageColumns = `id, work_id, job_id, status, attempts, not_before, prompt, size, quality, error, duration_ms,
cost_usd, file, width, height, upscale_status, upscale_error, upscaled_file, title_check, rejected, created_at, started_at, finished_at`

func scanShihuaImage(row shihuaScanner) (*service.ShihuaImage, error) {
	var img service.ShihuaImage
	var job sql.NullInt64
	var notBefore, started, finished sql.NullTime
	if err := row.Scan(&img.ID, &img.WorkID, &job, &img.Status, &img.Attempts, &notBefore, &img.Prompt, &img.Size, &img.Quality,
		&img.Error, &img.DurationMs, &img.CostUSD, &img.File, &img.Width, &img.Height, &img.UpscaleStatus, &img.UpscaleError,
		&img.UpscaledFile, &img.TitleCheck, &img.Rejected, &img.CreatedAt, &started, &finished); err != nil {
		return nil, err
	}
	img.JobID, img.NotBefore, img.StartedAt, img.FinishedAt = shihuaNullInt(job), shihuaNullTime(notBefore), shihuaNullTime(started), shihuaNullTime(finished)
	return &img, nil
}

func (r *shihuaRepository) claim(ctx context.Context, query string) (*service.ShihuaImage, error) {
	img, err := scanShihuaImage(r.db.QueryRowContext(ctx, query))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return img, err
}

func (r *shihuaRepository) ClaimNext(ctx context.Context) (*service.ShihuaImage, error) {
	return r.claim(ctx, `
UPDATE shihua_images SET status = 'running', attempts = attempts + 1, started_at = NOW(), not_before = NULL
WHERE id = (
    SELECT id FROM shihua_images WHERE status = 'queued' AND (not_before IS NULL OR not_before <= NOW())
    ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING `+shihuaImageColumns)
}

func (r *shihuaRepository) FinishImage(ctx context.Context, img *service.ShihuaImage) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE shihua_images SET status = $2, not_before = $3, error = $4, duration_ms = $5, cost_usd = $6, file = $7, width = $8,
    height = $9, finished_at = $10
WHERE id = $1 AND status = 'running'`,
		img.ID, img.Status, img.NotBefore, img.Error, img.DurationMs, img.CostUSD, img.File, img.Width, img.Height, img.FinishedAt)
	return err
}

func (r *shihuaRepository) RequeueRunning(ctx context.Context) (int64, error) {
	return r.exec(ctx, `UPDATE shihua_images SET status = 'queued', error = '服务器重启，重新排队（Requeued after a restart）' WHERE status = 'running'`)
}

func (r *shihuaRepository) QueueStats(ctx context.Context, dayStart time.Time) (service.ShihuaQueueStats, error) {
	var st service.ShihuaQueueStats
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status = 'queued'), COUNT(*) FILTER (WHERE status = 'running'),
       COUNT(*) FILTER (WHERE started_at >= $1), COUNT(*) FILTER (WHERE upscale_status IN ('queued', 'running'))
FROM shihua_images`, dayStart).Scan(&st.Queued, &st.Running, &st.StartedToday, &st.UpscaleQueue)
	return st, err
}

func (r *shihuaRepository) ListImages(ctx context.Context, q service.ShihuaImageQuery) ([]service.ShihuaImage, error) {
	var where []string
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if q.WorkIDs != nil {
		where = append(where, "work_id = ANY("+arg(pq.Array(q.WorkIDs))+")")
	}
	if q.JobID > 0 {
		where = append(where, "job_id = "+arg(q.JobID))
	}
	if q.Status != "" {
		where = append(where, "status = "+arg(q.Status))
	}
	query := `SELECT ` + shihuaImageColumns + ` FROM shihua_images`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY work_id, id LIMIT " + arg(q.Limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	images := []service.ShihuaImage{}
	for rows.Next() {
		img, err := scanShihuaImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, *img)
	}
	return images, rows.Err()
}

func (r *shihuaRepository) GetImage(ctx context.Context, id int64) (*service.ShihuaImage, error) {
	img, err := scanShihuaImage(r.db.QueryRowContext(ctx, `SELECT `+shihuaImageColumns+` FROM shihua_images WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return img, err
}

func (r *shihuaRepository) ReviewImage(ctx context.Context, id int64, rejected bool, titleCheck string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE shihua_images SET rejected = $2, title_check = $3 WHERE id = $1`, id, rejected, titleCheck)
	return err
}

func (r *shihuaRepository) SetUpscale(ctx context.Context, id int64, status, errText, file string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE shihua_images SET upscale_status = $2, upscale_error = $3, upscaled_file = $4 WHERE id = $1`,
		id, status, errText, file)
	return err
}

func (r *shihuaRepository) ClaimUpscale(ctx context.Context) (*service.ShihuaImage, error) {
	return r.claim(ctx, `
UPDATE shihua_images SET upscale_status = 'running', upscale_error = ''
WHERE id = (
    SELECT id FROM shihua_images WHERE upscale_status = 'queued' AND status = 'succeeded'
    ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
)
RETURNING `+shihuaImageColumns)
}

func (r *shihuaRepository) RequeueUpscaleRunning(ctx context.Context) (int64, error) {
	return r.exec(ctx, `UPDATE shihua_images SET upscale_status = 'queued' WHERE upscale_status = 'running'`)
}

// ---- renders ----

func (r *shihuaRepository) SaveRender(ctx context.Context, rd *service.ShihuaRender) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO shihua_renders (image_id, work_id, aspect, file, width, height, layout)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (image_id, aspect) DO UPDATE SET file = EXCLUDED.file, width = EXCLUDED.width, height = EXCLUDED.height,
    layout = EXCLUDED.layout, created_at = NOW()
RETURNING id, created_at`, rd.ImageID, rd.WorkID, rd.Aspect, rd.File, rd.Width, rd.Height, string(rd.Layout)).Scan(&rd.ID, &rd.CreatedAt)
}

func (r *shihuaRepository) ListRenders(ctx context.Context, workIDs []int64) ([]service.ShihuaRender, error) {
	query := `SELECT id, image_id, work_id, aspect, file, width, height, layout, created_at FROM shihua_renders`
	var args []any
	if workIDs != nil {
		query += ` WHERE work_id = ANY($1)`
		args = append(args, pq.Array(workIDs))
	}
	rows, err := r.db.QueryContext(ctx, query+` ORDER BY work_id, id`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ShihuaRender{}
	for rows.Next() {
		var rd service.ShihuaRender
		var layout []byte
		if err := rows.Scan(&rd.ID, &rd.ImageID, &rd.WorkID, &rd.Aspect, &rd.File, &rd.Width, &rd.Height, &layout, &rd.CreatedAt); err != nil {
			return nil, err
		}
		rd.Layout = json.RawMessage(layout)
		out = append(out, rd)
	}
	return out, rows.Err()
}

func shihuaNullInt(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func shihuaNullTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
