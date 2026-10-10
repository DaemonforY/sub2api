package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// A GEO 监测 run asks every enabled question to every enabled engine, two requests at a time, and
// stores each answer. Only one run happens at a time (a Redis lock, so several server instances
// don't double the bill); admins start one from the page, and with 自动运行 on, an hourly check
// starts one when the last run is a day / a week old.

const (
	geoEngineTimeout    = 120 * time.Second
	geoRunConcurrency   = 2
	geoRunLockTTL       = 2 * time.Hour
	geoRunTimeout       = geoRunLockTTL - 10*time.Minute
	geoSchedulerTick    = time.Hour
	geoSchedulerSlack   = 30 * time.Minute // an hourly tick shouldn't push the next run an hour later each time
	geoMaxResponseBytes = 4 << 20
	geoTrendWeeks       = 12
	geoTestQuestion     = "用一句话介绍你自己。"

	// GeoRunLockTTL: how long the one-run-at-a-time lock lives (a crashed run unlocks itself).
	GeoRunLockTTL = geoRunLockTTL
)

// GeoRunStatus is what the page polls while a run is going.
type GeoRunStatus struct {
	Running   bool       `json:"running"`
	RunID     string     `json:"run_id"`
	StartedAt *time.Time `json:"started_at"`
	LastRunAt *time.Time `json:"last_run_at"`
	Done      int        `json:"done"`
	Total     int        `json:"total"`
}

// GeoRunState holds the one-run-at-a-time lock and the progress of the current run (Redis in
// production, see repository.NewGeoRunState; in-process when nil).
type GeoRunState interface {
	TryStart(ctx context.Context, runID string, startedAt time.Time, total int) (bool, error)
	Progress(ctx context.Context) (GeoRunStatus, error)
	IncDone(ctx context.Context, runID string)
	Finish(ctx context.Context, runID string)
}

// ---- in-process state (no Redis: tests, single instance) ----

type geoMemoryRunState struct {
	mu sync.Mutex
	st GeoRunStatus
}

func newGeoMemoryRunState() *geoMemoryRunState { return &geoMemoryRunState{} }

func (m *geoMemoryRunState) TryStart(_ context.Context, runID string, startedAt time.Time, total int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.st.Running {
		return false, nil
	}
	t := startedAt
	m.st = GeoRunStatus{Running: true, RunID: runID, StartedAt: &t, Total: total}
	return true, nil
}

func (m *geoMemoryRunState) Progress(context.Context) (GeoRunStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.st, nil
}

func (m *geoMemoryRunState) IncDone(_ context.Context, runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.st.RunID == runID {
		m.st.Done++
	}
}

func (m *geoMemoryRunState) Finish(_ context.Context, runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.st.RunID == runID {
		m.st = GeoRunStatus{}
	}
}

// ---- wiring ----

// ProvideGeoMonitorService wires GEO 监测 and starts the hourly schedule check.
func ProvideGeoMonitorService(repo GeoMonitorRepository, settings SettingRepository, encryptor SecretEncryptor, state GeoRunState) *GeoMonitorService {
	var store learnSettingStore
	if settings != nil {
		store = settings
	}
	s := NewGeoMonitorService(repo, store, encryptor, state)
	go s.scheduleLoop()
	return s
}

func (s *GeoMonitorService) Stop() { s.stopOnce.Do(func() { close(s.stop) }) }

func (s *GeoMonitorService) scheduleLoop() {
	first := time.NewTimer(5 * time.Minute)
	defer first.Stop()
	select {
	case <-s.stop:
		return
	case <-first.C:
	}
	t := time.NewTicker(geoSchedulerTick)
	defer t.Stop()
	for {
		s.RunScheduled(context.Background())
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
	}
}

// geoScheduleDue: is a scheduled run due, given the last run?
func geoScheduleDue(schedule string, last, now time.Time) bool {
	var period time.Duration
	switch schedule {
	case GeoScheduleDaily:
		period = 24 * time.Hour
	case GeoScheduleWeekly:
		period = 7 * 24 * time.Hour
	default:
		return false
	}
	return last.IsZero() || now.Sub(last) >= period-geoSchedulerSlack
}

// RunScheduled starts a run when 自动运行 is on and one is due. Returns the run id, "" when none started.
func (s *GeoMonitorService) RunScheduled(ctx context.Context) string {
	st := s.Settings(ctx)
	if !geoScheduleDue(st.Schedule, s.lastRunAt(ctx), s.now()) {
		return ""
	}
	id, err := s.StartRun(ctx)
	if err != nil {
		if !errors.Is(err, ErrGeoRunning) && !errors.Is(err, ErrGeoNothingToRun) {
			logger.LegacyPrintf("service.geo_monitor", "[GeoMonitor] scheduled run failed to start: %v", err)
		}
		return ""
	}
	return id
}

