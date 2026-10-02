//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type memCloudFiles struct {
	files map[string]service.CanvasCloudFile
}

func (m *memCloudFiles) GetCanvasCloudFile(_ context.Context, _ int64, path string) (*service.CanvasCloudFile, error) {
	if f, ok := m.files[path]; ok {
		return &f, nil
	}
	return nil, nil
}
func (m *memCloudFiles) ListCanvasCloudFiles(context.Context, int64) ([]service.CanvasCloudFile, error) {
	out := []service.CanvasCloudFile{}
	for _, f := range m.files {
		out = append(out, f)
	}
	return out, nil
}
func (m *memCloudFiles) CanvasCloudUsage(context.Context, int64) (int64, int, error) {
	var total int64
	for _, f := range m.files {
		total += f.Size
	}
	return total, len(m.files), nil
}
func (m *memCloudFiles) PutCanvasCloudFile(_ context.Context, _ int64, f *service.CanvasCloudFile) (string, error) {
	old := m.files[f.Path].File
	m.files[f.Path] = *f
	return old, nil
}
func (m *memCloudFiles) DeleteCanvasCloudFile(_ context.Context, _ int64, path string) (string, error) {
	old := m.files[path].File
	delete(m.files, path)
	return old, nil
}

func TestCanvasCloudRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := &service.User{ID: 7, Email: "alice@example.com", Status: service.StatusActive, Role: "user"}
	sessions := service.NewCanvasSessionService(&memCanvasSessions{byHash: map[string]*service.CanvasSession{}}, canvasUsers{users: map[int64]*service.User{7: user}}, canvasSubs{})
	r := gin.New()
	r.POST("/api/v1/user/canvas-sessions", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Next()
	}, NewCanvasSessionHandler(sessions, nil).Create)
	cloud := NewCanvasCloudHandler(service.NewCanvasCloudService(&memCloudFiles{files: map[string]service.CanvasCloudFile{}}, t.TempDir(), nil, nil))
	group := r.Group("/api/v1/canvas/cloud", middleware.CanvasOriginGuard([]string{canvasOrigin}), middleware.CanvasSessionAuth(sessions))
	group.GET("/usage", cloud.Usage)
	group.GET("/files", cloud.List)
	group.GET("/files/*path", cloud.Get)
	group.PUT("/files/*path", cloud.Put)
	group.DELETE("/files/*path", cloud.Delete)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "https://hivegpt.example.test/api/v1/user/canvas-sessions", nil))
	token := strings.TrimPrefix(strings.Split(rec.Header().Get("Set-Cookie"), ";")[0], service.CanvasSessionCookie+"=")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, withBody(canvasRequest(http.MethodPut, "/api/v1/canvas/cloud/files/assets/files/x.html", token, nil), "text/html", "<script>alert(1)</script>"))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/cloud/files/assets/files/x.html", token, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	require.Equal(t, "attachment", rec.Header().Get("Content-Disposition"))
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Contains(t, rec.Header().Get("Content-Security-Policy"), "sandbox")
	require.Equal(t, "<script>alert(1)</script>", rec.Body.String())

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/cloud/files", token, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var list struct {
		Data struct {
			Files []service.CanvasCloudFile `json:"files"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list.Data.Files, 1)

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/cloud/files/assets/files/x.html", "", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code, "no session, no files")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/cloud/files/../../etc/passwd", token, nil))
	require.NotEqual(t, http.StatusOK, rec.Code)
}

func withBody(req *http.Request, contentType, body string) *http.Request {
	out := httptest.NewRequest(req.Method, req.URL.String(), strings.NewReader(body))
	out.Header = req.Header.Clone()
	out.Header.Set("Content-Type", contentType)
	return out
}
