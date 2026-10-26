package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type openAIZstdUpstreamRecorder struct {
	encoding      string
	contentLength int64
	body          []byte
}

func (u *openAIZstdUpstreamRecorder) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.encoding = req.Header.Get("Content-Encoding")
	u.contentLength = req.ContentLength
	u.body, _ = io.ReadAll(req.Body)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
}

func (u *openAIZstdUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func openAIZstdTestBody() []byte {
	// 模拟 Codex 请求：大段可压缩的历史文本 + 一张 base64 图片。
	var b strings.Builder
	_, _ = b.WriteString(`{"model":"gpt-6.1-sol","stream":true,"input":[`)
	for i := 0; i < 400; i++ {
		_, _ = b.WriteString(`{"type":"message","role":"user","content":[{"type":"input_text","text":"请帮我看看这个函数为什么返回空，日志如下：level=info msg=request_done status=200"}]},`)
	}
	_, _ = b.WriteString(`{"type":"message","role":"user","content":[{"type":"input_image","image_url":"data:image/png;base64,`)
	// 伪随机字节的 base64：只能靠熵编码省下约 1/4，用来盯住编码级别。
	rnd := make([]byte, 300_000)
	seed := uint32(7)
	for i := range rnd {
		seed = seed*1664525 + 1013904223
		rnd[i] = byte(seed >> 24)
	}
	_, _ = b.WriteString(base64.StdEncoding.EncodeToString(rnd))
	_, _ = b.WriteString(`"}]}]}`)
	return []byte(b.String())
}

func newOpenAIZstdTestRequest(t *testing.T, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://upstream.example/v1/responses", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAccount_IsOpenAIRequestZstdEnabled(t *testing.T) {
	on := map[string]any{"openai_request_zstd": true}
	require.True(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: on}).IsOpenAIRequestZstdEnabled())
	require.False(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: on}).IsOpenAIRequestZstdEnabled())
	require.False(t, (&Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Extra: on}).IsOpenAIRequestZstdEnabled())
	require.False(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}).IsOpenAIRequestZstdEnabled())
	require.False(t, (&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"openai_request_zstd": "true"}}).IsOpenAIRequestZstdEnabled())
	var nilAccount *Account
	require.False(t, nilAccount.IsOpenAIRequestZstdEnabled())
}

func TestCompressOpenAIUpstreamRequestBody_RoundTripsThroughSub2APIDecoder(t *testing.T) {
	body := openAIZstdTestBody()
	req := newOpenAIZstdTestRequest(t, body)

	compressOpenAIUpstreamRequestBody(req)

	require.Equal(t, "zstd", req.Header.Get("Content-Encoding"))
	require.Less(t, req.ContentLength, int64(len(body))*80/100)

	// 上游 sub2api 用 httputil.ReadRequestBodyWithPrealloc 读请求体，走真实 HTTP 往返验证能还原。
	var received []byte
	var receivedLen int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLen = r.ContentLength
		var err error
		received, err = httputil.ReadRequestBodyWithPrealloc(r)
		require.NoError(t, err)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	out, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL, req.Body)
	require.NoError(t, err)
	out.ContentLength = req.ContentLength
	out.Header = req.Header.Clone()
	resp, err := srv.Client().Do(out)
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.Equal(t, req.ContentLength, receivedLen)
	require.Equal(t, body, received)

	// GetBody 给重试用，必须返回同一份压缩内容。
	rc, err := req.GetBody()
	require.NoError(t, err)
	again, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Len(t, again, int(req.ContentLength))
}

func TestCompressOpenAIUpstreamRequestBody_ShrinksBase64Images(t *testing.T) {
	rnd := make([]byte, 600_000)
	seed := uint32(11)
	for i := range rnd {
		seed = seed*1664525 + 1013904223
		rnd[i] = byte(seed >> 24)
	}
	body := []byte(`{"input":[{"type":"input_image","image_url":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(rnd) + `"}]}`)
	req := newOpenAIZstdTestRequest(t, body)
	compressOpenAIUpstreamRequestBody(req)
	require.Equal(t, "zstd", req.Header.Get("Content-Encoding"))
	// 纯熵编码的理论下限约 75%；SpeedDefault 只能到 95% 左右。
	require.Less(t, req.ContentLength, int64(len(body))*80/100)
}

func TestCompressOpenAIUpstreamRequestBody_LeavesSmallOrEncodedBodiesAlone(t *testing.T) {
	small := []byte(`{"model":"gpt-6.1-sol","input":"hi"}`)
	req := newOpenAIZstdTestRequest(t, small)
	compressOpenAIUpstreamRequestBody(req)
	require.Empty(t, req.Header.Get("Content-Encoding"))
	got, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	require.Equal(t, small, got)

	body := openAIZstdTestBody()
	req = newOpenAIZstdTestRequest(t, body)
	req.Header.Set("Content-Encoding", "gzip")
	compressOpenAIUpstreamRequestBody(req)
	require.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
	require.Equal(t, int64(len(body)), req.ContentLength)
}

func TestCompressOpenAIUpstreamRequestBody_WithoutGetBody(t *testing.T) {
	body := openAIZstdTestBody()
	req := newOpenAIZstdTestRequest(t, body)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = nil
	req.ContentLength = -1

	compressOpenAIUpstreamRequestBody(req)

	require.Equal(t, "zstd", req.Header.Get("Content-Encoding"))
	compressed, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	decoded, err := openAIZstdDecodeForTest(compressed)
	require.NoError(t, err)
	require.Equal(t, body, decoded)
}

func TestDoOpenAIUpstream_CompressesOnlyWhenAccountOptsIn(t *testing.T) {
	body := openAIZstdTestBody()
	recorder := &openAIZstdUpstreamRecorder{}
	svc := &OpenAIGatewayService{httpUpstream: recorder}

	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
		Extra: map[string]any{"openai_request_zstd": true}}
	resp, err := svc.doOpenAIUpstream(newOpenAIZstdTestRequest(t, body), "", account)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, "zstd", recorder.encoding)
	require.Equal(t, int64(len(recorder.body)), recorder.contentLength)
	decoded, err := openAIZstdDecodeForTest(recorder.body)
	require.NoError(t, err)
	require.Equal(t, body, decoded)

	account.Extra = nil
	resp, err = svc.doOpenAIUpstream(newOpenAIZstdTestRequest(t, body), "", account)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Empty(t, recorder.encoding)
	require.Equal(t, body, recorder.body)
}

func openAIZstdDecodeForTest(data []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "http://decode.local", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Encoding", "zstd")
	return httputil.ReadRequestBodyWithPrealloc(req)
}
