//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

var editorCreds = EditorCredentials{AppID: "wx0123456789abcdef", Secret: "0123456789abcdef0123456789abcdef"}

// A tiny PNG (1×1) and JPEG header good enough for content sniffing.
var (
	editorPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	editorJPEG   = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, make([]byte, 64)...)
)

func newEditorWithWechat(t *testing.T, h http.HandlerFunc) *EditorService {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	svc := NewEditorService(nil)
	svc.wechatAPI = srv.URL
	return svc
}

func TestEditorWechatTokenCachedAndErrors(t *testing.T) {
	ctx := context.Background()
	var tokens atomic.Int32
	svc := newEditorWithWechat(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/stable_token" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["secret"] == "ffffffffffffffffffffffffffffffff" {
				_, _ = io.WriteString(w, `{"errcode":40164,"errmsg":"invalid ip 1.2.3.4 ipv6 ::ffff:1.2.3.4, not in whitelist rid: x"}`)
				return
			}
			tokens.Add(1)
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":7200}`)
		}
	})
	require.NoError(t, svc.WechatCheck(ctx, editorCreds))
	require.NoError(t, svc.WechatCheck(ctx, editorCreds))
	require.EqualValues(t, 1, tokens.Load(), "token is cached")

	err := svc.WechatCheck(ctx, EditorCredentials{AppID: editorCreds.AppID, Secret: "ffffffffffffffffffffffffffffffff"})
	require.Equal(t, "EDITOR_WECHAT_IP", infraerrors.Reason(err))
	require.Contains(t, infraerrors.Message(err), "1.2.3.4", "tells which IP to whitelist")
	require.Contains(t, infraerrors.Message(err), "开发密钥", "points to where the whitelist lives now")
	require.Equal(t, "EDITOR_WECHAT_SECRET", infraerrors.Reason(wechatError(40243, "appsecret frozen")))

	require.ErrorIs(t, svc.WechatCheck(ctx, EditorCredentials{AppID: "nope", Secret: "x"}), ErrEditorCredentials)
}

func TestEditorUploadAndDraft(t *testing.T) {
	ctx := context.Background()
	var drafts atomic.Int32
	var gotDraft map[string]any
	var rawDraft string
	svc := newEditorWithWechat(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":7200}`)
		case "/cgi-bin/media/uploadimg":
			require.Equal(t, "tok", r.URL.Query().Get("access_token"))
			_, header, err := r.FormFile("media")
			require.NoError(t, err)
			require.Equal(t, "photo.png", header.Filename)
			_, _ = io.WriteString(w, `{"url":"https://mmbiz.qpic.cn/x.png"}`)
		case "/cgi-bin/material/add_material":
			require.Equal(t, "image", r.URL.Query().Get("type"))
			_, _ = io.WriteString(w, `{"media_id":"cover-1","url":"https://mmbiz.qpic.cn/c.jpg"}`)
		case "/cgi-bin/draft/add":
			// The first call says the token expired: the service refreshes once and retries.
			if drafts.Add(1) == 1 {
				_, _ = io.WriteString(w, `{"errcode":42001,"errmsg":"access_token expired"}`)
				return
			}
			raw, _ := io.ReadAll(r.Body)
			rawDraft = string(raw)
			_ = json.Unmarshal(raw, &gotDraft)
			_, _ = io.WriteString(w, `{"media_id":"draft-1"}`)
		}
	})

	up, err := svc.WechatUpload(ctx, editorCreds, "content", "photo.png", editorPNG)
	require.NoError(t, err)
	require.Equal(t, "https://mmbiz.qpic.cn/x.png", up.URL)
	_, err = svc.WechatUpload(ctx, editorCreds, "content", "big.png", append(editorPNG, make([]byte, editorContentImageMax)...))
	require.Equal(t, "EDITOR_WECHAT_FILE", infraerrors.Reason(err), "body images must be under 1MB")
	_, err = svc.WechatUpload(ctx, editorCreds, "content", "a.gif", []byte("GIF89a......"))
	require.Equal(t, "EDITOR_WECHAT_FILE", infraerrors.Reason(err), "body images are jpg / png only")
	cover, err := svc.WechatUpload(ctx, editorCreds, "cover", "c.jpg", editorJPEG)
	require.NoError(t, err)
	require.Equal(t, "cover-1", cover.MediaID)

	_, err = svc.WechatDraft(ctx, EditorDraftInput{EditorCredentials: editorCreds, Article: EditorArticle{Title: "t", Content: "<p>x</p>"}})
	require.Equal(t, "EDITOR_DRAFT", infraerrors.Reason(err), "a cover is required")

	id, err := svc.WechatDraft(ctx, EditorDraftInput{EditorCredentials: editorCreds, Article: EditorArticle{
		Title: " 标题 ", Content: `<section style="color:red"><p>正文 & <b>粗体</b></p></section>`, ThumbMediaID: "cover-1", NeedOpenComment: true}})
	require.NoError(t, err)
	require.Equal(t, "draft-1", id)
	require.Contains(t, rawDraft, `<section style=\"color:red\">`, "HTML is not \\u-escaped")
	article := gotDraft["articles"].([]any)[0].(map[string]any)
	require.Equal(t, "标题", article["title"])
	require.EqualValues(t, 1, article["need_open_comment"])
}

