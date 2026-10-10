//go:build unit

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type siteDraftRepoStub struct {
	mu   sync.Mutex
	next int64
	rows map[int64]SiteDraft
}

func (r *siteDraftRepoStub) clone(d SiteDraft) SiteDraft {
	b, _ := json.Marshal(d.SiteDraftData)
	var data SiteDraftData
	_ = json.Unmarshal(b, &data)
	d.SiteDraftData = data
	return d
}

func (r *siteDraftRepoStub) Create(_ context.Context, d *SiteDraft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	d.ID, d.CreatedAt, d.UpdatedAt = r.next, time.Now(), time.Now()
	r.rows[d.ID] = r.clone(*d)
	return nil
}

func (r *siteDraftRepoStub) Get(_ context.Context, userID, id int64) (*SiteDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.rows[id]
	if !ok || d.UserID != userID {
		return nil, nil
	}
	c := r.clone(d)
	return &c, nil
}

func (r *siteDraftRepoStub) List(_ context.Context, userID int64, _ int) ([]SiteDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []SiteDraft{}
	for _, d := range r.rows {
		if d.UserID == userID {
			out = append(out, r.clone(d))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (r *siteDraftRepoStub) Save(_ context.Context, d *SiteDraft) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[d.ID] = r.clone(*d)
	return nil
}

func (r *siteDraftRepoStub) CountActive(_ context.Context, userID int64) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, d := range r.rows {
		if d.UserID == userID && siteDraftActive(d.Status) {
			n++
		}
	}
	return n, nil
}

func (r *siteDraftRepoStub) FailActive(context.Context, string) (int64, error) { return 0, nil }

func (r *siteDraftRepoStub) Delete(_ context.Context, _, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, id)
	return nil
}

func (r *siteDraftRepoStub) DeleteBefore(context.Context, time.Time) ([]int64, error) { return nil, nil }

type sitePublisherStub struct {
	uploads []SiteUpload
	siteIDs []int64
	err     error
}

func (p *sitePublisherStub) Create(_ context.Context, _ int64, up SiteUpload) (*Site, error) {
	if p.err != nil {
		return nil, p.err
	}
	p.uploads, p.siteIDs = append(p.uploads, up), append(p.siteIDs, 0)
	return &Site{ID: 31, Name: "cafe", URL: "https://cafe.s.example.com"}, nil
}

func (p *sitePublisherStub) Update(_ context.Context, _, siteID int64, up SiteUpload) (*Site, error) {
	p.uploads, p.siteIDs = append(p.uploads, up), append(p.siteIDs, siteID)
	return &Site{ID: siteID, Name: "cafe", URL: "https://cafe.s.example.com"}, nil
}

func sitePage(body string) string {
	return "<!DOCTYPE html>\n<html lang=\"zh-CN\"><head><title>街角咖啡</title></head><body>" + body + "</body></html>"
}

func newSiteBuilderForTest(t *testing.T) (*SiteBuilderService, *siteDraftRepoStub, *sitePublisherStub) {
	t.Helper()
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 5, Name: "建站", Key: "sk-user5", Status: StatusActive},
		{ID: 8, UserID: 6, Name: "别人的", Key: "sk-user6", Status: StatusActive},
	}}
	learn, _ := newL2Service(t, "http://127.0.0.1:1", LearnSources{Keys: keys})
	repo := &siteDraftRepoStub{rows: map[int64]SiteDraft{}}
	pub := &sitePublisherStub{}
	return NewSiteBuilderService(repo, learn, pub, t.TempDir()), repo, pub
}

func waitSiteDraft(t *testing.T, svc *SiteBuilderService, id int64) {
	t.Helper()
	require.Eventually(t, func() bool { return !svc.running(id) }, 5*time.Second, 5*time.Millisecond)
}

