package web

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	htmlpkg "html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Search engines and AI crawlers mostly read the HTML as served, without running the SPA. So every
// public page gets its own <title>, description, canonical, Open Graph tags and JSON-LD, plus a short
// text version of the page (heading, summary, links, FAQ) inside #app, which Vue replaces on mount.
// Pages behind sign-in are noindex. /robots.txt, /sitemap.xml, /llms.txt and /llms-full.txt are
// generated per host (main site and video.<domain>).

// SEOPage is a public page: a sitemap entry and the facts its <head> describes.
type SEOPage struct {
	Path        string
	Title       string
	Description string
	Image       string
	Price       float64 // courses: CNY, 0 = free or unknown
	Updated     time.Time
}

// SEOFAQ is an entry of the site FAQ (the support assistant's knowledge base).
type SEOFAQ struct {
	Question string
	Answer   string
	Link     string
}

// SEOLearnPage is a page of the /learn site, for llms.txt.
type SEOLearnPage struct {
	URL   string
	Title string
	Text  string
}

// SEOModel is a model on the public model plaza, with its display price per 1M tokens (USD).
type SEOModel struct {
	Name      string
	Group     string
	Input     *float64
	Output    *float64
	CacheRead *float64
}

// SEOContent supplies the dynamic public pages. Implementations should cache: crawlers call it often.
type SEOContent interface {
	// Models is the public model plaza; ok is false while the plaza is off or needs sign-in.
	Models(ctx context.Context) (models []SEOModel, ok bool)
	Courses(ctx context.Context) ([]SEOPage, error)
	VideoWorks(ctx context.Context) ([]SEOPage, error)
	VideoWork(ctx context.Context, id string) (*SEOPage, error)
	FAQ() []SEOFAQ
}

// VideoPlayerSource returns the player data (JSON) of a public work for /play/<id>, or ok=false.
type VideoPlayerSource interface {
	PlayerData(ctx context.Context, id string) (data []byte, ok bool)
}

// SEOTool is a video tool landing page (video/src/lib/tools.json, built to dist/video/seo-tools.json).
type SEOTool struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	Headline string   `json:"headline"`
	Summary  string   `json:"summary"`
	Points   []string `json:"points"`
	Prompts  []string `json:"prompts"`
}

type seoLink struct {
	Path  string
	Label string
}

// pageMeta is what a page's HTML says about it before the SPA runs.
type pageMeta struct {
	Title       string
	Description string
	Path        string // canonical path
	Image       string
	Index       bool
	OGType      string
	H1          string
	Paragraphs  []string
	List        []string
	Links       []seoLink
	FAQ         []SEOFAQ
	JSONLD      []any
}

// seoSite is the host-level context of a page.
type seoSite struct {
	Name   string // site name (main) or "<name> 视频" (video)
	Origin string // https://host
	Logo   string // absolute URL, may be empty
}

func (s seoSite) abs(p string) string {
	if p == "" || strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return s.Origin + p
}

// normalizeSEOPath drops a trailing slash (except the root) and lower-cases nothing: paths are case-sensitive.
func normalizeSEOPath(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
		if p == "" {
			p = "/"
		}
	}
	return p
}

func homeDescription(name string) string {
	return name + " 是一站式 AI 服务平台：一个账号、一个 API Key 即可调用 GPT、Codex 等大模型，兼容 OpenAI 接口，按实际 token 用量计费；" +
		"可接入 Codex、OpenCode、Cherry Studio 等 AI 工具，还有无限画布 AI 生图、AI 学习站、公众号排版和付费课程。"
}

func plazaPublic(ctx context.Context, content SEOContent) bool {
	if content == nil {
		return false
	}
	_, ok := content.Models(ctx)
	return ok
}

func mainNavLinks(ctx context.Context, content SEOContent) []seoLink {
	links := []seoLink{
		{"/", "首页"},
		{"/pricing", "价格与套餐"},
	}
	if plazaPublic(ctx, content) {
		links = append(links, seoLink{"/model-plaza", "模型广场"})
	}
	return append(links, []seoLink{
		{"/learn/", "AI 学习站（教程）"},
		{"/learn/connect/", "接入教程：Codex、OpenCode、Cherry Studio、SDK"},
		{"/courses", "AI 课程"},
		{"/contests", "创作比赛"},
		{"/editor/", "公众号排版"},
		{"/register", "注册"},
		{"/login", "登录"},
	}...)
}

