package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 诗词灵感 for users (/poems): the poem search and 按画面找 of the 灵感库, without the songs.
//
//   - Search and poem pages are open to visitors, limited per IP (per user when signed in) per minute
//     and per day, and paged no deeper than libPublicMaxOffset, so the corpus can't be pulled cheaply.
//   - 按画面找 needs an account: a few free runs a day, then a price taken from the balance after a
//     run succeeds (the balance is checked first). One run at a time per user, and a daily cap for the
//     whole site bounds the text-model spend on the admin's key.
//
// Counters live in Redis (the support assistant's counter store); days are Beijing days.

const (
	SettingKeyInspirationPublic = "inspiration_public"

	libPublicMaxOffset = 270 // 10 pages of 30
	libPublicPageSize  = 30
	libFindLockTTL     = 6 * time.Minute
)

// LibPublicSettings is the admin's configuration of the user-facing pages.
type LibPublicSettings struct {
	Enabled         bool    `json:"enabled"`            // search and poem pages
	FindEnabled     bool    `json:"find_enabled"`       // 按画面找
	FreeFindsPerDay int     `json:"free_finds_per_day"` // per user
	FindPrice       float64 `json:"find_price"`         // USD per run after the free ones
	FindModel       string  `json:"find_model"`         // "" = the 课本诗画 text model
	DailyFindCap    int     `json:"daily_find_cap"`     // all users together
	GuestPerMinute  int     `json:"guest_per_minute"`   // searches + poem pages per IP
	GuestPerDay     int     `json:"guest_per_day"`
	UserPerMinute   int     `json:"user_per_minute"`
	UserPerDay      int     `json:"user_per_day"`
	FindCount       int     `json:"find_count"` // picks per run
}

func defaultLibPublicSettings() LibPublicSettings {
	return LibPublicSettings{Enabled: true, FindEnabled: true, FreeFindsPerDay: 3, FindPrice: 0.1, DailyFindCap: 500,
		GuestPerMinute: 30, GuestPerDay: 300, UserPerMinute: 60, UserPerDay: 2000, FindCount: 8}
}

func normalizeLibPublicSettings(st *LibPublicSettings) {
	st.FreeFindsPerDay = min(max(st.FreeFindsPerDay, 0), 100)
	st.FindPrice = math.Round(math.Min(math.Max(st.FindPrice, 0), 10)*10000) / 10000
	st.FindModel = strings.TrimSpace(st.FindModel)
	st.DailyFindCap = min(max(st.DailyFindCap, 0), 100000)
	st.GuestPerMinute = min(max(st.GuestPerMinute, 1), 600)
	st.GuestPerDay = min(max(st.GuestPerDay, 1), 100000)
	st.UserPerMinute = min(max(st.UserPerMinute, 1), 600)
	st.UserPerDay = min(max(st.UserPerDay, 1), 100000)
	st.FindCount = min(max(st.FindCount, 1), 12)
}

// LibVisitor is who calls a user-facing endpoint: a signed-in user, else an IP.
type LibVisitor struct {
	UserID int64
	IP     string
}

func (v LibVisitor) key() string {
	if v.UserID > 0 {
		return "u" + strconv.FormatInt(v.UserID, 10)
	}
	return "ip" + v.IP
}

// LibFindBilling takes paid runs from the balance (inspirationRepository).
type LibFindBilling interface {
	Balance(ctx context.Context, userID int64) (float64, error)
	ChargeFind(ctx context.Context, userID int64, cost float64, scene, model string) error
	FindStats(ctx context.Context, since time.Time) (runs int, revenue float64, err error)
}

type libBalanceCache interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// EnablePublic turns on the user-facing pages (settings, Redis counters, billing).
func (s *InspirationService) EnablePublic(settings SettingRepository, counters AssistantQuotaCache, billing LibFindBilling, cache libBalanceCache) {
	s.settings, s.counters, s.billing, s.balanceCache = settings, counters, billing, cache
}

var (
	ErrLibPublicOff   = infraerrors.NotFound("POEMS_OFF", "诗词灵感暂未开放（Not available）")
	ErrLibFindOff     = infraerrors.BadRequest("POEMS_FIND_OFF", "按画面找诗暂未开放（Not available）")
	ErrLibLoginToFind = infraerrors.Unauthorized("POEMS_LOGIN", "登录后才能按画面找诗（Sign in to use this）")
	ErrLibFindBusy    = infraerrors.TooManyRequests("POEMS_FIND_BUSY", "上一次还在找，请等它完成（Your previous search is still running）")
	ErrLibFindCapped  = infraerrors.TooManyRequests("POEMS_FIND_CAP", "今天按画面找诗的人太多了，明天再来试试（Daily limit for everyone reached）")
	ErrLibDeepPage    = infraerrors.BadRequest("POEMS_DEEP_PAGE", "最多翻到第 10 页，请换个更具体的词搜（Only the first 10 pages）")
)

