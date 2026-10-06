package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// AI 学习 L2: quizzes graded on the server, checkpoints the server verifies, and certificates for
// finishing a track (all lessons, quizzes at 80% or more, its checkpoints and a project).

const (
	learnQuizPassPercent    = 80
	learnInterviewPassScore = 60
	learnCertCodeLen        = 10
	learnCertNameMaxRunes   = 20
)

var (
	ErrLearnQuizNotFound   = infraerrors.NotFound("LEARN_QUIZ_NOT_FOUND", "这节课没有测验（No quiz for this lesson）")
	ErrLearnQuizAnswers    = infraerrors.BadRequest("LEARN_QUIZ_ANSWERS", "请回答全部题目后再提交（Answer every question）")
	ErrLearnCheckpoint     = infraerrors.NotFound("LEARN_CHECKPOINT_NOT_FOUND", "检查点不存在（Unknown checkpoint）")
	ErrLearnTrack          = infraerrors.NotFound("LEARN_TRACK_NOT_FOUND", "学习路线不存在（Unknown track）")
	ErrLearnCertNotReady   = infraerrors.BadRequest("LEARN_CERT_NOT_READY", "还没达到结业条件，请先完成清单里没打勾的项目（Requirements not met）")
	ErrLearnCertName       = infraerrors.BadRequest("LEARN_CERT_NAME", "证书上的名字需要 1–20 个字，不能有特殊符号（Invalid name）")
	ErrLearnCertProject    = infraerrors.BadRequest("LEARN_CERT_PROJECT", "请选择你发布的网站，或填写结业项目的 https 公开链接（Choose a site or enter a project link）")
	ErrLearnCertNotFound   = infraerrors.NotFound("LEARN_CERT_NOT_FOUND", "证书不存在或已撤销（Certificate not found）")
	ErrLearnCertRestore    = infraerrors.Conflict("LEARN_CERT_RESTORE", "这位学员已经领了新证书，旧证书不能恢复（A newer certificate exists）")
	ErrLearnKeyPlatform    = infraerrors.BadRequest("LEARN_KEY_PLATFORM", "课程用的是 GPT 模型，这个 Key 的分组不支持，请换一个「GPT-按量」等 GPT 分组的 Key（Key group has no GPT models）")
	ErrLearnKeyInvalid     = infraerrors.BadRequest("LEARN_KEY_INVALID", "这个 Key 不可用，请换一个启用中的 Key（Key unavailable）")
	ErrLearnSourcesMissing = infraerrors.ServiceUnavailable("LEARN_CHECK_UNAVAILABLE", "暂时无法核对，请稍后再试（Check unavailable）")
)

// --- Sources the checkpoints look at ---------------------------------------------------------

type learnKeySource interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
	CountByUserID(ctx context.Context, userID int64) (int64, error)
	ListByUserID(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error)
}

type learnSiteSource interface {
	ListSitesByUser(ctx context.Context, userID int64) ([]Site, error)
}

type learnWorkSource interface {
	ListWorks(ctx context.Context, q WorkQuery) ([]Work, error)
	GetProfileByUser(ctx context.Context, userID int64) (*CommunityProfile, error)
}

// LearnSources are the other modules the checkpoints, certificates and own-key runs read.
type LearnSources struct {
	Keys  learnKeySource
	Sites learnSiteSource
	Works learnWorkSource
	// SiteURL is a hosted site's address (empty without site hosting).
	SiteURL func(name string) string
	// CanvasURL is the canvas site (profile links for track B).
	CanvasURL string
	// InviteCode returns the learner's invite code for the certificate's share poster.
	InviteCode func(ctx context.Context, userID int64) string
}

// --- Quizzes ---------------------------------------------------------------------------------

// LearnQuizView is a quiz without its answers.
type LearnQuizView struct {
	Lesson    string               `json:"lesson"`
	Questions []LearnQuizQuestionV `json:"questions"`
	Best      *LearnQuizResult     `json:"best,omitempty"`
}