// mainStaticPages are the main site's fixed public pages, in sitemap order.
var mainStaticPages = []struct {
	Path, Title, Description string
}{
	{"/", "", ""}, // filled from the site name
	{"/pricing", "价格与套餐：按量计费与订阅对比", "看清 GPT、Codex、画图等模型怎么计费：按实际 token 用量扣费或购买订阅套餐，在售套餐的额度对比和按用量估算，充值与支付方式说明。"},
	{"/model-plaza", "模型广场：可用 AI 模型与价格", "平台可调用的 AI 模型一览：模型能力、所属分组、输入输出 token 单价，以及如何用 API Key 调用。"},
	{"/courses", "AI 课程：编程、画图与工作流实战", "精选 AI 实战课程：AI 编程工具、AI 画图、自动化工作流等，试看后购买，讲师答疑。"},
	{"/contests", "AI 创作比赛", "参加 AI 画图、视频与编程创作比赛，投稿作品、查看榜单和获奖结果。"},
	{"/register", "注册账号", "注册账号，创建 API Key，即可调用 GPT、Codex 等 AI 模型并接入编程工具。"},
	{"/login", "登录", "登录账号，管理 API Key、查看用量记录、充值和订阅。"},
}

// mainPageMeta resolves a main-site path. Unknown paths and pages behind sign-in are noindex.
func mainPageMeta(ctx context.Context, site seoSite, content SEOContent, path string) pageMeta {
	p := normalizeSEOPath(path)
	if p == "/home" {
		p = "/"
	}
	m := pageMeta{Path: p, Image: site.Logo, OGType: "website", Links: mainNavLinks(ctx, content)}
	for _, sp := range mainStaticPages {
		if sp.Path != p {
			continue
		}
		if p == "/model-plaza" {
			if content == nil {
				break
			}
			models, ok := content.Models(ctx)
			if !ok {
				break // plaza off or behind sign-in: falls through to the noindex default below
			}
			m.Index = true
			m.Title = sp.Title + " | " + site.Name
			m.Description = sp.Description
			m.H1 = sp.Title
			m.Paragraphs = []string{sp.Description, "价格为每 100 万 token 的美元价，充值 ¥1 = $1 额度。"}
			for _, md := range models {
				m.List = append(m.List, modelPriceLine(md))
			}
			return m
		}
		m.Index = true
		if p == "/" {
			m.Title = site.Name + " - AI 模型 API 平台 · 兼容 OpenAI 接口 · 接入 Codex / OpenCode"
			m.Description = homeDescription(site.Name)
			m.H1 = site.Name + "：AI 模型 API 平台"
			m.Paragraphs = []string{m.Description}
			org := map[string]any{"@context": "https://schema.org", "@type": "Organization", "name": site.Name, "url": site.abs("/")}
			if site.Logo != "" {
				org["logo"] = site.Logo
			}
			m.JSONLD = append(m.JSONLD, org,
				map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": site.Name, "url": site.abs("/"), "inLanguage": "zh-CN"},
			)
			if content != nil {
				m.FAQ = content.FAQ()
			}
		} else {
			m.Title = sp.Title + " | " + site.Name
			m.Description = sp.Description
			m.H1 = sp.Title
			m.Paragraphs = []string{sp.Description}
		}
		if p == "/courses" && content != nil {
			if courses, err := content.Courses(ctx); err == nil {
				for _, c := range courses {
					m.Links = append(m.Links, seoLink{c.Path, c.Title})
				}
			}
		}
		return m
	}

	switch {
	case strings.HasPrefix(p, "/courses/") && content != nil:
		if courses, err := content.Courses(ctx); err == nil {
			for _, c := range courses {
				if c.Path != p {
					continue
				}
				m.Index = true
				m.OGType = "article"
				m.Title = c.Title + " | " + site.Name + " 课程"
				m.Description = clipText(firstNonEmpty(c.Description, c.Title+"："+site.Name+" AI 实战课程。"), 160)
				m.H1 = c.Title
				m.Paragraphs = []string{m.Description}
				if c.Image != "" {
					m.Image = site.abs(c.Image)
				}
				course := map[string]any{
					"@context": "https://schema.org", "@type": "Course", "name": c.Title, "description": m.Description,
					"url": site.abs(c.Path), "inLanguage": "zh-CN",
					"provider": map[string]any{"@type": "Organization", "name": site.Name, "url": site.abs("/")},
				}
				if c.Price > 0 {
					course["offers"] = map[string]any{"@type": "Offer", "price": c.Price, "priceCurrency": "CNY", "category": "Paid"}
				}
				m.JSONLD = append(m.JSONLD, course)
				return m
			}
		}
	case strings.HasPrefix(p, "/contests/"):
		m.Index = true
		m.Title = "AI 创作比赛 | " + site.Name
		m.Description = "AI 创作比赛详情：赛题、规则、投稿作品与榜单。"
		m.H1 = "AI 创作比赛"
		return m
	case strings.HasPrefix(p, "/legal/"):
		m.Index = true
		m.Title = "用户协议与隐私政策 | " + site.Name
		m.Description = site.Name + " 的服务条款、隐私政策等法律文件。"
		m.H1 = "法律文件"
		return m
	}

	m.Title = site.Name
	m.Description = homeDescription(site.Name)
	return m
}

