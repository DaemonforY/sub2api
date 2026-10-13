package service

import (
	"fmt"
	"strings"
)

// Prompts of the 内容引擎 pipeline. They share one stance: write like an experienced practitioner,
// never invent facts, numbers or personal experiences — where first-hand material is needed and not
// given, leave a 【素材卡：…】 placeholder for the owner — and declare AI content honestly (the
// version tells the owner which box to tick; removing the AI flavour is about quality, not about
// hiding).

const contentWriterStance = `你是一位写了多年中文技术自媒体的作者兼编辑，读者是想把 AI 用起来的程序员、学生和职场人。
写作原则：
1. 先给结论和能直接照做的步骤，再解释原因；少讲概念，多给具体的命令、配置、截图位置、数字。
2. 只写有依据的内容：资料、搜索结果和素材里有的才能写成事实。不要编造数据、案例、用户评价和亲身经历。
3. 需要一手素材（实测数据、踩坑经历、截图、具体配置）而资料里没有时，在那个位置写「【素材卡：需要补充什么】」，由作者补充，不要自己编。
4. 像真人说话：句子长短交错，用具体的词代替空泛的形容词，可以有态度，但不要煽情、不要夸张。
5. 不要用这些 AI 腔套话：值得注意的是、总而言之、综上所述、总的来说、在当今…时代、随着…的发展、赋能、助力、不可或缺、至关重要、深入探讨、让我们一起、一站式、全方位、无缝、扮演着…角色、日新月异、毋庸置疑、众所周知、颠覆性、革命性、不仅…更是。
6. 不写广告法极限词（最便宜、最稳定、全网第一、顶级、百分百…），不贬低其他产品，不出现翻墙、科学上网、梯子、VPN 这类词。`

func contentProductBlock(product string) string {
	return "\n\n文章最终要把读者自然地引到这个产品（只在合适的地方提一次，不要硬广）：\n" + product
}

func contentRulesBlock(style string, rules []string) string {
	var b strings.Builder
	if strings.TrimSpace(style) != "" {
		_, _ = b.WriteString("\n\n作者的写作风格（照这个风格写）：\n" + style)
	}
	if len(rules) > 0 {
		_, _ = b.WriteString("\n\n作者定的写作规则（必须遵守）：")
		for _, r := range rules {
			_, _ = b.WriteString("\n- " + r)
		}
	}
	return b.String()
}

func contentMaterialsBlock(it *ContentItem, passages []assistantHit) string {
	var b strings.Builder
	_, _ = b.WriteString("选题：" + it.Brief.Topic)
	if it.Brief.Angle != "" {
		_, _ = b.WriteString("\n角度：" + it.Brief.Angle)
	}
	if it.Brief.Notes != "" {
		_, _ = b.WriteString("\n\n作者给的素材（一手资料，优先使用）：\n" + it.Brief.Notes)
	}
	if len(passages) > 0 {
		_, _ = b.WriteString("\n\n本站学习站里相关的内容（可以引用，也说明了本站有哪些教程）：")
		for _, p := range passages {
			fmt.Fprintf(&b, "\n\n### %s（%s）\n%s", p.Title, p.URL, truncateRunes(p.Text, 900))
		}
	}
	if len(it.Sources) > 0 {
		_, _ = b.WriteString("\n\n联网搜到的资料：")
		for _, s := range it.Sources {
			fmt.Fprintf(&b, "\n- %s（%s）：%s", s.Title, s.URL, s.Snippet)
		}
	}
	return b.String()
}

// --- outline -----------------------------------------------------------------------------------

func contentOutlinePrompt(product string, search bool) string {
	p := contentWriterStance + contentProductBlock(product) + `

任务：为下面的选题定角度、写大纲。`
	if search {
		p += "先用 web_search 搜 1–3 次，找最新、具体的资料（版本号、配置方法、官方说明、常见报错），再出大纲。"
	}
	return p + `
只输出一个 JSON 对象，不要任何其他文字：
{
  "title": "主标题，像读者会搜的那句话，不超过 30 字",
  "angle": "这篇文章的切入角度，一句话",
  "reader": "写给谁看，一句话",
  "hook": "开头第一段要抛出的结论或问题，一句话",
  "sections": [{"heading": "小标题", "points": ["这一节要讲的要点", "..."]}],
  "materials_needed": ["需要作者补充的一手素材，没有就留空数组"],
  "landing_path": "从候选落地页里选一个最相关的路径；都不相关就填 /"
}
小节 3–6 个，每节 2–4 个要点。`
}

func contentOutlineRequest(it *ContentItem, passages []assistantHit, landing []ContentLanding) string {
	var b strings.Builder
	_, _ = b.WriteString(contentMaterialsBlock(it, passages))
	_, _ = b.WriteString("\n\n候选落地页（读者点链接后到达的本站页面）：\n- / 首页")
	for _, l := range landing {
		fmt.Fprintf(&b, "\n- %s %s", l.Path, l.Title)
	}
	return b.String()
}

// --- draft -------------------------------------------------------------------------------------

