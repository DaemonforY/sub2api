package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AI 助教: teachers build an assistant for their class from a template, add their own materials and
// share a link with a class password. Students join with just a name and ask; every answer is one
// streamed chat completion through this site's gateway on the teacher's own key (billed as usual).
// Per-student and per-assistant daily limits are counted in Redis; questions and answers are kept
// for the teacher's 学情 page.

const (
	TutorGuide  = "guide"  // lead students to the answer
	TutorAnswer = "answer" // explain and give answers

	TutorStandard     = "standard" // the learning site's model
	TutorEconomy      = "economy"  // a cheaper model, about a tenth of the price
	tutorEconomyModel = "gpt-5.6-luna"

	tutorMaxPerTeacher    = 20
	tutorMaxMaterials     = 20
	tutorMaxMaterialChars = 300000 // all materials of one assistant
	tutorFullContextChars = 24000  // up to this, all materials go into the prompt; above, the best passages
	tutorPassageChars     = 2400
	tutorPassages         = 8
	tutorMaxQuestion      = 2000
	tutorMaxHistory       = 10
	tutorMaxHistoryChars  = 2000
	tutorMaxOutputTokens  = 1500
	tutorTimeout          = 3 * time.Minute
	tutorCounterTTL       = 26 * time.Hour
	tutorMessageRetention = 180 * 24 * time.Hour
	tutorRecentMessages   = 100
	tutorInsightQuestions = 300
)

var (
	ErrTutorNotFound = infraerrors.NotFound("TUTOR_NOT_FOUND", "没有找到这个助教（Assistant not found）")
	ErrTutorClosed   = infraerrors.Forbidden("TUTOR_CLOSED", "老师暂时关闭了这个助教，有问题请直接问老师（Assistant is closed）")
	ErrTutorPass     = infraerrors.Forbidden("TUTOR_PASS", "口令不对，请向老师确认班级口令（Wrong class password）")
	ErrTutorName     = infraerrors.BadRequest("TUTOR_NAME", "请填写你的名字，1–20 个字（Name 1–20 characters）")
	ErrTutorSession  = infraerrors.Unauthorized("TUTOR_SESSION", "请先输入口令和名字进入（Please join first）")
	ErrTutorQuestion = infraerrors.BadRequest("TUTOR_QUESTION", fmt.Sprintf("请输入问题，最多 %d 字（Question 1–%d characters）", tutorMaxQuestion, tutorMaxQuestion))
	ErrTutorKey      = infraerrors.ServiceUnavailable("TUTOR_KEY", "助教暂时用不了，请告诉老师检查助教使用的 Key（Assistant's key unavailable）")
	ErrTutorInput    = infraerrors.BadRequest("TUTOR_INPUT", "请检查：名称 1–30 字，选择模板和 Key，次数在允许范围内（Invalid settings）")
	ErrTutorTooMany  = infraerrors.BadRequest("TUTOR_TOO_MANY", fmt.Sprintf("每位老师最多建 %d 个助教（Too many assistants）", tutorMaxPerTeacher))
	ErrTutorMaterial = infraerrors.BadRequest("TUTOR_MATERIAL", fmt.Sprintf("每个助教最多 %d 份资料、共 %d 万字（Too many materials）", tutorMaxMaterials, tutorMaxMaterialChars/10000))
)

