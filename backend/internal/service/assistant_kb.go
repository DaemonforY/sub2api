package service

import (
	_ "embed"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The support assistant's knowledge: the site FAQ (assistant_faq.md) and every page of the learning
// site, cut into passages and searched by keyword — Chinese as character bigrams, everything else as
// words — with BM25. No embeddings: the corpus is small and mostly Chinese, and this needs no model.

//go:embed assistant_faq.md
var assistantFAQ string

// AssistantPage is one source document: a page of the learning site or an FAQ entry.
type AssistantPage struct {
	URL   string
	Title string
	Text  string
}

type assistantChunk struct {
	page  int // index into kb.pages
	text  string
	terms map[string]int
	title map[string]bool // terms of the page title
	size  int
}

type assistantKB struct {
	pages  []AssistantPage
	chunks []assistantChunk
	df     map[string]int
	avgLen float64
}

// assistantHit is a passage that matched a question. Coverage is the share of the question's
// keywords (weighted by rarity) found in the passage or its page title: low coverage = loosely related.
type assistantHit struct {
	URL      string
	Title    string
	Text     string
	Score    float64
	Coverage float64
}

const (
	assistantChunkChars = 600
	assistantBM25K1     = 1.2
	assistantBM25B      = 0.75
	// A page title says what the page is about: its terms count this many extra times in each passage,
	// and a question term in the title adds this share of its weight on top.
	assistantTitleTF    = 3
	assistantTitleBoost = 0.5
)

// parseAssistantFAQ reads the FAQ: "## 标题", an optional "链接：/path" line, then the answer.
func parseAssistantFAQ(md string) []AssistantPage {
	var out []AssistantPage
	var cur *AssistantPage
	flush := func() {
		if cur != nil && strings.TrimSpace(cur.Text) != "" {
			cur.Text = strings.TrimSpace(cur.Text)
			out = append(out, *cur)
		}
	}
	for _, line := range strings.Split(md, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			flush()
			cur = &AssistantPage{Title: strings.TrimSpace(line[3:]), URL: "/"}
		case cur == nil || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "链接："):
			cur.URL = strings.TrimSpace(strings.TrimPrefix(line, "链接："))
		default:
			cur.Text += line + "\n"
		}
	}
	flush()
	return out
}

// assistantTerms: lower-cased words for Latin text and digits, bigrams for runs of CJK characters
// (a lone CJK character counts as itself).
func assistantTerms(s string) []string {
	var out []string
	var word []rune
	var cjk []rune
	flushWord := func() {
		if len(word) >= 2 || (len(word) == 1 && unicode.IsDigit(word[0])) {
			out = append(out, string(word))
		}
		word = word[:0]
	}
	flushCJK := func() {
		switch {
		case len(cjk) == 1:
			out = append(out, string(cjk))
		case len(cjk) > 1:
			for i := 0; i+1 < len(cjk); i++ {
				out = append(out, string(cjk[i:i+2]))
			}
		}
		cjk = cjk[:0]
	}
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.Is(unicode.Han, r):
			flushWord()
			cjk = append(cjk, r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			flushCJK()
			word = append(word, r)
		default:
			flushWord()
			flushCJK()
		}
	}
	flushWord()
	flushCJK()
	return out
}

// Question words and particles: a question's bigrams across them (怎么配置 → 么配, 有什么用 → 有什)
// carry no topic, so they are left out of the question's keywords.
const assistantQueryNoise = "么吗呢吧的了哪啊呀嘛什怎"

var assistantQueryStop = map[string]bool{"如何": true, "可以": true, "请问": true, "一下": true, "你们": true, "我们": true, "这个": true, "那个": true}

