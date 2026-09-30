package service

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Automatic classification of prompt-library entries coming from community sources. It mirrors
// web/src/lib/prompt-taxonomy.ts in the canvas (same scenes, same rules) so items look the same
// whether the canvas classifies them locally or reads them from here. Admin edits override it.

// PromptScenes is the fixed scene list shown as filter chips in the canvas.
var PromptScenes = []string{"poster", "ecommerce", "ui", "infographic", "portrait", "photo", "illustration", "3d", "brand", "social", "life", "history", "creative", "video", "other"}

var promptSceneSet = func() map[string]bool {
	set := make(map[string]bool, len(PromptScenes))
	for _, scene := range PromptScenes {
		set[scene] = true
	}
	return set
}()

// IsPromptScene reports whether s is one of PromptScenes.
func IsPromptScene(s string) bool { return promptSceneSet[s] }

type promptSceneRule struct {
	scene   string
	pattern *regexp.Regexp
}

// Order matters only for the primary scene (first match); an entry can belong to several scenes.
var promptSceneRules = []promptSceneRule{
	{"video", regexp.MustCompile(`(?i)视频模板|视频生成|seedance|short_video|dance_action|vfx_fantasy|^animation$|wuxia_history`)},
	{"ecommerce", regexp.MustCompile(`(?i)commerce|电商|product|产品|营销|主图|advertis|广告|包装|packaging`)},
	{"ui", regexp.MustCompile(`(?i)(^|[^a-z])ui([^a-z]|$)|(^|[^a-z])ux([^a-z]|$)|界面|interface|游戏截图|网页|\bapp\b`)},
	{"infographic", regexp.MustCompile(`(?i)infographic|信息图|chart|图表|ppt|document|文档|education|教育|学习|总结|article|publishing|手帐|笔记|流程图|blueprint|schematic`)},
	{"poster", regexp.MustCompile(`(?i)poster|海报|typograph|字体|排版|封面|thumbnail|缩略图|text_render|文字渲染|social_poster`)},
	{"brand", regexp.MustCompile(`(?i)logo|brand|品牌`)},
	{"portrait", regexp.MustCompile(`(?i)portrait|头像|肖像|个人资料|character|角色|人像|cosplay|穿搭|fashion|people`)},
	{"photo", regexp.MustCompile(`(?i)photo|摄影|realistic|写实|cinematic|电影`)},
	{"illustration", regexp.MustCompile(`(?i)illustration|插画|anime|动漫|漫画|comic|故事板|story|卡牌|\bcard\b|\bart\b`)},
	{"3d", regexp.MustCompile(`(?i)(^|[^a-z])3d|手办|潮玩|材质|微缩|微型|立体|雕塑|浮雕|sculpture|figure|diorama|isometric|等距|claymation|low poly|盲盒|键帽|keycap`)},
	{"social", regexp.MustCompile(`(?i)social|社交|表情包|meme|youtube`)},
	{"life", regexp.MustCompile(`(?i)travel|旅游|food|美食|生活|装修|architecture|建筑|空间|风景|家居`)},
	{"history", regexp.MustCompile(`(?i)history|historic|古风|classical|ancient|武侠|历史|国风|古书|中式`)},
	{"creative", regexp.MustCompile(`(?i)有趣|creative|想象|funny|吐槽|imagin|脑洞|game|游戏|胶囊|capsule|涂鸦|邮票|stamp|明信片|postcard|passport`)},
}

// Coarse labels one source stamps on almost everything; they only count when nothing more specific matched.
var promptWeakTags = map[string]bool{"commerce": true, "tech": true}

