package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 诗画素材库 (docs/MATERIAL_LIBRARY_PLAN.md): texts are split into segments, segments carry imagery,
// imagery and styles make the prompt, and every drawing records what went into it.
//
// Copyright comes first. A text is public_domain / original / licensed / reference_only, and only
// the first three may keep their words: popular lyrics and works of authors dead fewer than 50 years
// are reference_only, stored as metadata plus our own scene description. The database refuses
// original words for them (CHECK on mat_texts, trigger on mat_text_segments), so a bug here cannot
// leak lyrics into the library.

const (
	MaterialLicensePublicDomain = "public_domain"
	MaterialLicenseOriginal     = "original"
	MaterialLicenseLicensed     = "licensed"
	MaterialLicenseReference    = "reference_only"

	MaterialStatusDraft     = "draft"
	MaterialStatusReview    = "review"
	MaterialStatusPublished = "published"
	MaterialStatusHidden    = "hidden"

	MaterialBlockActionBlock = "block"
	MaterialBlockActionWarn  = "warn"
	MaterialBlockActionStrip = "strip"

	// China's term: the author's life plus 50 years, to 31 December of the 50th year.
	materialCopyrightYears = 50
	materialMaxTextsPage   = 200
)

var (
	ErrMaterialTextNotFound = infraerrors.NotFound("MATERIAL_TEXT_NOT_FOUND", "找不到这条文字素材（Text not found）")
	ErrMaterialInvalid      = infraerrors.BadRequest("MATERIAL_INVALID", "填写的内容不完整或格式不对（Invalid input）")
)

var materialTextKinds = map[string]bool{
	"shi": true, "ci": true, "qu": true, "fu": true, "wen": true, "modern_poem": true, "song": true, "original": true,
}

// materialAncientDynasties are periods whose authors all died long before the copyright term.
var materialAncientDynasties = map[string]bool{
	"先秦": true, "周": true, "春秋": true, "战国": true, "秦": true, "汉": true, "西汉": true, "东汉": true, "三国": true,
	"魏": true, "蜀": true, "吴": true, "晋": true, "西晋": true, "东晋": true, "南北朝": true, "南朝": true, "北朝": true,
	"齐": true, "梁": true, "陈": true, "北魏": true, "北齐": true, "北周": true, "隋": true, "唐": true,
	"五代": true, "五代十国": true, "宋": true, "北宋": true, "南宋": true, "辽": true, "金": true, "元": true, "明": true, "清": true,
}

type MaterialAuthor struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Aliases   []string  `json:"aliases"`
	Dynasty   string    `json:"dynasty"`
	BirthYear *int      `json:"birth_year,omitempty"`
	DeathYear *int      `json:"death_year,omitempty"`
	Bio       string    `json:"bio"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MaterialText struct {
	ID            int64           `json:"id"`
	Slug          string          `json:"slug"`
	Kind          string          `json:"kind"`
	Title         string          `json:"title"`
	AuthorID      *int64          `json:"author_id,omitempty"`
	Performer     string          `json:"performer"`
	Body          string          `json:"body"`
	Translation   string          `json:"translation"`
	Appreciation  string          `json:"appreciation"`
	LicenseStatus string          `json:"license_status"`
	LicenseNote   string          `json:"license_note"`
	Sources       string          `json:"sources"`
	Status        string          `json:"status"`
	Verify1By     *int64          `json:"verify1_by,omitempty"`
	Verify1At     *time.Time      `json:"verify1_at,omitempty"`
	Verify2By     *int64          `json:"verify2_by,omitempty"`
	Verify2At     *time.Time      `json:"verify2_at,omitempty"`
	ExternalURL   string          `json:"external_url"`
	Themes        []string        `json:"themes"`
	Seasons       []string        `json:"seasons"`
	Textbook      json.RawMessage `json:"textbook"`
	ShihuaWorkID  *int64          `json:"shihua_work_id,omitempty"`
	CreatedBy     *int64          `json:"created_by,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`

	// Joined from mat_authors.
	AuthorName string `json:"author_name"`
	Dynasty    string `json:"dynasty"`
	// Lists fill SegmentCount; GetText fills Segments.
	SegmentCount int               `json:"segment_count"`
	Segments     []MaterialSegment `json:"segments,omitempty"`
	// Computed from LicenseStatus: may the words go on a picture?
	TextOverlayAllowed bool `json:"text_overlay_allowed"`
}

