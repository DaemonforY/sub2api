package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// GEO 监测 (generative engine optimisation monitoring): do AI assistants mention or cite the site
// when someone asks them a typical question? Admins keep a list of questions and "engines" — any
// OpenAI-compatible chat endpoint, ideally one that searches the web (Perplexity sonar, 通义 with
// enable_search, 智谱 GLM with the web_search tool). A run asks every enabled question to every
// enabled engine (geo_monitor_run.go) and stores each answer with what was found in it
// (geo_monitor_detect.go). Apps without an API (豆包, 元宝) are recorded by hand: the admin pastes
// the answer and the links it cited.

const (
	settingGeoSchedule    = "geo_schedule"
	settingGeoBrand       = "geo_brand_keywords"
	settingGeoCompetitors = "geo_competitor_keywords"
	settingGeoLastRunAt   = "geo_last_run_at"

	GeoScheduleOff    = "off"
	GeoScheduleDaily  = "daily"
	GeoScheduleWeekly = "weekly"

	GeoSourceAuto   = "auto"
	GeoSourceManual = "manual"

	geoMaxQuestionLen = 500
	geoMaxCategoryLen = 32
	geoMaxEngineName  = 64
	geoMaxModelLen    = 128
	geoMaxAnswerLen   = 100000
	geoMaxExtraBody   = 8 << 10
	geoMaxKeywords    = 20
	geoMaxKeywordLen  = 64
)

var geoDefaultBrandKeywords = []string{"hivegpt", "hivegpt.cn"}

var (
	ErrGeoQuestionNotFound = infraerrors.NotFound("GEO_QUESTION_NOT_FOUND", "问题不存在，可能已被删除（Question not found）")
	ErrGeoEngineNotFound   = infraerrors.NotFound("GEO_ENGINE_NOT_FOUND", "引擎不存在，可能已被删除（Engine not found）")
	ErrGeoCheckNotFound    = infraerrors.NotFound("GEO_CHECK_NOT_FOUND", "记录不存在，可能已被删除（Check not found）")
	ErrGeoEngineNameTaken  = infraerrors.Conflict("GEO_ENGINE_NAME_TAKEN", "已经有同名的引擎，换个名称（An engine with this name already exists）")
	ErrGeoRunning          = infraerrors.Conflict("GEO_RUNNING", "已经有一次 GEO 监测在运行，等它跑完再试（A run is already in progress）")
	ErrGeoNothingToRun     = infraerrors.BadRequest("GEO_NOTHING_TO_RUN", "没有可运行的组合：请先启用至少一个问题和一个引擎（No enabled questions or engines）")
	ErrGeoBadRequest       = infraerrors.BadRequest("GEO_BAD_REQUEST", "请求格式不正确，请刷新页面后重试（Invalid request）")
	errGeoBadQuestion      = infraerrors.BadRequest("GEO_BAD_QUESTION", "请填写问题，最多 500 字；分类最多 32 字（Question required, up to 500 characters; category up to 32）")
	errGeoBadEngine        = infraerrors.BadRequest("GEO_BAD_ENGINE", "引擎设置不正确：名称（最多 64 字）和模型名必填，Base URL 要以 http:// 或 https:// 开头（Invalid engine settings）")
	errGeoEngineKey        = infraerrors.BadRequest("GEO_ENGINE_KEY", "请填写这个引擎的 API Key（API key required）")
	errGeoBadExtraBody     = infraerrors.BadRequest("GEO_BAD_EXTRA_BODY", `附加参数要是 JSON 对象，例如 {"enable_search": true}，最多 8KB（extra_body must be a JSON object）`)
	errGeoBadManual        = infraerrors.BadRequest("GEO_BAD_MANUAL", "请选择问题、填写引擎名称（最多 64 字）并粘贴回答内容（Question, engine name and answer are required）")
	errGeoBadSettings      = infraerrors.BadRequest("GEO_BAD_SETTINGS", "设置不正确：自动运行只能选 关闭 / 每天 / 每周，至少填一个品牌关键词，每类最多 20 个、每个最多 64 字（Invalid settings）")
	errGeoNoEncryptor      = infraerrors.ServiceUnavailable("GEO_NO_ENCRYPTOR", "服务器没有配置加密密钥，不能保存 API Key（Encryption is not configured）")
)

