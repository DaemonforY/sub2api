package service

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// Visit statistics: page views (HTML pages answered 200), distinct visitors (by IP, per day) and
// bytes served, counted in memory and added to site_daily_stats every minute.

type SiteDailyStat struct {
	SiteID   int64     `json:"-"`
	Day      time.Time `json:"day"`
	Views    int64     `json:"views"`
	Visitors int64     `json:"visitors"`
	Bytes    int64     `json:"bytes"`
}

type siteStatsKey struct {
	siteID int64
	day    string
}

type siteStatsCollector struct {
	mu      sync.Mutex
	now     func() time.Time
	pending map[siteStatsKey]*SiteDailyStat
	seen    map[siteStatsKey]map[[8]byte]struct{}
}

const siteStatsMaxVisitorsTracked = 20000

func newSiteStatsCollector(now func() time.Time) *siteStatsCollector {
	return &siteStatsCollector{now: now, pending: map[siteStatsKey]*SiteDailyStat{}, seen: map[siteStatsKey]map[[8]byte]struct{}{}}
}

func (c *siteStatsCollector) record(siteID int64, ip string, page bool, bytes int) {
	day := timezone.StartOfDay(c.now())
	key := siteStatsKey{siteID: siteID, day: day.Format("2006-01-02")}
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.pending[key]
	if st == nil {
		st = &SiteDailyStat{SiteID: siteID, Day: day}
		c.pending[key] = st
	}
	st.Bytes += int64(bytes)
	if !page {
		return
	}
	st.Views++
	visitors := c.seen[key]
	if visitors == nil {
		// A new day: forget the visitors of earlier days.
		for k := range c.seen {
			if k.day != key.day {
				delete(c.seen, k)
			}
		}
		visitors = map[[8]byte]struct{}{}
		c.seen[key] = visitors
	}
	sum := sha256.Sum256([]byte(ip))
	var id [8]byte
	copy(id[:], sum[:8])
	if _, ok := visitors[id]; !ok && len(visitors) < siteStatsMaxVisitorsTracked {
		visitors[id] = struct{}{}
		st.Visitors++
	}
}

func (c *siteStatsCollector) drain() []SiteDailyStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]SiteDailyStat, 0, len(c.pending))
	for _, st := range c.pending {
		out = append(out, *st)
	}
	c.pending = map[siteStatsKey]*SiteDailyStat{}
	return out
}

// FlushStats writes the counted visits (called every minute and on shutdown).
func (s *SiteHostingService) FlushStats(ctx context.Context) {
	stats := s.stats.drain()
	if len(stats) == 0 {
		return
	}
	if err := s.repo.AddDailyStats(ctx, stats); err != nil {
		slog.Warn("sites: saving visit stats failed", "error", err, "rows", len(stats))
	}
}

func (s *SiteHostingService) startStatsFlusher() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-s.stop:
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				s.FlushStats(ctx)
				cancel()
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				s.FlushStats(ctx)
				cancel()
			}
		}
	}()
}

// SiteStats is a site's visits by day (oldest first) and totals for the period.
type SiteStats struct {
	Days     []SiteDailyStat `json:"days"`
	Views    int64           `json:"views"`
	Visitors int64           `json:"visitors"`
	Bytes    int64           `json:"bytes"`
}

func (s *SiteHostingService) Stats(ctx context.Context, userID, siteID int64, days int) (*SiteStats, error) {
	site, err := s.owned(ctx, userID, siteID)
	if err != nil {
		return nil, err
	}
	if days < 1 || days > 90 {
		days = 30
	}
	since := timezone.StartOfDay(s.now()).AddDate(0, 0, -(days - 1))
	rows, err := s.repo.ListDailyStats(ctx, site.ID, since)
	if err != nil {
		return nil, err
	}
	byDay := map[string]SiteDailyStat{}
	for _, r := range rows {
		byDay[r.Day.Format("2006-01-02")] = r
	}
	out := &SiteStats{}
	for d := 0; d < days; d++ {
		day := since.AddDate(0, 0, d)
		st := byDay[day.Format("2006-01-02")]
		st.Day = day
		out.Days = append(out.Days, st)
		out.Views += st.Views
		out.Visitors += st.Visitors
		out.Bytes += st.Bytes
	}
	return out, nil
}
