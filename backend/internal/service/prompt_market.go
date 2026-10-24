package service

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 提示词交易: a user sells one of their prompts or a pack of them (合集) for 余额. The buyer pays from
// their balance; the seller is credited the price minus the platform fee, as balance only (there is
// no cash-out, so the site never pays anyone money). A listing snapshots the chosen prompts when it is
// submitted, and an admin reviews every new listing and every change before it goes live; a change to
// a live listing waits in Draft while the approved version keeps selling. Buyers see the titles and a
// short teaser until they buy; then the full prompts, also after later approved edits.

const (
	SettingKeyPromptMarket = "prompt_market"

	PromptListingPending   = "pending"
	PromptListingActive    = "active"
	PromptListingRejected  = "rejected"
	PromptListingHidden    = "hidden"
	PromptListingWithdrawn = "withdrawn"

	PromptPurchasePaid     = "paid"
	PromptPurchaseRefunded = "refunded"

	promptListingTitleMax   = 40
	promptListingDescMax    = 500
	promptListingMaxSamples = 6
	promptListingMaxItems   = 50
	promptSellerMaxListings = 50
	promptTeaserMaxRunes    = 40

	promptMarketDefaultFee = 0.2
	promptMarketDefaultMin = 0.1
	promptMarketDefaultMax = 50.0
)

var (
	ErrPromptMarketClosed      = infraerrors.BadRequest("PROMPT_MARKET_CLOSED", "提示词交易暂未开放（The prompt market is closed）")
	ErrPromptListingNotFound   = infraerrors.NotFound("PROMPT_LISTING_NOT_FOUND", "这个提示词已下架或不存在（Listing not found）")
	ErrPromptListingTitle      = infraerrors.BadRequest("PROMPT_LISTING_TITLE", "标题需为 1–40 个字（Title must be 1–40 characters）")
	ErrPromptListingItems      = infraerrors.BadRequest("PROMPT_LISTING_ITEMS", "请选择 1 条（单条出售）或 2–50 条（合集）自己的提示词（Choose 1–50 of your prompts）")
	ErrPromptListingPrice      = infraerrors.BadRequest("PROMPT_LISTING_PRICE", "价格超出允许范围，请重新填写（Price out of range）")
	ErrPromptListingSamples    = infraerrors.BadRequest("PROMPT_LISTING_SAMPLES", "示例图最多 6 张，且需是你自己上传的图片（Invalid sample images）")
	ErrPromptListingTooMany    = infraerrors.BadRequest("PROMPT_LISTING_TOO_MANY", "最多同时上架 50 个，请先下架一些（Too many listings）")
	ErrPromptListingResale     = infraerrors.BadRequest("PROMPT_LISTING_RESALE", "不能出售从市场买来的提示词（You cannot resell prompts you bought）")
	ErrPromptListingPublicCopy = infraerrors.BadRequest("PROMPT_LISTING_PUBLIC_COPY", "这些提示词在公开提示词库里已经可以免费使用，不能出售（Free public prompts cannot be sold）")
	ErrPromptListingState      = infraerrors.BadRequest("PROMPT_LISTING_STATE", "当前状态不能进行这个操作，请刷新后重试（Not allowed in the current state）")
	ErrPromptBuyOwn            = infraerrors.BadRequest("PROMPT_BUY_OWN", "这是你自己上架的提示词，不用购买（This is your own listing）")
	ErrPromptBoughtAlready     = infraerrors.Conflict("PROMPT_BOUGHT_ALREADY", "你已经买过了，在「已购」里可以查看（Already purchased）")
	ErrPromptPriceChanged      = infraerrors.Conflict("PROMPT_PRICE_CHANGED", "价格刚刚变了，请确认新价格后再购买（The price changed）")
	ErrPromptBalanceLow        = infraerrors.BadRequest("PROMPT_BALANCE_LOW", "余额不足，请先充值后再购买（Insufficient balance）")
	ErrPromptPurchaseNotFound  = infraerrors.NotFound("PROMPT_PURCHASE_NOT_FOUND", "找不到这笔购买记录（Purchase not found）")
	ErrPromptMarketBadConfig   = infraerrors.BadRequest("PROMPT_MARKET_BAD_CONFIG", "交易设置有误：分成 0–90%，最低价 0.01–最高价，最高价不超过 1000（Invalid market settings）")
)

