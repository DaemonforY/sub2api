package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// 写作台 (内容运营 V2, docs/CONTENT_ENGINE_V2.md §2): an article written in the owner's own voice,
// with the owner choosing at every step instead of a pipeline deciding for them.
//
//	材料 → 角度 → 三稿 → 精修 → 定稿 → 平台版本
//
//   - 作者档案 (persona): who the owner is, what they really did, their stances and how they write,
//     distilled from their own articles and edited by them. Every prompt writes as this person.
//   - 材料关: fact cards with sources (web search, the owner's notes and experiences) and at most
//     three questions to the owner; the owner keeps, doubts or rejects each card. Drafts may only
//     use the cards: nothing is invented, missing experiences become 【这里需要你的真实经历】.
//   - 角度: four angles with HKR scores, an archetype and a real opening; the owner picks or edits.
//     A hit article can be pasted to borrow its skeleton (never its content).
//   - 三稿: three drafts in different structures, a simulated reader marking where they would stop
//     reading, and a judge ranking them (twice, in two orders) and pulling out highlights; the
//     owner picks the base and the highlights to merge.
//   - 精修: the owner edits in the browser; the selection menu asks for rewrites, analogies or
//     evidence; the check runs fixed rules plus an editor's read and proposes replacements;
//     every AI step and manual snapshot is a version.
//   - 定稿: the AI draft and the final text are compared and the owner's edits become writing
//     rules (content_rules, origin "learned"); then the platform versions are written from the
//     final text and go through the existing 待发布 review.
//
// Desk articles are content_items with status "desk" and their state in data.desk. Background
// steps run one at a time per article (claim); their results are merged into a freshly loaded
// article under a per-article lock, so edits made meanwhile are not lost.

const (
	ContentDeskStatus = "desk"

	ContentStageMaterials = "materials"
	ContentStageAngles    = "angles"
	ContentStageDrafts    = "drafts"
	ContentStageEdit      = "edit"
	ContentStageFinal     = "final"

	ContentFactUser    = "user"
	ContentFactPersona = "persona"
	ContentFactAnswer  = "answer"
	ContentFactWeb     = "web"

	ContentFactNew      = "new"
	ContentFactOK       = "ok"
	ContentFactDoubt    = "doubt"
	ContentFactRejected = "rejected"

	ContentJobRunning = "running"
	ContentJobDone    = "done"
	ContentJobFailed  = "failed"

	contentMaterialsWanted = 5
	contentMaxFacts        = 60
	contentMaxFactText     = 600
	contentMaxQuestions    = 3
	contentMaxComments     = 100
	contentMaxSamples      = 5
	contentSampleChars     = 8000
	contentMaxRefText      = 20000
	contentMaxSelection    = 3000
	contentMaxListItem     = 300
	contentMaxListItems    = 20
	contentDeskSearches    = 5
	contentAssistTimeout   = 3 * time.Minute
	contentJudgeRounds     = 2
	contentDeskDraftTokens = 12000
)

var (
	ErrContentPersona     = infraerrors.BadRequest("CONTENT_PERSONA", "作者档案内容太长：每项最多 20 条、每条 300 字，自我介绍和表达习惯各 2000 字以内（Persona too long）")
	ErrContentSample      = infraerrors.BadRequest("CONTENT_SAMPLE", "范文需要一个公众号文章链接，或者标题加正文（至少 200 字）；最多 5 篇（Sample required）")
	ErrContentNoSamples   = infraerrors.BadRequest("CONTENT_NO_SAMPLES", "先在作者档案里加 2 篇以上你自己写的文章，AI 才能从里面提炼你的写法（Add samples first）")
	ErrContentDeskStage   = infraerrors.Conflict("CONTENT_DESK_STAGE", "这一步还不能做：先完成前面的步骤，或者等正在运行的 AI 任务结束（Not available at this step）")
	ErrContentFacts       = infraerrors.BadRequest("CONTENT_FACTS", fmt.Sprintf("事实卡最多 %d 张，每张不超过 %d 字，状态只能是 ok / doubt / rejected / new（Invalid fact cards）", contentMaxFacts, contentMaxFactText))
	ErrContentAngle       = infraerrors.BadRequest("CONTENT_ANGLE", "选一个角度，角度和标题不能为空（Angle required）")
	ErrContentRef         = infraerrors.BadRequest("CONTENT_REF", "贴一篇参考文章的正文（至少 300 字），或者一个公众号文章链接（Reference article required）")
	ErrContentDraftPick   = infraerrors.BadRequest("CONTENT_DRAFT_PICK", "选一份初稿做底稿（Pick a base draft）")
	ErrContentBody        = infraerrors.BadRequest("CONTENT_BODY", "正文不能为空，最多 3 万字（Body required）")
	ErrContentAssist      = infraerrors.BadRequest("CONTENT_ASSIST", "先选中一段文字（最多 3000 字），再选要做什么（Select some text first）")
	ErrContentVersion     = infraerrors.NotFound("CONTENT_VERSION", "没有找到这个版本（Version not found）")
	ErrContentPlaceholder = infraerrors.BadRequest("CONTENT_PLACEHOLDER", "正文里还有「【这里需要你的真实经历】」没补，补完或删掉再定稿（Fill the placeholders first）")
)

// --- types -------------------------------------------------------------------------------------

type ContentSample struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	URL     string    `json:"url,omitempty"`
	Text    string    `json:"text"`
	AddedAt time.Time `json:"added_at"`
}

// ContentPersona is the owner's 作者档案.
type ContentPersona struct {
	Name        string          `json:"name"`
	Identity    string          `json:"identity"`
	Experiences []string        `json:"experiences"`
	Stances     []string        `json:"stances"`
	Likes       []string        `json:"likes"`
	Dislikes    []string        `json:"dislikes"`
	Voice       string          `json:"voice"`
	Phrases     []string        `json:"phrases"`
	Taboos      []string        `json:"taboos"`
	Samples     []ContentSample `json:"samples"`
	UpdatedAt   *time.Time      `json:"updated_at,omitempty"`
}

func (p *ContentPersona) empty() bool {
	return p.Identity == "" && p.Voice == "" && len(p.Experiences) == 0 && len(p.Stances) == 0 && len(p.Phrases) == 0 && len(p.Samples) == 0
}

// excerpts: the first `chars` characters of up to n samples.
func (p *ContentPersona) excerpts(n, chars int) []string {
	var out []string
	for _, s := range p.Samples {
		if len(out) >= n {
			break
		}
		text := truncateRunes(strings.TrimSpace(s.Text), chars)
		if text != "" {
			out = append(out, "《"+s.Title+"》\n"+text)
		}
	}
	return out
}

