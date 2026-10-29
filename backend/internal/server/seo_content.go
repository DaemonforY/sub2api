package server

import (
	"context"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"
)

// seoContent feeds the frontend server's meta tags, sitemaps and llms.txt from the course and video
// services. Crawlers fetch a lot, so lists are cached for a few minutes.
type seoContent struct {
	courses  *service.CourseService
	videos   *service.VideoService
	contests *service.ContestService
	plaza    plazaSource
	faq      []web.SEOFAQ

	modelList  []web.SEOModel
	modelOK    bool
	modelUntil time.Time

	mu          sync.Mutex
	courseList  []web.SEOPage
	courseUntil time.Time
	workList    []web.SEOPage
	workUntil   time.Time
	works       map[string]seoWork
	contestHits map[string]seoWork
}

// plazaSource is the model plaza as an anonymous visitor sees it (handler.ModelPlazaHandler).
type plazaSource interface {
	PublicGroups(ctx context.Context) ([]service.PlazaGroup, bool)
}

type seoWork struct {
	page  *web.SEOPage
	until time.Time
}

const (
	seoContentTTL   = 10 * time.Minute
	seoGalleryPages = 10 // × 48 works in the sitemap
	seoWorkCacheMax = 2000
)

func newSEOContent(courses *service.CourseService, videos *service.VideoService, contests *service.ContestService, plaza plazaSource) *seoContent {
	s := &seoContent{courses: courses, videos: videos, contests: contests, plaza: plaza, works: map[string]seoWork{}, contestHits: map[string]seoWork{}}
	for _, f := range service.AssistantFAQ() {
		s.faq = append(s.faq, web.SEOFAQ{Question: f.Title, Answer: f.Text, Link: f.URL})
	}
	return s
}

func (s *seoContent) FAQ() []web.SEOFAQ { return s.faq }

func (s *seoContent) Models(ctx context.Context) ([]web.SEOModel, bool) {
	if s.plaza == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.modelUntil) {
		return s.modelList, s.modelOK
	}
	groups, ok := s.plaza.PublicGroups(ctx)
	var out []web.SEOModel
	for _, g := range groups {
		// Prices paid, as the plaza page shows them: the group multiplier, or for per-image models the
		// group's independent image multiplier when it has one (resolveImageRateMultiplier).
		rate := g.RateMultiplier
		if rate <= 0 {
			rate = 1
		}
		imageRate := rate
		if g.ImageRateIndependent {
			imageRate = g.ImageRateMultiplier
		}
		for _, m := range g.Models {
			sm := web.SEOModel{Name: m.Name, Group: g.Name}
			if p := m.Pricing; p != nil && p.BillingMode == service.BillingModeImage {
				for _, iv := range p.Intervals {
					if iv.PerRequestPrice != nil && iv.TierLabel != "" {
						sm.PerImage = append(sm.PerImage, web.SEOTierPrice{Tier: iv.TierLabel, Price: math.Round(*iv.PerRequestPrice*imageRate*1e4) / 1e4})
					}
				}
			} else if p != nil {
				sm.Input, sm.Output, sm.CacheRead = perMillion(p.InputPrice, rate), perMillion(p.OutputPrice, rate), perMillion(p.CacheReadPrice, rate)
			}
			out = append(out, sm)
		}
	}
	s.modelList, s.modelOK, s.modelUntil = out, ok, time.Now().Add(seoContentTTL)
	return out, ok
}

// perMillion turns a per-token USD price times rate into the per-1M-token price, rounded to 4 decimals.
func perMillion(perToken *float64, rate float64) *float64 {
	if perToken == nil {
		return nil
	}
	v := math.Round(*perToken*rate*1e6*1e4) / 1e4
	return &v
}

func (s *seoContent) Courses(ctx context.Context) ([]web.SEOPage, error) {
	if s.courses == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.courseUntil) {
		return s.courseList, nil
	}
	list, err := s.courses.Courses(ctx, 0)
	if err != nil {
		return s.courseList, err
	}
	out := make([]web.SEOPage, 0, len(list))
	for _, c := range list {
		out = append(out, web.SEOPage{
			Path:        "/courses/" + c.Slug,
			Title:       c.Title,
			Description: c.Subtitle,
			Image:       c.CoverURL,
			Price:       c.CurrentPrice,
			Updated:     c.UpdatedAt,
		})
	}
	s.courseList, s.courseUntil = out, time.Now().Add(seoContentTTL)
	return out, nil
}

