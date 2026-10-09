package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type articleProjectRepository struct {
	db *sql.DB
}

// NewArticleProjectRepository creates the AI 写文章 store (raw SQL, article_projects).
func NewArticleProjectRepository(db *sql.DB) service.ArticleProjectRepository {
	return &articleProjectRepository{db: db}
}

const articleProjectColumns = `id, user_id, api_key_id, status, title, error, created_at, updated_at`

func (r *articleProjectRepository) Create(ctx context.Context, p *service.ArticleProject) error {
	data, err := json.Marshal(p.ArticleData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO article_projects (user_id, api_key_id, status, title, data, error)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING id, created_at, updated_at`, p.UserID, p.KeyID, p.Status, p.Title, string(data), p.Error).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *articleProjectRepository) Get(ctx context.Context, userID, id int64) (*service.ArticleProject, error) {
	var p service.ArticleProject
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT `+articleProjectColumns+`, data FROM article_projects WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&p.ID, &p.UserID, &p.KeyID, &p.Status, &p.Title, &p.Error, &p.CreatedAt, &p.UpdatedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &p.ArticleData); err != nil {
		return nil, err
	}
	return &p, nil
}

// List leaves out the heavy parts (Markdown, sources, events): the editor opens one to see them.
func (r *articleProjectRepository) List(ctx context.Context, userID int64, limit int) ([]service.ArticleProject, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+articleProjectColumns+`, data->'brief', COALESCE(data->>'pushed_at', '')
FROM article_projects WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.ArticleProject{}
	for rows.Next() {
		var p service.ArticleProject
		var brief []byte
		var pushed string
		if err := rows.Scan(&p.ID, &p.UserID, &p.KeyID, &p.Status, &p.Title, &p.Error, &p.CreatedAt, &p.UpdatedAt, &brief, &pushed); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(brief, &p.Brief)
		if t, err := time.Parse(time.RFC3339Nano, pushed); err == nil {
			p.PushedAt = &t
		}
		p.Sources, p.Images, p.Events = []service.ArticleSource{}, []service.ArticleImage{}, []service.ArticleEvent{}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *articleProjectRepository) Save(ctx context.Context, p *service.ArticleProject) error {
	data, err := json.Marshal(p.ArticleData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
UPDATE article_projects SET status = $3, title = $4, data = $5::jsonb, error = $6, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING updated_at`, p.ID, p.UserID, p.Status, p.Title, string(data), p.Error).Scan(&p.UpdatedAt)
}

func (r *articleProjectRepository) CountActive(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM article_projects WHERE user_id = $1 AND status IN ('outlining', 'writing', 'drawing')`, userID).Scan(&n)
	return n, err
}

func (r *articleProjectRepository) FailActive(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE article_projects SET status = 'failed', error = $1, updated_at = NOW(),
       data = jsonb_set(data, '{events}', COALESCE(data->'events', '[]'::jsonb) || jsonb_build_array(jsonb_build_object('at', NOW(), 'kind', 'error', 'text', $1::text)))
WHERE status IN ('outlining', 'writing', 'drawing')`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

func (r *articleProjectRepository) Delete(ctx context.Context, userID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM article_projects WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *articleProjectRepository) DeleteBefore(ctx context.Context, before time.Time) ([]int64, error) {
	var ids []int64
	err := r.db.QueryRowContext(ctx, `
WITH gone AS (DELETE FROM article_projects WHERE created_at < $1 AND status NOT IN ('outlining', 'writing', 'drawing') RETURNING id)
SELECT COALESCE(array_agg(id), '{}') FROM gone`, before).Scan(pq.Array(&ids))
	return ids, err
}
