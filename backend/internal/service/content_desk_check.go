package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Fixed prose rules of the 写作台 check, run before the editor's read. The lists follow
// KKKKhazix/human-writing's check_prose.py and op7418/Humanizer-zh (both MIT): pivot sentences
// (翻案腔), nominalised verbs, signpost phrases, business jargon, dashes and prompt-style colons,
// plus the author's own taboos and leftover experience placeholders. Each hit quotes the exact
// text so the editor can find it.

// ContentProseStats are shape numbers shown with the check.
type ContentProseStats struct {
	Chars        int     `json:"chars"`
	Paras        int     `json:"paras"`
	SingleParas  int     `json:"single_paras"` // one-sentence paragraphs
	SingleRatio  float64 `json:"single_ratio"`
	Headings     int     `json:"headings"`
	Bullets      int     `json:"bullets"`
	Dashes       int     `json:"dashes"`
	Colons       int     `json:"colons"`
	Placeholders int     `json:"placeholders"`
}

type proseRule struct {
	re      *regexp.Regexp
	level   string
	problem string
}

var contentHardWords = []string{"说白了", "说穿了", "先说结论", "值得注意的是", "需要指出的是", "不难发现", "综上所述", "总的来说", "总而言之", "不可否认", "换句话说", "本质上",
	"意味着什么", "让我们来看看", "接下来让我们", "从某种意义上说", "更微妙的是", "还有一层",
	"赋能", "抓手", "底层逻辑", "顶层设计", "商业闭环", "价值闭环", "全链路", "降本增效", "认知跃迁", "组合拳", "打开想象空间", "结构性机会", "技术底座", "能力沉淀", "拉通", "深层逻辑"}

var contentProseRules = []proseRule{
	{regexp.MustCompile(`(?:并)?不是[^。！？\n]{0,60}而是`), "must", "翻案腔：先立一个读者没有的误解再推翻，直接从正面下判断"},
	{regexp.MustCompile(`并非[^。！？\n]{0,60}而是`), "must", "翻案腔：直接从正面下判断"},
	{regexp.MustCompile(`不在于[^。！？\n]{0,60}而在于`), "must", "翻案腔：直接说重点在哪"},
	{regexp.MustCompile(`与其说[^。！？\n]{0,60}(?:不如|倒不如)`), "must", "翻案腔：直接说你的判断"},
	{regexp.MustCompile(`(?:表面|看似)(?:上)?[^。！？\n]{0,60}(?:其实|实际|实则)`), "must", "翻案腔：「表面……其实……」"},
	{regexp.MustCompile(`(?:总|一直|都)?以为[^！？\n]{2,40}?(?:其实|才发现|才明白)`), "should", "「你以为……其实……」式反转，确认是不是真的走过了从误解到修正的过程"},
	{regexp.MustCompile(`回头(?:看|一看)?才(?:发现|明白)`), "should", "「回头才发现」是常见的 AI 反转句"},
	{regexp.MustCompile(`答案恰恰相反|恰恰相反`), "should", "「恰恰相反」式抬价"},
	{regexp.MustCompile(`真正[^，。！？\n]{0,12}的(?:，)?是`), "should", "「真正……的是」式抬价，直接说"},
	{regexp.MustCompile(`进行(?:了|一次|一场)?[^。，！？\n]{0,8}(?:调整|优化|升级|分析|讨论|沟通|梳理|复盘|迭代|探索|尝试|思考|规划)`), "should", "动词名词化：「进行优化」写成「改」「调」"},
	{regexp.MustCompile(`实现了?[^。，！？\n]{0,12}的?(?:提升|增长|突破|跃升|落地)`), "should", "动词名词化：写清快了多少、多了多少"},
	{regexp.MustCompile(`完成了?对[^。，！？\n]{0,14}的`), "should", "动词名词化：「完成了对……的……」"},
	{regexp.MustCompile(`起到了?[^。，！？\n]{0,10}的?作用|具有[^。，！？\n]{0,8}(?:意义|价值)`), "should", "空泛的「起到作用 / 具有意义」，写具体"},
	{regexp.MustCompile(`在当今[^。！？\n]{0,12}(?:时代|社会|世界)|随着[^。！？\n]{0,14}的(?:快速|飞速|迅猛|不断)?发展`), "must", "教科书式开头，换成一件具体的事"},
	{regexp.MustCompile(`——|—|–`), "must", "破折号：改成逗号、句号，或者拆成两句"},
}