func TestEditorParseWechatArticle(t *testing.T) {
	page := `<html><head><meta property="og:title" content="一篇文章"><meta name="author" content="作者甲"></head><body>
<h1 id="activity-name">  页面标题 </h1>
<div id="js_content" style="visibility: hidden;"><p>第一段</p><script>alert(1)</script>
<p><img data-src="https://mmbiz.qpic.cn/a.png" src="data:image/gif;base64,xx"><img data-src="//mmbiz.qpic.cn/b.jpg"><img data-src="https://mmbiz.qpic.cn/a.png"></p></div></body></html>`
	got, err := parseWechatArticle([]byte(page))
	require.NoError(t, err)
	require.Equal(t, "一篇文章", got.Title)
	require.Equal(t, "作者甲", got.Author)
	require.Equal(t, []string{"https://mmbiz.qpic.cn/a.png", "https://mmbiz.qpic.cn/b.jpg"}, got.Images)
	require.Contains(t, got.HTML, `src="https://mmbiz.qpic.cn/a.png"`)
	require.NotContains(t, got.HTML, "<script")

	_, err = parseWechatArticle([]byte(`<html><body><p>没有正文</p></body></html>`))
	require.Equal(t, "EDITOR_FETCH", infraerrors.Reason(err))
}

func TestEditorURLGuards(t *testing.T) {
	ctx := context.Background()
	svc := NewEditorService(nil)
	for _, u := range []string{"https://example.com/s/x", "https://mp.weixin.qq.com.evil.com/s/x", "file:///etc/passwd", "https://mp.weixin.qq.com/cgi-bin/x"} {
		_, err := svc.ImportArticle(ctx, u)
		require.ErrorIs(t, err, ErrEditorArticleURL, u)
	}
	for _, u := range []string{"https://127.0.0.1/a.png", "https://mmbiz.qpic.cn.evil.com/a.png", "ftp://mmbiz.qpic.cn/a.png"} {
		_, _, err := svc.FetchImage(ctx, u)
		require.ErrorIs(t, err, ErrEditorImageURL, u)
	}
}

func TestEditorAI(t *testing.T) {
	ctx := context.Background()
	var gotAuth string
	var gotBody map[string]any
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		if r.URL.Path == "/v1/images/generations" {
			_, _ = io.WriteString(w, `{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(editorPNG)+`"}]}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"润色\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"后\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer gw.Close()
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 1, Key: "sk-mine", Status: StatusActive},
		{ID: 8, UserID: 2, Key: "sk-other", Status: StatusActive},
	}}
	learn, _ := newL2Service(t, gw.URL, LearnSources{Keys: keys})
	svc := NewEditorService(learn)

	var out strings.Builder
	collect := func(d string) error { out.WriteString(d); return nil }
	require.ErrorIs(t, svc.AIText(ctx, 1, EditorAITextInput{KeyID: 7, Action: "nope", Text: "x"}, collect), ErrEditorAIAction)
	require.ErrorIs(t, svc.AIText(ctx, 1, EditorAITextInput{KeyID: 7, Action: "polish", Text: "  "}, collect), ErrEditorAIText)
	require.Equal(t, "LEARN_OWN_KEY_REQUIRED", infraerrors.Reason(svc.AIText(ctx, 1, EditorAITextInput{Action: "polish", Text: "x"}, collect)))
	require.Equal(t, "LEARN_KEY_INVALID", infraerrors.Reason(svc.AIText(ctx, 1, EditorAITextInput{KeyID: 8, Action: "polish", Text: "x"}, collect)), "someone else's key")
	require.Equal(t, "EDITOR_AI_INSTRUCTION", infraerrors.Reason(svc.AIText(ctx, 1, EditorAITextInput{KeyID: 7, Action: "custom", Text: "x"}, collect)))

	require.NoError(t, svc.AIText(ctx, 1, EditorAITextInput{KeyID: 7, Action: "polish", Text: "原文", Title: "标题"}, collect))
	require.Equal(t, "润色后", out.String())
	require.Equal(t, "Bearer sk-mine", gotAuth)
	msgs := gotBody["messages"].([]any)
	require.Contains(t, msgs[0].(map[string]any)["content"], "润色")
	require.Contains(t, msgs[1].(map[string]any)["content"], "文章标题：标题")

	_, err := svc.AIImage(ctx, 1, EditorAIImageInput{KeyID: 7, Prompt: "猫", Size: "800x600"})
	require.ErrorIs(t, err, ErrEditorAIPrompt)
	img, err := svc.AIImage(ctx, 1, EditorAIImageInput{KeyID: 7, Prompt: "一只猫", Size: "1536x1024"})
	require.NoError(t, err)
	require.Equal(t, "image/png", img.Mime)
	require.Equal(t, "gpt-image-2", gotBody["model"])
	require.Equal(t, "1536x1024", gotBody["size"])
}
