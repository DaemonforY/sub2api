package service

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// First-party usage analytics (埋点): the main site, /learn and /editor post small batches of
// events; each new user's first-touch source is saved at sign-up. Sign-ups, keys, API calls and
// payments are read from their own tables, so the server never has to emit events for them.
//
// Nothing personal is stored: no input text, no full IPs (only the /24 or /48), props are a few
// short whitelisted strings. Raw events are deleted after analyticsRetention.
const (
	analyticsRetention     = 90 * 24 * time.Hour
	analyticsMaxBatch      = 30
	analyticsMaxProps      = 8
	analyticsMaxPropLen    = 100
	analyticsMaxPathLen    = 200
	analyticsMaxOverview   = 90
	analyticsDefaultWindow = 30
)

// AnalyticsEventNames is the allowlist of browser events. page_view covers every page of every app;
// the rest mark steps that a page view can't show.
var AnalyticsEventNames = map[string]bool{
	"page_view":         true,
	"signup_view":       true, // register form shown
	"signup_code_sent":  true, // email code sent
	"signup_success":    true,
	"login_success":     true,
	"key_created":       true,
	"key_config_copied": true, // copied a key / client config from the "use key" dialog
	"pricing_view":      true,
	"checkout_start":    true,
	"invite_link_copy":  true,
	"canvas_click":      true, // a link out to the canvas
	"assistant_open":    true,
	"assistant_ask":     true,
	"learn_run":         true,
	"editor_copy":       true, // copied the formatted article
	"editor_draft_push": true, // pushed to the WeChat draft box
	"editor_ai_use":     true,
}

var analyticsApps = map[string]bool{"main": true, "learn": true, "editor": true}

var (
	analyticsIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
	analyticsTagPattern = regexp.MustCompile(`[^A-Za-z0-9_.\-/\p{Han}]+`)
	analyticsBotPattern = regexp.MustCompile(`(?i)bot|spider|crawl|slurp|headless|curl|wget|python|go-http|java/|okhttp|scrapy|httpclient|preview`)
	analyticsMobileUA   = regexp.MustCompile(`(?i)mobile|android|iphone|ipod`)
	analyticsTabletUA   = regexp.MustCompile(`(?i)ipad|tablet`)
)

var ErrAnalyticsBadBatch = infraerrors.BadRequest("ANALYTICS_BAD_BATCH", "统计数据格式不正确（Invalid analytics batch）")

// AnalyticsAttribution is a visitor's first touch, kept in the browser and sent with every batch.
type AnalyticsAttribution struct {
	Source    string `json:"source"` // utm_source
	Medium    string `json:"medium"`
	Campaign  string `json:"campaign"`
	Aff       string `json:"aff"`
	Referrer  string `json:"referrer"` // full referrer URL; only the host is kept
	Landing   string `json:"landing"`  // first path
	FirstSeen int64  `json:"first_seen"`
}

type AnalyticsBatchEvent struct {
	Name  string         `json:"name"`
	Path  string         `json:"path"`
	Props map[string]any `json:"props"`
}

type AnalyticsBatch struct {
	VisitorID string                `json:"visitor_id"`
	SessionID string                `json:"session_id"`
	App       string                `json:"app"`
	Attr      AnalyticsAttribution  `json:"attr"`
	Events    []AnalyticsBatchEvent `json:"events"`
}

// AnalyticsRequestMeta is what the handler knows about the request.
type AnalyticsRequestMeta struct {
	UserID    int64
	IP        string
	UserAgent string
}

// AnalyticsEventRow is one stored event.
type AnalyticsEventRow struct {
	Event, App, Path, VisitorID, SessionID string
	UserID                                 int64
	Source, Medium, Campaign, ReferrerHost string
	Device, IPPrefix                       string
	Props                                  map[string]string
}

// UserAttribution is saved once per user at sign-up.
type UserAttribution struct {
	UserID       int64
	VisitorID    string
	Source       string
	Medium       string
	Campaign     string
	AffCode      string
	ReferrerHost string
	LandingPath  string
	FirstSeenAt  *time.Time
}

type AnalyticsRepository interface {
	InsertEvents(ctx context.Context, rows []AnalyticsEventRow) error
	SaveAttribution(ctx context.Context, a UserAttribution) error
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
	Overview(ctx context.Context, since time.Time, days int) (*AnalyticsOverview, error)
}

type AnalyticsService struct {
	repo     AnalyticsRepository
	stopOnce sync.Once
	stop     chan struct{}
	now      func() time.Time
}

func NewAnalyticsService(repo AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo, stop: make(chan struct{}), now: time.Now}
}

// ProvideAnalyticsService starts the daily clean-up of old events.
func ProvideAnalyticsService(repo AnalyticsRepository) *AnalyticsService {
	s := NewAnalyticsService(repo)
	s.Start()
	return s
}

