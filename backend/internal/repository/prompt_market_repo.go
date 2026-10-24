package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type promptMarketRepository struct {
	db *sql.DB
}

// NewPromptMarketRepository creates the 提示词交易 repository (raw SQL).
func NewPromptMarketRepository(db *sql.DB) service.PromptMarketRepository {
	return &promptMarketRepository{db: db}
}

const promptListingSelect = `
SELECT l.id, l.seller_user_id,
       COALESCE(NULLIF(up.display_name, ''), NULLIF(up.handle, ''), NULLIF(u.username, ''), ''), COALESCE(u.email, ''),
       l.title, l.description, l.cover_url, l.sample_urls, l.price::float8, l.item_count, l.status, l.review_note, l.flags,
       l.draft, l.sales_count, l.published_at, l.created_at, l.updated_at
FROM prompt_listings l
LEFT JOIN users u ON u.id = l.seller_user_id
LEFT JOIN user_profiles up ON up.user_id = l.seller_user_id`

func scanPromptListing(row rowScanner) (*service.PromptListing, error) {
	var (
		l         service.PromptListing
		samples   []byte
		flags     pq.StringArray
		draft     []byte
		published sql.NullTime
	)
	if err := row.Scan(&l.ID, &l.SellerUserID, &l.SellerName, &l.SellerEmail, &l.Title, &l.Description, &l.CoverURL, &samples, &l.Price,
		&l.ItemCount, &l.Status, &l.ReviewNote, &flags, &draft, &l.SalesCount, &published, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return nil, err
	}
	if l.SellerName == "" {
		l.SellerName = "HiveGPT 用户"
	}
	l.SampleURLs = decodeStringList(samples)
	l.Flags = nonNilStrings(flags)
	if len(draft) > 0 && string(draft) != "null" {
		var d service.PromptListingDraft
		if json.Unmarshal(draft, &d) == nil {
			l.Draft = &d
		}
	}
	if published.Valid {
		t := published.Time
		l.PublishedAt = &t
	}
	return &l, nil
}

func (r *promptMarketRepository) loadItems(ctx context.Context, listings []service.PromptListing) error {
	if len(listings) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(listings))
	index := map[int64]int{}
	for i, l := range listings {
		ids = append(ids, l.ID)
		index[l.ID] = i
		listings[i].Items = []service.PromptListingItem{}
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT listing_id, position, title, prompt, tags, kind, dedupe_key, source_prompt_id
FROM prompt_listing_items WHERE listing_id = ANY($1) ORDER BY listing_id, position`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			listingID int64
			item      service.PromptListingItem
			tags      pq.StringArray
			source    sql.NullInt64
		)
		if err := rows.Scan(&listingID, &item.Position, &item.Title, &item.Prompt, &tags, &item.Kind, &item.DedupeKey, &source); err != nil {
			return err
		}
		item.Tags = nonNilStrings(tags)
		if source.Valid {
			id := source.Int64
			item.SourcePromptID = &id
		}
		i := index[listingID]
		listings[i].Items = append(listings[i].Items, item)
	}
	return rows.Err()
}

func (r *promptMarketRepository) ListListings(ctx context.Context, q service.PromptMarketListQuery) ([]service.PromptListing, int64, error) {
	w := &promptWhere{}
	switch q.Status {
	case "", "all":
	case "review":
		w.add("(l.status = 'pending' OR l.draft IS NOT NULL)")
	default:
		w.add("l.status = ?", q.Status)
	}
	if q.SellerID > 0 {
		w.add("l.seller_user_id = ?", q.SellerID)
	}
	if q.Keyword != "" {
		like := "%" + escapeLike(strings.ToLower(q.Keyword)) + "%"
		w.add("(lower(l.title) LIKE ? OR lower(l.description) LIKE ? OR EXISTS (SELECT 1 FROM prompt_listing_items k WHERE k.listing_id = l.id AND lower(k.title) LIKE ?))", like, like, like)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM prompt_listings l"+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := " ORDER BY l.sales_count DESC, COALESCE(l.published_at, l.created_at) DESC, l.id DESC"
	switch {
	case q.Status == "review":
		order = " ORDER BY l.updated_at, l.id"
	case q.Sort == "latest":
		order = " ORDER BY COALESCE(l.published_at, l.created_at) DESC, l.id DESC"
	case q.Sort == "price":
		order = " ORDER BY l.price, l.id DESC"
	}
	args := append(append([]any{}, w.args...), q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, promptListingSelect+w.sql()+order+fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptListing{}
	for rows.Next() {
		l, err := scanPromptListing(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, r.loadItems(ctx, out)
}

func (r *promptMarketRepository) GetListing(ctx context.Context, id int64) (*service.PromptListing, error) {
	l, err := scanPromptListing(r.db.QueryRowContext(ctx, promptListingSelect+" WHERE l.id = $1", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	list := []service.PromptListing{*l}
	if err := r.loadItems(ctx, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func encodeDraft(d *service.PromptListingDraft) any {
	if d == nil {
		return nil
	}
	raw, _ := json.Marshal(d)
	return string(raw)
}

func insertListingItems(ctx context.Context, tx *sql.Tx, listingID int64, items []service.PromptListingItem) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM prompt_listing_items WHERE listing_id = $1`, listingID); err != nil {
		return err
	}
	for i, item := range items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO prompt_listing_items (listing_id, position, title, prompt, tags, kind, dedupe_key, source_prompt_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			listingID, i, item.Title, item.Prompt, pq.Array(nonNilStrings(item.Tags)), item.Kind, item.DedupeKey, item.SourcePromptID); err != nil {
			return err
		}
	}
	return nil
}