type Tutor struct {
	ID            int64           `json:"id"`
	UserID        int64           `json:"-"`
	KeyID         int64           `json:"key_id"`
	Name          string          `json:"name"`
	Template      string          `json:"template"`
	Subject       string          `json:"subject"`
	Grade         string          `json:"grade"`
	Style         string          `json:"style"`
	AnswerMode    string          `json:"answer_mode"`
	ModelTier     string          `json:"model_tier"`
	Rules         string          `json:"rules"`
	Greeting      string          `json:"greeting"`
	ShareCode     string          `json:"share_code"`
	PassCode      string          `json:"pass_code"`
	PerStudentDay int             `json:"per_student_day"`
	DailyCap      int             `json:"daily_cap"`
	Enabled       bool            `json:"enabled"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	MaterialChars int             `json:"material_chars"`
	Materials     []TutorMaterial `json:"materials,omitempty"`
	KeyName       string          `json:"key_name,omitempty"`
	KeyProblem    string          `json:"key_problem,omitempty"`
	Cost          *TutorCost      `json:"cost,omitempty"`
	// Costs quotes both tiers (standard, economy) so the teacher can compare.
	Costs map[string]*TutorCost `json:"costs,omitempty"`
}

// TutorCost estimates one question on the assistant's key with its current materials: Low when the
// prompt (rules + materials) is read from the provider's prompt cache, as in a run of questions;
// High when it is not. Tokens are estimates (about 1.5 Chinese characters a token).
type TutorCost struct {
	Low          float64 `json:"low"`
	High         float64 `json:"high"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Model        string  `json:"model"`
}

// TutorPricer prices chat tokens on a key (the gateway's own pricing).
type TutorPricer interface {
	EstimateTextCost(ctx context.Context, apiKey *APIKey, model string, tokens UsageTokens) (float64, bool)
}