type MaterialSegment struct {
	ID        int64  `json:"id"`
	TextID    int64  `json:"text_id"`
	Seq       int    `json:"seq"`
	Label     string `json:"label"`
	Content   string `json:"content"`
	SceneDesc string `json:"scene_desc"`
	IsFamous  bool   `json:"is_famous"`
}

type MaterialTextQuery struct {
	Kind    string
	Status  string
	License string
	Q       string
	Limit   int
	Offset  int
}

type MaterialStyle struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Category        string          `json:"category"`
	DescZh          string          `json:"desc_zh"`
	PromptEn        string          `json:"prompt_en"`
	NegativeEn      string          `json:"negative_en"`
	SuitsEmotions   []string        `json:"suits_emotions"`
	SuitsCategories []string        `json:"suits_categories"`
	Options         json.RawMessage `json:"options"`
	DefaultSize     string          `json:"default_size"`
	DefaultQuality  string          `json:"default_quality"`
	Enabled         bool            `json:"enabled"`
	Sort            int             `json:"sort"`
}

type MaterialBlockTerm struct {
	ID        int64  `json:"id"`
	Term      string `json:"term"`
	Kind      string `json:"kind"`
	Match     string `json:"match"`
	Action    string `json:"action"`
	AppliesTo string `json:"applies_to"`
	Reason    string `json:"reason"`
	Enabled   bool   `json:"enabled"`
}

// MaterialRepository is the 素材库 store.
type MaterialRepository interface {
	// UpsertAuthor finds the author by (name, dynasty) or creates it; returns the id.
	UpsertAuthor(ctx context.Context, a *MaterialAuthor) (int64, error)
	// ImportText inserts a text with its segments in one transaction. It returns false without
	// writing when a text with the same shihua_work_id or slug already exists.
	ImportText(ctx context.Context, t *MaterialText, segments []MaterialSegment) (bool, error)
	ListTexts(ctx context.Context, q MaterialTextQuery) ([]MaterialText, int, error)
	// GetText returns the text with its segments, nil when it does not exist.
	GetText(ctx context.Context, id int64) (*MaterialText, error)
	ListStyles(ctx context.Context, includeDisabled bool) ([]MaterialStyle, error)
	ListBlocklist(ctx context.Context, enabledOnly bool) ([]MaterialBlockTerm, error)

	ListImagery(ctx context.Context, q MaterialImageryQuery) ([]MaterialImagery, int, error)
	// AllImagery returns every entry (names, aliases, status) for matching.
	AllImagery(ctx context.Context) ([]MaterialImagery, error)
	// CreateImagery inserts an entry and sets its ID; false (ID 0) when the name is taken.
	CreateImagery(ctx context.Context, im *MaterialImagery) (bool, error)
	// UpdateImagery saves the editable fields; false when the entry does not exist.
	UpdateImagery(ctx context.Context, im *MaterialImagery) (bool, error)
	SetImageryStatus(ctx context.Context, ids []int64, status string) (int64, error)
	// GetSegment returns the segment with its text's metadata, nil when it does not exist.
	GetSegment(ctx context.Context, id int64) (*MaterialSegmentContext, error)
	UpdateSegmentScene(ctx context.Context, id int64, scene string) error
	ListSegmentImagery(ctx context.Context, segmentID int64) ([]MaterialSegmentImagery, error)
	// ReplaceAISegmentImagery drops the segment's unconfirmed AI tags and adds links (existing tags win).
	ReplaceAISegmentImagery(ctx context.Context, segmentID int64, links []MaterialSegmentImagery) error
	// SetSegmentImagery replaces all of the segment's tags.
	SetSegmentImagery(ctx context.Context, segmentID int64, links []MaterialSegmentImagery) error

	materialArtworkRepo
	materialRenderRepo
}

