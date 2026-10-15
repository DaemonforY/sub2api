package service

import (
	"context"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// 静默限制 (shadow): an account the admin (or a sign-up rule) suspects of farming free credit. To its
// owner everything looks normal, but free credit — the sign-up trial subscription, gifted balance,
// promo codes — only works up to a small daily allowance; past it the gateway returns the very same
// 今日额度已用完 / 余额不足 errors a normal user gets. Accounts that have paid real money are not
// capped. Nothing here is exposed through user-facing APIs.

const (
	ShadowSourceAuto  = "auto"
	ShadowSourceAdmin = "admin"

	SettingKeyGrowthShadowLinkedFlagged = "growth_shadow_linked_flagged" // 注册指纹与已封禁/静默账号相同时自动静默限制（默认开启）
	SettingKeyGrowthShadowFreeDailyUSD  = "growth_shadow_free_daily_usd" // 静默限制账号每天可用的免费额度（美元）
	growthShadowFreeDailyUSDDefault     = 0.3
	growthShadowFreeDailyUSDMax         = 100.0

	shadowReloadInterval = 30 * time.Second
	shadowPaidCacheTTL   = 2 * time.Minute
	shadowReasonMaxRunes = 500
)

var ErrShadowReasonRequired = infraerrors.BadRequest("SHADOW_REASON_REQUIRED", "请填写静默限制原因（A reason is required）")

// UserShadow is one restricted account.
type UserShadow struct {
	UserID        int64     `json:"user_id"`
	Reason        string    `json:"reason"`
	Source        string    `json:"source"`
	RelatedUserID *int64    `json:"related_user_id,omitempty"`
	CreatedBy     *int64    `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserShadowRepository interface {
	ListShadows(ctx context.Context) ([]UserShadow, error)
	UpsertShadow(ctx context.Context, s UserShadow) error
	DeleteShadow(ctx context.Context, userID int64) error
	// HasRealPayment: a completed order paid through a payment provider (not with balance).
	HasRealPayment(ctx context.Context, userID int64) (bool, error)
	// SpentSince sums the user's usage cost (actual_cost) since `since`.
	SpentSince(ctx context.Context, userID int64, since time.Time) (float64, error)
	// FillSubscriptionDailyUsage raises the subscription's daily usage to at least limit.
	FillSubscriptionDailyUsage(ctx context.Context, subscriptionID int64, limit float64) error
}

// ShadowChecker is what other services need: is this account silently restricted?
type ShadowChecker interface {
	IsShadowed(userID int64) bool
}

// FreeCreditGate is the gateway hook (BillingCacheService.CheckBillingEligibility).
type FreeCreditGate interface {
	CheckFreeCredit(ctx context.Context, userID int64, subscriptionMode bool, group *Group, sub *UserSubscription) error
}

type shadowPaidEntry struct {
	paid bool
	at   time.Time
}

type UserShadowService struct {
	repo         UserShadowRepository
	growth       *GrowthService
	billingCache *BillingCacheService
	subs         *SubscriptionService

	mu       sync.RWMutex
	byUser   map[int64]UserShadow
	paid     sync.Map // userID → shadowPaidEntry
	now      func() time.Time
	stopOnce sync.Once
	stop     chan struct{}
}

func NewUserShadowService(repo UserShadowRepository, growth *GrowthService, billingCache *BillingCacheService, subs *SubscriptionService) *UserShadowService {
	return &UserShadowService{repo: repo, growth: growth, billingCache: billingCache, subs: subs,
		byUser: map[int64]UserShadow{}, now: time.Now, stop: make(chan struct{})}
}

// ProvideUserShadowService loads the restricted accounts, keeps them fresh, and hooks the gate into
// billing and the checker into invite bonuses and rebates.
func ProvideUserShadowService(repo UserShadowRepository, growth *GrowthService, billingCache *BillingCacheService,
	subs *SubscriptionService, affiliate *AffiliateService) *UserShadowService {
	s := NewUserShadowService(repo, growth, billingCache, subs)
	if billingCache != nil {
		billingCache.SetFreeCreditGate(s)
	}
	if growth != nil {
		growth.SetShadowChecker(s)
	}
	if affiliate != nil {
		affiliate.SetShadowChecker(s)
	}
	s.Start()
	return s
}

func (s *UserShadowService) Start() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.reload(ctx)
	cancel()
	go func() {
		t := time.NewTicker(shadowReloadInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				s.reload(ctx)
				cancel()
			}
		}
	}()
}

func (s *UserShadowService) Stop() { s.stopOnce.Do(func() { close(s.stop) }) }

func (s *UserShadowService) reload(ctx context.Context) {
	list, err := s.repo.ListShadows(ctx)
	if err != nil {
		logger.LegacyPrintf("service.user_shadow", "[Shadow] reload failed: %v", err)
		return
	}
	m := make(map[int64]UserShadow, len(list))
	for _, v := range list {
		m[v.UserID] = v
	}
	s.mu.Lock()
	s.byUser = m
	s.mu.Unlock()
}

// IsShadowed reads the in-memory set (refreshed every 30s and on every change made here).
func (s *UserShadowService) IsShadowed(userID int64) bool {
	if s == nil || userID <= 0 {
		return false
	}
	s.mu.RLock()
	_, ok := s.byUser[userID]
	s.mu.RUnlock()
	return ok
}

// Get returns the user's restriction, if any.
func (s *UserShadowService) Get(userID int64) (UserShadow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.byUser[userID]
	return v, ok
}

// Lookup returns the restrictions of the given users (admin lists).
func (s *UserShadowService) Lookup(userIDs []int64) map[int64]UserShadow {
	out := map[int64]UserShadow{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, id := range userIDs {
		if v, ok := s.byUser[id]; ok {
			out[id] = v
		}
	}
	return out
}

// Shadow restricts userID. An existing restriction keeps its original time.
func (s *UserShadowService) Shadow(ctx context.Context, userID int64, reason, source string, relatedUserID, adminID *int64) (*UserShadow, error) {
	reason = truncateRunes(strings.TrimSpace(reason), shadowReasonMaxRunes)
	if reason == "" {
		return nil, ErrShadowReasonRequired
	}
	if userID <= 0 {
		return nil, ErrUserNotFound
	}
	if source != ShadowSourceAuto {
		source = ShadowSourceAdmin
	}
	v := UserShadow{UserID: userID, Reason: reason, Source: source, RelatedUserID: relatedUserID, CreatedBy: adminID, CreatedAt: s.now()}
	if err := s.repo.UpsertShadow(ctx, v); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if old, ok := s.byUser[userID]; ok {
		v.CreatedAt = old.CreatedAt
	}
	s.byUser[userID] = v
	s.mu.Unlock()
	logger.LegacyPrintf("service.user_shadow", "[Shadow] user %d restricted (%s): %s", userID, source, reason)
	return &v, nil
}

// Unshadow lifts the restriction.
func (s *UserShadowService) Unshadow(ctx context.Context, userID int64) error {
	if err := s.repo.DeleteShadow(ctx, userID); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.byUser, userID)
	s.mu.Unlock()
	return nil
}

func (s *UserShadowService) freeDailyUSD(ctx context.Context) float64 {
	if s.growth == nil {
		return growthShadowFreeDailyUSDDefault
	}
	settings, err := s.growth.GetSettings(ctx)
	if err != nil {
		return growthShadowFreeDailyUSDDefault
	}
	return settings.ShadowFreeDailyUSD
}

func (s *UserShadowService) hasRealPayment(ctx context.Context, userID int64) (bool, error) {
	if v, ok := s.paid.Load(userID); ok {
		if e, ok := v.(shadowPaidEntry); ok && s.now().Sub(e.at) < shadowPaidCacheTTL {
			return e.paid, nil
		}
	}
	paid, err := s.repo.HasRealPayment(ctx, userID)
	if err != nil {
		return false, err
	}
	s.paid.Store(userID, shadowPaidEntry{paid: paid, at: s.now()})
	return paid, nil
}

// CheckFreeCredit caps a restricted, never-paying account at the daily free allowance. The error is
// exactly what a normal user gets when the same quota runs out; for a subscription the day's usage is
// also filled up to the limit, so the user's usage page agrees with the error. Lookup failures let
// the request through.
func (s *UserShadowService) CheckFreeCredit(ctx context.Context, userID int64, subscriptionMode bool, group *Group, sub *UserSubscription) error {
	if !s.IsShadowed(userID) {
		return nil
	}
	paid, err := s.hasRealPayment(ctx, userID)
	if err != nil || paid {
		return nil
	}
	allowance := s.freeDailyUSD(ctx)
	spent, err := s.repo.SpentSince(ctx, userID, timezone.StartOfDay(s.now()))
	if err != nil {
		logger.LegacyPrintf("service.user_shadow", "[Shadow] spend lookup for user %d failed: %v", userID, err)
		return nil
	}
	if spent < allowance {
		return nil
	}
	if !subscriptionMode || group == nil || sub == nil {
		return ErrInsufficientBalance
	}
	switch {
	case group.HasDailyLimit():
		s.fillSubscription(ctx, userID, group, sub, *group.DailyLimitUSD)
		return ErrDailyLimitExceeded
	case group.HasWeeklyLimit():
		return ErrWeeklyLimitExceeded
	case group.HasMonthlyLimit():
		return ErrMonthlyLimitExceeded
	}
	return ErrDailyLimitExceeded
}

func (s *UserShadowService) fillSubscription(ctx context.Context, userID int64, group *Group, sub *UserSubscription, limit float64) {
	if sub.DailyUsageUSD >= limit {
		return
	}
	if err := s.repo.FillSubscriptionDailyUsage(ctx, sub.ID, limit); err != nil {
		logger.LegacyPrintf("service.user_shadow", "[Shadow] fill daily usage of subscription %d failed: %v", sub.ID, err)
		return
	}
	// The handler builds the user's error message from this object.
	sub.DailyUsageUSD = limit
	if s.billingCache != nil {
		_ = s.billingCache.InvalidateSubscription(ctx, userID, group.ID)
	}
	if s.subs != nil {
		s.subs.InvalidateSubCache(userID, group.ID)
	}
}