type ContentFact struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	Kind        string `json:"kind"` // user | persona | answer | web
	Source      string `json:"source,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
	Status      string `json:"status"` // new | ok | doubt | rejected
}

type ContentQuestion struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Why      string `json:"why,omitempty"`
	Answer   string `json:"answer,omitempty"`
}

type ContentAngle struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Angle     string `json:"angle"`
	Archetype string `json:"archetype"`
	Reader    string `json:"reader"`
	Hook      string `json:"hook"`
	H         int    `json:"h"`
	K         int    `json:"k"`
	R         int    `json:"r"`
	Why       string `json:"why"`
	Note      string `json:"note,omitempty"` // the owner's extra instructions
}

type ContentSkeletonPart struct {
	Role string `json:"role"`
	How  string `json:"how"`
}

type ContentSkeleton struct {
	Source       string                `json:"source"` // title or URL of the reference
	Opening      string                `json:"opening"`
	Parts        []ContentSkeletonPart `json:"parts"`
	Emotion      string                `json:"emotion"`
	Escalation   string                `json:"escalation"`
	Callbacks    string                `json:"callbacks"`
	TitlePattern string                `json:"title_pattern"`
}

type ContentParaMark struct {
	I   int    `json:"i"`
	V   string `json:"v"` // read | skim | leave
	Why string `json:"why,omitempty"`
}

type ContentDraftReview struct {
	Paras     []ContentParaMark `json:"paras"`
	Hook      int               `json:"hook"`
	Flow      int               `json:"flow"`
	Voice     int               `json:"voice"`
	Substance int               `json:"substance"`
	BestLine  string            `json:"best_line"`
	Problems  []string          `json:"problems"`
	Summary   string            `json:"summary"`
}

type ContentDraft struct {
	ID        string              `json:"id"`
	Structure string              `json:"structure"`
	Label     string              `json:"label"`
	Markdown  string              `json:"markdown"`
	Review    *ContentDraftReview `json:"review,omitempty"`
	Error     string              `json:"error,omitempty"`
}

type ContentHighlight struct {
	ID      string `json:"id"`
	DraftID string `json:"draft_id"`
	Quote   string `json:"quote"`
	Why     string `json:"why"`
}

type ContentJudge struct {
	Ranking    []string           `json:"ranking"` // draft IDs, best first
	Reasons    []string           `json:"reasons"`
	Highlights []ContentHighlight `json:"highlights"`
}

type ContentIssue struct {
	ID      string `json:"id"`
	Para    int    `json:"para,omitempty"`
	Layer   string `json:"layer"` // rule | voice | flow | content | fact
	Level   string `json:"level"` // must | should
	Quote   string `json:"quote"`
	Problem string `json:"problem"`
	Fix     string `json:"fix,omitempty"`
}

type ContentCheckReport struct {
	At      time.Time         `json:"at"`
	Stats   ContentProseStats `json:"stats"`
	Issues  []ContentIssue    `json:"issues"`
	Scores  map[string]int    `json:"scores,omitempty"`
	Summary string            `json:"summary,omitempty"`
}

type ContentComment struct {
	ID       string    `json:"id"`
	Quote    string    `json:"quote"`
	Text     string    `json:"text"`
	At       time.Time `json:"at"`
	Resolved bool      `json:"resolved,omitempty"`
}

type ContentLearn struct {
	ID       string `json:"id"`
	Rule     string `json:"rule"`
	Evidence string `json:"evidence"`
	Status   string `json:"status"` // new | added | dismissed
}

type ContentJob struct {
	Kind       string     `json:"kind"`
	Status     string     `json:"status"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// ContentDesk is the 写作台 state of an article.
type ContentDesk struct {
	Stage        string              `json:"stage"`
	Job          *ContentJob         `json:"job,omitempty"`
	ResearchHint string              `json:"research_hint,omitempty"`
	Facts        []ContentFact       `json:"facts"`
	Questions    []ContentQuestion   `json:"questions"`
	Angles       []ContentAngle      `json:"angles"`
	Angle        *ContentAngle       `json:"angle,omitempty"`
	Skeleton     *ContentSkeleton    `json:"skeleton,omitempty"`
	Drafts       []ContentDraft      `json:"drafts"`
	Judge        *ContentJudge       `json:"judge,omitempty"`
	BaseID       string              `json:"base_id,omitempty"`
	Picked       []string            `json:"picked,omitempty"`
	Body         string              `json:"body"`
	BodyAI       string              `json:"body_ai,omitempty"` // the text as the AI handed it over, for learning
	Check        *ContentCheckReport `json:"check,omitempty"`
	Comments     []ContentComment    `json:"comments"`
	Learn        []ContentLearn      `json:"learn"`
	FinalAt      *time.Time          `json:"final_at,omitempty"`
	NextID       int                 `json:"next_id"`
}

func (d *ContentDesk) id(prefix string) string {
	d.NextID++
	return fmt.Sprintf("%s%d", prefix, d.NextID)
}

func (d *ContentDesk) usableFacts() int {
	n := 0
	for _, f := range d.Facts {
		if f.Status != ContentFactRejected {
			n++
		}
	}
	return n
}

func (d *ContentDesk) running() bool { return d.Job != nil && d.Job.Status == ContentJobRunning }

// ContentVersion is a snapshot of a desk article's text.
type ContentVersion struct {
	ID          int64     `json:"id"`
	ItemID      int64     `json:"item_id"`
	OwnerUserID int64     `json:"-"`
	Kind        string    `json:"kind"` // draft | merge | manual | ai | restore | final
	Label       string    `json:"label"`
	Markdown    string    `json:"markdown,omitempty"`
	Chars       int       `json:"chars"`
	CreatedAt   time.Time `json:"created_at"`
}

// ContentDeskInput starts a desk article.
type ContentDeskInput struct {
	TopicID int64  `json:"topic_id"`
	Title   string `json:"title"`
	Idea    string `json:"idea"`
	Notes   string `json:"notes"`
}

type ContentAssistInput struct {
	Action      string `json:"action"` // voice | tighten | expand | sharper | lighter | plain | custom | analogy | evidence
	Selection   string `json:"selection"`
	Before      string `json:"before"`
	After       string `json:"after"`
	Instruction string `json:"instruction"`
}

type ContentAssistOption struct {
	Text        string `json:"text"`
	Note        string `json:"note,omitempty"`
	Source      string `json:"source,omitempty"`
	SourceTitle string `json:"source_title,omitempty"`
}

type ContentAssistResult struct {
	Text    string                `json:"text,omitempty"`
	Note    string                `json:"note,omitempty"`
	Options []ContentAssistOption `json:"options,omitempty"`
}

// ContentFactfetcher reads 公众号 articles (WritingStyleService.Fetch).
type ContentArticleFetcher interface {
	Fetch(ctx context.Context, raw string) (*StyleArticle, error)
}

// SetFetcher wires the 公众号 article reader for samples and reference articles.
func (s *ContentEngineService) SetFetcher(f ContentArticleFetcher) { s.fetcher = f }

// --- per-article lock --------------------------------------------------------------------------

func (s *ContentEngineService) itemLock(id int64) *sync.Mutex {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()
	if s.locks == nil {
		s.locks = map[int64]*sync.Mutex{}
	}
	m := s.locks[id]
	if m == nil {
		m = &sync.Mutex{}
		s.locks[id] = m
	}
	return m
}

// updateDesk loads the article fresh, applies fn and saves, under the article's lock.
func (s *ContentEngineService) updateDesk(ctx context.Context, owner, id int64, fn func(it *ContentItem) error) (*ContentItem, error) {
	m := s.itemLock(id)
	m.Lock()
	defer m.Unlock()
	it, err := s.repo.GetItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it == nil || it.Desk == nil {
		return nil, ErrContentNotFound
	}
	if err := fn(it); err != nil {
		return nil, err
	}
	if err := s.repo.SaveItem(ctx, it); err != nil {
		return nil, err
	}
	it.Running = s.running(it.ID)
	return it, nil
}

func (s *ContentEngineService) deskItem(ctx context.Context, owner, id int64) (*ContentItem, error) {
	it, err := s.item(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it.Desk == nil {
		return nil, ErrContentNotFound
	}
	return it, nil
}

// --- persona -----------------------------------------------------------------------------------

func (s *ContentEngineService) persona(ctx context.Context, owner int64) (*ContentPersona, error) {
	p, err := s.repo.GetPersona(ctx, owner)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = &ContentPersona{}
	}
	if p.Samples == nil {
		p.Samples = []ContentSample{}
	}
	return p, nil
}

// Persona returns the owner's 作者档案.
func (s *ContentEngineService) Persona(ctx context.Context, owner int64) (*ContentPersona, error) {
	return s.persona(ctx, owner)
}

func cleanContentList(in []string) ([]string, bool) {
	out := []string{}
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if utf8.RuneCountInString(x) > contentMaxListItem || len(out) >= contentMaxListItems {
			return nil, false
		}
		out = append(out, x)
	}
	return out, true
}