func (s *AnalyticsService) Start() {
	go func() {
		t := time.NewTicker(6 * time.Hour)
		defer t.Stop()
		s.cleanup()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				s.cleanup()
			}
		}
	}()
}

func (s *AnalyticsService) Stop() { s.stopOnce.Do(func() { close(s.stop) }) }

func (s *AnalyticsService) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	n, err := s.repo.DeleteEventsBefore(ctx, s.now().Add(-analyticsRetention))
	if err != nil {
		logger.LegacyPrintf("service.analytics", "[Analytics] clean-up failed: %v", err)
		return
	}
	if n > 0 {
		logger.LegacyPrintf("service.analytics", "[Analytics] deleted %d events older than 90 days", n)
	}
}

// Ingest validates and stores a batch. Bots are dropped silently; unknown events are skipped.
func (s *AnalyticsService) Ingest(ctx context.Context, b AnalyticsBatch, meta AnalyticsRequestMeta) (int, error) {
	if analyticsBotPattern.MatchString(meta.UserAgent) || strings.TrimSpace(meta.UserAgent) == "" {
		return 0, nil
	}
	if !analyticsIDPattern.MatchString(b.VisitorID) || len(b.Events) == 0 || len(b.Events) > analyticsMaxBatch {
		return 0, ErrAnalyticsBadBatch
	}
	session := b.SessionID
	if !analyticsIDPattern.MatchString(session) {
		session = ""
	}
	app := b.App
	if !analyticsApps[app] {
		app = "main"
	}
	src := normalizeAttribution(b.Attr)
	device := analyticsDevice(meta.UserAgent)
	prefix := analyticsIPPrefix(meta.IP)
	rows := make([]AnalyticsEventRow, 0, len(b.Events))
	for _, e := range b.Events {
		if !AnalyticsEventNames[e.Name] {
			continue
		}
		rows = append(rows, AnalyticsEventRow{
			Event: e.Name, App: app, Path: cleanAnalyticsPath(e.Path), VisitorID: b.VisitorID, SessionID: session,
			UserID: meta.UserID, Source: src.Source, Medium: src.Medium, Campaign: src.Campaign,
			ReferrerHost: src.ReferrerHost, Device: device, IPPrefix: prefix, Props: cleanAnalyticsProps(e.Props),
		})
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.repo.InsertEvents(ctx, rows); err != nil {
		return 0, err
	}
	return len(rows), nil
}

// RecordSignup saves where a new user came from. Errors are logged, never shown to the user.
func (s *AnalyticsService) RecordSignup(ctx context.Context, userID int64, visitorID string, attr AnalyticsAttribution) {
	if s == nil || userID <= 0 {
		return
	}
	a := normalizeAttribution(attr)
	a.UserID = userID
	if analyticsIDPattern.MatchString(visitorID) {
		a.VisitorID = visitorID
	}
	if err := s.repo.SaveAttribution(ctx, a); err != nil {
		logger.LegacyPrintf("service.analytics", "[Analytics] save attribution for user %d failed: %v", userID, err)
	}
}

// Overview is the admin dashboard for the last `days` days (1–90).
func (s *AnalyticsService) Overview(ctx context.Context, days int) (*AnalyticsOverview, error) {
	if days <= 0 {
		days = analyticsDefaultWindow
	}
	if days > analyticsMaxOverview {
		days = analyticsMaxOverview
	}
	now := s.now().In(analyticsTZ)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, analyticsTZ)
	return s.repo.Overview(ctx, today.AddDate(0, 0, -(days-1)), days)
}

// analyticsTZ is the day boundary for the dashboard (the site's users are in China).
var analyticsTZ = time.FixedZone("CST", 8*3600)

// normalizeAttribution turns a first touch into a channel: utm_source, else "invite" for an
// invite link, else "search" / the referring site, else "direct".
func normalizeAttribution(a AnalyticsAttribution) UserAttribution {
	out := UserAttribution{
		Source:      analyticsTag(a.Source),
		Medium:      analyticsTag(a.Medium),
		Campaign:    analyticsTag(a.Campaign),
		AffCode:     analyticsTag(a.Aff),
		LandingPath: cleanAnalyticsPath(a.Landing),
	}
	if u, err := url.Parse(strings.TrimSpace(a.Referrer)); err == nil && u.Hostname() != "" {
		host := strings.ToLower(u.Hostname())
		if !strings.HasSuffix(host, "hivegpt.cn") {
			out.ReferrerHost = truncateRunes(host, 128)
		}
	}
	if a.FirstSeen > 0 {
		t := time.UnixMilli(a.FirstSeen)
		if t.Before(time.Now().Add(time.Minute)) && t.After(time.Now().AddDate(-1, 0, 0)) {
			out.FirstSeenAt = &t
		}
	}
	switch {
	case out.Source != "":
	case out.AffCode != "":
		out.Source = "invite"
	case out.ReferrerHost != "":
		out.Source = analyticsReferrerChannel(out.ReferrerHost)
	default:
		out.Source = "direct"
	}
	return out
}

