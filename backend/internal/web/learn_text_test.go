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
