package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Automatic checks on each platform version (内容引擎). Errors are things the platform rejects or
// that break the rules (title too long, a link where none is allowed); warnings are for the
// reviewer (套话, absolute claims, a version too close to another). Two things are fixed on the
// spot: links are taken out where a platform allows none, and a promo-code line is added where it
// is missing.

const (
	contentPlaceholderLink  = "{{link}}"
	contentPlaceholderPromo = "{{promo}}"
	contentBannedWarnAt     = 3    // 套话 hits before a warning
	contentSimilarWarnAt    = 0.72 // bigram containment between two versions
	contentHistoryWarnAt    = 0.6  // title bigram similarity with an earlier article
)

// contentBannedPhrases read as machine-written in Chinese posts; the humanize step is told to avoid
// them and the check counts what is left.
var contentBannedPhrases = []*regexp.Regexp{
	regexp.MustCompile(`值得注意的是`), regexp.MustCompile(`总而言之`), regexp.MustCompile(`综上所述`), regexp.MustCompile(`总的来说`),
	regexp.MustCompile(`在当今`), regexp.MustCompile(`随着.{0,12}的(快速|飞速|迅猛|不断)?发展`), regexp.MustCompile(`赋能`), regexp.MustCompile(`助力`),
	regexp.MustCompile(`不可或缺`), regexp.MustCompile(`至关重要`), regexp.MustCompile(`深入探讨`), regexp.MustCompile(`让我们一起`),
	regexp.MustCompile(`在这个.{0,6}时代`), regexp.MustCompile(`一站式`), regexp.MustCompile(`全方位`), regexp.MustCompile(`无缝`),
	regexp.MustCompile(`扮演着.{0,8}角色`), regexp.MustCompile(`日新月异`), regexp.MustCompile(`毋庸置疑`), regexp.MustCompile(`众所周知`),
	regexp.MustCompile(`划时代`), regexp.MustCompile(`颠覆性`), regexp.MustCompile(`革命性`), regexp.MustCompile(`不仅.{1,20}更是`),
	regexp.MustCompile(`希望(本文|这篇文章)(能|对)`), regexp.MustCompile(`宝子们`), regexp.MustCompile(`家人们`), regexp.MustCompile(`干货满满`),
}

// contentRiskyWords: 广告法 absolute claims and words that get posts about API access taken down.
// (Words like 第一 or 绝对 are left out: 第一步 and 绝对路径 are everywhere in tutorials.)
var contentRiskyWords = []string{"最便宜", "最低价", "最稳定", "全网最", "全网第一", "行业第一", "顶级", "百分百", "绝对稳定", "国家级", "翻墙", "科学上网", "梯子", "VPN", "破解", "免费无限"}

var (
	contentURLPattern      = regexp.MustCompile(`(?i)(https?://[^\s)\]）】"'<>]+|www\.[a-z0-9-]+\.[a-z]{2,}[^\s)\]）】"'<>]*)`)
	contentMDLinkPattern   = regexp.MustCompile(`\[([^\]]*)\]\((https?://[^)]+|\{\{link\}\})\)`)
	contentMaterialPattern = regexp.MustCompile(`【素材卡[：:][^】]*】`)
)