// MaterialService runs the 素材库 admin pages.
type MaterialService struct {
	repo   MaterialRepository
	shihua ShihuaRepository
	chat   materialChatFunc
	queue  *materialQueue // nil until EnableDrawing
	now    func() time.Time
	// lyrics refuses prompts that contain lyrics from the 灵感库 (nil = no check).
	lyrics *LyricsGuard
}

// SetLyricsGuard turns on the lyrics check for drawing prompts.
func (s *MaterialService) SetLyricsGuard(g *LyricsGuard) { s.lyrics = g }

// NewMaterialService: text-model calls use the key and model of the 课本诗画 settings (shihuaSvc may be
// nil in tests, then the AI features report that no model is configured).
func NewMaterialService(repo MaterialRepository, shihua ShihuaRepository, shihuaSvc *ShihuaService) *MaterialService {
	s := &MaterialService{repo: repo, shihua: shihua, now: time.Now}
	if shihuaSvc != nil {
		s.chat = shihuaSvc.AdminChat
	}
	return s
}

// ---- copyright ----

// SuggestMaterialLicense proposes a license for an author's work: ancient dynasties and authors dead
// for more than 50 full calendar years are public domain; anyone else is reference_only until an
// admin records an agreement. The reason is shown to the admin.
func SuggestMaterialLicense(dynasty string, deathYear *int, now time.Time) (string, string) {
	if materialAncientDynasties[strings.TrimSpace(dynasty)] {
		return MaterialLicensePublicDomain, "古代作品，已进入公有领域"
	}
	if deathYear == nil {
		return MaterialLicenseReference, "未填作者卒年，按仍受保护处理"
	}
	free := *deathYear + materialCopyrightYears + 1
	if now.Year() >= free {
		return MaterialLicensePublicDomain, fmt.Sprintf("作者 %d 年去世，%d 年起进入公有领域", *deathYear, free)
	}
	return MaterialLicenseReference, fmt.Sprintf("作者 %d 年去世，要到 %d 年 1 月 1 日才进入公有领域", *deathYear, free)
}

// MaterialTextOverlayAllowed tells whether a text's words may be set on a picture.
func MaterialTextOverlayAllowed(license string) bool {
	switch license {
	case MaterialLicensePublicDomain, MaterialLicenseOriginal, MaterialLicenseLicensed:
		return true
	}
	return false
}

// ---- segments ----

var materialParagraphBreak = regexp.MustCompile(`\n\s*\n`)

var materialSentenceEnd = map[rune]bool{'。': true, '！': true, '？': true, '；': true, '!': true, '?': true}
var materialClosers = map[rune]bool{'”': true, '’': true, '」': true, '』': true, '）': true, ')': true, '"': true}