// SavePersona stores the editable fields; samples are managed separately.
func (s *ContentEngineService) SavePersona(ctx context.Context, owner int64, in ContentPersona) (*ContentPersona, error) {
	p, err := s.persona(ctx, owner)
	if err != nil {
		return nil, err
	}
	in.Name, in.Identity, in.Voice = strings.TrimSpace(in.Name), strings.TrimSpace(in.Identity), strings.TrimSpace(in.Voice)
	if utf8.RuneCountInString(in.Name) > 40 || utf8.RuneCountInString(in.Identity) > 2000 || utf8.RuneCountInString(in.Voice) > 2000 {
		return nil, ErrContentPersona
	}
	lists := []*[]string{&in.Experiences, &in.Stances, &in.Likes, &in.Dislikes, &in.Phrases, &in.Taboos}
	for _, l := range lists {
		cleaned, ok := cleanContentList(*l)
		if !ok {
			return nil, ErrContentPersona
		}
		*l = cleaned
	}
	now := s.now()
	in.Samples, in.UpdatedAt = p.Samples, &now
	if err := s.repo.SavePersona(ctx, owner, &in); err != nil {
		return nil, err
	}
	return &in, nil
}

// AddPersonaSample adds one of the owner's own articles: a 公众号 link (fetched) or title + text.
func (s *ContentEngineService) AddPersonaSample(ctx context.Context, owner int64, url, title, text string) (*ContentPersona, error) {
	p, err := s.persona(ctx, owner)
	if err != nil {
		return nil, err
	}
	if len(p.Samples) >= contentMaxSamples {
		return nil, ErrContentSample
	}
	sample := ContentSample{URL: strings.TrimSpace(url), Title: strings.TrimSpace(title), Text: strings.TrimSpace(text), AddedAt: s.now()}
	if sample.URL != "" {
		if s.fetcher == nil {
			return nil, ErrContentSample
		}
		a, err := s.fetcher.Fetch(ctx, sample.URL)
		if err != nil {
			return nil, err
		}
		sample.Title, sample.Text = a.Title, a.Text
	}
	if utf8.RuneCountInString(sample.Text) < 200 {
		return nil, ErrContentSample
	}
	if sample.Title == "" {
		sample.Title = truncateRunes(strings.SplitN(sample.Text, "\n", 2)[0], 40)
	}
	sample.Title, sample.Text = truncateRunes(sample.Title, 100), truncateRunes(sample.Text, contentSampleChars)
	sample.ID = randomContentID()
	p.Samples = append(p.Samples, sample)
	now := s.now()
	p.UpdatedAt = &now
	if err := s.repo.SavePersona(ctx, owner, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ContentEngineService) DeletePersonaSample(ctx context.Context, owner int64, sampleID string) (*ContentPersona, error) {
	p, err := s.persona(ctx, owner)
	if err != nil {
		return nil, err
	}
	out := p.Samples[:0]
	for _, x := range p.Samples {
		if x.ID != sampleID {
			out = append(out, x)
		}
	}
	p.Samples = out
	if err := s.repo.SavePersona(ctx, owner, p); err != nil {
		return nil, err
	}
	return p, nil
}

// DistillPersona reads the samples and proposes a 作者档案; nothing is saved — the owner edits
// the proposal and saves it.
func (s *ContentEngineService) DistillPersona(ctx context.Context, owner int64) (*ContentPersona, error) {
	p, err := s.persona(ctx, owner)
	if err != nil {
		return nil, err
	}
	if len(p.Samples) < 2 {
		return nil, ErrContentNoSamples
	}
	r, err := s.deskRun(ctx, owner, 0)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for i, x := range p.Samples {
		fmt.Fprintf(&b, "=== 第 %d 篇：《%s》 ===\n%s\n\n", i+1, x.Title, x.Text)
	}
	var out ContentPersona
	if err := s.callJSON(ctx, nil, r, contentPersonaDistillPrompt, b.String(), 6000, nil, 1, &out); err != nil {
		return nil, err
	}
	for _, l := range []*[]string{&out.Experiences, &out.Stances, &out.Likes, &out.Dislikes, &out.Phrases, &out.Taboos} {
		cleaned := []string{}
		for _, x := range *l {
			if x = truncateRunes(strings.TrimSpace(x), contentMaxListItem); x != "" && len(cleaned) < contentMaxListItems {
				cleaned = append(cleaned, x)
			}
		}
		*l = cleaned
	}
	out.Name, out.Identity, out.Voice = truncateRunes(strings.TrimSpace(out.Name), 40), truncateRunes(strings.TrimSpace(out.Identity), 2000), truncateRunes(strings.TrimSpace(out.Voice), 2000)
	out.Samples = p.Samples
	return &out, nil
}

// --- run setup ---------------------------------------------------------------------------------

// deskRun builds the model call context for an owner: key (the article's, or the settings' when
// id is 0), model, persona, rules.
func (s *ContentEngineService) deskRun(ctx context.Context, owner int64, keyID int64) (*contentRun, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return nil, err
	}
	if keyID <= 0 {
		keyID = row.Settings.KeyID
	}
	key, err := s.key(ctx, owner, keyID)
	if err != nil {
		return nil, err
	}
	p, err := s.persona(ctx, owner)
	if err != nil {
		return nil, err
	}
	return &contentRun{key: key.Key, model: s.model(ctx, row.Settings), product: row.Settings.Product, style: row.Settings.StylePrompt,
		brand: s.siteName(ctx), rules: s.activeRules(ctx, owner), persona: contentPersonaBlock(p), personaData: p}, nil
}

// startDeskJob runs work in the background on a fresh snapshot of the article; the returned apply
// function is run on the then-current article, so edits made meanwhile survive.
func (s *ContentEngineService) startDeskJob(ctx context.Context, owner, id int64, kind string, allowed func(d *ContentDesk) bool,
	work func(ctx context.Context, snap *ContentItem, r *contentRun) (func(it *ContentItem), error)) (*ContentItem, error) {
	it, err := s.deskItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if it.Desk.running() || (allowed != nil && !allowed(it.Desk)) {
		return nil, ErrContentDeskStage
	}
	r, err := s.deskRun(ctx, owner, it.KeyID)
	if err != nil {
		return nil, err
	}
	runCtx, ok := s.claim(id)
	if !ok {
		return nil, ErrContentDeskStage
	}
	it, err = s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		it.Desk.Job = &ContentJob{Kind: kind, Status: ContentJobRunning, StartedAt: s.now()}
		it.Error = ""
		return nil
	})
	if err != nil {
		s.release(id)
		return nil, err
	}
	snap := cloneContentItem(it)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.release(id)
		var apply func(*ContentItem)
		var werr error
		p0, c0 := snap.PromptTokens, snap.CompletionTokens
		select {
		case s.slots <- struct{}{}:
			apply, werr = work(runCtx, snap, r)
			<-s.slots
		case <-runCtx.Done():
			werr = ErrContentSiteBusy
		}
		bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, serr := s.updateDesk(bg, owner, id, func(cur *ContentItem) error {
			cur.PromptTokens += snap.PromptTokens - p0
			cur.CompletionTokens += snap.CompletionTokens - c0
			if snap.Model != "" {
				cur.Model = snap.Model
			}
			now := s.now()
			job := &ContentJob{Kind: kind, Status: ContentJobDone, StartedAt: snap.Desk.Job.StartedAt, FinishedAt: &now}
			switch {
			case werr == nil:
				if apply != nil {
					apply(cur)
				}
			case errors.Is(runCtx.Err(), context.Canceled):
				job.Status, job.Error = ContentJobFailed, "已停止"
			case errors.Is(runCtx.Err(), context.DeadlineExceeded):
				job.Status, job.Error = ContentJobFailed, "超时了，再试一次"
			default:
				job.Status, job.Error = ContentJobFailed, contentErrorText(id, werr)
			}
			cur.Desk.Job = job
			return nil
		})
		if serr != nil {
			slog.Warn("content desk: saving job result", "item", id, "kind", kind, "err", serr)
		}
	}()
	return it, nil
}

// --- create & materials ------------------------------------------------------------------------

