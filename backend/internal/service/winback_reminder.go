package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Win-back email (老用户召回): someone who used the API and then stopped 14–60 days ago gets one email
// about what is new, with their balance, at most once a quarter. The text is an admin-editable
// template (邮件模板) so the "what's new" part can be kept current. Off until enabled on 推广设置.
const (
	SettingKeyGrowthWinbackEmail = "growth_winback_email" // 老用户召回邮件（默认关闭）

	winbackIdleAfter = 14 * 24 * time.Hour // last call at least this long ago
	winbackIdleUntil = 60 * 24 * time.Hour // and not longer ago than this
	winbackBatch     = 50
	winbackInterval  = 6 * time.Hour
)

// LapsedUser used the API before but not for a while.
type LapsedUser struct {
	UserID   int64
	Email    string
	Username string
	Balance  float64
	LastUsed time.Time
}

type WinbackRepository interface {
	// DueLapsedUsers: active non-admin users whose last API call falls in [from, to).
	DueLapsedUsers(ctx context.Context, from, to time.Time, limit int) ([]LapsedUser, error)
}

type winbackMailer interface {
	Send(ctx context.Context, input NotificationEmailSendInput) error
}

type WinbackReminderService struct {
	repo     WinbackRepository
	settings SettingRepository
	mailer   winbackMailer
	offer    trialOffer
	now      func() time.Time
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

func NewWinbackReminderService(repo WinbackRepository, settings SettingRepository, mailer winbackMailer, offer trialOffer) *WinbackReminderService {
	return &WinbackReminderService{repo: repo, settings: settings, mailer: mailer, offer: offer, now: time.Now, stop: make(chan struct{})}
}

// ProvideWinbackReminderService starts the run every 6 hours.
func ProvideWinbackReminderService(repo WinbackRepository, settings SettingRepository, notify *NotificationEmailService, growth *GrowthService) *WinbackReminderService {
	var mailer winbackMailer
	if notify != nil {
		mailer = notify
	}
	var offer trialOffer
	if growth != nil {
		offer = growth
	}
	s := NewWinbackReminderService(repo, settings, mailer, offer)
	go func() {
		t := time.NewTicker(winbackInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if _, err := s.RunOnce(ctx); err != nil {
					logger.LegacyPrintf("service.winback", "[Winback] run failed: %v", err)
				}
				cancel()
			}
		}
	}()
	return s
}

func (s *WinbackReminderService) Stop() {
	if s != nil {
		s.stopOnce.Do(func() { close(s.stop) })
	}
}

func (s *WinbackReminderService) Enabled(ctx context.Context) bool {
	if s == nil || s.settings == nil {
		return false
	}
	v, err := s.settings.GetValue(ctx, SettingKeyGrowthWinbackEmail)
	return err == nil && v == "true"
}

// RunOnce sends the emails that are due. Returns how many went out.
func (s *WinbackReminderService) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.mailer == nil || !s.Enabled(ctx) {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	users, err := s.repo.DueLapsedUsers(ctx, now.Add(-winbackIdleUntil), now.Add(-winbackIdleAfter), winbackBatch)
	if err != nil {
		return 0, err
	}
	quarter := fmt.Sprintf("%d-q%d", now.Year(), (int(now.Month())-1)/3+1)
	sent := 0
	for _, u := range users {
		if !reminderAddressOK(u.Email) {
			continue
		}
		// Keyed by user and quarter: the notification service sends each key once.
		if err := s.mailer.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventGrowthWinback,
			RecipientEmail: u.Email,
			RecipientName:  firstNonEmpty(u.Username, u.Email),
			UserID:         u.UserID,
			SourceType:     "winback",
			SourceID:       strconv.FormatInt(u.UserID, 10),
			ReminderKey:    quarter,
			Variables:      s.variables(ctx, u, now),
		}); err != nil {
			logger.LegacyPrintf("service.winback", "[Winback] send to user %d failed: %v", u.UserID, err)
			continue
		}
		sent++
	}
	return sent, nil
}

func (s *WinbackReminderService) variables(ctx context.Context, u LapsedUser, now time.Time) map[string]string {
	utm := func(base, campaign string) string {
		return base + "?utm_source=email&utm_medium=winback&utm_campaign=" + campaign
	}
	vars := map[string]string{
		"days_away":     strconv.Itoa(int(now.Sub(u.LastUsed).Hours() / 24)),
		"balance":       strconv.FormatFloat(u.Balance, 'f', 2, 64),
		"dashboard_url": utm("https://hivegpt.cn/dashboard", "dashboard"),
		"canvas_url":    utm("https://canvas.hivegpt.cn/image", "canvas"),
		"prompts_url":   utm("https://hivegpt.cn/learn/prompts/", "prompts"),
		"connect_url":   utm("https://hivegpt.cn/learn/connect/", "connect"),
		"pricing_url":   utm("https://hivegpt.cn/pricing", "pricing"),
		"offer_text":    "",
		"offer_text_en": "",
	}
	// Users who have never paid still qualify for the first top-up bonus; say so when it is on.
	if s.offer != nil {
		if g, err := s.offer.GetSettings(ctx); err == nil && g.FirstTopupBonusPercent > 0 && u.Balance < 1 {
			pct := strconv.FormatFloat(g.FirstTopupBonusPercent, 'f', -1, 64)
			vars["offer_text"] = "还没充过值的话，首次充值再送 " + pct + "%。"
			vars["offer_text_en"] = "If you haven't topped up yet, your first top-up gets " + pct + "% extra."
		}
	}
	return vars
}
