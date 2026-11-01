package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// The poem text for AI 做贴图 (tietu.go). Texts are never written by the model: they come from the poem
// library (灵感库) and 维基文库. When both have the work and the characters line up one to one, the
// places where they differ are picked by the model ("the reading most people know", the operator's
// rule) and applied to the library text; anything else keeps the library text and says so.

// fetchPoem finds the poem the brief names and cross-checks it.
func (s *TietuService) fetchPoem(ctx context.Context, p *TietuProject, key string) (*TietuPoem, error) {
	lib := s.findLibPoem(ctx, p.Brief.Input)
	var hints *tietuHints
	words := strings.Fields(p.Brief.Input)
	if lib == nil || (len(words) > 0 && tietuTitleScore(*lib, words[0], words) < 4) {
		// Not found, or only a loose match: the library may know the work under another title
		// (木兰诗 is 「木兰辞」 there). The model only gives search hints; the text still comes from the library.
		if hints = s.resolveHints(ctx, p, key); hints != nil {
			if found := s.findByHints(ctx, hints); found != nil {
				lib = found
			}
		}
	}
	if lib == nil {
		return nil, ErrTietuPoem
	}
	po := &TietuPoem{Title: tietuShortTitle(lib.Title), Author: lib.Author, Dynasty: lib.Dynasty, LibID: lib.ID,
		Text:     normalizeShihuaText(tietuDropNotes(lib.Content)),
		Sources:  []TietuSource{{Name: "诗词库", Text: normalizeShihuaText(tietuDropNotes(lib.Content))}},
		Variants: []TietuVariant{}}
	if hints != nil {
		// Show the title (and, for anonymous works, the attribution) most readers know.
		if hints.Title != "" && (hints.Title == po.Title || tietuIn(po.Title, hints.Aliases) || tietuIn(hints.Title, []string{lib.Title})) {
			po.Title = hints.Title
		} else if hints.Title != "" && strings.HasPrefix(shihuaCore(po.Text), shihuaCore(hints.FirstLine)) && shihuaCore(hints.FirstLine) != "" {
			po.Title = hints.Title
		}
		if (po.Author == "" || po.Author == "无名氏" || po.Author == "佚名") && hints.Author != "" {
			po.Author = hints.Author
			if hints.Dynasty != "" {
				po.Dynasty = hints.Dynasty
			}
		}
	}
	wikiText, wikiURL := s.findWikiText(ctx, po)
	if wikiText == "" && lib.Title != po.Title {
		alt := *po
		alt.Title = tietuShortTitle(lib.Title)
		wikiText, wikiURL = s.findWikiText(ctx, &alt)
	}
	if wikiText == "" {
		po.Note = "维基文库没找到对应的页面，只用了诗词库的版本"
	} else {
		po.Sources = append(po.Sources, TietuSource{Name: "维基文库", URL: wikiURL, Text: wikiText})
		s.mergeSources(ctx, p, po, wikiText, key)
	}
	s.commonReadings(ctx, p, po, key)
	return po, nil
}

// mergeSources aligns the library text with 维基文库's, lets the model pick the best-known reading at
// every difference and applies the picks to the library text (keeping its punctuation and lines).
func (s *TietuService) mergeSources(ctx context.Context, p *TietuProject, po *TietuPoem, wikiText, key string) {
	a, b := []rune(shihuaCore(po.Text)), []rune(shihuaCore(wikiText))
	if string(a) == string(b) {
		po.Agreed = true
		return
	}
	runs, ok := tietuAlign(a, b)
	if !ok || len(runs) == 0 || len(runs) > 24 {
		po.Note = "两个来源差异较多，用了诗词库的版本，发布前请对照课本看一眼"
		return
	}
	picks := s.pickVariants(ctx, p, po, runs, a, b, key)
	replace := map[int]tietuRun{} // library core index where a run starts → run (only runs taking 维基文库)
	for i, rn := range runs {
		la, wb := string(a[rn.ai:rn.aj]), string(b[rn.bi:rn.bj])
		chosen, why := la, ""
		if c, ok := picks[i]; ok {
			why = c.why
			if c.pick == 1 {
				chosen = wb
				replace[rn.ai] = rn
			}
		}
		po.Variants = append(po.Variants, TietuVariant{Options: []string{tietuShow(la), tietuShow(wb)}, Chosen: tietuShow(chosen), Why: why, Context: tietuContext(a, rn.ai, rn.aj)})
	}
	po.Text = tietuApply(po.Text, b, replace)
	po.Agreed = true
}

func tietuShow(s string) string {
	if s == "" {
		return "（无）"
	}
	return s
}

