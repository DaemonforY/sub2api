package service

import (
	"context"
	"time"
)

// Creator stats: the author's own numbers on the canvas — totals, a period compared with the one
// before it, a daily series and the best works. Likes, favorites and follows carry timestamps;
// views and remixes are counted per day in work_daily_stats from the day that table shipped.

// CreatorStatsLocation is the time zone days are counted in.
var CreatorStatsLocation = time.FixedZone("CST", 8*3600)

// CreatorCounts is one set of counters.
type CreatorCounts struct {
	Views     int `json:"views"`
	Likes     int `json:"likes"`
	Favorites int `json:"favorites"`
	Remixes   int `json:"remixes"`
	Followers int `json:"followers"`
}

func (c *CreatorCounts) add(o CreatorCounts) {
	c.Views += o.Views
	c.Likes += o.Likes
	c.Favorites += o.Favorites
	c.Remixes += o.Remixes
	c.Followers += o.Followers
}

// CreatorDay is one day of the series ("2026-10-02").
type CreatorDay struct {
	Day string `json:"day"`
	CreatorCounts
}

// CreatorTotals are all-time numbers.
type CreatorTotals struct {
	Works       int `json:"works"`
	PublicWorks int `json:"public_works"`
	CreatorCounts
}

// CreatorWork is one of the best works with its all-time and in-period numbers.
type CreatorWork struct {
	Work            Work `json:"work"`
	PeriodViews     int  `json:"period_views"`
	PeriodLikes     int  `json:"period_likes"`
	PeriodRemixes   int  `json:"period_remixes"`
	PeriodFavorites int  `json:"period_favorites"`
}

// CreatorStats is the creator stats page.
type CreatorStats struct {
	Days     int           `json:"days"`
	Totals   CreatorTotals `json:"totals"`
	Period   CreatorCounts `json:"period"`
	Previous CreatorCounts `json:"previous"`
	Series   []CreatorDay  `json:"series"`
	TopWorks []CreatorWork `json:"top_works"`
	// TrackedSince: first day views and remixes were counted per day ("" before any).
	TrackedSince string `json:"tracked_since"`
}

var creatorStatsDays = map[int]bool{7: true, 30: true, 90: true}

// CreatorStats returns the user's stats over the last `days` days (7, 30 or 90; default 30).
func (s *CommunityService) CreatorStats(ctx context.Context, userID int64, days int) (*CreatorStats, error) {
	if !creatorStatsDays[days] {
		days = 30
	}
	today := s.now().In(CreatorStatsLocation)
	end := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, CreatorStatsLocation)
	// Two periods: the one shown and the one before it, for the comparison.
	start := end.AddDate(0, 0, -(2*days - 1))
	series, err := s.repo.CreatorSeries(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	out := &CreatorStats{Days: days, Series: []CreatorDay{}, TopWorks: []CreatorWork{}}
	periodStart := end.AddDate(0, 0, -(days - 1)).Format(time.DateOnly)
	for _, d := range series {
		if d.Day >= periodStart {
			out.Series = append(out.Series, d)
			out.Period.add(d.CreatorCounts)
		} else {
			out.Previous.add(d.CreatorCounts)
		}
	}
	if out.Totals, err = s.repo.CreatorTotals(ctx, userID); err != nil {
		return nil, err
	}
	top, err := s.repo.CreatorTopWorks(ctx, userID, end.AddDate(0, 0, -(days-1)), 10)
	if err != nil {
		return nil, err
	}
	if len(top) > 0 {
		works := make([]Work, len(top))
		for i := range top {
			works[i] = top[i].Work
		}
		s.decorateWorks(ctx, works, userID)
		for i := range top {
			top[i].Work = works[i]
		}
		out.TopWorks = top
	}
	if out.TrackedSince, err = s.repo.CreatorTrackedSince(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
