//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeArchiveAPI struct {
	testIn      *service.ArchiveConfigInput
	testCalled  bool
	updateIn    service.ArchiveConfigInput
	updateAdmin int64
	filter      service.ArchiveSessionFilter
	turnsArgs   []any
	requestID   string
	image       []byte
	imageMime   string
	err         error
}

func (f *fakeArchiveAPI) GetConfig(context.Context) (*service.ArchiveConfigView, error) {
	return &service.ArchiveConfigView{Bucket: "b", SecretConfigured: true}, f.err
}

func (f *fakeArchiveAPI) UpdateConfig(_ context.Context, adminID int64, in service.ArchiveConfigInput, _, _ string) (*service.ArchiveConfigView, error) {
	f.updateAdmin, f.updateIn = adminID, in
	return &service.ArchiveConfigView{Bucket: in.Bucket}, f.err
}

func (f *fakeArchiveAPI) TestConnection(_ context.Context, in *service.ArchiveConfigInput) error {
	f.testCalled, f.testIn = true, in
	return f.err
}

func (f *fakeArchiveAPI) ApplyLifecycle(context.Context, int64, string, string) error { return f.err }

func (f *fakeArchiveAPI) ListSessions(_ context.Context, _ int64, filter service.ArchiveSessionFilter, _, _ string) (*service.ArchiveSessionPage, error) {
	f.filter = filter
	return &service.ArchiveSessionPage{Items: []service.ArchiveSession{}}, f.err
}

func (f *fakeArchiveAPI) ListTurns(_ context.Context, adminID, userID int64, sessionKey string, beforeID int64, limit int, _, _ string) (*service.ArchiveTurnPage, error) {
	f.turnsArgs = []any{adminID, userID, sessionKey, beforeID, limit}
	return &service.ArchiveTurnPage{Items: []service.ArchiveTurnView{}}, f.err
}

func (f *fakeArchiveAPI) GetTurnByRequest(_ context.Context, _ int64, requestID, _, _ string) (*service.ArchiveTurnView, error) {
	f.requestID = requestID
	return &service.ArchiveTurnView{ID: 5, UserID: 45, Email: "u@x", SessionKey: "s"}, f.err
}

func (f *fakeArchiveAPI) OpenImage(context.Context, int64) (io.ReadCloser, *service.ArchiveImage, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	return io.NopCloser(bytes.NewReader(f.image)), &service.ArchiveImage{Mime: f.imageMime, Bytes: len(f.image)}, nil
}

func (f *fakeArchiveAPI) Stats(context.Context) ([]service.ArchiveDayStats, error) {
	return []service.ArchiveDayStats{{Day: "2026-10-16", Turns: 3}}, f.err
}

func archiveRouter(api conversationArchiveAPI, admin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &ConversationArchiveHandler{svc: api}
	r := gin.New()
	g := r.Group("/a", func(c *gin.Context) {
		if admin {
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 1})
		}
	})
	g.GET("/config", h.GetConfig)
	g.PUT("/config", h.UpdateConfig)
	g.POST("/config/test", h.TestConfig)
	g.POST("/config/lifecycle", h.ApplyLifecycle)
	g.GET("/sessions", h.Sessions)
	g.GET("/sessions/turns", h.Turns)
	g.GET("/turns/by-request", h.TurnByRequest)
	g.GET("/images/:id", h.Image)
	g.GET("/stats", h.Stats)
	return r
}

func doArchive(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestConversationArchiveHandler_RequiresAdminSubject(t *testing.T) {
	api := &fakeArchiveAPI{}
	r := archiveRouter(api, false)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, "/a/config", `{"bucket":"b"}`},
		{http.MethodPost, "/a/config/lifecycle", ""},
		{http.MethodGet, "/a/sessions", ""},
		{http.MethodGet, "/a/sessions/turns?user_id=1&session_key=s", ""},
		{http.MethodGet, "/a/turns/by-request?request_id=x", ""},
	} {
		w := doArchive(r, tc.method, tc.path, tc.body)
		require.Equal(t, http.StatusUnauthorized, w.Code, tc.path)
	}
	require.Nil(t, api.filter.From)
	require.Nil(t, api.turnsArgs)

	// No service wired: a clear error, never a panic.
	w := doArchive(archiveRouter(nil, true), http.MethodGet, "/a/config", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "尚未配置")
}