var (
	promptNSFW = regexp.MustCompile(`(?i)nsfw|18\+|r18|成人|色情|nude|naked`)
	// Real public figures and national symbols: unsuitable for a campus product and the usual cause
	// of upstream moderation rejections. "名人名言" (quotes) is removed before matching "名人".
	promptSensitive      = regexp.MustCompile(`(?i)特朗普|川普|\btrump\b|拜登|\bbiden\b|普京|\bputin\b|泽连斯基|zelensky|金正恩|马斯克|elon musk|sam altman|习近平|毛泽东|毛主席|斯大林|stalin|列宁|lenin|希特勒|hitler|领导人|国家主席|国旗|national flag|国徽|总统|\bpresident\b|首相|名人|celebrit|政治|politic|纳粹|\bnazi\b|propaganda`)
	promptNeedsReference = regexp.MustCompile(`(?i)需要参考图|参考图|upload (a|an|your|the)|上传(一张|你的|照片|图片)|based on the (uploaded|attached)|attached (photo|image)`)
	promptCJK            = regexp.MustCompile(`[\x{3400}-\x{9FFF}]`)
	promptGenericTitle   = regexp.MustCompile(`(?i)^(图像模板|视频模板)\s*-|^(其他|other|unknown|untitled|无标题)$`)
	promptLangMarker     = regexp.MustCompile(`(?i)[\[【](中文|english|en|zh|cn)[\]】]\s*`)
	promptSpaces         = regexp.MustCompile(`\s+`)
	promptQuoted         = regexp.MustCompile(`[「『“"]([^」』”"]{2,60})[」』”"]`)
	promptPlaceholder    = regexp.MustCompile(`^[xX]+$|^【?XXX`)
	promptLeadingVerb    = regexp.MustCompile(`^(?:请|帮我|麻烦)?(?:生成|创建|设计|制作|画|绘制)?(?:一张|一幅|一个|一组)?`)
	promptClauseSplit    = regexp.MustCompile(`[，。,.:：；;!！?？（(\n]`)
	promptModelGPTImage2 = regexp.MustCompile(`gpt-?image-?2|gpt_image_2`)
	promptModelBanana    = regexp.MustCompile(`nano[- ]?banana|gemini`)
	promptModelGPT4o     = regexp.MustCompile(`gpt-?4o`)
)

// PromptTraits is the automatic classification of one entry.
type PromptTraits struct {
	Scenes         []string
	Model          string
	NSFW           bool
	Sensitive      bool
	NeedsReference bool
	Lang           string
}

// PromptTraitInput is what the classifier looks at.
type PromptTraitInput struct {
	Title       string
	Prompt      string
	Description string
	Tags        []string
	// ModelHint is the source's image model field (or any text naming the model).
	ModelHint string
	// SceneHints are scenes the source itself assigns (e.g. YouMind's use-case files).
	SceneHints []string
}

// DetectPromptModel maps free-form model mentions to the canvas' model buckets.
func DetectPromptModel(tags []string, hint string) string {
	value := strings.ToLower(hint + " " + strings.Join(tags, " "))
	switch {
	case promptModelGPTImage2.MatchString(value):
		return "gpt-image-2"
	case promptModelBanana.MatchString(value):
		return "nano-banana"
	case promptModelGPT4o.MatchString(value):
		return "gpt-4o"
	default:
		return "unknown"
	}
}

func matchPromptScenes(texts []string) []string {
	var scenes []string
	for _, rule := range promptSceneRules {
		for _, text := range texts {
			if rule.pattern.MatchString(text) {
				scenes = append(scenes, rule.scene)
				break
			}
		}
	}
	return scenes
}

