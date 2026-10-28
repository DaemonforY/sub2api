package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 「AI 添加篇目」: the admin describes a theme (or pastes titles), the text model lists matching
// classical works, and the server fetches each text from 维基文库 (zh-hans) — the model never supplies
// the text that goes on a poster. The model also writes the text from memory; the two are compared
// character by character so the admin only has to look at the works where they differ. Nothing is
// saved here: the page edits the draft and imports it through Import, as a JSON file would.

const (
	shihuaDraftMaxWorks     = 30
	shihuaDraftModelTimeout = 4 * time.Minute
	shihuaDraftFetchTimeout = 90 * time.Second
	shihuaDraftFetchers     = 2
	shihuaWikisourceGap     = 400 * time.Millisecond // between requests: the API answers 429 to bursts
	shihuaDraftMaxDiffRunes = 1500
	shihuaWikisourceBase    = "https://zh.wikisource.org"
)

const (
	ShihuaDraftMatch    = "match"    // 维基文库 text found, the model's memory agrees
	ShihuaDraftDiff     = "diff"     // found, but they differ somewhere
	ShihuaDraftAIOnly   = "ai_only"  // not found: the text is the model's, to be checked word by word
	ShihuaDraftExists   = "exists"   // already in the library
	ShihuaDraftRejected = "rejected" // not a classical work (copyright) or unusable
)

var ErrShihuaDraftEmpty = infraerrors.BadRequest("SHIHUA_DRAFT_EMPTY", "请写一句主题，或贴上篇名清单（Topic or titles required）")

type ShihuaDraftRequest struct {
	Topic  string `json:"topic"`  // e.g. 小学春天田园、有孩子的古诗
	Titles string `json:"titles"` // one per line, e.g. 村居 高鼎
	Stage  string `json:"stage"`  // primary / junior / senior; "" = the model decides
	Count  int    `json:"count"`  // 1–30, used with a topic
}

type ShihuaTextDiff struct {
	Source string `json:"source"` // 维基文库
	AI     string `json:"ai"`
}

type ShihuaDraftItem struct {
	Work    ShihuaWorkInput  `json:"work"`
	Status  string           `json:"status"`
	Diffs   []ShihuaTextDiff `json:"diffs"`
	AIText  string           `json:"ai_text"`
	Page    string           `json:"page"`
	PageURL string           `json:"page_url"`
	Reason  string           `json:"reason"`
	Problem string           `json:"problem"`
}

type ShihuaDraftResult struct {
	Items []ShihuaDraftItem `json:"items"`
	Model string            `json:"model"`
}

// shihuaDraftEntry is how the model lists a work.
type shihuaDraftEntry struct {
	Title   string   `json:"title"`
	Author  string   `json:"author"`
	Dynasty string   `json:"dynasty"`
	Genre   string   `json:"genre"`
	Stage   string   `json:"stage"`
	Grade   string   `json:"grade"`
	Slug    string   `json:"slug"`
	Page    string   `json:"page"`   // 维基文库 page title, traditional
	Pages   []string `json:"pages"`  // other likely page titles
	Search  string   `json:"search"` // first line in traditional, for the search fallback
	First   string   `json:"first"`  // first characters of the text (simplified)
	Last    string   `json:"last"`   // last characters, no punctuation
	Text    string   `json:"text"`   // the text from memory
	Reason  string   `json:"reason"`
}