// mainRobotsDisallow: pages behind sign-in, callbacks and the admin. Everything else may be crawled.
var mainRobotsDisallow = []string{
	"/api/", "/admin", "/auth/", "/setup", "/dashboard", "/keys", "/usage", "/profile", "/redeem",
	"/subscriptions", "/purchase", "/orders", "/payment/", "/affiliate", "/sites", "/image-tool-uses",
	"/available-channels", "/my-learning", "/my-courses", "/creator", "/video-connect", "/canvas-connect",
	"/key-usage", "/email-verify", "/reset-password", "/forgot-password", "/custom/", "/monitor", "/batch-image",
}

var videoRobotsDisallow = []string{"/api/", "/p/", "/review", "/upload"}

// seoCrawlers are named in robots.txt so it is explicit that search and AI crawlers are welcome.
var seoCrawlers = []string{
	"*", "Baiduspider", "Bingbot", "Googlebot", "Sogou web spider", "360Spider", "YisouSpider", "Bytespider",
	"GPTBot", "OAI-SearchBot", "ChatGPT-User", "ClaudeBot", "Claude-SearchBot", "Claude-User", "PerplexityBot",
	"Google-Extended", "Applebot", "Applebot-Extended", "DeepSeekBot", "Amazonbot", "meta-externalagent", "CCBot",
}

func robotsTxt(site seoSite, disallow []string, sitemaps []string) string {
	var b strings.Builder
	_, _ = b.WriteString("# " + site.Name + "\n# Search engines and AI crawlers are welcome. Summary for language models: " + site.abs("/llms.txt") + "\n\n")
	for _, ua := range seoCrawlers {
		_, _ = b.WriteString("User-agent: " + ua + "\n")
	}
	_, _ = b.WriteString("Allow: /\n")
	for _, d := range disallow {
		_, _ = b.WriteString("Disallow: " + d + "\n")
	}
	_, _ = b.WriteString("\n")
	for _, s := range sitemaps {
		_, _ = b.WriteString("Sitemap: " + site.abs(s) + "\n")
	}
	return b.String()
}

type sitemapURL struct {
	Loc      string `xml:"loc"`
	LastMod  string `xml:"lastmod,omitempty"`
	Priority string `xml:"priority,omitempty"`
}

