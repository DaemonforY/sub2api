//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memWinbackRepo struct {
	due      []LapsedUser
	from, to time.Time
}

func (m *memWinbackRepo) DueLapsedUsers(_ context.Context, from, to time.Time, _ int) ([]LapsedUser, error) {
	m.from, m.to = from, to
	return m.due, nil
}

func TestWinbackReminder(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	repo := &memWinbackRepo{due: []LapsedUser{
		{UserID: 5, Email: "a@qq.com", Username: "小红", Balance: 0.4, LastUsed: now.Add(-21 * 24 * time.Hour)},
		{UserID: 6, Email: "u@wechat.invalid", LastUsed: now.Add(-20 * 24 * time.Hour)},
	}}
	settings := newMockSettingRepo()
	mailer := &memNotifyMailer{}
	s := NewWinbackReminderService(repo, settings, mailer, staticOffer{GrowthSettings{FirstTopupBonusPercent: 20}})
	s.now = func() time.Time { return now }
	ctx := context.Background()

	n, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "off by default")

	settings.data[SettingKeyGrowthWinbackEmail] = "true"
	n, err = s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n, "placeholder addresses are skipped")
	require.Equal(t, now.Add(-60*24*time.Hour), repo.from)
	require.Equal(t, now.Add(-14*24*time.Hour), repo.to)

	in := mailer.sent[0]
	require.Equal(t, NotificationEmailEventGrowthWinback, in.Event)
	require.Equal(t, "5", in.SourceID)
	require.Equal(t, "2026-q4", in.ReminderKey, "at most once a quarter")
	require.Equal(t, "21", in.Variables["days_away"])
	require.Equal(t, "0.40", in.Variables["balance"])
	require.Contains(t, in.Variables["canvas_url"], "utm_medium=winback")
	require.Equal(t, "还没充过值的话，首次充值再送 20%。", in.Variables["offer_text"])

	rich := s.variables(ctx, LapsedUser{Balance: 30, LastUsed: now}, now)
	require.Empty(t, rich["offer_text"], "no top-up pitch to people with plenty of balance")
}

func TestWinbackEmailTemplates(t *testing.T) {
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)
	for _, locale := range []string{"zh", "en"} {
		p, err := svc.PreviewTemplate(context.Background(), NotificationEmailPreviewInput{
			Event: NotificationEmailEventGrowthWinback, Locale: locale,
			Variables: map[string]string{"dashboard_url": "https://hivegpt.cn/dashboard?utm_source=email"},
		})
		require.NoError(t, err)
		require.Contains(t, p.HTML, `href="https://hivegpt.cn/dashboard?utm_source=email"`, locale)
	}
}
