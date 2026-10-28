//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShihuaCutText(t *testing.T) {
	page := "版本信息\n\n草长莺飞二月天， 拂堤杨柳醉春烟。 \n儿童散学归来早， 忙趁东风放纸鸢。\n\n↑ 注释"
	cut, ok := shihuaCutText(page, "草长莺飞", "放纸鸢")
	require.True(t, ok)
	require.Equal(t, "草长莺飞二月天， 拂堤杨柳醉春烟。 \n儿童散学归来早， 忙趁东风放纸鸢。", cut, "keeps the closing mark")
	// Matching ignores line breaks inside the phrases.
	_, ok = shihuaCutText("草长莺飞二月天，\n拂堤杨柳", "二月天拂堤", "杨柳")
	require.True(t, ok)
	_, ok = shihuaCutText(page, "大漠孤烟", "放纸鸢")
	require.False(t, ok)
	_, ok = shihuaCutText(page, "", "放纸鸢")
	require.False(t, ok)
}

func TestShihuaTidyText(t *testing.T) {
	text, ok := shihuaTidyText("草长莺飞二月天， 拂堤杨柳醉春烟。 \n\n儿童散学归来早， 忙趁东风放纸鸢。", "")
	require.True(t, ok)
	require.Equal(t, "草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。", text)

	// One clause per line is joined into couplets.
	text, ok = shihuaTidyText("草满池塘水满陂，\n山衔落日浸寒漪。\n牧童归去横牛背，\n短笛无腔信口吹。", "")
	require.True(t, ok)
	require.Equal(t, "草满池塘水满陂，山衔落日浸寒漪。\n牧童归去横牛背，短笛无腔信口吹。", text)

	// Regular 7-character verse without punctuation.
	text, ok = shihuaTidyText("草铺横野六七里    笛弄晚风三四声\n归来饱饭黄昏后    不脱蓑衣卧月明", "")
	require.True(t, ok)
	require.Equal(t, "草铺横野六七里，笛弄晚风三四声。\n归来饱饭黄昏后，不脱蓑衣卧月明。", text)

	// Irregular lines take the model's punctuation when the characters are the same.
	ci := "茅檐低小 溪上青青草\n醉里吴音相媚好 白发谁家翁媪"
	ai := "茅檐低小，溪上青青草。\n醉里吴音相媚好，白发谁家翁媪？"
	text, ok = shihuaTidyText(ci, ai)
	require.True(t, ok)
	require.Equal(t, ai, text)
	// …and stay bare when they are not.
	text, ok = shihuaTidyText(ci, "茅檐低小，溪上青青草。\n醉里吴音相媚好，白发谁家老翁？")
	require.False(t, ok)
	require.Equal(t, "茅檐低小\n溪上青青草\n醉里吴音相媚好\n白发谁家翁媪", text)
}

func TestShihuaDiff(t *testing.T) {
	require.Empty(t, shihuaDiff("春眠不觉晓，处处闻啼鸟。", "春眠不觉晓\n处处闻啼鸟"))
	require.Equal(t, []ShihuaTextDiff{{Source: "支", AI: "枝"}}, shihuaDiff("一支红杏出墙来", "一枝红杏出墙来"))
	require.Equal(t, []ShihuaTextDiff{{Source: "", AI: "新"}}, shihuaDiff("树头绿未成阴", "树头新绿未成阴"))
	require.Equal(t, []ShihuaTextDiff{{Source: "上", AI: "坐"}, {Source: "他", AI: "生"}},
		shihuaDiff("收篙停棹上船中怪他无雨", "收篙停棹坐船中怪生无雨"))
}

func TestShihuaDraftSlugAndDynasty(t *testing.T) {
	require.Equal(t, "cun-ju", shihuaDraftSlug(shihuaDraftEntry{Slug: " Cun Ju "}))
	require.Equal(t, "cun-ju", shihuaDraftSlug(shihuaDraftEntry{Slug: "cun--ju!"}))
	s := shihuaDraftSlug(shihuaDraftEntry{Slug: "村居", Title: "村居", Author: "高鼎"})
	require.True(t, strings.HasPrefix(s, "work-"))
	require.True(t, shihuaSlugRe.MatchString(s))
	require.True(t, shihuaAncient("南宋"))
	require.True(t, shihuaAncient("唐代"))
	require.False(t, shihuaAncient("现代"))
	require.False(t, shihuaAncient(""))
}

