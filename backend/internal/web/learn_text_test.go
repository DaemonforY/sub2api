//go:build unit

package web

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestLessonTextFrom(t *testing.T) {
	page := `<html><head><title>结构化输出：让模型稳定返回 JSON · HiveGPT AI 学习</title><script src="/a.js"></script></head>
<body><nav>侧边栏 a1 a2</nav><main class="main"><div class="lesson-head">header</div>
<div style="position:relative;" class="vp-doc _learn_a_a3"><div><p>做应用时，模型的输出往往要交给程序。</p>
<h2 id="x">三种做法 <a class="header-anchor" href="#x">&#8203;</a></h2><table><tr><td>提示词 &amp; JSON</td></tr></table>
<button class="copy">复制</button><pre><code>{"strict": true}</code></pre><style>.x{}</style></div></div></main><footer>页脚</footer></body></html>`
	fsys := fstest.MapFS{"learn/a/a3.html": {Data: []byte(page)}}

	title, text := lessonTextFrom(fsys, "a3")
	require.Equal(t, "结构化输出：让模型稳定返回 JSON", title)
	require.Contains(t, text, "做应用时，模型的输出往往要交给程序。")
	require.Contains(t, text, "三种做法")
	require.Contains(t, text, "提示词 & JSON")
	require.Contains(t, text, `{"strict": true}`)
	for _, out := range []string{"侧边栏", "页脚", "复制", ".x{}", "<", "header"} {
		require.NotContains(t, text, out)
	}

	fsys["learn/bigdata/spark/lecture-03-shuffle.html"] = &fstest.MapFile{Data: []byte(strings.Replace(page, "做应用时", "Shuffle 的写和读", 1))}
	title, text = lessonTextFrom(fsys, "bigdata/spark/lecture-03-shuffle")
	require.NotEmpty(t, title)
	require.Contains(t, text, "Shuffle 的写和读")
	title, text = lessonTextFrom(fsys, "bigdata/spark/../../a/a3")
	require.Empty(t, title+text)

	title, text = lessonTextFrom(fsys, "../etc")
	require.Empty(t, title+text)
	title, text = lessonTextFrom(fsys, "b9")
	require.Empty(t, title+text)

}

func TestLessonTextFromBuiltSite(t *testing.T) {
	dir := "../../../learn/.vitepress/dist"
	if _, err := os.Stat(dir + "/a/a6.html"); err != nil {
		t.Skip("learn site not built")
	}
	learnTextCache.Delete("a6")
	sub := fstest.MapFS{}
	raw, err := os.ReadFile(dir + "/a/a6.html")
	require.NoError(t, err)
	sub["learn/a/a6.html"] = &fstest.MapFile{Data: raw}
	title, text := lessonTextFrom(sub, "a6")
	require.Equal(t, "Agent 循环：让模型自己一步步完成任务", title)
	require.Contains(t, text, "最小的 Agent 循环")
	require.Contains(t, text, "max_steps")
	require.False(t, strings.Contains(text, "延伸阅读"), "sidebar leaked in")
	t.Logf("%d chars: %.300s", len([]rune(text)), text)
	learnTextCache.Delete("a6")
}

func TestLearnPagesFrom(t *testing.T) {
	page := func(title, body string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(`<html><head><title>` + title + ` · HiveGPT AI 学习</title></head><body><main><div class="vp-doc"><p>` + body + `</p></div></main></body></html>`)}
	}
	fsys := fstest.MapFS{
		"learn/index.html":         page("AI 学习", "首页"),
		"learn/editor/index.html":  page("公众号排版使用教程", "AppID 在开发者平台"),
		"learn/codex/hivegpt.html": page("接入 HiveGPT", "配置 config.toml"),
		"learn/404.html":           page("404", "找不到"),
		"learn/assets/app.html":    page("x", "y"),
		"learn/empty.html":         {Data: []byte(`<html><head><title>空</title></head><body></body></html>`)},
		"index.html":               page("主站", "不是学习站"),
	}
	pages := learnPagesFrom(fsys)
	byURL := map[string]LearnPage{}
	for _, p := range pages {
		byURL[p.URL] = p
	}
	require.Len(t, pages, 3)
	require.Equal(t, "公众号排版使用教程", byURL["/learn/editor/"].Title)
	require.Contains(t, byURL["/learn/editor/"].Text, "AppID 在开发者平台")
	require.Contains(t, byURL, "/learn/")
	require.Contains(t, byURL, "/learn/codex/hivegpt")
	require.Nil(t, learnPagesFrom(nil))
}