func errLibTooFast(perMinute int) error {
	return infraerrors.TooManyRequests("POEMS_RATE", fmt.Sprintf("搜得太快了，每分钟最多 %d 次，稍等一下再试（Too many searches per minute）", perMinute))
}

func errLibDaily(perDay int, guest bool) error {
	if guest {
		return infraerrors.TooManyRequests("POEMS_DAILY", fmt.Sprintf("今天已经看了 %d 次，登录后每天可以看更多（Daily limit reached; sign in for more）", perDay))
	}
	return infraerrors.TooManyRequests("POEMS_DAILY", fmt.Sprintf("今天已经看了 %d 次，明天再来（Daily limit reached）", perDay))
}

func errLibBalance(price float64) error {
	return infraerrors.BadRequest("POEMS_BALANCE", fmt.Sprintf("今天的免费次数用完了，再找一次要 $%.2f，余额不够，请先充值（Insufficient balance）", price))
}

func (s *InspirationService) PublicSettings(ctx context.Context) LibPublicSettings {
	st := defaultLibPublicSettings()
	if s.settings == nil {
		return st
	}
	if raw, err := s.settings.GetValue(ctx, SettingKeyInspirationPublic); err == nil && strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	normalizeLibPublicSettings(&st)
	return st
}

func (s *InspirationService) SavePublicSettings(ctx context.Context, st LibPublicSettings) (LibPublicSettings, error) {
	if s.settings == nil {
		return st, ErrLibInvalid
	}
	normalizeLibPublicSettings(&st)
	if len(st.FindModel) > 100 {
		return st, ErrLibInvalid
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return st, err
	}
	if err := s.settings.Set(ctx, SettingKeyInspirationPublic, string(raw)); err != nil {
		return st, err
	}
	return st, nil
}

func (s *InspirationService) dayKey() string {
	return shihuaDayStart(s.clock()).Format("20060102")
}

func (s *InspirationService) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// view counts one search or poem page against the visitor's per-minute and per-day limits.
func (s *InspirationService) view(ctx context.Context, v LibVisitor, st LibPublicSettings) error {
	if s.counters == nil {
		return nil
	}
	perMinute, perDay := st.GuestPerMinute, st.GuestPerDay
	if v.UserID > 0 {
		perMinute, perDay = st.UserPerMinute, st.UserPerDay
	}
	now := s.clock()
	minute, err := s.counters.Incr(ctx, "poems:m:"+v.key()+":"+now.Format("200601021504"), 2*time.Minute)
	if err != nil {
		slog.Warn("poems: rate counter", "err", err)
		return nil
	}
	if minute > int64(perMinute) {
		return errLibTooFast(perMinute)
	}
	day, err := s.counters.Incr(ctx, "poems:d:"+v.key()+":"+s.dayKey(), 26*time.Hour)
	if err == nil && day > int64(perDay) {
		return errLibDaily(perDay, v.UserID == 0)
	}
	return nil
}

// LibPublicCollection is a collection with its size, for the filter.
type LibPublicCollection struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type LibPublicConfig struct {
	Enabled     bool                  `json:"enabled"`
	FindEnabled bool                  `json:"find_enabled"`
	Collections []LibPublicCollection `json:"collections"`
	Total       int                   `json:"total"`
	// For signed-in users.
	FreeLeft  int      `json:"free_left"`
	FreePer   int      `json:"free_per_day"`
	FindPrice float64  `json:"find_price"`
	Balance   *float64 `json:"balance,omitempty"`
}

// PublicConfig is what the /poems page needs before the first search.
func (s *InspirationService) PublicConfig(ctx context.Context, v LibVisitor) (*LibPublicConfig, error) {
	st := s.PublicSettings(ctx)
	cfg := &LibPublicConfig{Enabled: st.Enabled, FindEnabled: st.FindEnabled && s.chat != nil, FreePer: st.FreeFindsPerDay,
		FindPrice: st.FindPrice, Collections: []LibPublicCollection{}}
	if !st.Enabled {
		return cfg, nil
	}
	counts, err := s.PoemCounts(ctx)
	if err != nil {
		return nil, err
	}
	for id, n := range counts {
		cfg.Collections = append(cfg.Collections, LibPublicCollection{ID: id, Name: LibCollections[id], Count: n})
		cfg.Total += n
	}
	sortLibCollections(cfg.Collections)
	if v.UserID > 0 {
		cfg.FreeLeft = s.freeLeft(ctx, v.UserID, st)
		if s.billing != nil {
			if b, err := s.billing.Balance(ctx, v.UserID); err == nil {
				cfg.Balance = &b
			}
		}
	}
	return cfg, nil
}