type GeoQuestion struct {
	ID        int64     `json:"id"`
	Question  string    `json:"question"`
	Category  string    `json:"category"`
	Enabled   bool      `json:"enabled"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GeoQuestionInput struct {
	Question string `json:"question"`
	Category string `json:"category"`
	Enabled  *bool  `json:"enabled"`
	Sort     int    `json:"sort"`
}

// GeoEngine is a stored engine; the key stays encrypted and never leaves the service.
type GeoEngine struct {
	ID              int64
	Name            string
	BaseURL         string
	APIKeyEncrypted string
	Model           string
	ExtraBody       json.RawMessage
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// GeoEngineView is what the admin page sees: whether a key is stored and its last 4 characters.
type GeoEngineView struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	BaseURL   string          `json:"base_url"`
	Model     string          `json:"model"`
	ExtraBody json.RawMessage `json:"extra_body"`
	Enabled   bool            `json:"enabled"`
	HasKey    bool            `json:"has_key"`
	KeyMasked string          `json:"key_masked"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// GeoEngineInput: an empty APIKey on update keeps the stored one.
type GeoEngineInput struct {
	Name      string          `json:"name"`
	BaseURL   string          `json:"base_url"`
	APIKey    string          `json:"api_key"`
	Model     string          `json:"model"`
	ExtraBody json.RawMessage `json:"extra_body"`
	Enabled   *bool           `json:"enabled"`
}

type GeoCheck struct {
	ID          int64     `json:"id"`
	QuestionID  *int64    `json:"question_id"`
	Question    string    `json:"question"`
	EngineID    *int64    `json:"engine_id"`
	EngineName  string    `json:"engine_name"`
	Source      string    `json:"source"`
	Answer      string    `json:"answer"`
	Mentioned   bool      `json:"mentioned"`
	CitedURLs   []string  `json:"cited_urls"`
	OurURLs     []string  `json:"our_urls"`
	Competitors []string  `json:"competitors"`
	Error       string    `json:"error"`
	RunID       string    `json:"run_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type GeoManualInput struct {
	QuestionID int64    `json:"question_id"`
	EngineName string   `json:"engine_name"`
	Answer     string   `json:"answer"`
	CitedURLs  []string `json:"cited_urls"`
}

type GeoCheckFilter struct {
	QuestionID int64
	Engine     string
	Source     string
	Mentioned  *bool
	Limit      int
	Offset     int
}

type GeoCheckPage struct {
	Items []GeoCheck `json:"items"`
	Total int64      `json:"total"`
}

type GeoSettings struct {
	Schedule           string   `json:"schedule"`
	BrandKeywords      []string `json:"brand_keywords"`
	CompetitorKeywords []string `json:"competitor_keywords"`
}

// GeoWeekRate: per engine and week, among the latest answer to each question that week.
type GeoWeekRate struct {
	EngineName string    `json:"engine_name"`
	WeekStart  time.Time `json:"-"`
	Week       string    `json:"week"`       // ISO week, "2026-W41"
	WeekOf     string    `json:"week_start"` // Monday, "2026-10-05"
	Total      int       `json:"total"`
	Mentioned  int       `json:"mentioned"`
	Rate       float64   `json:"rate"`
}

type GeoMonitorRepository interface {
	ListQuestions(ctx context.Context) ([]GeoQuestion, error)
	GetQuestion(ctx context.Context, id int64) (*GeoQuestion, error) // ErrGeoQuestionNotFound
	CreateQuestion(ctx context.Context, q *GeoQuestion) error
	UpdateQuestion(ctx context.Context, q *GeoQuestion) error
	DeleteQuestion(ctx context.Context, id int64) error

	ListEngines(ctx context.Context) ([]GeoEngine, error)
	GetEngine(ctx context.Context, id int64) (*GeoEngine, error) // ErrGeoEngineNotFound
	CreateEngine(ctx context.Context, e *GeoEngine) error        // ErrGeoEngineNameTaken
	UpdateEngine(ctx context.Context, e *GeoEngine) error
	DeleteEngine(ctx context.Context, id int64) error

	InsertCheck(ctx context.Context, c *GeoCheck) error
	ListChecks(ctx context.Context, f GeoCheckFilter) ([]GeoCheck, int64, error)
	DeleteCheck(ctx context.Context, id int64) error
	// LatestChecks: the newest answer without an error for each (question, engine name).
	LatestChecks(ctx context.Context) ([]GeoCheck, error)
	// WeeklyRates since the given time (WeekStart set; Week/WeekOf/Rate filled by the service).
	WeeklyRates(ctx context.Context, since time.Time) ([]GeoWeekRate, error)
}

type GeoMonitorService struct {
	repo        GeoMonitorRepository
	settings    learnSettingStore
	encryptor   SecretEncryptor
	state       GeoRunState
	client      *http.Client
	now         func() time.Time
	concurrency int
	runs        sync.WaitGroup
	stop        chan struct{}
	stopOnce    sync.Once
}

func NewGeoMonitorService(repo GeoMonitorRepository, settings learnSettingStore, encryptor SecretEncryptor, state GeoRunState) *GeoMonitorService {
	if state == nil {
		state = newGeoMemoryRunState()
	}
	return &GeoMonitorService{repo: repo, settings: settings, encryptor: encryptor, state: state,
		client: &http.Client{Timeout: geoEngineTimeout}, now: time.Now, concurrency: geoRunConcurrency, stop: make(chan struct{})}
}

// ---- questions ----

func (s *GeoMonitorService) ListQuestions(ctx context.Context) ([]GeoQuestion, error) {
	out, err := s.repo.ListQuestions(ctx)
	if out == nil {
		out = []GeoQuestion{}
	}
	return out, err
}

func validateGeoQuestion(in GeoQuestionInput) (*GeoQuestion, error) {
	q := &GeoQuestion{Question: strings.TrimSpace(in.Question), Category: strings.TrimSpace(in.Category), Enabled: true, Sort: in.Sort}
	if in.Enabled != nil {
		q.Enabled = *in.Enabled
	}
	if q.Question == "" || utf8.RuneCountInString(q.Question) > geoMaxQuestionLen || utf8.RuneCountInString(q.Category) > geoMaxCategoryLen {
		return nil, errGeoBadQuestion
	}
	return q, nil
}

func (s *GeoMonitorService) CreateQuestion(ctx context.Context, in GeoQuestionInput) (*GeoQuestion, error) {
	q, err := validateGeoQuestion(in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateQuestion(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *GeoMonitorService) UpdateQuestion(ctx context.Context, id int64, in GeoQuestionInput) (*GeoQuestion, error) {
	q, err := validateGeoQuestion(in)
	if err != nil {
		return nil, err
	}
	q.ID = id
	if err := s.repo.UpdateQuestion(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *GeoMonitorService) DeleteQuestion(ctx context.Context, id int64) error {
	return s.repo.DeleteQuestion(ctx, id)
}

// ---- engines ----

func (s *GeoMonitorService) engineView(e GeoEngine) GeoEngineView {
	v := GeoEngineView{ID: e.ID, Name: e.Name, BaseURL: e.BaseURL, Model: e.Model, ExtraBody: e.ExtraBody, Enabled: e.Enabled,
		HasKey: e.APIKeyEncrypted != "", CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt}
	if v.HasKey {
		v.KeyMasked = "••••"
		if s.encryptor != nil {
			if key, err := s.encryptor.Decrypt(e.APIKeyEncrypted); err == nil && len(key) >= 12 {
				v.KeyMasked = "••••" + key[len(key)-4:]
			}
		}
	}
	return v
}

func (s *GeoMonitorService) ListEngines(ctx context.Context) ([]GeoEngineView, error) {
	list, err := s.repo.ListEngines(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]GeoEngineView, 0, len(list))
	for _, e := range list {
		out = append(out, s.engineView(e))
	}
	return out, nil
}

// normalizeGeoBaseURL: http(s) URL without a trailing slash or a pasted /chat/completions.
func normalizeGeoBaseURL(raw string) (string, bool) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	raw = strings.TrimRight(strings.TrimSuffix(raw, "/chat/completions"), "/")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || len(raw) > 500 {
		return "", false
	}
	return raw, true
}

func normalizeGeoExtraBody(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return nil, nil
	}
	if len(trimmed) > geoMaxExtraBody {
		return nil, errGeoBadExtraBody
	}
	// Accept a JSON string holding the object too (a textarea's content).
	var s string
	if json.Unmarshal([]byte(trimmed), &s) == nil {
		trimmed = strings.TrimSpace(s)
		if trimmed == "" {
			return nil, nil
		}
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil || obj == nil {
		return nil, errGeoBadExtraBody
	}
	if len(obj) == 0 {
		return nil, nil
	}
	b, _ := json.Marshal(obj)
	return b, nil
}

func (s *GeoMonitorService) validateEngine(in GeoEngineInput, e *GeoEngine) error {
	name := strings.TrimSpace(in.Name)
	model := strings.TrimSpace(in.Model)
	base, ok := normalizeGeoBaseURL(in.BaseURL)
	if name == "" || utf8.RuneCountInString(name) > geoMaxEngineName || model == "" || len(model) > geoMaxModelLen || !ok {
		return errGeoBadEngine
	}
	extra, err := normalizeGeoExtraBody(in.ExtraBody)
	if err != nil {
		return err
	}
	e.Name, e.Model, e.BaseURL, e.ExtraBody = name, model, base, extra
	if in.Enabled != nil {
		e.Enabled = *in.Enabled
	}
	if key := strings.TrimSpace(in.APIKey); key != "" {
		if s.encryptor == nil {
			return errGeoNoEncryptor
		}
		enc, err := s.encryptor.Encrypt(key)
		if err != nil {
			return err
		}
		e.APIKeyEncrypted = enc
	}
	if e.APIKeyEncrypted == "" {
		return errGeoEngineKey
	}
	return nil
}

func (s *GeoMonitorService) CreateEngine(ctx context.Context, in GeoEngineInput) (*GeoEngineView, error) {
	e := &GeoEngine{Enabled: true}
	if err := s.validateEngine(in, e); err != nil {
		return nil, err
	}
	if err := s.repo.CreateEngine(ctx, e); err != nil {
		return nil, err
	}
	v := s.engineView(*e)
	return &v, nil
}

func (s *GeoMonitorService) UpdateEngine(ctx context.Context, id int64, in GeoEngineInput) (*GeoEngineView, error) {
	e, err := s.repo.GetEngine(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.validateEngine(in, e); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateEngine(ctx, e); err != nil {
		return nil, err
	}
	v := s.engineView(*e)
	return &v, nil
}

func (s *GeoMonitorService) DeleteEngine(ctx context.Context, id int64) error {
	return s.repo.DeleteEngine(ctx, id)
}

// ---- checks ----

func (s *GeoMonitorService) ListChecks(ctx context.Context, f GeoCheckFilter) (*GeoCheckPage, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Source != GeoSourceAuto && f.Source != GeoSourceManual {
		f.Source = ""
	}
	f.Engine = strings.TrimSpace(f.Engine)
	items, total, err := s.repo.ListChecks(ctx, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []GeoCheck{}
	}
	return &GeoCheckPage{Items: items, Total: total}, nil
}

// AddManual records an answer pasted from an app without an API (豆包, 元宝…).
func (s *GeoMonitorService) AddManual(ctx context.Context, in GeoManualInput) (*GeoCheck, error) {
	name := strings.TrimSpace(in.EngineName)
	answer := strings.TrimSpace(in.Answer)
	if in.QuestionID <= 0 || name == "" || utf8.RuneCountInString(name) > geoMaxEngineName || answer == "" {
		return nil, errGeoBadManual
	}
	q, err := s.repo.GetQuestion(ctx, in.QuestionID)
	if err != nil {
		return nil, err
	}
	st := s.Settings(ctx)
	answer = truncateRunes(answer, geoMaxAnswerLen)
	d := DetectGeo(answer, in.CitedURLs, st.BrandKeywords, st.CompetitorKeywords)
	qid := q.ID
	c := &GeoCheck{QuestionID: &qid, Question: q.Question, EngineName: name, Source: GeoSourceManual, Answer: answer,
		Mentioned: d.Mentioned, CitedURLs: d.CitedURLs, OurURLs: d.OurURLs, Competitors: d.Competitors}
	if err := s.repo.InsertCheck(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *GeoMonitorService) DeleteCheck(ctx context.Context, id int64) error {
	return s.repo.DeleteCheck(ctx, id)
}

// ---- settings ----

// Settings: brand keywords default to the site's name and domain until an admin saves a list.
func (s *GeoMonitorService) Settings(ctx context.Context) GeoSettings {
	out := GeoSettings{Schedule: GeoScheduleOff, BrandKeywords: append([]string(nil), geoDefaultBrandKeywords...), CompetitorKeywords: []string{}}
	if s.settings == nil {
		return out
	}
	v, err := s.settings.GetMultiple(ctx, []string{settingGeoSchedule, settingGeoBrand, settingGeoCompetitors})
	if err != nil {
		return out
	}
	switch v[settingGeoSchedule] {
	case GeoScheduleDaily, GeoScheduleWeekly:
		out.Schedule = v[settingGeoSchedule]
	}
	var list []string
	if raw := v[settingGeoBrand]; raw != "" && json.Unmarshal([]byte(raw), &list) == nil && len(list) > 0 {
		out.BrandKeywords = list
	}
	list = nil
	if raw := v[settingGeoCompetitors]; raw != "" && json.Unmarshal([]byte(raw), &list) == nil && list != nil {
		out.CompetitorKeywords = list
	}
	return out
}

func cleanGeoKeywords(in []string) ([]string, bool) {
	out := []string{}
	seen := map[string]bool{}
	for _, kw := range in {
		kw = strings.TrimSpace(kw)
		if kw == "" || seen[strings.ToLower(kw)] {
			continue
		}
		if utf8.RuneCountInString(kw) > geoMaxKeywordLen {
			return nil, false
		}
		seen[strings.ToLower(kw)] = true
		out = append(out, kw)
	}
	return out, len(out) <= geoMaxKeywords
}

func (s *GeoMonitorService) SaveSettings(ctx context.Context, in GeoSettings) (GeoSettings, error) {
	brand, ok1 := cleanGeoKeywords(in.BrandKeywords)
	comp, ok2 := cleanGeoKeywords(in.CompetitorKeywords)
	switch in.Schedule {
	case GeoScheduleOff, GeoScheduleDaily, GeoScheduleWeekly:
	default:
		return GeoSettings{}, errGeoBadSettings
	}
	if !ok1 || !ok2 || len(brand) == 0 {
		return GeoSettings{}, errGeoBadSettings
	}
	if s.settings != nil {
		b, _ := json.Marshal(brand)
		c, _ := json.Marshal(comp)
		if err := s.settings.SetMultiple(ctx, map[string]string{settingGeoSchedule: in.Schedule, settingGeoBrand: string(b), settingGeoCompetitors: string(c)}); err != nil {
			return GeoSettings{}, err
		}
	}
	return s.Settings(ctx), nil
}

func (s *GeoMonitorService) lastRunAt(ctx context.Context) time.Time {
	if s.settings == nil {
		return time.Time{}
	}
	v, err := s.settings.GetMultiple(ctx, []string{settingGeoLastRunAt})
	if err != nil {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, v[settingGeoLastRunAt])
	return t
}

func (s *GeoMonitorService) setLastRunAt(ctx context.Context, t time.Time) {
	if s.settings != nil {
		_ = s.settings.SetMultiple(ctx, map[string]string{settingGeoLastRunAt: t.UTC().Format(time.RFC3339)})
	}
}

// ParseGeoCheckFilter reads the list query (?question_id=&engine=&source=&mentioned=&limit=&offset=).
func ParseGeoCheckFilter(get func(string) string) GeoCheckFilter {
	f := GeoCheckFilter{Engine: get("engine"), Source: get("source")}
	f.QuestionID, _ = strconv.ParseInt(get("question_id"), 10, 64)
	f.Limit, _ = strconv.Atoi(get("limit"))
	f.Offset, _ = strconv.Atoi(get("offset"))
	switch get("mentioned") {
	case "true", "1":
		v := true
		f.Mentioned = &v
	case "false", "0":
		v := false
		f.Mentioned = &v
	}
	return f
}
