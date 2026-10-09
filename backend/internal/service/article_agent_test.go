//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"github.com/stretchr/testify/require"
)

type articleRepoStub struct {
	mu   sync.Mutex
	next int64
	rows map[int64]ArticleProject
}

func (r *articleRepoStub) clone(p ArticleProject) ArticleProject {
	b, _ := json.Marshal(p.ArticleData)
	var d ArticleData
	_ = json.Unmarshal(b, &d)
	p.ArticleData = d
	return p
}

func (r *articleRepoStub) Create(_ context.Context, p *ArticleProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	p.ID, p.CreatedAt, p.UpdatedAt = r.next, time.Now(), time.Now()
	r.rows[p.ID] = r.clone(*p)
	return nil
}

func (r *articleRepoStub) Get(_ context.Context, userID, id int64) (*ArticleProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.rows[id]
	if !ok || p.UserID != userID {
		return nil, nil
	}
	c := r.clone(p)
	return &c, nil
}

func (r *articleRepoStub) List(_ context.Context, userID int64, limit int) ([]ArticleProject, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []ArticleProject{}
	for _, p := range r.rows {
		if p.UserID == userID {
			out = append(out, r.clone(p))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (r *articleRepoStub) Save(_ context.Context, p *ArticleProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[p.ID] = r.clone(*p)
	return nil
}

func (r *articleRepoStub) CountActive(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.rows {
		if p.UserID == userID && articleActive(p.Status) {
			n++
		}
	}
	return n, nil
}

func (r *articleRepoStub) FailActive(context.Context, string) (int64, error) { return 0, nil }

func (r *articleRepoStub) Delete(_ context.Context, userID, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, id)
	return nil
}

func (r *articleRepoStub) DeleteBefore(context.Context, time.Time) ([]int64, error) { return nil, nil }

type articleSearchStub struct{ queries []string }

func (s *articleSearchStub) Available(context.Context) bool { return true }
func (s *articleSearchStub) Search(_ context.Context, q string, _ int) ([]websearch.SearchResult, error) {
	s.queries = append(s.queries, q)
	return []websearch.SearchResult{{Title: "国庆出游数据", URL: "https://www.mct.gov.cn/a", Snippet: "国内出游 8 亿人次"}}, nil
}

func articlePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := 0; x < 64; x++ {
		img.Set(x, 10, color.RGBA{R: 200, A: 255})
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, img))
	return b.Bytes()
}

const articleTestOutline = `{"titles":["国庆出游，3 个省钱办法","标题二","标题三"],"title":"国庆出游，3 个省钱办法","digest":"错峰、比价、选对交通",
"cover_prompt":"秋天的山路","sections":[{"heading":"为什么贵","points":["需求集中"],"image":{"prompt":"拥挤的景区","alt":"景区"}},
{"heading":"错峰","points":["提前两天走"]},{"heading":"比价","points":["多平台比"],"image":{"prompt":"手机比价","alt":"比价"}},{"heading":"多余","points":["x"],"image":{"prompt":"多余的图","alt":"x"}}]}`

func newArticleForTest(t *testing.T) (*ArticleAgentService, *articleRepoStub, *articleSearchStub) {
	t.Helper()
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 5, Name: "写作", Key: "sk-user5", Status: StatusActive},
		{ID: 8, UserID: 6, Name: "别人的", Key: "sk-user6", Status: StatusActive},
	}}
	learn, _ := newL2Service(t, "http://127.0.0.1:1", LearnSources{Keys: keys})
	repo := &articleRepoStub{rows: map[int64]ArticleProject{}}
	search := &articleSearchStub{}
	svc := NewArticleAgentService(repo, learn, search, t.TempDir())
	return svc, repo, search
}

// waitArticle waits until the background run of an article stops.
func waitArticle(t *testing.T, svc *ArticleAgentService, id int64) {
	t.Helper()
	require.Eventually(t, func() bool { return !svc.running(id) }, 5*time.Second, 5*time.Millisecond)
}

