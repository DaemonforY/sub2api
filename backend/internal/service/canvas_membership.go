package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 创作会员 (canvas creator membership): while it lasts, images saved from the canvas image
// workbench come without the 「AI 生成 · <site>」 watermark. Plans (name, days, CNY price) are an
// admin setting; a plan is bought as a payment order (order_type = membership) and extends the
// membership from the later of now and its current end. The canvas logs every unwatermarked save
// (《人工智能生成合成内容标识办法》第九条: keep who received unlabelled content ≥ 6 months).
const (
	SettingKeyCanvasMembership = "canvas_membership"

	canvasMembershipMaxPlans   = 6
	canvasMembershipMaxDays    = 3660
	canvasMembershipMaxPrice   = 100000.0
	canvasMembershipMaxGrant   = 3660
	canvasUnmarkedSaveKeepDays = 200
)

var (
	ErrCanvasMembershipUnavailable = infraerrors.BadRequest("MEMBERSHIP_UNAVAILABLE", "创作会员暂未开放购买（Creator membership is not on sale）")
	ErrCanvasMembershipPlan        = infraerrors.NotFound("MEMBERSHIP_PLAN_NOT_FOUND", "这个会员套餐不存在或已下架，请刷新页面重新选择（Membership plan not found）")
	ErrCanvasMembershipCurrency    = infraerrors.BadRequest("MEMBERSHIP_CURRENCY", "创作会员只能用人民币支付，请换一种支付方式（Membership is paid in CNY only）")
	ErrCanvasMembershipTerms       = infraerrors.BadRequest("MEMBERSHIP_TERMS_REQUIRED", "请先勾选同意：公开发布无水印图片时自行标注「AI 生成」（Please accept the AI-label terms）")
	errCanvasMembershipBadConfig   = infraerrors.BadRequest("MEMBERSHIP_BAD_CONFIG", "会员套餐设置有误：最多 6 个套餐，名称 1–20 个字，天数 1–3660，价格 0.01–100000（Invalid membership plans）")
	errCanvasMembershipBadGrant    = infraerrors.BadRequest("MEMBERSHIP_BAD_GRANT", "赠送天数需在 1 到 3660 之间（Days must be 1–3660）")
)

// CanvasMembershipPlan is one plan on sale. ID is stable so orders and the WeChat payment resume
// can refer to it; the price is always read from here, never from the request.
type CanvasMembershipPlan struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Days          int     `json:"days"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price,omitempty"`
}

type CanvasMembershipConfig struct {
	Enabled bool                   `json:"enabled"`
	Plans   []CanvasMembershipPlan `json:"plans"`
}

// CanvasMembershipStatus is what the purchase page and the canvas show.
type CanvasMembershipStatus struct {
	OnSale bool                   `json:"on_sale"`
	Plans  []CanvasMembershipPlan `json:"plans"`
	Until  *time.Time             `json:"until,omitempty"`
}

type CanvasUnmarkedSave struct {
	UserID    int64
	Width     int
	Height    int
	ClientIP  string
	UserAgent string
}

// CanvasMember is a row of the admin member list.
type CanvasMember struct {
	UserID          int64      `json:"user_id"`
	Email           string     `json:"email"`
	Username        string     `json:"username"`
	MemberUntil     time.Time  `json:"member_until"`
	TermsAcceptedAt *time.Time `json:"terms_accepted_at,omitempty"`
	UnmarkedSaves   int64      `json:"unmarked_saves"`
}

type CanvasMembershipRepository interface {
	Until(ctx context.Context, userID int64) (*time.Time, error)
	// Extend adds days to max(now, member_until); acceptTerms stamps terms_accepted_at.
	Extend(ctx context.Context, userID int64, days int, acceptTerms bool) (time.Time, error)
	// Shorten removes days (refunds), never below now.
	Shorten(ctx context.Context, userID int64, days int) error
	AcceptTerms(ctx context.Context, userID int64) error
	TermsAccepted(ctx context.Context, userID int64) (bool, error)
	LogUnmarkedSave(ctx context.Context, save CanvasUnmarkedSave) error
	PruneUnmarkedSaves(ctx context.Context, before time.Time) (int64, error)
	ListMembers(ctx context.Context, activeOnly bool, limit int) ([]CanvasMember, error)
}

type CanvasMembershipService struct {
	repo     CanvasMembershipRepository
	settings SettingRepository
	now      func() time.Time
}

func NewCanvasMembershipService(repo CanvasMembershipRepository, settings SettingRepository) *CanvasMembershipService {
	return &CanvasMembershipService{repo: repo, settings: settings, now: time.Now}
}

// Config is the admin setting; an unset or unreadable setting means off with no plans.
func (s *CanvasMembershipService) Config(ctx context.Context) CanvasMembershipConfig {
	cfg := CanvasMembershipConfig{Plans: []CanvasMembershipPlan{}}
	raw, err := s.settings.GetValue(ctx, SettingKeyCanvasMembership)
	if err != nil || strings.TrimSpace(raw) == "" {
		return cfg
	}
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return CanvasMembershipConfig{Plans: []CanvasMembershipPlan{}}
	}
	if cfg.Plans == nil {
		cfg.Plans = []CanvasMembershipPlan{}
	}
	return cfg
}

