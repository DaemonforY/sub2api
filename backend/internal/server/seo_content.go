package server

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/web"
)

// seoContent feeds the frontend server's meta tags, sitemaps and llms.txt from the course and video
// services. Crawlers fetch a lot, so lists are cached for a few minutes.
type seoContent struct {
	courses *service.CourseService
	videos  *service.VideoService
	plaza   plazaSource
	faq     []web.SEOFAQ

	modelList  []web.SEOModel
	modelOK    bool
	modelUntil time.Time

	mu          sync.Mutex
	courseList  []web.SEOPage
	courseUntil time.Time
	workList    []web.SEOPage
	workUntil   time.Time
	works       map[string]seoWork
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

func newSEOContent(courses *service.CourseService, videos *service.VideoService, plaza plazaSource) *seoContent {
	s := &seoContent{courses: courses, videos: videos, plaza: plaza, works: map[string]seoWork{}}
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
		for _, m := range g.Models {
			sm := web.SEOModel{Name: m.Name, Group: g.Name}
			if p := m.Pricing; p != nil && p.BillingMode == service.BillingModeImage {
				for _, iv := range p.Intervals {
					if iv.PerRequestPrice != nil && iv.TierLabel != "" {
						sm.PerImage = append(sm.PerImage, web.SEOTierPrice{Tier: iv.TierLabel, Price: math.Round(*iv.PerRequestPrice*1e4) / 1e4})
					}
				}
			} else if p != nil {
				sm.Input, sm.Output, sm.CacheRead = perMillion(p.InputPrice), perMillion(p.OutputPrice), perMillion(p.CacheReadPrice)
			}
			out = append(out, sm)
		}
	}
	s.modelList, s.modelOK, s.modelUntil = out, ok, time.Now().Add(seoContentTTL)
	return out, ok
}

// perMillion turns a per-token USD price into the per-1M-token price, rounded to 4 decimals.
func perMillion(perToken *float64) *float64 {
	if perToken == nil {
		return nil
	}
	v := math.Round(*perToken*1e6*1e4) / 1e4
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