// tietuApply rewrites text (with punctuation) so that each run in replace takes 维基文库's characters.
func tietuApply(text string, b []rune, replace map[int]tietuRun) string {
	var out strings.Builder
	k := 0    // core index in the library text
	skip := 0 // library core characters still to drop
	for _, r := range text {
		isCore := shihuaCore(string(r)) != ""
		if isCore {
			if rn, ok := replace[k]; ok {
				_, _ = out.WriteString(string(b[rn.bi:rn.bj]))
				skip = rn.aj - rn.ai
			}
			k++
			if skip > 0 {
				skip--
				continue
			}
		}
		_, _ = out.WriteRune(r)
	}
	if rn, ok := replace[k]; ok && rn.ai == rn.aj { // insertion at the very end
		_, _ = out.WriteString(string(b[rn.bi:rn.bj]))
	}
	return out.String()
}

// commonReadings asks for the few places where the reading most people learned (the current
// textbook or the common anthologies) differs from the text, e.g. 「两兔」 → 「双兔」. Only same-length
// changes of a phrase that occurs once are applied, at most three, and each is listed as a variant.
func (s *TietuService) commonReadings(ctx context.Context, p *TietuProject, po *TietuPoem, key string) {
	msgs := []LearnMessage{{Role: "system", Content: `下面是一首古诗文的原文。对照现行语文课本或最通行的选本：如果有个别字的写法和大多数人背的不同，列出来；没有就返回空数组。只列你确定的，最多 3 处；from 是原文里连续的一小段（不含标点，2–8 个字），to 是通行写法，字数必须相同。只输出一个 JSON 对象：{"changes":[{"from":"","to":"","why":"20 字以内"}]}`},
		{Role: "user", Content: fmt.Sprintf("《%s》%s\n%s", po.Title, po.Author, po.Text)}}
	text, res, err := s.chat(ctx, key, s.model(ctx), msgs, tietuVariantTokens)
	s.usage(p, res)
	if err != nil {
		return
	}
	var r struct {
		Changes []struct {
			From string `json:"from"`
			To   string `json:"to"`
			Why  string `json:"why"`
		} `json:"changes"`
	}
	if parseTietuJSON(text, &r) != nil {
		return
	}
	applied := 0
	for _, c := range r.Changes {
		from, to := strings.TrimSpace(c.From), strings.TrimSpace(c.To)
		fr, tr := []rune(from), []rune(to)
		if applied >= 3 || from == to || len(fr) < 2 || len(fr) > 8 || len(fr) != len(tr) || shihuaCore(from) != from || shihuaCore(to) != to {
			continue
		}
		diff := 0
		for i := range fr {
			if fr[i] != tr[i] {
				diff++
			}
		}
		if diff > 2 || strings.Count(po.Text, from) != 1 || tietuDecided(po.Variants, from) {
			continue
		}
		po.Text = strings.Replace(po.Text, from, to, 1)
		po.Variants = append(po.Variants, TietuVariant{Options: []string{from, to}, Chosen: to, Context: from,
			Why: "通行本写法（AI 提示）" + tietuNoteSuffix(articleClip(strings.TrimSpace(c.Why), 40))})
		applied++
	}
}

// tietuDecided: a place the source comparison already settled (「亡赖」 picked over 「无赖」) is not
// changed again by the common-reading pass.
func tietuDecided(variants []TietuVariant, from string) bool {
	for _, v := range variants {
		for _, o := range v.Options {
			if o != "（无）" && o != "" && strings.Contains(from, o) {
				return true
			}
		}
	}
	return false
}

type tietuRun struct{ ai, aj, bi, bj int }

// tietuAlign aligns two texts (longest common subsequence) and returns the stretches that differ;
// false when the texts are too long to align here.
func tietuAlign(a, b []rune) ([]tietuRun, bool) {
	n, m := len(a), len(b)
	if n*m > 4_000_000 {
		return nil, false
	}
	// lcs[i][j]: LCS length of a[i:] and b[j:].
	lcs := make([][]int32, n+1)
	for i := range lcs {
		lcs[i] = make([]int32, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = lcs[i+1][j]
				if lcs[i][j+1] > lcs[i][j] {
					lcs[i][j] = lcs[i][j+1]
				}
			}
		}
	}
	var out []tietuRun
	i, j := 0, 0
	open := false
	var cur tietuRun
	flush := func() {
		if open {
			cur.aj, cur.bj = i, j
			out = append(out, cur)
			open = false
		}
	}
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			flush()
			i++
			j++
		default:
			if !open {
				cur, open = tietuRun{ai: i, bi: j}, true
			}
			if j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]) {
				j++
			} else {
				i++
			}
		}
	}
	flush()
	return out, true
}