func (s *CanvasMembershipService) SaveConfig(ctx context.Context, cfg CanvasMembershipConfig) (CanvasMembershipConfig, error) {
	if len(cfg.Plans) > canvasMembershipMaxPlans || (cfg.Enabled && len(cfg.Plans) == 0) {
		return cfg, errCanvasMembershipBadConfig
	}
	var maxID int64
	for _, p := range cfg.Plans {
		if p.ID > maxID {
			maxID = p.ID
		}
	}
	seen := map[int64]bool{}
	for i := range cfg.Plans {
		p := &cfg.Plans[i]
		p.Name = strings.TrimSpace(p.Name)
		n := utf8.RuneCountInString(p.Name)
		if n == 0 || n > 20 || p.Days < 1 || p.Days > canvasMembershipMaxDays || !validMembershipPrice(p.Price) || (p.OriginalPrice != 0 && !validMembershipPrice(p.OriginalPrice)) {
			return cfg, errCanvasMembershipBadConfig
		}
		p.Price = math.Round(p.Price*100) / 100
		p.OriginalPrice = math.Round(p.OriginalPrice*100) / 100
		// New plans get the next id; ids of existing plans never change.
		if p.ID <= 0 || seen[p.ID] {
			maxID++
			p.ID = maxID
		}
		seen[p.ID] = true
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, s.settings.Set(ctx, SettingKeyCanvasMembership, string(raw))
}

func validMembershipPrice(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0.01 && v <= canvasMembershipMaxPrice
}

// OnSale reports whether plans can be bought now.
func (s *CanvasMembershipService) OnSale(ctx context.Context) bool {
	cfg := s.Config(ctx)
	return cfg.Enabled && len(cfg.Plans) > 0
}

// PlanForOrder returns the plan a new order is for, rejecting it while sales are off.
func (s *CanvasMembershipService) PlanForOrder(ctx context.Context, planID int64) (*CanvasMembershipPlan, error) {
	cfg := s.Config(ctx)
	if !cfg.Enabled {
		return nil, ErrCanvasMembershipUnavailable
	}
	for _, p := range cfg.Plans {
		if p.ID == planID {
			plan := p
			return &plan, nil
		}
	}
	return nil, ErrCanvasMembershipPlan
}

// Until is when the user's membership ends; nil when they are not (or no longer) a member.
func (s *CanvasMembershipService) Until(ctx context.Context, userID int64) (*time.Time, error) {
	until, err := s.repo.Until(ctx, userID)
	if err != nil || until == nil || !until.After(s.now()) {
		return nil, err
	}
	return until, nil
}

func (s *CanvasMembershipService) Status(ctx context.Context, userID int64) (*CanvasMembershipStatus, error) {
	cfg := s.Config(ctx)
	status := &CanvasMembershipStatus{OnSale: cfg.Enabled && len(cfg.Plans) > 0, Plans: []CanvasMembershipPlan{}}
	if status.OnSale {
		status.Plans = cfg.Plans
	}
	if userID > 0 {
		until, err := s.Until(ctx, userID)
		if err != nil {
			return nil, err
		}
		status.Until = until
	}
	return status, nil
}

// AcceptTerms records that the buyer agreed to label published unwatermarked images themselves;
// orders cannot be created without it.
func (s *CanvasMembershipService) AcceptTerms(ctx context.Context, userID int64) error {
	return s.repo.AcceptTerms(ctx, userID)
}

func (s *CanvasMembershipService) RequireTerms(ctx context.Context, userID int64) error {
	ok, err := s.repo.TermsAccepted(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCanvasMembershipTerms
	}
	return nil
}

// FulfillOrder extends the membership by the days of a paid order.
func (s *CanvasMembershipService) FulfillOrder(ctx context.Context, userID int64, days int) error {
	if days <= 0 {
		return errors.New("membership order without days")
	}
	_, err := s.repo.Extend(ctx, userID, days, false)
	return err
}

// RefundOrder takes the days of a refunded order back.
func (s *CanvasMembershipService) RefundOrder(ctx context.Context, userID int64, days int) error {
	if days <= 0 {
		return nil
	}
	return s.repo.Shorten(ctx, userID, days)
}

// Grant gives (days > 0) or takes back (days < 0) membership days by hand.
func (s *CanvasMembershipService) Grant(ctx context.Context, userID int64, days int) (*time.Time, error) {
	if days == 0 || days > canvasMembershipMaxGrant || days < -canvasMembershipMaxGrant {
		return nil, errCanvasMembershipBadGrant
	}
	if days > 0 {
		if _, err := s.repo.Extend(ctx, userID, days, false); err != nil {
			return nil, err
		}
	} else if err := s.repo.Shorten(ctx, userID, -days); err != nil {
		return nil, err
	}
	return s.Until(ctx, userID)
}

// RecordUnmarkedSave logs one unwatermarked save; only members may save unwatermarked.
func (s *CanvasMembershipService) RecordUnmarkedSave(ctx context.Context, save CanvasUnmarkedSave) error {
	until, err := s.Until(ctx, save.UserID)
	if err != nil {
		return err
	}
	if until == nil {
		return infraerrors.Forbidden("NOT_A_MEMBER", "创作会员已到期，保存的图片会带水印（Membership expired）")
	}
	save.ClientIP = truncateRunes(save.ClientIP, 64)
	save.UserAgent = truncateRunes(save.UserAgent, 255)
	save.Width = clampDimension(save.Width)
	save.Height = clampDimension(save.Height)
	if err := s.repo.LogUnmarkedSave(ctx, save); err != nil {
		return err
	}
	// Cheap enough to do inline now and then; keeps the table at ~200 days.
	if s.now().UnixNano()%50 == 0 {
		_, _ = s.repo.PruneUnmarkedSaves(ctx, s.now().AddDate(0, 0, -canvasUnmarkedSaveKeepDays))
	}
	return nil
}

func (s *CanvasMembershipService) ListMembers(ctx context.Context, activeOnly bool) ([]CanvasMember, error) {
	return s.repo.ListMembers(ctx, activeOnly, 500)
}

func clampDimension(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100000 {
		return 100000
	}
	return v
}
