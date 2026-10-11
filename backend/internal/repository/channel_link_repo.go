package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelLinkRepository struct {
	db *sql.DB
}

// NewChannelLinkRepository stores 渠道链接 and counts each link's funnel and money from the
// analytics first touch (utm_campaign = the link's code), the invitee list of the link's invite
// code, users, usage_logs, payment_orders and the affiliate ledger (raw SQL).
func NewChannelLinkRepository(db *sql.DB) service.ChannelLinkRepository {
	return &channelLinkRepository{db: db}
}

const channelLinkColumns = `id, code, name, source, medium, target_path, aff_code, promo_code, note, clicks, created_at, updated_at`

func scanChannelLink(row interface{ Scan(...any) error }, l *service.ChannelLink, extra ...any) error {
	return row.Scan(append([]any{&l.ID, &l.Code, &l.Name, &l.Source, &l.Medium, &l.TargetPath, &l.AffCode, &l.PromoCode, &l.Note,
		&l.Clicks, &l.CreatedAt, &l.UpdatedAt}, extra...)...)
}

// List: a link's people (m) are the members whose first touch was the link, plus those invited by
// the owner of the link's invite code. Money is in CNY: orders by what was paid less the payment
// fee; a rebate by the same paid-per-credited ratio of the order it came from (as withdrawals value
// it), at face value when it has no order (redeem codes).
func (r *channelLinkRepository) List(ctx context.Context, since time.Time, affCode string) ([]service.ChannelLink, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH coh AS (
  SELECT c.id AS link_id, a.user_id FROM channel_links c JOIN user_attributions a ON a.campaign = c.code
  UNION
  SELECT c.id, x.user_id FROM channel_links c
  JOIN user_affiliates o ON c.aff_code <> '' AND o.aff_code = c.aff_code
  JOIN user_affiliates x ON x.inviter_id = o.user_id
), m AS (
  SELECT coh.link_id, u.id, u.created_at FROM coh
  JOIN users u ON u.id = coh.user_id AND u.role <> 'admin' AND u.deleted_at IS NULL
), paid AS (
  SELECT m.link_id, o.user_id, o.pay_amount / (1 + COALESCE(o.fee_rate, 0) / 100) AS cny
  FROM m JOIN payment_orders o ON o.user_id = m.id
  WHERE o.status = 'COMPLETED' AND o.payment_type <> 'balance'
    AND COALESCE(o.paid_at, o.completed_at, o.created_at) >= $1
)
SELECT `+channelLinkColumns+`,
  (SELECT COUNT(DISTINCT e.visitor_id) FROM analytics_events e WHERE e.campaign = c.code AND e.created_at >= $1),
  (SELECT COUNT(*) FROM m WHERE m.link_id = c.id AND m.created_at >= $1),
  (SELECT COUNT(*) FROM m WHERE m.link_id = c.id AND m.created_at >= $1
     AND EXISTS (SELECT 1 FROM usage_logs l WHERE l.user_id = m.id)),
  (SELECT COUNT(DISTINCT p.user_id) FROM paid p WHERE p.link_id = c.id),
  (SELECT COALESCE(SUM(p.cny), 0)::float8 FROM paid p WHERE p.link_id = c.id),
  (SELECT COALESCE(SUM(l.total_cost), 0)::float8 FROM m JOIN usage_logs l ON l.user_id = m.id
     WHERE m.link_id = c.id AND l.created_at >= $1),
  (SELECT COALESCE(SUM(CASE WHEN o.id IS NOT NULL AND o.amount > 0
       THEN g.amount * o.pay_amount / (1 + COALESCE(o.fee_rate, 0) / 100) / o.amount
       ELSE g.amount END), 0)::float8
     FROM m JOIN user_affiliate_ledger g ON g.source_user_id = m.id AND g.action = 'accrue'
     LEFT JOIN payment_orders o ON o.id = g.source_order_id
     WHERE m.link_id = c.id AND g.created_at >= $1)
FROM channel_links c WHERE ($2 = '' OR c.aff_code = $2)
ORDER BY c.created_at DESC, c.id DESC`, since, affCode)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.ChannelLink
	for rows.Next() {
		var l service.ChannelLink
		if err := scanChannelLink(rows, &l, &l.Visitors, &l.Signups, &l.Activated, &l.PaidUsers, &l.Revenue, &l.UsageUSD, &l.Rebates); err != nil {
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
INSERT INTO channel_links (code, name, source, medium, target_path, aff_code, promo_code, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, clicks, created_at, updated_at`,
		l.Code, l.Name, l.Source, l.Medium, l.TargetPath, l.AffCode, l.PromoCode, l.Note,
	).Scan(&l.ID, &l.Clicks, &l.CreatedAt, &l.UpdatedAt)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return service.ErrChannelLinkCodeTaken
	}
	return err
}

func (r *channelLinkRepository) Update(ctx context.Context, l *service.ChannelLink) error {
	err := r.db.QueryRowContext(ctx, `
UPDATE channel_links SET name = $2, source = $3, medium = $4, target_path = $5, aff_code = $6, promo_code = $7, note = $8, updated_at = NOW()
WHERE id = $1
RETURNING code, clicks, created_at, updated_at`,
		l.ID, l.Name, l.Source, l.Medium, l.TargetPath, l.AffCode, l.PromoCode, l.Note,
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