func sortLibCollections(c []LibPublicCollection) {
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && c[j].Count > c[j-1].Count; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}

// publicPoem strips the admin's marks.
func publicPoem(p LibPoem) LibPoem {
	p.Starred, p.InShihua = false, ""
	return p
}

func (s *InspirationService) PublicSearch(ctx context.Context, v LibVisitor, q LibPoemQuery) ([]LibPoem, bool, error) {
	st := s.PublicSettings(ctx)
	if !st.Enabled {
		return nil, false, ErrLibPublicOff
	}
	if q.Offset > libPublicMaxOffset {
		return nil, false, ErrLibDeepPage
	}
	if err := s.view(ctx, v, st); err != nil {
		return nil, false, err
	}
	q.Starred, q.Limit = false, libPublicPageSize
	poems, more, err := s.SearchPoems(ctx, q)
	if err != nil {
		return nil, false, err
	}
	for i := range poems {
		poems[i] = publicPoem(poems[i])
	}
	return poems, more && q.Offset+q.Limit <= libPublicMaxOffset, nil
}

func (s *InspirationService) PublicPoem(ctx context.Context, v LibVisitor, id int64) (*LibPoem, error) {
	st := s.PublicSettings(ctx)
	if !st.Enabled {
		return nil, ErrLibPublicOff
	}
	if err := s.view(ctx, v, st); err != nil {
		return nil, err
	}
	p, err := s.repo.GetPoem(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrLibPoemNotFound
	}
	out := publicPoem(*p)
	return &out, nil
}

// PublicCandidates is the corpus half of 按画面找 for apps that run the model calls on the user's own
// key (the canvas 诗画): the poems containing the most of the given words. Counts as one view.
func (s *InspirationService) PublicCandidates(ctx context.Context, v LibVisitor, words []string, q LibPoemQuery) ([]LibPoem, error) {
	st := s.PublicSettings(ctx)
	if !st.Enabled {
		return nil, ErrLibPublicOff
	}
	words = libCleanWords(words)
	if len(words) == 0 {
		return nil, ErrLibInvalid
	}
	if q.Collection != "" && LibCollections[q.Collection] == "" {
		return nil, ErrLibInvalid
	}
	if err := s.view(ctx, v, st); err != nil {
		return nil, err
	}
	poems, err := s.repo.FindPoems(ctx, words, LibPoemQuery{Dynasty: q.Dynasty, Collection: q.Collection}, 40)
	if err != nil {
		return nil, err
	}
	for i := range poems {
		poems[i] = publicPoem(poems[i])
	}
	return poems, nil
}

func (s *InspirationService) freeKey(userID int64) string {
	return "poems:free:" + strconv.FormatInt(userID, 10) + ":" + s.dayKey()
}

func (s *InspirationService) freeLeft(ctx context.Context, userID int64, st LibPublicSettings) int {
	if s.counters == nil {
		return st.FreeFindsPerDay
	}
	used, err := s.counters.Get(ctx, s.freeKey(userID))
	if err != nil {
		return 0
	}
	return max(st.FreeFindsPerDay-int(used), 0)
}

// LibPublicFindResult is a user's 按画面找: the picks and candidates (never songs) and what it cost.
type LibPublicFindResult struct {
	Keywords   []string  `json:"keywords"`
	Picks      []LibPick `json:"picks"`
	Candidates []LibPoem `json:"candidates"`
	Free       bool      `json:"free"`
	Charged    float64   `json:"charged"`
	FreeLeft   int       `json:"free_left"`
	Balance    *float64  `json:"balance,omitempty"`
}