func (r *promptMarketRepository) CreateListing(ctx context.Context, l *service.PromptListing) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.QueryRowContext(ctx, `
INSERT INTO prompt_listings (seller_user_id, title, description, cover_url, sample_urls, price, item_count, status, flags)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)
RETURNING id, created_at, updated_at`,
		l.SellerUserID, l.Title, l.Description, l.CoverURL, encodeStringList(l.SampleURLs), l.Price, len(l.Items), l.Status, pq.Array(nonNilStrings(l.Flags)),
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return err
	}
	if err := insertListingItems(ctx, tx, l.ID, l.Items); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *promptMarketRepository) UpdateListing(ctx context.Context, l *service.PromptListing, items []service.PromptListingItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	count := l.ItemCount
	if items != nil {
		count = len(items)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE prompt_listings SET title = $2, description = $3, cover_url = $4, sample_urls = $5::jsonb, price = $6, item_count = $7,
    status = $8, review_note = $9, flags = $10, draft = $11::jsonb, published_at = $12, updated_at = NOW()
WHERE id = $1`,
		l.ID, l.Title, l.Description, l.CoverURL, encodeStringList(l.SampleURLs), l.Price, count, l.Status, l.ReviewNote,
		pq.Array(nonNilStrings(l.Flags)), encodeDraft(l.Draft), l.PublishedAt); err != nil {
		return err
	}
	if items != nil {
		if err := insertListingItems(ctx, tx, l.ID, items); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *promptMarketRepository) CountSellerListings(ctx context.Context, sellerID int64) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prompt_listings WHERE seller_user_id = $1 AND status <> 'withdrawn'`, sellerID).Scan(&n)
	return n, err
}

func (r *promptMarketRepository) SellerPrompts(ctx context.Context, userID int64, ids []int64) ([]service.PromptItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, kind, title, prompt, tags FROM prompt_items WHERE owner_user_id = $1 AND id = ANY($2)`, userID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptItem{}
	for rows.Next() {
		var item service.PromptItem
		var tags pq.StringArray
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Prompt, &tags); err != nil {
			return nil, err
		}
		item.Tags = nonNilStrings(tags)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *promptMarketRepository) keySet(ctx context.Context, query string, args ...any) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out[key] = true
	}
	return out, rows.Err()
}

func (r *promptMarketRepository) BoughtDedupeKeys(ctx context.Context, userID int64, keys []string) (map[string]bool, error) {
	return r.keySet(ctx, `
SELECT DISTINCT i.dedupe_key FROM prompt_purchases p JOIN prompt_listing_items i ON i.listing_id = p.listing_id
WHERE p.buyer_user_id = $1 AND p.status = 'paid' AND i.dedupe_key = ANY($2)`, userID, pq.Array(keys))
}

func (r *promptMarketRepository) PublicCopies(ctx context.Context, userID int64, keys []string) (map[string]bool, error) {
	return r.keySet(ctx, `
SELECT DISTINCT dedupe_key FROM prompt_items
WHERE dedupe_key = ANY($2) AND status = 'active' AND visibility = 'public' AND (owner_user_id IS NULL OR owner_user_id <> $1)`, userID, pq.Array(keys))
}

func (r *promptMarketRepository) PurchasedListingIDs(ctx context.Context, buyerID int64, ids []int64) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT listing_id FROM prompt_purchases WHERE buyer_user_id = $1 AND status = 'paid' AND listing_id = ANY($2)`, buyerID, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

func (r *promptMarketRepository) Purchase(ctx context.Context, p *service.PromptPurchase) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Lock the listing: its price and status are read again under the lock.
	var (
		status string
		price  float64
		seller int64
	)
	err = tx.QueryRowContext(ctx, `SELECT status, price::float8, seller_user_id FROM prompt_listings WHERE id = $1 FOR UPDATE`, p.ListingID).Scan(&status, &price, &seller)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && status != service.PromptListingActive) {
		return service.ErrPromptListingNotFound
	}
	if err != nil {
		return err
	}
	if math.Abs(price-p.Price) > 0.001 {
		return service.ErrPromptPriceChanged
	}
	if seller != p.SellerUserID || seller == p.BuyerUserID {
		return service.ErrPromptBuyOwn
	}
	err = tx.QueryRowContext(ctx, `
INSERT INTO prompt_purchases (listing_id, buyer_user_id, seller_user_id, price, fee_rate, seller_amount, status)
VALUES ($1, $2, $3, $4, $5, $6, 'paid')
ON CONFLICT (listing_id, buyer_user_id) WHERE status = 'paid' DO NOTHING
RETURNING id, created_at`,
		p.ListingID, p.BuyerUserID, p.SellerUserID, p.Price, p.FeeRate, p.SellerAmount).Scan(&p.ID, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrPromptBoughtAlready
	}
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, p.Price, p.BuyerUserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrPromptBalanceLow
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2`, p.SellerAmount, p.SellerUserID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE prompt_listings SET sales_count = sales_count + 1 WHERE id = $1`, p.ListingID); err != nil {
		return err
	}
	return tx.Commit()
}

