//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type tietuRepoStub struct {
	mu   sync.Mutex
	next int64
	rows map[int64]TietuProject
}

func (r *tietuRepoStub) clone(p TietuProject) TietuProject {
	b, _ := json.Marshal(p.TietuData)
	var d TietuData
	_ = json.Unmarshal(b, &d)
	p.TietuData = d
	return p
}

func (r *tietuRepoStub) Create(_ context.Context, p *TietuProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	p.ID, p.CreatedAt, p.UpdatedAt = r.next, time.Now(), time.Now()
	r.rows[p.ID] = r.clone(*p)
	return nil
}

func (r *tietuRepoStub) Get(_ context.Context, userID, id int64) (*TietuProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.rows[id]
	if !ok || p.UserID != userID {
		return nil, nil
	}
	c := r.clone(p)
	return &c, nil
}

func (r *tietuRepoStub) List(_ context.Context, userID int64, _ int) ([]TietuProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []TietuProject{}
	for _, p := range r.rows {
		if p.UserID == userID {
			out = append(out, r.clone(p))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (r *tietuRepoStub) Save(_ context.Context, p *TietuProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[p.ID] = r.clone(*p)
	return nil
}

func (r *tietuRepoStub) CountActive(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.rows {
		if p.UserID == userID && tietuActive(p.Status) {
			n++
		}
	}
	return n, nil
}

func (r *tietuRepoStub) FailActive(context.Context, string) (int64, error) { return 0, nil }
func (r *tietuRepoStub) Delete(_ context.Context, _ int64, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, id)
	return nil
}
func (r *tietuRepoStub) DeleteBefore(context.Context, time.Time) ([]int64, error) { return nil, nil }

type tietuPoemsStub struct{ poems []LibPoem }

func (s tietuPoemsStub) SearchPoems(_ context.Context, q LibPoemQuery) ([]LibPoem, bool, error) {
	out := []LibPoem{}
	for _, p := range s.poems {
		ok := true
		for _, w := range strings.Fields(q.Q) {
			if !strings.Contains(p.Title+p.Author+p.Content, w) {
				ok = false
			}
		}
		if ok {
			out = append(out, p)
		}
	}
	return out, false, nil
}

type tietuWikiStub struct{ pages map[string]string }

func (w tietuWikiStub) pageText(_ context.Context, title string) (string, bool, error) {
	t, ok := w.pages[title]
	return t, ok, nil
}
func (w tietuWikiStub) search(context.Context, string) ([]string, error) {
	out := []string{}
	for k := range w.pages {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}
func (w tietuWikiStub) pageURL(title string) string { return "https://zh.wikisource.org/wiki/" + title }

const tietuTestPlan = `{"kind":"series","kind_reason":"有渔人进村又出村的完整故事","concepts":[{"name":"世外村落","idea":"洞口豁然开朗"},{"name":"迷路","idea":"雾中寻路"}],"concept":0,
"character":"头戴斗笠、身披蓑衣、红短褂的中年渔人","color":"桃花粉、石绿为主，朱红点睛",
"cover":{"scene":"高位俯瞰桃花溪和村落","details":["水车","赶鹅的孩子"],"find":"洞口的小渔船"},
"panels":[{"caption":"缘溪行","line":"渔人沿溪而上","scene":"小船在青山间","details":["鸬鹚"],"find":"船头的鸬鹚"},
{"caption":"忽逢桃花林","line":"两岸都是桃花","scene":"桃花林","details":["鱼吃花瓣"],"find":"吃花瓣的鱼"},
{"caption":"豁然开朗","line":"村落出现","scene":"村落全景","details":["牛耕"],"find":"牛耕"}],
"titles":["课本里的桃花源，我画成了连环画","标题二"],"question":"最想住进哪一幕？"}`

type tietuFakes struct {
	mu        sync.Mutex
	chats     []string
	draws     int
	edits     int
	refs      int
	variantTo string
}

func newTietuForTest(t *testing.T, poems []LibPoem, wiki map[string]string) (*TietuService, *tietuRepoStub, *tietuFakes) {
	t.Helper()
	keys := &learnKeysStub{keys: []APIKey{{ID: 7, UserID: 5, Name: "画图", Key: "sk-user5", Status: StatusActive}}}
	learn, _ := newL2Service(t, "http://127.0.0.1:1", LearnSources{Keys: keys})
	repo := &tietuRepoStub{rows: map[int64]TietuProject{}}
	svc := NewTietuService(repo, learn, tietuPoemsStub{poems: poems}, t.TempDir())
	svc.wiki = tietuWikiStub{pages: wiki}
	f := &tietuFakes{}
	png := articlePNG(t)
	svc.chat = func(_ context.Context, _, _ string, msgs []LearnMessage, _ int) (string, *LearnTutorResult, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		sys := msgs[0].Content
		f.chats = append(f.chats, sys[:min(len(sys), 30)])
		res := &LearnTutorResult{Model: "gpt-test", PromptTokens: 10, CompletionTokens: 5}
		switch {
		case strings.Contains(sys, "按主题推荐"):
			return `{"suggestions":[{"title":"桃花源记","author":"陶渊明","reason":"有完整故事"},{"title":"不存在的诗","author":"某人","reason":"x"}]}`, res, nil
		case strings.Contains(sys, "检索线索"):
			if strings.Contains(msgs[1].Content, "木兰") {
				return `{"title":"木兰诗","aliases":["木兰辞"],"author":"北朝民歌","dynasty":"北朝","first_line":"唧唧复唧唧"}`, res, nil
			}
			return "", res, nil
		case strings.Contains(sys, "两个来源里有几处不同"):
			return `{"choices":[{"index":0,"pick":` + f.variantTo + `,"why":"课本写法"}]}`, res, nil
		case strings.Contains(sys, "对照现行语文课本"):
			if strings.Contains(msgs[1].Content, "两兔傍地走") {
				return `{"changes":[{"from":"两兔傍地走","to":"双兔傍地走","why":"课本作双兔"},{"from":"雌兔眼迷离","to":"雌兔眼迷离","why":"没有变化不改"},{"from":"安能辨我是雄雌","to":"安能辨我","why":"字数不同不改"}]}`, res, nil
			}
			return `{"changes":[]}`, res, nil
		case strings.Contains(sys, "质检编辑"):
			require.Len(t, msgs[1].Images, 1, "the picture goes to the vision model")
			if strings.Contains(msgs[1].Content, "第 2 张") {
				return `{"ok":false,"issues":["提前出现了村落"],"find_visible":false,"find_seen":"水面的花瓣","focal":"clear"}`, res, nil
			}
			return `{"ok":true,"issues":[],"find_visible":true,"focal":"clear"}`, res, nil
		case strings.Contains(sys, "艺术总监"):
			return "```json\n" + tietuTestPlan + "\n```", res, nil
		}
		return "", res, nil
	}
	svc.draw = func(_ context.Context, _, prompt string) ([]byte, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.draws++
		require.Contains(t, prompt, "只写主标题「桃花源记」和作者名「陶渊明」")
		require.Contains(t, prompt, "不要写诗句正文")
		return png, nil
	}
	svc.edit = func(_ context.Context, _, prompt string, ref []byte) ([]byte, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.edits++
		if len(ref) > 0 {
			f.refs++
		}
		require.Contains(t, prompt, "参考图只用来保持一致")
		return png, nil
	}
	return svc, repo, f
}

func waitTietu(t *testing.T, svc *TietuService, id int64) {
	t.Helper()
	require.Eventually(t, func() bool { return !svc.running(id) }, 5*time.Second, 5*time.Millisecond)
}

var tietuTaohua = LibPoem{ID: 342532, Title: "桃花源记", Author: "陶渊明", Dynasty: "魏晋",
	Content: "晋太元中，武陵人捕鱼为业。缘溪行，忘路之远近。(一作：沿溪行)\n忽逢桃花林，夹岸数百步。", Famous: true, Popularity: 5}

func TestTietuFullRunSeries(t *testing.T) {
	ctx := context.Background()
	wiki := map[string]string{"桃花源記": "前言\n晋太元中，武陵人捕鱼为业。缘溪行，忘路之远近。忽逢桃花林，夹岸数百步。\n后记"}
	svc, repo, f := newTietuForTest(t, []LibPoem{tietuTaohua}, wiki)

	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "桃花源记 陶渊明", Panels: 3}})
	require.NoError(t, err)
	_, err = svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "别的"}})
	require.ErrorIs(t, err, ErrTietuBusy, "one run at a time")
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuPlanned, got.Status, got.Error)
	require.True(t, got.Poem.Agreed, "library and 维基文库 agree once the note is dropped")
	require.NotContains(t, got.Poem.Text, "一作", "notes inside the library text are dropped")
	require.Len(t, got.Poem.Sources, 2)
	require.Equal(t, "series", got.Plan.Kind)
	require.Len(t, got.Plan.Panels, 3)

	_, err = svc.Plan(ctx, 5, p.ID, TietuPlanInput{Confirm: true})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuCovered, got.Status, got.Error)
	require.Len(t, got.Covers, tietuCovers)
	require.Equal(t, tietuCovers, f.draws)
	require.NotNil(t, got.Covers[0].Check, "covers are checked too")

	_, err = svc.Cover(ctx, 5, p.ID, TietuCoverInput{Pick: 9})
	require.Equal(t, "TIETU_COVER", infraerrors.Reason(err))
	_, err = svc.Cover(ctx, 5, p.ID, TietuCoverInput{Pick: 2})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuDone, got.Status, got.Error)
	require.Equal(t, 3, f.edits)
	require.Equal(t, 3, f.refs, "every panel is drawn with the chosen cover as the reference")
	require.False(t, got.Panels[1].Check.OK)
	require.Contains(t, got.Post.Body, "【二 忽逢桃花林】两岸都是桃花（找一找：水面的花瓣）", "a 找一找 the check couldn't see is replaced by one it saw")
	require.Contains(t, got.Post.Body, "【一 缘溪行】渔人沿溪而上（找一找：船头的鸬鹚）")
	require.Contains(t, got.Post.Body, "《桃花源记》〔魏晋〕陶渊明")
	require.True(t, strings.HasSuffix(got.Post.Body, "画面由 AI 生成"))
	require.Equal(t, "桃花源记｜课本里的桃花源，我画成了连环画", got.Post.Title, "the title names the poem")

	path, err := svc.ImageFile(ctx, 5, p.ID, "p1")
	require.NoError(t, err)
	require.FileExists(t, path)
	_, err = svc.ImageFile(ctx, 5, p.ID, "../x")
	require.ErrorIs(t, err, ErrTietuNotFound)
	_, err = svc.ImageFile(ctx, 6, p.ID, "p1")
	require.ErrorIs(t, err, ErrTietuNotFound, "another user's pictures stay hidden")

	_, err = svc.Redraw(ctx, 5, p.ID, TietuRedrawInput{Name: "p2", Note: "不要出现村落"})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuDone, got.Status)
	require.Equal(t, 4, f.edits, "only the one panel is redrawn")
	require.Equal(t, "不要出现村落", got.Panels[1].Note)

	_, err = svc.SavePost(ctx, 5, p.ID, TietuPostInput{Title: "新标题", Body: "正文"})
	require.NoError(t, err)
	_, err = svc.Redraw(ctx, 5, p.ID, TietuRedrawInput{Name: "p3"})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, "新标题", got.Post.Title, "a redraw keeps the user's edited post")
	require.NoError(t, svc.Pushed(ctx, 5, p.ID))
	require.NoError(t, svc.Delete(ctx, 5, p.ID))
	require.Empty(t, repo.rows)
}

