//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// 真实 muse2api v1.6.6 的任务响应（2026-11-03 本地实测，提示词与 ID 已缩短）。
const (
	muse2APICreateBody     = `{"id":"task_906815","task_id":"task_906815","object":"video.task","status":"queued","progress":10,"created_at":1793642482}`
	muse2APIProcessingBody = `{"id":"task_906815","object":"task","kind":"video","prompt":"猫眨眼","api_prompt":"基于我本次上传的参考图片……","status":"processing","progress":15,"account":"9362983968a6"}`
	muse2APICompletedBody  = `{"id":"task_906815","object":"task","kind":"video","prompt":"猫眨眼","api_prompt":"基于我本次上传的参考图片……","status":"completed","result":{"url":"/v1/media/4de972.mp4","filename":"4de972.mp4","bytes":3833648,"kind":"video"},"error":null,"progress":100,"account":"9362983968a6","elapsed":76.5,"url":"/v1/media/4de972.mp4","video":{"url":"/v1/media/4de972.mp4"}}`
	muse2APIFailedBody     = `{"id":"task_6ea246","object":"task","kind":"video","status":"failed","result":null,"error":"提示词未成功送达后端（Still sending / Delivery not confirmed），终止等待并重试","progress":20}`
)

func muse2APITestAccount() *Account {
	return &Account{
		ID:       21,
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "m2a-key",
			"base_url": "http://muse2api:18610/v1",
		},
		Extra: map[string]any{grokVideoMuse2APICompatExtraKey: true},
	}
}

// muse2api 通常部署在内网、走 http：与默认配置一致，允许 http 上游地址。
func muse2APITestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.AllowPrivateHosts = true
	return cfg
}

func TestIsGrokVideoMuse2APICompat(t *testing.T) {
	require.True(t, muse2APITestAccount().IsGrokVideoMuse2APICompat())

	off := muse2APITestAccount()
	off.Extra = nil
	require.False(t, off.IsGrokVideoMuse2APICompat())

	oauth := muse2APITestAccount()
	oauth.Type = AccountTypeOAuth
	require.False(t, oauth.IsGrokVideoMuse2APICompat())

	openai := muse2APITestAccount()
	openai.Platform = PlatformOpenAI
	require.False(t, openai.IsGrokVideoMuse2APICompat())
}

func TestNormalizeMuse2APIVideoBodyMapsStatusesAndHidesPrivateFields(t *testing.T) {
	created := normalizeMuse2APIVideoBody([]byte(muse2APICreateBody))
	require.Equal(t, "pending", gjson.GetBytes(created, "status").String())
	require.Equal(t, "task_906815", gjson.GetBytes(created, "id").String())

	processing := normalizeMuse2APIVideoBody([]byte(muse2APIProcessingBody))
	require.Equal(t, "pending", gjson.GetBytes(processing, "status").String())
	require.False(t, IsGrokVideoStatusBillable(processing))

	done := normalizeMuse2APIVideoBody([]byte(muse2APICompletedBody))
	require.Equal(t, "done", gjson.GetBytes(done, "status").String())
	require.Equal(t, "/v1/media/4de972.mp4", gjson.GetBytes(done, "video.url").String())
	require.True(t, IsGrokVideoStatusBillable(done))
	for _, key := range []string{"url", "result", "account", "api_prompt"} {
		require.False(t, gjson.GetBytes(done, key).Exists(), key)
	}

	failed := normalizeMuse2APIVideoBody([]byte(muse2APIFailedBody))
	require.Equal(t, "failed", gjson.GetBytes(failed, "status").String())
	require.Contains(t, gjson.GetBytes(failed, "error").String(), "提示词未成功送达")
	require.False(t, gjson.GetBytes(failed, "video").Exists())
	require.False(t, IsGrokVideoStatusBillable(failed))

	// URL only at the top level (older muse2api builds) still ends up in video.url.
	topLevel := normalizeMuse2APIVideoBody([]byte(`{"status":"completed","url":"/v1/media/a.mp4"}`))
	require.Equal(t, "/v1/media/a.mp4", gjson.GetBytes(topLevel, "video.url").String())

	require.Equal(t, "not json", string(normalizeMuse2APIVideoBody([]byte("not json"))))
}

func TestMuse2APIVideoContentURLStaysOnUpstream(t *testing.T) {
	account := muse2APITestAccount()

	got, err := muse2APIVideoContentURL(account, []byte(`{"status":"done","video":{"url":"/v1/media/4de972.mp4"}}`))
	require.NoError(t, err)
	require.Equal(t, "http://muse2api:18610/v1/media/4de972.mp4", got)

	got, err = muse2APIVideoContentURL(account, []byte(`{"status":"done","video":{"url":"http://muse2api:18610/v1/media/b.mp4"}}`))
	require.NoError(t, err)
	require.Equal(t, "http://muse2api:18610/v1/media/b.mp4", got)

	for _, bad := range []string{
		`{"status":"done","video":{"url":"https://evil.example/v1/media/x.mp4"}}`,
		`{"status":"done","video":{"url":"//evil.example/x.mp4"}}`,
		`{"status":"done","video":{"url":"https://muse2api:18610/v1/media/x.mp4"}}`,
		`{"status":"done","video":{"url":"http://user@muse2api:18610/v1/media/x.mp4"}}`,
		`{"status":"done"}`,
	} {
		_, err := muse2APIVideoContentURL(account, []byte(bad))
		require.Error(t, err, bad)
	}
}