func contentDraftPrompt(product, style string, rules []string) string {
	return contentWriterStance + contentProductBlock(product) + contentRulesBlock(style, rules) + `

任务：按大纲写一篇完整的母稿（后面会再改写成各平台版本）。
要求：
- Markdown 格式，一级标题是文章标题，正文用二级、三级标题分节；代码、命令、配置放进代码块。
- 1800–3000 字。开头一段直接给结论或抛出问题，不要铺垫背景。
- 文中不要放任何链接，也不要写「点击链接」之类的话（各平台版本会自己加）。
- 资料里没有的一手信息，用「【素材卡：…】」标出来。
只输出 Markdown 正文。`
}

func contentDraftRequest(it *ContentItem, passages []assistantHit) string {
	o := it.Outline
	var b strings.Builder
	fmt.Fprintf(&b, "标题：%s\n角度：%s\n读者：%s\n开头：%s\n\n大纲：", o.Title, o.Angle, o.Reader, o.Hook)
	for _, s := range o.Sections {
		fmt.Fprintf(&b, "\n## %s", s.Heading)
		for _, p := range s.Points {
			_, _ = b.WriteString("\n- " + p)
		}
	}
	if len(o.MaterialsNeeded) > 0 {
		_, _ = b.WriteString("\n\n需要作者补充的素材：" + strings.Join(o.MaterialsNeeded, "；"))
	}
	_, _ = b.WriteString("\n\n---\n\n" + contentMaterialsBlock(it, passages))
	return b.String()
}

// --- review ------------------------------------------------------------------------------------

const contentReviewPrompt = `你是一位严格的技术编辑，负责审校下面这篇稿子。逐条找出问题：
1. 事实：和资料不符、资料里找不到依据的说法、过时的信息、可能写错的命令和配置。
2. 读者：哪里跳步了、哪里看不懂、缺了哪一步操作。
3. AI 腔：空话、套话、排比堆砌、每段都用「首先 / 其次 / 最后」、结尾强行升华。
4. 风险：广告法极限词、夸大承诺、贬低别人、敏感词。
只输出一个 JSON 对象，不要任何其他文字：
{"notes": ["具体问题和改法，每条一句，最多 12 条"], "ai_score": 0到100的整数，越像 AI 写的越高}`

func contentReviewRequest(it *ContentItem, passages []assistantHit) string {
	return "稿子：\n\n" + it.Draft + "\n\n---\n\n参考资料：\n" + contentMaterialsBlock(it, passages)
}

// --- humanize ----------------------------------------------------------------------------------

func contentHumanizePrompt(product, style string, rules []string) string {
	return contentWriterStance + contentProductBlock(product) + contentRulesBlock(style, rules) + `

任务：根据编辑的审校意见修改稿子，并去掉 AI 味：
- 逐条落实审校意见；有依据问题的说法删掉或改成有依据的。
- 去掉套话和空洞的总结，把抽象的说法换成具体的例子、数字、操作。
- 调整节奏：长短句交错，段落不要一样长，不要每段都是「观点 + 三点展开」。
- 保留结构、代码块和「【素材卡：…】」占位，不要新增链接。
只输出修改后的完整 Markdown。`
}

func contentHumanizeRequest(text string, notes []string) string {
	var b strings.Builder
	_, _ = b.WriteString("审校意见：")
	if len(notes) == 0 {
		_, _ = b.WriteString("\n- （没有具体意见，重点去 AI 味）")
	}
	for _, n := range notes {
		_, _ = b.WriteString("\n- " + n)
	}
	_, _ = b.WriteString("\n\n稿子：\n\n" + text)
	return b.String()
}

// --- platform versions -------------------------------------------------------------------------

