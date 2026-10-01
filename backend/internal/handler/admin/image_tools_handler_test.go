package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imageToolsAdminRepo struct {
	service.ImageToolsRepository
	query service.ImageToolUseQuery
}

func (r *imageToolsAdminRepo) ListUses(_ context.Context, q service.ImageToolUseQuery) ([]service.ImageToolUseRecord, int64, error) {
	r.query = q
	return []service.ImageToolUseRecord{{ID: 1, UserID: 7, UserEmail: "a@b.c", Tool: service.ImageToolUpscale, Cost: 0.05}}, 1, nil
}

type imageToolsAdminSettings struct{ values map[string]string }

func (s *imageToolsAdminSettings) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	return s.values, nil
}
func (s *imageToolsAdminSettings) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		s.values[k] = v
	}
	return nil
}

func TestImageToolsAdminHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &imageToolsAdminRepo{}
	settings := &imageToolsAdminSettings{values: map[string]string{}}
	svc := service.NewImageToolsService(repo, nil, nil, service.ImageToolsConfig{BaseURL: "http://imagetools:7000", PriceRemoveBg: 0.02, PriceUpscale: 0.05, FreeDaily: 3}).WithSettings(settings)
	h := NewImageToolsHandler(svc)
	r := gin.New()
	r.GET("/settings", h.Settings)
	r.PUT("/settings", h.SaveSettings)
	r.GET("/uses", h.Uses)
	do := func(method, path string, body any) (int, map[string]any) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	code, out := do(http.MethodGet, "/settings", nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, map[string]any{"enabled": true, "price_remove_bg": 0.02, "price_upscale": 0.05, "free_daily": float64(3), "service_configured": true}, out["data"])

	code, _ = do(http.MethodPut, "/settings", map[string]any{"enabled": true, "price_remove_bg": 200, "price_upscale": 0.05, "free_daily": 3})
	require.Equal(t, http.StatusBadRequest, code)
	code, out = do(http.MethodPut, "/settings", map[string]any{"enabled": false, "price_remove_bg": 0.03, "price_upscale": 0.08, "free_daily": 5})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, false, out["data"].(map[string]any)["enabled"])
	require.Equal(t, "0.08", settings.values["image_tools_price_upscale"])

	code, out = do(http.MethodGet, "/uses?q=%20a@b%20&tool=upscale&start_date=2026-10-01&end_date=2026-10-01&page=2&page_size=10", nil)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, float64(1), out["data"].(map[string]any)["total"])
	require.Equal(t, "a@b", repo.query.Keyword)
	require.Equal(t, service.ImageToolUpscale, repo.query.Tool)
	require.Equal(t, 2, repo.query.Page)
	require.NotNil(t, repo.query.From)
	require.Equal(t, 24*time.Hour, repo.query.To.Sub(*repo.query.From), "the end date is inclusive")
}