// CreateDesk starts an article on the 写作台 (from a topic or from scratch) and starts the
// research for its fact cards.
func (s *ContentEngineService) CreateDesk(ctx context.Context, owner int64, in ContentDeskInput) (*ContentItem, error) {
	row, err := s.loadRow(ctx, owner)
	if err != nil {
		return nil, err
	}
	if _, err := s.key(ctx, owner, row.Settings.KeyID); err != nil {
		return nil, err
	}
	var topic *ContentTopic
	if in.TopicID > 0 {
		if topic, err = s.topic(ctx, owner, in.TopicID); err != nil {
			return nil, err
		}
		if in.Title == "" {
			in.Title = topic.Title
		}
		if in.Idea == "" {
			in.Idea = topic.Angle
		}
		if in.Notes == "" {
			in.Notes = topic.Notes
		}
	}
	ti := ContentTopicInput{Title: in.Title, Angle: in.Idea, Notes: in.Notes}
	if err := normalizeContentTopic(&ti); err != nil {
		return nil, err
	}
	it := &ContentItem{OwnerUserID: owner, KeyID: row.Settings.KeyID, Status: ContentDeskStatus, Title: truncateRunes(ti.Title, 200),
		ContentItemData: ContentItemData{
			Brief:   ContentBrief{Topic: ti.Title, Angle: ti.Angle, Notes: ti.Notes, Source: "desk", Platforms: row.Settings.Platforms, Search: row.Settings.Search && s.searchAvailable(ctx)},
			Sources: []ArticleSource{}, Events: []ArticleEvent{},
			Desk: &ContentDesk{Stage: ContentStageMaterials, Facts: []ContentFact{}, Questions: []ContentQuestion{}, Angles: []ContentAngle{}, Drafts: []ContentDraft{}, Comments: []ContentComment{}, Learn: []ContentLearn{}},
		}}
	if topic != nil {
		tid := topic.ID
		it.TopicID = &tid
	}
	if err := s.repo.CreateItem(ctx, it); err != nil {
		return nil, err
	}
	if topic != nil {
		itemID := it.ID
		topic.Status, topic.ItemID = ContentTopicUsed, &itemID
		if err := s.repo.UpdateTopic(ctx, topic); err != nil {
			slog.Warn("content desk: marking topic used", "topic", topic.ID, "err", err)
		}
	}
	return s.Research(ctx, owner, it.ID, "")
}

// Research (re)builds the fact cards: the owner's notes and experiences plus web search, and up
// to three questions to the owner. Existing cards are kept.
func (s *ContentEngineService) Research(ctx context.Context, owner, id int64, hint string) (*ContentItem, error) {
	hint = truncateRunes(strings.TrimSpace(hint), 300)
	if hint != "" {
		if _, err := s.updateDesk(ctx, owner, id, func(it *ContentItem) error { it.Desk.ResearchHint = hint; return nil }); err != nil {
			return nil, err
		}
	}
	return s.startDeskJob(ctx, owner, id, "research", nil, func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
		search := snap.Brief.Search && s.searchAvailable(ctx)
		var tools []AgentTool
		calls := 1
		if search {
			tools, calls = []AgentTool{s.searchTool(snap, true)}, contentDeskSearches+1
		}
		var out struct {
			Facts []struct {
				Text        string `json:"text"`
				Kind        string `json:"kind"`
				Source      string `json:"source"`
				SourceTitle string `json:"source_title"`
			} `json:"facts"`
			Questions []struct {
				Question string `json:"question"`
				Why      string `json:"why"`
			} `json:"questions"`
		}
		if err := s.callJSON(ctx, snap, r, contentFactsPrompt(r.persona, search), contentFactsRequest(snap), 5000, tools, calls, &out); err != nil {
			return nil, err
		}
		seen := contentSourceSet(snap.Sources)
		return func(it *ContentItem) {
			d := it.Desk
			have := map[string]bool{}
			for _, f := range d.Facts {
				have[f.Text] = true
			}
			for _, f := range out.Facts {
				text := truncateRunes(strings.TrimSpace(f.Text), contentMaxFactText)
				if text == "" || have[text] || len(d.Facts) >= contentMaxFacts {
					continue
				}
				fact := ContentFact{ID: d.id("F"), Text: text, Kind: f.Kind, Source: truncateRunes(strings.TrimSpace(f.Source), 500), SourceTitle: truncateRunes(strings.TrimSpace(f.SourceTitle), 120), Status: ContentFactNew}
				switch fact.Kind {
				case ContentFactUser, ContentFactPersona:
					fact.Source, fact.Status = "作者", ContentFactOK
				default:
					// A web fact must point at a page the search actually returned.
					fact.Kind = ContentFactWeb
					if !seen[contentURLKey(fact.Source)] {
						continue
					}
				}
				have[text] = true
				d.Facts = append(d.Facts, fact)
			}
			open := 0
			for _, q := range d.Questions {
				if q.Answer == "" {
					open++
				}
			}
			for _, q := range out.Questions {
				text := truncateRunes(strings.TrimSpace(q.Question), 300)
				if text == "" || open >= contentMaxQuestions {
					continue
				}
				d.Questions = append(d.Questions, ContentQuestion{ID: d.id("Q"), Question: text, Why: truncateRunes(strings.TrimSpace(q.Why), 200)})
				open++
			}
			it.Sources = snap.Sources
		}, nil
	})
}

// ContentMaterialsInput is the owner's edit of the fact cards and answers.
type ContentMaterialsInput struct {
	Facts   []ContentFact     `json:"facts"`
	Answers map[string]string `json:"answers"`
}

// SaveMaterials stores the owner's fact cards (kept, doubted, rejected, added) and turns answers
// into first-hand cards.
func (s *ContentEngineService) SaveMaterials(ctx context.Context, owner, id int64, in ContentMaterialsInput) (*ContentItem, error) {
	if len(in.Facts) > contentMaxFacts {
		return nil, ErrContentFacts
	}
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		d := it.Desk
		old := map[string]ContentFact{}
		for _, f := range d.Facts {
			old[f.ID] = f
		}
		facts := []ContentFact{}
		for _, f := range in.Facts {
			f.Text = strings.TrimSpace(f.Text)
			if f.Text == "" {
				continue
			}
			if utf8.RuneCountInString(f.Text) > contentMaxFactText {
				return ErrContentFacts
			}
			switch f.Status {
			case ContentFactNew, ContentFactOK, ContentFactDoubt, ContentFactRejected:
			default:
				return ErrContentFacts
			}
			if prev, ok := old[f.ID]; ok {
				// Sources stay as found; the owner edits the wording and the status.
				prev.Text, prev.Status = f.Text, f.Status
				facts = append(facts, prev)
				continue
			}
			facts = append(facts, ContentFact{ID: d.id("F"), Text: f.Text, Kind: ContentFactUser, Source: "作者", Status: ContentFactOK})
		}
		for qid, answer := range in.Answers {
			answer = strings.TrimSpace(answer)
			for i := range d.Questions {
				q := &d.Questions[i]
				if q.ID != qid || answer == "" || q.Answer == answer {
					continue
				}
				if utf8.RuneCountInString(answer) > 2000 {
					return ErrContentFacts
				}
				q.Answer = answer
				facts = append(facts, ContentFact{ID: d.id("F"), Text: truncateRunes("问「"+truncateRunes(q.Question, 60)+"」，作者答："+answer, contentMaxFactText+80), Kind: ContentFactAnswer, Source: "作者", Status: ContentFactOK})
			}
		}
		if len(facts) > contentMaxFacts {
			return ErrContentFacts
		}
		d.Facts = facts
		return nil
	})
}

// --- angles & reference ------------------------------------------------------------------------