// ClassifyPrompt returns the automatic traits of an entry.
func ClassifyPrompt(in PromptTraitInput) PromptTraits {
	tags := make([]string, 0, len(in.Tags))
	specific := make([]string, 0, len(in.Tags))
	for _, tag := range in.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
		if !promptWeakTags[strings.ToLower(tag)] {
			specific = append(specific, tag)
		}
	}
	// Match scenes on tags first; fall back to the title when the tags say nothing useful.
	scenes := matchPromptScenes(specific)
	if len(scenes) == 0 && in.Title != "" {
		scenes = matchPromptScenes([]string{in.Title})
	}
	if len(scenes) == 0 {
		scenes = matchPromptScenes(tags)
	}
	scenes = mergePromptSceneHints(in.SceneHints, scenes)
	if len(scenes) == 0 {
		scenes = []string{"other"}
	}
	joined := strings.Join(tags, " ") + " " + in.Title + " " + in.Description
	sensitiveText := strings.ReplaceAll(joined+" "+in.Prompt, "名人名言", "")
	head := in.Prompt
	if utf8.RuneCountInString(head) > 400 {
		head = string([]rune(head)[:400])
	}
	lang := "en"
	if promptCJK.MatchString(in.Prompt) {
		lang = "zh"
	}
	return PromptTraits{
		Scenes:         scenes,
		Model:          DetectPromptModel(tags, in.ModelHint),
		NSFW:           promptNSFW.MatchString(joined),
		Sensitive:      promptSensitive.MatchString(sensitiveText),
		NeedsReference: promptNeedsReference.MatchString(strings.Join(tags, " ") + " " + in.Description + " " + head),
		Lang:           lang,
	}
}

// Source hints such as "social" or "creative" are catch-alls; specific rule matches go first then.
var promptWeakSceneHints = map[string]bool{"social": true, "creative": true, "other": true}

func mergePromptSceneHints(hints, matched []string) []string {
	var strong, weak []string
	for _, hint := range hints {
		if !IsPromptScene(hint) {
			continue
		}
		if promptWeakSceneHints[hint] {
			weak = append(weak, hint)
		} else {
			strong = append(strong, hint)
		}
	}
	return uniquePromptStrings(append(append(strong, matched...), weak...), 4)
}

func uniquePromptStrings(values []string, max int) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

// PromptDisplayTitle replaces titles that say nothing on a card (a category name used as the title)
// with one derived from the prompt: a quoted name first, else the first clause.
func PromptDisplayTitle(title, prompt string, tags []string) string {
	title = strings.TrimSpace(title)
	generic := title == "" || promptGenericTitle.MatchString(title)
	for _, tag := range tags {
		if tag == title {
			generic = true
		}
	}
	if !generic {
		return truncateRunes(title, 80)
	}
	text := promptLangMarker.ReplaceAllString(prompt, "")
	text = strings.TrimSpace(promptSpaces.ReplaceAllString(text, " "))
	if m := promptQuoted.FindStringSubmatch(text); m != nil {
		quoted := strings.TrimSpace(strings.Split(m[1], "/")[0])
		if quoted != "" && utf8.RuneCountInString(quoted) <= 24 && !promptPlaceholder.MatchString(quoted) {
			return quoted
		}
	}
	rest := promptLeadingVerb.ReplaceAllString(text, "")
	for _, part := range promptClauseSplit.Split(rest, -1) {
		part = strings.TrimSpace(part)
		if utf8.RuneCountInString(part) >= 2 {
			if utf8.RuneCountInString(part) > 20 {
				return string([]rune(part)[:20]) + "…"
			}
			return part
		}
	}
	if title != "" {
		return title
	}
	return truncateRunes(text, 20)
}

// PromptDedupeKey identifies the same prompt published by several sources.
func PromptDedupeKey(prompt string) string {
	key := strings.ToLower(promptSpaces.ReplaceAllString(prompt, ""))
	return truncateRunes(key, 300)
}

// PromptQualityScore ranks entries when usage does not decide: usable here, has a picture, Chinese first.
func PromptQualityScore(coverURL, title string, traits PromptTraits) int {
	score := 0
	if coverURL != "" {
		score += 4
	}
	switch traits.Model {
	case "gpt-image-2":
		score += 3
	case "unknown":
		score++
	}
	if traits.Lang == "zh" {
		score += 2
	}
	if promptCJK.MatchString(title) {
		score++
	}
	if len(traits.Scenes) > 0 {
		switch traits.Scenes[0] {
		case "other":
			score--
		case "video":
			score -= 2
		}
	}
	return score
}
