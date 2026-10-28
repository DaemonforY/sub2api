package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 灵感库 (admin /admin/inspiration): where the admin looks for ideas before making 课本诗画 / 素材库
// works.
//
//   - Poems: ~720k public-domain works (chinese-poetry + Werneror/Poetry, loaded by tools/poetry_corpus)
//     to browse, search by words, or find by a described scene (the text model only turns the scene
//     into search words and picks among real corpus hits, so no poem is written from memory). A poem
//     can be copied into 课本诗画 as a draft, which still goes through both checks.
//   - Songs: cards with the lyrics the admin pastes in, for internal reference and search only. The
//     lyrics are served by the admin API alone and never exported or put into prompts; LyricsGuard
//     refuses work texts and prompts that contain a lyric line.

var (
	ErrLibPoemNotFound = infraerrors.NotFound("LIB_POEM_NOT_FOUND", "找不到这首诗（Poem not found）")
	ErrLibSongNotFound = infraerrors.NotFound("LIB_SONG_NOT_FOUND", "找不到这首歌（Song not found）")
	ErrLibInvalid      = infraerrors.BadRequest("LIB_INVALID", "填写的内容不完整或格式不对（Invalid input）")
	ErrLibNoModel      = infraerrors.BadRequest("LIB_NO_MODEL", "还没有配置文本模型：请在「课本诗画 → 设置」里选一把 API Key（No text model configured）")
	ErrLibModelFailed  = infraerrors.BadRequest("LIB_MODEL_FAILED", "文本模型没有返回可用的结果，请重试（The text model returned nothing usable）")
)

const (
	libMaxQueryTerms  = 8
	libMaxFindWords   = 16
	libFindCandidates = 60
	libRerankPoems    = 30
	libMaxLyricsRunes = 20000
	libModelTimeout   = 4 * time.Minute
)

// LibCollections names the corpus collections (keys of lib_poems.collection).
var LibCollections = map[string]string{
	"tang": "全唐诗", "song_shi": "全宋诗", "song_ci": "全宋词", "yuanqu": "元曲", "shijing": "诗经", "chuci": "楚辞",
	"huajian": "花间集", "nantang": "南唐二主词", "caocao": "曹操诗集", "nalan": "纳兰性德", "guwen": "古文观止",
	"lunyu": "论语", "lidai": "历代诗词",
}

type LibPoem struct {
	ID         int64    `json:"id"`
	Collection string   `json:"collection"`
	Title      string   `json:"title"`
	Author     string   `json:"author"`
	Dynasty    string   `json:"dynasty"`
	Content    string   `json:"content"`
	Notes      string   `json:"notes,omitempty"`
	Tags       []string `json:"tags"`
	Famous     bool     `json:"famous"`
	Popularity float64  `json:"popularity"`
	Starred    bool     `json:"starred"`
	// Score is how many of the find words the poem contains (find only).
	Score int `json:"score,omitempty"`
	// InShihua is the slug of the 课本诗画 work with the same title and author (detail only).
	InShihua string `json:"in_shihua,omitempty"`
}

type LibPoemQuery struct {
	Q          string // words, all required (title, author or text)
	Dynasty    string
	Collection string
	Author     string
	Famous     bool
	Starred    bool
	Random     bool
	Limit      int
	Offset     int
}

type LibAllusion struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	PoemID *int64 `json:"poem_id,omitempty"`
	Note   string `json:"note"`
}