func (s *ContentEngineService) Angles(ctx context.Context, owner, id int64) (*ContentItem, error) {
	return s.startDeskJob(ctx, owner, id, "angles", func(d *ContentDesk) bool { return d.usableFacts() > 0 },
		func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
			var out struct {
				Angles []ContentAngle `json:"angles"`
			}
			if err := s.callJSON(ctx, snap, r, contentAnglesPrompt(r.persona), contentAnglesRequest(snap), 5000, nil, 1, &out); err != nil {
				return nil, err
			}
			if len(out.Angles) == 0 {
				return nil, errLearnRunFailed("模型没有给出角度")
			}
			return func(it *ContentItem) {
				d := it.Desk
				d.Angles = []ContentAngle{}
				for _, a := range out.Angles {
					if len(d.Angles) >= 6 || strings.TrimSpace(a.Angle) == "" {
						continue
					}
					a.ID = d.id("A")
					a.Title, a.Angle, a.Archetype = truncateRunes(strings.TrimSpace(a.Title), 60), truncateRunes(strings.TrimSpace(a.Angle), 300), truncateRunes(strings.TrimSpace(a.Archetype), 20)
					a.Reader, a.Hook, a.Why = truncateRunes(strings.TrimSpace(a.Reader), 100), truncateRunes(strings.TrimSpace(a.Hook), 600), truncateRunes(strings.TrimSpace(a.Why), 300)
					a.H, a.K, a.R = max(1, min(a.H, 5)), max(1, min(a.K, 5)), max(1, min(a.R, 5))
					a.Note = ""
					d.Angles = append(d.Angles, a)
				}
				if d.Stage == ContentStageMaterials {
					d.Stage = ContentStageAngles
				}
			}, nil
		})
}

// ChooseAngle stores the owner's pick (possibly edited) and moves on to the drafts.
func (s *ContentEngineService) ChooseAngle(ctx context.Context, owner, id int64, in ContentAngle) (*ContentItem, error) {
	in.Title, in.Angle = strings.TrimSpace(in.Title), strings.TrimSpace(in.Angle)
	if in.Title == "" || in.Angle == "" {
		return nil, ErrContentAngle
	}
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		d := it.Desk
		if d.running() {
			return ErrContentDeskStage
		}
		a := ContentAngle{ID: in.ID, Title: truncateRunes(in.Title, 60), Angle: truncateRunes(in.Angle, 300), Archetype: truncateRunes(strings.TrimSpace(in.Archetype), 20),
			Reader: truncateRunes(strings.TrimSpace(in.Reader), 100), Hook: truncateRunes(strings.TrimSpace(in.Hook), 600), Note: truncateRunes(strings.TrimSpace(in.Note), 500),
			H: in.H, K: in.K, R: in.R, Why: truncateRunes(in.Why, 300)}
		d.Angle = &a
		if d.Stage == ContentStageMaterials || d.Stage == ContentStageAngles {
			d.Stage = ContentStageDrafts
		}
		it.Title = a.Title
		return nil
	})
}

// Reference reads a hit article (pasted text or a 公众号 link) and keeps only its skeleton.
func (s *ContentEngineService) Reference(ctx context.Context, owner, id int64, url, text string) (*ContentItem, error) {
	url, text = strings.TrimSpace(url), strings.TrimSpace(text)
	source := "粘贴的文章"
	if url != "" {
		if s.fetcher == nil {
			return nil, ErrContentRef
		}
		a, err := s.fetcher.Fetch(ctx, url)
		if err != nil {
			return nil, err
		}
		text, source = a.Text, a.Title
	}
	if utf8.RuneCountInString(text) < 300 {
		return nil, ErrContentRef
	}
	text = truncateRunes(text, contentMaxRefText)
	return s.startDeskJob(ctx, owner, id, "reference", nil, func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
		var sk ContentSkeleton
		if err := s.callJSON(ctx, snap, r, contentSkeletonPrompt, "参考文章：\n\n"+text, 3000, nil, 1, &sk); err != nil {
			return nil, err
		}
		sk.Source = truncateRunes(source, 100)
		return func(it *ContentItem) { it.Desk.Skeleton = &sk }, nil
	})
}

func (s *ContentEngineService) ClearReference(ctx context.Context, owner, id int64) (*ContentItem, error) {
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error { it.Desk.Skeleton = nil; return nil })
}

// --- drafts ------------------------------------------------------------------------------------

// Drafts writes the three drafts, has a reader go through each and the judge rank them.
func (s *ContentEngineService) Drafts(ctx context.Context, owner, id int64) (*ContentItem, error) {
	return s.startDeskJob(ctx, owner, id, "drafts", func(d *ContentDesk) bool { return d.Angle != nil },
		func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
			drafts := make([]ContentDraft, len(contentStructures))
			req := contentDraftDeskRequest(snap)
			g := new(errgroup.Group)
			g.SetLimit(3)
			for i, st := range contentStructures {
				drafts[i] = ContentDraft{Structure: st.ID, Label: st.Label}
				g.Go(func() error {
					text, err := s.call(ctx, snap, r, []AgentMessage{agentText("system", contentDraftDeskPrompt(r.persona, r.product, r.rules, st.How)), agentText("user", req)},
						contentDeskDraftTokens, nil, 1)
					if err != nil {
						if ctx.Err() != nil {
							return ctx.Err()
						}
						drafts[i].Error = contentErrorText(snap.ID, err)
						return nil
					}
					drafts[i].Markdown = contentCleanMarkdown(text)
					review, err := s.readerReview(ctx, snap, r, drafts[i].Markdown)
					if err == nil {
						drafts[i].Review = review
					} else if ctx.Err() != nil {
						return ctx.Err()
					}
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return nil, err
			}
			ok := 0
			for _, d := range drafts {
				if d.Markdown != "" {
					ok++
				}
			}
			if ok == 0 {
				return nil, errLearnRunFailed("三份初稿都没写成，再试一次")
			}
			// IDs are assigned on the fresh article in apply; the judge works on letters.
			judge := s.judgeDrafts(ctx, snap, r, drafts)
			return func(it *ContentItem) {
				d := it.Desk
				letters := map[string]string{}
				for i := range drafts {
					drafts[i].ID = d.id("D")
					letters[string(rune('A'+i))] = drafts[i].ID
				}
				d.Drafts = drafts
				d.Judge = nil
				if judge != nil {
					j := &ContentJudge{Reasons: judge.Reasons}
					for _, l := range judge.Ranking {
						if id, ok := letters[l]; ok {
							j.Ranking = append(j.Ranking, id)
						}
					}
					for _, h := range judge.Highlights {
						if id, ok := letters[h.DraftID]; ok {
							h.DraftID, h.ID = id, d.id("H")
							j.Highlights = append(j.Highlights, h)
						}
					}
					d.Judge = j
				}
				d.BaseID, d.Picked = "", nil
				if d.Stage == ContentStageDrafts || d.Stage == ContentStageAngles || d.Stage == ContentStageMaterials {
					d.Stage = ContentStageDrafts
				}
			}, nil
		})
}

// contentParas splits Markdown into paragraphs (blank-line separated, code blocks kept whole).
func contentParas(md string) []string {
	var out []string
	var cur []string
	inCode := false
	flush := func() {
		if p := strings.TrimSpace(strings.Join(cur, "\n")); p != "" {
			out = append(out, p)
		}
		cur = nil
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
		}
		if !inCode && strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		cur = append(cur, line)
	}
	flush()
	return out
}

func (s *ContentEngineService) readerReview(ctx context.Context, snap *ContentItem, r *contentRun, md string) (*ContentDraftReview, error) {
	paras := contentParas(md)
	var rv ContentDraftReview
	if err := s.callJSON(ctx, snap, r, contentReaderPrompt, numberedParas(paras), 4000, nil, 1, &rv); err != nil {
		return nil, err
	}
	marks := []ContentParaMark{}
	for _, m := range rv.Paras {
		if m.I < 1 || m.I > len(paras) {
			continue
		}
		switch m.V {
		case "read", "skim", "leave":
		default:
			m.V = "read"
		}
		m.Why = truncateRunes(strings.TrimSpace(m.Why), 200)
		marks = append(marks, m)
	}
	rv.Paras = marks
	for _, p := range []*int{&rv.Hook, &rv.Flow, &rv.Voice, &rv.Substance} {
		*p = max(0, min(*p, 5))
	}
	rv.BestLine, rv.Summary = truncateRunes(rv.BestLine, 200), truncateRunes(rv.Summary, 200)
	if len(rv.Problems) > 3 {
		rv.Problems = rv.Problems[:3]
	}
	return &rv, nil
}