func TestTietuThemeSuggestsOnlyLibraryPoems(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTietuForTest(t, []LibPoem{tietuTaohua}, nil)
	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "隐居", Theme: true}})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuSuggested, got.Status, got.Error)
	require.Len(t, got.Suggestions, 1, "a poem the library lacks is dropped")
	require.EqualValues(t, 342532, got.Suggestions[0].LibID)

	_, err = svc.Pick(ctx, 5, p.ID, 0)
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuPlanned, got.Status, got.Error)
	require.Contains(t, got.Poem.Note, "维基文库没找到")
}

func TestTietuVariantsUseTheBestKnownReading(t *testing.T) {
	ctx := context.Background()
	lib := LibPoem{ID: 1, Title: "清平乐·茅檐低小", Author: "辛弃疾", Dynasty: "宋", Content: "茅檐低小，溪上青青草。醉里蛮音相媚好，白发谁家翁媪。"}
	wiki := map[string]string{"清平樂 (辛棄疾)": "茅檐低小\n溪上青青草\n醉里吴音相媚好\n白发谁家翁媪"}
	svc, _, f := newTietuForTest(t, []LibPoem{lib}, wiki)
	f.variantTo = "1"
	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "清平乐 辛弃疾"}})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuPlanned, got.Status, got.Error)
	require.Equal(t, "茅檐低小，溪上青青草。醉里吴音相媚好，白发谁家翁媪。", got.Poem.Text, "the chosen character replaces the library's, punctuation kept")
	require.Len(t, got.Poem.Variants, 1)
	require.Equal(t, []string{"蛮", "吴"}, got.Poem.Variants[0].Options)
	require.Equal(t, "吴", got.Poem.Variants[0].Chosen)
	require.True(t, tietuDecided(got.Poem.Variants, "吴音相媚"), "the common-reading pass leaves settled places alone")
	require.False(t, tietuDecided(got.Poem.Variants, "白发谁家"))

	// An invalid pick keeps the library text.
	svc2, _, f2 := newTietuForTest(t, []LibPoem{lib}, wiki)
	f2.variantTo = "7"
	p2, _ := svc2.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "清平乐 辛弃疾"}})
	waitTietu(t, svc2, p2.ID)
	got2, _ := svc2.Get(ctx, 5, p2.ID)
	require.Contains(t, got2.Poem.Text, "蛮音")
}