// SplitMaterialSegments cuts a text into the parts pictures are drawn from: one sentence each (for 诗
// a sentence is a 联). Paragraphs are kept apart; a 词 with two paragraphs gets 上阕 / 下阕 labels.
// famous is the work's 名句 field; a segment overlapping one is marked famous.
func SplitMaterialSegments(kind, body, famous string) []MaterialSegment {
	body = strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\r", "\n")
	var paragraphs [][]string
	for _, para := range materialParagraphBreak.Split(body, -1) {
		var sentences []string
		var cur strings.Builder
		runes := []rune(strings.TrimSpace(para))
		for i := 0; i < len(runes); i++ {
			r := runes[i]
			if r == '\n' {
				continue
			}
			_, _ = cur.WriteRune(r)
			if materialSentenceEnd[r] {
				for i+1 < len(runes) && materialClosers[runes[i+1]] {
					i++
					_, _ = cur.WriteRune(runes[i])
				}
				if s := strings.TrimSpace(cur.String()); s != "" {
					sentences = append(sentences, s)
				}
				cur.Reset()
			}
		}
		if s := strings.TrimSpace(cur.String()); s != "" {
			sentences = append(sentences, s)
		}
		if len(sentences) > 0 {
			paragraphs = append(paragraphs, sentences)
		}
	}

	famousCores := []string{}
	for _, line := range strings.FieldsFunc(famous, func(r rune) bool { return r == '\n' || r == '|' || r == '／' }) {
		if c := materialCore(line); utf8.RuneCountInString(c) >= 4 {
			famousCores = append(famousCores, c)
		}
	}

	unit := "句"
	if kind == "shi" {
		unit = "联"
	}
	halves := kind == "ci" && len(paragraphs) == 2
	segments := []MaterialSegment{}
	for pi, sentences := range paragraphs {
		for si, s := range sentences {
			label := fmt.Sprintf("第%d%s", len(segments)+1, unit)
			if halves {
				label = []string{"上阕", "下阕"}[pi] + fmt.Sprintf("·第%d句", si+1)
			}
			core := materialCore(s)
			isFamous := false
			for _, f := range famousCores {
				if strings.Contains(core, f) || (utf8.RuneCountInString(core) >= 4 && strings.Contains(f, core)) {
					isFamous = true
					break
				}
			}
			segments = append(segments, MaterialSegment{Seq: len(segments) + 1, Label: label, Content: s, IsFamous: isFamous})
		}
	}
	return segments
}

// materialCore keeps letters only, so punctuation differences don't break matching.
func materialCore(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, s)
}

// ---- blocklist ----