type LearnQuizQuestionV struct {
	Q       string   `json:"q"`
	Options []string `json:"options"`
	Multi   bool     `json:"multi"`
}

type LearnQuizResult struct {
	Correct   int       `json:"correct"`
	Total     int       `json:"total"`
	Attempts  int       `json:"attempts"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r LearnQuizResult) Passed() bool {
	return r.Total > 0 && r.Correct*100 >= r.Total*learnQuizPassPercent
}

type LearnQuizGrade struct {
	Correct int                  `json:"correct"`
	Total   int                  `json:"total"`
	Passed  bool                 `json:"passed"`
	Items   []LearnQuizGradeItem `json:"items"`
	Best    LearnQuizResult      `json:"best"`
}

type LearnQuizGradeItem struct {
	Correct bool   `json:"correct"`
	Answer  []int  `json:"answer"`
	Explain string `json:"explain"`
}

// Quiz returns a lesson's questions (and the learner's best result when userID > 0).
func (s *LearnService) Quiz(ctx context.Context, userID int64, lesson string) (*LearnQuizView, error) {
	qs, ok := s.catalog.Quizzes[lesson]
	if !ok {
		return nil, ErrLearnQuizNotFound
	}
	out := &LearnQuizView{Lesson: lesson, Questions: make([]LearnQuizQuestionV, len(qs))}
	for i, q := range qs {
		out.Questions[i] = LearnQuizQuestionV{Q: q.Q, Options: q.Options, Multi: len(q.Answer) > 1}
	}
	if userID > 0 {
		results, err := s.repo.QuizResults(ctx, userID)
		if err != nil {
			return nil, err
		}
		if r, ok := results[lesson]; ok {
			out.Best = &r
		}
	}
	return out, nil
}

func sameAnswer(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	set := map[int]bool{}
	for _, a := range want {
		set[a] = true
	}
	for _, a := range got {
		if !set[a] {
			return false
		}
		delete(set, a)
	}
	return len(set) == 0
}

// SubmitQuiz grades the answers (one list of option indexes per question), keeps the best result,
// and marks the lesson done when it passes.
func (s *LearnService) SubmitQuiz(ctx context.Context, userID int64, lesson string, answers [][]int) (*LearnQuizGrade, error) {
	qs, ok := s.catalog.Quizzes[lesson]
	if !ok {
		return nil, ErrLearnQuizNotFound
	}
	if len(answers) != len(qs) {
		return nil, ErrLearnQuizAnswers
	}
	out := &LearnQuizGrade{Total: len(qs), Items: make([]LearnQuizGradeItem, len(qs))}
	for i, q := range qs {
		if len(answers[i]) == 0 || len(answers[i]) > len(q.Options) {
			return nil, ErrLearnQuizAnswers
		}
		ok := sameAnswer(answers[i], q.Answer)
		if ok {
			out.Correct++
		}
		out.Items[i] = LearnQuizGradeItem{Correct: ok, Answer: q.Answer, Explain: q.Explain}
	}
	best, err := s.repo.SaveQuizResult(ctx, userID, lesson, out.Correct, out.Total)
	if err != nil {
		return nil, err
	}
	out.Best = best
	out.Passed = LearnQuizResult{Correct: out.Correct, Total: out.Total}.Passed()
	if out.Passed {
		if err := s.repo.MarkDone(ctx, userID, []string{lesson}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// --- Checkpoints -----------------------------------------------------------------------------

type LearnCheckpointResult struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Passed bool   `json:"passed"`
	Hint   string `json:"hint,omitempty"`
}

// checkpointMet looks the rule up in the other modules (no record is written).
func (s *LearnService) checkpointMet(ctx context.Context, userID int64, def LearnCheckpointDef) (bool, error) {
	src := s.sources
	switch def.Rule {
	case "key":
		if src.Keys == nil {
			return false, ErrLearnSourcesMissing
		}
		n, err := src.Keys.CountByUserID(ctx, userID)
		return n > 0, err
	case "call":
		if src.Keys == nil {
			return false, ErrLearnSourcesMissing
		}
		keys, _, err := src.Keys.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{})
		if err != nil {
			return false, err
		}
		for _, k := range keys {
			if k.LastUsedAt != nil {
				return true, nil
			}
		}
		return false, nil
	case "site":
		sites, err := s.activeSites(ctx, userID)
		return len(sites) > 0, err
	case "works":
		n, err := s.publicWorks(ctx, userID, def.Kind)
		return n >= def.N, err
	}
	return false, nil
}

func (s *LearnService) activeSites(ctx context.Context, userID int64) ([]Site, error) {
	if s.sources.Sites == nil {
		return nil, nil
	}
	sites, err := s.sources.Sites.ListSitesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := sites[:0]
	for _, site := range sites {
		if site.Status == SiteStatusActive && site.Version > 0 {
			out = append(out, site)
		}
	}
	return out, nil
}

func (s *LearnService) publicWorks(ctx context.Context, userID int64, kind string) (int, error) {
	if s.sources.Works == nil {
		return 0, ErrLearnSourcesMissing
	}
	works, err := s.sources.Works.ListWorks(ctx, WorkQuery{Feed: "user", ViewerID: userID, UserID: userID, Kind: kind, Limit: 50})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, w := range works {
		if w.Status == WorkStatusApproved && w.Visibility == WorkVisibilityPublic {
			n++
		}
	}
	return n, nil
}

// VerifyCheckpoint checks a checkpoint now and records it when met.
func (s *LearnService) VerifyCheckpoint(ctx context.Context, userID int64, id string) (*LearnCheckpointResult, error) {
	def, ok := s.catalog.Checkpoints[id]
	if !ok {
		return nil, ErrLearnCheckpoint
	}
	done, err := s.repo.Checkpoints(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := &LearnCheckpointResult{ID: id, Title: def.Title}
	if _, ok := done[id]; ok {
		out.Passed = true
		return out, nil
	}
	met, err := s.checkpointMet(ctx, userID, def)
	if err != nil {
		return nil, err
	}
	if met {
		if err := s.repo.PassCheckpoint(ctx, userID, id); err != nil {
			return nil, err
		}
		out.Passed = true
	} else {
		out.Hint = def.Hint
	}
	return out, nil
}

// CheckpointInfo is a checkpoint's title and hint (public).
func (s *LearnService) CheckpointInfo(id string) (LearnCheckpointDef, bool) {
	def, ok := s.catalog.Checkpoints[id]
	return def, ok
}

// --- Certificates ----------------------------------------------------------------------------

type LearnCertificate struct {
	Code        string     `json:"code"`
	UserID      int64      `json:"-"`
	UserEmail   string     `json:"user_email,omitempty"`
	Track       string     `json:"track"`
	TrackTitle  string     `json:"track_title"`
	DisplayName string     `json:"display_name"`
	ProjectURL  string     `json:"project_url"`
	QuizScore   int        `json:"quiz_score"`
	IssuedAt    time.Time  `json:"issued_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	// Showcase: the holder shows the project on the learner wall; ShowcaseHidden: an admin took it off.
	Showcase       bool `json:"showcase"`
	ShowcaseHidden bool `json:"showcase_hidden"`
	// InviteCode (public view only): the holder's invite code for the share poster.
	InviteCode string `json:"invite_code,omitempty"`
}