func analyticsReferrerChannel(host string) string {
	for _, s := range []string{"google.", "bing.com", "baidu.com", "sogou.com", "so.com", "sm.cn", "yandex.", "duckduckgo.com"} {
		if strings.Contains(host, s) {
			return "search"
		}
	}
	return truncateRunes("ref:"+strings.TrimPrefix(host, "www."), 64)
}

func analyticsTag(v string) string {
	v = analyticsTagPattern.ReplaceAllString(strings.TrimSpace(v), "")
	return truncateRunes(v, 64)
}

// cleanAnalyticsPath keeps the path only (no query or fragment, which may carry tokens).
func cleanAnalyticsPath(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return truncateRunes(p, analyticsMaxPathLen)
}

func cleanAnalyticsProps(in map[string]any) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range in {
		if len(out) >= analyticsMaxProps {
			break
		}
		k = analyticsTag(k)
		if k == "" || len(k) > 32 {
			continue
		}
		var s string
		switch x := v.(type) {
		case string:
			s = x
		case float64:
			s = strconv.FormatFloat(x, 'f', -1, 64)
		case bool:
			s = strconv.FormatBool(x)
		default:
			continue
		}
		out[k] = truncateRunes(strings.TrimSpace(s), analyticsMaxPropLen)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func analyticsDevice(ua string) string {
	switch {
	case analyticsTabletUA.MatchString(ua):
		return "tablet"
	case analyticsMobileUA.MatchString(ua):
		return "mobile"
	default:
		return "desktop"
	}
}

// analyticsIPPrefix keeps the network only: 1.2.3.0 or the first 48 bits of an IPv6 address.
func analyticsIPPrefix(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String()
}

// ---------- dashboard ----------

// AnalyticsOverview is everything the admin dashboard shows for a window.
type AnalyticsOverview struct {
	Since     string                `json:"since"`
	Days      int                   `json:"days"`
	Totals    AnalyticsTotals       `json:"totals"`
	Daily     []AnalyticsDay        `json:"daily"`
	Funnel    []AnalyticsFunnelStep `json:"funnel"`
	Channels  []AnalyticsChannel    `json:"channels"`
	Pages     []AnalyticsPage       `json:"pages"`
	Features  []AnalyticsFeature    `json:"features"`
	Retention []AnalyticsCohort     `json:"retention"`
	Devices   map[string]int64      `json:"devices"`
}

type AnalyticsTotals struct {
	Visitors    int64   `json:"visitors"`
	ActiveUsers int64   `json:"active_users"` // signed-in activity or an API call in the window
	WAU         int64   `json:"wau"`          // last 7 days
	Signups     int64   `json:"signups"`
	Activated   int64   `json:"activated"` // signed up in the window and made an API call within 24h
	PaidUsers   int64   `json:"paid_users"`
	Revenue     float64 `json:"revenue"`
}

type AnalyticsDay struct {
	Day         string  `json:"day"`
	Visitors    int64   `json:"visitors"`
	PageViews   int64   `json:"page_views"`
	ActiveUsers int64   `json:"active_users"`
	APIUsers    int64   `json:"api_users"`
	Signups     int64   `json:"signups"`
	Activated   int64   `json:"activated"`
	Orders      int64   `json:"orders"`
	Revenue     float64 `json:"revenue"`
}

type AnalyticsFunnelStep struct {
	Step  string `json:"step"`
	Count int64  `json:"count"`
}

type AnalyticsChannel struct {
	Source    string  `json:"source"`
	Visitors  int64   `json:"visitors"`
	Signups   int64   `json:"signups"`
	Activated int64   `json:"activated"`
	PaidUsers int64   `json:"paid_users"`
	Revenue   float64 `json:"revenue"`
}

type AnalyticsPage struct {
	App      string `json:"app"`
	Path     string `json:"path"`
	Views    int64  `json:"views"`
	Visitors int64  `json:"visitors"`
}

type AnalyticsFeature struct {
	Event    string `json:"event"`
	Count    int64  `json:"count"`
	Visitors int64  `json:"visitors"`
	Users    int64  `json:"users"`
}

type AnalyticsCohort struct {
	Week      string `json:"week"` // Monday of the sign-up week
	Signups   int64  `json:"signups"`
	Day1      int64  `json:"day1"`    // active the day after signing up
	Day2to7   int64  `json:"day2_7"`  // active on any of days 2–7
	Day8to30  int64  `json:"day8_30"` // active on any of days 8–30
	Activated int64  `json:"activated"`
}
