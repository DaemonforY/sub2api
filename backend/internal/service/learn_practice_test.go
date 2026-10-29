//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func learnTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, h/2, color.RGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// learnPracticeGateway fakes the image and chat endpoints; the critic answers with the given items.
func learnPracticeGateway(t *testing.T, items []LearnCritiqueItem, seen *[]map[string]any) *httptest.Server {
	t.Helper()
	pic := base64.StdEncoding.EncodeToString(learnTestPNG(t, 2048, 1024))
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		body["_path"] = r.URL.Path
		*seen = append(*seen, body)
		switch r.URL.Path {
		case "/v1/images/generations":
			if strings.Contains(body["prompt"].(string), "拦截") {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"message":"图片没有生成：触发了内容安全审核"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + pic + `"}]}`))
		case "/v1/chat/completions":
			content, _ := json.Marshal(map[string]any{"items": items, "praise": "光的方向统一", "suggestion": "补上光的软硬", "revised_prompt": "改写后的提示词"})
			out, _ := json.Marshal(map[string]any{"model": "gpt-5.5", "choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(content)}}},
				"usage": map[string]any{"prompt_tokens": 5000, "completion_tokens": 300}})
			_, _ = w.Write(out)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestLearnDraw(t *testing.T) {
	ctx := context.Background()
	var seen []map[string]any
	items := []LearnCritiqueItem{{Dim: "光的方向", Score: 90, Note: "夕阳在身后"}, {Dim: "氛围", Score: 80, Note: "很温暖"},
		{Dim: "光效亮点", Score: 70, Note: "有轮廓光"}, {Dim: "提示词", Score: 50, Note: "没写光的颜色"}}
	gw := learnPracticeGateway(t, items, &seen)
	defer gw.Close()
	svc, repo := newL2Service(t, gw.URL, LearnSources{})

	res, err := svc.Draw(ctx, 1, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "桌上一个茶壶，光从左边来", Size: "1536x1024"})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(res.Image, "data:image/jpeg;base64,"))
	pic, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(res.Image, "data:image/jpeg;base64,"))
	require.NoError(t, err)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(pic))
	require.NoError(t, err)
	require.Equal(t, 1536, cfg.Width) // shrunk to the long-edge limit
	require.Equal(t, 768, cfg.Height)

	// Weighted total (30/30/20/20), computed here rather than taken from the model.
	require.NotNil(t, res.Critique)
	require.Equal(t, 75, res.Critique.Score) // (90*30 + 80*30 + 70*20 + 50*20) / 100
	require.True(t, res.Critique.Passed)
	require.Equal(t, "改写后的提示词", res.Critique.RevisedPrompt)
	require.Equal(t, 1, res.DrawsLeft)

	// The image request carries the admin's image model and quality; the critic gets a small picture.
	require.Equal(t, "/v1/images/generations", seen[0]["_path"])
	require.Equal(t, "gpt-image-2", seen[0]["model"])
	require.Equal(t, "low", seen[0]["quality"])
	require.Equal(t, "1536x1024", seen[0]["size"])
	critic, _ := json.Marshal(seen[1]["messages"])
	require.Contains(t, string(critic), "题目要求：给一个动漫场景打光")
	require.Contains(t, string(critic), "data:image/jpeg;base64,")
	require.Equal(t, "draw", repo.runs[0].kind)
	require.Len(t, repo.practice, 1)
	require.Equal(t, LearnPractice{UserID: 1, Lesson: "i3", Exercise: "i3-mood-light", Kind: "draw", Prompt: "桌上一个茶壶，光从左边来", Score: 75, Items: res.Critique.Items}, repo.practice[0])

	me, err := svc.Me(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, me.DrawsLeft)
	require.Equal(t, map[string]int{"i3-mood-light": 75}, me.Practice)

	// A blocked drawing is not counted against the free drawings.
	_, err = svc.Draw(ctx, 1, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "会被拦截的描述"})
	require.Equal(t, "LEARN_RUN_FAILED", infraerrors.Reason(err))
	require.Contains(t, infraerrors.Message(err), "内容安全审核")
	_, err = svc.Draw(ctx, 1, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "第二张"})
	require.NoError(t, err)
	_, err = svc.Draw(ctx, 1, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "第三张"})
	require.Equal(t, "LEARN_DRAW_QUOTA", infraerrors.Reason(err))

	for _, in := range []LearnDrawInput{
		{Exercise: "i3-nope", Prompt: "x"},
		{Exercise: "i5-photo", Prompt: "x"}, // an upload exercise, not a drawing
	} {
		_, err = svc.Draw(ctx, 2, in)
		require.Equal(t, "LEARN_EXERCISE_INVALID", infraerrors.Reason(err), in.Exercise)
	}
	_, err = svc.Draw(ctx, 2, LearnDrawInput{Exercise: "i3-mood-light", Prompt: " "})
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
	_, err = svc.Draw(ctx, 2, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "x", Size: "512x512"})
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
	_, err = svc.Draw(ctx, 2, LearnDrawInput{Exercise: "i3-mood-light", Prompt: strings.Repeat("光", learnMaxPromptRunes+1)})
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
}