type LearnCertItem struct {
	Kind   string `json:"kind"` // lessons | quiz | checkpoint | project
	ID     string `json:"id,omitempty"`
	Title  string `json:"title"`
	Done   bool   `json:"done"`
	Detail string `json:"detail,omitempty"`
}

type LearnProjectOption struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type LearnCertStatus struct {
	Track     string               `json:"track"`
	Title     string               `json:"title"`
	Eligible  bool                 `json:"eligible"`
	QuizScore int                  `json:"quiz_score"`
	Items     []LearnCertItem      `json:"items"`
	Projects  []LearnProjectOption `json:"projects"`
	// ProjectLink: the learner may enter a project link instead of picking a site.
	ProjectLink bool              `json:"project_link"`
	Certificate *LearnCertificate `json:"certificate,omitempty"`
}

// CertStatus is a track's certificate checklist for the learner (verifying open checkpoints).
func (s *LearnService) CertStatus(ctx context.Context, userID int64, trackID string) (*LearnCertStatus, error) {
	track := s.catalog.track(trackID)
	if track == nil {
		return nil, ErrLearnTrack
	}
	out := &LearnCertStatus{Track: track.ID, Title: track.Title, Items: []LearnCertItem{}, Projects: []LearnProjectOption{}}
	certs, err := s.repo.CertificatesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range certs {
		if certs[i].Track == track.ID && certs[i].RevokedAt == nil {
			c := certs[i]
			c.TrackTitle = track.Title
			out.Certificate = &c
		}
	}

	done, err := s.repo.Progress(ctx, userID)
	if err != nil {
		return nil, err
	}
	finished := 0
	for _, id := range track.Lessons {
		if _, ok := done[id]; ok {
			finished++
		}
	}
	out.Items = append(out.Items, LearnCertItem{Kind: "lessons", Title: fmt.Sprintf("学完全部 %d 课", len(track.Lessons)),
		Done: len(track.Lessons) > 0 && finished == len(track.Lessons), Detail: fmt.Sprintf("已完成 %d / %d", finished, len(track.Lessons))})

	results, err := s.repo.QuizResults(ctx, userID)
	if err != nil {
		return nil, err
	}
	correct, total, missing := 0, 0, 0
	for _, id := range track.Lessons {
		qs, ok := s.catalog.Quizzes[id]
		if !ok {
			continue
		}
		r, ok := results[id]
		if !ok {
			missing++
			total += len(qs)
			continue
		}
		correct += r.Correct
		total += r.Total
	}
	if total > 0 {
		out.QuizScore = int(math.Round(float64(correct) * 100 / float64(total)))
		detail := fmt.Sprintf("目前 %d%%", out.QuizScore)
		if missing > 0 {
			detail += fmt.Sprintf("，还有 %d 课的测验没做", missing)
		}
		out.Items = append(out.Items, LearnCertItem{Kind: "quiz", Title: fmt.Sprintf("各课测验总正确率不低于 %d%%", learnQuizPassPercent),
			Done: missing == 0 && out.QuizScore >= learnQuizPassPercent, Detail: detail})
	}

	passed, err := s.repo.Checkpoints(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, id := range track.Checkpoints {
		def := s.catalog.Checkpoints[id]
		item := LearnCertItem{Kind: "checkpoint", ID: id, Title: def.Title}
		if _, ok := passed[id]; ok {
			item.Done = true
		} else if met, err := s.checkpointMet(ctx, userID, def); err == nil && met {
			if err := s.repo.PassCheckpoint(ctx, userID, id); err != nil {
				return nil, err
			}
			item.Done = true
		} else {
			item.Detail = def.Hint
		}
		out.Items = append(out.Items, item)
	}

	project, err := s.projectItem(ctx, userID, track, out)
	if err != nil {
		return nil, err
	}
	out.Items = append(out.Items, project)

	out.Eligible = true
	for _, it := range out.Items {
		if it.Kind == "project" && out.ProjectLink {
			continue // a link entered when claiming
		}
		if !it.Done {
			out.Eligible = false
		}
	}
	return out, nil
}

func (s *LearnService) projectItem(ctx context.Context, userID int64, track *LearnTrackDef, out *LearnCertStatus) (LearnCertItem, error) {
	switch track.Project {
	case "site":
		sites, err := s.activeSites(ctx, userID)
		if err != nil {
			return LearnCertItem{}, err
		}
		for _, site := range sites {
			if s.sources.SiteURL == nil {
				break
			}
			title := site.Title
			if title == "" {
				title = site.Name
			}
			out.Projects = append(out.Projects, LearnProjectOption{URL: s.sources.SiteURL(site.Name), Title: title})
		}
		out.ProjectLink = true
		return LearnCertItem{Kind: "project", Title: "结业项目：把你做的东西发布出去", Done: len(out.Projects) > 0,
			Detail: "在「我的网站」发布的网站可以直接选；也可以在领证书时填写项目的公开链接（GitHub、Gitee 等）"}, nil
	case "works":
		n, err := s.publicWorks(ctx, userID, "")
		if err != nil {
			return LearnCertItem{}, err
		}
		const need = 3
		if n >= need && s.sources.Works != nil {
			if p, err := s.sources.Works.GetProfileByUser(ctx, userID); err == nil && p != nil && p.Handle != "" && s.sources.CanvasURL != "" {
				out.Projects = append(out.Projects, LearnProjectOption{URL: strings.TrimRight(s.sources.CanvasURL, "/") + "/u/" + p.Handle, Title: "我的画布主页"})
			}
		}
		return LearnCertItem{Kind: "project", Title: fmt.Sprintf("结业项目：在画布社区公开发布至少 %d 个作品", need),
			Done: n >= need, Detail: fmt.Sprintf("已公开发布 %d 个（审核通过后计入）", n)}, nil
	case "interviews":
		best, err := s.repo.BestInterviewScores(ctx, userID)
		if err != nil {
			return LearnCertItem{}, err
		}
		passed := 0
		var missing []string
		topics := track.Interviews
		for _, topic := range topics {
			if best[topic] >= learnInterviewPassScore {
				passed++
			} else {
				missing = append(missing, s.catalog.Interviews[topic].Title)
			}
		}
		detail := fmt.Sprintf("已通过 %d / %d 组", passed, len(topics))
		if len(missing) > 0 {
			detail += "，还差：" + strings.Join(missing, "、")
		}
		title := fmt.Sprintf("每组模拟面试都拿到 %d 分以上", learnInterviewPassScore)
		if len(topics) == 1 {
			title = fmt.Sprintf("「%s」模拟面试拿到 %d 分以上", s.catalog.Interviews[topics[0]].Title, learnInterviewPassScore)
		}
		return LearnCertItem{Kind: "project", Title: title, Done: passed == len(topics), Detail: detail}, nil
	}
	return LearnCertItem{Kind: "project", Title: "结业项目"}, nil
}

func cleanCertName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	n := utf8.RuneCountInString(name)
	if n == 0 || n > learnCertNameMaxRunes {
		return "", false
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(" ·-_.", r) {
			return "", false
		}
	}
	return name, true
}

func newLearnCertCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, learnCertCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}

// ClaimCertificate issues the track's certificate once every requirement is met (again: returns it).
func (s *LearnService) ClaimCertificate(ctx context.Context, userID int64, trackID, displayName, projectURL string, showcase bool) (*LearnCertificate, error) {
	st, err := s.CertStatus(ctx, userID, trackID)
	if err != nil {
		return nil, err
	}
	if st.Certificate != nil {
		return st.Certificate, nil
	}
	if !st.Eligible {
		return nil, ErrLearnCertNotReady
	}
	name, ok := cleanCertName(displayName)
	if !ok {
		return nil, ErrLearnCertName
	}
	project, err := pickLearnProject(st, strings.TrimSpace(projectURL))
	if err != nil {
		return nil, err
	}
	code, err := newLearnCertCode()
	if err != nil {
		return nil, err
	}
	cert := &LearnCertificate{Code: code, UserID: userID, Track: st.Track, TrackTitle: st.Title, DisplayName: name,
		ProjectURL: project, QuizScore: st.QuizScore, IssuedAt: s.now(), Showcase: showcase}
	if err := s.repo.CreateCertificate(ctx, cert); err != nil {
		return nil, err
	}
	// A concurrent claim may have won: return whichever is stored.
	certs, err := s.repo.CertificatesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range certs {
		if certs[i].Track == st.Track && certs[i].RevokedAt == nil {
			c := certs[i]
			c.TrackTitle = st.Title
			return &c, nil
		}
	}
	return cert, nil
}