// contentBannedHits lists the 套话 found in text.
func contentBannedHits(text string) []string {
	var hits []string
	for _, re := range contentBannedPhrases {
		hits = append(hits, re.FindAllString(text, 3)...)
	}
	return hits
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// checkContentVariant checks one version (and fixes links / the promo line in place); others are
// the article's other versions, for the similarity warning.
func checkContentVariant(v *ContentVariant, others []ContentVariant, brand string) []ContentCheck {
	spec, ok := contentPlatformSpecs[v.Platform]
	if !ok {
		return nil
	}
	var out []ContentCheck
	add := func(level, code, text string) { out = append(out, ContentCheck{Level: level, Code: code, Text: text}) }

	if !spec.Links {
		if stripped, n := stripContentLinks(v.Body); n > 0 {
			v.Body = stripped
			add("info", "links_removed", fmt.Sprintf("%s正文不能放外链，已删掉 %d 处链接", spec.Name, n))
		}
		if v.Micro != "" {
			if stripped, n := stripContentLinks(v.Micro); n > 0 {
				v.Micro = stripped
				add("info", "links_removed", "微头条里的链接已删掉")
			}
		}
	}
	if spec.Promo && !strings.Contains(v.Body, contentPlaceholderPromo) {
		v.Body = strings.TrimRight(v.Body, "\n ") + "\n\n" + fmt.Sprintf("想试试的话，搜「%s」注册，填优惠码 %s 多送体验额度。", brand, contentPlaceholderPromo)
		add("info", "promo_added", "文末补上了优惠码（这个平台靠优惠码统计带来的注册）")
	}

	switch n := runeLen(v.Title); {
	case n == 0:
		add("error", "title_empty", "标题是空的")
	case n > spec.TitleMax:
		add("error", "title_long", fmt.Sprintf("标题 %d 字，%s最多 %d 字", n, spec.Name, spec.TitleMax))
	}
	if v.Platform == ContentPlatformZhihu && v.Kind == "answer" && strings.TrimSpace(v.Question) == "" {
		add("warn", "question_empty", "知乎回答还没写要去回答哪个问题")
	}
	plain := contentPlainText(v.Body)
	switch n := runeLen(plain); {
	case n > spec.BodyMax:
		add("error", "body_long", fmt.Sprintf("正文约 %d 字，%s最多 %d 字", n, spec.Name, spec.BodyMax))
	case n < spec.BodyMin:
		add("warn", "body_short", fmt.Sprintf("正文约 %d 字，%s建议至少 %d 字", n, spec.Name, spec.BodyMin))
	}
	if spec.Links && !strings.Contains(v.Body, contentPlaceholderLink) {
		add("warn", "link_missing", "正文没有放引流链接（{{link}}），带来的访问统计不到")
	}
	if spec.MicroMax > 0 {
		if n := runeLen(v.Micro); n > spec.MicroMax {
			add("error", "micro_long", fmt.Sprintf("微头条 %d 字，最多 %d 字", n, spec.MicroMax))
		}
	}
	if spec.TagsMax > 0 && (len(v.Tags) < spec.TagsMin || len(v.Tags) > spec.TagsMax) {
		add("warn", "tags", fmt.Sprintf("标签 %d 个，%s建议 %d–%d 个", len(v.Tags), spec.Name, spec.TagsMin, spec.TagsMax))
	}
	if spec.Cards {
		switch n := len(v.Cards); {
		case n < 3 || n > 8:
			add("warn", "cards", fmt.Sprintf("图卡 %d 张，小红书建议 3–8 张", n))
		}
		for i, c := range v.Cards {
			if runeLen(c.Title) > 18 {
				add("warn", "card_title", fmt.Sprintf("第 %d 张图卡标题太长，图上放不下", i+1))
			}
		}
	}
	if m := contentMaterialPattern.FindAllString(v.Body, 5); len(m) > 0 {
		add("warn", "material", "有待补充的一手素材："+strings.Join(m, " "))
	}
	if hits := contentBannedHits(v.Body); len(hits) >= contentBannedWarnAt {
		add("warn", "ai_phrases", "AI 套话偏多："+strings.Join(contentUnique(hits, 8), "、"))
	}
	if words := contentRiskyHits(v.Title + "\n" + v.Body + "\n" + v.Micro); len(words) > 0 {
		add("warn", "risky_words", "有广告法极限词或敏感词，建议改掉："+strings.Join(words, "、"))
	}
	for _, o := range others {
		if o.ID == v.ID || o.Platform == v.Platform || o.Status == ContentVariantFailed || o.Body == "" {
			continue
		}
		if sim := contentContainment(plain, contentPlainText(o.Body)); sim >= contentSimilarWarnAt {
			add("warn", "similar", fmt.Sprintf("和%s版本太像（%d%%），多平台同发可能被判搬运", contentPlatformSpecs[o.Platform].Name, int(sim*100)))
		}
	}
	if out == nil {
		out = []ContentCheck{}
	}
	return out
}

func contentRiskyHits(text string) []string {
	var out []string
	lower := strings.ToLower(text)
	for _, w := range contentRiskyWords {
		if strings.Contains(lower, strings.ToLower(w)) {
			out = append(out, w)
		}
	}
	return out
}

func contentUnique(in []string, limit int) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] && len(out) < limit {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// stripContentLinks removes URLs and the {{link}} placeholder; Markdown links keep their text.
func stripContentLinks(body string) (string, int) {
	n := 0
	body = contentMDLinkPattern.ReplaceAllStringFunc(body, func(m string) string {
		n++
		return contentMDLinkPattern.FindStringSubmatch(m)[1]
	})
	body = contentURLPattern.ReplaceAllStringFunc(body, func(string) string {
		n++
		return ""
	})
	if c := strings.Count(body, contentPlaceholderLink); c > 0 {
		n += c
		body = strings.ReplaceAll(body, contentPlaceholderLink, "")
	}
	return body, n
}

// renderContentBody fills in the link and promo code; placeholders stay until they exist.
func renderContentBody(body, link, promo string) string {
	if link != "" {
		body = strings.ReplaceAll(body, contentPlaceholderLink, link)
	}
	if promo != "" {
		body = strings.ReplaceAll(body, contentPlaceholderPromo, promo)
	}
	return body
}

var contentMDNoise = regexp.MustCompile("(?m)^#{1,6}\\s+|[*_`>|]|!\\[[^\\]]*\\]\\([^)]*\\)")

// contentPlainText drops Markdown marks and blank space, so lengths count what a reader sees.
func contentPlainText(md string) string {
	md = contentMDNoise.ReplaceAllString(md, "")
	var b strings.Builder
	for _, r := range md {
		if !unicode.IsSpace(r) {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

// contentBigrams: character pairs of the letters, digits and Han characters in s.
func contentBigrams(s string) map[string]struct{} {
	var rs []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			rs = append(rs, r)
		}
	}
	out := make(map[string]struct{}, len(rs))
	for i := 0; i+1 < len(rs); i++ {
		out[string(rs[i:i+2])] = struct{}{}
	}
	return out
}

// contentContainment: the share of the shorter text's bigrams found in the longer one — a short
// version lifted from a long one scores high even though the lengths differ.
func contentContainment(a, b string) float64 {
	ba, bb := contentBigrams(a), contentBigrams(b)
	if len(ba) == 0 || len(bb) == 0 {
		return 0
	}
	if len(ba) > len(bb) {
		ba, bb = bb, ba
	}
	common := 0
	for g := range ba {
		if _, ok := bb[g]; ok {
			common++
		}
	}
	return float64(common) / float64(len(ba))
}

// contentJaccard: bigram overlap of two short texts (titles).
func contentJaccard(a, b string) float64 {
	ba, bb := contentBigrams(a), contentBigrams(b)
	if len(ba) == 0 || len(bb) == 0 {
		return 0
	}
	common := 0
	for g := range ba {
		if _, ok := bb[g]; ok {
			common++
		}
	}
	return float64(common) / float64(len(ba)+len(bb)-common)
}

// checkContentMaster checks the master article: an earlier article with nearly the same title,
// leftover 套话 and missing first-hand material.
func checkContentMaster(it *ContentItem, earlier []string) []ContentCheck {
	out := []ContentCheck{}
	title := it.Title
	for _, e := range earlier {
		if e == "" || e == title {
			continue
		}
		if sim := contentJaccard(title, e); sim >= contentHistoryWarnAt {
			out = append(out, ContentCheck{Level: "warn", Code: "repeat", Text: fmt.Sprintf("和以前的「%s」很像（%d%%），注意别重复", truncateRunes(e, 30), int(sim*100))})
			break
		}
	}
	if hits := contentBannedHits(it.Markdown); len(hits) >= contentBannedWarnAt {
		out = append(out, ContentCheck{Level: "warn", Code: "ai_phrases", Text: "母稿 AI 套话偏多：" + strings.Join(contentUnique(hits, 8), "、")})
	}
	if m := contentMaterialPattern.FindAllString(it.Markdown, 5); len(m) > 0 {
		out = append(out, ContentCheck{Level: "warn", Code: "material", Text: "母稿有待补充的一手素材：" + strings.Join(m, " ")})
	}
	return out
}

func cleanContentTags(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, t := range in {
		t = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(t), "#＃"))
		if t == "" || seen[t] || len(out) >= 10 {
			continue
		}
		seen[t] = true
		out = append(out, truncateRunes(t, 30))
	}
	return out
}

func cleanContentCards(in []ContentCard) []ContentCard {
	out := []ContentCard{}
	for _, c := range in {
		title := truncateRunes(strings.TrimSpace(c.Title), 40)
		points := []string{}
		for _, p := range c.Points {
			if p = truncateRunes(strings.TrimSpace(p), 80); p != "" && len(points) < 6 {
				points = append(points, p)
			}
		}
		if title == "" && len(points) == 0 {
			continue
		}
		if len(out) < 9 {
			out = append(out, ContentCard{Title: title, Points: points})
		}
	}
	return out
}