type contentJudgeRaw struct {
	Ranking    []string           `json:"ranking"`
	Reasons    []string           `json:"-"`
	Reason     string             `json:"reason"`
	Highlights []ContentHighlight `json:"highlights"`
}

// judgeDrafts ranks the drafts twice, in two orders, and adds up the places (Borda), which takes
// out most of a judge's preference for whatever comes first. Drafts are lettered A, B, C.
func (s *ContentEngineService) judgeDrafts(ctx context.Context, snap *ContentItem, r *contentRun, drafts []ContentDraft) *contentJudgeRaw {
	var letters []string
	for i, d := range drafts {
		if d.Markdown != "" {
			letters = append(letters, string(rune('A'+i)))
		}
	}
	if len(letters) < 2 {
		if len(letters) == 1 {
			return &contentJudgeRaw{Ranking: letters}
		}
		return nil
	}
	orders := [][]string{letters, reversed(letters)}
	points := map[string]int{}
	out := &contentJudgeRaw{}
	seenQuote := map[string]bool{}
	for round := 0; round < contentJudgeRounds && round < len(orders); round++ {
		// Present the drafts as 稿1, 稿2… in this round's order, then map back to letters.
		var b strings.Builder
		_, _ = b.WriteString(contentFactsBlock(snap.Desk) + "\n\n")
		for i, l := range orders[round] {
			fmt.Fprintf(&b, "===== 稿%d =====\n%s\n\n", i+1, drafts[int(l[0]-'A')].Markdown)
		}
		var raw contentJudgeRaw
		if err := s.callJSON(ctx, snap, r, r.persona+"\n\n"+contentJudgePrompt, b.String(), 3000, nil, 1, &raw); err != nil {
			continue
		}
		toLetter := func(v string) string {
			v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "稿"))
			for i, l := range orders[round] {
				if v == fmt.Sprint(i+1) {
					return l
				}
			}
			return ""
		}
		for rank, v := range raw.Ranking {
			if l := toLetter(v); l != "" {
				points[l] += len(letters) - rank
			}
		}
		if raw.Reason != "" {
			out.Reasons = append(out.Reasons, truncateRunes(raw.Reason, 400))
		}
		for _, h := range raw.Highlights {
			l := toLetter(h.DraftID)
			q := strings.TrimSpace(h.Quote)
			if l == "" || q == "" || seenQuote[q] || !strings.Contains(drafts[int(l[0]-'A')].Markdown, q) {
				continue
			}
			seenQuote[q] = true
			out.Highlights = append(out.Highlights, ContentHighlight{DraftID: l, Quote: truncateRunes(q, 300), Why: truncateRunes(strings.TrimSpace(h.Why), 200)})
		}
	}
	if len(points) == 0 {
		return nil
	}
	out.Ranking = append([]string(nil), letters...)
	sort.SliceStable(out.Ranking, func(i, j int) bool { return points[out.Ranking[i]] > points[out.Ranking[j]] })
	// Highlights of the winner are not "to merge".
	kept := out.Highlights[:0]
	for _, h := range out.Highlights {
		if h.DraftID != out.Ranking[0] {
			kept = append(kept, h)
		}
	}
	out.Highlights = kept
	return out
}

func reversed(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

// ChooseBase picks the base draft; with highlights picked, they are merged in (background),
// otherwise the base becomes the text to edit right away.
func (s *ContentEngineService) ChooseBase(ctx context.Context, owner, id int64, baseID string, picked []string) (*ContentItem, error) {
	it, err := s.deskItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	var base *ContentDraft
	for i := range it.Desk.Drafts {
		if it.Desk.Drafts[i].ID == baseID && it.Desk.Drafts[i].Markdown != "" {
			base = &it.Desk.Drafts[i]
		}
	}
	if base == nil {
		return nil, ErrContentDraftPick
	}
	var highlights []ContentHighlight
	if it.Desk.Judge != nil {
		want := map[string]bool{}
		for _, p := range picked {
			want[p] = true
		}
		for _, h := range it.Desk.Judge.Highlights {
			if want[h.ID] {
				highlights = append(highlights, h)
			}
		}
	}
	baseMD := base.Markdown
	if len(highlights) == 0 {
		it, err = s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
			if it.Desk.running() {
				return ErrContentDeskStage
			}
			d := it.Desk
			d.BaseID, d.Picked, d.Body, d.BodyAI, d.Stage = baseID, nil, baseMD, baseMD, ContentStageEdit
			return nil
		})
		if err == nil {
			s.snapshot(ctx, it, "draft", "选定底稿："+base.Label+"稿")
		}
		return it, err
	}
	return s.startDeskJob(ctx, owner, id, "merge", nil, func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
		var b strings.Builder
		_, _ = b.WriteString("底稿：\n\n" + baseMD + "\n\n要融合进来的亮点：")
		for _, h := range highlights {
			fmt.Fprintf(&b, "\n- 「%s」（好在：%s）", h.Quote, h.Why)
		}
		_, _ = b.WriteString("\n\n" + contentFactsBlock(snap.Desk))
		text, err := s.call(ctx, snap, r, []AgentMessage{agentText("system", contentMergePrompt(r.persona, r.rules)), agentText("user", b.String())}, contentDeskDraftTokens, nil, 1)
		if err != nil {
			return nil, err
		}
		merged := contentCleanMarkdown(text)
		// A merge folds highlights in; coming back at half the base's length means it cut it down.
		if runeLen(merged) < runeLen(baseMD)/2 {
			return nil, errLearnRunFailed("融合后的稿子太短，再试一次")
		}
		return func(it *ContentItem) {
			d := it.Desk
			d.BaseID, d.Picked, d.Body, d.BodyAI, d.Stage = baseID, picked, merged, merged, ContentStageEdit
			s.snapshotLocked(it, "merge", fmt.Sprintf("底稿融合了 %d 处亮点", len(highlights)))
		}, nil
	})
}

// --- editing -----------------------------------------------------------------------------------

func (s *ContentEngineService) snapshot(ctx context.Context, it *ContentItem, kind, label string) {
	if it == nil || it.Desk == nil || strings.TrimSpace(it.Desk.Body) == "" {
		return
	}
	v := &ContentVersion{ItemID: it.ID, OwnerUserID: it.OwnerUserID, Kind: kind, Label: truncateRunes(label, 100), Markdown: it.Desk.Body}
	if err := s.repo.CreateVersion(ctx, v); err != nil {
		slog.Warn("content desk: snapshot", "item", it.ID, "err", err)
	}
}

// snapshotLocked is snapshot from inside an apply function (a background context).
func (s *ContentEngineService) snapshotLocked(it *ContentItem, kind, label string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s.snapshot(ctx, it, kind, label)
}

// SaveBody stores the text being edited; with label set it is also kept as a version.
func (s *ContentEngineService) SaveBody(ctx context.Context, owner, id int64, body, label string) (*ContentItem, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > 30000 {
		return nil, ErrContentBody
	}
	it, err := s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		d := it.Desk
		if d.Stage != ContentStageEdit && d.Stage != ContentStageFinal {
			return ErrContentDeskStage
		}
		d.Body = body
		if m := contentH1.FindStringSubmatch(body); m != nil {
			it.Title = truncateRunes(strings.TrimSpace(m[1]), 200)
		}
		return nil
	})
	if err == nil && strings.TrimSpace(label) != "" {
		s.snapshot(ctx, it, "manual", label)
	}
	return it, err
}

func (s *ContentEngineService) Versions(ctx context.Context, owner, id int64) ([]ContentVersion, error) {
	if _, err := s.deskItem(ctx, owner, id); err != nil {
		return nil, err
	}
	return s.repo.ListVersions(ctx, owner, id)
}