func TestConversationArchiveHandler_ConfigEndpoints(t *testing.T) {
	api := &fakeArchiveAPI{}
	r := archiveRouter(api, true)

	w := doArchive(r, http.MethodGet, "/a/config", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"secret_configured":true`)

	// Secret omitted = keep the stored one (empty string reaches the service).
	w = doArchive(r, http.MethodPut, "/a/config", `{"enabled":true,"bucket":"b-1","access_key_id":"AK","retention_days":180}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(1), api.updateAdmin)
	require.Equal(t, "", api.updateIn.SecretAccessKey)
	require.Nil(t, api.updateIn.SSE)
	w = doArchive(r, http.MethodPut, "/a/config", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// Test: empty body tests the saved configuration; a form body is passed through.
	w = doArchive(r, http.MethodPost, "/a/config/test", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, api.testCalled)
	require.Nil(t, api.testIn)
	require.Contains(t, w.Body.String(), `"ok":true`)
	w = doArchive(r, http.MethodPost, "/a/config/test", `{"bucket":"b2","access_key_id":"AK"}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "b2", api.testIn.Bucket)

	api.err = service.ErrArchiveIncomplete
	w = doArchive(r, http.MethodPost, "/a/config/test", `{"bucket":"b2"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "Secret Access Key")
	api.err = nil

	w = doArchive(r, http.MethodPost, "/a/config/lifecycle", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"ok":true`)

	w = doArchive(r, http.MethodGet, "/a/stats", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"days":[{"day":"2026-10-16"`)
}

func TestConversationArchiveHandler_BrowseParams(t *testing.T) {
	api := &fakeArchiveAPI{}
	r := archiveRouter(api, true)
	w := doArchive(r, http.MethodGet, "/a/sessions?user_id=45&q=x&from=2026-10-01T00:00:00%2B08:00&to=2026-10-17T00:00:00Z&before=2026-10-16T10:00:00Z&limit=10", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(45), api.filter.UserID)
	require.Equal(t, "x", api.filter.Q)
	require.Equal(t, 10, api.filter.Limit)
	require.True(t, api.filter.From.Equal(time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)))
	require.True(t, api.filter.To.Equal(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)))
	require.NotNil(t, api.filter.Before)
	require.Contains(t, w.Body.String(), `"next_before":null`)

	w = doArchive(r, http.MethodGet, "/a/sessions?from=yesterday", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "RFC3339")

	w = doArchive(r, http.MethodGet, "/a/sessions/turns?user_id=45&session_key=abc&before_id=99&limit=5", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []any{int64(1), int64(45), "abc", int64(99), 5}, api.turnsArgs)

	w = doArchive(r, http.MethodGet, "/a/turns/by-request?request_id=client:1", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "client:1", api.requestID)
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.EqualValues(t, 45, resp.Data["user_id"])
	require.Equal(t, "s", resp.Data["session_key"])
	require.Equal(t, "u@x", resp.Data["email"])
}

func TestConversationArchiveHandler_ImageProxyHeaders(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\nrest")
	api := &fakeArchiveAPI{image: png, imageMime: "image/png"}
	r := archiveRouter(api, true)
	w := doArchive(r, http.MethodGet, "/a/images/12", "")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, png, w.Body.Bytes(), "raw bytes, not the JSON wrapper")
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
	require.Equal(t, "private, max-age=300", w.Header().Get("Cache-Control"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

	// A non-image type is never served as such.
	api.imageMime = "text/html"
	w = doArchive(r, http.MethodGet, "/a/images/12", "")
	require.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))

	w = doArchive(r, http.MethodGet, "/a/images/abc", "")
	require.Equal(t, http.StatusBadRequest, w.Code)
	api.err = service.ErrArchiveImageNotFound
	w = doArchive(r, http.MethodGet, "/a/images/13", "")
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestArchiveGrokImagesRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	// Off: nothing happens.
	archiveGrokImagesRequest(c, service.GrokMediaEndpointImagesGenerations, []byte(`{"prompt":"p"}`))
	require.Nil(t, service.ContentCaptureSnapshot(c))
}
