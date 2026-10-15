//go:build unit

package handler

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeContentLogTargets map[int64]bool

func (f fakeContentLogTargets) IsTarget(userID int64) bool { return f[userID] }

// runCapture sends the request through the capture step and a handler that reads the body the
// way the gateway handlers do; it returns what the handler read and the capture snapshot.
func runCapture(t *testing.T, targets fakeContentLogTargets, userID int64, req *http.Request) ([]byte, *service.ContentCapture, io.ReadCloser) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var handlerBody []byte
	var snap *service.ContentCapture
	var bodyBefore io.ReadCloser
	r := gin.New()
	r.POST("/*path", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{ID: 1, UserID: userID})
		bodyBefore = c.Request.Body
		captureContentLogRequest(c, targets)
		c.Next()
	}, func(c *gin.Context) {
		body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		require.NoError(t, err)
		handlerBody = body
		snap = service.ContentCaptureSnapshot(c)
		c.Status(http.StatusOK)
	})
	r.ServeHTTP(httptest.NewRecorder(), req)
	return handlerBody, snap, bodyBefore
}

func TestContentCapture_BodyIdenticalForHandler(t *testing.T) {
	body := []byte(`{"model":"gpt-5","input":[{"role":"user","content":[{"type":"input_text","text":"hello there"}]}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got, snap, _ := runCapture(t, fakeContentLogTargets{45: true}, 45, req)
	require.Equal(t, body, got)
	require.NotNil(t, snap)
	require.Equal(t, "hello there", snap.UserText)
}

func TestContentCapture_NonTargetUntouched(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	original := req.Body
	got, snap, before := runCapture(t, fakeContentLogTargets{45: true}, 7, req)
	require.Equal(t, body, got)
	require.Nil(t, snap)
	require.True(t, original == before)
	require.True(t, original == req.Body, "non-target requests keep their original body reader")

	// Targets on endpoints without a known protocol are not captured either.
	req = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(body))
	original = req.Body
	got, snap, _ = runCapture(t, fakeContentLogTargets{45: true}, 45, req)
	require.Equal(t, body, got)
	require.Nil(t, snap)
	require.True(t, original == req.Body)
}

func TestContentCapture_CompressedBody(t *testing.T) {
	plain := []byte(`{"system":"s","messages":[{"role":"user","content":[{"type":"text","text":"压缩的请求"}]}]}`)
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	_, _ = w.Write(plain)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(gz.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	got, snap, _ := runCapture(t, fakeContentLogTargets{45: true}, 45, req)
	require.Equal(t, plain, got, "the handler still decodes the body itself")
	require.Equal(t, "压缩的请求", snap.UserText)
}

func TestContentCapture_MultipartImagePrompt(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	file, _ := mw.CreateFormFile("image", "a.png")
	_, _ = file.Write(bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 1000))
	_ = mw.WriteField("model", "gpt-image-1")
	_ = mw.WriteField("prompt", "make it blue")
	_ = mw.Close()
	body := append([]byte(nil), buf.Bytes()...)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	got, snap, _ := runCapture(t, fakeContentLogTargets{45: true}, 45, req)
	require.Equal(t, body, got)
	require.Equal(t, "make it blue", snap.UserText)
}

func TestBufferRequestBodyForCapture_ReadErrorReplayed(t *testing.T) {
	readErr := errors.New("boom")
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.Body = io.NopCloser(io.MultiReader(strings.NewReader("partial"), errorReader{err: readErr}))
	data, ok := bufferRequestBodyForCapture(req)
	require.False(t, ok)
	require.Nil(t, data)
	got, err := io.ReadAll(req.Body)
	require.ErrorIs(t, err, readErr)
	require.Equal(t, "partial", string(got))
}

func TestContentLogProtocolForPath(t *testing.T) {
	cases := map[string]string{
		"/v1/responses":                                       "openai_responses",
		"/backend-api/codex/responses":                        "openai_responses",
		"/responses/compact":                                  "openai_responses",
		"/v1/responses/input_tokens":                          "",
		"/v1/chat/completions":                                "openai_chat_completions",
		"/v1/messages":                                        "anthropic_messages",
		"/v1/messages/count_tokens":                           "",
		"/v1/images/generations":                              "openai_images",
		"/v1/images/generations/async":                        "",
		"/v1beta/models/gemini-2.5-pro:streamGenerateContent": "gemini",
		"/v1/embeddings":                                      "",
	}
	for path, want := range cases {
		require.Equal(t, want, contentLogProtocolForPath(path), path)
	}
}