type geoPair struct {
	q GeoQuestion
	e GeoEngine
}

func newGeoRunID(now time.Time) string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return now.UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b)
}

// StartRun asks every enabled question to every enabled engine in the background; returns the run id
// right away, ErrGeoRunning when one is already going.
func (s *GeoMonitorService) StartRun(ctx context.Context) (string, error) {
	questions, err := s.repo.ListQuestions(ctx)
	if err != nil {
		return "", err
	}
	engines, err := s.repo.ListEngines(ctx)
	if err != nil {
		return "", err
	}
	var pairs []geoPair
	for _, q := range questions {
		if !q.Enabled {
			continue
		}
		for _, e := range engines {
			if e.Enabled {
				pairs = append(pairs, geoPair{q: q, e: e})
			}
		}
	}
	if len(pairs) == 0 {
		return "", ErrGeoNothingToRun
	}
	now := s.now()
	runID := newGeoRunID(now)
	ok, err := s.state.TryStart(ctx, runID, now, len(pairs))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrGeoRunning
	}
	s.runs.Add(1)
	go func() {
		defer s.runs.Done()
		s.runPairs(runID, pairs)
	}()
	return runID, nil
}

func (s *GeoMonitorService) runPairs(runID string, pairs []geoPair) {
	ctx, cancel := context.WithTimeout(context.Background(), geoRunTimeout)
	defer cancel()
	defer s.state.Finish(context.Background(), runID)
	st := s.Settings(ctx)
	keys := map[int64]string{}
	keyErr := map[int64]bool{}
	for _, p := range pairs {
		if _, seen := keys[p.e.ID]; seen || keyErr[p.e.ID] {
			continue
		}
		key, err := s.decryptKey(p.e)
		if err != nil {
			keyErr[p.e.ID] = true
			continue
		}
		keys[p.e.ID] = key
	}

	sem := make(chan struct{}, max(1, s.concurrency))
	var wg sync.WaitGroup
	failed := 0
	var mu sync.Mutex
	for _, p := range pairs {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(p geoPair) {
			defer func() { <-sem; wg.Done() }()
			c := s.askPair(ctx, p, keys[p.e.ID], keyErr[p.e.ID], st)
			c.RunID = runID
			insertCtx, cancelInsert := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelInsert()
			if err := s.repo.InsertCheck(insertCtx, c); err != nil {
				logger.LegacyPrintf("service.geo_monitor", "[GeoMonitor] run %s: save check failed: %v", runID, err)
			}
			if c.Error != "" {
				mu.Lock()
				failed++
				mu.Unlock()
			}
			s.state.IncDone(context.Background(), runID)
		}(p)
	}
	wg.Wait()
	s.setLastRunAt(context.Background(), s.now())
	logger.LegacyPrintf("service.geo_monitor", "[GeoMonitor] run %s finished: %d answers, %d failed", runID, len(pairs), failed)
}

func (s *GeoMonitorService) decryptKey(e GeoEngine) (string, error) {
	if s.encryptor == nil || e.APIKeyEncrypted == "" {
		return "", errGeoNoEncryptor
	}
	key, err := s.encryptor.Decrypt(e.APIKeyEncrypted)
	if err != nil || strings.TrimSpace(key) == "" {
		return "", errors.New("decrypt failed")
	}
	return key, nil
}

func (s *GeoMonitorService) askPair(ctx context.Context, p geoPair, key string, badKey bool, st GeoSettings) *GeoCheck {
	qid, eid := p.q.ID, p.e.ID
	c := &GeoCheck{QuestionID: &qid, Question: p.q.Question, EngineID: &eid, EngineName: p.e.Name, Source: GeoSourceAuto,
		CitedURLs: []string{}, OurURLs: []string{}, Competitors: []string{}}
	if badKey {
		c.Error = "API Key 无法解密，请在「引擎」里重新填写（cannot decrypt the API key）"
		return c
	}
	answer, cited, err := s.ask(ctx, p.e, key, p.q.Question)
	if err != nil {
		c.Error = err.Error()
		return c
	}
	d := DetectGeo(answer, cited, st.BrandKeywords, st.CompetitorKeywords)
	c.Answer = truncateRunes(answer, geoMaxAnswerLen)
	c.Mentioned, c.CitedURLs, c.OurURLs, c.Competitors = d.Mentioned, d.CitedURLs, d.OurURLs, d.Competitors
	return c
}

