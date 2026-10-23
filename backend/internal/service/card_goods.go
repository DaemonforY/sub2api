package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 卡密商品 (card-key goods, e.g. Muse): the admin imports card keys / accounts into a product's
// stock; a buyer pays a payment order (order_type = goods, goods_id) and fulfilment hands them the
// oldest available key, shown on the product page and the payment result page. One key per order.
// Keys are stored encrypted and only decrypted for their buyer.
//
// Stock offered for sale = available keys − keys promised to open orders (pending, or paid but not
// delivered yet), so a sold-out product stops taking new orders. A paid order that still finds no
// key (two buyers raced for the last one) fails with 「库存不足」 and is delivered by the retry that
// runs after the admin's next import.
const (
	CardGoodsItemAvailable = "available"
	CardGoodsItemSold      = "sold"
	CardGoodsItemVoid      = "void"

	cardGoodsMaxPrice      = 100000.0
	cardGoodsMaxImport     = 5000
	cardGoodsMaxKeyLen     = 1000
	cardGoodsMaxTextLen    = 5000
	cardGoodsMaxBatchNote  = 60
	cardGoodsMaxItemsPage  = 200
	cardGoodsPreviewKeep   = 4
	cardGoodsMaxPerUserCap = 1000
)

var cardGoodsSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

var (
	ErrCardGoodsNotFound  = infraerrors.NotFound("GOODS_NOT_FOUND", "商品不存在或已下架（Goods not found）")
	ErrCardGoodsSoldOut   = infraerrors.Conflict("GOODS_SOLD_OUT", "这个商品暂时缺货，补货后再来购买（Out of stock）")
	ErrCardGoodsCurrency  = infraerrors.BadRequest("GOODS_CURRENCY", "这个商品只能用人民币支付，请换一种支付方式（Paid in CNY only）")
	ErrCardGoodsSlugTaken = infraerrors.Conflict("GOODS_SLUG_TAKEN", "商品地址已被占用，换一个（Slug taken）")
	ErrCardGoodsHasSales  = infraerrors.Conflict("GOODS_HAS_SALES", "已经卖出过的商品不能删除，可以下架（Goods has sales）")
	errCardGoodsBadInput  = infraerrors.BadRequest("GOODS_BAD_INPUT", "商品设置有误：地址只能用小写字母、数字和横线（最多 40 个），名称 1–60 个字，价格 0.01–100000（Invalid goods）")
	errCardGoodsNoKeys    = infraerrors.BadRequest("GOODS_IMPORT_EMPTY", "没有可导入的卡密：每行填一条（No keys to import）")
	errCardGoodsTooMany   = infraerrors.BadRequest("GOODS_IMPORT_TOO_MANY", "一次最多导入 5000 条卡密，请分批导入（Too many keys）")
	errCardGoodsOutOfKeys = errors.New("卡密库存不足，补货后自动发货（out of stock, delivered after restock）")
)

func errCardGoodsLimit(limit int) error {
	return infraerrors.Conflict("GOODS_LIMIT_REACHED", fmt.Sprintf("这个商品每人限购 %d 件，你已经买满了（Purchase limit reached）", limit))
}