func TestTietuFindsAWorkUnderAnotherTitle(t *testing.T) {
	ctx := context.Background()
	poems := []LibPoem{
		{ID: 385123, Title: "木兰辞", Author: "杨维桢", Dynasty: "明", Content: "金韹韹，鼓声声，行人且勿行。"},
		{ID: 392142, Title: "木兰辞", Author: "无名氏", Dynasty: "南北朝", Content: "唧唧复唧唧，木兰当户织。不闻机杼声，唯闻女叹息。"},
		{ID: 72926, Title: "感兴五首其五", Author: "欧阳修", Dynasty: "宋", Content: "唧唧复唧唧，夜叹晓未息。"},
	}
	svc, _, _ := newTietuForTest(t, poems, nil)
	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "木兰诗"}})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuPlanned, got.Status, got.Error)
	require.EqualValues(t, 392142, got.Poem.LibID, "the work whose text starts with the first line, not another 木兰辞")
	require.Equal(t, "木兰诗", got.Poem.Title, "the title most readers know")
	require.Equal(t, "北朝民歌", got.Poem.Author, "an anonymous work gets the usual attribution")
	require.Equal(t, "唧唧复唧唧，木兰当户织。不闻机杼声，唯闻女叹息。", got.Poem.Text, "the text still comes from the library")
}