// geoCallError is an engine failure as stored and shown to the admin: Chinese, with the upstream's
// own words in parentheses, and never the API key.
type geoCallError struct{ msg string }

func (e *geoCallError) Error() string { return e.msg }

func geoRedact(s, key string) string {
	if len(key) >= 4 {
		s = strings.ReplaceAll(s, key, "***")
	}
	return s
}

// ask sends one question to an engine (POST {base_url}/chat/completions, not streamed).
func (s *GeoMonitorService) ask(ctx context.Context, e GeoEngine, key, question string) (string, []string, error) {
	body, err := buildGeoRequestBody(e.ExtraBody, e.Model, question)
	if err != nil {
		return "", nil, &geoCallError{msg: "附加参数不是合法的 JSON 对象，请在「引擎」里修改（invalid extra_body）"}
	}
	reqCtx, cancel := context.WithTimeout(ctx, geoEngineTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimRight(e.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, &geoCallError{msg: "Base URL 不正确（" + geoRedact(err.Error(), key) + "）"}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := s.client.Do(req)
	if err != nil {
		var ne net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
			return "", nil, &geoCallError{msg: fmt.Sprintf("等了 %d 秒没有回答，引擎太慢或网络不通（timeout）", int(geoEngineTimeout/time.Second))}
		}
		return "", nil, &geoCallError{msg: "连不上引擎，检查 Base URL 和服务器网络（" + truncateRunes(geoRedact(err.Error(), key), 300) + "）"}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, geoMaxResponseBytes))
	if err != nil {
		return "", nil, &geoCallError{msg: "读取回答失败（" + truncateRunes(geoRedact(err.Error(), key), 300) + "）"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.Join(strings.Fields(truncateRunes(string(raw), 300)), " ")
		hint := ""
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			hint = "，API Key 不对或没有权限"
		case http.StatusNotFound:
			hint = "，Base URL 或模型名可能不对"
		case http.StatusTooManyRequests:
			hint = "，被限流或余额不足"
		}
		return "", nil, &geoCallError{msg: fmt.Sprintf("引擎返回 HTTP %d%s（%s）", resp.StatusCode, hint, geoRedact(snippet, key))}
	}
	answer, cited, err := parseGeoChatResponse(raw)
	if err != nil {
		snippet := strings.Join(strings.Fields(truncateRunes(string(raw), 200)), " ")
		return "", nil, &geoCallError{msg: "引擎返回的不是 Chat Completions 格式的 JSON，Base URL 可能要加 /v1（" + geoRedact(snippet, key) + "）"}
	}
	if strings.TrimSpace(answer) == "" {
		return "", nil, &geoCallError{msg: "引擎返回了空回答（empty answer）"}
	}
	return answer, cited, nil
}

// GeoEngineTestResult is POST /admin/geo/engines/:id/test.
type GeoEngineTestResult struct {
	OK        bool   `json:"ok"`
	LatencyMS int64  `json:"latency_ms"`
	Answer    string `json:"answer,omitempty"`
	Error     string `json:"error,omitempty"`
}

// TestEngine asks one short fixed question, to check the address, key and model. Nothing is stored.
func (s *GeoMonitorService) TestEngine(ctx context.Context, id int64) (*GeoEngineTestResult, error) {
	e, err := s.repo.GetEngine(ctx, id)
	if err != nil {
		return nil, err
	}
	key, err := s.decryptKey(*e)
	if err != nil {
		return &GeoEngineTestResult{Error: "API Key 无法解密，请重新填写（cannot decrypt the API key）"}, nil
	}
	start := time.Now()
	answer, _, err := s.ask(ctx, *e, key, geoTestQuestion)
	out := &GeoEngineTestResult{LatencyMS: time.Since(start).Milliseconds()}
	if err != nil {
		out.Error = err.Error()
		return out, nil
	}
	out.OK, out.Answer = true, truncateRunes(answer, 500)
	return out, nil
}

// Status of the current run, and when the last one finished.
func (s *GeoMonitorService) Status(ctx context.Context) (GeoRunStatus, error) {
	st, err := s.state.Progress(ctx)
	if err != nil {
		return GeoRunStatus{}, err
	}
	if t := s.lastRunAt(ctx); !t.IsZero() {
		st.LastRunAt = &t
	}
	return st, nil
}

// ---- summary ----

