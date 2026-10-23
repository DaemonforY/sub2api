package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type cardGoodsRepository struct {
	db *sql.DB
}

// NewCardGoodsRepository creates the 卡密商品 store (raw SQL, migration 301).
func NewCardGoodsRepository(db *sql.DB) service.CardGoodsRepository {
	return &cardGoodsRepository{db: db}
}

const cardGoodsColumns = `id, slug, name, subtitle, description, usage_note, price::float8, original_price::float8,
       per_user_limit, enabled, sort_order, created_at, updated_at`

// cardGoodsOpenOrder: an order (alias o) that will take a key — pending and not expired, or paid
// and not delivered yet. Expired / cancelled ones, and orders whose provider call failed before
// payment (FAILED without paid_at), release their key.
const cardGoodsOpenOrder = `o.order_type = 'goods'
  AND (o.status IN ('PAID', 'RECHARGING')
       OR (o.status = 'FAILED' AND o.paid_at IS NOT NULL)
       OR (o.status = 'PENDING' AND o.expires_at > NOW()))
  AND NOT EXISTS (SELECT 1 FROM card_goods_items ci WHERE ci.order_id = o.id)`

func scanCardGoods(row interface{ Scan(...any) error }) (*service.CardGoods, error) {
	var g service.CardGoods
	err := row.Scan(&g.ID, &g.Slug, &g.Name, &g.Subtitle, &g.Description, &g.UsageNote, &g.Price, &g.OriginalPrice,
		&g.PerUserLimit, &g.Enabled, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *cardGoodsRepository) List(ctx context.Context, enabledOnly bool) ([]service.CardGoods, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+cardGoodsColumns+` FROM card_goods
WHERE ($1 = FALSE OR enabled) ORDER BY sort_order, id`, enabledOnly)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.CardGoods{}
	for rows.Next() {
		g, err := scanCardGoods(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func (r *cardGoodsRepository) getOne(ctx context.Context, where string, arg any) (*service.CardGoods, error) {
	g, err := scanCardGoods(r.db.QueryRowContext(ctx, `SELECT `+cardGoodsColumns+` FROM card_goods WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return g, err
}

func (r *cardGoodsRepository) GetByID(ctx context.Context, id int64) (*service.CardGoods, error) {
	return r.getOne(ctx, `id = $1`, id)
}

func (r *cardGoodsRepository) GetBySlug(ctx context.Context, slug string) (*service.CardGoods, error) {
	return r.getOne(ctx, `slug = $1`, slug)
}

func (r *cardGoodsRepository) SlugTaken(ctx context.Context, slug string, exceptID int64) (bool, error) {
	var taken bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM card_goods WHERE slug = $1 AND id <> $2)`, slug, exceptID).Scan(&taken)
	return taken, err
}

func (r *cardGoodsRepository) Create(ctx context.Context, g *service.CardGoods) error {
	return r.db.QueryRowContext(ctx, `
INSERT INTO card_goods (slug, name, subtitle, description, usage_note, price, original_price, per_user_limit, enabled, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, created_at, updated_at`,
		g.Slug, g.Name, g.Subtitle, g.Description, g.UsageNote, g.Price, g.OriginalPrice, g.PerUserLimit, g.Enabled, g.SortOrder,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt)
}

func (r *cardGoodsRepository) Update(ctx context.Context, g *service.CardGoods) error {
	return r.db.QueryRowContext(ctx, `
UPDATE card_goods SET slug = $2, name = $3, subtitle = $4, description = $5, usage_note = $6, price = $7,
       original_price = $8, per_user_limit = $9, enabled = $10, sort_order = $11, updated_at = NOW()
WHERE id = $1
RETURNING created_at, updated_at`,
		g.ID, g.Slug, g.Name, g.Subtitle, g.Description, g.UsageNote, g.Price, g.OriginalPrice, g.PerUserLimit, g.Enabled, g.SortOrder,
	).Scan(&g.CreatedAt, &g.UpdatedAt)
}

func (r *cardGoodsRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM card_goods WHERE id = $1`, id)
	return err
}

func (r *cardGoodsRepository) Stock(ctx context.Context, goodsID int64) (service.CardGoodsStock, error) {
	var s service.CardGoodsStock
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FILTER (WHERE status = 'available'),
       COUNT(*) FILTER (WHERE status = 'sold'),
       COUNT(*) FILTER (WHERE status = 'void'),
       (SELECT COUNT(*) FROM payment_orders o WHERE o.goods_id = $1 AND `+cardGoodsOpenOrder+`)
FROM card_goods_items WHERE goods_id = $1`, goodsID).Scan(&s.Available, &s.Sold, &s.Void, &s.Reserved)
	return s, err
}

func (r *cardGoodsRepository) UserOpenCount(ctx context.Context, userID, goodsID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
SELECT (SELECT COUNT(*) FROM card_goods_items WHERE goods_id = $2 AND user_id = $1 AND status = 'sold')
     + (SELECT COUNT(*) FROM payment_orders o WHERE o.user_id = $1 AND o.goods_id = $2 AND `+cardGoodsOpenOrder+`)`,
		userID, goodsID).Scan(&n)
	return n, err
}

func (r *cardGoodsRepository) InsertItems(ctx context.Context, goodsID int64, items []service.CardGoodsNewItem, batchNote string) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	encs := make([]string, len(items))
	hashes := make([]string, len(items))
	previews := make([]string, len(items))
	for i, it := range items {
		encs[i], hashes[i], previews[i] = it.ContentEnc, it.Hash, it.Preview
	}
	res, err := r.db.ExecContext(ctx, `
INSERT INTO card_goods_items (goods_id, content_enc, content_hash, preview, batch_note)
SELECT $1, t.enc, t.hash, t.preview, $5
FROM UNNEST($2::text[], $3::text[], $4::text[]) WITH ORDINALITY AS t(enc, hash, preview, n)
ORDER BY t.n
ON CONFLICT (goods_id, content_hash) DO NOTHING`,
		goodsID, pq.Array(encs), pq.Array(hashes), pq.Array(previews), batchNote)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (r *cardGoodsRepository) ListItems(ctx context.Context, goodsID int64, status string, limit, offset int) ([]service.CardGoodsItem, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM card_goods_items WHERE goods_id = $1 AND ($2 = '' OR status = $2)`,
		goodsID, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT i.id, i.preview, i.status, i.order_id, i.user_id, COALESCE(u.email, ''), COALESCE(u.username, ''),
       i.batch_note, i.created_at, i.delivered_at
FROM card_goods_items i
LEFT JOIN users u ON u.id = i.user_id
WHERE i.goods_id = $1 AND ($2 = '' OR i.status = $2)
ORDER BY CASE WHEN i.status = 'sold' THEN i.delivered_at END DESC NULLS LAST, i.id DESC
LIMIT $3 OFFSET $4`, goodsID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.CardGoodsItem{}
	for rows.Next() {
		var it service.CardGoodsItem
		var orderID, userID sql.NullInt64
		var delivered sql.NullTime
		if err := rows.Scan(&it.ID, &it.Preview, &it.Status, &orderID, &userID, &it.UserEmail, &it.Username,
			&it.BatchNote, &it.CreatedAt, &delivered); err != nil {
			return nil, 0, err
		}
		if orderID.Valid {
			it.OrderID = &orderID.Int64
		}
		if userID.Valid {
			it.UserID = &userID.Int64
		}
		if delivered.Valid {
			it.DeliveredAt = &delivered.Time
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *cardGoodsRepository) VoidItems(ctx context.Context, goodsID int64, ids []int64) (int, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE card_goods_items SET status = 'void'
WHERE goods_id = $1 AND id = ANY($2::bigint[]) AND status = 'available'`, goodsID, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

func (r *cardGoodsRepository) orderHasItem(ctx context.Context, orderID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM card_goods_items WHERE order_id = $1)`, orderID).Scan(&ok)
	return ok, err
}

func (r *cardGoodsRepository) ClaimForOrder(ctx context.Context, goodsID, orderID, userID int64) (bool, error) {
	if ok, err := r.orderHasItem(ctx, orderID); err != nil || ok {
		return ok, err
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `
UPDATE card_goods_items SET status = 'sold', order_id = $2, user_id = $3, delivered_at = NOW()
WHERE id = (
    SELECT id FROM card_goods_items
    WHERE goods_id = $1 AND status = 'available'
    ORDER BY id
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING id`, goodsID, orderID, userID).Scan(&id)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case isUniqueViolation(err):
		// A concurrent fulfilment of the same order got there first.
		return r.orderHasItem(ctx, orderID)
	default:
		return false, err
	}
}

const cardGoodsDeliverySelect = `
SELECT i.order_id, i.goods_id, g.slug, g.name, g.usage_note, i.content_enc, i.delivered_at
FROM card_goods_items i
JOIN card_goods g ON g.id = i.goods_id`

func scanCardGoodsDelivery(row interface{ Scan(...any) error }) (*service.CardGoodsDeliveryRecord, error) {
	var d service.CardGoodsDeliveryRecord
	if err := row.Scan(&d.OrderID, &d.GoodsID, &d.GoodsSlug, &d.GoodsName, &d.UsageNote, &d.ContentEnc, &d.DeliveredAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *cardGoodsRepository) UserDeliveries(ctx context.Context, userID, goodsID int64) ([]service.CardGoodsDeliveryRecord, error) {
	rows, err := r.db.QueryContext(ctx, cardGoodsDeliverySelect+`
WHERE i.user_id = $1 AND i.status = 'sold' AND i.order_id IS NOT NULL AND ($2 = 0 OR i.goods_id = $2)
ORDER BY i.delivered_at DESC
LIMIT 200`, userID, goodsID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.CardGoodsDeliveryRecord{}
	for rows.Next() {
		d, err := scanCardGoodsDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

func (r *cardGoodsRepository) OrderDelivery(ctx context.Context, userID, orderID int64) (*service.CardGoodsDeliveryRecord, error) {
	d, err := scanCardGoodsDelivery(r.db.QueryRowContext(ctx, cardGoodsDeliverySelect+`
WHERE i.order_id = $2 AND i.user_id = $1 AND i.status = 'sold'`, userID, orderID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return d, err
}

func (r *cardGoodsRepository) WaitingOrderIDs(ctx context.Context, goodsID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT o.id FROM payment_orders o
WHERE o.goods_id = $1 AND o.order_type = 'goods' AND o.paid_at IS NOT NULL
  AND o.status IN ('PAID', 'RECHARGING', 'FAILED')
  AND NOT EXISTS (SELECT 1 FROM card_goods_items ci WHERE ci.order_id = o.id)
ORDER BY o.paid_at, o.id`, goodsID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