func TestSiteBuilderFlow(t *testing.T) {
	ctx := context.Background()
	svc, _, pub := newSiteBuilderForTest(t)

	pages := []string{
		// first page: picture 1 plus a picture over the limit and one with no description
		"好的：\n```html\n" + sitePage(`<img src="img/1.jpg" data-ai-prompt="清晨的咖啡馆 &amp; 阳光" alt="店内"><img src="img/2.jpg" data-ai-prompt="多余"><img src="img/3.jpg" alt="无描述">`) + "\n```",
		// change: keeps picture 1, adds picture 2 (square)
		sitePage(`<h1>新标题</h1><img src="img/1.jpg" data-ai-prompt="清晨的咖啡馆 &amp; 阳光" alt="店内"><img src="img/2.jpg" data-ai-prompt="拿铁拉花" data-ai-size="square" alt="拉花">`),
		// change: redraws picture 1
		sitePage(`<h1>新标题</h1><img src="img/1.jpg" data-ai-prompt="夜晚的咖啡馆" alt="店内"><img src="img/2.jpg" data-ai-prompt="拿铁拉花" data-ai-size="square" alt="拉花">`),
	}
	var requests []string
	var keysSeen []string
	svc.chat = func(_ context.Context, key, _ string, msgs []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		keysSeen = append(keysSeen, key)
		requests = append(requests, msgs[0].Content+"\n----\n"+msgs[1].Content)
		_ = onDelta(pages[len(requests)-1])
		return &LearnTutorResult{Model: "gpt-test", PromptTokens: 100, CompletionTokens: 200}, nil
	}
	png := articlePNG(t)
	var drawn []string
	var drawMu sync.Mutex
	svc.draw = func(_ context.Context, key, prompt, size string) ([]byte, error) {
		drawMu.Lock()
		defer drawMu.Unlock()
		keysSeen = append(keysSeen, key)
		drawn = append(drawn, size+" "+strings.SplitN(prompt, "\n", 2)[0])
		return png, nil
	}

	_, err := svc.Create(ctx, 5, SiteDraftCreateInput{KeyID: 8, SiteDraftBrief: SiteDraftBrief{Description: "咖啡馆"}})
	require.ErrorIs(t, err, ErrLearnKeyInvalid, "only the user's own key")
	_, err = svc.Create(ctx, 5, SiteDraftCreateInput{KeyID: 7, SiteDraftBrief: SiteDraftBrief{Description: "  "}})
	require.ErrorIs(t, err, ErrSiteDraftDescription)

	d, err := svc.Create(ctx, 5, SiteDraftCreateInput{KeyID: 7, SiteDraftBrief: SiteDraftBrief{Description: "街角咖啡馆官网\n要有菜单", Images: 1}})
	require.NoError(t, err)
	require.Equal(t, SiteDraftGenerating, d.Status)
	require.Equal(t, "街角咖啡馆官网", d.Title)
	waitSiteDraft(t, svc, d.ID)

	got, err := svc.Get(ctx, 5, d.ID)
	require.NoError(t, err)
	require.Equal(t, SiteDraftReady, got.Status, got.Error)
	require.Equal(t, "街角咖啡", got.Title, "from the page's <title>")
	require.True(t, strings.HasPrefix(got.HTML, "<!DOCTYPE html>"), "code fence and chatter removed")
	require.NotContains(t, got.HTML, "img/2.jpg", "over the asked number of pictures")
	require.NotContains(t, got.HTML, "img/3.jpg", "a picture with no description")
	require.Equal(t, []SiteDraftImage{{N: 1, Prompt: "清晨的咖啡馆 & 阳光", Size: "1536x1024", Status: "ok", Bytes: got.Images[0].Bytes}}, got.Images)
	require.Equal(t, []string{"1536x1024 清晨的咖啡馆 & 阳光"}, drawn)
	require.False(t, got.CanUndo)
	require.Equal(t, "ok", got.Turns[0].Status)
	require.Contains(t, requests[0], "可以新增图片的编号：1")
	_, err = svc.Get(ctx, 6, d.ID)
	require.ErrorIs(t, err, ErrSiteDraftNotFound, "another user's draft")

	// A change keeps picture 1 and draws only the new one.
	_, err = svc.Revise(ctx, 5, d.ID, "加一张拉花的图，标题改成新标题")
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	got, _ = svc.Get(ctx, 5, d.ID)
	require.Equal(t, SiteDraftReady, got.Status, got.Error)
	require.Contains(t, requests[1], "已有的图片编号：1")
	require.Contains(t, requests[1], "可以新增图片的编号：2、3、4")
	require.Contains(t, requests[1], "清晨的咖啡馆", "the current page goes with the request")
	require.Equal(t, []string{"1536x1024 清晨的咖啡馆 & 阳光", "1024x1024 拿铁拉花"}, drawn)
	require.True(t, got.CanUndo)

	// Redrawing picture 1, then undo: the earlier picture is still on disk, nothing is drawn again.
	_, err = svc.Revise(ctx, 5, d.ID, "改成夜景")
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	require.Len(t, drawn, 3)
	got, err = svc.Undo(ctx, 5, d.ID)
	require.NoError(t, err)
	require.Contains(t, got.HTML, "清晨的咖啡馆")
	for _, im := range got.Images {
		require.Equal(t, "ok", im.Status, im.N)
	}
	require.Len(t, drawn, 3)

	// Publish: a zip with index.html (no drawing hints) and the pictures.
	site, err := svc.Publish(ctx, 5, d.ID, SiteDraftPublishInput{Name: "cafe"})
	require.NoError(t, err)
	require.Equal(t, int64(31), site.ID)
	up := pub.uploads[0]
	require.Equal(t, "街角咖啡", up.Title)
	require.Equal(t, "cafe", up.Name)
	files := unzipForTest(t, up.Data)
	require.Contains(t, files, "img/1.jpg")
	require.Contains(t, files, "img/2.jpg")
	require.NotContains(t, files["index.html"], "data-ai-")
	require.Contains(t, files["index.html"], `src="img/2.jpg"`)
	got, _ = svc.Get(ctx, 5, d.ID)
	require.Equal(t, int64(31), got.SiteID)
	require.Equal(t, "https://cafe.s.example.com", got.SiteURL)

	_, err = svc.Publish(ctx, 5, d.ID, SiteDraftPublishInput{SiteID: 31})
	require.NoError(t, err)
	require.Equal(t, []int64{0, 31}, pub.siteIDs, "then a new version of the same site")

	for _, k := range keysSeen {
		require.Equal(t, "sk-user5", k)
	}
}