func (s *ContentEngineService) Version(ctx context.Context, owner, id, vid int64) (*ContentVersion, error) {
	v, err := s.repo.GetVersion(ctx, owner, vid)
	if err != nil {
		return nil, err
	}
	if v == nil || v.ItemID != id {
		return nil, ErrContentVersion
	}
	return v, nil
}

// RestoreVersion makes a version the current text again (the current text is kept as a version
// first).
func (s *ContentEngineService) RestoreVersion(ctx context.Context, owner, id, vid int64) (*ContentItem, error) {
	v, err := s.Version(ctx, owner, id, vid)
	if err != nil {
		return nil, err
	}
	cur, err := s.deskItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	s.snapshot(ctx, cur, "restore", "恢复前的版本")
	return s.SaveBody(ctx, owner, id, v.Markdown, "")
}

// SaveComments replaces the article's comments (批注).
func (s *ContentEngineService) SaveComments(ctx context.Context, owner, id int64, comments []ContentComment) (*ContentItem, error) {
	if len(comments) > contentMaxComments {
		return nil, ErrContentBody
	}
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		out := []ContentComment{}
		for _, c := range comments {
			c.Text, c.Quote = truncateRunes(strings.TrimSpace(c.Text), 1000), truncateRunes(strings.TrimSpace(c.Quote), 300)
			if c.Text == "" {
				continue
			}
			if c.ID == "" {
				c.ID, c.At = it.Desk.id("C"), s.now()
			}
			out = append(out, c)
		}
		it.Desk.Comments = out
		return nil
	})
}

// Assist rewrites the selected text (or offers analogies / evidence) synchronously.
func (s *ContentEngineService) Assist(ctx context.Context, owner, id int64, in ContentAssistInput) (*ContentAssistResult, error) {
	in.Selection = strings.TrimSpace(in.Selection)
	if in.Selection == "" || utf8.RuneCountInString(in.Selection) > contentMaxSelection {
		return nil, ErrContentAssist
	}
	it, err := s.deskItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	r, err := s.deskRun(ctx, owner, it.KeyID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, contentAssistTimeout)
	defer cancel()
	excerpt := fmt.Sprintf("前文：%s\n\n【选中的文字】%s\n\n后文：%s", truncateRunes(in.Before, 800), in.Selection, truncateRunes(in.After, 800))
	scratch := &ContentItem{OwnerUserID: owner, ContentItemData: ContentItemData{Sources: []ArticleSource{}, Desk: it.Desk}}
	var out ContentAssistResult
	switch in.Action {
	case "analogy":
		err = s.callJSON(ctx, scratch, r, r.persona+"\n\n"+contentAnalogyPrompt, excerpt, 2000, nil, 1, &out)
	case "evidence":
		if !s.searchAvailable(ctx) {
			return nil, infraerrors.ServiceUnavailable("CONTENT_SEARCH_OFF", "联网搜索没有配置（后台 → 设置 → 联网搜索），找不了证据（Web search unavailable）")
		}
		err = s.callJSON(ctx, scratch, r, contentEvidencePrompt, excerpt, 2000, []AgentTool{s.searchTool(scratch, true)}, 4, &out)
		seen := contentSourceSet(scratch.Sources)
		kept := out.Options[:0]
		for _, o := range out.Options {
			if seen[contentURLKey(o.Source)] {
				kept = append(kept, o)
			}
		}
		out.Options = kept
	default:
		task, ok := contentAssistActions[in.Action]
		if in.Action == "custom" {
			task, ok = strings.TrimSpace(in.Instruction), strings.TrimSpace(in.Instruction) != ""
		}
		if !ok {
			return nil, ErrContentAssist
		}
		err = s.callJSON(ctx, scratch, r, contentAssistPrompt(r.persona, r.rules),
			"任务："+truncateRunes(task, 500)+"\n\n"+contentFactsBlock(it.Desk)+"\n\n"+excerpt, 4000, nil, 1, &out)
	}
	if err != nil {
		return nil, err
	}
	out.Text = strings.TrimSpace(out.Text)
	if len(out.Options) > 5 {
		out.Options = out.Options[:5]
	}
	bg, bgCancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer bgCancel()
	if _, uerr := s.updateDesk(bg, owner, id, func(cur *ContentItem) error {
		cur.PromptTokens += scratch.PromptTokens
		cur.CompletionTokens += scratch.CompletionTokens
		return nil
	}); uerr != nil {
		slog.Warn("content desk: counting assist tokens", "item", id, "err", uerr)
	}
	return &out, nil
}

// contentURLKey compares URLs loosely (scheme and trailing slash ignored), so a source the model
// copies from a search result still matches it.
func contentURLKey(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	return strings.TrimRight(u, "/")
}

func contentSourceSet(sources []ArticleSource) map[string]bool {
	out := map[string]bool{}
	for _, s := range sources {
		out[contentURLKey(s.URL)] = true
	}
	return out
}

// cloneContentItem deep-copies an article (a background step works on its own copy).
func cloneContentItem(it *ContentItem) *ContentItem {
	cp := contentCloneJSON(*it)
	cp.OwnerUserID, cp.Running = it.OwnerUserID, it.Running
	return &cp
}

// AddFact adds a card from the evidence finder (or typed in) while editing.
func (s *ContentEngineService) AddFact(ctx context.Context, owner, id int64, f ContentFact) (*ContentItem, error) {
	f.Text = strings.TrimSpace(f.Text)
	if f.Text == "" || utf8.RuneCountInString(f.Text) > contentMaxFactText {
		return nil, ErrContentFacts
	}
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		d := it.Desk
		if len(d.Facts) >= contentMaxFacts {
			return ErrContentFacts
		}
		fact := ContentFact{ID: d.id("F"), Text: f.Text, Kind: ContentFactUser, Source: "作者", Status: ContentFactOK}
		if strings.HasPrefix(f.Source, "http") {
			fact.Kind, fact.Source, fact.SourceTitle = ContentFactWeb, truncateRunes(f.Source, 500), truncateRunes(f.SourceTitle, 120)
		}
		d.Facts = append(d.Facts, fact)
		return nil
	})
}

// Check runs the fixed rules and an editor's read of the current text.
func (s *ContentEngineService) Check(ctx context.Context, owner, id int64) (*ContentItem, error) {
	return s.startDeskJob(ctx, owner, id, "check", func(d *ContentDesk) bool { return strings.TrimSpace(d.Body) != "" },
		func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
			body := snap.Desk.Body
			var taboos []string
			if r.personaData != nil {
				taboos = r.personaData.Taboos
			}
			stats, hits := checkProse(body, taboos)
			report := &ContentCheckReport{At: s.now(), Stats: stats, Issues: hits}
			paras := contentParas(body)
			var raw struct {
				Issues  []ContentIssue `json:"issues"`
				Scores  map[string]int `json:"scores"`
				Summary string         `json:"summary"`
			}
			req := contentFactsBlock(snap.Desk) + "\n\n文章：\n\n" + numberedParas(paras)
			if err := s.callJSON(ctx, snap, r, contentReviewDeskPrompt(r.persona), req, 6000, nil, 1, &raw); err != nil {
				return nil, err
			}
			for _, is := range raw.Issues {
				is.Quote = strings.TrimSpace(is.Quote)
				if is.Quote == "" || !strings.Contains(body, is.Quote) || len(report.Issues) >= 40 {
					continue
				}
				switch is.Layer {
				case "voice", "flow", "content", "fact":
				default:
					is.Layer = "content"
				}
				if is.Level != "must" {
					is.Level = "should"
				}
				is.Problem, is.Fix = truncateRunes(strings.TrimSpace(is.Problem), 300), truncateRunes(strings.TrimSpace(is.Fix), 1000)
				report.Issues = append(report.Issues, is)
			}
			report.Scores, report.Summary = raw.Scores, truncateRunes(raw.Summary, 300)
			return func(it *ContentItem) {
				for i := range report.Issues {
					report.Issues[i].ID = it.Desk.id("I")
				}
				it.Desk.Check = report
			}, nil
		})
}

