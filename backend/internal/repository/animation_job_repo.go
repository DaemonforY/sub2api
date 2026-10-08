package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type animationJobRepository struct {
	db *sql.DB
}

// NewAnimationJobRepository creates the canvas AI 动画 job store (raw SQL).
func NewAnimationJobRepository(db *sql.DB) service.AnimationJobRepository {
	return &animationJobRepository{db: db}
}

const animationJobColumns = `id, status, model, meta, error, chars, prompt_tokens, completion_tokens, created_at, started_at, finished_at`

func (r *animationJobRepository) Create(ctx context.Context, userID, apiKeyID int64, model string, meta json.RawMessage) (*service.AnimationJob, error) {
	return scanAnimationJob(r.db.QueryRowContext(ctx, `
INSERT INTO animation_jobs (user_id, api_key_id, model, meta)
VALUES ($1, $2, $3, $4::jsonb)
RETURNING `+animationJobColumns, userID, apiKeyID, model, string(meta)), false)
}

func (r *animationJobRepository) CountActive(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM animation_jobs WHERE user_id = $1 AND status IN ('pending', 'running')`, userID).Scan(&n)
	return n, err
}

func (r *animationJobRepository) Get(ctx context.Context, userID, id int64) (*service.AnimationJob, error) {
	job, err := scanAnimationJob(r.db.QueryRowContext(ctx,
		`SELECT `+animationJobColumns+`, svg FROM animation_jobs WHERE id = $1 AND user_id = $2`, id, userID), true)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *animationJobRepository) List(ctx context.Context, userID int64, limit int) ([]service.AnimationJob, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+animationJobColumns+` FROM animation_jobs WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	jobs := []service.AnimationJob{}
	for rows.Next() {
		job, err := scanAnimationJob(rows, false)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

func (r *animationJobRepository) MarkRunning(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE animation_jobs SET status = 'running', started_at = NOW() WHERE id = $1 AND status = 'pending'`, id)
	return err
}

func (r *animationJobRepository) Finish(ctx context.Context, id int64, res service.AnimationJobResult) (bool, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE animation_jobs
SET status = $2, svg = $3, error = $4, chars = $5, prompt_tokens = $6, completion_tokens = $7, finished_at = NOW()
WHERE id = $1 AND status IN ('pending', 'running')`,
		id, res.Status, res.SVG, res.Error, res.Chars, res.PromptTokens, res.CompletionTokens)
	if err != nil {
		return false, err
	}
	n, err := out.RowsAffected()
	return n > 0, err
}

func (r *animationJobRepository) FailActive(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `UPDATE animation_jobs SET status = 'failed', error = $1, finished_at = NOW() WHERE status IN ('pending', 'running')`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

func (r *animationJobRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	out, err := r.db.ExecContext(ctx, `DELETE FROM animation_jobs WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

type animationJobScanner interface {
	Scan(dest ...any) error
}

func scanAnimationJob(row animationJobScanner, withSVG bool) (*service.AnimationJob, error) {
	var job service.AnimationJob
	var meta []byte
	var started, finished sql.NullTime
	dest := []any{&job.ID, &job.Status, &job.Model, &meta, &job.Error, &job.Chars, &job.PromptTokens, &job.CompletionTokens, &job.CreatedAt, &started, &finished}
	if withSVG {
		dest = append(dest, &job.SVG)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	job.Meta = json.RawMessage(meta)
	if started.Valid {
		job.StartedAt = &started.Time
	}
	if finished.Valid {
		job.FinishedAt = &finished.Time
	}
	return &job, nil
}