type TutorMaterial struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Chars     int       `json:"chars"`
	Content   string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type TutorStudent struct {
	ID         int64     `json:"id"`
	TutorID    int64     `json:"-"`
	Name       string    `json:"name"`
	Questions  int       `json:"questions"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type TutorMessage struct {
	ID          int64     `json:"id"`
	StudentID   int64     `json:"student_id"`
	StudentName string    `json:"student_name"`
	Question    string    `json:"question"`
	Answer      string    `json:"answer"`
	CreatedAt   time.Time `json:"created_at"`
}

type TutorRepository interface {
	Create(ctx context.Context, t *Tutor) error
	Update(ctx context.Context, t *Tutor) error
	Get(ctx context.Context, userID, id int64) (*Tutor, error) // nil when not found
	GetByCode(ctx context.Context, code string) (*Tutor, error)
	List(ctx context.Context, userID int64) ([]Tutor, error)
	Count(ctx context.Context, userID int64) (int, error)
	Delete(ctx context.Context, userID, id int64) error

	AddMaterial(ctx context.Context, tutorID int64, name, content string) (*TutorMaterial, error)
	ListMaterials(ctx context.Context, tutorID int64, withContent bool) ([]TutorMaterial, error)
	DeleteMaterial(ctx context.Context, tutorID, id int64) error

	// UpsertStudent adds a student by name, or gives an existing one a new token.
	UpsertStudent(ctx context.Context, tutorID int64, name, tokenHash string) (*TutorStudent, error)
	StudentByToken(ctx context.Context, tutorID int64, tokenHash string) (*TutorStudent, error)
	ListStudents(ctx context.Context, tutorID int64) ([]TutorStudent, error)
	// AddMessage records a question and answer and counts it on the student.
	AddMessage(ctx context.Context, tutorID, studentID int64, question, answer string) error
	ListMessages(ctx context.Context, tutorID int64, since time.Time, limit int) ([]TutorMessage, error)
	CountMessages(ctx context.Context, tutorID int64, since time.Time) (int, error)
	DeleteMessagesBefore(ctx context.Context, before time.Time) (int64, error)
}

type tutorChatFunc func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error)

type TutorService struct {
	repo  TutorRepository
	learn *LearnService
	quota AssistantQuotaCache
	chat  tutorChatFunc
	now   func() time.Time
	price TutorPricer

	kbMu sync.Mutex
	kbs  map[int64]tutorKB
}

// tutorKB caches an assistant's materials: all of them when short, a search index when long.
type tutorKB struct {
	full string
	kb   *assistantKB
}

func NewTutorService(repo TutorRepository, learn *LearnService, quota AssistantQuotaCache) *TutorService {
	s := &TutorService{repo: repo, learn: learn, quota: quota, now: time.Now, kbs: map[int64]tutorKB{}}
	client := &http.Client{Timeout: tutorTimeout}
	s.chat = func(ctx context.Context, key, model string, messages []LearnMessage, maxTokens int, onDelta func(string) error) (*LearnTutorResult, error) {
		gateway := ""
		if s.learn != nil {
			gateway = s.learn.gatewayURL
		}
		return streamChatCompletion(ctx, client, gateway, "hivegpt-tutor/1", key, model, messages, maxTokens, onDelta)
	}
	return s
}

// SetPricer lets Get quote what a question costs.
func (s *TutorService) SetPricer(p TutorPricer) { s.price = p }

const (
	tutorEstPromptChars  = 1500 // the template's task and the rules
	tutorEstHistoryChars = 1200 // earlier turns and the question
	tutorEstOutputTokens = 400  // answers measured on hivegpt.cn ran 100–400 tokens
	tutorEstPassageChars = 6000 // what the search puts in when materials are long
	// The upstream adds fixed instructions to every chat request: a question with a 175-character
	// handout was billed ~5 000 input tokens on hivegpt.cn (2026-10-10), ~4 000 more than our prompt.
	// It is the same every time, so it is cached along with our prompt.
	tutorEstUpstreamTokens = 4000
)

// estimate prices a typical question: prompt + materials (all, or the passages) + history in, an answer out.
func (s *TutorService) estimate(ctx context.Context, key *APIKey, model string, materialChars int) *TutorCost {
	if s.price == nil || key == nil {
		return nil
	}
	if materialChars > tutorFullContextChars {
		materialChars = tutorEstPassageChars
	}
	prefix := tutorEstUpstreamTokens + (tutorEstPromptChars+materialChars)*2/3
	rest := tutorEstHistoryChars * 2 / 3
	high, ok := s.price.EstimateTextCost(ctx, key, model, UsageTokens{InputTokens: prefix + rest, OutputTokens: tutorEstOutputTokens})
	if !ok {
		return nil
	}
	low, ok := s.price.EstimateTextCost(ctx, key, model, UsageTokens{InputTokens: rest, CacheReadTokens: prefix, OutputTokens: tutorEstOutputTokens})
	if !ok || low > high {
		low = high
	}
	return &TutorCost{Low: low, High: high, InputTokens: prefix + rest, OutputTokens: tutorEstOutputTokens, Model: model}
}

// tierModel is the model behind a tier: 经济 is a fixed cheaper model, 标准 the learning site's.
func (s *TutorService) tierModel(ctx context.Context, tier string) string {
	if tier == TutorEconomy {
		return tutorEconomyModel
	}
	return s.model(ctx)
}

func (s *TutorService) model(ctx context.Context) string {
	if s.learn != nil {
		if st, _ := s.learn.loadSettings(ctx); st.Model != "" {
			return st.Model
		}
	}
	return "gpt-5.5"
}

func (s *TutorService) day() string { return learnDayStart(s.now()).Format("20060102") }

func tutorRandom(n int, alphabet string) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

func tutorTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// --- teacher: settings ------------------------------------------------------------------------------

// TutorInput is what the teacher sets.
type TutorInput struct {
	KeyID         int64  `json:"key_id"`
	Name          string `json:"name"`
	Template      string `json:"template"`
	Subject       string `json:"subject"`
	Grade         string `json:"grade"`
	Style         string `json:"style"`
	AnswerMode    string `json:"answer_mode"`
	ModelTier     string `json:"model_tier"`
	Rules         string `json:"rules"`
	Greeting      string `json:"greeting"`
	PassCode      string `json:"pass_code"`
	PerStudentDay int    `json:"per_student_day"`
	DailyCap      int    `json:"daily_cap"`
	Enabled       bool   `json:"enabled"`
}

func (s *TutorService) apply(ctx context.Context, userID int64, t *Tutor, in TutorInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if n := utf8.RuneCountInString(in.Name); n == 0 || n > 30 {
		return ErrTutorInput
	}
	if _, ok := findTutorTemplate(in.Template); !ok {
		return ErrTutorInput
	}
	if in.PerStudentDay < 1 || in.PerStudentDay > 200 || in.DailyCap < 0 || in.DailyCap > 20000 {
		return ErrTutorInput
	}
	if in.AnswerMode != TutorAnswer {
		in.AnswerMode = TutorGuide
	}
	if in.ModelTier != TutorEconomy {
		in.ModelTier = TutorStandard
	}
	pass := strings.TrimSpace(in.PassCode)
	if utf8.RuneCountInString(pass) > 20 {
		return ErrTutorInput
	}
	if in.KeyID != t.KeyID || t.ID == 0 {
		if s.learn == nil {
			return ErrTutorKey
		}
		if _, err := s.learn.ownKey(ctx, userID, in.KeyID); err != nil {
			return err
		}
	}
	t.KeyID, t.Name, t.Template = in.KeyID, in.Name, in.Template
	t.Subject = articleClip(strings.TrimSpace(in.Subject), 40)
	t.Grade = articleClip(strings.TrimSpace(in.Grade), 40)
	t.Style = articleClip(strings.TrimSpace(in.Style), 100)
	t.AnswerMode = in.AnswerMode
	t.ModelTier = in.ModelTier
	t.Rules = articleClip(strings.TrimSpace(in.Rules), 2000)
	t.Greeting = articleClip(strings.TrimSpace(in.Greeting), 500)
	t.PassCode = pass
	t.PerStudentDay, t.DailyCap, t.Enabled = in.PerStudentDay, in.DailyCap, in.Enabled
	return nil
}

func (s *TutorService) Create(ctx context.Context, userID int64, in TutorInput) (*Tutor, error) {
	if n, err := s.repo.Count(ctx, userID); err != nil {
		return nil, err
	} else if n >= tutorMaxPerTeacher {
		return nil, ErrTutorTooMany
	}
	if in.PerStudentDay == 0 {
		in.PerStudentDay = 20
	}
	t := &Tutor{UserID: userID, ShareCode: tutorRandom(8, "abcdefghjkmnpqrstuvwxyz23456789")}
	if err := s.apply(ctx, userID, t, in); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, t.ID)
}

func (s *TutorService) Update(ctx context.Context, userID, id int64, in TutorInput) (*Tutor, error) {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.apply(ctx, userID, t, in); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

func (s *TutorService) own(ctx context.Context, userID, id int64) (*Tutor, error) {
	t, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTutorNotFound
	}
	return t, nil
}

// Get returns the assistant with its materials (names only) and the state of its key.
func (s *TutorService) Get(ctx context.Context, userID, id int64) (*Tutor, error) {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if t.Materials, err = s.repo.ListMaterials(ctx, id, false); err != nil {
		return nil, err
	}
	t.MaterialChars = 0
	for _, m := range t.Materials {
		t.MaterialChars += m.Chars
	}
	if s.learn != nil {
		if k, err := s.learn.ownKey(ctx, userID, t.KeyID); err == nil {
			t.KeyName = k.Name
			t.Costs = map[string]*TutorCost{}
			for _, tier := range []string{TutorStandard, TutorEconomy} {
				if c := s.estimate(ctx, k, s.tierModel(ctx, tier), t.MaterialChars); c != nil {
					t.Costs[tier] = c
				}
			}
			t.Cost = t.Costs[t.ModelTier]
		} else {
			t.KeyProblem = infraerrors.Message(err)
		}
	}
	return t, nil
}

func (s *TutorService) List(ctx context.Context, userID int64) ([]Tutor, error) {
	return s.repo.List(ctx, userID)
}

func (s *TutorService) Delete(ctx context.Context, userID, id int64) error {
	if _, err := s.own(ctx, userID, id); err != nil {
		return err
	}
	s.dropKB(id)
	return s.repo.Delete(ctx, userID, id)
}

// AddMaterial adds a file (text is extracted) or pasted text (data nil, text set).
func (s *TutorService) AddMaterial(ctx context.Context, userID, id int64, name string, data []byte, text string) (*TutorMaterial, error) {
	t, err := s.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	name = articleClip(strings.TrimSpace(name), 200)
	if data != nil {
		if len(data) > tutorUploadMax {
			return nil, infraerrors.BadRequest("TUTOR_FILE", "文件太大，单个最多 20MB（File too large）")
		}
		text, err = s.extract(ctx, name, data)
		if err != nil {
			return nil, infraerrors.BadRequest("TUTOR_FILE", err.Error())
		}
	} else {
		text = tutorCleanText(text)
		if name == "" {
			name = "粘贴的资料"
		}
	}
	chars := utf8.RuneCountInString(text)
	if chars == 0 {
		return nil, infraerrors.BadRequest("TUTOR_FILE", "资料是空的（Empty material）")
	}
	if len(t.Materials) >= tutorMaxMaterials || t.MaterialChars+chars > tutorMaxMaterialChars {
		return nil, ErrTutorMaterial
	}
	m, err := s.repo.AddMaterial(ctx, id, name, text)
	if err != nil {
		return nil, err
	}
	s.dropKB(id)
	return m, nil
}

// extract reads a file's text, giving up after 30 s (a hostile PDF must not hang the request).
func (s *TutorService) extract(ctx context.Context, name string, data []byte) (string, error) {
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := extractTutorText(name, data)
		done <- result{text, err}
	}()
	select {
	case r := <-done:
		return r.text, r.err
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("文件太复杂，读取超时；可以直接粘贴文字")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *TutorService) DeleteMaterial(ctx context.Context, userID, id, materialID int64) error {
	if _, err := s.own(ctx, userID, id); err != nil {
		return err
	}
	s.dropKB(id)
	return s.repo.DeleteMaterial(ctx, id, materialID)
}

func (s *TutorService) dropKB(id int64) {
	s.kbMu.Lock()
	delete(s.kbs, id)
	s.kbMu.Unlock()
}

// context returns the materials for a question: all of them when short, else the best passages.
func (s *TutorService) context(ctx context.Context, tutorID int64, query string) (string, bool, error) {
	s.kbMu.Lock()
	cached, ok := s.kbs[tutorID]
	s.kbMu.Unlock()
	if !ok {
		mats, err := s.repo.ListMaterials(ctx, tutorID, true)
		if err != nil {
			return "", false, err
		}
		total := 0
		for _, m := range mats {
			total += m.Chars
		}
		if total <= tutorFullContextChars {
			var b strings.Builder
			for _, m := range mats {
				fmt.Fprintf(&b, "【%s】\n%s\n\n", m.Name, m.Content)
			}
			cached = tutorKB{full: strings.TrimSpace(b.String())}
		} else {
			var pages []AssistantPage
			for _, m := range mats {
				r := []rune(m.Content)
				for i, part := 0, 1; i < len(r); i, part = i+tutorPassageChars, part+1 {
					end := min(i+tutorPassageChars, len(r))
					pages = append(pages, AssistantPage{Title: fmt.Sprintf("%s（第 %d 部分）", m.Name, part), Text: string(r[i:end])})
				}
			}
			cached = tutorKB{kb: newAssistantKB(pages)}
		}
		s.kbMu.Lock()
		s.kbs[tutorID] = cached
		s.kbMu.Unlock()
	}
	if cached.kb == nil {
		return cached.full, false, nil
	}
	var b strings.Builder
	for _, h := range cached.kb.search(query, tutorPassages) {
		fmt.Fprintf(&b, "【%s】\n%s\n\n", h.Title, h.Text)
	}
	return strings.TrimSpace(b.String()), true, nil
}

// --- students ------------------------------------------------------------------------------------

// TutorPublic is what a student sees before joining.
type TutorPublic struct {
	Name        string   `json:"name"`
	Template    string   `json:"template"`
	Subject     string   `json:"subject"`
	Greeting    string   `json:"greeting"`
	Suggestions []string `json:"suggestions"`
	NeedsPass   bool     `json:"needs_pass"`
	Enabled     bool     `json:"enabled"`
}

func (s *TutorService) byCode(ctx context.Context, code string) (*Tutor, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" || len(code) > 16 {
		return nil, ErrTutorNotFound
	}
	t, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTutorNotFound
	}
	return t, nil
}

func (s *TutorService) Public(ctx context.Context, code string) (*TutorPublic, error) {
	t, err := s.byCode(ctx, code)
	if err != nil {
		return nil, err
	}
	tpl, _ := findTutorTemplate(t.Template)
	return &TutorPublic{Name: t.Name, Template: t.Template, Subject: t.Subject, Greeting: t.greetingText(), Suggestions: tpl.Suggestions,
		NeedsPass: t.PassCode != "", Enabled: t.Enabled}, nil
}

// TutorJoin is a student's session: the token goes with every question.
type TutorJoin struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Left  int    `json:"left"`
}

func (s *TutorService) Join(ctx context.Context, code, name, pass string) (*TutorJoin, error) {
	t, err := s.byCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !t.Enabled {
		return nil, ErrTutorClosed
	}
	if t.PassCode != "" && strings.TrimSpace(pass) != t.PassCode {
		return nil, ErrTutorPass
	}
	name = strings.Join(strings.Fields(name), " ")
	if n := utf8.RuneCountInString(name); n == 0 || n > 20 {
		return nil, ErrTutorName
	}
	token := tutorRandom(40, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	st, err := s.repo.UpsertStudent(ctx, t.ID, name, tutorTokenHash(token))
	if err != nil {
		return nil, err
	}
	s.cleanupMessages()
	return &TutorJoin{Token: token, Name: st.Name, Left: s.left(ctx, t, st.ID)}, nil
}

func (s *TutorService) cleanupMessages() {
	if s.now().Unix()%50 != 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, _ = s.repo.DeleteMessagesBefore(ctx, s.now().Add(-tutorMessageRetention))
	}()
}

func (s *TutorService) studentKey(studentID int64) string {
	return "tutor:" + s.day() + ":s:" + strconv.FormatInt(studentID, 10)
}

func (s *TutorService) tutorKey(tutorID int64) string {
	return "tutor:" + s.day() + ":t:" + strconv.FormatInt(tutorID, 10)
}

func (s *TutorService) left(ctx context.Context, t *Tutor, studentID int64) int {
	if s.quota == nil {
		return t.PerStudentDay
	}
	used, _ := s.quota.Get(ctx, s.studentKey(studentID))
	return max(t.PerStudentDay-int(used), 0)
}

// TutorChatInput: the conversation so far, the last message being the new question.
type TutorChatInput struct {
	Messages []LearnMessage `json:"messages"`
}

type TutorChatResult struct {
	Left int `json:"left"`
}

func validateTutorChat(in *TutorChatInput) (string, error) {
	msgs := in.Messages
	if len(msgs) == 0 || msgs[len(msgs)-1].Role != "user" {
		return "", ErrTutorQuestion
	}
	q := strings.TrimSpace(msgs[len(msgs)-1].Content)
	if q == "" || utf8.RuneCountInString(q) > tutorMaxQuestion {
		return "", ErrTutorQuestion
	}
	history := msgs[:len(msgs)-1]
	if len(history) > tutorMaxHistory {
		history = history[len(history)-tutorMaxHistory:]
	}
	var kept []LearnMessage
	for _, m := range history {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(m.Content)
		if r := []rune(text); len(r) > tutorMaxHistoryChars {
			text = string(r[:tutorMaxHistoryChars]) + "…"
		}
		if text != "" {
			kept = append(kept, LearnMessage{Role: m.Role, Content: text})
		}
	}
	in.Messages = append(kept, LearnMessage{Role: "user", Content: q})
	return q, nil
}

// Chat answers a student's question on the teacher's key.
func (s *TutorService) Chat(ctx context.Context, code, token string, in TutorChatInput, onDelta func(string) error) (*TutorChatResult, error) {
	t, err := s.byCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !t.Enabled {
		return nil, ErrTutorClosed
	}
	if strings.TrimSpace(token) == "" {
		return nil, ErrTutorSession
	}
	st, err := s.repo.StudentByToken(ctx, t.ID, tutorTokenHash(token))
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrTutorSession
	}
	question, err := validateTutorChat(&in)
	if err != nil {
		return nil, err
	}
	key, err := s.teacherKey(ctx, t)
	if err != nil {
		return nil, err
	}
	refund, used, err := s.count(ctx, t, st.ID)
	if err != nil {
		return nil, err
	}
	var answer strings.Builder
	err = s.answer(ctx, t, key, in, func(d string) error {
		_, _ = answer.WriteString(d)
		return onDelta(d)
	})
	if err != nil {
		refund()
		return nil, err
	}
	if err := s.repo.AddMessage(context.WithoutCancel(ctx), t.ID, st.ID, question, answer.String()); err != nil {
		return nil, err
	}
	return &TutorChatResult{Left: max(t.PerStudentDay-used, 0)}, nil
}

func (s *TutorService) teacherKey(ctx context.Context, t *Tutor) (string, error) {
	if s.learn == nil {
		return "", ErrTutorKey
	}
	k, err := s.learn.ownKey(ctx, t.UserID, t.KeyID)
	if err != nil {
		return "", ErrTutorKey
	}
	return k.Key, nil
}

// count takes one question off the student's and the assistant's daily allowance.
func (s *TutorService) count(ctx context.Context, t *Tutor, studentID int64) (func(), int, error) {
	if s.quota == nil {
		return func() {}, 0, nil
	}
	mine, all := s.studentKey(studentID), s.tutorKey(t.ID)
	used, err := s.quota.Incr(ctx, mine, tutorCounterTTL)
	if err != nil {
		return nil, 0, ErrTutorKey
	}
	if int(used) > t.PerStudentDay {
		_ = s.quota.Decr(ctx, mine)
		return nil, 0, infraerrors.TooManyRequests("TUTOR_QUOTA", fmt.Sprintf("今天的 %d 次提问用完了，明天再来，或者直接问老师（Daily questions used up）", t.PerStudentDay))
	}
	total, err := s.quota.Incr(ctx, all, tutorCounterTTL)
	if err != nil || (t.DailyCap > 0 && int(total) > t.DailyCap) {
		_ = s.quota.Decr(ctx, mine)
		if err == nil {
			_ = s.quota.Decr(ctx, all)
		}
		return nil, 0, infraerrors.TooManyRequests("TUTOR_BUSY", "今天全班的提问次数用完了，明天再来，或者直接问老师（Class limit reached）")
	}
	return func() {
		bg := context.WithoutCancel(ctx)
		_ = s.quota.Decr(bg, mine)
		_ = s.quota.Decr(bg, all)
	}, int(used), nil
}

func (s *TutorService) answer(ctx context.Context, t *Tutor, key string, in TutorChatInput, onDelta func(string) error) error {
	query := in.Messages[len(in.Messages)-1].Content
	for i := len(in.Messages) - 2; i >= 0; i-- {
		if in.Messages[i].Role == "user" {
			query += "\n" + in.Messages[i].Content
			break
		}
	}
	materials, partial, err := s.context(ctx, t.ID, query)
	if err != nil {
		return err
	}
	msgs := append([]LearnMessage{{Role: "system", Content: tutorSystemPrompt(t, materials, partial)}}, in.Messages...)

	// A call that fails before any text reached the student is tried again (the single upstream
	// account is briefly unavailable now and then, more often for the cheaper model): once on the
	// same model, then, for 经济, once on the standard model. Once text has been sent it is not
	// repeated; the error goes to the student.
	models := []string{s.tierModel(ctx, t.ModelTier), s.tierModel(ctx, t.ModelTier)}
	if t.ModelTier == TutorEconomy {
		models = append(models, s.model(ctx))
	}
	sent := false
	stream := func(d string) error {
		sent = true
		return onDelta(d)
	}
	for i, model := range models {
		_, err = s.chat(ctx, key, model, msgs, tutorMaxOutputTokens, stream)
		if err == nil || sent || ctx.Err() != nil || i == len(models)-1 {
			return err
		}
		slog.Info("tutor: retrying an answer", "tutor", t.ID, "model", model, "next", models[i+1], "err", err)
	}
	return err
}

// Preview lets the teacher try the assistant as a student (not counted, not recorded).
func (s *TutorService) Preview(ctx context.Context, userID, id int64, in TutorChatInput, onDelta func(string) error) error {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return err
	}
	if _, err := validateTutorChat(&in); err != nil {
		return err
	}
	key, err := s.teacherKey(ctx, t)
	if err != nil {
		return err
	}
	return s.answer(ctx, t, key, in, onDelta)
}

// --- teacher: 学情 --------------------------------------------------------------------------------------

type TutorStats struct {
	Students []TutorStudent `json:"students"`
	Today    int            `json:"today"`
	Week     int            `json:"week"`
	Recent   []TutorMessage `json:"recent"`
}

func (s *TutorService) Stats(ctx context.Context, userID, id int64) (*TutorStats, error) {
	if _, err := s.own(ctx, userID, id); err != nil {
		return nil, err
	}
	out := &TutorStats{}
	var err error
	if out.Students, err = s.repo.ListStudents(ctx, id); err != nil {
		return nil, err
	}
	if out.Today, err = s.repo.CountMessages(ctx, id, learnDayStart(s.now())); err != nil {
		return nil, err
	}
	if out.Week, err = s.repo.CountMessages(ctx, id, s.now().Add(-7*24*time.Hour)); err != nil {
		return nil, err
	}
	if out.Recent, err = s.repo.ListMessages(ctx, id, time.Time{}, tutorRecentMessages); err != nil {
		return nil, err
	}
	return out, nil
}

var ErrTutorNoQuestions = infraerrors.BadRequest("TUTOR_NO_QUESTIONS", "最近 7 天还没有学生提问，先把链接发给学生吧（No questions yet）")

// Insights asks the model to sum up the last week's questions for the teacher (on the teacher's key).
func (s *TutorService) Insights(ctx context.Context, userID, id int64, onDelta func(string) error) error {
	t, err := s.own(ctx, userID, id)
	if err != nil {
		return err
	}
	msgs, err := s.repo.ListMessages(ctx, id, s.now().Add(-7*24*time.Hour), tutorInsightQuestions)
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return ErrTutorNoQuestions
	}
	key, err := s.teacherKey(ctx, t)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "助教：%s（%s %s）\n最近 7 天共 %d 个问题：\n", t.Name, t.Grade, t.Subject, len(msgs))
	for _, m := range msgs {
		fmt.Fprintf(&b, "%s：%s\n", m.StudentName, strings.ReplaceAll(articleClip(m.Question, 200), "\n", " "))
	}
	_, err = s.chat(ctx, key, s.model(ctx), []LearnMessage{{Role: "system", Content: tutorInsightsPrompt}, {Role: "user", Content: b.String()}}, 2000, onDelta)
	return err
}
