//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIImageAspectHintNamesOrientationAndRatio(t *testing.T) {
	cases := map[string]string{
		"1024x1536": "【画幅】竖版构图，画幅宽:高 = 2:3（1024×1536）",
		"1536x1024": "【画幅】横版构图，画幅宽:高 = 3:2（1536×1024）",
		"1024x1024": "【画幅】方形构图，画幅宽:高 = 1:1（1024×1024）",
		"864x1536":  "竖版构图，画幅宽:高 = 9:16",
		"2016x864":  "横版构图，画幅宽:高 = 21:9",
		"768x1024":  "竖版构图，画幅宽:高 = 3:4",
	}
	for size, want := range cases {
		require.Contains(t, openAIImageAspectHint(size), want, size)
	}
	require.Contains(t, openAIImageAspectHint("1024x1536"), "Aspect ratio: portrait 2:3.")
	for _, size := range []string{"", "auto", "1024", "1:1", "0x100"} {
		require.Empty(t, openAIImageAspectHint(size), size)
	}

	p, ok := withOpenAIImageAspectHint("画一只猫  \n", "1024x1536")
	require.True(t, ok)
	require.True(t, strings.HasPrefix(p, "画一只猫\n\n【画幅】"))
	_, ok = withOpenAIImageAspectHint(p, "1024x1536")
	require.False(t, ok, "a prompt that already has the hint is left alone")
	_, ok = withOpenAIImageAspectHint("", "1024x1536")
	require.False(t, ok)
	_, ok = withOpenAIImageAspectHint("cat", "auto")
	require.False(t, ok)
}

func aspectHintService(value string) *OpenAIGatewayService {
	InvalidateOpenAIImageAspectHint()
	repo := &libSettingsFake{v: map[string]string{}}
	if value != "" {
		repo.v[SettingKeyOpenAIImageAspectHint] = value
	}
	return &OpenAIGatewayService{settingService: &SettingService{settingRepo: repo}}
}

func TestOpenAIImageAspectHintRewritesJSONAndMultipart(t *testing.T) {
	t.Cleanup(InvalidateOpenAIImageAspectHint)
	ctx := context.Background()
	svc := aspectHintService("")

	body := []byte(`{"model":"gpt-image-2","prompt":"画一只猫","size":"1024x1536","n":1}`)
	parsed := &OpenAIImagesRequest{Model: "gpt-image-2", Prompt: "画一只猫", Size: "1024x1536", ContentType: "application/json"}
	out, hinted, err := svc.applyOpenAIImageAspectHint(ctx, body, parsed, "")
	require.NoError(t, err)
	require.Contains(t, gjson.GetBytes(out, "prompt").String(), "【画幅】竖版构图")
	require.Equal(t, "1024x1536", gjson.GetBytes(out, "size").String())
	require.Contains(t, hinted.Prompt, "【画幅】")
	require.Equal(t, "画一只猫", parsed.Prompt, "the parsed request is copied, not changed")

	// Edits (multipart): the prompt field is replaced, the image part kept.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("model", "gpt-image-2"))
	require.NoError(t, w.WriteField("prompt", "把背景换成海边"))
	require.NoError(t, w.WriteField("size", "1536x1024"))
	fw, err := w.CreateFormFile("image", "a.png")
	require.NoError(t, err)
	_, _ = fw.Write([]byte("PNGDATA"))
	require.NoError(t, w.Close())
	mp := &OpenAIImagesRequest{Model: "gpt-image-2", Prompt: "把背景换成海边", Size: "1536x1024", ContentType: w.FormDataContentType()}
	out, hinted, err = svc.applyOpenAIImageAspectHint(ctx, buf.Bytes(), mp, "")
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(hinted.ContentType)
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(out), params["boundary"])
	fields := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, _ := io.ReadAll(part)
		fields[part.FormName()] = string(data)
	}
	require.Contains(t, fields["prompt"], "把背景换成海边\n\n【画幅】横版构图，画幅宽:高 = 3:2")
	require.Equal(t, "PNGDATA", fields["image"])
	require.Equal(t, "gpt-image-2", fields["model"])

	// Not gpt-image, no explicit size, or switched off: untouched.
	grok := &OpenAIImagesRequest{Model: "grok-imagine-image", Prompt: "猫", Size: "1024x1024", ContentType: "application/json"}
	out, _, err = svc.applyOpenAIImageAspectHint(ctx, body, grok, "")
	require.NoError(t, err)
	require.Equal(t, string(body), string(out))
	off := aspectHintService("false")
	out, same, err := off.applyOpenAIImageAspectHint(ctx, body, parsed, "")
	require.NoError(t, err)
	require.Equal(t, string(body), string(out))
	require.Same(t, parsed, same)
}

func TestOpenAIGatewayServiceForwardImages_APIKeySendsTheAspectHint(t *testing.T) {
	t.Cleanup(InvalidateOpenAIImageAspectHint)
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat","size":"1024x1024","response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := aspectHintService("")
	svc.cfg = &config.Config{}
	svc.httpUpstream = &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`)),
	}}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	account := &Account{ID: 6, Name: "openai-apikey", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "test-api-key", "base_url": "https://image-upstream.example/v1"}}

	_, err = svc.ForwardImages(context.Background(), c, account, body, parsed, "")
	require.NoError(t, err)
	upstream := svc.httpUpstream.(*httpUpstreamRecorder)
	require.Equal(t, "draw a cat\n\n【画幅】方形构图，画幅宽:高 = 1:1（1024×1024），请按这个比例构图。Aspect ratio: square 1:1.", gjson.GetBytes(upstream.lastBody, "prompt").String())
	require.Equal(t, "1024x1024", gjson.GetBytes(upstream.lastBody, "size").String())
}

func TestImageToolsAspectHintSwitch(t *testing.T) {
	t.Cleanup(InvalidateOpenAIImageAspectHint)
	settings := &imageToolSettingsStub{values: map[string]string{}}
	svc := NewImageToolsService(&imageToolsRepoStub{}, nil, nil, ImageToolsConfig{}).WithSettings(settings)
	v, err := svc.AspectHint(context.Background())
	require.NoError(t, err)
	require.True(t, v.Enabled, "on by default")
	require.Equal(t, int64(7), v.Week[0].Wrong)
	v, err = svc.SaveAspectHint(context.Background(), false)
	require.NoError(t, err)
	require.False(t, v.Enabled)
	require.Equal(t, "false", settings.values[SettingKeyOpenAIImageAspectHint])
}