func TestArticleAgentFlow(t *testing.T) {
	ctx := context.Background()
	svc, repo, search := newArticleForTest(t)

	var keysSeen []string
	var outlineCalls int
	var lastOutlineRequest string
	svc.agent = func(ctx context.Context, in AgentRunInput) (*AgentRunResult, error) {
		keysSeen = append(keysSeen, in.Key)
		outlineCalls++
		lastOutlineRequest = *in.Messages[len(in.Messages)-1].Content
		res := &AgentRunResult{Model: "gpt-test", PromptTokens: 100, CompletionTokens: 50}
		if len(in.Tools) > 0 {
			_, step, err := runAgentTool(ctx, map[string]AgentTool{"web_search": in.Tools[0]},
				AgentToolCall{ID: "c1", Function: AgentFunctionCall{Name: "web_search", Arguments: `{"query":"2026 国庆 出游 人次"}`}}, nil)
			require.NoError(t, err)
			res.Steps = append(res.Steps, step)
		}
		if outlineCalls == 1 {
			res.Text = "好的，下面是大纲：\n```json\n" + articleTestOutline + "\n```"
		} else {
			res.Text = strings.Replace(articleTestOutline, "为什么贵", "为什么这么贵", 1)
		}
		return res, nil
	}
	var writeRequest string
	svc.chat = func(_ context.Context, key, _ string, msgs []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		keysSeen = append(keysSeen, key)
		writeRequest = msgs[1].Content
		body := "国庆出门，钱花在哪了？" + strings.Repeat("正文内容。", 60) + "\n\n## 为什么这么贵\n\n![景区](img:1)\n\n## 比价\n\n![乱写的](img:9)\n\n结尾。"
		_ = onDelta(body)
		return &LearnTutorResult{Model: "gpt-test", PromptTokens: 300, CompletionTokens: 900}, nil
	}
	png := articlePNG(t)
	var drawn []string
	var drawMu sync.Mutex
	svc.draw = func(_ context.Context, key, prompt, size string) ([]byte, error) {
		drawMu.Lock()
		defer drawMu.Unlock()
		keysSeen = append(keysSeen, key)
		drawn = append(drawn, prompt)
		require.Equal(t, articleImageSize, size)
		return png, nil
	}

	_, err := svc.Create(ctx, 5, ArticleCreateInput{KeyID: 8, ArticleBrief: ArticleBrief{Topic: "国庆出游"}})
	require.ErrorIs(t, err, ErrLearnKeyInvalid, "only the user's own key")
	_, err = svc.Create(ctx, 5, ArticleCreateInput{KeyID: 7, ArticleBrief: ArticleBrief{Topic: " "}})
	require.ErrorIs(t, err, ErrArticleTopic)

	p, err := svc.Create(ctx, 5, ArticleCreateInput{KeyID: 7, ArticleBrief: ArticleBrief{Topic: "国庆出游怎么省钱", Images: 2, Search: true, Length: "weird"}})
	require.NoError(t, err)
	require.Equal(t, ArticleOutlining, p.Status)
	require.Equal(t, "standard", p.Brief.Length)
	waitArticle(t, svc, p.ID)

	got, err := svc.Get(ctx, 5, p.ID)
	require.NoError(t, err)
	require.Equal(t, ArticleOutlined, got.Status, got.Error)
	require.Equal(t, "国庆出游，3 个省钱办法", got.Title)
	require.Len(t, got.Outline.Sections, 4)
	require.NotNil(t, got.Outline.Sections[2].Image)
	require.Nil(t, got.Outline.Sections[3].Image, "only as many pictures as asked for")
	require.Equal(t, []string{"2026 国庆 出游 人次"}, search.queries)
	require.Equal(t, []ArticleSource{{Title: "国庆出游数据", URL: "https://www.mct.gov.cn/a", Snippet: "国内出游 8 亿人次"}}, got.Sources)
	require.Equal(t, 150, got.PromptTokens+got.CompletionTokens)

	_, err = svc.Get(ctx, 6, p.ID)
	require.ErrorIs(t, err, ErrArticleNotFound, "another user's article")
	_, err = svc.Outline(ctx, 6, p.ID, ArticleOutlineInput{Confirm: true})
	require.ErrorIs(t, err, ErrArticleNotFound)

	// Feedback redoes the outline with the previous one in the request.
	_, err = svc.Outline(ctx, 5, p.ID, ArticleOutlineInput{Feedback: "第一节换个说法"})
	require.NoError(t, err)
	waitArticle(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleOutlined, got.Status)
	require.Equal(t, "为什么这么贵", got.Outline.Sections[0].Heading)
	require.Contains(t, lastOutlineRequest, "第一节换个说法")
	require.Contains(t, lastOutlineRequest, "上一版大纲")

	// The user edits the outline (drops the last section) and confirms.
	edited := *got.Outline
	edited.Sections = edited.Sections[:3]
	edited.Title = "我改的标题"
	_, err = svc.Outline(ctx, 5, p.ID, ArticleOutlineInput{Outline: &edited, Confirm: true})
	require.NoError(t, err)
	waitArticle(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleDone, got.Status, got.Error)
	require.Equal(t, "我改的标题", got.Title)
	require.Contains(t, writeRequest, "我改的标题")
	require.Contains(t, writeRequest, "国内出游 8 亿人次", "search results reach the writer")
	require.Contains(t, got.Markdown, "![景区](img:1)")
	require.NotContains(t, got.Markdown, "img:9", "made-up picture numbers are dropped")
	require.Contains(t, got.Markdown, "![比价](img:2)", "a missing picture is added")
	require.Len(t, got.Images, 3)
	require.Equal(t, "cover", got.Images[0].Kind)
	require.Len(t, drawn, 3)
	for _, im := range got.Images {
		require.Equal(t, "ok", im.Status)
		path, err := svc.ImageFile(ctx, 5, p.ID, im.N)
		require.NoError(t, err)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "image/jpeg", httpDetect(data))
	}
	_, err = svc.ImageFile(ctx, 6, p.ID, 0)
	require.ErrorIs(t, err, ErrArticleNotFound)
	for _, k := range keysSeen {
		require.Equal(t, "sk-user5", k, "everything runs on the user's own key")
	}

	require.NoError(t, svc.Pushed(ctx, 5, p.ID))
	got, _ = svc.Get(ctx, 5, p.ID)
	require.NotNil(t, got.PushedAt)

	list, err := svc.List(ctx, 5)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NoError(t, svc.Delete(ctx, 5, p.ID))
	_, err = os.Stat(svc.projectDir(p.ID))
	require.True(t, os.IsNotExist(err))
	require.Empty(t, repo.rows)
}