// Finalize marks the text final (no placeholders left), keeps it as a version and has the
// owner's edits turned into proposed writing rules.
func (s *ContentEngineService) Finalize(ctx context.Context, owner, id int64) (*ContentItem, error) {
	it, err := s.deskItem(ctx, owner, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(it.Desk.Body) == "" {
		return nil, ErrContentDeskStage
	}
	if strings.Contains(it.Desk.Body, "【这里需要你的真实经历") {
		return nil, ErrContentPlaceholder
	}
	it, err = s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		now := s.now()
		it.Desk.FinalAt, it.Desk.Stage = &now, ContentStageFinal
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.snapshot(ctx, it, "final", "定稿")
	before, after := it.Desk.BodyAI, it.Desk.Body
	if before == "" || before == after {
		return it, nil
	}
	return s.startDeskJob(ctx, owner, id, "learn", nil, func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
		diff := contentParaDiff(before, after, 30)
		if diff == "" {
			return nil, nil
		}
		var out struct {
			Rules []struct {
				Rule     string `json:"rule"`
				Evidence string `json:"evidence"`
			} `json:"rules"`
		}
		if err := s.callJSON(ctx, snap, r, contentLearnPrompt, diff, 3000, nil, 1, &out); err != nil {
			return nil, err
		}
		return func(it *ContentItem) {
			it.Desk.Learn = []ContentLearn{}
			for _, x := range out.Rules {
				rule := truncateRunes(strings.TrimSpace(x.Rule), contentMaxRule)
				if rule == "" || len(it.Desk.Learn) >= 6 {
					continue
				}
				it.Desk.Learn = append(it.Desk.Learn, ContentLearn{ID: it.Desk.id("L"), Rule: rule, Evidence: truncateRunes(strings.TrimSpace(x.Evidence), 300), Status: "new"})
			}
		}, nil
	})
}

// contentParaDiff lists the paragraphs of the AI text that the owner changed, with what they
// became (by best bigram match), up to limit pairs.
func contentParaDiff(before, after string, limit int) string {
	a, b := contentParas(before), contentParas(after)
	inAfter := map[string]bool{}
	for _, p := range b {
		inAfter[p] = true
	}
	var out strings.Builder
	n := 0
	for _, p := range a {
		if inAfter[p] || n >= limit {
			continue
		}
		best, bestSim := "", 0.0
		for _, q := range b {
			if sim := contentJaccard(p, q); sim > bestSim {
				best, bestSim = q, sim
			}
		}
		if bestSim < 0.2 {
			best = "（删掉了）"
		}
		n++
		fmt.Fprintf(&out, "【AI 原稿】%s\n【作者改成】%s\n\n", truncateRunes(p, 600), truncateRunes(best, 600))
	}
	return out.String()
}

// ApplyLearn adds a proposed rule to the writing rules, or dismisses it.
func (s *ContentEngineService) ApplyLearn(ctx context.Context, owner, id int64, learnID string, accept bool) (*ContentItem, error) {
	var rule string
	it, err := s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		for i := range it.Desk.Learn {
			l := &it.Desk.Learn[i]
			if l.ID != learnID || l.Status != "new" {
				continue
			}
			if accept {
				l.Status, rule = "added", l.Rule
			} else {
				l.Status = "dismissed"
			}
			return nil
		}
		return ErrContentNotFound
	})
	if err != nil {
		return nil, err
	}
	if rule != "" {
		if err := s.repo.CreateRule(ctx, &ContentRule{OwnerUserID: owner, Rule: rule, Origin: "learned", Enabled: true}); err != nil {
			return nil, err
		}
	}
	return it, nil
}

// Platforms writes the platform versions from the final text; the 公众号 version is the text
// itself. The article then goes to 待发布 review like any other.
func (s *ContentEngineService) Platforms(ctx context.Context, owner, id int64) (*ContentItem, error) {
	return s.startDeskJob(ctx, owner, id, "platforms", func(d *ContentDesk) bool { return d.FinalAt != nil },
		func(ctx context.Context, snap *ContentItem, r *contentRun) (func(*ContentItem), error) {
			body := snap.Desk.Body
			snap.Markdown = body
			if m := contentH1.FindStringSubmatch(body); m != nil {
				snap.Title = truncateRunes(strings.TrimSpace(m[1]), 200)
			}
			if err := s.writeDeskWechat(ctx, snap); err != nil {
				return nil, err
			}
			// Running it again after editing the text rewrites the versions not yet approved.
			vs, err := s.repo.ListVariants(ctx, snap.OwnerUserID, snap.ID)
			if err != nil {
				return nil, err
			}
			for i := range vs {
				if vs[i].Platform != ContentPlatformWechat && vs[i].Status == ContentVariantPending {
					vs[i].Status = ContentVariantFailed
					if err := s.repo.SaveVariant(ctx, &vs[i]); err != nil {
						return nil, err
					}
				}
			}
			if err := s.adaptAll(ctx, snap, r); err != nil {
				return nil, err
			}
			earlier, _ := s.repo.RecentTitles(ctx, snap.OwnerUserID, 80)
			checks := checkContentMaster(snap, earlier)
			return func(it *ContentItem) {
				it.Markdown, it.Title, it.Checks, it.Status = body, snap.Title, checks, ContentPending
			}, nil
		})
}

// writeDeskWechat makes the 公众号 version from the final text as it is (the author wrote it for
// the 公众号): title from the H1, body without it, digest from the first paragraph.
func (s *ContentEngineService) writeDeskWechat(ctx context.Context, it *ContentItem) error {
	if !contains(it.Brief.Platforms, ContentPlatformWechat) {
		return nil
	}
	existing, err := s.repo.ListVariants(ctx, it.OwnerUserID, it.ID)
	if err != nil {
		return err
	}
	for _, v := range existing {
		if v.Platform == ContentPlatformWechat && v.Status != ContentVariantFailed && v.Status != ContentVariantPending {
			return nil // already approved or published: leave it
		}
	}
	body := strings.TrimSpace(contentH1.ReplaceAllString(it.Markdown, ""))
	digest := ""
	for _, p := range contentParas(body) {
		if !strings.HasPrefix(p, "#") {
			digest = truncateRunes(contentPlainText(p), 110)
			break
		}
	}
	v := &ContentVariant{ItemID: it.ID, OwnerUserID: it.OwnerUserID, Platform: ContentPlatformWechat, Kind: "article", Status: ContentVariantPending,
		Title: it.Title, Body: body, ContentVariantData: ContentVariantData{Digest: digest}}
	return s.repo.UpsertVariant(ctx, v)
}

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}

// CancelDeskJob stops the article's running AI step.
func (s *ContentEngineService) CancelDeskJob(ctx context.Context, owner, id int64) error {
	if _, err := s.deskItem(ctx, owner, id); err != nil {
		return err
	}
	s.runsMu.Lock()
	cancel := s.runs[id]
	s.runsMu.Unlock()
	if cancel == nil {
		return ErrContentDeskStage
	}
	cancel()
	return nil
}

// SetStage lets the owner go back to an earlier step (the later work is kept).
func (s *ContentEngineService) SetStage(ctx context.Context, owner, id int64, stage string) (*ContentItem, error) {
	return s.updateDesk(ctx, owner, id, func(it *ContentItem) error {
		d := it.Desk
		switch stage {
		case ContentStageMaterials, ContentStageAngles:
		case ContentStageDrafts:
			if d.Angle == nil {
				return ErrContentDeskStage
			}
		case ContentStageEdit:
			if strings.TrimSpace(d.Body) == "" {
				return ErrContentDeskStage
			}
		case ContentStageFinal:
			if d.FinalAt == nil {
				return ErrContentDeskStage
			}
		default:
			return ErrContentDeskStage
		}
		d.Stage = stage
		return nil
	})
}

// contentCloneJSON deep-copies a value through JSON (fields tagged "-" are dropped).
func contentCloneJSON[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}

func randomContentID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