func TestTietuLengthDifferencesAndCommonReadings(t *testing.T) {
	ctx := context.Background()
	lib := LibPoem{ID: 2, Title: "木兰诗", Author: "北朝民歌", Dynasty: "北朝", Content: "雄兔脚扑朔，雌兔眼迷离。\n两兔傍地走，安能辨我是雄雌。"}
	wiki := map[string]string{"木蘭辭": "雄兔脚扑朔雌兔眼迷离两兔傍地走安能辨我是雄雌"}
	svc, _, _ := newTietuForTest(t, []LibPoem{lib}, wiki)
	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "木兰诗 北朝民歌"}})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuPlanned, got.Status, got.Error)
	require.Equal(t, "雄兔脚扑朔，雌兔眼迷离。\n双兔傍地走，安能辨我是雄雌。", got.Poem.Text, "only the same-length change of a phrase found once is applied")
	require.Len(t, got.Poem.Variants, 1)
	require.Contains(t, got.Poem.Variants[0].Why, "通行本写法")
}

func TestTietuPoemNotFound(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newTietuForTest(t, nil, nil)
	p, err := svc.Create(ctx, 5, TietuCreateInput{KeyID: 7, TietuBrief: TietuBrief{Input: "没有这首"}})
	require.NoError(t, err)
	waitTietu(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, TietuFailed, got.Status)
	require.Contains(t, got.Error, "没找到这首诗")
}