// contentPromptColon: a colon that introduces a list or a summary, not someone's words.
var contentPromptColon = regexp.MustCompile(`([^\n。！？“”「」]{0,12})[：:]\s*(?:\n|$)|(?:一句话总结|核心是|关键是|结论是|答案是|原因是|重点是|总结一下)[：:]`)

var contentQuoteVerbs = regexp.MustCompile(`(?:说|问|道|讲|答|喊|写着|写道|回复|提示|报错)\s*$`)

var (
	contentSentenceEnd   = regexp.MustCompile(`[。！？!?…]+`)
	contentNumbered      = regexp.MustCompile(`^\d+[.、]`)
	contentPlaceholderRe = regexp.MustCompile(`【这里需要你的真实经历[^】]*】`)
)

// checkProse returns the shape numbers and the rule hits of a Markdown article.
func checkProse(md string, taboos []string) (ContentProseStats, []ContentIssue) {
	var stats ContentProseStats
	var hits []ContentIssue
	seen := map[string]bool{}
	add := func(quote, level, problem string) {
		quote = strings.TrimSpace(quote)
		if quote == "" || seen[quote+problem] || len(hits) >= 40 {
			return
		}
		seen[quote+problem] = true
		hits = append(hits, ContentIssue{Layer: "rule", Level: level, Quote: quote, Problem: problem})
	}

	inCode := false
	var prose strings.Builder
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		switch {
		case strings.HasPrefix(t, "#"):
			if !strings.HasPrefix(t, "# ") {
				stats.Headings++
			}
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || contentNumbered.MatchString(t):
			stats.Bullets++
		}
		_, _ = prose.WriteString(line + "\n")
	}
	text := prose.String()
	stats.Chars = utf8.RuneCountInString(contentPlainText(text))

	for _, p := range contentParas(text) {
		if strings.HasPrefix(p, "#") {
			continue
		}
		stats.Paras++
		sentences := 0
		for _, s := range contentSentenceEnd.Split(p, -1) {
			if strings.TrimSpace(s) != "" {
				sentences++
			}
		}
		if sentences <= 1 {
			stats.SingleParas++
		}
	}
	if stats.Paras > 0 {
		stats.SingleRatio = float64(stats.SingleParas) / float64(stats.Paras)
	}

	for _, w := range contentHardWords {
		if strings.Contains(text, w) {
			add(w, "must", "套话 / 黑话「"+w+"」，换成普通的说法或者删掉")
		}
	}
	for _, t := range taboos {
		t = strings.TrimSpace(t)
		if utf8.RuneCountInString(t) >= 2 && utf8.RuneCountInString(t) <= 20 && strings.Contains(text, t) {
			add(t, "must", "你的作者档案里写了不这样写：「"+t+"」")
		}
	}
	for _, r := range contentProseRules {
		for _, m := range r.re.FindAllString(text, 5) {
			if strings.HasPrefix(r.problem, "破折号") {
				stats.Dashes++
			}
			add(m, r.level, r.problem)
		}
	}
	for _, loc := range contentPromptColon.FindAllStringSubmatchIndex(text, 10) {
		m := text[loc[0]:loc[1]]
		before := ""
		if loc[2] >= 0 {
			before = text[loc[2]:loc[3]]
		}
		if contentQuoteVerbs.MatchString(before) {
			continue
		}
		stats.Colons++
		add(strings.TrimRight(m, "\n "), "should", "提示性冒号：把要说的直接说出来，或者改成逗号")
	}
	for _, m := range contentPlaceholderRe.FindAllString(text, 20) {
		stats.Placeholders++
		add(m, "must", "这里等你补一段真实经历，补完或删掉")
	}
	if stats.Paras >= 8 && stats.SingleRatio > 0.4 {
		hits = append(hits, ContentIssue{Layer: "rule", Level: "should", Problem: fmt.Sprintf("单句成段太多（%d / %d 段），单句段只留给真正需要停一下的地方", stats.SingleParas, stats.Paras)})
	}
	if stats.Headings > 6 || stats.Bullets > 12 {
		hits = append(hits, ContentIssue{Layer: "rule", Level: "should", Problem: fmt.Sprintf("小标题 %d 个、列表 %d 行，读起来像说明书；除非是教程，少用小标题和列表，靠转场句往下推", stats.Headings, stats.Bullets)})
	}
	if hits == nil {
		hits = []ContentIssue{}
	}
	return stats, hits
}
