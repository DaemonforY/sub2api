//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memAbandonedRepo struct {
	due      []AbandonedOrder
	from, to time.Time
}

func (m *memAbandonedRepo) DueAbandonedOrders(_ context.Context, from, to time.Time, _ int) ([]AbandonedOrder, error) {
	m.from, m.to = from, to
	return m.due, nil
}

type memNotifyMailer struct{ sent []NotificationEmailSendInput }

func (m *memNotifyMailer) Send(_ context.Context, in NotificationEmailSendInput) error {
	m.sent = append(m.sent, in)
	return nil
}

func TestAbandonedOrderReminder(t *testing.T) {
	repo := &memAbandonedRepo{due: []AbandonedOrder{
		{OrderID: 11, UserID: 1, Email: "a@qq.com", Username: "小明", OrderType: "balance", Amount: 50},
		{OrderID: 12, UserID: 2, Email: "oauth-2@linuxdo.invalid", OrderType: "subscription", Amount: 120},
	}}
	settings := newMockSettingRepo()
	mailer := &memNotifyMailer{}
	s := NewAbandonedOrderReminderService(repo, settings, mailer)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ctx := context.Background()

	n, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "off by default")

	settings.data[SettingKeyGrowthAbandonedOrderReminder] = "true"
	n, err = s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n, "placeholder addresses are skipped")
	require.Equal(t, now.Add(-24*time.Hour), repo.from)
	require.Equal(t, now.Add(-time.Hour), repo.to)

	in := mailer.sent[0]
	require.Equal(t, NotificationEmailEventPaymentOrderAbandoned, in.Event)
	require.Equal(t, "abandoned_order", in.SourceType)
	require.Equal(t, "1", in.SourceID, "keyed by user")
	require.Equal(t, "2026-w41", in.ReminderKey, "at most once per ISO week")
	require.Equal(t, "余额充值", in.Variables["order_item"])
	require.Equal(t, "50.00", in.Variables["order_amount"])
	require.Contains(t, in.Variables["purchase_url"], RechargeURL+"?utm_source=email&utm_medium=abandoned_order")
}

func TestAbandonedOrderEmailTemplates(t *testing.T) {
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)
	for _, locale := range []string{"zh", "en"} {
		p, err := svc.PreviewTemplate(context.Background(), NotificationEmailPreviewInput{
			Event: NotificationEmailEventPaymentOrderAbandoned, Locale: locale,
			Variables: map[string]string{"purchase_url": "https://hivegpt.cn/purchase?utm_source=email"},
		})
		require.NoError(t, err)
		require.Contains(t, p.HTML, `href="https://hivegpt.cn/purchase?utm_source=email"`, locale)
	}
}