// PromptMarketConfig is the admin setting.
type PromptMarketConfig struct {
	Enabled bool `json:"enabled"`
	// FeeRate is the platform's share of each sale (0.2 = 20%).
	FeeRate  float64 `json:"fee_rate"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

// PromptListingItem is one prompt of a listing.
type PromptListingItem struct {
	Position int      `json:"position"`
	Title    string   `json:"title"`
	Prompt   string   `json:"prompt,omitempty"`
	Tags     []string `json:"tags"`
	Kind     string   `json:"kind"`
	// DedupeKey / SourcePromptID are stored but never shown.
	DedupeKey      string `json:"dedupe_key,omitempty"`
	SourcePromptID *int64 `json:"source_prompt_id,omitempty"`
}

// PromptListingDraft is a change to an approved listing waiting for review.
type PromptListingDraft struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	CoverURL    string              `json:"cover_url"`
	SampleURLs  []string            `json:"sample_urls"`
	Price       float64             `json:"price"`
	Items       []PromptListingItem `json:"items"`
}

type PromptListing struct {
	ID           int64    `json:"id"`
	SellerUserID int64    `json:"-"`
	SellerName   string   `json:"seller_name"`
	SellerEmail  string   `json:"seller_email,omitempty"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	CoverURL     string   `json:"cover_url"`
	SampleURLs   []string `json:"sample_urls"`
	Price        float64  `json:"price"`
	ItemCount    int      `json:"item_count"`
	Status       string   `json:"status"`
	ReviewNote   string   `json:"review_note,omitempty"`
	Flags        []string `json:"flags,omitempty"`
	// Draft: shown to the seller and the admin only.
	Draft       *PromptListingDraft `json:"draft,omitempty"`
	SalesCount  int                 `json:"sales_count"`
	PublishedAt *time.Time          `json:"published_at,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	// Items carry the full prompts only when Unlocked; otherwise titles and the teaser.
	Items     []PromptListingItem `json:"items"`
	Teaser    string              `json:"teaser,omitempty"`
	Unlocked  bool                `json:"unlocked"`
	Purchased bool                `json:"purchased"`
	Mine      bool                `json:"mine"`
}

type PromptPurchase struct {
	ID           int64      `json:"id"`
	ListingID    int64      `json:"listing_id"`
	BuyerUserID  int64      `json:"buyer_user_id"`
	SellerUserID int64      `json:"seller_user_id"`
	Price        float64    `json:"price"`
	FeeRate      float64    `json:"fee_rate"`
	SellerAmount float64    `json:"seller_amount"`
	Status       string     `json:"status"`
	RefundedAt   *time.Time `json:"refunded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ListingTitle string     `json:"listing_title"`
	BuyerEmail   string     `json:"buyer_email,omitempty"`
	SellerEmail  string     `json:"seller_email,omitempty"`
}

// PromptListingInput is what a seller submits.
type PromptListingInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CoverURL    string   `json:"cover_url"`
	SampleURLs  []string `json:"sample_urls"`
	Price       float64  `json:"price"`
	// PromptIDs are the seller's own prompts (「我的」), in order.
	PromptIDs []int64 `json:"prompt_ids"`
}

type PromptMarketListQuery struct {
	Keyword  string
	Sort     string // popular / latest / price
	Status   string // admin only; "review" = pending or with a draft
	SellerID int64
	Page     int
	PageSize int
}

type PromptPurchaseQuery struct {
	BuyerID  int64
	SellerID int64
	Status   string
	Page     int
	PageSize int
}

type PromptSellerSummary struct {
	Sales  int64   `json:"sales"`
	Earned float64 `json:"earned"`
}