// tietuHints are search hints for a work the library may file under another title.
type tietuHints struct {
	Title     string   `json:"title"`
	Aliases   []string `json:"aliases"`
	Author    string   `json:"author"`
	Dynasty   string   `json:"dynasty"`
	FirstLine string   `json:"first_line"`
}

func (s *TietuService) resolveHints(ctx context.Context, p *TietuProject, key string) *tietuHints {
	msgs := []LearnMessage{{Role: "system", Content: `用户想找一首古诗文。把输入解析成检索线索：最通行的篇名（语文课本里的叫法）、其他常见篇名或别称、作者（民歌写通行署名，如「北朝民歌」）、朝代、第一句（不含标点，只用来检索和核对）。只输出一个 JSON 对象：{"title":"","aliases":[""],"author":"","dynasty":"","first_line":""}`},
		{Role: "user", Content: p.Brief.Input}}
	text, res, err := s.chat(ctx, key, s.model(ctx), msgs, tietuVariantTokens)
	s.usage(p, res)
	if err != nil {
		return nil
	}
	var h tietuHints
	if parseTietuJSON(text, &h) != nil {
		return nil
	}
	h.Title, h.Author, h.Dynasty, h.FirstLine = strings.TrimSpace(h.Title), strings.TrimSpace(h.Author), strings.TrimSpace(h.Dynasty), strings.TrimSpace(h.FirstLine)
	if h.Title == "" && h.FirstLine == "" {
		return nil
	}
	return &h
}

// findByHints searches the library by every title and the first line, and keeps a work whose text
// starts with the first line (or, without one, whose title and author match).
func (s *TietuService) findByHints(ctx context.Context, h *tietuHints) *LibPoem {
	if s.poems == nil {
		return nil
	}
	first := shihuaCore(h.FirstLine)
	queries := []string{}
	for _, t := range append([]string{h.Title}, h.Aliases...) {
		if t = strings.TrimSpace(t); t != "" && len(queries) < 5 {
			queries = append(queries, t)
		}
	}
	if r := []rune(first); len(r) >= 4 {
		queries = append(queries, string(r[:min(len(r), 7)]))
	}
	var best *LibPoem
	bestScore := 0
	for _, q := range queries {
		list, _, err := s.poems.SearchPoems(ctx, LibPoemQuery{Q: q, Limit: 20})
		if err != nil {
			continue
		}
		for i := range list {
			lp := list[i]
			score := 0
			if first != "" && strings.HasPrefix(shihuaCore(lp.Content), first) {
				score += 6
			}
			if h.Author != "" && lp.Author == h.Author {
				score += 2
			}
			if tietuShortTitle(lp.Title) == h.Title || tietuIn(tietuShortTitle(lp.Title), h.Aliases) {
				score += 2
			}
			if lp.Famous {
				score++
			}
			if first == "" && score < 4 {
				continue
			}
			if first != "" && score < 6 {
				continue
			}
			if score > bestScore || (score == bestScore && best != nil && lp.Popularity > best.Popularity) {
				best, bestScore = &list[i], score
			}
		}
	}
	return best
}

func tietuIn(s string, list []string) bool {
	for _, x := range list {
		if strings.TrimSpace(x) == s {
			return true
		}
	}
	return false
}

type tietuChoice struct {
	pick int // 0 library, 1 维基文库
	why  string
}

func (s *TietuService) pickVariants(ctx context.Context, p *TietuProject, po *TietuPoem, runs []tietuRun, a, b []rune, key string) map[int]tietuChoice {
	type item struct {
		Index   int      `json:"index"`
		Context string   `json:"context"`
		Options []string `json:"options"`
	}
	items := []item{}
	for i, rn := range runs {
		items = append(items, item{Index: i, Context: tietuContext(a, rn.ai, rn.aj), Options: []string{tietuShow(string(a[rn.ai:rn.aj])), tietuShow(string(b[rn.bi:rn.bj]))}})
	}
	req, _ := json.Marshal(map[string]any{"poem": po.Title + " · " + po.Author, "differences": items})
	text, res, err := s.chat(ctx, key, s.model(ctx), []LearnMessage{{Role: "system", Content: tietuVariantPrompt()}, {Role: "user", Content: string(req)}}, tietuVariantTokens)
	s.usage(p, res)
	out := map[int]tietuChoice{}
	if err != nil {
		return out
	}
	var r tietuVariantReply
	if parseTietuJSON(text, &r) != nil {
		return out
	}
	for _, c := range r.Choices {
		if c.Pick == 0 || c.Pick == 1 {
			out[c.Index] = tietuChoice{pick: c.Pick, why: articleClip(strings.TrimSpace(c.Why), 40)}
		}
	}
	return out
}