func httpDetect(b []byte) string {
	if len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return "image/jpeg"
	}
	return "other"
}

func TestArticleAgentFailuresAndRetry(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newArticleForTest(t)
	calls := 0
	svc.agent = func(_ context.Context, in AgentRunInput) (*AgentRunResult, error) {
		calls++
		require.Empty(t, in.Tools, "no search unless asked")
		if calls == 1 {
			return &AgentRunResult{Text: "抱歉我不太确定"}, nil
		}
		return &AgentRunResult{Text: articleTestOutline}, nil
	}
	svc.chat = func(_ context.Context, _, _ string, _ []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		_ = onDelta(strings.Repeat("正文。", 120))
		return &LearnTutorResult{}, nil
	}
	png := articlePNG(t)
	failCover := true
	var mu sync.Mutex
	svc.draw = func(_ context.Context, _, prompt, _ string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		if failCover && strings.Contains(prompt, "封面") {
			return nil, errLearnRunFailed("画图服务繁忙")
		}
		return png, nil
	}

	p, err := svc.Create(ctx, 5, ArticleCreateInput{KeyID: 7, ArticleBrief: ArticleBrief{Topic: "主题", Images: 1}})
	require.NoError(t, err)
	waitArticle(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleOutlined, got.Status, "a reply that isn't JSON is retried once")
	require.Equal(t, 2, calls)

	_, err = svc.Outline(ctx, 5, p.ID, ArticleOutlineInput{Outline: &ArticleOutline{Title: "x"}, Confirm: true})
	require.ErrorIs(t, err, ErrArticleOutline, "an outline without sections")
	_, err = svc.Outline(ctx, 5, p.ID, ArticleOutlineInput{Confirm: true})
	require.NoError(t, err)
	waitArticle(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleDone, got.Status)
	require.Equal(t, "failed", got.Images[0].Status)
	require.Equal(t, "ok", got.Images[1].Status)
	require.Contains(t, got.Events[len(got.Events)-1].Text, "1 张失败")

	// Retry draws only what failed.
	failCover = false
	_, err = svc.Retry(ctx, 5, p.ID)
	require.NoError(t, err)
	waitArticle(t, svc, p.ID)
	got, _ = svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleDone, got.Status)
	require.Equal(t, "ok", got.Images[0].Status)
	require.Equal(t, 2, got.ImagesDrawn)
	_, err = svc.Retry(ctx, 5, p.ID)
	require.ErrorIs(t, err, ErrArticleState, "nothing left to retry")

	// A failed write can be retried; cancel stops a run.
	writing := make(chan struct{}, 1)
	svc.chat = func(ctx context.Context, _, _ string, _ []LearnMessage, _ int, _ func(string) error) (*LearnTutorResult, error) {
		writing <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	p2, err := svc.Create(ctx, 5, ArticleCreateInput{KeyID: 7, ArticleBrief: ArticleBrief{Topic: "第二篇"}})
	require.NoError(t, err)
	waitArticle(t, svc, p2.ID)
	_, err = svc.Outline(ctx, 5, p2.ID, ArticleOutlineInput{Confirm: true})
	require.NoError(t, err)
	_, err = svc.Outline(ctx, 5, p2.ID, ArticleOutlineInput{Confirm: true})
	require.ErrorIs(t, err, ErrArticleState, "already writing")
	<-writing
	require.NoError(t, svc.Cancel(ctx, 5, p2.ID))
	waitArticle(t, svc, p2.ID)
	got, _ = svc.Get(ctx, 5, p2.ID)
	require.Equal(t, ArticleCanceled, got.Status, got.Error)
	require.Empty(t, got.Markdown)

	svc.chat = func(_ context.Context, _, _ string, _ []LearnMessage, _ int, _ func(string) error) (*LearnTutorResult, error) {
		return nil, errors.New("boom")
	}
	_, err = svc.Retry(ctx, 5, p2.ID)
	require.NoError(t, err)
	waitArticle(t, svc, p2.ID)
	got, _ = svc.Get(ctx, 5, p2.ID)
	require.Equal(t, ArticleFailed, got.Status)
	require.Equal(t, "生成出错了，点「重试」再来一次", got.Error, "raw errors are not shown")
}

