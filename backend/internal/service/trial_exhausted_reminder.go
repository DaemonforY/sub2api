package service

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Trial-exhausted email (体验金用完提醒): someone who signed up recently, actually used the product and
// has burned through the sign-up credit without ever paying is at the moment they are most likely to
// buy. They get one email, with the first top-up bonus when it is on. Off until enabled on 推广设置.
const (
	SettingKeyGrowthTrialExhaustedEmail = "growth_trial_exhausted_email" // 体验金用完提醒（默认关闭）

	trialExhaustedBalance  = 0.3
	trialExhaustedSignedUp = 30 * 24 * time.Hour
	trialExhaustedBatch    = 50
	trialExhaustedInterval = time.Hour
)

// TrialUser is a recent sign-up whose trial credit has run out.
type TrialUser struct {
	UserID   int64
	Email    string
	Username string
}

type TrialExhaustedRepository interface {
	// DueTrialUsers: active non-admin users created after signedUpAfter with balance below
	// maxBalance, at least one API call, no gateway-paid order and no running subscription.
	DueTrialUsers(ctx context.Context, signedUpAfter time.Time, maxBalance float64, limit int) ([]TrialUser, error)
}

type trialMailer interface {
	Send(ctx context.Context, input NotificationEmailSendInput) error
}

type trialOffer interface {
	GetSettings(ctx context.Context) (*GrowthSettings, error)
}

type TrialExhaustedReminderService struct {
	repo     TrialExhaustedRepository
	settings SettingRepository
	mailer   trialMailer
	offer    trialOffer
	now      func() time.Time
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

func NewTrialExhaustedReminderService(repo TrialExhaustedRepository, settings SettingRepository, mailer trialMailer, offer trialOffer) *TrialExhaustedReminderService {
	return &TrialExhaustedReminderService{repo: repo, settings: settings, mailer: mailer, offer: offer, now: time.Now, stop: make(chan struct{})}
}

// ProvideTrialExhaustedReminderService starts the hourly run.
func ProvideTrialExhaustedReminderService(repo TrialExhaustedRepository, settings SettingRepository, notify *NotificationEmailService, growth *GrowthService) *TrialExhaustedReminderService {
	var mailer trialMailer
	if notify != nil {
		mailer = notify
	}
	var offer trialOffer
	if growth != nil {
		offer = growth
	}
	s := NewTrialExhaustedReminderService(repo, settings, mailer, offer)
	go func() {
		t := time.NewTicker(trialExhaustedInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if _, err := s.RunOnce(ctx); err != nil {
					logger.LegacyPrintf("service.trial_exhausted", "[TrialExhausted] run failed: %v", err)
				}
				cancel()
			}
		}
	}()
	return s
}

func (s *TrialExhaustedReminderService) Stop() {
	if s != nil {
		s.stopOnce.Do(func() { close(s.stop) })
	}
}

func (s *TrialExhaustedReminderService) Enabled(ctx context.Context) bool {
	if s == nil || s.settings == nil {
		return false
	}
	v, err := s.settings.GetValue(ctx, SettingKeyGrowthTrialExhaustedEmail)
	return err == nil && v == "true"
}

// RunOnce sends the emails that are due. Returns how many went out.
func (s *TrialExhaustedReminderService) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.mailer == nil || !s.Enabled(ctx) {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	users, err := s.repo.DueTrialUsers(ctx, s.now().Add(-trialExhaustedSignedUp), trialExhaustedBalance, trialExhaustedBatch)
	if err != nil {
		return 0, err
	}
	vars := s.offerVariables(ctx)
	sent := 0
	for _, u := range users {
		if !reminderAddressOK(u.Email) {
			continue
		}
		// One per user, ever: the notification service sends each (source, key) once.
		if err := s.mailer.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventGrowthTrialExhausted,
			RecipientEmail: u.Email,
			RecipientName:  firstNonEmpty(u.Username, u.Email),
			UserID:         u.UserID,
			SourceType:     "trial_exhausted",
			SourceID:       strconv.FormatInt(u.UserID, 10),
			ReminderKey:    "once",
			Variables:      vars,
		}); err != nil {
			logger.LegacyPrintf("service.trial_exhausted", "[TrialExhausted] send to user %d failed: %v", u.UserID, err)
			continue
		}
		sent++
	}
	return sent, nil
}

func (s *TrialExhaustedReminderService) offerVariables(ctx context.Context) map[string]string {
	vars := map[string]string{
		"purchase_url":  RechargeURL + "?utm_source=email&utm_medium=trial_exhausted",
		"offer_text":    "充值后马上就能接着用。",
		"offer_text_en": "Top up to keep going.",
	}
	if s.offer == nil {
		return vars
	}
	g, err := s.offer.GetSettings(ctx)
	if err != nil || g.FirstTopupBonusPercent <= 0 {
		return vars
	}
	pct := strconv.FormatFloat(g.FirstTopupBonusPercent, 'f', -1, 64)
	vars["offer_text"] = "现在首次充值再送 " + pct + "%"
	vars["offer_text_en"] = "Your first top-up gets " + pct + "% extra"
	if g.FirstTopupBonusCap > 0 {
		cap := strconv.FormatFloat(g.FirstTopupBonusCap, 'f', -1, 64)
		vars["offer_text"] += "（最多送 $" + cap + "）"
		vars["offer_text_en"] += " (up to $" + cap + ")"
	}
	vars["offer_text"] += "，付款后自动到账。"
	vars["offer_text_en"] += ", credited automatically."
	return vars
}