func sitemapXML(site seoSite, pages []SEOPage) []byte {
	type urlset struct {
		XMLName xml.Name     `xml:"urlset"`
		XMLNS   string       `xml:"xmlns,attr"`
		URLs    []sitemapURL `xml:"url"`
	}
	set := urlset{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	seen := map[string]bool{}
	for _, p := range pages {
		loc := site.abs(p.Path)
		if seen[loc] {
			continue
		}
		seen[loc] = true
		u := sitemapURL{Loc: loc}
		if !p.Updated.IsZero() {
			u.LastMod = p.Updated.UTC().Format("2006-01-02")
		}
		if p.Path == "/" {
			u.Priority = "1.0"
		}
		set.URLs = append(set.URLs, u)
	}
	out, _ := xml.MarshalIndent(set, "", "  ")
	return append([]byte(xml.Header), append(out, '\n')...)
}

// mainSitemapPages: fixed public pages and the courses on sale (/learn has its own sitemap).
func mainSitemapPages(ctx context.Context, content SEOContent) []SEOPage {
	var out []SEOPage
	for _, sp := range mainStaticPages {
		if sp.Path == "/login" || sp.Path == "/register" || (sp.Path == "/model-plaza" && !plazaPublic(ctx, content)) {
			continue
		}
		out = append(out, SEOPage{Path: sp.Path})
	}
	if content != nil {
		if courses, err := content.Courses(ctx); err == nil {
			out = append(out, courses...)
		}
	}
	return out
}

// mainLLMsTxt is /llms.txt (https://llmstxt.org): what the site is and where its content is. With full
// set, the FAQ answers and the learning site's text follow (/llms-full.txt).
func mainLLMsTxt(ctx context.Context, site seoSite, content SEOContent, learn []SEOLearnPage, full bool) string {
	var b strings.Builder
	_, _ = b.WriteString("# " + site.Name + "\n\n> " + homeDescription(site.Name) + "\n\n")
	_, _ = b.WriteString("- 网站：" + site.abs("/") + "\n")
	_, _ = b.WriteString("- OpenAI 兼容接口（Base URL）：" + site.abs("/v1") + "（如 /v1/chat/completions、/v1/responses、/v1/images/generations）\n")
	_, _ = b.WriteString("- 可用模型：用自己的 API Key 请求 " + site.abs("/v1/models") + "\n")
	_, _ = b.WriteString("- 计费：按实际 token 用量扣费，或购买订阅套餐；具体价格以价格页为准\n\n")

	_, _ = b.WriteString("## 主要页面\n\n")
	for _, sp := range mainStaticPages {
		if sp.Path == "/" || (sp.Path == "/model-plaza" && !plazaPublic(ctx, content)) {
			continue
		}
		_, _ = b.WriteString("- [" + sp.Title + "](" + site.abs(sp.Path) + "): " + sp.Description + "\n")
	}
	_, _ = b.WriteString("- [公众号排版](" + site.abs("/editor/") + "): 在线排版微信公众号文章，一键复制到公众号后台。\n\n")

	if len(learn) > 0 {
		_, _ = b.WriteString("## AI 学习站（教程）\n\n")
		for _, p := range learn {
			_, _ = b.WriteString("- [" + p.Title + "](" + site.abs(p.URL) + ")\n")
		}
		_, _ = b.WriteString("\n")
	}

	if content != nil {
		if models, ok := content.Models(ctx); ok && len(models) > 0 {
			_, _ = b.WriteString("## 可用模型与价格（每 100 万 token，美元）\n\n")
			for _, md := range models {
				_, _ = b.WriteString("- " + modelPriceLine(md) + "\n")
			}
			_, _ = b.WriteString("\n完整列表：" + site.abs("/model-plaza") + "\n\n")
		}
		if courses, err := content.Courses(ctx); err == nil && len(courses) > 0 {
			_, _ = b.WriteString("## 课程\n\n")
			for _, c := range courses {
				line := "- [" + c.Title + "](" + site.abs(c.Path) + ")"
				if c.Description != "" {
					line += ": " + clipText(c.Description, 100)
				}
				_, _ = b.WriteString(line + "\n")
			}
			_, _ = b.WriteString("\n")
		}
	}

	if !full {
		_, _ = b.WriteString("## Optional\n\n- [完整版：常见问题与全部教程正文](" + site.abs("/llms-full.txt") + ")\n")
		return b.String()
	}

	if content != nil {
		if faq := content.FAQ(); len(faq) > 0 {
			_, _ = b.WriteString("## 常见问题\n\n")
			for _, f := range faq {
				_, _ = b.WriteString("### " + f.Question + "\n\n")
				if f.Link != "" {
					_, _ = b.WriteString("链接：" + site.abs(f.Link) + "\n\n")
				}
				_, _ = b.WriteString(f.Answer + "\n\n")
			}
		}
	}
	if len(learn) > 0 {
		_, _ = b.WriteString("## 教程正文\n\n")
		for _, p := range learn {
			_, _ = b.WriteString("### " + p.Title + "\n\n来源：" + site.abs(p.URL) + "\n\n" + strings.TrimSpace(p.Text) + "\n\n")
		}
	}
	return b.String()
}

func videoDescription() string {
	return "输入一个主题，AI 写脚本、配音、做动画，几分钟得到带字幕的 HTML 讲解视频；也能一句话生成 Logo、文字、图表、流程图等网页动画。"
}

func videoNavLinks(tools []SEOTool) []seoLink {
	links := []seoLink{{"/", "开始创作"}, {"/gallery", "作品广场"}, {"/tools", "全部工具"}, {"/pricing", "价格"}}
	for _, t := range tools {
		links = append(links, seoLink{"/tools/" + t.Slug, t.Name})
	}
	return links
}

// videoPageMeta resolves a path of the video site.
func videoPageMeta(ctx context.Context, site seoSite, content SEOContent, tools []SEOTool, path string) pageMeta {
	p := normalizeSEOPath(path)
	m := pageMeta{Path: p, Image: site.Logo, OGType: "website", Index: true, Links: videoNavLinks(tools)}
	switch {
	case p == "/":
		m.Title = site.Name + " · 一句话生成讲解视频与动画"
		m.Description = videoDescription()
		m.H1 = site.Name + "：AI 讲解视频与网页动画生成"
		m.JSONLD = append(m.JSONLD, map[string]any{
			"@context": "https://schema.org", "@type": "WebApplication", "name": site.Name, "url": site.abs("/"),
			"applicationCategory": "MultimediaApplication", "operatingSystem": "Web", "inLanguage": "zh-CN", "description": m.Description,
		})
	case p == "/gallery" || strings.HasPrefix(p, "/gallery/"):
		m.Title = "作品广场：AI 生成的讲解视频与动画 | " + site.Name
		m.Description = "浏览用户用 AI 生成的讲解视频和网页动画：科普、数据可视化、Logo 动画、流程图等，一键制作同款。"
		m.H1 = "作品广场"
		if content != nil {
			if works, err := content.VideoWorks(ctx); err == nil {
				for i, w := range works {
					if i >= 60 {
						break
					}
					m.Links = append(m.Links, seoLink{w.Path, w.Title})
				}
			}
		}
	case p == "/tools":
		m.Title = "AI 视频与动画工具大全 | " + site.Name
		m.Description = "HTML 讲解视频、理工科普动画、数据可视化、Logo 动画等 AI 工具，一句话生成，可继续对话修改。"
		m.H1 = "全部工具"
	case strings.HasPrefix(p, "/tools/"):
		slug := strings.TrimPrefix(p, "/tools/")
		for _, t := range tools {
			if t.Slug != slug {
				continue
			}
			m.Title = t.Name + "：" + t.Headline + " | " + site.Name
			m.Description = clipText(t.Summary, 160)
			m.H1 = t.Name
			m.Paragraphs = []string{t.Headline}
			m.List = append(append([]string{}, t.Points...), t.Prompts...)
			break
		}
		if m.H1 == "" {
			m.Index = false
			m.Title = site.Name
			m.Description = videoDescription()
		}
	case p == "/pricing":
		m.Title = "价格 | " + site.Name
		m.Description = "按 token 计费，通过账号余额或订阅扣费；配音、导出和发布免费。"
		m.H1 = "价格"
	case strings.HasPrefix(p, "/w/") && content != nil:
		w, err := content.VideoWork(ctx, strings.TrimPrefix(p, "/w/"))
		if err == nil && w != nil {
			m.OGType = "video.other"
			m.Title = w.Title + " | " + site.Name + " 作品"
			m.Description = clipText(firstNonEmpty(w.Description, w.Title), 160)
			m.H1 = w.Title
			m.JSONLD = append(m.JSONLD, map[string]any{
				"@context": "https://schema.org", "@type": "CreativeWork", "name": w.Title, "description": m.Description,
				"url": site.abs(w.Path), "inLanguage": "zh-CN", "dateModified": w.Updated.UTC().Format(time.RFC3339),
			})
			break
		}
		m.Index = false
		m.Title = site.Name
		m.Description = videoDescription()
	default:
		m.Index = false
		m.Title = site.Name
		m.Description = videoDescription()
	}
	if len(m.Paragraphs) == 0 && m.Description != "" {
		m.Paragraphs = []string{m.Description}
	}
	return m
}

func videoSitemapPages(ctx context.Context, content SEOContent, tools []SEOTool) []SEOPage {
	out := []SEOPage{{Path: "/"}, {Path: "/gallery"}, {Path: "/tools"}, {Path: "/pricing"}}
	for _, t := range tools {
		out = append(out, SEOPage{Path: "/tools/" + t.Slug})
	}
	if content != nil {
		if works, err := content.VideoWorks(ctx); err == nil {
			out = append(out, works...)
		}
	}
	return out
}

func videoLLMsTxt(ctx context.Context, site seoSite, content SEOContent, tools []SEOTool) string {
	var b strings.Builder
	_, _ = b.WriteString("# " + site.Name + "\n\n> " + videoDescription() + "按所用 API Key 的分组计费（余额或订阅），配音、导出和发布免费。\n\n")
	_, _ = b.WriteString("## 页面\n\n")
	_, _ = b.WriteString("- [开始创作](" + site.abs("/") + "): 输入主题，选择横屏 / 竖屏 / 方形、音色、时长和视觉风格\n")
	_, _ = b.WriteString("- [作品广场](" + site.abs("/gallery") + "): 用户公开的作品，可制作同款\n")
	_, _ = b.WriteString("- [价格](" + site.abs("/pricing") + ")\n\n")
	if len(tools) > 0 {
		_, _ = b.WriteString("## 工具\n\n")
		for _, t := range tools {
			_, _ = b.WriteString("- [" + t.Name + "](" + site.abs("/tools/"+t.Slug) + "): " + t.Headline + "。" + t.Summary + "\n")
		}
		_, _ = b.WriteString("\n")
	}
	if content != nil {
		if works, err := content.VideoWorks(ctx); err == nil && len(works) > 0 {
			_, _ = b.WriteString("## 精选作品\n\n")
			for i, w := range works {
				if i >= 50 {
					break
				}
				_, _ = b.WriteString("- [" + w.Title + "](" + site.abs(w.Path) + ")\n")
			}
		}
	}
	return b.String()
}

// applyPageMeta writes m into an index.html: <title>, the head tags and the text version inside #app.
func applyPageMeta(html []byte, site seoSite, m pageMeta, nonce string) []byte {
	esc := htmlpkg.EscapeString
	html = replaceTitle(html, m.Title)
	html = removeMetaDescription(html)

	var head strings.Builder
	_, _ = head.WriteString(`<meta name="description" content="` + esc(m.Description) + `" />`)
	if m.Index {
		_, _ = head.WriteString(`<meta name="robots" content="index,follow,max-image-preview:large" />`)
		_, _ = head.WriteString(`<link rel="canonical" href="` + esc(site.abs(m.Path)) + `" />`)
		_, _ = head.WriteString(`<meta property="og:type" content="` + esc(firstNonEmpty(m.OGType, "website")) + `" />`)
		_, _ = head.WriteString(`<meta property="og:site_name" content="` + esc(site.Name) + `" />`)
		_, _ = head.WriteString(`<meta property="og:title" content="` + esc(m.Title) + `" />`)
		_, _ = head.WriteString(`<meta property="og:description" content="` + esc(m.Description) + `" />`)
		_, _ = head.WriteString(`<meta property="og:url" content="` + esc(site.abs(m.Path)) + `" />`)
		_, _ = head.WriteString(`<meta property="og:locale" content="zh_CN" />`)
		if m.Image != "" {
			_, _ = head.WriteString(`<meta property="og:image" content="` + esc(m.Image) + `" />`)
		}
		_, _ = head.WriteString(`<meta name="twitter:card" content="summary" />`)
		ld := m.JSONLD
		if len(m.FAQ) > 0 {
			var items []any
			for _, f := range m.FAQ {
				items = append(items, map[string]any{
					"@type": "Question", "name": f.Question,
					"acceptedAnswer": map[string]any{"@type": "Answer", "text": f.Answer},
				})
			}
			ld = append(ld, map[string]any{"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": items})
		}
		for _, item := range ld {
			raw, err := json.Marshal(item)
			if err != nil {
				continue
			}
			// json.Marshal escapes <, > and & so the data cannot close the script element.
			_, _ = head.WriteString(`<script type="application/ld+json" nonce="` + esc(nonce) + `">` + string(raw) + `</script>`)
		}
	} else {
		_, _ = head.WriteString(`<meta name="robots" content="noindex,follow" />`)
	}
	_, _ = head.WriteString(`<style>.seo-shell{position:absolute;width:1px;height:1px;margin:-1px;padding:0;overflow:hidden;clip:rect(0,0,0,0);border:0}</style>`)
	html = bytes.Replace(html, []byte("</head>"), []byte(head.String()+"</head>"), 1)

	if !m.Index {
		return html
	}
	var body strings.Builder
	_, _ = body.WriteString(`<div id="app"><div class="seo-shell">`)
	if m.H1 != "" {
		_, _ = body.WriteString("<h1>" + esc(m.H1) + "</h1>")
	}
	for _, p := range m.Paragraphs {
		_, _ = body.WriteString("<p>" + esc(p) + "</p>")
	}
	if len(m.List) > 0 {
		_, _ = body.WriteString("<ul>")
		for _, item := range m.List {
			_, _ = body.WriteString("<li>" + esc(item) + "</li>")
		}
		_, _ = body.WriteString("</ul>")
	}
	if len(m.FAQ) > 0 {
		_, _ = body.WriteString("<section><h2>常见问题</h2>")
		for _, f := range m.FAQ {
			_, _ = body.WriteString("<h3>" + esc(f.Question) + "</h3><p>" + esc(f.Answer) + "</p>")
		}
		_, _ = body.WriteString("</section>")
	}
	if len(m.Links) > 0 {
		_, _ = body.WriteString("<nav><ul>")
		for _, l := range m.Links {
			_, _ = body.WriteString(`<li><a href="` + esc(l.Path) + `">` + esc(l.Label) + "</a></li>")
		}
		_, _ = body.WriteString("</ul></nav>")
	}
	_, _ = body.WriteString("</div></div>")
	return bytes.Replace(html, []byte(`<div id="app"></div>`), []byte(body.String()), 1)
}

func replaceTitle(html []byte, title string) []byte {
	start := bytes.Index(html, []byte("<title>"))
	end := bytes.Index(html, []byte("</title>"))
	if start == -1 || end == -1 || end <= start || title == "" {
		return html
	}
	var buf bytes.Buffer
	_, _ = buf.Write(html[:start])
	_, _ = buf.WriteString("<title>" + htmlpkg.EscapeString(title) + "</title>")
	_, _ = buf.Write(html[end+len("</title>"):])
	return buf.Bytes()
}

func removeMetaDescription(html []byte) []byte {
	start := bytes.Index(html, []byte(`<meta name="description"`))
	if start == -1 {
		return html
	}
	end := bytes.IndexByte(html[start:], '>')
	if end == -1 {
		return html
	}
	var buf bytes.Buffer
	_, _ = buf.Write(html[:start])
	_, _ = buf.Write(html[start+end+1:])
	return buf.Bytes()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// clipText collapses whitespace and cuts s to n runes (with "…").
func clipText(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n])) + "…"
}

// modelPriceLine is "gpt-5.5（GPT-按量）：输入 $5 / 输出 $30 / 缓存 $0.5（每 100 万 token）".
func modelPriceLine(m SEOModel) string {
	line := m.Name
	if m.Group != "" {
		line += "（" + m.Group + "）"
	}
	var parts []string
	for _, p := range []struct {
		label string
		v     *float64
	}{{"输入", m.Input}, {"输出", m.Output}, {"缓存", m.CacheRead}} {
		if p.v != nil {
			parts = append(parts, p.label+" $"+strconv.FormatFloat(*p.v, 'f', -1, 64))
		}
	}
	if len(parts) == 0 {
		return line
	}
	return line + "：" + strings.Join(parts, " / ") + "（每 100 万 token）"
}