// fakeWikisource serves api.php: parse by page title, search by query.
func fakeWikisource(t *testing.T, pages map[string]string, searches map[string][]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		switch q.Get("action") {
		case "parse":
			require.Equal(t, "zh-hans", q.Get("variant"))
			html, ok := pages[q.Get("page")]
			if !ok {
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "missingtitle"}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"parse": map[string]any{"text": html}})
		case "query":
			hits := []map[string]any{}
			for _, title := range searches[q.Get("srsearch")] {
				hits = append(hits, map[string]any{"title": title})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"query": map[string]any{"search": hits}})
		default:
			http.Error(w, "bad", http.StatusBadRequest)
		}
	}))
}

func TestShihuaDraftWorks(t *testing.T) {
	s, repo := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7})
	require.NoError(t, err)
	existing := shihuaTestWork()
	_, err = s.CreateWork(ctx, existing)
	require.NoError(t, err)
	_ = repo

	wiki := fakeWikisource(t, map[string]string{
		"村居 (高鼎)": `<div><p>草长莺飞二月天， 拂堤杨柳醉春烟。<br/>儿童散学归来早， 忙趁东风放纸鸢。</p><sup class="reference">[1]</sup></div>`,
		"答鍾弱翁":    `<p>草铺横野六七里    笛弄晚风三四声</p><p>归来饱饭黄昏后    不脱蓑衣卧月明</p>`,
		"遊園不值":    `<p>应怜屐齿印苍苔<br>小扣柴扉久不开<br>春色满园关不住<br>一支红杏出墙来</p>`,
		"宿新市徐公店":  `<p>篱落疏疏一迳深，树头新绿未成阴。<br>儿童急走追黄蝶，飞入菜花无处寻。</p>`,
		"池上二絕":    `<p>山僧对棋坐，局上竹阴清。<br>映竹无人见，时闻下子声。</p><p>小娃撑小艇，偷采白莲回。<br>不解藏踪迹，浮萍一道开。</p>`,
	}, map[string][]string{
		"草鋪橫野六七里": {"千家詩/卷三", "答鍾弱翁"},
	})
	defer wiki.Close()
	s.wikisource = newShihuaWikisource(wiki.URL)
	s.wikisource.gap = 0

	var prompts []string
	s.chat = func(_ context.Context, key, model string, messages []LearnMessage) (string, error) {
		prompts = append(prompts, messages[1].Content)
		require.Equal(t, "sk-test", key)
		return "```json\n" + `{"works":[
			{"title":"村居","author":"高鼎","dynasty":"清","genre":"shi","stage":"primary","grade":"二下","slug":"cun-ju","page":"村居 (高鼎)","search":"草長鶯飛二月天","first":"草长莺飞","last":"放纸鸢","text":"草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。","reason":"早春放风筝"},
			{"title":"牧童","author":"吕岩","dynasty":"唐","genre":"shi","slug":"mu-tong","page":"牧童 (呂岩)","search":"草鋪橫野六七里","first":"草铺横野","last":"卧月明","text":"草铺横野六七里，笛弄晚风三四声。\n归来饱饭黄昏后，不脱蓑衣卧月明。"},
			{"title":"游园不值","author":"叶绍翁","dynasty":"宋代","genre":"shi","slug":"you-yuan-bu-zhi","page":"遊園不值","first":"应怜屐齿","last":"出墙来","text":"应怜屐齿印苍苔，小扣柴扉久不开。\n春色满园关不住，一枝红杏出墙来。"},
			{"title":"所见","author":"袁枚","dynasty":"清","genre":"shi","slug":"suo-jian","page":"所見","first":"牧童骑黄牛","last":"闭口立","text":"牧童骑黄牛，歌声振林樾。\n意欲捕鸣蝉，忽然闭口立。"},
			{"title":"宿新市徐公店","author":"杨万里","dynasty":"南宋","slug":"su-xin-shi-xu-gong-dian","page":"宿新市徐公店","first":"篱落疏疏一径","last":"无处寻","text":"篱落疏疏一径深，树头新绿未成阴。\n儿童急走追黄蝶，飞入菜花无处寻。"},
			{"title":"池上","author":"白居易","dynasty":"唐","slug":"chi-shang","page":"池上 (白居易)","pages":["池上二絕"],"first":"小娃撑小艇","last":"一道开","text":"小娃撑小艇，偷采白莲回。\n不解藏踪迹，浮萍一道开。"},
			{"title":"某首现代诗","author":"某诗人","dynasty":"现代","slug":"modern","text":"不收"},
			{"title":"山居秋暝","author":"王维","dynasty":"唐","slug":"shan-ju-qiu-ming","text":"空山新雨后"},
			{"title":"村居","author":"高鼎","dynasty":"清","slug":"cun-ju","page":"村居 (高鼎)","first":"草长莺飞","last":"放纸鸢","text":"草长莺飞二月天"}
		]}` + "\n```", nil
	}

	_, err = s.DraftWorks(ctx, ShihuaDraftRequest{})
	require.ErrorIs(t, err, ErrShihuaDraftEmpty)
	_, err = s.DraftWorks(ctx, ShihuaDraftRequest{Topic: "春天", Stage: "kindergarten"})
	require.ErrorIs(t, err, ErrShihuaInvalid)

	res, err := s.DraftWorks(ctx, ShihuaDraftRequest{Topic: "小学春天田园、有孩子的古诗", Stage: "primary", Count: 99})
	require.NoError(t, err)
	require.Contains(t, prompts[0], "数量：30 篇", "count is capped")
	require.Contains(t, prompts[0], "学段：小学")
	require.Len(t, res.Items, 9)
	by := map[string]ShihuaDraftItem{}
	for _, it := range res.Items {
		by[it.Work.Slug] = it
	}

	cunju := by["cun-ju"]
	require.Equal(t, ShihuaDraftMatch, cunju.Status)
	require.Equal(t, "草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。", cunju.Work.FullText)
	require.Equal(t, "村居 (高鼎)", cunju.Page)
	require.Contains(t, cunju.PageURL, wiki.URL+"/wiki/")
	require.Contains(t, cunju.Work.Sources, "维基文库《村居 (高鼎)》")
	require.Contains(t, cunju.Work.VariantNote, "年级「二下」是 AI 猜的")
	require.Equal(t, "primary", cunju.Work.Stage)
	require.Equal(t, "早春放风筝", cunju.Reason)

	mutong := by["mu-tong"]
	require.Equal(t, ShihuaDraftMatch, mutong.Status, "found by search, punctuated as 7-character verse")
	require.Equal(t, "答鍾弱翁", mutong.Page)
	require.Equal(t, "草铺横野六七里，笛弄晚风三四声。\n归来饱饭黄昏后，不脱蓑衣卧月明。", mutong.Work.FullText)

	youyuan := by["you-yuan-bu-zhi"]
	require.Equal(t, ShihuaDraftDiff, youyuan.Status)
	require.Equal(t, []ShihuaTextDiff{{Source: "支", AI: "枝"}}, youyuan.Diffs)
	require.Contains(t, youyuan.Work.FullText, "一支红杏", "the poster text is 维基文库's, not the model's")
	require.Contains(t, youyuan.Work.VariantNote, "维基文库「支」/ AI「枝」")
	require.Equal(t, "宋", youyuan.Work.Dynasty)

	suojian := by["suo-jian"]
	require.Equal(t, ShihuaDraftAIOnly, suojian.Status)
	require.Equal(t, "牧童骑黄牛，歌声振林樾。\n意欲捕鸣蝉，忽然闭口立。", suojian.Work.FullText)
	require.Contains(t, suojian.Work.Sources, "AI 默写")

	xugong := by["su-xin-shi-xu-gong-dian"]
	require.Equal(t, ShihuaDraftDiff, xugong.Status, "found with the short anchor despite the variant character")
	require.Equal(t, []ShihuaTextDiff{{Source: "迳", AI: "径"}}, xugong.Diffs)

	chishang := by["chi-shang"]
	require.Equal(t, ShihuaDraftMatch, chishang.Status, "found on the second page guess; only the second poem is cut")
	require.Equal(t, "池上二絕", chishang.Page)
	require.Equal(t, "小娃撑小艇，偷采白莲回。\n不解藏踪迹，浮萍一道开。", chishang.Work.FullText)

	require.Equal(t, ShihuaDraftRejected, by["modern"].Status)
	require.Contains(t, by["modern"].Problem, "版权")
	require.Equal(t, ShihuaDraftExists, by["shan-ju-qiu-ming"].Status)
	// The same slug twice gets a suffix; its memory text is only the first line, so it differs.
	require.Equal(t, ShihuaDraftDiff, by["cun-ju-2"].Status)
	require.Equal(t, cunju.Work.FullText, by["cun-ju-2"].Work.FullText)

	// What the page keeps imports like a file would.
	keep := []ShihuaWorkInput{cunju.Work, mutong.Work, youyuan.Work, suojian.Work, xugong.Work}
	imp, err := s.Import(ctx, keep, true)
	require.NoError(t, err)
	require.Equal(t, 5, imp.Created)
	require.Empty(t, imp.Errors)
}