const shihuaDraftSystemPrompt = `你是中学语文教研员兼古籍编辑，帮管理员挑选适合画成插画海报的古诗文。

规则：
1. 只选作者在 1900 年以前去世的古代作品（先秦到清），不选近现代作品。
2. 先选统编语文课本（小学、初中、高中）里的篇目，其次是《千家诗》《唐诗三百首》等常见选本里的名篇，少选冷门作品；都要画面感强。组诗只选课本选的那一首，标题写成课本里的样子（如「绝句（两个黄鹂鸣翠柳）」）。
3. 若给了篇名清单，只按清单逐条给出，顺序不变，认不出的也照样列出并在 reason 里说明。
4. 每条字段：
   - title 篇名；author 作者；dynasty 朝代（如 唐、北宋、南宋、清）；genre：shi 诗 / ci 词 / qu 曲 / wen 文 / fu 赋
   - stage：primary 小学 / junior 初中 / senior 高中；grade：课本所在册（如 二下、八上），拿不准就留空
   - slug：篇名的汉语拼音，小写字母和连字符，如 cun-ju
   - page：这篇在中文维基文库最可能的页面名（繁体），同名页面常带括号作者，如「村居 (高鼎)」「春曉 (孟浩然)」
   - pages：另外 1–3 个可能的页面名（繁体），如词牌页「清平樂 (辛棄疾)」、组诗总题「池上二絕」
   - search：正文第一句的繁体写法
   - first：正文开头 4–6 个字（简体，不含标点）；last：正文结尾 3–5 个字（简体，不含标点）
   - text：你记忆中的全文（简体，带标点，诗每联一行，词按上下阕分行）
   - reason：一句话说明为什么适合这个主题、画面里有什么
只输出一个 JSON 对象：{"works":[{...}]}`

func shihuaDraftUserPrompt(in ShihuaDraftRequest) string {
	var b strings.Builder
	if t := strings.TrimSpace(in.Titles); t != "" {
		fmt.Fprintf(&b, "篇名清单：\n%s\n", t)
	} else {
		fmt.Fprintf(&b, "主题：%s\n数量：%d 篇\n", strings.TrimSpace(in.Topic), in.Count)
	}
	if st := map[string]string{"primary": "小学", "junior": "初中", "senior": "高中"}[in.Stage]; st != "" {
		fmt.Fprintf(&b, "学段：%s\n", st)
	}
	return b.String()
}

// ---- 维基文库 ----

type shihuaWikisource struct {
	base   string
	client *http.Client
	gap    time.Duration
	mu     sync.Mutex
	last   time.Time
}

// wait spaces requests out by gap.
func (w *shihuaWikisource) wait(ctx context.Context) error {
	w.mu.Lock()
	next := w.last.Add(w.gap)
	now := time.Now()
	if next.Before(now) {
		next = now
	}
	w.last = next
	w.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(next)):
		return nil
	}
}

func newShihuaWikisource(base string) *shihuaWikisource {
	return &shihuaWikisource{base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 30 * time.Second}, gap: shihuaWikisourceGap}
}

func (w *shihuaWikisource) api(ctx context.Context, params url.Values) (map[string]any, error) {
	params.Set("format", "json")
	params.Set("formatversion", "2")
	for attempt := 0; ; attempt++ {
		if err := w.wait(ctx); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.base+"/w/api.php?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}
		// Wikimedia asks automated clients to name themselves with a contact address.
		req.Header.Set("User-Agent", "HiveGPT-shihua/1.0 (https://hivegpt.cn; textbook poem import, low volume)")
		resp, err := w.client.Do(req)
		if err != nil {
			if attempt < 1 && ctx.Err() == nil {
				continue // one more try on a dropped connection
			}
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < 3 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(3+attempt*5) * time.Second):
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("维基文库 HTTP %d", resp.StatusCode)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
		// Busy / rate-limit errors come back as 200 with an error object; wait and try again.
		if e, ok := out["error"].(map[string]any); ok {
			code, _ := e["code"].(string)
			if (strings.Contains(code, "too-busy") || code == "ratelimited" || code == "maxlag") && attempt < 3 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(3+attempt*5) * time.Second):
				}
				continue
			}
		}
		return out, nil
	}
}

var (
	shihuaHTMLDrop    = regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</(style|script)>|<sup[^>]*class="reference".*?</sup>`)
	shihuaHTMLLicense = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*licenseContainer.*`)
	shihuaHTMLBreak   = regexp.MustCompile(`<br\s*/?>|</p>|</div>|</li>|</dd>|</h\d>`)
	shihuaHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	shihuaVariantRe   = regexp.MustCompile(`一作「[^」]*」(，一作「[^」]*」)*|〈[^〉]*〉|\[[^\]]*\]|（一作[^）]*）|\(一作[^)]*\)`)
)

