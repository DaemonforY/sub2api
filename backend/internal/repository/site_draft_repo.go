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

type siteDraftRepository struct {
	db *sql.DB
}

// NewSiteDraftRepository creates the AI 建站 store (raw SQL, site_drafts).
func NewSiteDraftRepository(db *sql.DB) service.SiteDraftRepository {
	return &siteDraftRepository{db: db}
}

const siteDraftColumns = `id, user_id, api_key_id, status, title, error, created_at, updated_at`

func (r *siteDraftRepository) Create(ctx context.Context, d *service.SiteDraft) error {
	data, err := json.Marshal(d.SiteDraftData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
INSERT INTO site_drafts (user_id, api_key_id, status, title, data, error)
VALUES ($1, $2, $3, $4, $5::jsonb, $6)
RETURNING id, created_at, updated_at`, d.UserID, d.KeyID, d.Status, d.Title, string(data), d.Error).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *siteDraftRepository) Get(ctx context.Context, userID, id int64) (*service.SiteDraft, error) {
	var d service.SiteDraft
	var data []byte
	err := r.db.QueryRowContext(ctx, `SELECT `+siteDraftColumns+`, data FROM site_drafts WHERE id = $1 AND user_id = $2`, id, userID).
		Scan(&d.ID, &d.UserID, &d.KeyID, &d.Status, &d.Title, &d.Error, &d.CreatedAt, &d.UpdatedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &d.SiteDraftData); err != nil {
		return nil, err
	}
	return &d, nil
}

// List leaves out the heavy parts (pages, events): the page opens one to see them.
func (r *siteDraftRepository) List(ctx context.Context, userID int64, limit int) ([]service.SiteDraft, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT `+siteDraftColumns+`, data->'brief', COALESCE(data->>'site_url', ''), COALESCE((data->>'site_id')::bigint, 0)
FROM site_drafts WHERE user_id = $1 ORDER BY updated_at DESC, id DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.SiteDraft{}
	for rows.Next() {
		var d service.SiteDraft
		var brief []byte
		if err := rows.Scan(&d.ID, &d.UserID, &d.KeyID, &d.Status, &d.Title, &d.Error, &d.CreatedAt, &d.UpdatedAt, &brief, &d.SiteURL, &d.SiteID); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(brief, &d.Brief)
		d.Turns, d.Images, d.Events = []service.SiteDraftTurn{}, []service.SiteDraftImage{}, []service.ArticleEvent{}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *siteDraftRepository) Save(ctx context.Context, d *service.SiteDraft) error {
	data, err := json.Marshal(d.SiteDraftData)
	if err != nil {
		return err
	}
	return r.db.QueryRowContext(ctx, `
UPDATE site_drafts SET status = $3, title = $4, data = $5::jsonb, error = $6, updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING updated_at`, d.ID, d.UserID, d.Status, d.Title, string(data), d.Error).Scan(&d.UpdatedAt)
}

func (r *siteDraftRepository) CountActive(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_drafts WHERE user_id = $1 AND status IN ('generating', 'drawing')`, userID).Scan(&n)
	return n, err
}

// FailActive settles runs a previous process left behind: a draft that already has a page goes back
// to ready (the page is still usable), one without is failed.
func (r *siteDraftRepository) FailActive(ctx context.Context, message string) (int64, error) {
	out, err := r.db.ExecContext(ctx, `
UPDATE site_drafts SET
       status = CASE WHEN COALESCE(data->>'html', '') = '' THEN 'failed' ELSE 'ready' END,
       error = $1, updated_at = NOW(),
       data = jsonb_set(data - 'draft_html', '{events}', COALESCE(data->'events', '[]'::jsonb) || jsonb_build_array(jsonb_build_object('at', NOW(), 'kind', 'error', 'text', $1::text)))
WHERE status IN ('generating', 'drawing')`, message)
	if err != nil {
		return 0, err
	}
	return out.RowsAffected()
}

func (r *siteDraftRepository) Delete(ctx context.Context, userID, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM site_drafts WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *siteDraftRepository) DeleteBefore(ctx context.Context, before time.Time) ([]int64, error) {
	var ids []int64
	err := r.db.QueryRowContext(ctx, `
WITH gone AS (DELETE FROM site_drafts WHERE updated_at < $1 AND status NOT IN ('generating', 'drawing') RETURNING id)
SELECT COALESCE(array_agg(id), '{}') FROM gone`, before).Scan(pq.Array(&ids))
	return ids, err
}
