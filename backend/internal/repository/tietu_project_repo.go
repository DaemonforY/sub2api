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

type tietuProjectRepository struct {
	db *sql.DB
}

// NewTietuProjectRepository creates the AI 做贴图 store (raw SQL, tietu_projects).
func NewTietuProjectRepository(db *sql.DB) service.TietuProjectRepository {
	return &tietuProjectRepository{db: db}
}

const tietuProjectColumns = `id, user_id, api_key_id, status, title, error, created_at, updated_at`

func (r *tietuProjectRepository) Create(ctx context.Context, p *service.TietuProject) error {
	data, err := json.Marshal(p.TietuData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO tietu_projects (user_id, api_key_id, status, title, data, error)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING id, created_at, updated_at`, p.UserID, p.KeyID, p.Status, p.Title, string(data), p.Error).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *tietuProjectRepository) Get(ctx context.Context, userID, id int64) (*service.TietuProject, error) {
	var p service.TietuProject
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT `+tietuProjectColumns+`, data FROM tietu_projects WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&p.ID, &p.UserID, &p.KeyID, &p.Status, &p.Title, &p.Error, &p.CreatedAt, &p.UpdatedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &p.TietuData); err != nil {
		return nil, err
	}
	return &p, nil
}

// List leaves out the heavy parts (plan, pictures, events): the editor opens one to see them.
func (r *tietuProjectRepository) List(ctx context.Context, userID int64, limit int) ([]service.TietuProject, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+tietuProjectColumns+`, data->'brief', COALESCE(data->>'pushed_at', '')
FROM tietu_projects WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.TietuProject{}
	for rows.Next() {
		var p service.TietuProject
		var brief []byte
		var pushed string
		if err := rows.Scan(&p.ID, &p.UserID, &p.KeyID, &p.Status, &p.Title, &p.Error, &p.CreatedAt, &p.UpdatedAt, &brief, &pushed); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(brief, &p.Brief)
		if t, err := time.Parse(time.RFC3339Nano, pushed); err == nil {
			p.PushedAt = &t
		}
		p.Suggestions, p.Covers, p.Panels, p.Events = []service.TietuSuggestion{}, []service.TietuImage{}, []service.TietuImage{}, []service.ArticleEvent{}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *tietuProjectRepository) Save(ctx context.Context, p *service.TietuProject) error {
	data, err := json.Marshal(p.TietuData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
UPDATE tietu_projects SET status = $3, title = $4, data = $5::jsonb, error = $6, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING updated_at`, p.ID, p.UserID, p.Status, p.Title, string(data), p.Error).Scan(&p.UpdatedAt)
}

func (r *tietuProjectRepository) CountActive(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tietu_projects WHERE user_id = $1 AND status IN ('suggesting', 'planning', 'covering', 'drawing', 'checking')`, userID).Scan(&n)
	return n, err
}

func (r *tietuProjectRepository) FailActive(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE tietu_projects SET status = 'failed', error = $1, updated_at = NOW(),
       data = jsonb_set(data, '{events}', COALESCE(data->'events', '[]'::jsonb) || jsonb_build_array(jsonb_build_object('at', NOW(), 'kind', 'error', 'text', $1::text)))
WHERE status IN ('suggesting', 'planning', 'covering', 'drawing', 'checking')`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

func (r *tietuProjectRepository) Delete(ctx context.Context, userID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tietu_projects WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *tietuProjectRepository) DeleteBefore(ctx context.Context, before time.Time) ([]int64, error) {
	var ids []int64
	err := r.db.QueryRowContext(ctx, `
WITH gone AS (DELETE FROM tietu_projects WHERE created_at < $1 AND status NOT IN ('suggesting', 'planning', 'covering', 'drawing', 'checking') RETURNING id)
SELECT COALESCE(array_agg(id), '{}') FROM gone`, before).Scan(pq.Array(&ids))
	return ids, err
}