func assistantQueryTerms(q string) []string {
	var out []string
	for _, t := range assistantTerms(q) {
		r := []rune(t)
		if assistantQueryStop[t] || (len(r) <= 2 && unicode.Is(unicode.Han, r[0]) && strings.ContainsAny(t, assistantQueryNoise)) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// assistantChunks cuts a page into passages of about assistantChunkChars, on line breaks.
func assistantChunks(text string) []string {
	var out []string
	var b strings.Builder
	n := 0
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		l := utf8.RuneCountInString(line)
		if n > 0 && n+l > assistantChunkChars {
			out = append(out, b.String())
			b.Reset()
			n = 0
		}
		// A single very long line (code, a table row) is cut hard.
		for l > assistantChunkChars*2 {
			r := []rune(line)
			out = append(out, string(r[:assistantChunkChars]))
			line = string(r[assistantChunkChars:])
			l = utf8.RuneCountInString(line)
		}
		if n > 0 {
			_ = b.WriteByte('\n')
		}
		_, _ = b.WriteString(line)
		n += l
	}
	if n > 0 {
		out = append(out, b.String())
	}
	return out
}

func newAssistantKB(pages []AssistantPage) *assistantKB {
	kb := &assistantKB{pages: pages, df: map[string]int{}}
	total := 0
	for i, p := range pages {
		titleTerms := assistantTerms(p.Title)
		title := map[string]bool{}
		for _, t := range titleTerms {
			title[t] = true
		}
		for _, text := range assistantChunks(p.Text) {
			terms := map[string]int{}
			size := 0
			for _, t := range assistantTerms(text) {
				terms[t]++
				size++
			}
			for _, t := range titleTerms {
				terms[t] += assistantTitleTF
				size += assistantTitleTF
			}
			if size == 0 {
				continue
			}
			for t := range terms {
				kb.df[t]++
			}
			kb.chunks = append(kb.chunks, assistantChunk{page: i, text: text, terms: terms, title: title, size: size})
			total += size
		}
	}
	if len(kb.chunks) > 0 {
		kb.avgLen = float64(total) / float64(len(kb.chunks))
	}
	return kb
}

// search returns up to limit passages for the query, best first, at most two per page.
func (kb *assistantKB) search(query string, limit int) []assistantHit {
	if kb == nil || len(kb.chunks) == 0 {
		return nil
	}
	n := float64(len(kb.chunks))
	seen := map[string]bool{}
	var q []string
	idfSum := 0.0 // over every keyword of the question, also those no passage has
	for _, t := range assistantQueryTerms(query) {
		if seen[t] {
			continue
		}
		seen[t] = true
		df := float64(kb.df[t])
		idfSum += math.Log(1 + (n-df+0.5)/(df+0.5))
		if df > 0 {
			q = append(q, t)
		}
	}
	if len(q) == 0 {
		return nil
	}
	idf := make([]float64, len(q))
	for j, t := range q {
		df := float64(kb.df[t])
		idf[j] = math.Log(1 + (n-df+0.5)/(df+0.5))
	}
	type scored struct {
		i        int
		score    float64
		coverage float64
	}
	var all []scored
	for i, c := range kb.chunks {
		score, matched := 0.0, 0.0
		for j, t := range q {
			tf := float64(c.terms[t])
			if tf == 0 {
				continue
			}
			matched += idf[j]
			score += idf[j] * tf * (assistantBM25K1 + 1) / (tf + assistantBM25K1*(1-assistantBM25B+assistantBM25B*float64(c.size)/kb.avgLen))
			if c.title[t] {
				score += idf[j] * assistantTitleBoost
			}
		}
		if score > 0 {
			all = append(all, scored{i, score, matched / idfSum})
		}
	}
	sort.Slice(all, func(a, b int) bool { return all[a].score > all[b].score })
	perPage := map[int]int{}
	var out []assistantHit
	for _, s := range all {
		c := kb.chunks[s.i]
		if perPage[c.page] >= 2 {
			continue
		}
		perPage[c.page]++
		p := kb.pages[c.page]
		out = append(out, assistantHit{URL: p.URL, Title: p.Title, Text: c.text, Score: s.score, Coverage: s.coverage})
		if len(out) >= limit {
			break
		}
	}
	return out
}

// AssistantFAQ is the site FAQ: each entry's question (Title), reference link (URL) and answer (Text).
func AssistantFAQ() []AssistantPage { return parseAssistantFAQ(assistantFAQ) }
