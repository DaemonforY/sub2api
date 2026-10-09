package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// Unpaid-order reminder (未付款订单提醒): an order that timed out unpaid is often someone who was
// interrupted at the payment step. About an hour after it expires, if they haven't paid or started
// another order since, they get one email with a link back to the purchase page. At most one such
// email per user per week; off until enabled on 推广设置, and users can unsubscribe.
const (
	SettingKeyGrowthAbandonedOrderReminder = "growth_abandoned_order_reminder" // 未付款订单提醒（默认关闭）

	abandonedOrderAfter    = time.Hour      // wait this long after the order expired
	abandonedOrderUntil    = 24 * time.Hour // older ones are left alone
	abandonedOrderBatch    = 50
	abandonedOrderInterval = 15 * time.Minute
)

// AbandonedOrder is an expired, unpaid order the reminder may be about.
type AbandonedOrder struct {
	OrderID   int64
	UserID    int64
	Email     string
	Username  string
	OrderType string  // balance | subscription
	Amount    float64 // what they were about to pay
}

type AbandonedOrderRepository interface {
	// DueAbandonedOrders: the latest balance / subscription order per active user that expired
	// unpaid in [from, to), when the user has no paid or pending order created after it.
	DueAbandonedOrders(ctx context.Context, from, to time.Time, limit int) ([]AbandonedOrder, error)
}

type abandonedOrderMailer interface {
	Send(ctx context.Context, input NotificationEmailSendInput) error
}

type AbandonedOrderReminderService struct {
	repo     AbandonedOrderRepository
	settings SettingRepository
	mailer   abandonedOrderMailer
	now      func() time.Time
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

func NewAbandonedOrderReminderService(repo AbandonedOrderRepository, settings SettingRepository, mailer abandonedOrderMailer) *AbandonedOrderReminderService {
	return &AbandonedOrderReminderService{repo: repo, settings: settings, mailer: mailer, now: time.Now, stop: make(chan struct{})}
}

// ProvideAbandonedOrderReminderService starts the run every 15 minutes.
func ProvideAbandonedOrderReminderService(repo AbandonedOrderRepository, settings SettingRepository, notify *NotificationEmailService) *AbandonedOrderReminderService {
	var mailer abandonedOrderMailer
	if notify != nil {
		mailer = notify
	}
	s := NewAbandonedOrderReminderService(repo, settings, mailer)
	go func() {
		t := time.NewTicker(abandonedOrderInterval)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if _, err := s.RunOnce(ctx); err != nil {
					logger.LegacyPrintf("service.abandoned_order", "[AbandonedOrder] run failed: %v", err)
				}
				cancel()
			}
		}
	}()
	return s
}

func (s *AbandonedOrderReminderService) Stop() {
	if s != nil {
		s.stopOnce.Do(func() { close(s.stop) })
	}
}

func (s *AbandonedOrderReminderService) Enabled(ctx context.Context) bool {
	if s == nil || s.settings == nil {
		return false
	}
	v, err := s.settings.GetValue(ctx, SettingKeyGrowthAbandonedOrderReminder)
	return err == nil && v == "true"
}

// RunOnce sends the reminders that are due. Returns how many went out.
func (s *AbandonedOrderReminderService) RunOnce(ctx context.Context) (int, error) {
	if s == nil || s.mailer == nil || !s.Enabled(ctx) {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	orders, err := s.repo.DueAbandonedOrders(ctx, now.Add(-abandonedOrderUntil), now.Add(-abandonedOrderAfter), abandonedOrderBatch)
	if err != nil {
		return 0, err
	}
	year, week := now.ISOWeek()
	sent := 0
	for _, o := range orders {
		if !reminderAddressOK(o.Email) {
			continue
		}
		// Keyed by user and ISO week: the notification service sends each key once.
		if err := s.mailer.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventPaymentOrderAbandoned,
			RecipientEmail: o.Email,
			RecipientName:  firstNonEmpty(o.Username, o.Email),
			UserID:         o.UserID,
			SourceType:     "abandoned_order",
			SourceID:       strconv.FormatInt(o.UserID, 10),
			ReminderKey:    fmt.Sprintf("%d-w%02d", year, week),
			Variables:      abandonedOrderVariables(o),
		}); err != nil {
			logger.LegacyPrintf("service.abandoned_order", "[AbandonedOrder] send for order %d failed: %v", o.OrderID, err)
			continue
		}
		sent++
	}
	return sent, nil
}

func abandonedOrderVariables(o AbandonedOrder) map[string]string {
	item, itemEN := "余额充值", "balance top-up"
	if o.OrderType == "subscription" {
		item, itemEN = "订阅套餐", "subscription"
	}
	return map[string]string{
		"order_item":    item,
		"order_item_en": itemEN,
		"order_amount":  strconv.FormatFloat(o.Amount, 'f', 2, 64),
		"purchase_url":  RechargeURL + "?utm_source=email&utm_medium=abandoned_order&utm_campaign=" + o.OrderType,
	}
}