// contentPlatformGuide is what each platform's readers and rules want.
var contentPlatformGuide = map[string]string{
	ContentPlatformWechat: `平台：微信公众号（作者已有约 1000 粉丝，是私域主阵地）。
- 长文，1500–3000 字，保留小标题和代码块；语气可以更像和老读者聊天，有作者自己的观点。
- 标题不超过 30 字，可以有一点悬念，但不要标题党。digest 写 120 字以内的摘要。
- 正文里不能放外链。结尾一句话引导读者点「阅读原文」看完整教程或去注册体验，不要写网址。
- kind 填 "article"。`,
	ContentPlatformZhihu: `平台：知乎（新号，没有粉丝，靠搜索和问题流量）。
- 优先写成「回答」：kind 填 "answer"，question 写一个知乎上真实可能存在、搜索量大的问题（作者会去搜这个问题再回答）；如果选题更适合长文，kind 填 "article"，question 留空。
- 第一段直接回答问题、给结论，知乎和 AI 搜索最常截取第一段。语气专业、克制，像有经验的人在分享，不要营销腔。
- 600–2500 字，可以用小标题、列表和代码块。
- 文末只放一次引流：一句话说明完整教程或工具在哪里，后面接 {{link}}（这是链接占位符，原样写出）。
- tags 写 3–5 个知乎话题。title 不超过 50 字（回答也写一个标题，作者备用）。`,
	ContentPlatformCSDN: `平台：CSDN（新号，靠百度和站内搜索流量）。
- 技术教程风格：标题就是搜索词，包含技术名词和版本号，不超过 60 字。
- 结构清楚：问题/背景 → 环境与准备 → 步骤（代码块要完整可运行）→ 常见报错和解决 → 小结。1500–5000 字。
- 文末一句话引导读者去看完整教程或在线体验，后面接 {{link}}（链接占位符，原样写出），再另起一行写「本文由 AI 辅助创作，内容经作者审核」。
- tags 写 3–5 个技术标签，category 写一个分类（如「人工智能」）。kind 填 "article"。`,
	ContentPlatformToutiao: `平台：今日头条（新号，推荐算法不看粉丝数，读者是泛科技兴趣人群）。
- 标题 10–30 字，口语化、有信息量，讲清楚「对读者有什么用」，不要震惊体。
- 正文 800–2000 字，短段落（每段 1–3 句），少用代码，多用通俗的比喻和场景；可以用小标题。
- 正文绝对不能放链接或网址。结尾一句话引导读者搜索产品名注册，并提到注册时填优惠码 {{promo}} 多送体验额度（优惠码占位符，原样写出）。
- micro 写一条 300 字以内的微头条，单独发：一个有用的观点或技巧 + 一句互动提问，同样不能有链接。
- kind 填 "article"。`,
	ContentPlatformXHS: `平台：小红书（新号，读者是年轻人和学生，靠搜索和推荐）。
- 图文笔记：内容主要放在图卡上。cards 写 4–7 张：第 1 张是封面（title 是大字标题，points 写 1–2 句副标题），中间每张一个要点（title 不超过 12 字，points 2–4 条，每条不超过 25 字），最后一张是总结或行动建议。
- title 不超过 20 字，口语化，可以带一个 emoji。
- body 是笔记正文，300–800 字，分短段，可以适量用 emoji，结尾提一个问题引导评论。
- 正文和图卡都绝对不能放链接、网址、二维码或「私信我」。正文结尾写：搜「产品名」注册，填优惠码 {{promo}} 多送体验额度（优惠码占位符，原样写出）。
- tags 写 5–8 个话题（不带 #）。kind 填 "note"。`,
}

func contentAdaptPrompt(platform, product, style, brand string, rules []string) string {
	return contentWriterStance + contentProductBlock(product) + contentRulesBlock(style, rules) + `

任务：把母稿改写成下面这个平台的版本。必须真正改写（换标题、开头、结构、长度和语气），不能只是删减或换格式——多个平台同时发同一篇会被判搬运。
产品名写作「` + brand + `」。

` + contentPlatformGuide[platform] + `

只输出一个 JSON 对象，不要任何其他文字（不需要的字段填空字符串或空数组）：
{"kind": "", "title": "", "question": "", "digest": "", "body": "正文 Markdown", "micro": "", "tags": [], "category": "", "cards": [{"title": "", "points": []}]}`
}

func contentAdaptRequest(it *ContentItem, feedback string) string {
	req := "母稿：\n\n" + it.Markdown
	if feedback != "" {
		req += "\n\n---\n\n作者对上一版的意见（这次必须改好）：" + feedback
	}
	return req
}

// --- topic suggestions -------------------------------------------------------------------------

func contentTopicsPrompt(product string, platforms []string, search bool, n int) string {
	names := make([]string, 0, len(platforms))
	for _, p := range platforms {
		names = append(names, contentPlatformSpecs[p].Name)
	}
	p := contentWriterStance + contentProductBlock(product) + fmt.Sprintf(`

任务：为这些平台的账号策划 %d 个选题：%s。账号除公众号外都是新号，选题要能靠搜索和推荐拿到流量，并且能自然地引到上面的产品。
选题来源（按优先级）：
1. 用户真实问过的问题（support）：最能解决痛点，优先。
2. AI 搜索会被问到的问题（geo）：适合知乎回答和 CSDN 教程。
3. 学习站已有的教程（learn）：改写成适合平台的版本，省力且有一手内容。
4. AI 圈的新热点（hot）`, n, strings.Join(names, "、"))
	if search {
		p += "：可以用 web_search 搜 1–2 次最近的热点。"
	} else {
		p += "。"
	}
	return p + `
不要和「已经写过的选题」重复或太像。每个选题给出明确的角度，不要只写一个宽泛的主题。
只输出一个 JSON 对象，不要任何其他文字：
{"topics": [{"title": "选题（像读者会搜的一句话）", "angle": "切入角度和要解决的问题", "source": "support|geo|learn|hot|idea", "reason": "为什么值得写，一句话", "score": 0到100的整数}]}`
}

func contentTopicsRequest(questions, geo, pages, done []string) string {
	var b strings.Builder
	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		_, _ = b.WriteString("\n\n" + title + "：")
		for _, x := range items {
			_, _ = b.WriteString("\n- " + x)
		}
	}
	_, _ = b.WriteString("请策划选题。")
	section("用户最近问智能客服的问题", questions)
	section("GEO 监测里的问题", geo)
	section("学习站已有的教程", pages)
	section("已经写过或已在选题池里的选题（不要重复）", done)
	return b.String()
}