type LibSong struct {
	ID        int64         `json:"id"`
	Title     string        `json:"title"`
	Artist    string        `json:"artist"`
	Lyricist  string        `json:"lyricist"`
	Composer  string        `json:"composer"`
	Year      *int          `json:"year,omitempty"`
	Album     string        `json:"album"`
	Lyrics    string        `json:"lyrics,omitempty"` // detail only
	HasLyrics bool          `json:"has_lyrics"`
	Summary   string        `json:"summary"`
	Imagery   []string      `json:"imagery"`
	Moods     []string      `json:"moods"`
	Tags      []string      `json:"tags"`
	Allusions []LibAllusion `json:"allusions"`
	Notes     string        `json:"notes"`
	LinkURL   string        `json:"link_url"`
	Starred   bool          `json:"starred"`
	CreatedBy *int64        `json:"created_by,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	// Matches are the lyric lines a search hit (lists only).
	Matches []string `json:"matches,omitempty"`
}

type LibSongQuery struct {
	Q        string
	Artist   string
	Tag      string
	YearFrom int
	YearTo   int
	Starred  bool
	Limit    int
	Offset   int
}

type LibFacet struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type LibSongFacets struct {
	Artists []LibFacet `json:"artists"`
	Tags    []LibFacet `json:"tags"`
	Years   []int      `json:"years"` // min, max ([] when no song has a year)
}

type InspirationRepository interface {
	// SearchPoems returns a page and whether there are more.
	SearchPoems(ctx context.Context, q LibPoemQuery) ([]LibPoem, bool, error)
	GetPoem(ctx context.Context, id int64) (*LibPoem, error)
	StarPoem(ctx context.Context, id int64, starred bool) error
	PoemCounts(ctx context.Context) (map[string]int, error)
	// FindPoems returns poems containing any of the words, the most words first.
	FindPoems(ctx context.Context, words []string, q LibPoemQuery, limit int) ([]LibPoem, error)
	FindPoemByTitle(ctx context.Context, title, author string) (*LibPoem, error)
	PoemsContain(ctx context.Context, han string) (bool, error)

	ListSongs(ctx context.Context, q LibSongQuery) ([]LibSong, int, error) // with lyrics, for matching
	GetSong(ctx context.Context, id int64) (*LibSong, error)
	CreateSong(ctx context.Context, s *LibSong) error
	UpdateSong(ctx context.Context, s *LibSong) error
	DeleteSong(ctx context.Context, id int64) error
	StarSong(ctx context.Context, id int64, starred bool) error
	SongFacets(ctx context.Context) (*LibSongFacets, error)
	AllLyrics(ctx context.Context) ([]LibSong, error)
}

type InspirationService struct {
	repo   InspirationRepository
	guard  *LyricsGuard
	shihua *ShihuaService
	chat   materialChatFunc

	countsMu sync.Mutex
	counts   map[string]int
	countsAt time.Time
}

// ProvideInspirationService also hands the lyrics guard to 课本诗画 and the 素材库.
func ProvideInspirationService(repo InspirationRepository, shihuaSvc *ShihuaService, materialSvc *MaterialService) *InspirationService {
	s := NewInspirationService(repo, shihuaSvc)
	if shihuaSvc != nil {
		shihuaSvc.SetLyricsGuard(s.guard)
	}
	if materialSvc != nil {
		materialSvc.SetLyricsGuard(s.guard)
	}
	return s
}

func NewInspirationService(repo InspirationRepository, shihuaSvc *ShihuaService) *InspirationService {
	s := &InspirationService{repo: repo, guard: NewLyricsGuard(repo), shihua: shihuaSvc}
	if shihuaSvc != nil {
		s.chat = shihuaSvc.AdminChat
	}
	return s
}

func (s *InspirationService) Guard() *LyricsGuard { return s.guard }

// ---- poems ----

// libTerms splits a query into at most libMaxQueryTerms words.
func libTerms(q string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(q, func(r rune) bool { return r == ' ' || r == '　' || r == ',' || r == '，' || r == '、' }) {
		if f = strings.TrimSpace(f); f != "" && len(out) < libMaxQueryTerms {
			out = append(out, f)
		}
	}
	return out
}

func clampLimit(n, def, max int) int {
	if n <= 0 {
		return def
	}
	return min(n, max)
}

func (s *InspirationService) SearchPoems(ctx context.Context, q LibPoemQuery) ([]LibPoem, bool, error) {
	q.Limit = clampLimit(q.Limit, 30, 100)
	q.Offset = max(q.Offset, 0)
	if q.Collection != "" && LibCollections[q.Collection] == "" {
		return nil, false, ErrLibInvalid
	}
	return s.repo.SearchPoems(ctx, q)
}

func (s *InspirationService) GetPoem(ctx context.Context, id int64) (*LibPoem, error) {
	p, err := s.repo.GetPoem(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrLibPoemNotFound
	}
	if s.shihua != nil {
		if works, err := s.shihua.ListWorks(ctx, ShihuaWorkQuery{Q: p.Title}); err == nil {
			for _, w := range works {
				if libSameTitle(w.Title, p.Title) && w.Author == p.Author {
					p.InShihua = w.Slug
					break
				}
			}
		}
	}
	return p, nil
}

func libSameTitle(a, b string) bool {
	strip := func(s string) string {
		return strings.TrimSpace(regexp.MustCompile(`[（(].*?[）)]`).ReplaceAllString(s, ""))
	}
	return strip(a) == strip(b)
}

func (s *InspirationService) StarPoem(ctx context.Context, id int64, starred bool) error {
	if _, err := s.GetPoem(ctx, id); err != nil {
		return err
	}
	return s.repo.StarPoem(ctx, id, starred)
}

// PoemCounts is the number of works per collection (cached for an hour: the corpus changes only on
// reload).
func (s *InspirationService) PoemCounts(ctx context.Context) (map[string]int, error) {
	s.countsMu.Lock()
	defer s.countsMu.Unlock()
	if s.counts != nil && time.Since(s.countsAt) < time.Hour {
		return s.counts, nil
	}
	c, err := s.repo.PoemCounts(ctx)
	if err != nil {
		return nil, err
	}
	s.counts, s.countsAt = c, time.Now()
	return c, nil
}

var libGenreOf = map[string]string{"song_ci": "ci", "huajian": "ci", "nantang": "ci", "nalan": "ci", "yuanqu": "qu", "guwen": "wen", "lunyu": "wen", "chuci": "fu"}

// ToShihua copies a poem into 课本诗画 as a draft (or returns the work already there). The corpus text
// is not a textbook edition, so it still goes through both checks.
func (s *InspirationService) ToShihua(ctx context.Context, id int64, stage string) (*ShihuaWork, bool, error) {
	if s.shihua == nil {
		return nil, false, ErrLibInvalid
	}
	p, err := s.GetPoem(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if p.InShihua != "" {
		works, err := s.shihua.ListWorks(ctx, ShihuaWorkQuery{Q: p.Title})
		if err != nil {
			return nil, false, err
		}
		for i := range works {
			if works[i].Slug == p.InShihua {
				return &works[i], false, nil
			}
		}
	}
	genre := libGenreOf[p.Collection]
	if genre == "" {
		genre = "shi"
	}
	title := p.Title
	if utf8.RuneCountInString(title) > 32 {
		title = string([]rune(title)[:32])
	}
	w, err := s.shihua.CreateWork(ctx, ShihuaWorkInput{
		Slug: fmt.Sprintf("poem-%d", p.ID), Title: title, Author: p.Author, Dynasty: p.Dynasty, Genre: genre, Stage: stage,
		FullText: p.Content, Sort: 1000,
		Sources: fmt.Sprintf("灵感库 #%d（%s，%s）。语料版本不一定是课本版本，请对照课本校对", p.ID, LibCollections[p.Collection], "chinese-poetry / Werneror/Poetry，MIT"),
	})
	if errors.Is(err, ErrShihuaSlugTaken) {
		works, lerr := s.shihua.ListWorks(ctx, ShihuaWorkQuery{Q: title})
		if lerr == nil {
			for i := range works {
				if works[i].Slug == fmt.Sprintf("poem-%d", p.ID) {
					return &works[i], false, nil
				}
			}
		}
	}
	if err != nil {
		return nil, false, err
	}
	return w, true, nil
}

// ---- find by scene ----

type LibFindRequest struct {
	Scene      string `json:"scene"`
	Dynasty    string `json:"dynasty"`
	Collection string `json:"collection"`
	Count      int    `json:"count"`  // how many picks (default 8)
	Rerank     *bool  `json:"rerank"` // let the model pick and explain (default true)
}

type LibPick struct {
	Poem   LibPoem `json:"poem"`
	Reason string  `json:"reason"`
}

type LibFindResult struct {
	Keywords   []string  `json:"keywords"`
	Picks      []LibPick `json:"picks"`
	Candidates []LibPoem `json:"candidates"`
	Songs      []LibSong `json:"songs"`
	Model      string    `json:"model"`
}

const libKeywordsPrompt = `你是中国古典诗词检索助手。用户描述一个想画的画面或想表达的情绪。请给出 8–16 个检索词，用来在古诗词原文里做字面检索：
- 每个词 1–3 个字，必须是古诗词原文里会直接出现的字词（如「纸鸢」「东风」「柳」「儿童」「归来」），不要现代词，不要解释性的词。
- 先写最能代表画面的核心意象，再写季节、时辰、动作、情绪相关的字词。
只输出一个 JSON 对象：{"keywords":["…"]}`

const libRerankPrompt = `你是中国古典诗词与插画的美术指导。给你一个想画的画面描述和一批候选诗词（都来自真实语料库）。请从中选出最适合画成这幅画的几首：意象和场景贴合、画面感强、适合做海报的优先，名篇优先。
每首写一句理由（不超过 40 字，说明哪几句或哪些意象契合）。只能从候选里选，用候选的 id。
只输出一个 JSON 对象：{"picks":[{"id":123,"reason":"…"}]}`

func (s *InspirationService) FindByScene(ctx context.Context, in LibFindRequest) (*LibFindResult, error) {
	in.Scene = strings.TrimSpace(in.Scene)
	if in.Scene == "" || utf8.RuneCountInString(in.Scene) > 500 {
		return nil, ErrLibInvalid
	}
	if s.chat == nil {
		return nil, ErrLibNoModel
	}
	count := clampLimit(in.Count, 8, 20)
	ctx, cancel := context.WithTimeout(ctx, libModelTimeout)
	defer cancel()
	answer, model, err := s.chat(ctx, libKeywordsPrompt, "画面："+in.Scene)
	if err != nil {
		return nil, infraerrors.BadRequest("LIB_MODEL_FAILED", "文本模型调用失败："+infraerrors.Message(err))
	}
	var kw struct {
		Keywords []string `json:"keywords"`
	}
	if parseMaterialJSON(answer, &kw) != nil {
		return nil, ErrLibModelFailed
	}
	words := libCleanWords(kw.Keywords)
	if len(words) == 0 {
		return nil, ErrLibModelFailed
	}
	res := &LibFindResult{Keywords: words, Picks: []LibPick{}, Songs: []LibSong{}, Model: model}
	cands, err := s.repo.FindPoems(ctx, words, LibPoemQuery{Dynasty: in.Dynasty, Collection: in.Collection}, libFindCandidates)
	if err != nil {
		return nil, err
	}
	res.Candidates = cands
	if (in.Rerank == nil || *in.Rerank) && len(cands) > 0 {
		res.Picks = s.rerank(ctx, in.Scene, cands, count)
	}
	// Songs whose pasted lyrics share the words (admin-only reference).
	songs, _, err := s.repo.ListSongs(ctx, LibSongQuery{Limit: 200})
	if err == nil {
		res.Songs = libSongsWithWords(songs, words, 10)
	}
	return res, nil
}

func libCleanWords(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, w := range in {
		w = hanOnly(w)
		if n := utf8.RuneCountInString(w); n == 0 || n > 4 || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
		if len(out) == libMaxFindWords {
			break
		}
	}
	return out
}

func (s *InspirationService) rerank(ctx context.Context, scene string, cands []LibPoem, count int) []LibPick {
	byID := map[int64]LibPoem{}
	var b strings.Builder
	fmt.Fprintf(&b, "画面：%s\n要选：%d 首\n\n候选：\n", scene, count)
	for i, p := range cands {
		if i == libRerankPoems {
			break
		}
		byID[p.ID] = p
		lines := strings.Split(p.Content, "\n")
		if len(lines) > 8 {
			lines = append(lines[:8], "……")
		}
		famous := ""
		if p.Famous {
			famous = "（名篇）"
		}
		fmt.Fprintf(&b, "id=%d 《%s》%s %s%s\n%s\n\n", p.ID, p.Title, p.Dynasty, p.Author, famous, strings.Join(lines, " "))
	}
	answer, _, err := s.chat(ctx, libRerankPrompt, b.String())
	if err != nil {
		return []LibPick{}
	}
	var out struct {
		Picks []struct {
			ID     int64  `json:"id"`
			Reason string `json:"reason"`
		} `json:"picks"`
	}
	if parseMaterialJSON(answer, &out) != nil {
		return []LibPick{}
	}
	picks := []LibPick{}
	seen := map[int64]bool{}
	for _, p := range out.Picks {
		poem, ok := byID[p.ID]
		if !ok || seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		picks = append(picks, LibPick{Poem: poem, Reason: strings.TrimSpace(p.Reason)})
		if len(picks) == count {
			break
		}
	}
	return picks
}

// ---- songs ----

func (s *InspirationService) ListSongs(ctx context.Context, q LibSongQuery) ([]LibSong, int, error) {
	terms := libTerms(q.Q)
	q.Limit = clampLimit(q.Limit, 50, 200)
	q.Offset = max(q.Offset, 0)
	songs, total, err := s.repo.ListSongs(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	for i := range songs {
		songs[i].Matches = libLyricLines(songs[i].Lyrics, terms, 3)
		songs[i].HasLyrics = strings.TrimSpace(songs[i].Lyrics) != ""
		songs[i].Lyrics = ""
	}
	return songs, total, nil
}

func (s *InspirationService) GetSong(ctx context.Context, id int64) (*LibSong, error) {
	song, err := s.repo.GetSong(ctx, id)
	if err != nil {
		return nil, err
	}
	if song == nil {
		return nil, ErrLibSongNotFound
	}
	song.HasLyrics = strings.TrimSpace(song.Lyrics) != ""
	return song, nil
}

func normalizeLibSong(in *LibSong) error {
	in.Title, in.Artist = strings.TrimSpace(in.Title), strings.TrimSpace(in.Artist)
	in.Lyricist, in.Composer, in.Album = strings.TrimSpace(in.Lyricist), strings.TrimSpace(in.Composer), strings.TrimSpace(in.Album)
	in.Lyrics = strings.TrimSpace(strings.ReplaceAll(in.Lyrics, "\r\n", "\n"))
	in.Summary, in.Notes, in.LinkURL = strings.TrimSpace(in.Summary), strings.TrimSpace(in.Notes), strings.TrimSpace(in.LinkURL)
	in.Imagery, in.Moods, in.Tags = libCleanList(in.Imagery), libCleanList(in.Moods), libCleanList(in.Tags)
	if in.Allusions == nil {
		in.Allusions = []LibAllusion{}
	}
	if in.Year != nil && (*in.Year < 1900 || *in.Year > 2100) {
		return ErrLibInvalid
	}
	if in.LinkURL != "" {
		if u, err := url.Parse(in.LinkURL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return infraerrors.BadRequest("LIB_INVALID_LINK", "链接要以 http:// 或 https:// 开头（Invalid link）")
		}
	}
	switch {
	case in.Title == "", utf8.RuneCountInString(in.Title) > 100, utf8.RuneCountInString(in.Artist) > 100,
		utf8.RuneCountInString(in.Lyrics) > libMaxLyricsRunes, utf8.RuneCountInString(in.Summary) > 2000,
		utf8.RuneCountInString(in.Notes) > 5000, len(in.LinkURL) > 500, len(in.Allusions) > 30:
		return ErrLibInvalid
	}
	return nil
}

func libCleanList(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] || utf8.RuneCountInString(v) > 20 || len(out) == 30 {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func (s *InspirationService) CreateSong(ctx context.Context, adminID int64, in LibSong) (*LibSong, error) {
	if err := normalizeLibSong(&in); err != nil {
		return nil, err
	}
	in.CreatedBy = &adminID
	if err := s.repo.CreateSong(ctx, &in); err != nil {
		return nil, err
	}
	s.guard.Invalidate()
	return s.GetSong(ctx, in.ID)
}

func (s *InspirationService) UpdateSong(ctx context.Context, id int64, in LibSong) (*LibSong, error) {
	old, err := s.GetSong(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := normalizeLibSong(&in); err != nil {
		return nil, err
	}
	in.ID, in.CreatedBy, in.Starred = id, old.CreatedBy, old.Starred
	if err := s.repo.UpdateSong(ctx, &in); err != nil {
		return nil, err
	}
	s.guard.Invalidate()
	return s.GetSong(ctx, id)
}

func (s *InspirationService) DeleteSong(ctx context.Context, id int64) error {
	if _, err := s.GetSong(ctx, id); err != nil {
		return err
	}
	if err := s.repo.DeleteSong(ctx, id); err != nil {
		return err
	}
	s.guard.Invalidate()
	return nil
}

func (s *InspirationService) StarSong(ctx context.Context, id int64, starred bool) error {
	if _, err := s.GetSong(ctx, id); err != nil {
		return err
	}
	return s.repo.StarSong(ctx, id, starred)
}

func (s *InspirationService) SongFacets(ctx context.Context) (*LibSongFacets, error) {
	return s.repo.SongFacets(ctx)
}

// libLyricLines are up to max lyric lines containing any of the terms.
func libLyricLines(lyrics string, terms []string, max int) []string {
	if len(terms) == 0 || lyrics == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(lyrics, "\n") {
		line = strings.TrimSpace(line)
		for _, t := range terms {
			if line != "" && strings.Contains(strings.ToLower(line), strings.ToLower(t)) {
				out = append(out, line)
				break
			}
		}
		if len(out) == max {
			break
		}
	}
	return out
}

// libSongsWithWords are the songs whose lyrics contain the most of the words (with those lines),
// without the lyrics themselves.
func libSongsWithWords(songs []LibSong, words []string, max int) []LibSong {
	type scored struct {
		song  LibSong
		score int
	}
	var list []scored
	for _, s := range songs {
		n := 0
		for _, w := range words {
			if strings.Contains(s.Lyrics, w) {
				n++
			}
		}
		if n == 0 {
			continue
		}
		s.Matches = libLyricLines(s.Lyrics, words, 3)
		s.HasLyrics, s.Lyrics = true, ""
		list = append(list, scored{s, n})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	out := []LibSong{}
	for i := 0; i < len(list) && i < max; i++ {
		out = append(out, list[i].song)
	}
	return out
}

// ---- AI analysis of a song ----

type LibSongAnalysis struct {
	Summary   string        `json:"summary"`
	Imagery   []string      `json:"imagery"`
	Moods     []string      `json:"moods"`
	Tags      []string      `json:"tags"`
	Allusions []LibAllusion `json:"allusions"`
	Model     string        `json:"model"`
	// Dropped lists fields left out because they repeated the lyrics.
	Dropped []string `json:"dropped"`
}

const libAnalyzePrompt = `你是流行音乐与中国古典文学研究者。下面是一首歌的信息和歌词（仅供内部分析，不会发表）。请用你自己的话分析，输出：
- summary：60–120 字的意境与情绪概括。不得引用歌词原句，不要连续照搬歌词里 5 个以上的字。
- imagery：5–12 个适合入画的画面意象（名词或短语，如「江南雨巷」「青瓷」）。
- moods：2–5 个情绪词。
- tags：1–4 个风格标签（如 中国风、民谣、摇滚）。
- allusions：歌词化用或呼应的古诗词（只写你确定的；没有就给空数组）。每项：title（古诗标题）、author（古诗作者）、note（用自己的话说明呼应了什么，不抄歌词原句）。
只输出一个 JSON 对象：{"summary":"…","imagery":[],"moods":[],"tags":[],"allusions":[{"title":"…","author":"…","note":"…"}]}`

// AnalyzeSong drafts a summary, imagery, moods, tags and allusions from the song's lyrics. Nothing is
// saved: the admin edits and saves the card. Any field that repeats a lyric line is dropped.
func (s *InspirationService) AnalyzeSong(ctx context.Context, id int64) (*LibSongAnalysis, error) {
	song, err := s.GetSong(ctx, id)
	if err != nil {
		return nil, err
	}
	if !song.HasLyrics {
		return nil, infraerrors.BadRequest("LIB_NO_LYRICS", "这首歌还没有歌词，先粘贴歌词再分析（No lyrics yet）")
	}
	if s.chat == nil {
		return nil, ErrLibNoModel
	}
	ctx, cancel := context.WithTimeout(ctx, libModelTimeout)
	defer cancel()
	user := fmt.Sprintf("歌名：%s\n演唱：%s\n作词：%s\n\n歌词：\n%s", song.Title, song.Artist, song.Lyricist, song.Lyrics)
	answer, model, err := s.chat(ctx, libAnalyzePrompt, user)
	if err != nil {
		return nil, infraerrors.BadRequest("LIB_MODEL_FAILED", "文本模型调用失败："+infraerrors.Message(err))
	}
	var a LibSongAnalysis
	if parseMaterialJSON(answer, &a) != nil {
		return nil, ErrLibModelFailed
	}
	a.Model, a.Dropped = model, []string{}
	own := map[string]bool{}
	for _, line := range strings.Split(song.Lyrics, "\n") {
		for _, g := range lyricsGrams(hanOnly(line)) {
			own[g] = true
		}
	}
	repeats := func(t string) bool {
		for _, g := range lyricsGrams(hanOnly(t)) {
			if own[g] {
				return true
			}
		}
		return false
	}
	a.Summary = strings.TrimSpace(a.Summary)
	if repeats(a.Summary) {
		a.Summary, a.Dropped = "", append(a.Dropped, "summary")
	}
	a.Imagery, a.Moods, a.Tags = libCleanList(a.Imagery), libCleanList(a.Moods), libCleanList(a.Tags)
	allusions := []LibAllusion{}
	for _, al := range a.Allusions {
		al.Title, al.Author, al.Note = strings.TrimSpace(al.Title), strings.TrimSpace(al.Author), strings.TrimSpace(al.Note)
		if al.Title == "" {
			continue
		}
		if repeats(al.Note) {
			al.Note = ""
			a.Dropped = append(a.Dropped, "allusion note: "+al.Title)
		}
		if p, err := s.repo.FindPoemByTitle(ctx, al.Title, al.Author); err == nil && p != nil {
			al.PoemID = &p.ID
		}
		allusions = append(allusions, al)
	}
	a.Allusions = allusions
	return &a, nil
}
