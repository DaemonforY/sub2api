package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelLinkRepository struct {
	db *sql.DB
}

// NewChannelLinkRepository stores 渠道链接 and counts each link's funnel from the analytics first
// touch (utm_campaign = the link's code), users, usage_logs and payment_orders (raw SQL).
func NewChannelLinkRepository(db *sql.DB) service.ChannelLinkRepository {
	return &channelLinkRepository{db: db}
}

const channelLinkColumns = `id, code, name, source, medium, target_path, aff_code, note, clicks, created_at, updated_at`

func scanChannelLink(row interface{ Scan(...any) error }, l *service.ChannelLink, extra ...any) error {
	return row.Scan(append([]any{&l.ID, &l.Code, &l.Name, &l.Source, &l.Medium, &l.TargetPath, &l.AffCode, &l.Note,
		&l.Clicks, &l.CreatedAt, &l.UpdatedAt}, extra...)...)
}

func (r *channelLinkRepository) List(ctx context.Context) ([]service.ChannelLink, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH su AS (
  SELECT a.campaign, u.id,
    EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = u.id) AS activated
  FROM user_attributions a JOIN users u ON u.id = a.user_id
  WHERE a.campaign <> '' AND u.role <> 'admin' AND u.deleted_at IS NULL
), paid AS (
  SELECT su.campaign, o.user_id, o.amount FROM su JOIN payment_orders o ON o.user_id = su.id
  WHERE o.status = 'COMPLETED' AND o.payment_type <> 'balance'
)
SELECT `+channelLinkColumns+`,
  (SELECT COUNT(DISTINCT e.visitor_id) FROM analytics_events e WHERE e.campaign = c.code),
  (SELECT COUNT(*) FROM su WHERE su.campaign = c.code),
  (SELECT COUNT(*) FROM su WHERE su.campaign = c.code AND su.activated),
  (SELECT COUNT(DISTINCT p.user_id) FROM paid p WHERE p.campaign = c.code),
  (SELECT COALESCE(SUM(p.amount), 0)::float8 FROM paid p WHERE p.campaign = c.code)
FROM channel_links c ORDER BY c.created_at DESC, c.id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.ChannelLink
	for rows.Next() {
		var l service.ChannelLink
		if err := scanChannelLink(rows, &l, &l.Visitors, &l.Signups, &l.Activated, &l.PaidUsers, &l.Revenue); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *channelLinkRepository) GetByCode(ctx context.Context, code string) (*service.ChannelLink, error) {
	var l service.ChannelLink
	err := scanChannelLink(r.db.QueryRowContext(ctx, `SELECT `+channelLinkColumns+` FROM channel_links WHERE code = $1`, code), &l)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *channelLinkRepository) Create(ctx context.Context, l *service.ChannelLink) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO channel_links (code, name, source, medium, target_path, aff_code, note)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, clicks, created_at, updated_at`,
		l.Code, l.Name, l.Source, l.Medium, l.TargetPath, l.AffCode, l.Note,
	).Scan(&l.ID, &l.Clicks, &l.CreatedAt, &l.UpdatedAt)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrChannelLinkCodeTaken
	}
	return err
}

func (r *channelLinkRepository) Update(ctx context.Context, l *service.ChannelLink) error {
	err := r.db.QueryRowContext(ctx, `
UPDATE channel_links SET name = $2, source = $3, medium = $4, target_path = $5, aff_code = $6, note = $7, updated_at = NOW()
WHERE id = $1
RETURNING code, clicks, created_at, updated_at`,
		l.ID, l.Name, l.Source, l.Medium, l.TargetPath, l.AffCode, l.Note,
	).Scan(&l.Code, &l.Clicks, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrChannelLinkNotFound
	}
	return err
}

func (r *channelLinkRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM channel_links WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrChannelLinkNotFound
	}
	return nil
}

func (r *channelLinkRepository) AddClick(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE channel_links SET clicks = clicks + 1 WHERE id = $1`, id)
	return err
}