func (s *seoContent) VideoWorks(ctx context.Context) ([]web.SEOPage, error) {
	if s.videos == nil {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.workUntil) {
		return s.workList, nil
	}
	var out []web.SEOPage
	for page := 1; page <= seoGalleryPages; page++ {
		cards, total, err := s.videos.Gallery(ctx, service.VideoGalleryQuery{Sort: "new", Page: page, PageSize: 48})
		if err != nil {
			return s.workList, err
		}
		for _, w := range cards {
			out = append(out, web.SEOPage{Path: "/w/" + w.ID, Title: w.Title, Description: w.Prompt, Updated: w.UpdatedAt})
		}
		if len(cards) == 0 || page*48 >= total {
			break
		}
	}
	s.workList, s.workUntil = out, time.Now().Add(seoContentTTL)
	return out, nil
}

func (s *seoContent) VideoWork(ctx context.Context, id string) (*web.SEOPage, error) {
	if s.videos == nil || id == "" || len(id) > 64 || strings.ContainsAny(id, "/?#") {
		return nil, nil
	}
	s.mu.Lock()
	if w, ok := s.works[id]; ok && time.Now().Before(w.until) {
		s.mu.Unlock()
		return w.page, nil
	}
	s.mu.Unlock()

	var page *web.SEOPage
	p, err := s.videos.Work(ctx, id)
	if err == nil && p != nil {
		desc := p.Prompt
		if p.Spec != nil && strings.TrimSpace(p.Spec.Summary) != "" {
			desc = p.Spec.Summary
		}
		page = &web.SEOPage{Path: "/w/" + p.ID, Title: p.Title, Description: desc, Updated: p.UpdatedAt}
	} else if err != nil && err != service.ErrVideoNotFound {
		return nil, err
	}

	s.mu.Lock()
	if len(s.works) >= seoWorkCacheMax {
		s.works = map[string]seoWork{}
	}
	s.works[id] = seoWork{page: page, until: time.Now().Add(seoContentTTL)}
	s.mu.Unlock()
	return page, nil
}

// contestPreviewTTL is short: vote counts and titles change during a contest, and a link preview
// should not keep showing a withdrawn entry for long.
const contestPreviewTTL = time.Minute

func (s *seoContent) ContestPage(ctx context.Context, contestID, entryID int64) (*web.SEOPage, error) {
	if s.contests == nil || contestID <= 0 || entryID < 0 {
		return nil, nil
	}
	key := strconv.FormatInt(contestID, 10) + "/" + strconv.FormatInt(entryID, 10)
	s.mu.Lock()
	if hit, ok := s.contestHits[key]; ok && time.Now().Before(hit.until) {
		s.mu.Unlock()
		return hit.page, nil
	}
	s.mu.Unlock()

	page, err := s.contestPage(ctx, contestID, entryID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if len(s.contestHits) >= seoWorkCacheMax {
		s.contestHits = map[string]seoWork{}
	}
	s.contestHits[key] = seoWork{page: page, until: time.Now().Add(contestPreviewTTL)}
	s.mu.Unlock()
	return page, nil
}

func (s *seoContent) contestPage(ctx context.Context, contestID, entryID int64) (*web.SEOPage, error) {
	detail, err := s.contests.GetPublicContest(ctx, contestID, 0)
	if err != nil {
		if infraerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	c := detail.Contest
	if entryID == 0 {
		desc := c.Description
		if desc == "" {
			desc = c.Title + "：投稿作品、给喜欢的作品点赞，查看排行榜。"
		}
		return &web.SEOPage{Path: "/contests/" + strconv.FormatInt(c.ID, 10), Title: c.Title, Description: desc, Image: service.ContestCoverOrDefault(c)}, nil
	}
	e, err := s.contests.PublicEntry(ctx, contestID, entryID, 0)
	if err != nil {
		if infraerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	desc := "「" + e.Title + "」正在参加" + c.Title + "，来给 " + e.AuthorName + " 点个赞吧。"
	if e.Description != "" {
		desc += e.Description
	}
	return &web.SEOPage{
		Path:        "/contests/" + strconv.FormatInt(c.ID, 10) + "/e/" + strconv.FormatInt(e.ID, 10),
		Title:       e.Title + " · " + c.Title,
		Description: desc,
		Image:       e.ImageURL,
	}, nil
}