func TestCleanTietuPlan(t *testing.T) {
	var raw TietuPlan
	require.NoError(t, json.Unmarshal([]byte(tietuTestPlan), &raw))
	pl, err := cleanTietuPlan(raw, TietuBrief{Mode: "single", Panels: 2})
	require.NoError(t, err)
	require.Equal(t, "single", pl.Kind, "a chosen mode wins over the model")
	require.Len(t, pl.Panels, 2)
	raw.Panels = raw.Panels[:1]
	_, err = cleanTietuPlan(raw, TietuBrief{Mode: "auto", Panels: 6})
	require.ErrorIs(t, err, ErrTietuPlan)
}

func TestTietuTextHelpers(t *testing.T) {
	require.Equal(t, "江城子·老夫聊发少年狂", tietuShortTitle("江神子・江城子·老夫聊发少年狂"))
	require.Equal(t, "遂与外人间隔。", tietuDropNotes("遂与外人间隔。(间隔一作：隔绝)"))
	runs, ok := tietuAlign([]rune("木兰当户织不闻机杼声唯闻女叹息"), []rune("木兰当戸织不闻机杼声但闻女女叹息"))
	require.True(t, ok)
	require.Equal(t, []tietuRun{{3, 4, 3, 4}, {10, 11, 10, 11}, {13, 13, 13, 14}}, runs, "replacements and an insertion")
	text := tietuApply("木兰当户织，不闻机杼声。", []rune("木兰当戸织不闻机杼鸣"), map[int]tietuRun{3: {3, 4, 3, 4}, 9: {9, 10, 9, 10}})
	require.Equal(t, "木兰当戸织，不闻机杼鸣。", text, "punctuation stays where it was")
	b := TietuBrief{Input: "  x ", Panels: 99, Style: "nope"}
	require.NoError(t, normalizeTietuBrief(&b))
	require.Equal(t, tietuMaxPanels, b.Panels)
	require.Equal(t, tietuDefaultStyle, b.Style)
	require.Equal(t, "auto", b.Mode)
}

func TestEditorWechatNewspic(t *testing.T) {
	ctx := context.Background()
	var got map[string]any
	svc := newEditorWithWechat(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":7200}`)
		case "/cgi-bin/draft/add":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &got)
			_, _ = io.WriteString(w, `{"media_id":"draft-9"}`)
		}
	})
	_, err := svc.WechatNewspic(ctx, EditorNewspicInput{EditorCredentials: editorCreds, Title: "t"})
	require.Equal(t, "EDITOR_DRAFT", infraerrors.Reason(err), "pictures are required")
	_, err = svc.WechatNewspic(ctx, EditorNewspicInput{EditorCredentials: editorCreds, Title: strings.Repeat("长", 33), ImageMediaIDs: []string{"m1"}})
	require.Equal(t, "EDITOR_DRAFT", infraerrors.Reason(err), "titles are at most 32 characters")

	id, err := svc.WechatNewspic(ctx, EditorNewspicInput{EditorCredentials: editorCreds, Title: " 桃花源记 ", Content: "正文 & 原文",
		ImageMediaIDs: []string{"m1", " ", "m2"}, NeedOpenComment: true})
	require.NoError(t, err)
	require.Equal(t, "draft-9", id)
	a := got["articles"].([]any)[0].(map[string]any)
	require.Equal(t, "newspic", a["article_type"])
	require.Equal(t, "桃花源记", a["title"])
	require.Equal(t, "正文 & 原文", a["content"])
	list := a["image_info"].(map[string]any)["image_list"].([]any)
	require.Len(t, list, 2)
	require.Equal(t, "m1", list[0].(map[string]any)["image_media_id"])
	require.EqualValues(t, 1, a["need_open_comment"])
}