func tietuContext(a []rune, from, to int) string {
	l, r := max(0, from-4), min(len(a), to+4)
	return string(a[l:from]) + "【" + string(a[from:to]) + "】" + string(a[to:r])
}

// findLibPoem looks the input up in the poem library: the full input first, then without the last
// word (an author), preferring an exact title, then famous and popular works.
func (s *TietuService) findLibPoem(ctx context.Context, input string) *LibPoem {
	if s.poems == nil {
		return nil
	}
	words := strings.Fields(strings.NewReplacer("《", " ", "》", " ", "，", " ", ",", " ", "·", " ").Replace(input))
	if len(words) == 0 {
		return nil
	}
	tries := []string{strings.Join(words, " ")}
	if len(words) > 1 {
		tries = append(tries, words[0])
	}
	for _, q := range tries {
		list, _, err := s.poems.SearchPoems(ctx, LibPoemQuery{Q: q, Limit: 20})
		if err != nil || len(list) == 0 {
			continue
		}
		want := words[0]
		sort.SliceStable(list, func(i, j int) bool {
			si, sj := tietuTitleScore(list[i], want, words), tietuTitleScore(list[j], want, words)
			if si != sj {
				return si > sj
			}
			if list[i].Famous != list[j].Famous {
				return list[i].Famous
			}
			return list[i].Popularity > list[j].Popularity
		})
		if tietuTitleScore(list[0], want, words) > 0 {
			return &list[0]
		}
	}
	return nil
}

func tietuTitleScore(p LibPoem, want string, words []string) int {
	score := 0
	t := strings.NewReplacer("・", "·").Replace(p.Title)
	switch {
	case tietuShortTitle(t) == want || t == want:
		score += 4
	case strings.Contains(t, want):
		score += 2
	}
	for _, w := range words[1:] {
		if p.Author == w || strings.Contains(t, w) {
			score++
		}
	}
	return score
}

// tietuShortTitle drops the library's duplicated tune names (「江神子・江城子·老夫聊发少年狂」).
func tietuShortTitle(t string) string {
	t = strings.NewReplacer("・", "·").Replace(strings.TrimSpace(t))
	if parts := strings.Split(t, "·"); len(parts) == 3 {
		return parts[1] + "·" + parts[2]
	}
	return t
}

// tietuDropNotes removes notes the library keeps inside the text, like 「(间隔一作：隔绝)」.
func tietuDropNotes(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(', '（':
			depth++
			continue
		case ')', '）':
			if depth > 0 {
				depth--
				continue
			}
		}
		if depth == 0 {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

// findWikiText searches 维基文库 for the work and cuts it out by its first and last characters.
func (s *TietuService) findWikiText(ctx context.Context, po *TietuPoem) (string, string) {
	if s.wiki == nil {
		return "", ""
	}
	core := []rune(shihuaCore(po.Text))
	if len(core) < 8 {
		return "", ""
	}
	first, last := string(core[:4]), string(core[len(core)-3:])
	titles := []string{}
	for _, q := range []string{po.Title + " " + po.Author, string(core[:7])} {
		hits, err := s.wiki.search(ctx, q)
		if err != nil {
			continue
		}
		titles = append(titles, hits...)
	}
	tried := 0
	for _, t := range titles {
		if tried >= 6 {
			break
		}
		tried++
		page, ok, err := s.wiki.pageText(ctx, t)
		if err != nil || !ok {
			continue
		}
		cut, found := shihuaCutText(page, first, last)
		if !found {
			continue
		}
		// A cut much longer than the work is another text that happens to contain both ends.
		if n := utf8.RuneCountInString(shihuaCore(cut)); n > len(core)*5/4+4 {
			continue
		}
		tidy, _ := shihuaTidyText(cut, po.Text)
		return tidy, s.wiki.pageURL(t)
	}
	return "", ""
}

func tietuPoemSummary(po *TietuPoem) string {
	if po.Agreed && len(po.Variants) == 0 {
		return fmt.Sprintf("原文核对好了：《%s》%s，诗词库和维基文库一致", po.Title, po.Author)
	}
	if po.Agreed {
		return fmt.Sprintf("原文核对好了：《%s》%s，%d 处异文按通行写法取", po.Title, po.Author, len(po.Variants))
	}
	return fmt.Sprintf("原文取自诗词库：《%s》%s（%s）", po.Title, po.Author, po.Note)
}
