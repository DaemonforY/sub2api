//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type libCountersFake struct {
	mu sync.Mutex
	n  map[string]int64
}

func (c *libCountersFake) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[key]++
	return c.n[key], nil
}
func (c *libCountersFake) Decr(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[key]--
	return nil
}
func (c *libCountersFake) Get(_ context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n[key], nil
}

type libBillingFake struct {
	balance map[int64]float64
	charges []float64
}

func (b *libBillingFake) Balance(_ context.Context, id int64) (float64, error) { return b.balance[id], nil }
func (b *libBillingFake) ChargeFind(_ context.Context, id int64, cost float64, _, _ string) error {
	b.balance[id] -= cost
	b.charges = append(b.charges, cost)
	return nil
}
func (b *libBillingFake) FindStats(context.Context, time.Time) (int, float64, error) {
	sum := 0.0
	for _, c := range b.charges {
		sum += c
	}
	return len(b.charges), sum, nil
}

type libSettingsFake struct{ v map[string]string }

func (s *libSettingsFake) Get(context.Context, string) (*Setting, error) { return nil, nil }
func (s *libSettingsFake) GetValue(_ context.Context, k string) (string, error) {
	return s.v[k], nil
}
func (s *libSettingsFake) Set(_ context.Context, k, v string) error { s.v[k] = v; return nil }
func (s *libSettingsFake) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (s *libSettingsFake) SetMultiple(context.Context, map[string]string) error { return nil }
func (s *libSettingsFake) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (s *libSettingsFake) Delete(context.Context, string) error                 { return nil }

func newLibPublicTest(t *testing.T) (*InspirationService, *libCountersFake, *libBillingFake, *[]string) {
	t.Helper()
	s := NewInspirationService(newLibRepoFake(), nil)
	counters := &libCountersFake{n: map[string]int64{}}
	billing := &libBillingFake{balance: map[int64]float64{1: 0.15, 2: 0}}
	s.EnablePublic(&libSettingsFake{v: map[string]string{}}, counters, billing, nil)
	s.now = func() time.Time { return time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC) }
	var models []string
	s.chat = func(_ context.Context, model, system, _ string) (string, string, error) {
		models = append(models, model)
		if len(system) > 0 && system[0:3] == libKeywordsPrompt[0:3] && system == libKeywordsPrompt {
			return `{"keywords":["纸鸢","东风"]}`, "m", nil
		}
		return `{"picks":[{"id":1,"reason":"放纸鸢"}]}`, "m", nil
	}
	return s, counters, billing, &models
}

func TestPoemsPublicFindIsFreeThenPaidAndNeverHasSongs(t *testing.T) {
	s, _, billing, models := newLibPublicTest(t)
	ctx := context.Background()
	_, err := s.CreateSong(ctx, 1, LibSong{Title: "风筝歌", Lyrics: "东风吹着纸鸢飞得很高很高"})
	require.NoError(t, err)
	st := defaultLibPublicSettings()
	st.FreeFindsPerDay, st.FindModel = 1, "fast-model"
	_, err = s.SavePublicSettings(ctx, st)
	require.NoError(t, err)

	_, err = s.PublicFind(ctx, 0, LibFindRequest{Scene: "放风筝"})
	require.ErrorIs(t, err, ErrLibLoginToFind)

	res, err := s.PublicFind(ctx, 1, LibFindRequest{Scene: "春天孩子放风筝"})
	require.NoError(t, err)
	require.True(t, res.Free)
	require.Zero(t, res.Charged)
	require.Equal(t, 0, res.FreeLeft)
	require.Len(t, res.Picks, 1)
	require.Equal(t, []string{"fast-model", "fast-model"}, *models)

	// The second run is paid: $0.10 of the 0.15 balance.
	res, err = s.PublicFind(ctx, 1, LibFindRequest{Scene: "春天孩子放风筝"})
	require.NoError(t, err)
	require.False(t, res.Free)
	require.InDelta(t, 0.1, res.Charged, 1e-9)
	require.InDelta(t, 0.05, *res.Balance, 1e-9)

	// Not enough left for a third: refused before the model is called.
	calls := len(*models)
	_, err = s.PublicFind(ctx, 1, LibFindRequest{Scene: "春天孩子放风筝"})
	require.Error(t, err)
	require.Contains(t, infraerrors.Message(err), "余额不够")
	require.Equal(t, calls, len(*models))
	require.Len(t, billing.charges, 1)

	stats, err := s.PublicStats(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, stats.RunsToday)
	require.Equal(t, 1, stats.PaidToday)
}

