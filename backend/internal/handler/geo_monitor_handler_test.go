//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// geoHandlerRepo answers the few calls these tests make; anything else panics.
type geoHandlerRepo struct {
	service.GeoMonitorRepository
	added []service.GeoCheck
}

func (r *geoHandlerRepo) ListQuestions(context.Context) ([]service.GeoQuestion, error) {
	return []service.GeoQuestion{{ID: 1, Question: "Q", Enabled: true}}, nil
}
func (r *geoHandlerRepo) ListEngines(context.Context) ([]service.GeoEngine, error) { return nil, nil }
func (r *geoHandlerRepo) GetQuestion(_ context.Context, id int64) (*service.GeoQuestion, error) {
	return &service.GeoQuestion{ID: id, Question: "Q"}, nil
}
func (r *geoHandlerRepo) InsertCheck(_ context.Context, c *service.GeoCheck) error {
	r.added = append(r.added, *c)
	return nil
}

func newGeoTestRouter(repo *geoHandlerRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewGeoMonitorHandler(service.NewGeoMonitorService(repo, nil, nil, nil))
	r := gin.New()
	r.POST("/geo/run", h.Run)
	r.POST("/geo/checks", h.AddManual)
	r.DELETE("/geo/checks/:id", h.DeleteCheck)
	return r
}

func geoDo(r *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestGeoHandlerManualAcceptsLinkText(t *testing.T) {
	repo := &geoHandlerRepo{}
	r := newGeoTestRouter(repo)
	w, _ := geoDo(r, http.MethodPost, "/geo/checks", `{"question_id":1,"engine_name":"豆包","answer":"推荐 HiveGPT","cited_urls":"https://hivegpt.cn/a\nhttps://b.com/ "}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.Len(t, repo.added, 1)
	require.Equal(t, []string{"https://hivegpt.cn/a", "https://b.com/"}, repo.added[0].CitedURLs)
	require.Equal(t, []string{"https://hivegpt.cn/a"}, repo.added[0].OurURLs)
	require.True(t, repo.added[0].Mentioned)

	w, out := geoDo(r, http.MethodPost, "/geo/checks", `{"question_id":"x"}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, out["message"], "请求格式不正确")
}

func TestGeoHandlerRunWithoutEngines(t *testing.T) {
	r := newGeoTestRouter(&geoHandlerRepo{})
	w, out := geoDo(r, http.MethodPost, "/geo/run", ``)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, out["message"], "没有可运行的组合")

	w, _ = geoDo(r, http.MethodDelete, "/geo/checks/abc", ``)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
