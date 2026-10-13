//go:build unit

package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const deskBody = "# 我用 Codex 改了一周代码\n\n上周三我把公司的报表脚本交给 Codex。\n\n它第一次就跑通了，我愣了一下。\n\n【这里需要你的真实经历：第二天出了什么问题】\n\n最后我还是自己改了两行。"

// deskFakeAgent answers each 写作台 step by what its system prompt asks for, and records what it was
// asked.
type deskFakeAgent struct {
	mu    sync.Mutex
	steps []string
	asked map[string]string // step → last user message
}

func (f *deskFakeAgent) run(t *testing.T) articleAgentFunc {
	return func(_ context.Context, in AgentRunInput) (*AgentRunResult, error) {
		system, user := *in.Messages[0].Content, *in.Messages[len(in.Messages)-1].Content
		res := &AgentRunResult{Model: "gpt-test", PromptTokens: 10, CompletionTokens: 5}
		step := ""
		switch {
		case strings.Contains(system, "把这个人「是谁"):
			step = "distill"
			res.Text = `{"name":"老苗","identity":"我在做一个 API 网关","experiences":["上线第一周被刷了 3.9 万次撞库（《复盘》）"],"stances":["工具要先自己用"],"phrases":["说真的"],"taboos":["家人们"],"voice":"你写的时候短句多"}`
		case strings.Contains(system, "研究助理"):
			step = "research"
			res.Text = `{"facts":[{"text":"作者上周三把报表脚本交给 Codex","kind":"user","source":"作者"},{"text":"随便编的网页事实","kind":"web","source":"https://not-searched.example"}],
"questions":[{"question":"第二天出了什么问题？","why":"补亲历"},{"question":"你最后改了几行？","why":"补数字"},{"question":"q3","why":""},{"question":"q4 多出来的","why":""}]}`
		case strings.Contains(system, "从哪个角度写"):
			step = "angles"
			res.Text = `{"angles":[{"title":"我把报表脚本交给了 Codex","angle":"亲历一周","archetype":"亲历实验型","reader":"程序员","hook":"上周三……","h":5,"k":4,"r":9,"why":"真实"},{"title":"t2","angle":"a2","archetype":"观点评论型","reader":"r","hook":"h","h":2,"k":3,"r":2,"why":"w"}]}`
		case strings.Contains(system, "你在写一篇公众号长文"):
			step = "draft"
			switch {
			case strings.Contains(system, "真实做这件事的过程"):
				res.Text = "# 叙事稿\n\n上周三我把报表脚本交给 Codex。\n\n它跑通了。"
			case strings.Contains(system, "最关心的那个问题"):
				res.Text = "# 问答稿\n\nCodex 能不能改老代码？能，但要盯着。\n\n这是问答稿的亮点句。"
			default:
				res.Text = "# 观点稿\n\n我觉得 Codex 值得用。"
			}
		case strings.Contains(system, "刷到这篇文章的读者"):
			step = "reader"
			res.Text = `{"paras":[{"i":1,"v":"read"},{"i":2,"v":"leave","why":"太短"},{"i":9,"v":"leave"}],"hook":4,"flow":3,"voice":4,"substance":2,"best_line":"它跑通了。","problems":["太短"],"summary":"还行"}`
		case strings.Contains(system, "挑一份做底稿"):
			step = "judge"
			// The prompt numbers the drafts 稿1… in this round's order; name the one holding the
			// 问答 text as best and quote its highlight.
			best := ""
			for i, part := range strings.Split(user, "===== 稿")[1:] {
				if strings.Contains(part, "问答稿") {
					best = "稿" + string(rune('1'+i))
				}
			}
			res.Text = `{"ranking":["` + best + `"],"reason":"问答稿开头最好","highlights":[{"draft":"` + best + `","quote":"这是问答稿的亮点句。","why":"好"}]}`
		case strings.Contains(system, "以「底稿」为基础"):
			step = "merge"
			res.Text = deskBody
		case strings.Contains(system, "只改「选中的文字」"):
			step = "assist"
			res.Text = `{"text":"它居然一次就跑通了。","note":"更口语"}`
		case strings.Contains(system, "最后一遍审读"):
			step = "check"
			res.Text = `{"issues":[{"para":3,"layer":"voice","level":"should","quote":"我愣了一下","problem":"可以更具体","fix":"我盯着屏幕愣了三秒"},{"quote":"正文里没有的话","layer":"fact","problem":"x"}],"scores":{"voice":4},"summary":"不错"}`
		case strings.Contains(system, "从作者的改动里总结"):
			step = "learn"
			res.Text = `{"rules":[{"rule":"开头直接从自己做的事说起","evidence":"原来……改成……"}]}`
		case strings.Contains(system, "改写成下面这个平台的版本"):
			step = "adapt"
			res.Text = `{"kind":"answer","title":"Codex 能改老代码吗","question":"Codex 能改老代码吗？","body":"` + strings.Repeat("知乎回答正文。", 30) + `{{link}}","tags":["Codex"]}`
		default:
			t.Fatalf("unexpected prompt: %s", system[:60])
		}
		f.mu.Lock()
		f.steps = append(f.steps, step)
		if f.asked == nil {
			f.asked = map[string]string{}
		}
		f.asked[step] = system + "\n=====\n" + user
		f.mu.Unlock()
		return res, nil
	}
}