func TestShihuaDraftWorksModelErrors(t *testing.T) {
	s, _ := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.DraftWorks(ctx, ShihuaDraftRequest{Titles: "村居 高鼎"})
	require.Error(t, err, "no key configured")
	require.Contains(t, err.Error(), "API Key")

	_, err = s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7})
	require.NoError(t, err)
	s.chat = func(context.Context, string, string, []LearnMessage) (string, error) { return "没有 JSON", nil }
	_, err = s.DraftWorks(ctx, ShihuaDraftRequest{Titles: "村居 高鼎"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "没有返回可用的篇目")
}

func TestShihuaDraftWorksWikisourceDown(t *testing.T) {
	s, _ := newShihuaTestService(t)
	ctx := context.Background()
	_, err := s.SaveSettings(ctx, 1, ShihuaSettings{KeyID: 7})
	require.NoError(t, err)
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "nope", http.StatusForbidden) }))
	defer down.Close()
	s.wikisource = newShihuaWikisource(down.URL)
	s.wikisource.gap = 0
	s.chat = func(context.Context, string, string, []LearnMessage) (string, error) {
		return `{"works":[{"title":"春晓","author":"孟浩然","dynasty":"唐","slug":"chun-xiao","page":"春曉 (孟浩然)","first":"春眠不觉晓","last":"知多少","text":"春眠不觉晓，处处闻啼鸟。\n夜来风雨声，花落知多少。"}]}`, nil
	}
	res, err := s.DraftWorks(ctx, ShihuaDraftRequest{Titles: "春晓 孟浩然"})
	require.NoError(t, err)
	it := res.Items[0]
	require.Equal(t, ShihuaDraftAIOnly, it.Status)
	require.Contains(t, it.Problem, "访问维基文库失败", "unreachable is not reported as not found")
	require.Contains(t, it.Problem, "403")
}

func TestShihuaWikisourceRetriesBusySearch(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Contains(t, r.Header.Get("User-Agent"), "https://hivegpt.cn")
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "cirrussearch-too-busy-error"}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"query": map[string]any{"search": []map[string]any{{"title": "村居 (高鼎)"}}}})
	}))
	defer srv.Close()
	ws := newShihuaWikisource(srv.URL)
	ws.gap = 0
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	hits, err := ws.search(ctx, "村居")
	require.NoError(t, err)
	require.Equal(t, []string{"村居 (高鼎)"}, hits)
	require.Equal(t, 2, calls)
}