// shihuaGlyphs are variant forms zh-hans conversion leaves that textbooks write in the standard form.
var shihuaGlyphs = strings.NewReplacer("朶", "朵", "隣", "邻", "皐", "皋")

// pageText returns a page's text in simplified Chinese; ok is false when the page does not exist.
func (w *shihuaWikisource) pageText(ctx context.Context, title string) (string, bool, error) {
	data, err := w.api(ctx, url.Values{"action": {"parse"}, "page": {title}, "prop": {"text"}, "redirects": {"1"}, "variant": {"zh-hans"}})
	if err != nil {
		return "", false, err
	}
	parse, _ := data["parse"].(map[string]any)
	h, _ := parse["text"].(string)
	if h == "" {
		return "", false, nil
	}
	h = shihuaHTMLDrop.ReplaceAllString(h, "")
	h = shihuaHTMLLicense.ReplaceAllString(h, "")
	h = shihuaHTMLBreak.ReplaceAllString(h, "\n")
	t := html.UnescapeString(shihuaHTMLTag.ReplaceAllString(h, ""))
	return shihuaGlyphs.Replace(shihuaVariantRe.ReplaceAllString(t, "")), true, nil
}

func (w *shihuaWikisource) search(ctx context.Context, q string) ([]string, error) {
	data, err := w.api(ctx, url.Values{"action": {"query"}, "list": {"search"}, "srsearch": {q}, "srlimit": {"5"}})
	if err != nil {
		return nil, err
	}
	query, _ := data["query"].(map[string]any)
	hits, _ := query["search"].([]any)
	out := []string{}
	for _, h := range hits {
		if m, ok := h.(map[string]any); ok {
			if t, _ := m["title"].(string); t != "" {
				out = append(out, t)
			}
		}
	}
	return out, nil
}

func (w *shihuaWikisource) pageURL(title string) string {
	return w.base + "/wiki/" + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
}

// ---- text helpers ----

var shihuaPunct = regexp.MustCompile(`[，。！？；、：]`)

