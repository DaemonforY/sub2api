package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 注册试用从第一次调用开始计时: a sign-up default subscription shorter than the "unused" window
// (推广设置, default 7 days) is granted for that window first and recorded as pending. Once the user's
// first request is billed to it, its term becomes [first call, first call + trial days) — so someone
// who signs up in the evening and sets up Codex the next day still gets the whole trial day. Left
// unused, it simply expires at the end of the window. A purchase, redeem code or admin change to the
// subscription before the first call takes it out of this (the new expiry stands).

const (
	SettingKeyGrowthSignupTrialUnusedDays = "growth_signup_trial_unused_days" // 注册试用未使用多少天后作废；>0 时试用从第一次调用开始计时（0=注册即开始计时）
	growthSignupTrialUnusedDaysDefault    = 7
	growthSignupTrialUnusedDaysMax        = 90

	signupTrialStartInterval = 2 * time.Minute
	signupTrialStartBatch    = 200
)

// SignupTrialStarted is a pending trial whose term was just moved to its first call.
type SignupTrialStarted struct {
	SubscriptionID int64
	UserID         int64
	GroupID        int64
	StartsAt       time.Time
	ExpiresAt      time.Time
}

// PendingSignupTrial is a sign-up trial that hasn't been used yet.
type PendingSignupTrial struct {
	SubscriptionID   int64
	TrialDays        int
	PendingExpiresAt time.Time // the trial is void after this if still unused
}

type SignupTrialStartRepository interface {
	// Create records the subscription as pending; its current expiry is the end of the unused window.
	Create(ctx context.Context, subscriptionID, userID int64, trialDays int) error
	// StartUsed moves every open trial that has a billed request and still carries the expiry it was
	// granted with to [first call, first call + trial_days), closes it, and returns what it moved.
	StartUsed(ctx context.Context, limit int) ([]SignupTrialStarted, error)
	// CloseStale closes open trials whose subscription was changed or deleted, and unused ones whose
	// window ended before now.
	CloseStale(ctx context.Context, now time.Time) (int64, error)
	// PendingForUsers returns the users' open trials, keyed by subscription ID.
	PendingForUsers(ctx context.Context, userIDs []int64) (map[int64]PendingSignupTrial, error)
}

type SignupTrialStartService struct {
	repo   SignupTrialStartRepository
	growth *GrowthService
	subs   *SubscriptionService
	mu     sync.Mutex
	stop   chan struct{}
	once   sync.Once
}

func NewSignupTrialStartService(repo SignupTrialStartRepository, growth *GrowthService, subs *SubscriptionService) *SignupTrialStartService {
	return &SignupTrialStartService{repo: repo, growth: growth, subs: subs, stop: make(chan struct{})}
}

// ProvideSignupTrialStartService starts the job and lets the subscription list and the expiry
// reminders see pending trials.
func ProvideSignupTrialStartService(repo SignupTrialStartRepository, growth *GrowthService, subs *SubscriptionService, expiry *SubscriptionExpiryService) *SignupTrialStartService {
	s := NewSignupTrialStartService(repo, growth, subs)
	if subs != nil {
		subs.SetPendingTrialLookup(s)
	}
	if expiry != nil {
		expiry.SetPendingTrialLookup(s)
	}
	go func() {
		t := time.NewTicker(signupTrialStartInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				if _, err := s.RunOnce(ctx); err != nil {
					logger.LegacyPrintf("service.signup_trial_start", "[SignupTrialStart] run failed: %v", err)
				}
				cancel()
			}
		}
	}()
	return s
}

func (s *SignupTrialStartService) Stop() { s.once.Do(func() { close(s.stop) }) }

// UnusedDays is the 推广设置 window; 0 means trials start at sign-up.
func (s *SignupTrialStartService) UnusedDays(ctx context.Context) int {
	if s == nil || s.repo == nil {
		return 0
	}
	if s.growth == nil {
		return growthSignupTrialUnusedDaysDefault
	}
	settings, err := s.growth.GetSettings(ctx)
	if err != nil {
		logger.LegacyPrintf("service.signup_trial_start", "[SignupTrialStart] load settings failed: %v", err)
		return 0 // fall back to the old behaviour rather than granting a long trial
	}
	return settings.SignupTrialUnusedDays
}

// Register records a sign-up subscription granted for the unused window that should run trialDays
// from its first call.
func (s *SignupTrialStartService) Register(ctx context.Context, sub *UserSubscription, trialDays int) error {
	return s.repo.Create(ctx, sub.ID, sub.UserID, trialDays)
}

// RunOnce starts the trials that were used since the last run and closes stale ones. Returns how
// many started.
func (s *SignupTrialStartService) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	started, err := s.repo.StartUsed(ctx, signupTrialStartBatch)
	if err != nil {
		return 0, err
	}
	for _, st := range started {
		if s.subs != nil {
			if err := s.subs.invalidateSubscriptionCaches(st.UserID, st.GroupID); err != nil {
				logger.LegacyPrintf("service.signup_trial_start", "[SignupTrialStart] invalidate caches for user %d failed: %v", st.UserID, err)
			}
		}
	}
	if len(started) > 0 {
		logger.LegacyPrintf("service.signup_trial_start", "[SignupTrialStart] started %d sign-up trials", len(started))
	}
	if _, err := s.repo.CloseStale(ctx, time.Now()); err != nil {
		return len(started), err
	}
	return len(started), nil
}

// PendingForUsers returns the users' unused sign-up trials, keyed by subscription ID (empty on error).
func (s *SignupTrialStartService) PendingForUsers(ctx context.Context, userIDs []int64) map[int64]PendingSignupTrial {
	if s == nil || s.repo == nil || len(userIDs) == 0 {
		return nil
	}
	out, err := s.repo.PendingForUsers(ctx, userIDs)
	if err != nil {
		logger.LegacyPrintf("service.signup_trial_start", "[SignupTrialStart] pending lookup failed: %v", err)
		return nil
	}
	return out
}

// pendingTrialLookup is what the subscription list and expiry reminders need.
type pendingTrialLookup interface {
	PendingForUsers(ctx context.Context, userIDs []int64) map[int64]PendingSignupTrial
}

// markPendingTrials sets TrialPendingDays on the user's subscriptions that are unused sign-up trials.
func markPendingTrials(ctx context.Context, lookup pendingTrialLookup, userID int64, subs []UserSubscription) {
	if lookup == nil || len(subs) == 0 {
		return
	}
	pending := lookup.PendingForUsers(ctx, []int64{userID})
	for i := range subs {
		if p, ok := pending[subs[i].ID]; ok {
			subs[i].TrialPendingDays = p.TrialDays
		}
	}
}