type GeoEngineSummary struct {
	Name          string     `json:"name"`
	Auto          bool       `json:"auto"` // an API engine configured here (otherwise manual entries only)
	Enabled       bool       `json:"enabled"`
	Answered      int        `json:"answered"` // questions with a result
	Mentioned     int        `json:"mentioned"`
	MentionRate   float64    `json:"mention_rate"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
}

type GeoSummaryTotals struct {
	Questions   int     `json:"questions"` // enabled
	Engines     int     `json:"engines"`   // enabled API engines
	Answered    int     `json:"answered"`  // question × engine pairs with a result
	Mentioned   int     `json:"mentioned"`
	MentionRate float64 `json:"mention_rate"`
}

type GeoSummary struct {
	Engines   []GeoEngineSummary `json:"engines"`
	Questions []GeoQuestion      `json:"questions"`
	Latest    []GeoCheck         `json:"latest"`
	Weekly    []GeoWeekRate      `json:"weekly"`
	Totals    GeoSummaryTotals   `json:"totals"`
}

func geoRate(mentioned, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(mentioned) / float64(total)
}

// Summary: per engine, the latest result for each question and its mention rate; the weekly trend.
func (s *GeoMonitorService) Summary(ctx context.Context) (*GeoSummary, error) {
	questions, err := s.ListQuestions(ctx)
	if err != nil {
		return nil, err
	}
	engines, err := s.repo.ListEngines(ctx)
	if err != nil {
		return nil, err
	}
	latest, err := s.repo.LatestChecks(ctx)
	if err != nil {
		return nil, err
	}
	since := geoWeekStart(s.now()).AddDate(0, 0, -7*(geoTrendWeeks-1))
	weekly, err := s.repo.WeeklyRates(ctx, since)
	if err != nil {
		return nil, err
	}
	out := &GeoSummary{Questions: questions, Latest: latest, Weekly: weekly, Engines: []GeoEngineSummary{}}
	if out.Latest == nil {
		out.Latest = []GeoCheck{}
	}
	if out.Weekly == nil {
		out.Weekly = []GeoWeekRate{}
	}
	for i := range out.Weekly {
		w := &out.Weekly[i]
		y, wk := w.WeekStart.ISOWeek()
		w.Week = fmt.Sprintf("%d-W%02d", y, wk)
		w.WeekOf = w.WeekStart.Format("2006-01-02")
		w.Rate = geoRate(w.Mentioned, w.Total)
	}

	byName := map[string]*GeoEngineSummary{}
	var order []string
	add := func(name string) *GeoEngineSummary {
		if es, ok := byName[name]; ok {
			return es
		}
		byName[name] = &GeoEngineSummary{Name: name}
		order = append(order, name)
		return byName[name]
	}
	for _, e := range engines {
		es := add(e.Name)
		es.Auto, es.Enabled = true, e.Enabled
		if e.Enabled {
			out.Totals.Engines++
		}
	}
	var manualNames []string
	for _, c := range latest {
		if _, ok := byName[c.EngineName]; !ok {
			manualNames = append(manualNames, c.EngineName)
		}
		es := add(c.EngineName)
		es.Answered++
		if c.Mentioned {
			es.Mentioned++
		}
		if es.LastCheckedAt == nil || c.CreatedAt.After(*es.LastCheckedAt) {
			t := c.CreatedAt
			es.LastCheckedAt = &t
		}
		out.Totals.Answered++
		if c.Mentioned {
			out.Totals.Mentioned++
		}
	}
	// API engines first (as configured), then manual-only names alphabetically.
	sort.Strings(manualNames)
	rank := map[string]int{}
	for i, n := range order {
		rank[n] = i
	}
	for i, n := range manualNames {
		rank[n] = len(engines) + i
	}
	sort.SliceStable(order, func(i, j int) bool { return rank[order[i]] < rank[order[j]] })
	for _, n := range order {
		es := byName[n]
		es.MentionRate = geoRate(es.Mentioned, es.Answered)
		out.Engines = append(out.Engines, *es)
	}
	for _, q := range questions {
		if q.Enabled {
			out.Totals.Questions++
		}
	}
	out.Totals.MentionRate = geoRate(out.Totals.Mentioned, out.Totals.Answered)
	return out, nil
}

// geoWeekStart: Monday 00:00 (Beijing time) of the week containing t.
func geoWeekStart(t time.Time) time.Time {
	t = t.In(geoTZ)
	d := (int(t.Weekday()) + 6) % 7
	y, m, day := t.Date()
	return time.Date(y, m, day-d, 0, 0, 0, 0, geoTZ)
}

var geoTZ = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}()
