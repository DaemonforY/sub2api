//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memAnalyticsRepo struct {
	rows  []AnalyticsEventRow
	attrs []UserAttribution
}

func (m *memAnalyticsRepo) InsertEvents(_ context.Context, rows []AnalyticsEventRow) error {
	m.rows = append(m.rows, rows...)
	return nil
}
func (m *memAnalyticsRepo) SaveAttribution(_ context.Context, a UserAttribution) error {
	m.attrs = append(m.attrs, a)
	return nil
}
func (m *memAnalyticsRepo) DeleteEventsBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (m *memAnalyticsRepo) Overview(context.Context, time.Time, int) (*AnalyticsOverview, error) {
	return &AnalyticsOverview{}, nil
}

const browserUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/140"

func TestAnalyticsIngestCleansEvents(t *testing.T) {
	repo := &memAnalyticsRepo{}
	s := NewAnalyticsService(repo)
	n, err := s.Ingest(context.Background(), AnalyticsBatch{
		VisitorID: "visitor-123456", SessionID: "bad id!", App: "unknown",
		Attr: AnalyticsAttribution{Source: " poster<script>", Referrer: "https://www.google.com/search?q=x"},
		Events: []AnalyticsBatchEvent{
			{Name: "page_view", Path: "/keys?token=abc#x"},
			{Name: "drop_table", Path: "/"},
			{Name: "assistant_ask", Path: "learn", Props: map[string]any{"len": 12.0, "ok": true, "obj": map[string]any{}, "text": strings.Repeat("长", 300)}},
		},
	}, AnalyticsRequestMeta{UserID: 7, IP: "198.51.100.23", UserAgent: browserUA})
	require.NoError(t, err)
	require.Equal(t, 2, n, "unknown events are skipped")
	e := repo.rows[0]
	require.Equal(t, "/keys", e.Path, "query and fragment are dropped")
	require.Equal(t, "main", e.App)
	require.Empty(t, e.SessionID)
	require.Equal(t, "posterscript", e.Source)
	require.Equal(t, "198.51.100.0", e.IPPrefix)
	require.Equal(t, "desktop", e.Device)
	require.EqualValues(t, 7, e.UserID)
	p := repo.rows[1].Props
	require.Equal(t, "12", p["len"])
	require.Equal(t, "true", p["ok"])
	require.NotContains(t, p, "obj")
	require.Len(t, []rune(p["text"]), analyticsMaxPropLen)
	require.Equal(t, "/learn", repo.rows[1].Path)
}

func TestAnalyticsIngestRejectsBadBatchesAndIgnoresBots(t *testing.T) {
	repo := &memAnalyticsRepo{}
	s := NewAnalyticsService(repo)
	ctx := context.Background()
	ev := []AnalyticsBatchEvent{{Name: "page_view", Path: "/"}}

	_, err := s.Ingest(ctx, AnalyticsBatch{VisitorID: "x", Events: ev}, AnalyticsRequestMeta{UserAgent: browserUA})
	require.ErrorIs(t, err, ErrAnalyticsBadBatch)
	many := make([]AnalyticsBatchEvent, analyticsMaxBatch+1)
	_, err = s.Ingest(ctx, AnalyticsBatch{VisitorID: "visitor-123456", Events: many}, AnalyticsRequestMeta{UserAgent: browserUA})
	require.ErrorIs(t, err, ErrAnalyticsBadBatch)

	n, err := s.Ingest(ctx, AnalyticsBatch{VisitorID: "visitor-123456", Events: ev}, AnalyticsRequestMeta{UserAgent: "Googlebot/2.1"})
	require.NoError(t, err)
	require.Zero(t, n)
	n, err = s.Ingest(ctx, AnalyticsBatch{VisitorID: "visitor-123456", Events: ev}, AnalyticsRequestMeta{})
	require.NoError(t, err)
	require.Zero(t, n)
	require.Empty(t, repo.rows)
}

func TestAnalyticsAttributionChannels(t *testing.T) {
	cases := []struct {
		in   AnalyticsAttribution
		want string
	}{
		{AnalyticsAttribution{Source: "poster", Aff: "ABC"}, "poster"},
		{AnalyticsAttribution{Aff: "ABC", Referrer: "https://www.baidu.com/"}, "invite"},
		{AnalyticsAttribution{Referrer: "https://cn.bing.com/search"}, "search"},
		{AnalyticsAttribution{Referrer: "https://www.v2ex.com/t/1"}, "ref:v2ex.com"},
		{AnalyticsAttribution{Referrer: "https://canvas.hivegpt.cn/"}, "direct"},
		{AnalyticsAttribution{}, "direct"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, normalizeAttribution(c.in).Source, "%+v", c.in)
	}
	a := normalizeAttribution(AnalyticsAttribution{Referrer: "https://www.v2ex.com/t/1?x=1", Landing: "/register?aff=ABC", FirstSeen: time.Now().UnixMilli()})
	require.Equal(t, "www.v2ex.com", a.ReferrerHost)
	require.Equal(t, "/register", a.LandingPath)
	require.NotNil(t, a.FirstSeenAt)
	require.Nil(t, normalizeAttribution(AnalyticsAttribution{FirstSeen: 1}).FirstSeenAt, "implausible timestamps are dropped")
}

func TestAnalyticsRecordSignup(t *testing.T) {
	repo := &memAnalyticsRepo{}
	s := NewAnalyticsService(repo)
	s.RecordSignup(context.Background(), 9, "visitor-123456", AnalyticsAttribution{Aff: "INV1"})
	s.RecordSignup(context.Background(), 0, "visitor-123456", AnalyticsAttribution{})
	require.Len(t, repo.attrs, 1)
	require.Equal(t, UserAttribution{UserID: 9, VisitorID: "visitor-123456", Source: "invite", AffCode: "INV1", LandingPath: "/"}, repo.attrs[0])
}

func TestAnalyticsIPPrefix(t *testing.T) {
	require.Equal(t, "10.1.2.0", analyticsIPPrefix("10.1.2.3"))
	require.Equal(t, "2001:db8:1::", analyticsIPPrefix("2001:db8:1:2::3"))
	require.Empty(t, analyticsIPPrefix("nope"))
}