// pickLearnProject: one of the offered projects, or (where allowed) a public https link.
func pickLearnProject(st *LearnCertStatus, projectURL string) (string, error) {
	for _, p := range st.Projects {
		if projectURL == "" || p.URL == projectURL {
			return p.URL, nil
		}
	}
	if !st.ProjectLink {
		if len(st.Projects) == 0 {
			return "", nil
		}
		return "", ErrLearnCertProject
	}
	u, err := url.Parse(projectURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || len(projectURL) > 300 {
		return "", ErrLearnCertProject
	}
	return u.String(), nil
}

// Certificate is the public view of a certificate (with the holder's invite code).
func (s *LearnService) Certificate(ctx context.Context, code string) (*LearnCertificate, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != learnCertCodeLen {
		return nil, ErrLearnCertNotFound
	}
	c, err := s.repo.CertificateByCode(ctx, code)
	if err != nil || c == nil || c.RevokedAt != nil {
		return nil, ErrLearnCertNotFound
	}
	if t := s.catalog.track(c.Track); t != nil {
		c.TrackTitle = t.Title
	}
	if s.sources.InviteCode != nil {
		c.InviteCode = s.sources.InviteCode(ctx, c.UserID)
	}
	c.UserEmail = ""
	return c, nil
}

// AdminCertificates lists certificates (newest first).
func (s *LearnService) AdminCertificates(ctx context.Context, page, pageSize int) ([]LearnCertificate, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	certs, total, err := s.repo.ListCertificates(ctx, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	for i := range certs {
		if t := s.catalog.track(certs[i].Track); t != nil {
			certs[i].TrackTitle = t.Title
		}
	}
	return certs, total, nil
}

// SetCertificateRevoked revokes a certificate (the learner may then claim again) or restores it.
func (s *LearnService) SetCertificateRevoked(ctx context.Context, code string, revoked bool) error {
	return s.repo.SetCertificateRevoked(ctx, strings.ToUpper(strings.TrimSpace(code)), revoked)
}

// --- The learner's own key -------------------------------------------------------------------

// ownKey is the learner's active key keyID (calls on it are billed as usual and not free).
func (s *LearnService) ownKey(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if s.sources.Keys == nil {
		return nil, ErrLearnKeyInvalid
	}
	k, err := s.sources.Keys.GetByID(ctx, keyID)
	if err != nil || k == nil || k.UserID != userID || !k.IsActive() || k.Key == "" {
		return nil, ErrLearnKeyInvalid
	}
	if !learnKeyUsable(k) {
		return nil, ErrLearnKeyPlatform
	}
	return k, nil
}

// learnKeyUsable: the lessons call GPT models, so only keys in an OpenAI-platform group can run them.
func learnKeyUsable(k *APIKey) bool {
	return k.Group == nil || k.Group.Platform == "" || k.Group.Platform == PlatformOpenAI
}

// LearnKeyOption is one of the learner's keys offered for runs (own-key-only mode, or past the free ones).
type LearnKeyOption struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Group string `json:"group"`
}

// OwnKeys lists the learner's active keys (names only).
func (s *LearnService) OwnKeys(ctx context.Context, userID int64) ([]LearnKeyOption, error) {
	out := []LearnKeyOption{}
	if s.sources.Keys == nil {
		return out, nil
	}
	keys, _, err := s.sources.Keys.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{})
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		if !k.IsActive() || !learnKeyUsable(&k) {
			continue
		}
		opt := LearnKeyOption{ID: k.ID, Name: k.Name}
		if k.Group != nil {
			opt.Group = k.Group.Name
		}
		out = append(out, opt)
	}
	return out, nil
}