// PublicFind runs 按画面找 for a signed-in user: free while the day's free runs last, then paid from
// the balance after the run succeeds.
func (s *InspirationService) PublicFind(ctx context.Context, userID int64, in LibFindRequest) (*LibPublicFindResult, error) {
	st := s.PublicSettings(ctx)
	if !st.Enabled || !st.FindEnabled {
		return nil, ErrLibFindOff
	}
	if userID <= 0 {
		return nil, ErrLibLoginToFind
	}
	if s.chat == nil {
		return nil, ErrLibFindOff
	}
	in.Scene = strings.TrimSpace(in.Scene)
	if in.Scene == "" || utf8.RuneCountInString(in.Scene) > 200 {
		return nil, infraerrors.BadRequest("POEMS_SCENE", "请用 200 字以内描述想要的画面（Describe the scene in up to 200 characters）")
	}
	// The scene goes to the text model: no lyric lines through this door either.
	if err := s.guard.Refuse(ctx, in.Scene); err != nil {
		return nil, err
	}
	release := func() {}
	if s.counters != nil {
		lock := "poems:busy:" + strconv.FormatInt(userID, 10)
		n, err := s.counters.Incr(ctx, lock, libFindLockTTL)
		if err == nil && n > 1 {
			_ = s.counters.Decr(ctx, lock)
			return nil, ErrLibFindBusy
		}
		release = func() { _ = s.counters.Decr(context.WithoutCancel(ctx), lock) }
	}
	defer release()

	// Free run, or paid: the balance must cover the price before anything is spent.
	free := false
	freeKey := s.freeKey(userID)
	if s.counters != nil && st.FreeFindsPerDay > 0 {
		if used, err := s.counters.Incr(ctx, freeKey, 26*time.Hour); err == nil {
			if used <= int64(st.FreeFindsPerDay) {
				free = true
			} else {
				_ = s.counters.Decr(ctx, freeKey)
			}
		}
	}
	if !free && st.FindPrice > 0 {
		if s.billing == nil {
			return nil, ErrLibFindOff
		}
		balance, err := s.billing.Balance(ctx, userID)
		if err != nil {
			return nil, err
		}
		if balance < st.FindPrice {
			return nil, errLibBalance(st.FindPrice)
		}
	}
	undoFree := func() {
		if free {
			_ = s.counters.Decr(context.WithoutCancel(ctx), freeKey)
		}
	}
	capKey := "poems:cap:" + s.dayKey()
	if s.counters != nil && st.DailyFindCap > 0 {
		if n, err := s.counters.Incr(ctx, capKey, 26*time.Hour); err == nil && n > int64(st.DailyFindCap) {
			_ = s.counters.Decr(ctx, capKey)
			undoFree()
			return nil, ErrLibFindCapped
		}
	}
	rerank := true
	res, err := s.find(ctx, LibFindRequest{Scene: in.Scene, Dynasty: in.Dynasty, Collection: in.Collection, Count: st.FindCount, Rerank: &rerank}, st.FindModel, false)
	if err != nil {
		undoFree()
		if s.counters != nil && st.DailyFindCap > 0 {
			_ = s.counters.Decr(context.WithoutCancel(ctx), capKey)
		}
		return nil, err
	}
	out := &LibPublicFindResult{Keywords: res.Keywords, Picks: []LibPick{}, Candidates: []LibPoem{}, Free: free || st.FindPrice == 0}
	for _, p := range res.Picks {
		out.Picks = append(out.Picks, LibPick{Poem: publicPoem(p.Poem), Reason: p.Reason})
	}
	for i, p := range res.Candidates {
		if i == 30 {
			break
		}
		out.Candidates = append(out.Candidates, publicPoem(p))
	}
	if !out.Free {
		if err := s.billing.ChargeFind(context.WithoutCancel(ctx), userID, st.FindPrice, in.Scene, res.Model); err != nil {
			// The run is done and shown; a failed charge is logged, not turned into an error.
			slog.Error("poems: charge find", "user", userID, "cost", st.FindPrice, "err", err)
		} else {
			out.Charged = st.FindPrice
			if s.balanceCache != nil {
				if err := s.balanceCache.InvalidateUserBalance(context.WithoutCancel(ctx), userID); err != nil {
					slog.Warn("poems: invalidate balance cache", "user", userID, "err", err)
				}
			}
		}
	}
	out.FreeLeft = s.freeLeft(ctx, userID, st)
	if s.billing != nil {
		if b, err := s.billing.Balance(ctx, userID); err == nil {
			out.Balance = &b
		}
	}
	return out, nil
}

// LibPublicStats is the admin's view of the user-facing 按画面找.
type LibPublicStats struct {
	RunsToday    int     `json:"runs_today"` // all runs, free and paid
	PaidToday    int     `json:"paid_today"`
	RevenueToday float64 `json:"revenue_today"`
	Paid7d       int     `json:"paid_7d"`
	Revenue7d    float64 `json:"revenue_7d"`
}

func (s *InspirationService) PublicStats(ctx context.Context) (*LibPublicStats, error) {
	st := &LibPublicStats{}
	if s.counters != nil {
		if n, err := s.counters.Get(ctx, "poems:cap:"+s.dayKey()); err == nil {
			st.RunsToday = int(n)
		}
	}
	if s.billing == nil {
		return st, nil
	}
	day := shihuaDayStart(s.clock())
	var err error
	if st.PaidToday, st.RevenueToday, err = s.billing.FindStats(ctx, day); err != nil {
		return nil, err
	}
	if st.Paid7d, st.Revenue7d, err = s.billing.FindStats(ctx, day.AddDate(0, 0, -6)); err != nil {
		return nil, err
	}
	return st, nil
}
