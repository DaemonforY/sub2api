//go:build unit

package web

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeSEOContent struct{ plazaOff bool }

func (f fakeSEOContent) Models(context.Context) ([]SEOModel, bool) {
	if f.plazaOff {
		return nil, false
	}
	in, out := 5.0, 30.0
	return []SEOModel{{Name: "gpt-5.5", Group: "GPT-按量", Input: &in, Output: &out}}, true
}

func (fakeSEOContent) Courses(context.Context) ([]SEOPage, error) {
	return []SEOPage{{Path: "/courses/ai-coding", Title: "AI 编程实战", Description: "从零用 Codex 写项目", Price: 99, Updated: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}}, nil
}

func (fakeSEOContent) VideoWorks(context.Context) ([]SEOPage, error) {
	return []SEOPage{{Path: "/w/abc", Title: "区块链为什么不能篡改"}}, nil
}

func (fakeSEOContent) VideoWork(_ context.Context, id string) (*SEOPage, error) {
	if id != "abc" {
		return nil, nil
	}
	return &SEOPage{Path: "/w/abc", Title: "区块链为什么不能篡改", Description: "60 秒讲清楚"}, nil
}

func (fakeSEOContent) FAQ() []SEOFAQ {
	return []SEOFAQ{{Question: "怎么创建 API Key", Answer: "进入「API 密钥」点创建 <b>", Link: "/keys"}}
}

var testSite = seoSite{Name: "HiveGPT", Origin: "https://hivegpt.cn", Logo: "https://hivegpt.cn/apple-touch-icon.png"}

const testIndexHTML = `<!doctype html><html><head><title>HiveGPT - AI API Gateway</title></head><body><div id="app"></div></body></html>`

func TestMainPageMeta(t *testing.T) {
	ctx := context.Background()

	home := mainPageMeta(ctx, testSite, fakeSEOContent{}, "/home")
	require.True(t, home.Index)
	require.Equal(t, "/", home.Path)
	require.Len(t, home.FAQ, 1)

	course := mainPageMeta(ctx, testSite, fakeSEOContent{}, "/courses/ai-coding/")
	require.True(t, course.Index)
	require.Equal(t, "AI 编程实战", course.H1)
	require.Contains(t, course.Title, "AI 编程实战")

	missing := mainPageMeta(ctx, testSite, fakeSEOContent{}, "/courses/nope")
	require.False(t, missing.Index)

	for _, p := range []string{"/dashboard", "/admin/users", "/keys", "/whatever"} {
		require.False(t, mainPageMeta(ctx, testSite, fakeSEOContent{}, p).Index, p)
	}
	require.True(t, mainPageMeta(ctx, testSite, nil, "/pricing").Index)
}

func TestApplyPageMetaIndexable(t *testing.T) {
	m := mainPageMeta(context.Background(), testSite, fakeSEOContent{}, "/")
	out := string(applyPageMeta([]byte(testIndexHTML), testSite, m, "NONCE"))

	require.Contains(t, out, "<title>HiveGPT - AI 模型 API 平台")
	require.Equal(t, 1, strings.Count(out, "<title>"))
	require.Contains(t, out, `<link rel="canonical" href="https://hivegpt.cn/" />`)
	require.Contains(t, out, `<meta name="robots" content="index,follow`)
	require.Contains(t, out, `"@type":"FAQPage"`)
	require.Contains(t, out, `<script type="application/ld+json" nonce="NONCE">`)
	// JSON-LD escapes markup so the answer cannot end the script element.
	require.NotContains(t, out, "创建 <b>")
	// The text version sits inside #app, escaped.
	require.Contains(t, out, `<div id="app"><div class="seo-shell"><h1>`)
	require.Contains(t, out, "创建 &lt;b&gt;")
	require.Contains(t, out, `<a href="/pricing">`)
}

func TestApplyPageMetaNoindex(t *testing.T) {
	m := mainPageMeta(context.Background(), testSite, nil, "/dashboard")
	out := string(applyPageMeta([]byte(testIndexHTML), testSite, m, "N"))
	require.Contains(t, out, `<meta name="robots" content="noindex,follow" />`)
	require.NotContains(t, out, "canonical")
	require.Contains(t, out, `<div id="app"></div>`)
}

func TestApplyPageMetaReplacesDescription(t *testing.T) {
	html := `<html><head><title>x</title><meta name="description" content="old" /></head><body><div id="app"></div></body></html>`
	m := videoPageMeta(context.Background(), seoSite{Name: "HiveGPT 视频", Origin: "https://video.hivegpt.cn"}, fakeSEOContent{}, nil, "/")
	out := string(applyPageMeta([]byte(html), testSite, m, ""))
	require.NotContains(t, out, `content="old"`)
	require.Equal(t, 1, strings.Count(out, `<meta name="description"`))
}