func TestPoemsPublicFindGivesBackTheFreeRunWhenTheModelFails(t *testing.T) {
	s, counters, _, _ := newLibPublicTest(t)
	ctx := context.Background()
	s.chat = func(context.Context, string, string, string) (string, string, error) { return "not json", "m", nil }
	_, err := s.PublicFind(ctx, 2, LibFindRequest{Scene: "秋夜思乡"})
	require.Error(t, err)
	cfg, err := s.PublicConfig(ctx, LibVisitor{UserID: 2})
	require.NoError(t, err)
	require.Equal(t, 3, cfg.FreeLeft)
	require.Zero(t, counters.n["poems:busy:2"], "the lock is released")
	require.Zero(t, counters.n["poems:cap:20261029"])
}

func TestPoemsPublicFindLimits(t *testing.T) {
	s, counters, _, _ := newLibPublicTest(t)
	ctx := context.Background()
	counters.n["poems:busy:1"] = 1
	_, err := s.PublicFind(ctx, 1, LibFindRequest{Scene: "秋夜"})
	require.ErrorIs(t, err, ErrLibFindBusy)
	counters.n["poems:busy:1"] = 0

	st := defaultLibPublicSettings()
	st.DailyFindCap = 1
	_, err = s.SavePublicSettings(ctx, st)
	require.NoError(t, err)
	_, err = s.PublicFind(ctx, 1, LibFindRequest{Scene: "秋夜"})
	require.NoError(t, err)
	_, err = s.PublicFind(ctx, 1, LibFindRequest{Scene: "秋夜"})
	require.ErrorIs(t, err, ErrLibFindCapped)
	cfg, _ := s.PublicConfig(ctx, LibVisitor{UserID: 1})
	require.Equal(t, 2, cfg.FreeLeft, "a capped run doesn't use a free one")

	st.FindEnabled = false
	_, _ = s.SavePublicSettings(ctx, st)
	_, err = s.PublicFind(ctx, 1, LibFindRequest{Scene: "秋夜"})
	require.ErrorIs(t, err, ErrLibFindOff)
}

func TestPoemsPublicSearchIsLimitedAndHidesAdminMarks(t *testing.T) {
	s, _, _, _ := newLibPublicTest(t)
	ctx := context.Background()
	st := defaultLibPublicSettings()
	st.GuestPerMinute, st.GuestPerDay = 2, 2
	_, _ = s.SavePublicSettings(ctx, st)
	guest := LibVisitor{IP: "1.2.3.4"}

	poems, _, err := s.PublicSearch(ctx, guest, LibPoemQuery{Q: "纸鸢", Limit: 500})
	require.NoError(t, err)
	require.Len(t, poems, 1)
	require.False(t, poems[0].Starred)
	_, err = s.PublicPoem(ctx, guest, 1)
	require.NoError(t, err)
	_, _, err = s.PublicSearch(ctx, guest, LibPoemQuery{Q: "纸鸢"})
	require.Error(t, err, "per-minute limit")
	require.Contains(t, infraerrors.Message(err), "每分钟")

	// Another minute, but the day is used up.
	s.now = func() time.Time { return time.Date(2026, 10, 29, 10, 5, 0, 0, time.UTC) }
	_, _, err = s.PublicSearch(ctx, guest, LibPoemQuery{Q: "纸鸢"})
	require.Error(t, err)
	require.Contains(t, infraerrors.Message(err), "登录")

	_, _, err = s.PublicSearch(ctx, LibVisitor{UserID: 9}, LibPoemQuery{Q: "纸鸢", Offset: 300})
	require.ErrorIs(t, err, ErrLibDeepPage)

	st.Enabled = false
	_, _ = s.SavePublicSettings(ctx, st)
	_, _, err = s.PublicSearch(ctx, LibVisitor{UserID: 9}, LibPoemQuery{Q: "纸鸢"})
	require.ErrorIs(t, err, ErrLibPublicOff)
}
