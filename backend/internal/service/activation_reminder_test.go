//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memReminderRepo struct {
	due  []ActivationReminderUser
	sent map[int64]bool
	from time.Time
	to   time.Time
}

func (m *memReminderRepo) DueUsers(_ context.Context, from, to time.Time, limit int) ([]ActivationReminderUser, error) {
	m.from, m.to = from, to
	var out []ActivationReminderUser
	for _, u := range m.due {
		if !m.sent[u.ID] && len(out) < limit {
			out = append(out, u)
		}
	}
	return out, nil
}
func (m *memReminderRepo) MarkSent(_ context.Context, id int64) error { m.sent[id] = true; return nil }
func (m *memReminderRepo) SentCount(context.Context) (int64, error) {
	return int64(len(m.sent)), nil
}

type memMailer struct {
	to      []string
	failAt  int
	subject string
	body    string
}

func (m *memMailer) SendEmail(_ context.Context, to, subject, body string) error {
	if m.failAt > 0 && len(m.to)+1 == m.failAt {
		return errors.New("smtp down")
	}
	m.to = append(m.to, to)
	m.subject, m.body = subject, body
	return nil
}

func newReminderFixture() (*ActivationReminderService, *memReminderRepo, *memMailer, *mockSettingRepo) {
	repo := &memReminderRepo{sent: map[int64]bool{}, due: []ActivationReminderUser{
		{ID: 1, Email: "a@qq.com", Username: "小明<b>"},
		{ID: 2, Email: "oauth-123@linuxdo.invalid"},
		{ID: 3, Email: "c@163.com"},
	}}
	settings := newMockSettingRepo()
	mailer := &memMailer{}
	s := NewActivationReminderService(repo, settings, mailer)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, repo, mailer, settings
}

func TestActivationReminderIsOffByDefault(t *testing.T) {
	s, _, mailer, _ := newReminderFixture()
	n, err := s.RunOnce(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
	require.Empty(t, mailer.to)
}

func TestActivationReminderSendsOnceToRealAddresses(t *testing.T) {
	s, repo, mailer, settings := newReminderFixture()
	ctx := context.Background()
	require.NoError(t, s.SetEnabled(ctx, true))
	settings.data[SettingKeyFrontendURL] = "https://hivegpt.cn/"

	n, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, []string{"a@qq.com", "c@163.com"}, mailer.to)
	require.True(t, repo.sent[2], "placeholder addresses are marked and never retried")
	require.Equal(t, s.now().Add(-72*time.Hour), repo.from)
	require.Equal(t, s.now().Add(-24*time.Hour), repo.to)
	require.Equal(t, activationReminderSubject, mailer.subject)

	n, err = s.RunOnce(ctx)
	require.NoError(t, err)
	require.Zero(t, n, "one email per user")
}

func TestActivationReminderStopsWhenMailFails(t *testing.T) {
	s, repo, mailer, _ := newReminderFixture()
	ctx := context.Background()
	require.NoError(t, s.SetEnabled(ctx, true))
	mailer.failAt = 1
	n, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	require.False(t, repo.sent[1], "retried next hour")
}

func TestActivationReminderBodyAndStatus(t *testing.T) {
	body := ActivationReminderBody("https://hivegpt.cn", "小明<b>")
	require.Contains(t, body, "你好，小明&lt;b&gt;")
	require.Contains(t, body, `href="https://hivegpt.cn/keys?action=create&amp;utm_source=email&amp;utm_medium=activation&amp;utm_campaign=key"`)
	require.Contains(t, body, "/learn/codex/hivegpt?utm_source=email")
	require.Contains(t, body, "只发这一封")
	require.Contains(t, body, `href="https://canvas.hivegpt.cn/?utm_source=email&amp;utm_medium=activation&amp;utm_campaign=canvas"`)
	require.Contains(t, body, "/purchase?utm_source=email")
	require.NotContains(t, body, "Claude Code", "Claude Code cannot use HiveGPT's groups yet")
	require.NotContains(t, activationReminderSubject, "Claude Code")
	require.False(t, strings.Contains(ActivationReminderBody("https://x", ""), "你好，"))

	s, _, _, _ := newReminderFixture()
	st, err := s.Status(context.Background())
	require.NoError(t, err)
	require.False(t, st.Enabled)
	require.Equal(t, 2, st.Due)
	require.Contains(t, st.Preview, "https://hivegpt.cn/dashboard")
}

func TestReminderAddressOK(t *testing.T) {
	for _, ok := range []string{"a@qq.com", "x.y@school.edu.cn"} {
		require.True(t, reminderAddressOK(ok), ok)
	}
	for _, bad := range []string{"", "nobody", "a@", "u@wechat.invalid", "u@test.local", "u@localhost"} {
		require.False(t, reminderAddressOK(bad), bad)
	}
}