func TestVideoPageMeta(t *testing.T) {
	ctx := context.Background()
	site := seoSite{Name: "HiveGPT 视频", Origin: "https://video.hivegpt.cn"}
	tools := []SEOTool{{Slug: "logo-animation", Name: "Logo 动画生成器", Headline: "让 Logo 动起来", Summary: "上传或描述 Logo"}}

	tool := videoPageMeta(ctx, site, fakeSEOContent{}, tools, "/tools/logo-animation")
	require.True(t, tool.Index)
	require.Equal(t, "Logo 动画生成器", tool.H1)
	require.False(t, videoPageMeta(ctx, site, fakeSEOContent{}, tools, "/tools/nope").Index)

	work := videoPageMeta(ctx, site, fakeSEOContent{}, tools, "/w/abc")
	require.True(t, work.Index)
	require.Equal(t, "区块链为什么不能篡改", work.H1)
	require.False(t, videoPageMeta(ctx, site, fakeSEOContent{}, tools, "/w/zzz").Index)
	require.False(t, videoPageMeta(ctx, site, fakeSEOContent{}, tools, "/p/123").Index)
}

func TestRobotsTxt(t *testing.T) {
	out := robotsTxt(testSite, mainRobotsDisallow, []string{"/sitemap.xml", "/learn/sitemap.xml"})
	require.Contains(t, out, "User-agent: *\n")
	require.Contains(t, out, "User-agent: GPTBot\n")
	require.Contains(t, out, "Disallow: /admin\n")
	require.Contains(t, out, "Sitemap: https://hivegpt.cn/learn/sitemap.xml\n")
	// One group: every User-agent line comes before the rules.
	require.Less(t, strings.LastIndex(out, "User-agent:"), strings.Index(out, "Allow: /"))
}

func TestSitemapXML(t *testing.T) {
	raw := sitemapXML(testSite, mainSitemapPages(context.Background(), fakeSEOContent{}))
	var set struct {
		URLs []sitemapURL `xml:"url"`
	}
	require.NoError(t, xml.Unmarshal(raw, &set))
	var locs []string
	for _, u := range set.URLs {
		locs = append(locs, u.Loc)
	}
	require.Contains(t, locs, "https://hivegpt.cn/")
	require.Contains(t, locs, "https://hivegpt.cn/pricing")
	require.Contains(t, locs, "https://hivegpt.cn/courses/ai-coding")
	require.NotContains(t, locs, "https://hivegpt.cn/login")
}

func TestMainLLMsTxt(t *testing.T) {
	learn := []SEOLearnPage{{URL: "/learn/connect/sdk", Title: "用 SDK 调用", Text: "Python 示例 ..."}}
	short := mainLLMsTxt(context.Background(), testSite, fakeSEOContent{}, learn, false)
	require.True(t, strings.HasPrefix(short, "# HiveGPT\n\n> "))
	require.Contains(t, short, "[用 SDK 调用](https://hivegpt.cn/learn/connect/sdk)")
	require.Contains(t, short, "[AI 编程实战](https://hivegpt.cn/courses/ai-coding)")
	require.Contains(t, short, "/llms-full.txt")
	require.NotContains(t, short, "### 怎么创建 API Key")

	full := mainLLMsTxt(context.Background(), testSite, fakeSEOContent{}, learn, true)
	require.Contains(t, full, "### 怎么创建 API Key")
	require.Contains(t, full, "Python 示例")
}

func TestModelPlazaFollowsTheSwitch(t *testing.T) {
	ctx := context.Background()

	on := mainPageMeta(ctx, testSite, fakeSEOContent{}, "/model-plaza")
	require.True(t, on.Index)
	require.Equal(t, []string{"gpt-5.5（GPT-按量）：输入 $5 / 输出 $30（每 100 万 token）"}, on.List)
	require.Contains(t, mainLLMsTxt(ctx, testSite, fakeSEOContent{}, nil, false), "gpt-5.5（GPT-按量）：输入 $5 / 输出 $30")

	off := fakeSEOContent{plazaOff: true}
	require.False(t, mainPageMeta(ctx, testSite, off, "/model-plaza").Index)
	for _, p := range mainSitemapPages(ctx, off) {
		require.NotEqual(t, "/model-plaza", p.Path)
	}
	require.NotContains(t, mainLLMsTxt(ctx, testSite, off, nil, false), "model-plaza")
	for _, l := range mainPageMeta(ctx, testSite, off, "/").Links {
		require.NotEqual(t, "/model-plaza", l.Path)
	}
	require.False(t, mainPageMeta(ctx, testSite, nil, "/model-plaza").Index)
}
