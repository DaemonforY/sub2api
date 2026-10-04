package service

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AI 学习 L3: the learner showcase wall (certificate holders who chose to show their project, with
// covers of their public canvas works) and learning insights for the admin page.

const (
	learnShowcaseMax      = 48
	learnShowcaseImages   = 3
	learnShowcaseCacheTTL = 5 * time.Minute
	learnInsightDays      = 14
)

// LearnShowcaseItem is one card on the wall.
type LearnShowcaseItem struct {
	Code        string              `json:"code"`
	Track       string              `json:"track"`
	TrackTitle  string              `json:"track_title"`
	DisplayName string              `json:"display_name"`
	ProjectURL  string              `json:"project_url"`
	IssuedAt    time.Time           `json:"issued_at"`
	Works       []LearnShowcaseWork `json:"works"`
}

// LearnShowcaseWork is a public canvas work shown on a card.
type LearnShowcaseWork struct {
	URL   string `json:"url"`
	Image string `json:"image"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

type learnShowcaseCache struct {
	mu    sync.Mutex
	at    map[string]time.Time
	items map[string][]LearnShowcaseItem
}

func (c *learnShowcaseCache) get(key string, now time.Time) ([]LearnShowcaseItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at, ok := c.at[key]; ok && now.Sub(at) < learnShowcaseCacheTTL {
		return c.items[key], true
	}
	return nil, false
}

func (c *learnShowcaseCache) put(key string, now time.Time, items []LearnShowcaseItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.at == nil {
		c.at, c.items = map[string]time.Time{}, map[string][]LearnShowcaseItem{}
	}
	c.at[key], c.items[key] = now, items
}

func (c *learnShowcaseCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at, c.items = nil, nil
}

// showcaseWorks: the holder's public works for the card (site works first for the coding tracks).
func (s *LearnService) showcaseWorks(ctx context.Context, userID int64, track string) []LearnShowcaseWork {
	if s.sources.Works == nil || track == "d" {
		return []LearnShowcaseWork{}
	}
	works, err := s.sources.Works.ListWorks(ctx, WorkQuery{Feed: "user", ViewerID: userID, UserID: userID, Limit: 24})
	if err != nil {
		return []LearnShowcaseWork{}
	}
	var sites, others []LearnShowcaseWork
	canvas := strings.TrimRight(s.sources.CanvasURL, "/")
	for _, w := range works {
		if w.Status != WorkStatusApproved || w.Visibility != WorkVisibilityPublic {
			continue
		}
		thumb := CommunityMediaURL(w.CoverThumb)
		if thumb == "" {
			thumb = CommunityMediaURL(w.CoverFile)
		}
		if thumb == "" {
			continue
		}
		item := LearnShowcaseWork{URL: canvas + "/w/" + strconv.FormatInt(w.ID, 10), Image: thumb, Title: w.Title, Kind: w.Kind}
		if w.Kind == "site" {
			sites = append(sites, item)
		} else {
			others = append(others, item)
		}
	}
	list := others
	if track == "a" || track == "c" {
		list = append(sites, others...)
	}
	if len(list) > learnShowcaseImages {
		list = list[:learnShowcaseImages]
	}
	if list == nil {
		list = []LearnShowcaseWork{}
	}
	return list
}

// Showcase is the public learner wall (track "" = every track).
func (s *LearnService) Showcase(ctx context.Context, track string, limit int) ([]LearnShowcaseItem, error) {
	if track != "" && s.catalog.track(track) == nil {
		return nil, ErrLearnTrack
	}
	if limit < 1 || limit > learnShowcaseMax {
		limit = learnShowcaseMax
	}
	key := track + ":" + strconv.Itoa(limit)
	if items, ok := s.showcase.get(key, s.now()); ok {
		return items, nil
	}
	certs, err := s.repo.Showcase(ctx, track, limit)
	if err != nil {
		return nil, err
	}
	items := make([]LearnShowcaseItem, 0, len(certs))
	for _, c := range certs {
		title := c.Track
		if t := s.catalog.track(c.Track); t != nil {
			title = t.Title
		}
		items = append(items, LearnShowcaseItem{Code: c.Code, Track: c.Track, TrackTitle: title, DisplayName: c.DisplayName,
			ProjectURL: c.ProjectURL, IssuedAt: c.IssuedAt, Works: s.showcaseWorks(ctx, c.UserID, c.Track)})
	}
	s.showcase.put(key, s.now(), items)
	return items, nil
}

// SetShowcase lets a certificate holder show or hide their project on the wall.
func (s *LearnService) SetShowcase(ctx context.Context, userID int64, track string, on bool) error {
	if s.catalog.track(track) == nil {
		return ErrLearnTrack
	}
	if err := s.repo.SetShowcase(ctx, userID, track, on); err != nil {
		return err
	}
	s.showcase.clear()
	return nil
}

// SetShowcaseHidden takes an entry off the wall (admin) or puts it back.
func (s *LearnService) SetShowcaseHidden(ctx context.Context, code string, hidden bool) error {
	if err := s.repo.SetShowcaseHidden(ctx, strings.ToUpper(strings.TrimSpace(code)), hidden); err != nil {
		return err
	}
	s.showcase.clear()
	return nil
}

// --- Insights --------------------------------------------------------------------------------

type LearnTrackProgress struct {
	Track   string
	UserID  int64
	Lessons int
}

type LearnDay struct {
	Date         string `json:"date"`
	Learners     int    `json:"learners"`
	Completions  int    `json:"completions"`
	Runs         int    `json:"runs"`
	Tutor        int    `json:"tutor"`
	Interviews   int    `json:"interviews"`
	Certificates int    `json:"certificates"`
}

type LearnQuizStat struct {
	Takers   int `json:"takers"`
	Passed   int `json:"passed"`
	AvgScore int `json:"avg_score"`
	Attempts int `json:"attempts"`
}

// LearnFunnel: how far learners got in a track.
type LearnFunnel struct {
	Track        string `json:"track"`
	Title        string `json:"title"`
	Lessons      int    `json:"lessons"`
	Started      int    `json:"started"`
	Half         int    `json:"half"`
	Finished     int    `json:"finished"`
	Certificates int    `json:"certificates"`
}

type LearnInsights struct {
	Funnels []LearnFunnel            `json:"funnels"`
	Days    []LearnDay               `json:"days"`
	Quizzes map[string]LearnQuizStat `json:"quizzes"`
}

// Insights (admin): per-track funnel, the last two weeks day by day, and quiz results per lesson.
func (s *LearnService) Insights(ctx context.Context) (*LearnInsights, error) {
	progress, err := s.repo.TrackProgress(ctx)
	if err != nil {
		return nil, err
	}
	certs, err := s.repo.CertificateCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := &LearnInsights{Funnels: []LearnFunnel{}}
	for _, t := range s.catalog.Tracks {
		f := LearnFunnel{Track: t.ID, Title: t.Title, Lessons: len(t.Lessons), Certificates: certs[t.ID]}
		for _, p := range progress {
			if p.Track != t.ID || p.Lessons == 0 {
				continue
			}
			f.Started++
			if p.Lessons*2 >= len(t.Lessons) {
				f.Half++
			}
			if p.Lessons >= len(t.Lessons) {
				f.Finished++
			}
		}
		out.Funnels = append(out.Funnels, f)
	}

	today := learnDayStart(s.now())
	since := today.AddDate(0, 0, -(learnInsightDays - 1))
	days, err := s.repo.Daily(ctx, since)
	if err != nil {
		return nil, err
	}
	byDate := map[string]LearnDay{}
	for _, d := range days {
		byDate[d.Date] = d
	}
	for i := 0; i < learnInsightDays; i++ {
		date := since.AddDate(0, 0, i).Format("2006-01-02")
		d, ok := byDate[date]
		if !ok {
			d = LearnDay{Date: date}
		}
		out.Days = append(out.Days, d)
	}

	if out.Quizzes, err = s.repo.QuizStats(ctx); err != nil {
		return nil, err
	}
	return out, nil
}