func (f *deskFakeAgent) count(step string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.steps {
		if s == step {
			n++
		}
	}
	return n
}

func (f *deskFakeAgent) last(step string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.asked[step]
}

func waitDesk(t *testing.T, svc *ContentEngineService, id int64) *ContentItem {
	t.Helper()
	require.Eventually(t, func() bool { return !svc.running(id) }, 5*time.Second, 5*time.Millisecond)
	it, err := svc.item(context.Background(), contentTestOwner, id)
	require.NoError(t, err)
	require.NotNil(t, it.Desk)
	if it.Desk.Job != nil {
		require.NotEqual(t, ContentJobFailed, it.Desk.Job.Status, it.Desk.Job.Error)
	}
	return it
}

func TestContentDeskFlow(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _ := newContentForTest(t)
	fake := &deskFakeAgent{}
	svc.agent = fake.run(t)
	saveContentSettings(t, svc, func(s *ContentSettings) {
		s.Platforms, s.Search = []string{ContentPlatformWechat, ContentPlatformZhihu}, false
	})

	// 作者档案: samples, then a distilled proposal the owner saves.
	_, err := svc.AddPersonaSample(ctx, contentTestOwner, "", "", "太短")
	require.ErrorIs(t, err, ErrContentSample)
	_, err = svc.DistillPersona(ctx, contentTestOwner)
	require.ErrorIs(t, err, ErrContentNoSamples)
	for i := 0; i < 2; i++ {
		_, err = svc.AddPersonaSample(ctx, contentTestOwner, "", "复盘", strings.Repeat("我自己的文章，说真的。", 40))
		require.NoError(t, err)
	}
	proposal, err := svc.DistillPersona(ctx, contentTestOwner)
	require.NoError(t, err)
	require.Equal(t, "老苗", proposal.Name)
	saved, err := svc.SavePersona(ctx, contentTestOwner, *proposal)
	require.NoError(t, err)
	require.Len(t, saved.Samples, 2, "samples survive a save")

	// 材料关: the owner's notes and persona become cards; a web card without a searched page is
	// dropped; at most three open questions.
	it, err := svc.CreateDesk(ctx, contentTestOwner, ContentDeskInput{Title: "Codex 改老代码", Notes: "上周三把报表脚本交给 Codex"})
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Equal(t, ContentDeskStatus, it.Status)
	require.Equal(t, ContentStageMaterials, it.Desk.Stage)
	require.Len(t, it.Desk.Facts, 1)
	require.Equal(t, ContentFactOK, it.Desk.Facts[0].Status)
	require.Len(t, it.Desk.Questions, 3)
	require.Contains(t, fake.last("research"), "上周三把报表脚本交给 Codex")

	// The persona goes into every prompt.
	_, err = svc.SaveMaterials(ctx, contentTestOwner, it.ID, ContentMaterialsInput{
		Facts:   append(it.Desk.Facts, ContentFact{Text: "我改了两行", Status: ContentFactOK}),
		Answers: map[string]string{it.Desk.Questions[0].ID: "第二天它把日期格式改错了"},
	})
	require.NoError(t, err)
	it, _ = svc.item(ctx, contentTestOwner, it.ID)
	require.Len(t, it.Desk.Facts, 3)
	require.Equal(t, ContentFactAnswer, it.Desk.Facts[2].Kind)
	require.Equal(t, "第二天它把日期格式改错了", it.Desk.Questions[0].Answer)

	_, err = svc.Angles(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Equal(t, ContentStageAngles, it.Desk.Stage)
	require.Len(t, it.Desk.Angles, 2)
	require.Equal(t, 5, it.Desk.Angles[0].R, "scores clamped to 1–5")
	require.Contains(t, fake.last("angles"), "第二天它把日期格式改错了", "answers are cards")

	// Drafts need an angle.
	_, err = svc.Drafts(ctx, contentTestOwner, it.ID)
	require.ErrorIs(t, err, ErrContentDeskStage)
	pick := it.Desk.Angles[0]
	pick.Note = "别写成教程"
	it, err = svc.ChooseAngle(ctx, contentTestOwner, it.ID, pick)
	require.NoError(t, err)
	require.Equal(t, ContentStageDrafts, it.Desk.Stage)

	_, err = svc.Drafts(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Len(t, it.Desk.Drafts, 3)
	require.Equal(t, 3, fake.count("draft"))
	require.Equal(t, 3, fake.count("reader"))
	require.Equal(t, contentJudgeRounds, fake.count("judge"))
	require.Contains(t, fake.last("draft"), "别写成教程")
	require.Contains(t, fake.last("draft"), "老苗", "persona in the prompt")
	byID := map[string]ContentDraft{}
	for _, d := range it.Desk.Drafts {
		byID[d.ID] = d
		require.NotNil(t, d.Review)
		require.Len(t, d.Review.Paras, 2, "out-of-range paragraph marks dropped")
	}
	require.NotNil(t, it.Desk.Judge)
	best := byID[it.Desk.Judge.Ranking[0]]
	require.Equal(t, "问答", best.Label, "the judge's pick survives both orders")
	require.Empty(t, it.Desk.Judge.Highlights, "the winner's own highlights are not offered for merging")

	// Base + no highlights: straight to editing, kept as a version.
	story := it.Desk.Drafts[0]
	it, err = svc.ChooseBase(ctx, contentTestOwner, it.ID, story.ID, nil)
	require.NoError(t, err)
	require.Equal(t, ContentStageEdit, it.Desk.Stage)
	require.Equal(t, story.Markdown, it.Desk.Body)
	// Merge with a highlight (fake judge data): background, then the merged text.
	_, err = svc.updateDesk(ctx, contentTestOwner, it.ID, func(it *ContentItem) error {
		it.Desk.Judge.Highlights = []ContentHighlight{{ID: "H9", DraftID: best.ID, Quote: "这是问答稿的亮点句。"}}
		return nil
	})
	require.NoError(t, err)
	_, err = svc.ChooseBase(ctx, contentTestOwner, it.ID, story.ID, []string{"H9"})
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Equal(t, deskBody, it.Desk.Body)
	require.Equal(t, deskBody, it.Desk.BodyAI)
	require.Contains(t, fake.last("merge"), "这是问答稿的亮点句。")

	// Editing: assist, a manual save kept as a version, the check.
	res, err := svc.Assist(ctx, contentTestOwner, it.ID, ContentAssistInput{Action: "voice", Selection: "它第一次就跑通了"})
	require.NoError(t, err)
	require.Equal(t, "它居然一次就跑通了。", res.Text)
	_, err = svc.Assist(ctx, contentTestOwner, it.ID, ContentAssistInput{Action: "nope", Selection: "x"})
	require.ErrorIs(t, err, ErrContentAssist)

	edited := strings.Replace(deskBody, "上周三我把公司的报表脚本交给 Codex。", "上周三，我把公司的报表脚本交给 Codex——说真的，我没抱希望。", 1)
	_, err = svc.SaveBody(ctx, contentTestOwner, it.ID, edited, "手动改了开头")
	require.NoError(t, err)
	_, err = svc.Check(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.NotNil(t, it.Desk.Check)
	layers := map[string]int{}
	for _, is := range it.Desk.Check.Issues {
		layers[is.Layer]++
		require.Contains(t, edited, is.Quote, "every issue quotes the text")
	}
	require.Positive(t, layers["rule"], "the dash and the placeholder are rule hits")
	require.Equal(t, 1, layers["voice"], "the issue quoting text that isn't there is dropped")
	require.Equal(t, 1, it.Desk.Check.Stats.Placeholders)

	// Final needs the placeholder filled; then the edits become proposed rules.
	_, err = svc.Finalize(ctx, contentTestOwner, it.ID)
	require.ErrorIs(t, err, ErrContentPlaceholder)
	final := strings.Replace(edited, "【这里需要你的真实经历：第二天出了什么问题】", "第二天它把日期格式改错了。", 1)
	_, err = svc.SaveBody(ctx, contentTestOwner, it.ID, final, "")
	require.NoError(t, err)
	_, err = svc.Finalize(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Equal(t, ContentStageFinal, it.Desk.Stage)
	require.Contains(t, fake.last("learn"), "【作者改成】")
	require.Len(t, it.Desk.Learn, 1)
	it, err = svc.ApplyLearn(ctx, contentTestOwner, it.ID, it.Desk.Learn[0].ID, true)
	require.NoError(t, err)
	require.Equal(t, "added", it.Desk.Learn[0].Status)
	rules, _ := svc.Rules(ctx, contentTestOwner)
	require.Equal(t, "learned", rules[len(rules)-1].Origin)

	// Versions: chosen draft, merge, manual, final.
	vs, err := svc.Versions(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	kinds := []string{}
	for _, v := range vs {
		kinds = append(kinds, v.Kind)
	}
	require.Equal(t, []string{"final", "manual", "merge", "draft"}, kinds)

	// Platform versions: the 公众号 one is the final text, the others are rewritten.
	_, err = svc.Platforms(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	it = waitDesk(t, svc, it.ID)
	require.Equal(t, ContentPending, it.Status)
	detail, err := svc.Item(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	require.Len(t, detail.Versions, 2)
	wechat := detail.Versions[0]
	require.Equal(t, ContentPlatformWechat, wechat.Platform)
	require.Equal(t, "我用 Codex 改了一周代码", wechat.Title)
	require.NotContains(t, wechat.Body, "# 我用 Codex", "the H1 becomes the title")
	require.Contains(t, wechat.Body, "第二天它把日期格式改错了")
	require.Equal(t, 1, fake.count("adapt"), "only 知乎 went through the model")
	require.Contains(t, fake.last("adapt"), "第二天它把日期格式改错了", "versions come from the final text")
	_ = repo
}

func TestContentDeskJobKeepsConcurrentEdits(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _ := newContentForTest(t)
	release := make(chan struct{})
	fake := &deskFakeAgent{}
	inner := fake.run(t)
	svc.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		if strings.Contains(*in.Messages[0].Content, "最后一遍审读") {
			<-release
		}
		return inner(ctx, in)
	}
	saveContentSettings(t, svc, func(s *ContentSettings) { s.Search = false })
	it, err := svc.CreateDesk(ctx, contentTestOwner, ContentDeskInput{Title: "t", Notes: "n"})
	require.NoError(t, err)
	waitDesk(t, svc, it.ID)
	_, err = svc.updateDesk(ctx, contentTestOwner, it.ID, func(it *ContentItem) error {
		it.Desk.Stage, it.Desk.Body = ContentStageEdit, deskBody
		return nil
	})
	require.NoError(t, err)

	_, err = svc.Check(ctx, contentTestOwner, it.ID)
	require.NoError(t, err)
	// A second step can't start while one runs.
	_, err = svc.Check(ctx, contentTestOwner, it.ID)
	require.ErrorIs(t, err, ErrContentDeskStage)
	// The owner keeps typing while the check runs.
	_, err = svc.SaveBody(ctx, contentTestOwner, it.ID, deskBody+"\n\n新加的一段。", "")
	require.NoError(t, err)
	close(release)
	got := waitDesk(t, svc, it.ID)
	require.Contains(t, got.Desk.Body, "新加的一段。", "the check's result didn't overwrite the edit")
	require.NotNil(t, got.Desk.Check)
	require.Equal(t, ContentJobDone, got.Desk.Job.Status)
}

func TestCheckProse(t *testing.T) {
	md := "# 标题\n\n## 小节\n\n这不是工具的问题，而是习惯的问题。\n\n我们进行了全面优化——效果很好。\n\n核心是：\n\n他说：好的。\n\n```go\n// 赋能 in code is fine\n```\n\n家人们，冲。\n\n【这里需要你的真实经历：截图】"
	stats, hits := checkProse(md, []string{"家人们"})
	problems := []string{}
	for _, h := range hits {
		problems = append(problems, h.Problem)
		require.Contains(t, md, h.Quote)
	}
	joined := strings.Join(problems, "|")
	require.Contains(t, joined, "翻案腔")
	require.Contains(t, joined, "动词名词化")
	require.Contains(t, joined, "破折号")
	require.Contains(t, joined, "提示性冒号")
	require.Contains(t, joined, "家人们")
	require.Contains(t, joined, "真实经历")
	require.NotContains(t, joined, "赋能", "code blocks are skipped")
	require.Equal(t, 1, stats.Colons, "a colon introducing someone's words is fine")
	require.Equal(t, 1, stats.Headings)
	require.Equal(t, 1, stats.Placeholders)
}

func TestContentPersonaBlockAndParaDiff(t *testing.T) {
	require.Empty(t, contentPersonaBlock(&ContentPersona{}))
	p := &ContentPersona{Name: "老苗", Identity: "做 API 网关的", Experiences: []string{"被撞库 3.9 万次"}, Phrases: []string{"说真的"},
		Samples: []ContentSample{{Title: "复盘", Text: strings.Repeat("原文。", 1000)}}}
	block := contentPersonaBlock(p)
	require.Contains(t, block, "你就是「老苗」本人")
	require.Contains(t, block, "被撞库 3.9 万次")
	require.Less(t, len([]rune(block)), 1600, "samples are excerpted")

	diff := contentParaDiff("第一段没动。\n\n第二段原来这样写的很长很长。\n\n第三段删掉。", "第一段没动。\n\n第二段我改成这样写的很长很长。", 10)
	require.NotContains(t, diff, "第一段")
	require.Contains(t, diff, "【作者改成】第二段我改成")
	require.Contains(t, diff, "（删掉了）")
	require.Equal(t, []string{"一", "```\ncode\n\nmore\n```", "二"}, contentParas("一\n\n```\ncode\n\nmore\n```\n\n二"))
}