// A run still waiting for a site-wide slot is canceled, not failed as "site busy".
func TestArticleAgentCancelWhileQueued(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newArticleForTest(t)
	svc.agent = func(context.Context, AgentRunInput) (*AgentRunResult, error) {
		t.Error("a canceled run that never got a slot must not call the model")
		return nil, errors.New("unexpected")
	}
	for i := 0; i < cap(svc.slots); i++ {
		svc.slots <- struct{}{}
	}
	p, err := svc.Create(ctx, 5, ArticleCreateInput{KeyID: 7, ArticleBrief: ArticleBrief{Topic: "排队中"}})
	require.NoError(t, err)
	require.NoError(t, svc.Cancel(ctx, 5, p.ID))
	waitArticle(t, svc, p.ID)
	got, _ := svc.Get(ctx, 5, p.ID)
	require.Equal(t, ArticleCanceled, got.Status, got.Error)
	require.Empty(t, got.Error)
	require.Equal(t, "已停止", got.Events[len(got.Events)-1].Text)
	for i := 0; i < cap(svc.slots); i++ {
		<-svc.slots
	}
}

func TestArticleHelpers(t *testing.T) {
	o, err := parseArticleOutline("```json\n"+articleTestOutline+"\n```", 0)
	require.NoError(t, err)
	for _, s := range o.Sections {
		require.Nil(t, s.Image)
	}
	_, err = parseArticleOutline(`{"title":"","sections":[]}`, 2)
	require.Error(t, err)

	md := articleFixImages("a\n\n![x](img:2)\n\n![x](img:2)\n\nb", &ArticleOutline{Sections: []ArticleSection{
		{Heading: "1", Image: &ArticleOutlineImage{Alt: "一"}}, {Heading: "2", Image: &ArticleOutlineImage{Alt: "二"}}}})
	require.Equal(t, 1, strings.Count(md, "img:2"))
	require.Contains(t, md, "![一](img:1)")

	require.Equal(t, "正文", articleStripFence("```markdown\n正文\n```"))
	require.Equal(t, "mct.gov.cn", articleSiteName("https://www.mct.gov.cn/a/b?c=1"))

	big := image.NewRGBA(image.Rect(0, 0, 1536, 1024))
	for i := range big.Pix {
		big.Pix[i] = byte(i * 7 % 251)
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, big))
	jpg, err := articleJPEG(b.Bytes())
	require.NoError(t, err)
	require.LessOrEqual(t, len(jpg), articleJPEGMaxBytes+200<<10)
	_, err = articleJPEG([]byte("not an image"))
	require.Error(t, err)
}

type articlePricerStub struct{ keys []int64 }

func (p *articlePricerStub) EstimateImageCost(_ context.Context, k *APIKey, model, size string) (float64, bool) {
	p.keys = append(p.keys, k.ID)
	return 0.20104, model == "gpt-image-2" && size == articleImageSize
}

func TestArticleConfig(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newArticleForTest(t)
	cfg := svc.Config(ctx, 5, 7)
	require.True(t, cfg.Search)
	require.Equal(t, 1, cfg.DefaultImages)
	require.Nil(t, cfg.ImagePrice, "no pricer, no quote")

	pricer := &articlePricerStub{}
	svc.SetPricer(pricer)
	cfg = svc.Config(ctx, 5, 7)
	require.NotNil(t, cfg.ImagePrice)
	require.InDelta(t, 0.201, *cfg.ImagePrice, 0.00001)
	require.Nil(t, svc.Config(ctx, 5, 8).ImagePrice, "another user's key is not quoted")
	require.Nil(t, svc.Config(ctx, 5, 0).ImagePrice)
	require.Equal(t, []int64{7}, pricer.keys)
}