func TestLearnDrawSiteCap(t *testing.T) {
	ctx := context.Background()
	var seen []map[string]any
	gw := learnPracticeGateway(t, []LearnCritiqueItem{{Dim: "光的方向", Score: 90}, {Dim: "氛围", Score: 90}, {Dim: "光效亮点", Score: 90}, {Dim: "提示词", Score: 90}}, &seen)
	defer gw.Close()
	svc, _ := newL2Service(t, gw.URL, LearnSources{})
	st := svc.Settings(ctx)
	st.DrawDailyCap = 1
	_, err := svc.SaveSettings(ctx, st)
	require.NoError(t, err)

	_, err = svc.Draw(ctx, 1, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "茶壶"})
	require.NoError(t, err)
	_, err = svc.Draw(ctx, 2, LearnDrawInput{Exercise: "i3-mood-light", Prompt: "茶壶"})
	require.Equal(t, "LEARN_DRAW_BUSY", infraerrors.Reason(err))
}

func TestLearnCritiqueUpload(t *testing.T) {
	ctx := context.Background()
	var seen []map[string]any
	items := []LearnCritiqueItem{{Dim: "主体", Score: 80}, {Dim: "构图方式", Score: 70}, {Dim: "平衡与留白", Score: 60}, {Dim: "边缘", Score: 50}}
	gw := learnPracticeGateway(t, items, &seen)
	defer gw.Close()
	svc, repo := newL2Service(t, gw.URL, LearnSources{})
	photo := "data:image/png;base64," + base64.StdEncoding.EncodeToString(learnTestPNG(t, 1200, 900))

	res, err := svc.Critique(ctx, 1, LearnCritiqueInput{Exercise: "i5-photo", Image: photo, Text: "古镇老街"})
	require.NoError(t, err)
	require.Equal(t, 67, res.Critique.Score) // 80*.3 + 70*.3 + 60*.2 + 50*.2
	require.True(t, res.Critique.Passed)     // pass mark 60
	require.Empty(t, res.Critique.RevisedPrompt)
	require.Equal(t, 9, res.CritiquesLeft)
	require.Equal(t, "critique", repo.runs[0].kind)
	require.Equal(t, "", repo.practice[0].Prompt) // a photo's note is not kept
	msgs, _ := json.Marshal(seen[0]["messages"])
	require.Contains(t, string(msgs), "学员的说明：古镇老街")
	require.Contains(t, string(msgs), "revised_prompt：留空字符串")

	for _, bad := range []string{"", "data:image/gif;base64,R0lGOD", "data:image/png;base64,!!!", "https://example.com/a.png",
		"data:image/png;base64," + strings.Repeat("A", learnMaxUploadBytes*4/3+8)} {
		_, err = svc.Critique(ctx, 1, LearnCritiqueInput{Exercise: "i5-photo", Image: bad})
		require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err), bad[:min(len(bad), 30)])
	}
	_, err = svc.Critique(ctx, 1, LearnCritiqueInput{Exercise: "i5-photo", Image: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not a picture"))})
	require.Equal(t, "LEARN_RUN_INVALID", infraerrors.Reason(err))
}

func TestGradeLearnCritique(t *testing.T) {
	ex := LearnExerciseDef{Pass: 70, Rubric: []LearnRubricItem{{Dim: "甲", Weight: 3}, {Dim: "乙", Weight: 1}}}
	c, err := gradeLearnCritique(ex, []LearnCritiqueItem{{Dim: "乙", Score: 40}, {Dim: "甲", Score: 120, Note: "好"}})
	require.NoError(t, err)
	require.Equal(t, []LearnCritiqueItem{{Dim: "甲", Score: 100, Note: "好"}, {Dim: "乙", Score: 40}}, c.Items) // rubric order, clamped
	require.Equal(t, 85, c.Score)
	require.True(t, c.Passed)

	// A renamed dimension is taken by position; a missing one is an error.
	c, err = gradeLearnCritique(ex, []LearnCritiqueItem{{Dim: "甲项", Score: 60}, {Dim: "乙", Score: 60}})
	require.NoError(t, err)
	require.Equal(t, 60, c.Score)
	require.False(t, c.Passed)
	_, err = gradeLearnCritique(ex, []LearnCritiqueItem{{Dim: "甲", Score: 60}})
	require.Error(t, err)
}

func TestLearnPracticeCertificate(t *testing.T) {
	ctx := context.Background()
	svc, repo := newL2Service(t, "http://127.0.0.1:1", LearnSources{})
	_, err := svc.CertStatus(ctx, 1, "i")
	require.Equal(t, "LEARN_CERT_CLOSED", infraerrors.Reason(err))

	// Once the track is complete: every exercise of its lessons must reach its pass mark.
	svc.catalog.track("i").CertClosed = false
	defer func() { svc.catalog.track("i").CertClosed = true }()
	repo.practice = append(repo.practice, LearnPractice{UserID: 1, Exercise: "i3-mood-light", Score: 82}, LearnPractice{UserID: 1, Exercise: "i5-photo", Score: 40})
	st, err := svc.CertStatus(ctx, 1, "i")
	require.NoError(t, err)
	project := st.Items[len(st.Items)-1]
	require.False(t, project.Done)
	require.Contains(t, project.Detail, "已达标 1 / 3 个")
	require.Contains(t, project.Detail, "给你的照片做构图体检")
	repo.practice = append(repo.practice, LearnPractice{UserID: 1, Exercise: "i5-photo", Score: 61}, LearnPractice{UserID: 1, Exercise: "i5-wallpaper", Score: 70})
	st, err = svc.CertStatus(ctx, 1, "i")
	require.NoError(t, err)
	require.True(t, st.Items[len(st.Items)-1].Done)
}

// Pages use only exercises the server knows, each on its own lesson's page.
func TestLearnExercisesMatchPages(t *testing.T) {
	c, err := loadLearnCatalog(learnCatalogJSON)
	require.NoError(t, err)
	used := map[string]bool{}
	for _, track := range c.Tracks {
		for _, id := range track.Lessons {
			page, err := os.ReadFile("../../../learn/" + id[:1] + "/" + id + ".md")
			if os.IsNotExist(err) {
				continue // 大数据 article pages have no exercises
			}
			require.NoError(t, err)
			for _, m := range regexp.MustCompile(`<(DrawBox|CritiqueBox)\s+exercise="([a-z0-9-]+)"`).FindAllStringSubmatch(string(page), -1) {
				ex, ok := c.Exercises[m[2]]
				require.True(t, ok, "%s uses unknown exercise %s", id, m[2])
				require.Equal(t, id, ex.Lesson, "exercise %s", m[2])
				require.Equal(t, m[1] == "DrawBox", ex.Kind == learnExerciseDraw, "exercise %s component", m[2])
				used[m[2]] = true
			}
		}
	}
	for id := range c.Exercises {
		require.True(t, used[id], "exercise %s is on no page", id)
	}
}