func unzipForTest(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		out[f.Name] = string(b)
	}
	return out
}

func TestSiteBuilderFailures(t *testing.T) {
	ctx := context.Background()
	svc, _, pub := newSiteBuilderForTest(t)
	reply := sitePage(`<img src="img/1.jpg" data-ai-prompt="门头" alt="门头">`)
	var chatErr error
	svc.chat = func(_ context.Context, _, _ string, _ []LearnMessage, _ int, onDelta func(string) error) (*LearnTutorResult, error) {
		if chatErr != nil {
			return nil, chatErr
		}
		_ = onDelta(reply)
		return &LearnTutorResult{}, nil
	}
	drawFails := true
	svc.draw = func(context.Context, string, string, string) ([]byte, error) {
		if drawFails {
			return nil, errLearnRunFailed("画图服务忙")
		}
		return articlePNG(t), nil
	}

	// A first page that is cut off fails the draft; retry writes it again.
	reply = "<!DOCTYPE html><html><body><h1>写到一半"
	d, err := svc.Create(ctx, 5, SiteDraftCreateInput{KeyID: 7, SiteDraftBrief: SiteDraftBrief{Description: "咖啡馆", Images: 1}})
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	got, _ := svc.Get(ctx, 5, d.ID)
	require.Equal(t, SiteDraftFailed, got.Status)
	require.Contains(t, got.Error, "没写完")
	require.Empty(t, got.DraftHTML)
	_, err = svc.Publish(ctx, 5, d.ID, SiteDraftPublishInput{})
	require.ErrorIs(t, err, ErrSiteDraftEmpty)

	reply = sitePage(`<img src="img/1.jpg" data-ai-prompt="门头" alt="门头"><p>菜单</p>`)
	_, err = svc.Retry(ctx, 5, d.ID)
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	got, _ = svc.Get(ctx, 5, d.ID)
	require.Equal(t, SiteDraftReady, got.Status, "a failed picture does not fail the page")
	require.Equal(t, "failed", got.Images[0].Status)
	require.Contains(t, got.Images[0].Error, "画图服务忙")

	// Publishing now leaves out the picture that failed.
	_, err = svc.Publish(ctx, 5, d.ID, SiteDraftPublishInput{})
	require.NoError(t, err)
	files := unzipForTest(t, pub.uploads[0].Data)
	require.NotContains(t, files["index.html"], "<img")
	require.NotContains(t, files, "img/1.jpg")

	// Retry draws the missing picture.
	drawFails = false
	_, err = svc.Retry(ctx, 5, d.ID)
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	got, _ = svc.Get(ctx, 5, d.ID)
	require.Equal(t, "ok", got.Images[0].Status)
	_, err = svc.Retry(ctx, 5, d.ID)
	require.ErrorIs(t, err, ErrSiteDraftState, "nothing left to retry")

	// A failed change keeps the page that was there.
	before := got.HTML
	chatErr = errors.New("upstream down")
	_, err = svc.Revise(ctx, 5, d.ID, "改成蓝色")
	require.NoError(t, err)
	waitSiteDraft(t, svc, d.ID)
	got, _ = svc.Get(ctx, 5, d.ID)
	require.Equal(t, SiteDraftReady, got.Status)
	require.Equal(t, before, got.HTML)
	require.NotEmpty(t, got.Error)
	require.Equal(t, "failed", got.Turns[len(got.Turns)-1].Status)
	require.False(t, got.CanUndo)
	_, err = svc.Undo(ctx, 5, d.ID)
	require.ErrorIs(t, err, ErrSiteDraftNoUndo)

	_, err = svc.Revise(ctx, 5, d.ID, " ")
	require.ErrorIs(t, err, ErrSiteDraftInstruction)
	require.NoError(t, svc.Delete(ctx, 5, d.ID))
	_, err = svc.Get(ctx, 5, d.ID)
	require.ErrorIs(t, err, ErrSiteDraftNotFound)
}

func TestCleanSitePage(t *testing.T) {
	page, err := cleanSitePage("说明文字\n<!doctype html><html><body>hi</body></html>\n以上")
	require.NoError(t, err)
	require.Equal(t, "<!doctype html><html><body>hi</body></html>", page)
	_, err = cleanSitePage("我不会写网页")
	require.Error(t, err)
	_, err = cleanSitePage("<html><head></head></html>")
	require.Error(t, err, "no body")

	out, imgs := pageImages(`<IMG SRC='img/2.jpg' data-ai-prompt="a"><img src="img/2.jpg" data-ai-prompt="b"><img src="logo.png"><img src="img/9.jpg" data-ai-prompt="x">`, 4)
	require.Equal(t, []SiteDraftImage{{N: 2, Prompt: "a", Size: "1536x1024", Status: "pending"}}, imgs, "first use of a number wins")
	require.Contains(t, out, `logo.png`, "other pictures are left alone")
	require.NotContains(t, out, "img/9.jpg")
}