type PromptMarketRepository interface {
	ListListings(ctx context.Context, q PromptMarketListQuery) ([]PromptListing, int64, error)
	// GetListing returns the listing with its items (nil when missing).
	GetListing(ctx context.Context, id int64) (*PromptListing, error)
	CreateListing(ctx context.Context, l *PromptListing) error
	// UpdateListing writes the listing; non-nil items replace the live content.
	UpdateListing(ctx context.Context, l *PromptListing, items []PromptListingItem) error
	CountSellerListings(ctx context.Context, sellerID int64) (int, error)
	// SellerPrompts loads the given prompts the user owns (others are skipped).
	SellerPrompts(ctx context.Context, userID int64, ids []int64) ([]PromptItem, error)
	// BoughtDedupeKeys: which keys appear in listings the user bought (paid).
	BoughtDedupeKeys(ctx context.Context, userID int64, keys []string) (map[string]bool, error)
	// PublicCopies: which keys belong to public library prompts of someone else (or a community source).
	PublicCopies(ctx context.Context, userID int64, keys []string) (map[string]bool, error)
	PurchasedListingIDs(ctx context.Context, buyerID int64, ids []int64) (map[int64]bool, error)
	// Purchase moves the balance and records the sale in one transaction; ErrPromptBalanceLow,
	// ErrPromptBoughtAlready, ErrPromptPriceChanged or ErrPromptListingNotFound when it cannot.
	Purchase(ctx context.Context, p *PromptPurchase) error
	ListPurchases(ctx context.Context, q PromptPurchaseQuery) ([]PromptPurchase, int64, error)
	// Refund gives the buyer the price back and takes the seller's share back (their balance may go negative).
	Refund(ctx context.Context, purchaseID int64) (*PromptPurchase, error)
	SellerSummary(ctx context.Context, sellerID int64) (PromptSellerSummary, error)
	CoverOwnedBy(ctx context.Context, file string, userID int64) (bool, error)
}

type PromptMarketService struct {
	repo         PromptMarketRepository
	settings     SettingRepository
	billingCache *BillingCacheService
	authCache    APIKeyAuthCacheInvalidator
	now          func() time.Time
}

func NewPromptMarketService(repo PromptMarketRepository, settings SettingRepository, billingCache *BillingCacheService, authCache APIKeyAuthCacheInvalidator) *PromptMarketService {
	return &PromptMarketService{repo: repo, settings: settings, billingCache: billingCache, authCache: authCache, now: time.Now}
}

// Config is the admin setting with defaults (off, 20% fee, $0.1–$50).
func (s *PromptMarketService) Config(ctx context.Context) PromptMarketConfig {
	cfg := PromptMarketConfig{FeeRate: promptMarketDefaultFee, MinPrice: promptMarketDefaultMin, MaxPrice: promptMarketDefaultMax}
	raw, err := s.settings.GetValue(ctx, SettingKeyPromptMarket)
	if err != nil || strings.TrimSpace(raw) == "" {
		return cfg
	}
	var saved PromptMarketConfig
	if json.Unmarshal([]byte(raw), &saved) != nil || validatePromptMarketConfig(&saved) != nil {
		return cfg
	}
	return saved
}

func validatePromptMarketConfig(cfg *PromptMarketConfig) error {
	bad := func(v float64) bool { return math.IsNaN(v) || math.IsInf(v, 0) }
	if bad(cfg.FeeRate) || bad(cfg.MinPrice) || bad(cfg.MaxPrice) || cfg.FeeRate < 0 || cfg.FeeRate > 0.9 ||
		cfg.MinPrice < 0.01 || cfg.MaxPrice < cfg.MinPrice || cfg.MaxPrice > 1000 {
		return ErrPromptMarketBadConfig
	}
	cfg.FeeRate = math.Round(cfg.FeeRate*10000) / 10000
	cfg.MinPrice = roundCents(cfg.MinPrice)
	cfg.MaxPrice = roundCents(cfg.MaxPrice)
	return nil
}

func (s *PromptMarketService) SaveConfig(ctx context.Context, cfg PromptMarketConfig) (PromptMarketConfig, error) {
	if err := validatePromptMarketConfig(&cfg); err != nil {
		return cfg, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, s.settings.Set(ctx, SettingKeyPromptMarket, string(raw))
}

// teaser is the start of the first prompt: enough to judge the style, not enough to use it.
func promptTeaser(items []PromptListingItem) string {
	if len(items) == 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(items[0].Prompt))
	n := len(runes) / 4
	if n > promptTeaserMaxRunes {
		n = promptTeaserMaxRunes
	}
	if n < 1 {
		return ""
	}
	return string(runes[:n]) + "…"
}