type MaterialBlockHit struct {
	Term   string `json:"term"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
	Match  string `json:"match"`
	Reason string `json:"reason"`
}

type MaterialBlockResult struct {
	Blocked bool               `json:"blocked"`
	Hits    []MaterialBlockHit `json:"hits"`
	// Prompt is the input with "strip" terms removed.
	Prompt string `json:"prompt"`
}

var materialRegexCache sync.Map // term → *regexp.Regexp (nil value = invalid pattern)

func materialTermRegexp(t MaterialBlockTerm) *regexp.Regexp {
	if t.Match != "regex" {
		return regexp.MustCompile(`(?i)` + regexp.QuoteMeta(t.Term))
	}
	if v, ok := materialRegexCache.Load(t.Term); ok {
		re, _ := v.(*regexp.Regexp)
		return re
	}
	re, err := regexp.Compile(t.Term)
	if err != nil {
		re = nil
	}
	materialRegexCache.Store(t.Term, re)
	return re
}

// CheckMaterialBlocklist checks a prompt ("prompt") or a text ("text") against the enabled terms.
func CheckMaterialBlocklist(input, target string, terms []MaterialBlockTerm) MaterialBlockResult {
	res := MaterialBlockResult{Hits: []MaterialBlockHit{}, Prompt: input}
	for _, t := range terms {
		if !t.Enabled || (t.AppliesTo != "both" && t.AppliesTo != target) {
			continue
		}
		re := materialTermRegexp(t)
		if re == nil {
			continue
		}
		m := re.FindString(res.Prompt)
		if m == "" {
			continue
		}
		res.Hits = append(res.Hits, MaterialBlockHit{Term: t.Term, Kind: t.Kind, Action: t.Action, Match: m, Reason: t.Reason})
		switch t.Action {
		case MaterialBlockActionBlock:
			res.Blocked = true
		case MaterialBlockActionStrip:
			res.Prompt = strings.Join(strings.Fields(re.ReplaceAllString(res.Prompt, "")), " ")
		}
	}
	return res
}

// ---- service ----

func (s *MaterialService) ListTexts(ctx context.Context, q MaterialTextQuery) ([]MaterialText, int, error) {
	if q.Limit <= 0 || q.Limit > materialMaxTextsPage {
		q.Limit = 50
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	texts, total, err := s.repo.ListTexts(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	for i := range texts {
		texts[i].TextOverlayAllowed = MaterialTextOverlayAllowed(texts[i].LicenseStatus)
	}
	return texts, total, nil
}

func (s *MaterialService) GetText(ctx context.Context, id int64) (*MaterialText, error) {
	t, err := s.repo.GetText(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrMaterialTextNotFound
	}
	t.TextOverlayAllowed = MaterialTextOverlayAllowed(t.LicenseStatus)
	return t, nil
}

func (s *MaterialService) ListStyles(ctx context.Context) ([]MaterialStyle, error) {
	return s.repo.ListStyles(ctx, true)
}

func (s *MaterialService) ListBlocklist(ctx context.Context) ([]MaterialBlockTerm, error) {
	return s.repo.ListBlocklist(ctx, false)
}

// CheckPrompt runs a prompt or a text through the enabled blocklist.
func (s *MaterialService) CheckPrompt(ctx context.Context, input, target string) (MaterialBlockResult, error) {
	if target != "prompt" && target != "text" {
		return MaterialBlockResult{}, ErrMaterialInvalid
	}
	terms, err := s.repo.ListBlocklist(ctx, true)
	if err != nil {
		return MaterialBlockResult{}, err
	}
	return CheckMaterialBlocklist(input, target, terms), nil
}

type MaterialImportSkip struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

type MaterialImportResult struct {
	Imported int                  `json:"imported"`
	Existing int                  `json:"existing"`
	Skipped  []MaterialImportSkip `json:"skipped"`
}

// ImportShihua copies the 课本诗画 works whose text passed both checks into the library. Works
// already imported are left alone, so it can be run again after new works are verified.
func (s *MaterialService) ImportShihua(ctx context.Context, adminID int64) (*MaterialImportResult, error) {
	works, err := s.shihua.ListWorks(ctx, ShihuaWorkQuery{Status: ShihuaTextVerified2})
	if err != nil {
		return nil, err
	}
	res := &MaterialImportResult{Skipped: []MaterialImportSkip{}}
	for i := range works {
		w := &works[i]
		skip := func(reason string) {
			res.Skipped = append(res.Skipped, MaterialImportSkip{Slug: w.Slug, Title: w.Title, Reason: reason})
		}
		if !materialTextKinds[w.Genre] {
			skip("体裁不在素材库范围内：" + w.Genre)
			continue
		}
		license, reason := SuggestMaterialLicense(w.Dynasty, nil, s.now())
		if license != MaterialLicensePublicDomain {
			skip(reason + "，请先在作者库填卒年再导入")
			continue
		}
		segments := SplitMaterialSegments(w.Genre, w.FullText, w.FamousLines)
		if len(segments) == 0 {
			skip("正文为空")
			continue
		}
		authorID, err := s.repo.UpsertAuthor(ctx, &MaterialAuthor{Name: strings.TrimSpace(w.Author), Dynasty: strings.TrimSpace(w.Dynasty), Kind: "poet"})
		if err != nil {
			return nil, err
		}
		textbook, _ := json.Marshal(w.Textbooks)
		sources := strings.TrimSpace(w.Sources)
		if sources != "" {
			sources += "\n"
		}
		sources += fmt.Sprintf("导入自课本诗画 #%d（两次校对已通过）", w.ID)
		workID, by := w.ID, adminID
		t := &MaterialText{
			Slug: w.Slug, Kind: w.Genre, Title: w.Title, AuthorID: &authorID, Body: w.FullText,
			LicenseStatus: license, LicenseNote: reason, Sources: sources, Status: MaterialStatusPublished,
			Verify1By: w.Verify1By, Verify1At: w.Verify1At, Verify2By: w.Verify2By, Verify2At: w.Verify2At,
			Themes: []string{}, Seasons: []string{}, Textbook: textbook, ShihuaWorkID: &workID, CreatedBy: &by,
		}
		created, err := s.repo.ImportText(ctx, t, segments)
		if err != nil {
			return nil, err
		}
		if created {
			res.Imported++
		} else {
			res.Existing++
		}
	}
	return res, nil
}