const promptPurchaseSelect = `
SELECT p.id, p.listing_id, p.buyer_user_id, p.seller_user_id, p.price::float8, p.fee_rate::float8, p.seller_amount::float8, p.status,
       p.refunded_at, p.created_at, COALESCE(l.title, ''), COALESCE(b.email, ''), COALESCE(s.email, '')
FROM prompt_purchases p
LEFT JOIN prompt_listings l ON l.id = p.listing_id
LEFT JOIN users b ON b.id = p.buyer_user_id
LEFT JOIN users s ON s.id = p.seller_user_id`

func scanPromptPurchase(row rowScanner) (*service.PromptPurchase, error) {
	var p service.PromptPurchase
	var refunded sql.NullTime
	if err := row.Scan(&p.ID, &p.ListingID, &p.BuyerUserID, &p.SellerUserID, &p.Price, &p.FeeRate, &p.SellerAmount, &p.Status,
		&refunded, &p.CreatedAt, &p.ListingTitle, &p.BuyerEmail, &p.SellerEmail); err != nil {
		return nil, err
	}
	if refunded.Valid {
		t := refunded.Time
		p.RefundedAt = &t
	}
	return &p, nil
}

func (r *promptMarketRepository) ListPurchases(ctx context.Context, q service.PromptPurchaseQuery) ([]service.PromptPurchase, int64, error) {
	w := &promptWhere{}
	if q.BuyerID > 0 {
		w.add("p.buyer_user_id = ?", q.BuyerID)
	}
	if q.SellerID > 0 {
		w.add("p.seller_user_id = ?", q.SellerID)
	}
	if q.Status != "" {
		w.add("p.status = ?", q.Status)
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM prompt_purchases p"+w.sql(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(append([]any{}, w.args...), q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, promptPurchaseSelect+w.sql()+" ORDER BY p.created_at DESC, p.id DESC"+
		" LIMIT $"+strconv.Itoa(len(w.args)+1)+" OFFSET $"+strconv.Itoa(len(w.args)+2), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := []service.PromptPurchase{}
	for rows.Next() {
		p, err := scanPromptPurchase(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}

func (r *promptMarketRepository) Refund(ctx context.Context, purchaseID int64) (*service.PromptPurchase, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var (
		status        string
		buyer, seller int64
		listingID     int64
		price, amount float64
	)
	err = tx.QueryRowContext(ctx, `SELECT status, buyer_user_id, seller_user_id, listing_id, price::float8, seller_amount::float8 FROM prompt_purchases WHERE id = $1 FOR UPDATE`, purchaseID).
		Scan(&status, &buyer, &seller, &listingID, &price, &amount)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && status != service.PromptPurchasePaid) {
		return nil, service.ErrPromptPurchaseNotFound
	}
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2`, price, buyer); err != nil {
		return nil, err
	}
	// The seller's share comes back even if it was spent (the balance may go negative).
	if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW() WHERE id = $2`, amount, seller); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE prompt_purchases SET status = 'refunded', refunded_at = NOW() WHERE id = $1`, purchaseID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE prompt_listings SET sales_count = GREATEST(sales_count - 1, 0) WHERE id = $1`, listingID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return scanPromptPurchase(r.db.QueryRowContext(ctx, promptPurchaseSelect+" WHERE p.id = $1", purchaseID))
}

func (r *promptMarketRepository) SellerSummary(ctx context.Context, sellerID int64) (service.PromptSellerSummary, error) {
	var s service.PromptSellerSummary
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(seller_amount), 0)::float8 FROM prompt_purchases WHERE seller_user_id = $1 AND status = 'paid'`, sellerID).Scan(&s.Sales, &s.Earned)
	return s, err
}

func (r *promptMarketRepository) CoverOwnedBy(ctx context.Context, file string, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM prompt_covers WHERE file = $1 AND user_id = $2)`, file, userID).Scan(&ok)
	return ok, err
}
