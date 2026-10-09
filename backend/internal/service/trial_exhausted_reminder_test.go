//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memTrialRepo struct {
	due   []TrialUser
	after time.Time
	max   float64
}

func (m *memTrialRepo) DueTrialUsers(_ context.Context, after time.Time, max float64, _ int) ([]TrialUser, error) {
	m.after, m.max = after, max
	return m.due, nil
}

type staticOffer struct{ g GrowthSettings }

func (o staticOffer) GetSettings(context.Context) (*GrowthSettings, error) { g := o.g; return &g, nil }

func TestTrialExhaustedReminder(t *testing.T) {
	repo := &memTrialRepo{due: []TrialUser{{UserID: 7, Email: "a@qq.com", Username: "小明"}, {UserID: 8, Email: "u@wechat.invalid"}}}
	settings := newMockSettingRepo()
	mailer := &memNotifyMailer{}
	s := NewTrialExhaustedReminderService(repo, settings, mailer, staticOffer{GrowthSettings{FirstTopupBonusPercent: 20, FirstTopupBonusCap: 10}})
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	ctx := context.Background()

	n, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "off by default")

	settings.data[SettingKeyGrowthTrialExhaustedEmail] = "true"
	n, err = s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n, "placeholder addresses are skipped")
	require.Equal(t, now.Add(-30*24*time.Hour), repo.after)
	require.InDelta(t, 0.3, repo.max, 1e-9)

	in := mailer.sent[0]
	require.Equal(t, NotificationEmailEventGrowthTrialExhausted, in.Event)
	require.Equal(t, "7", in.SourceID)
	require.Equal(t, "once", in.ReminderKey, "one email per user, ever")
	require.Equal(t, "现在首次充值再送 20%（最多送 $10），付款后自动到账。", in.Variables["offer_text"])
	require.Contains(t, in.Variables["purchase_url"], "utm_medium=trial_exhausted")

	plain := NewTrialExhaustedReminderService(repo, settings, &memNotifyMailer{}, staticOffer{})
	require.Equal(t, "充值后马上就能接着用。", plain.offerVariables(ctx)["offer_text"], "no bonus line when the first top-up bonus is off")
}

func TestTrialExhaustedEmailTemplates(t *testing.T) {
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)
	for _, locale := range []string{"zh", "en"} {
		p, err := svc.PreviewTemplate(context.Background(), NotificationEmailPreviewInput{
			Event: NotificationEmailEventGrowthTrialExhausted, Locale: locale,
			Variables: map[string]string{"purchase_url": "https://hivegpt.cn/purchase?utm_source=email"},
		})
		require.NoError(t, err)
		require.Contains(t, p.HTML, `href="https://hivegpt.cn/purchase?utm_source=email"`, locale)
	}
}