// view shapes a listing for a viewer: full prompts for the seller and buyers, titles + teaser otherwise.
func (s *PromptMarketService) view(l PromptListing, viewerID int64, purchased bool) PromptListing {
	l.Mine = viewerID > 0 && l.SellerUserID == viewerID
	l.Purchased = purchased
	l.Unlocked = l.Mine || purchased
	l.SellerEmail = ""
	l.Flags = nil
	if !l.Mine {
		l.Draft = nil
		l.ReviewNote = ""
	}
	if !l.Unlocked {
		l.Teaser = promptTeaser(l.Items)
	}
	items := make([]PromptListingItem, 0, len(l.Items))
	for _, item := range l.Items {
		item.DedupeKey = ""
		// The seller edits a listing by its prompts in 「我的」; nobody else sees which they were.
		if !l.Mine {
			item.SourcePromptID = nil
		}
		if !l.Unlocked {
			item.Prompt = ""
		}
		items = append(items, item)
	}
	l.Items = items
	return l
}

// ListPublic lists live listings (viewerID 0 = signed out).
func (s *PromptMarketService) ListPublic(ctx context.Context, viewerID int64, q PromptMarketListQuery) ([]PromptListing, int64, error) {
	q.Status = PromptListingActive
	q.SellerID = 0
	normalizeMarketQuery(&q, 60)
	list, total, err := s.repo.ListListings(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	bought := map[int64]bool{}
	if viewerID > 0 && len(list) > 0 {
		ids := make([]int64, 0, len(list))
		for _, l := range list {
			ids = append(ids, l.ID)
		}
		if bought, err = s.repo.PurchasedListingIDs(ctx, viewerID, ids); err != nil {
			return nil, 0, err
		}
	}
	out := make([]PromptListing, 0, len(list))
	for _, l := range list {
		out = append(out, s.view(l, viewerID, bought[l.ID]))
	}
	return out, total, nil
}

func normalizeMarketQuery(q *PromptMarketListQuery, maxSize int) {
	q.Keyword = truncateRunes(strings.TrimSpace(q.Keyword), 60)
	if q.Sort != "latest" && q.Sort != "price" {
		q.Sort = "popular"
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > maxSize {
		q.PageSize = 24
	}
}

// Get returns one listing for a viewer; hidden / pending listings only for their seller and buyers.
func (s *PromptMarketService) Get(ctx context.Context, viewerID, id int64) (*PromptListing, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrPromptListingNotFound
	}
	purchased := false
	if viewerID > 0 {
		bought, err := s.repo.PurchasedListingIDs(ctx, viewerID, []int64{id})
		if err != nil {
			return nil, err
		}
		purchased = bought[id]
	}
	if l.Status != PromptListingActive && l.SellerUserID != viewerID && !purchased {
		return nil, ErrPromptListingNotFound
	}
	v := s.view(*l, viewerID, purchased)
	return &v, nil
}

// Purchase buys a listing with the buyer's balance at the price they saw.
func (s *PromptMarketService) Purchase(ctx context.Context, buyerID, listingID int64, seenPrice float64) (*PromptListing, error) {
	cfg := s.Config(ctx)
	if !cfg.Enabled {
		return nil, ErrPromptMarketClosed
	}
	l, err := s.repo.GetListing(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if l == nil || l.Status != PromptListingActive {
		return nil, ErrPromptListingNotFound
	}
	if l.SellerUserID == buyerID {
		return nil, ErrPromptBuyOwn
	}
	if roundCents(seenPrice) != roundCents(l.Price) {
		return nil, ErrPromptPriceChanged
	}
	p := &PromptPurchase{
		ListingID: listingID, BuyerUserID: buyerID, SellerUserID: l.SellerUserID, Price: roundCents(l.Price), FeeRate: cfg.FeeRate,
		SellerAmount: math.Round(l.Price*(1-cfg.FeeRate)*10000) / 10000, Status: PromptPurchasePaid,
	}
	if err := s.repo.Purchase(ctx, p); err != nil {
		return nil, err
	}
	s.invalidateBalances(ctx, buyerID, l.SellerUserID)
	return s.Get(ctx, buyerID, listingID)
}

func (s *PromptMarketService) invalidateBalances(ctx context.Context, userIDs ...int64) {
	for _, id := range userIDs {
		if s.authCache != nil {
			s.authCache.InvalidateAuthCacheByUserID(ctx, id)
		}
	}
	if s.billingCache == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, id := range userIDs {
			_ = s.billingCache.InvalidateUserBalance(cacheCtx, id)
		}
	}()
}

// Purchases lists what the user bought, with the full prompts.
func (s *PromptMarketService) Purchases(ctx context.Context, buyerID int64) ([]PromptListing, error) {
	purchases, _, err := s.repo.ListPurchases(ctx, PromptPurchaseQuery{BuyerID: buyerID, Status: PromptPurchasePaid, Page: 1, PageSize: 500})
	if err != nil {
		return nil, err
	}
	out := []PromptListing{}
	for _, p := range purchases {
		l, err := s.repo.GetListing(ctx, p.ListingID)
		if err != nil {
			return nil, err
		}
		if l != nil {
			out = append(out, s.view(*l, buyerID, true))
		}
	}
	return out, nil
}

// Selling lists the seller's own listings (every status) and what they earned.
func (s *PromptMarketService) Selling(ctx context.Context, sellerID int64) ([]PromptListing, PromptSellerSummary, error) {
	list, _, err := s.repo.ListListings(ctx, PromptMarketListQuery{SellerID: sellerID, Sort: "latest", Page: 1, PageSize: promptSellerMaxListings + 10})
	if err != nil {
		return nil, PromptSellerSummary{}, err
	}
	summary, err := s.repo.SellerSummary(ctx, sellerID)
	if err != nil {
		return nil, PromptSellerSummary{}, err
	}
	out := make([]PromptListing, 0, len(list))
	for _, l := range list {
		out = append(out, s.view(l, sellerID, false))
	}
	return out, summary, nil
}

// buildDraft validates a submission and snapshots the seller's prompts.
func (s *PromptMarketService) buildDraft(ctx context.Context, sellerID int64, in PromptListingInput) (*PromptListingDraft, []string, error) {
	cfg := s.Config(ctx)
	title := strings.Join(strings.Fields(in.Title), " ")
	if n := utf8.RuneCountInString(title); n == 0 || n > promptListingTitleMax {
		return nil, nil, ErrPromptListingTitle
	}
	price := roundCents(in.Price)
	if math.IsNaN(in.Price) || price < cfg.MinPrice || price > cfg.MaxPrice {
		return nil, nil, ErrPromptListingPrice
	}
	ids := uniqueInt64s(in.PromptIDs)
	if len(ids) == 0 || len(ids) > promptListingMaxItems {
		return nil, nil, ErrPromptListingItems
	}
	prompts, err := s.repo.SellerPrompts(ctx, sellerID, ids)
	if err != nil {
		return nil, nil, err
	}
	if len(prompts) != len(ids) {
		return nil, nil, ErrPromptListingItems
	}
	byID := map[int64]PromptItem{}
	for _, p := range prompts {
		byID[p.ID] = p
	}
	draft := &PromptListingDraft{Title: title, Description: truncateRunes(strings.TrimSpace(in.Description), promptListingDescMax), Price: price, SampleURLs: []string{}}
	keys := make([]string, 0, len(ids))
	flags := map[string]bool{}
	for i, id := range ids {
		p := byID[id]
		source := id
		item := PromptListingItem{Position: i, Title: truncateRunes(p.Title, 60), Prompt: p.Prompt, Tags: nonNilStringSlice(p.Tags), Kind: p.Kind, DedupeKey: PromptDedupeKey(p.Prompt), SourcePromptID: &source}
		draft.Items = append(draft.Items, item)
		keys = append(keys, item.DedupeKey)
		traits := ClassifyPrompt(PromptTraitInput{Title: p.Title, Prompt: p.Prompt, Tags: p.Tags})
		for _, flag := range promptAutoFlags(traits) {
			flags[flag] = true
		}
	}
	bought, err := s.repo.BoughtDedupeKeys(ctx, sellerID, keys)
	if err != nil {
		return nil, nil, err
	}
	if len(bought) > 0 {
		return nil, nil, ErrPromptListingResale
	}
	copies, err := s.repo.PublicCopies(ctx, sellerID, keys)
	if err != nil {
		return nil, nil, err
	}
	if len(copies) > 0 {
		return nil, nil, ErrPromptListingPublicCopy
	}
	images := append([]string{in.CoverURL}, in.SampleURLs...)
	if len(in.SampleURLs) > promptListingMaxSamples {
		return nil, nil, ErrPromptListingSamples
	}
	for i, raw := range images {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		file, ok := PromptCoverFileFromURL(raw)
		if !ok {
			return nil, nil, ErrPromptListingSamples
		}
		owned, err := s.repo.CoverOwnedBy(ctx, file, sellerID)
		if err != nil {
			return nil, nil, err
		}
		if !owned {
			return nil, nil, ErrPromptListingSamples
		}
		if i == 0 {
			draft.CoverURL = PromptCoverURL(file)
		} else {
			draft.SampleURLs = append(draft.SampleURLs, PromptCoverURL(file))
		}
	}
	if draft.CoverURL == "" && len(draft.SampleURLs) > 0 {
		draft.CoverURL = draft.SampleURLs[0]
	}
	flagList := make([]string, 0, len(flags))
	for _, f := range []string{"nsfw", "sensitive"} {
		if flags[f] {
			flagList = append(flagList, f)
		}
	}
	return draft, flagList, nil
}

func uniqueInt64s(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func nonNilStringSlice(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func applyDraft(l *PromptListing, d *PromptListingDraft) {
	l.Title = d.Title
	l.Description = d.Description
	l.CoverURL = d.CoverURL
	l.SampleURLs = d.SampleURLs
	l.Price = d.Price
	l.ItemCount = len(d.Items)
	l.Items = d.Items
}

// CreateListing submits a new listing for review (selling must be open).
func (s *PromptMarketService) CreateListing(ctx context.Context, sellerID int64, in PromptListingInput) (*PromptListing, error) {
	if !s.Config(ctx).Enabled {
		return nil, ErrPromptMarketClosed
	}
	n, err := s.repo.CountSellerListings(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	if n >= promptSellerMaxListings {
		return nil, ErrPromptListingTooMany
	}
	draft, flags, err := s.buildDraft(ctx, sellerID, in)
	if err != nil {
		return nil, err
	}
	l := &PromptListing{SellerUserID: sellerID, Status: PromptListingPending, Flags: flags}
	applyDraft(l, draft)
	if err := s.repo.CreateListing(ctx, l); err != nil {
		return nil, err
	}
	return s.Get(ctx, sellerID, l.ID)
}

// UpdateListing changes a listing. Not yet approved: replaced and (re)submitted. Approved (live,
// withdrawn or hidden): the change waits in draft for review and the approved version stays.
func (s *PromptMarketService) UpdateListing(ctx context.Context, sellerID, id int64, in PromptListingInput) (*PromptListing, error) {
	if !s.Config(ctx).Enabled {
		return nil, ErrPromptMarketClosed
	}
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil || l.SellerUserID != sellerID {
		return nil, ErrPromptListingNotFound
	}
	draft, flags, err := s.buildDraft(ctx, sellerID, in)
	if err != nil {
		return nil, err
	}
	l.Flags = flags
	l.ReviewNote = ""
	var items []PromptListingItem
	if l.Status == PromptListingPending || l.Status == PromptListingRejected {
		applyDraft(l, draft)
		l.Status = PromptListingPending
		l.Draft = nil
		items = l.Items
	} else {
		l.Draft = draft
	}
	if err := s.repo.UpdateListing(ctx, l, items); err != nil {
		return nil, err
	}
	return s.Get(ctx, sellerID, id)
}

// SetListed takes an approved listing off the market (withdrawn) or puts it back (active).
func (s *PromptMarketService) SetListed(ctx context.Context, sellerID, id int64, listed bool) (*PromptListing, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil || l.SellerUserID != sellerID {
		return nil, ErrPromptListingNotFound
	}
	switch {
	case listed && l.Status == PromptListingWithdrawn:
		if !s.Config(ctx).Enabled {
			return nil, ErrPromptMarketClosed
		}
		// Only an approved listing goes straight back on sale; one withdrawn before approval is reviewed again.
		if l.PublishedAt != nil {
			l.Status = PromptListingActive
		} else {
			l.Status = PromptListingPending
		}
	case !listed && (l.Status == PromptListingActive || l.Status == PromptListingPending || l.Status == PromptListingRejected):
		l.Status = PromptListingWithdrawn
	default:
		return nil, ErrPromptListingState
	}
	if err := s.repo.UpdateListing(ctx, l, nil); err != nil {
		return nil, err
	}
	return s.Get(ctx, sellerID, id)
}

// --- admin ---

// AdminList lists listings by status ("review" = pending or with a draft), with full content.
func (s *PromptMarketService) AdminList(ctx context.Context, q PromptMarketListQuery) ([]PromptListing, int64, error) {
	normalizeMarketQuery(&q, 100)
	if q.Status != "" && q.Status != "review" && q.Status != PromptListingPending && q.Status != PromptListingActive && q.Status != PromptListingRejected && q.Status != PromptListingHidden && q.Status != PromptListingWithdrawn {
		q.Status = ""
	}
	if q.Status == "" {
		q.Status = "all"
	}
	return s.repo.ListListings(ctx, q)
}

// Approve puts a pending listing live, or applies a pending change to an approved one.
func (s *PromptMarketService) Approve(ctx context.Context, id int64) (*PromptListing, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrPromptListingNotFound
	}
	var items []PromptListingItem
	switch {
	case l.Draft != nil:
		applyDraft(l, l.Draft)
		items = l.Items
		l.Draft = nil
	case l.Status == PromptListingPending:
	default:
		return nil, ErrPromptListingState
	}
	if l.Status == PromptListingPending || l.Status == PromptListingRejected {
		l.Status = PromptListingActive
	}
	if l.PublishedAt == nil {
		now := s.now()
		l.PublishedAt = &now
	}
	l.ReviewNote = ""
	if err := s.repo.UpdateListing(ctx, l, items); err != nil {
		return nil, err
	}
	return s.repo.GetListing(ctx, id)
}

// Reject turns down a pending listing, or drops a pending change (the approved version stays).
func (s *PromptMarketService) Reject(ctx context.Context, id int64, note string) (*PromptListing, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrPromptListingNotFound
	}
	switch {
	case l.Draft != nil:
		l.Draft = nil
	case l.Status == PromptListingPending:
		l.Status = PromptListingRejected
	default:
		return nil, ErrPromptListingState
	}
	l.ReviewNote = truncateRunes(strings.TrimSpace(note), 500)
	if err := s.repo.UpdateListing(ctx, l, nil); err != nil {
		return nil, err
	}
	return s.repo.GetListing(ctx, id)
}

// SetHidden takes a listing down (buyers keep what they bought) or restores it.
func (s *PromptMarketService) SetHidden(ctx context.Context, id int64, hidden bool, note string) (*PromptListing, error) {
	l, err := s.repo.GetListing(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrPromptListingNotFound
	}
	switch {
	case hidden && l.Status != PromptListingHidden:
		l.Status = PromptListingHidden
		l.ReviewNote = truncateRunes(strings.TrimSpace(note), 500)
	case !hidden && l.Status == PromptListingHidden && l.PublishedAt != nil:
		l.Status = PromptListingActive
		l.ReviewNote = ""
	default:
		return nil, ErrPromptListingState
	}
	if err := s.repo.UpdateListing(ctx, l, nil); err != nil {
		return nil, err
	}
	return s.repo.GetListing(ctx, id)
}

func (s *PromptMarketService) AdminPurchases(ctx context.Context, q PromptPurchaseQuery) ([]PromptPurchase, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 50
	}
	return s.repo.ListPurchases(ctx, q)
}

// Refund returns the price to the buyer and takes the seller's share back; the buyer loses access.
func (s *PromptMarketService) Refund(ctx context.Context, purchaseID int64) (*PromptPurchase, error) {
	p, err := s.repo.Refund(ctx, purchaseID)
	if err != nil {
		return nil, err
	}
	s.invalidateBalances(ctx, p.BuyerUserID, p.SellerUserID)
	return p, nil
}