func TestForwardGrokMediaMuse2APIStatusIsBillableAndHidesUpstream(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(muse2APICompletedBody)}
	svc := &OpenAIGatewayService{cfg: muse2APITestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task_906815", nil)

	result, err := svc.ForwardGrokMedia(
		context.Background(), c, muse2APITestAccount(),
		GrokMediaEndpointVideoStatus, "task_906815", nil, "",
	)

	require.NoError(t, err)
	require.Equal(t, "http://muse2api:18610/v1/videos/task_906815", upstream.request.URL.String())
	require.Equal(t, "Bearer m2a-key", upstream.request.Header.Get("Authorization"))
	body := recorder.Body.String()
	require.Equal(t, "done", gjson.Get(body, "status").String())
	require.Equal(t, "/v1/videos/task_906815/content", gjson.Get(body, "video.url").String())
	require.NotContains(t, body, "/v1/media/")
	require.NotContains(t, body, "9362983968a6")
	require.NotContains(t, body, "api_prompt")
	// Billable once observed; the model comes from the create-time snapshot, not a Grok default.
	require.Equal(t, 1, result.VideoCount)
	require.Empty(t, result.Model)
}

func TestForwardGrokMediaMuse2APIProcessingIsNotBillable(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(muse2APIProcessingBody)}
	svc := &OpenAIGatewayService{cfg: muse2APITestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task_906815", nil)

	result, err := svc.ForwardGrokMedia(
		context.Background(), c, muse2APITestAccount(),
		GrokMediaEndpointVideoStatus, "task_906815", nil, "",
	)

	require.NoError(t, err)
	require.Equal(t, "pending", gjson.Get(recorder.Body.String(), "status").String())
	require.Zero(t, result.VideoCount)
}

func TestForwardGrokMediaMuse2APIContentFetchesMediaFromUpstream(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{
		responses: []*http.Response{
			grokMediaContentStatusResponse(muse2APICompletedBody),
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"video/mp4"}, "Content-Length": []string{"13"}},
				Body:       io.NopCloser(strings.NewReader("video-payload")),
			},
		},
	}
	svc := &OpenAIGatewayService{cfg: muse2APITestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task_906815/content", nil)

	result, err := svc.ForwardGrokMedia(
		context.Background(), c, muse2APITestAccount(),
		GrokMediaEndpointVideoContent, "task_906815", nil, "",
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "video-payload", recorder.Body.String())
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "http://muse2api:18610/v1/media/4de972.mp4", upstream.requests[1].URL.String())
	require.Equal(t, "Bearer m2a-key", upstream.requests[1].Header.Get("Authorization"))
	require.True(t, HTTPUpstreamRedirectsDisabled(upstream.requests[1].Context()))
	require.Equal(t, 1, result.VideoCount)
	require.Empty(t, result.Model)
}

func TestForwardGrokMediaMuse2APIContentBeforeDoneReturnsConflict(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(muse2APIProcessingBody)}
	svc := &OpenAIGatewayService{cfg: muse2APITestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodGet, "https://api.example/v1/videos/task_906815/content", nil)

	result, err := svc.ForwardGrokMedia(
		context.Background(), c, muse2APITestAccount(),
		GrokMediaEndpointVideoContent, "task_906815", nil, "",
	)

	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "video is not ready yet")
	require.Len(t, upstream.requests, 1)
	require.Zero(t, result.VideoCount)
}

func TestForwardGrokMediaMuse2APICreateNormalizesStatus(t *testing.T) {
	upstream := &grokMediaContentUpstreamStub{response: grokMediaContentStatusResponse(muse2APICreateBody)}
	svc := &OpenAIGatewayService{cfg: muse2APITestConfig(), httpUpstream: upstream}
	c, recorder := grokMediaContentTestContext(http.MethodPost, "https://api.example/v1/videos", nil)

	result, err := svc.ForwardGrokMedia(
		context.Background(), c, muse2APITestAccount(),
		GrokMediaEndpointVideosGenerations, "",
		[]byte(`{"model":"muse-video","prompt":"猫眨眼","duration":6,"aspect_ratio":"9:16","image":"data:image/png;base64,iVBORw0KGgo="}`),
		"application/json",
	)

	require.NoError(t, err)
	require.Equal(t, "http://muse2api:18610/v1/videos/generations", upstream.request.URL.String())
	sent, _ := io.ReadAll(upstream.request.Body)
	require.Equal(t, "muse-video", gjson.GetBytes(sent, "model").String())
	require.Equal(t, "9:16", gjson.GetBytes(sent, "aspect_ratio").String())
	require.NotEmpty(t, gjson.GetBytes(sent, "image").String())
	require.Equal(t, "pending", gjson.Get(recorder.Body.String(), "status").String())
	require.Equal(t, "task_906815", result.ResponseID)
}

func TestGrokVideoStatusWithoutModelKeepsSnapshotModel(t *testing.T) {
	body := []byte(`{"status":"done","video":{"url":"https://vidgen.x.ai/x.mp4"}}`)
	require.Empty(t, ExtractGrokVideoBillingFromStatusBody(body, nil, "req").Model)

	pending := &GrokVideoPendingBilling{Model: "muse-video", BillingModel: "muse-video", VideoDurationSeconds: 6}
	require.Equal(t, "muse-video", ExtractGrokVideoBillingFromStatusBody(body, pending, "req").BillingModel)
}