// shihuaCore keeps the characters only (no punctuation, spaces or line breaks).
func shihuaCore(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF) || (r >= 0x20000 && r <= 0x2FA1F) {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

// shihuaCutText finds the work between first and last in a page's text (matching on characters only,
// so line breaks and spaces inside the page don't matter) and returns that stretch as the page has it.
func shihuaCutText(page, first, last string) (string, bool) {
	first, last = shihuaCore(first), shihuaCore(last)
	if first == "" || last == "" {
		return "", false
	}
	// Map each kept character back to its byte offset in page.
	type pos struct{ start, end int }
	var core []rune
	var at []pos
	for i, r := range page {
		if shihuaCore(string(r)) != "" {
			core = append(core, r)
			at = append(at, pos{i, i + utf8.RuneLen(r)})
		}
	}
	cs := string(core)
	i := strings.Index(cs, first)
	if i < 0 {
		return "", false
	}
	j := strings.Index(cs[i:], last)
	if j < 0 {
		return "", false
	}
	startRune := utf8.RuneCountInString(cs[:i])
	endRune := utf8.RuneCountInString(cs[:i+j]) + utf8.RuneCountInString(last) - 1
	end := at[endRune].end
	// Keep a closing punctuation mark right after the last character.
	if r, n := utf8.DecodeRuneInString(page[end:]); n > 0 && strings.ContainsRune("。！？；”」", r) {
		end += n
	}
	return page[at[startRune].start:end], true
}

// shihuaTidyText turns a cut into lines: punctuated text keeps its line breaks without spaces; text
// without punctuation is punctuated when its lines are regular 5- or 7-character verse, or takes the
// model's punctuation when the characters are exactly the model's. ok is false when it stays bare.
func shihuaTidyText(cut, aiText string) (string, bool) {
	cut = strings.ReplaceAll(strings.ReplaceAll(cut, "「", "“"), "」", "”")
	if shihuaPunct.MatchString(cut) {
		lines := []string{}
		for _, l := range strings.Split(cut, "\n") {
			if l = strings.Join(strings.Fields(l), ""); l == "" {
				continue
			}
			// A line ending in a comma continues on the next one (pages that set one clause per line).
			if n := len(lines); n > 0 && (strings.HasSuffix(lines[n-1], "，") || strings.HasSuffix(lines[n-1], "、")) {
				lines[n-1] += l
				continue
			}
			lines = append(lines, l)
		}
		return strings.Join(lines, "\n"), true
	}
	units := strings.Fields(cut)
	n := 0
	if len(units) >= 2 && len(units)%2 == 0 {
		n = utf8.RuneCountInString(units[0])
		for _, u := range units {
			if utf8.RuneCountInString(u) != n {
				n = 0
				break
			}
		}
	}
	if n == 5 || n == 7 {
		lines := []string{}
		for i := 0; i < len(units); i += 2 {
			lines = append(lines, units[i]+"，"+units[i+1]+"。")
		}
		return strings.Join(lines, "\n"), true
	}
	if aiText != "" && shihuaCore(aiText) == shihuaCore(cut) {
		return normalizeShihuaText(aiText), true
	}
	return strings.Join(units, "\n"), false
}

// shihuaDiff lists where a and b differ, on characters only (an LCS walk; texts are short).
func shihuaDiff(a, b string) []ShihuaTextDiff {
	ra, rb := []rune(shihuaCore(a)), []rune(shihuaCore(b))
	if string(ra) == string(rb) {
		return []ShihuaTextDiff{}
	}
	if len(ra) > shihuaDraftMaxDiffRunes || len(rb) > shihuaDraftMaxDiffRunes {
		return []ShihuaTextDiff{{Source: "（全文太长，未逐字比对）", AI: ""}}
	}
	lcs := make([][]int, len(ra)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(rb)+1)
	}
	for i := len(ra) - 1; i >= 0; i-- {
		for j := len(rb) - 1; j >= 0; j-- {
			if ra[i] == rb[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	out := []ShihuaTextDiff{}
	var da, db []rune
	flush := func() {
		if len(da) > 0 || len(db) > 0 {
			out = append(out, ShihuaTextDiff{Source: string(da), AI: string(db)})
			da, db = nil, nil
		}
	}
	i, j := 0, 0
	for i < len(ra) || j < len(rb) {
		switch {
		case i < len(ra) && j < len(rb) && ra[i] == rb[j]:
			flush()
			i++
			j++
		case j < len(rb) && (i == len(ra) || lcs[i][j+1] >= lcs[i+1][j]):
			db = append(db, rb[j])
			j++
		default:
			da = append(da, ra[i])
			i++
		}
	}
	flush()
	return out
}

var shihuaDynastyAlias = map[string]string{"宋代": "宋", "唐代": "唐", "清代": "清", "明代": "明", "元代": "元", "汉代": "汉", "晋代": "晋"}

func shihuaAncient(dynasty string) bool {
	d := strings.TrimSpace(dynasty)
	if a, ok := shihuaDynastyAlias[d]; ok {
		d = a
	}
	return materialAncientDynasties[d]
}

func shihuaDraftSlug(e shihuaDraftEntry) string {
	s := strings.ToLower(strings.TrimSpace(e.Slug))
	s = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(strings.ReplaceAll(s, " ", "-"), "")
	s = strings.Trim(regexp.MustCompile(`-+`).ReplaceAllString(s, "-"), "-")
	if len(s) > 64 {
		s = strings.Trim(s[:64], "-")
	}
	if shihuaSlugRe.MatchString(s) {
		return s
	}
	h := sha1.Sum([]byte(e.Title + "|" + e.Author))
	return "work-" + hex.EncodeToString(h[:4])
}

// ---- service ----

// DraftWorks lists works for a theme (or a list of titles) with their texts from 维基文库.
func (s *ShihuaService) DraftWorks(ctx context.Context, in ShihuaDraftRequest) (*ShihuaDraftResult, error) {
	in.Topic, in.Titles = strings.TrimSpace(in.Topic), strings.TrimSpace(in.Titles)
	if in.Topic == "" && in.Titles == "" {
		return nil, ErrShihuaDraftEmpty
	}
	if utf8.RuneCountInString(in.Topic) > 300 || utf8.RuneCountInString(in.Titles) > 3000 {
		return nil, ErrShihuaInvalid
	}
	if in.Stage != "" && !shihuaStages[in.Stage] {
		return nil, ErrShihuaInvalid
	}
	if in.Count <= 0 {
		in.Count = 10
	}
	in.Count = min(in.Count, shihuaDraftMaxWorks)

	mctx, cancel := context.WithTimeout(ctx, shihuaDraftModelTimeout)
	answer, model, err := s.AdminChat(mctx, shihuaDraftSystemPrompt, shihuaDraftUserPrompt(in))
	cancel()
	if err != nil {
		// Typed errors (no key…) carry their own message; plain ones (network, upstream) keep their text.
		msg := infraerrors.Message(err)
		var app *infraerrors.ApplicationError
		if !errors.As(err, &app) {
			msg = clipRunes(err.Error(), 300)
		}
		return nil, infraerrors.BadRequest("SHIHUA_DRAFT_FAILED", "文本模型调用失败："+msg)
	}
	start, end := strings.Index(answer, "{"), strings.LastIndex(answer, "}")
	var doc struct {
		Works []shihuaDraftEntry `json:"works"`
	}
	if start < 0 || end <= start || json.Unmarshal([]byte(answer[start:end+1]), &doc) != nil || len(doc.Works) == 0 {
		return nil, infraerrors.BadRequest("SHIHUA_DRAFT_FAILED", "文本模型没有返回可用的篇目，请重试（The text model returned no usable list）")
	}
	if len(doc.Works) > shihuaDraftMaxWorks {
		doc.Works = doc.Works[:shihuaDraftMaxWorks]
	}

	existing, err := s.repo.ListWorks(ctx, ShihuaWorkQuery{})
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	slugs := map[string]bool{}
	for _, w := range existing {
		have[w.Title+"|"+w.Author] = true
		slugs[w.Slug] = true
	}

	items := make([]ShihuaDraftItem, len(doc.Works))
	fctx, cancel := context.WithTimeout(ctx, shihuaDraftFetchTimeout)
	defer cancel()
	var wg sync.WaitGroup
	sem := make(chan struct{}, shihuaDraftFetchers)
	used := map[string]bool{}
	for i, e := range doc.Works {
		e.Title, e.Author, e.Dynasty = strings.TrimSpace(e.Title), strings.TrimSpace(e.Author), strings.TrimSpace(e.Dynasty)
		if a, ok := shihuaDynastyAlias[e.Dynasty]; ok {
			e.Dynasty = a
		}
		stage := in.Stage
		if stage == "" {
			stage = strings.TrimSpace(e.Stage)
		}
		if !shihuaStages[stage] {
			stage = "primary"
		}
		genre := strings.TrimSpace(e.Genre)
		if !shihuaGenres[genre] {
			genre = "shi"
		}
		base := shihuaDraftSlug(e)
		slug := base
		for n := 2; used[slug]; n++ {
			slug = fmt.Sprintf("%s-%d", strings.TrimRight(base[:min(len(base), 60)], "-"), n)
		}
		used[slug] = true
		item := ShihuaDraftItem{
			Work: ShihuaWorkInput{Slug: slug, Title: e.Title, Author: e.Author, Dynasty: e.Dynasty, Genre: genre, Stage: stage,
				Grade: strings.TrimSpace(e.Grade), Textbooks: []ShihuaTextbook{}, Sort: 1000 + i*10},
			AIText: normalizeShihuaText(e.Text), Reason: strings.TrimSpace(e.Reason), Diffs: []ShihuaTextDiff{},
		}
		switch {
		case e.Title == "":
			item.Status, item.Problem = ShihuaDraftRejected, "模型没给出篇名"
		case !shihuaAncient(e.Dynasty):
			item.Status, item.Problem = ShihuaDraftRejected, "不是古代作品（"+e.Dynasty+"），可能仍有版权，不收"
		case have[e.Title+"|"+e.Author] || slugs[slug]:
			item.Status, item.Problem = ShihuaDraftExists, "库里已有这篇"
		}
		items[i] = item
		if item.Status != "" {
			continue
		}
		wg.Add(1)
		go func(i int, e shihuaDraftEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s.fillDraftText(fctx, &items[i], e)
		}(i, e)
	}
	wg.Wait()
	return &ShihuaDraftResult{Items: items, Model: model}, nil
}

// fillDraftText finds the work on 维基文库 and compares it with the model's text.
func (s *ShihuaService) fillDraftText(ctx context.Context, item *ShihuaDraftItem, e shihuaDraftEntry) {
	// Anchors to cut the work out of a page: the model's own, then short ones from its text (a single
	// variant character in a longer anchor, 「一径」 against the page's 「一迳」, would miss the page).
	type anchor struct{ first, last string }
	var anchors []anchor
	if shihuaCore(e.First) != "" && shihuaCore(e.Last) != "" {
		anchors = append(anchors, anchor{e.First, e.Last})
	}
	if core := []rune(shihuaCore(e.Text)); len(core) >= 8 {
		anchors = append(anchors, anchor{string(core[:4]), string(core[len(core)-3:])})
	}
	candidates := []string{}
	seen := map[string]bool{}
	add := func(t string) {
		if t = strings.TrimSpace(t); t != "" && !seen[t] && len(candidates) < 6 {
			seen[t] = true
			candidates = append(candidates, t)
		}
	}
	add(e.Page)
	for _, p := range e.Pages {
		add(p)
	}
	var cut, page string
	var fetchErr error // the last network / HTTP failure: "not found" and "unreachable" read differently
	reached := false
	tryPages := func() {
		for _, title := range candidates {
			if cut != "" {
				return
			}
			text, ok, err := s.wikisource.pageText(ctx, title)
			if err != nil {
				fetchErr = err
				continue
			}
			reached = true
			if !ok {
				continue
			}
			for _, a := range anchors {
				if c, found := shihuaCutText(text, a.first, a.last); found {
					cut, page = c, title
					break
				}
			}
		}
	}
	tryPages()
	if cut == "" {
		first := ""
		if len(anchors) > 0 {
			first = anchors[len(anchors)-1].first
		}
		for _, q := range []string{e.Search, e.Title + " " + e.Author, first} {
			if strings.TrimSpace(q) == "" || cut != "" {
				continue
			}
			hits, err := s.wikisource.search(ctx, q)
			if err != nil {
				fetchErr = err
				continue
			}
			reached = true
			for _, h := range hits {
				add(h)
			}
			tryPages()
		}
	}

	w := &item.Work
	if cut == "" {
		item.Status = ShihuaDraftAIOnly
		item.Problem = "维基文库没找到这篇，正文是 AI 默写的，必须逐字核对"
		if !reached && fetchErr != nil {
			item.Problem = "访问维基文库失败（" + clipRunes(fetchErr.Error(), 120) + "），正文是 AI 默写的，必须逐字核对；可以稍后重新生成"
		}
		w.FullText = item.AIText
		w.Sources = "AI 默写（未找到维基文库原文）；须与课本逐字核对"
		if w.FullText == "" {
			item.Status, item.Problem = ShihuaDraftRejected, "维基文库没找到，模型也没给出正文"
		}
		return
	}
	text, punctuated := shihuaTidyText(cut, item.AIText)
	w.FullText = text
	item.Page, item.PageURL = page, s.wikisource.pageURL(page)
	w.Sources = fmt.Sprintf("维基文库《%s》（简体）；须与课本逐字核对", page)
	item.Diffs = shihuaDiff(text, item.AIText)
	notes := []string{}
	if !punctuated {
		notes = append(notes, "维基文库本无标点，请按课本补上")
	}
	if len(item.Diffs) == 0 {
		item.Status = ShihuaDraftMatch
	} else {
		item.Status = ShihuaDraftDiff
		parts := []string{}
		for _, d := range item.Diffs {
			parts = append(parts, fmt.Sprintf("维基文库「%s」/ AI「%s」", d.Source, d.AI))
		}
		notes = append(notes, "与 AI 默写不同："+strings.Join(parts, "；"))
	}
	if g := strings.TrimSpace(w.Grade); g != "" {
		notes = append(notes, "年级「"+g+"」是 AI 猜的，请核对")
	}
	w.VariantNote = strings.Join(notes, "；")
}