// CardGoods is a product. Stock is the number offered for sale (see the package comment).
type CardGoods struct {
	ID            int64     `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Subtitle      string    `json:"subtitle"`
	Description   string    `json:"description"`
	UsageNote     string    `json:"usage_note"`
	Price         float64   `json:"price"`
	OriginalPrice float64   `json:"original_price"`
	PerUserLimit  int       `json:"per_user_limit"`
	Enabled       bool      `json:"enabled"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CardGoodsStock counts the keys of a product. Reserved: keys promised to open orders.
type CardGoodsStock struct {
	Available int `json:"available"`
	Sold      int `json:"sold"`
	Void      int `json:"void"`
	Reserved  int `json:"reserved"`
}

// ForSale is what new orders can still take.
func (s CardGoodsStock) ForSale() int {
	if n := s.Available - s.Reserved; n > 0 {
		return n
	}
	return 0
}

// CardGoodsAdminRow is a product in the admin list.
type CardGoodsAdminRow struct {
	CardGoods
	Stock   CardGoodsStock `json:"stock"`
	Waiting int            `json:"waiting"`
}

// CardGoodsPublic is the product page. Bought / Deliveries are filled for a signed-in viewer.
type CardGoodsPublic struct {
	ID            int64               `json:"id"`
	Slug          string              `json:"slug"`
	Name          string              `json:"name"`
	Subtitle      string              `json:"subtitle"`
	Description   string              `json:"description"`
	Price         float64             `json:"price"`
	OriginalPrice float64             `json:"original_price"`
	PerUserLimit  int                 `json:"per_user_limit"`
	Stock         int                 `json:"stock"`
	Bought        int                 `json:"bought"`
	Deliveries    []CardGoodsDelivery `json:"deliveries"`
}

// CardGoodsDelivery is a key delivered to its buyer.
type CardGoodsDelivery struct {
	OrderID     int64     `json:"order_id"`
	GoodsID     int64     `json:"goods_id"`
	GoodsSlug   string    `json:"goods_slug"`
	GoodsName   string    `json:"goods_name"`
	Content     string    `json:"content"`
	UsageNote   string    `json:"usage_note"`
	DeliveredAt time.Time `json:"delivered_at"`
}

// CardGoodsDeliveryRecord is a stored delivery (key still encrypted).
type CardGoodsDeliveryRecord struct {
	OrderID     int64
	GoodsID     int64
	GoodsSlug   string
	GoodsName   string
	UsageNote   string
	ContentEnc  string
	DeliveredAt time.Time
}

// CardGoodsItem is a key in the admin list (never decrypted there).
type CardGoodsItem struct {
	ID          int64      `json:"id"`
	Preview     string     `json:"preview"`
	Status      string     `json:"status"`
	OrderID     *int64     `json:"order_id,omitempty"`
	UserID      *int64     `json:"user_id,omitempty"`
	UserEmail   string     `json:"user_email,omitempty"`
	Username    string     `json:"username,omitempty"`
	BatchNote   string     `json:"batch_note"`
	CreatedAt   time.Time  `json:"created_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// CardGoodsNewItem is a key to import.
type CardGoodsNewItem struct {
	ContentEnc string
	Hash       string
	Preview    string
}

// CardGoodsImportResult: Added keys went into stock; Duplicates were already there (or repeated in
// the paste); Retried counts paid orders that were waiting for stock and were delivered now.
type CardGoodsImportResult struct {
	Added      int `json:"added"`
	Duplicates int `json:"duplicates"`
	Retried    int `json:"retried"`
}

type CardGoodsRepository interface {
	List(ctx context.Context, enabledOnly bool) ([]CardGoods, error)
	// GetByID / GetBySlug return nil, nil when there is no such product.
	GetByID(ctx context.Context, id int64) (*CardGoods, error)
	GetBySlug(ctx context.Context, slug string) (*CardGoods, error)
	SlugTaken(ctx context.Context, slug string, exceptID int64) (bool, error)
	Create(ctx context.Context, g *CardGoods) error
	Update(ctx context.Context, g *CardGoods) error
	Delete(ctx context.Context, id int64) error
	Stock(ctx context.Context, goodsID int64) (CardGoodsStock, error)
	// UserOpenCount: keys the user has bought plus their open orders for the product.
	UserOpenCount(ctx context.Context, userID, goodsID int64) (int, error)
	InsertItems(ctx context.Context, goodsID int64, items []CardGoodsNewItem, batchNote string) (int, error)
	ListItems(ctx context.Context, goodsID int64, status string, limit, offset int) ([]CardGoodsItem, int, error)
	// VoidItems removes keys still in stock (sold ones are left alone).
	VoidItems(ctx context.Context, goodsID int64, ids []int64) (int, error)
	// ClaimForOrder gives the order a key (idempotent). False when the order has none and stock is empty.
	ClaimForOrder(ctx context.Context, goodsID, orderID, userID int64) (bool, error)
	UserDeliveries(ctx context.Context, userID, goodsID int64) ([]CardGoodsDeliveryRecord, error)
	OrderDelivery(ctx context.Context, userID, orderID int64) (*CardGoodsDeliveryRecord, error)
	// WaitingOrderIDs: paid orders of the product that have no key yet.
	WaitingOrderIDs(ctx context.Context, goodsID int64) ([]int64, error)
}

type CardGoodsService struct {
	repo      CardGoodsRepository
	encryptor SecretEncryptor
}

func NewCardGoodsService(repo CardGoodsRepository, encryptor SecretEncryptor) *CardGoodsService {
	return &CardGoodsService{repo: repo, encryptor: encryptor}
}

// --- Buyer -----------------------------------------------------------------------------------

// Product is the product page; a disabled product is not found (except for its buyers, who keep
// seeing their keys).
func (s *CardGoodsService) Product(ctx context.Context, slug string, viewerID int64) (*CardGoodsPublic, error) {
	g, err := s.repo.GetBySlug(ctx, strings.TrimSpace(strings.ToLower(slug)))
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, ErrCardGoodsNotFound
	}
	out := &CardGoodsPublic{
		ID: g.ID, Slug: g.Slug, Name: g.Name, Subtitle: g.Subtitle, Description: g.Description,
		Price: g.Price, OriginalPrice: g.OriginalPrice, PerUserLimit: g.PerUserLimit,
		Deliveries: []CardGoodsDelivery{},
	}
	if viewerID > 0 {
		if out.Deliveries, err = s.Deliveries(ctx, viewerID, g.ID); err != nil {
			return nil, err
		}
		if out.Bought, err = s.repo.UserOpenCount(ctx, viewerID, g.ID); err != nil {
			return nil, err
		}
	}
	if !g.Enabled {
		if len(out.Deliveries) == 0 {
			return nil, ErrCardGoodsNotFound
		}
		return out, nil
	}
	stock, err := s.repo.Stock(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	out.Stock = stock.ForSale()
	return out, nil
}

// GoodsForOrder checks a new order: on sale, in stock and within the per-user limit.
func (s *CardGoodsService) GoodsForOrder(ctx context.Context, userID, goodsID int64) (*CardGoods, error) {
	g, err := s.repo.GetByID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	if g == nil || !g.Enabled || g.Price <= 0 {
		return nil, ErrCardGoodsNotFound
	}
	if g.PerUserLimit > 0 {
		n, err := s.repo.UserOpenCount(ctx, userID, g.ID)
		if err != nil {
			return nil, err
		}
		if n >= g.PerUserLimit {
			return nil, errCardGoodsLimit(g.PerUserLimit)
		}
	}
	stock, err := s.repo.Stock(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	if stock.ForSale() <= 0 {
		return nil, ErrCardGoodsSoldOut
	}
	return g, nil
}

// DeliverOrder hands a paid order its key (idempotent).
func (s *CardGoodsService) DeliverOrder(ctx context.Context, orderID, userID, goodsID int64) error {
	ok, err := s.repo.ClaimForOrder(ctx, goodsID, orderID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errCardGoodsOutOfKeys
	}
	return nil
}

// Deliveries lists the keys a user bought (goodsID 0 = every product), newest first.
func (s *CardGoodsService) Deliveries(ctx context.Context, userID, goodsID int64) ([]CardGoodsDelivery, error) {
	recs, err := s.repo.UserDeliveries(ctx, userID, goodsID)
	if err != nil {
		return nil, err
	}
	out := make([]CardGoodsDelivery, 0, len(recs))
	for _, r := range recs {
		d, err := s.open(r)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// OrderDelivery is the key of one of the user's orders; nil while it has not been delivered.
func (s *CardGoodsService) OrderDelivery(ctx context.Context, userID, orderID int64) (*CardGoodsDelivery, error) {
	rec, err := s.repo.OrderDelivery(ctx, userID, orderID)
	if err != nil || rec == nil {
		return nil, err
	}
	d, err := s.open(*rec)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *CardGoodsService) open(r CardGoodsDeliveryRecord) (CardGoodsDelivery, error) {
	content, err := s.encryptor.Decrypt(r.ContentEnc)
	if err != nil {
		return CardGoodsDelivery{}, fmt.Errorf("decrypt card key of order %d: %w", r.OrderID, err)
	}
	return CardGoodsDelivery{
		OrderID: r.OrderID, GoodsID: r.GoodsID, GoodsSlug: r.GoodsSlug, GoodsName: r.GoodsName,
		Content: content, UsageNote: r.UsageNote, DeliveredAt: r.DeliveredAt,
	}, nil
}

// --- Admin -----------------------------------------------------------------------------------

func (s *CardGoodsService) AdminList(ctx context.Context) ([]CardGoodsAdminRow, error) {
	list, err := s.repo.List(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make([]CardGoodsAdminRow, 0, len(list))
	for _, g := range list {
		row := CardGoodsAdminRow{CardGoods: g}
		if row.Stock, err = s.repo.Stock(ctx, g.ID); err != nil {
			return nil, err
		}
		waiting, err := s.repo.WaitingOrderIDs(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		row.Waiting = len(waiting)
		out = append(out, row)
	}
	return out, nil
}

// Save creates (id 0) or updates a product.
func (s *CardGoodsService) Save(ctx context.Context, id int64, in CardGoods) (*CardGoods, error) {
	g, err := normalizeCardGoods(in)
	if err != nil {
		return nil, err
	}
	taken, err := s.repo.SlugTaken(ctx, g.Slug, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrCardGoodsSlugTaken
	}
	if id == 0 {
		if err := s.repo.Create(ctx, &g); err != nil {
			return nil, err
		}
		return &g, nil
	}
	cur, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, ErrCardGoodsNotFound
	}
	g.ID = id
	if err := s.repo.Update(ctx, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func normalizeCardGoods(in CardGoods) (CardGoods, error) {
	g := in
	g.Slug = strings.ToLower(strings.TrimSpace(g.Slug))
	g.Name = strings.TrimSpace(g.Name)
	g.Subtitle = strings.TrimSpace(g.Subtitle)
	g.Description = strings.TrimSpace(g.Description)
	g.UsageNote = strings.TrimSpace(g.UsageNote)
	n := utf8.RuneCountInString(g.Name)
	if !cardGoodsSlugRe.MatchString(g.Slug) || n == 0 || n > 60 ||
		utf8.RuneCountInString(g.Subtitle) > 120 ||
		utf8.RuneCountInString(g.Description) > cardGoodsMaxTextLen ||
		utf8.RuneCountInString(g.UsageNote) > cardGoodsMaxTextLen ||
		!validCardGoodsPrice(g.Price) || (g.OriginalPrice != 0 && !validCardGoodsPrice(g.OriginalPrice)) ||
		g.PerUserLimit < 0 || g.PerUserLimit > cardGoodsMaxPerUserCap {
		return g, errCardGoodsBadInput
	}
	g.Price = math.Round(g.Price*100) / 100
	g.OriginalPrice = math.Round(g.OriginalPrice*100) / 100
	return g, nil
}

func validCardGoodsPrice(p float64) bool {
	return !math.IsNaN(p) && !math.IsInf(p, 0) && p >= 0.01 && p <= cardGoodsMaxPrice
}

// Delete removes a product that never sold anything (sold products are disabled instead).
func (s *CardGoodsService) Delete(ctx context.Context, id int64) error {
	stock, err := s.repo.Stock(ctx, id)
	if err != nil {
		return err
	}
	if stock.Sold > 0 || stock.Reserved > 0 {
		return ErrCardGoodsHasSales
	}
	return s.repo.Delete(ctx, id)
}

// Import adds keys, one per line (blank lines skipped, a key repeated or already in stock counted
// as a duplicate).
func (s *CardGoodsService) Import(ctx context.Context, goodsID int64, raw, batchNote string) (CardGoodsImportResult, error) {
	var res CardGoodsImportResult
	g, err := s.repo.GetByID(ctx, goodsID)
	if err != nil {
		return res, err
	}
	if g == nil {
		return res, ErrCardGoodsNotFound
	}
	keys, dup, err := parseCardKeys(raw)
	if err != nil {
		return res, err
	}
	items := make([]CardGoodsNewItem, 0, len(keys))
	for _, k := range keys {
		enc, err := s.encryptor.Encrypt(k)
		if err != nil {
			return res, fmt.Errorf("encrypt card key: %w", err)
		}
		items = append(items, CardGoodsNewItem{ContentEnc: enc, Hash: cardKeyHash(k), Preview: cardKeyPreview(k)})
	}
	note := strings.TrimSpace(batchNote)
	if r := []rune(note); len(r) > cardGoodsMaxBatchNote {
		note = string(r[:cardGoodsMaxBatchNote])
	}
	added, err := s.repo.InsertItems(ctx, goodsID, items, note)
	if err != nil {
		return res, err
	}
	res.Added = added
	res.Duplicates = dup + len(items) - added
	return res, nil
}

func parseCardKeys(raw string) ([]string, int, error) {
	seen := map[string]bool{}
	var keys []string
	dup := 0
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		k := strings.TrimSpace(line)
		if k == "" {
			continue
		}
		if utf8.RuneCountInString(k) > cardGoodsMaxKeyLen {
			return nil, 0, infraerrors.BadRequest("GOODS_KEY_TOO_LONG", fmt.Sprintf("有一行卡密超过 %d 个字，请检查是否少了换行（Key too long）", cardGoodsMaxKeyLen))
		}
		if seen[k] {
			dup++
			continue
		}
		seen[k] = true
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return nil, 0, errCardGoodsNoKeys
	}
	if len(keys) > cardGoodsMaxImport {
		return nil, 0, errCardGoodsTooMany
	}
	return keys, dup, nil
}

func cardKeyHash(k string) string {
	sum := sha256.Sum256([]byte(k))
	return hex.EncodeToString(sum[:])
}

// cardKeyPreview keeps the first and last few characters so the admin can tell keys apart.
func cardKeyPreview(k string) string {
	r := []rune(k)
	if len(r) <= cardGoodsPreviewKeep*2+2 {
		if len(r) <= 2 {
			return "**"
		}
		return string(r[:1]) + "…" + string(r[len(r)-1:])
	}
	return string(r[:cardGoodsPreviewKeep]) + "…" + string(r[len(r)-cardGoodsPreviewKeep:])
}

func (s *CardGoodsService) Items(ctx context.Context, goodsID int64, status string, page, pageSize int) ([]CardGoodsItem, int, error) {
	switch status {
	case "", CardGoodsItemAvailable, CardGoodsItemSold, CardGoodsItemVoid:
	default:
		status = ""
	}
	if pageSize <= 0 || pageSize > cardGoodsMaxItemsPage {
		pageSize = 50
	}
	if page < 1 {
		page = 1
	}
	return s.repo.ListItems(ctx, goodsID, status, pageSize, (page-1)*pageSize)
}

func (s *CardGoodsService) VoidItems(ctx context.Context, goodsID int64, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	return s.repo.VoidItems(ctx, goodsID, ids)
}

// WaitingOrderIDs: paid orders of the product still waiting for a key.
func (s *CardGoodsService) WaitingOrderIDs(ctx context.Context, goodsID int64) ([]int64, error) {
	return s.repo.WaitingOrderIDs(ctx, goodsID)
}
