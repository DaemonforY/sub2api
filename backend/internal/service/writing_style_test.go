//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTMLToTextParagraphs(t *testing.T) {
	src := `<section><p>大家好，我是<strong>J哥</strong>。</p><p><br></p><h2>灵感从哪来？</h2>` +
		`<p>第一行<br>第二行</p><script>alert(1)</script><style>p{}</style><p>&nbsp;</p><ul><li>一</li><li>二</li></ul></section>`
	require.Equal(t, "大家好，我是J哥。\n\n灵感从哪来？\n\n第一行\n\n第二行\n\n一\n\n二", htmlToText(src))
}

func TestStyleCheckLimits(t *testing.T) {
	long := strings.Repeat("字", 400)
	_, _, err := styleCheck(nil)
	require.ErrorIs(t, err, ErrStyleArticles)

	_, _, err = styleCheck([]StyleArticle{{Text: "太短了"}})
	require.ErrorIs(t, err, ErrStyleArticles, "under 300 characters in total")

	six := make([]StyleArticle, 6)
	for i := range six {
		six[i] = StyleArticle{Text: long}
	}
	_, _, err = styleCheck(six)
	require.ErrorIs(t, err, ErrStyleArticles, "more than 5 articles")

	out, total, err := styleCheck([]StyleArticle{{Title: " 标题 ", Text: " " + strings.Repeat("长", 7000) + " "}, {Text: "   "}})
	require.NoError(t, err)
	require.Len(t, out, 1, "blank articles are dropped")
	require.Equal(t, "标题", out[0].Title)
	require.Equal(t, styleArticleMax, total, "each article is clipped")
}

func TestParseStyleResult(t *testing.T) {
	raw := "```json\n" + `{"archetype":"理工科段子手","portrait":"像饭桌上聊天","tags":["自嘲"," ","干货","口语","真诚","故事","多余"],` +
		`"avoid":["赋能"],"prompt":"请你用第一人称……"}` + "\n```"
	r, ok := parseStyleResult(raw)
	require.True(t, ok)
	require.Equal(t, "理工科段子手", r.Archetype)
	require.Equal(t, []string{"自嘲", "干货", "口语", "真诚", "故事"}, r.Tags, "blanks dropped, at most 5")
	require.Equal(t, []string{}, r.Openings, "missing lists come back empty, not null")

	_, ok = parseStyleResult(`{"portrait":"","prompt":"x"}`)
	require.False(t, ok, "a portrait is required")
	_, ok = parseStyleResult("抱歉，我不能")
	require.False(t, ok)
}
