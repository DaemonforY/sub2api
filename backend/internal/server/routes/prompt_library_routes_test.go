package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// memPromptRepo is the part of PromptLibraryRepository the user routes touch.
type memPromptRepo struct {
	service.PromptLibraryRepository
	covers map[string]int64
	items  []service.PromptItem
}

func (r *memPromptRepo) InsertCover(_ context.Context, file string, userID, _ int64) error {
	r.covers[file] = userID
	return nil
}
func (r *memPromptRepo) CountCoversSince(context.Context, int64, time.Time) (int, error) {
	return 0, nil
}
func (r *memPromptRepo) CoverOwnedBy(_ context.Context, file string, userID int64) (bool, error) {
	return r.covers[file] == userID, nil
}
func (r *memPromptRepo) CountOwned(context.Context, int64) (int, error) { return len(r.items), nil }
func (r *memPromptRepo) Insert(_ context.Context, item *service.PromptItem) error {
	item.ID = int64(len(r.items) + 1)
	r.items = append(r.items, *item)
	return nil
}
func (r *memPromptRepo) List(_ context.Context, q service.PromptListQuery) ([]service.PromptItem, int64, error) {
	var out []service.PromptItem
	for _, item := range r.items {
		if q.OwnerUserID > 0 && (item.OwnerUserID == nil || *item.OwnerUserID != q.OwnerUserID) {
			continue
		}
		if q.OwnerUserID == 0 && (item.Status != service.PromptStatusActive || item.Visibility != service.PromptVisibilityPublic) {
			continue
		}
		out = append(out, item)
	}
	return out, int64(len(out)), nil
}
func (r *memPromptRepo) SceneCounts(context.Context, service.PromptListQuery) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (r *memPromptRepo) ListSources(context.Context) ([]service.PromptSource, error) { return nil, nil }

func TestPromptLibraryRoutes_UserPromptsWithCovers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &memPromptRepo{covers: map[string]int64{}}
	svc := service.NewPromptLibraryService(repo, service.NewPromptCoverStore(t.TempDir()))
	h := &handler.Handlers{PromptLibrary: handler.NewPromptLibraryHandler(svc)}
	apiKeyAuth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		switch c.GetHeader("Authorization") {
		case "Bearer sk-owner":
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		case "Bearer sk-other":
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 8})
		default:
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	r := gin.New()
	// Public GET /items next to keyed POST /items/:id/use and GET /covers/:file next to POST /covers must not panic.
	RegisterPromptLibraryRoutes(r.Group("/api/v1"), h, apiKeyAuth, nil, middleware.NewPanelRateLimiter(nil, nil))

	do := func(method, path, key string, body *bytes.Buffer, contentType string) *httptest.ResponseRecorder {
		if body == nil {
			body = &bytes.Buffer{}
		}
		req := httptest.NewRequest(method, path, body)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if key != "" {
			req.Header.Set("Authorization", key)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	var img bytes.Buffer
	require.NoError(t, png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	var form bytes.Buffer
	mw := multipart.NewWriter(&form)
	part, _ := mw.CreateFormFile("file", "cover.png")
	_, _ = part.Write(img.Bytes())
	require.NoError(t, mw.Close())
	require.Equal(t, http.StatusUnauthorized, do(http.MethodPost, "/api/v1/prompt-library/covers", "", bytes.NewBuffer(form.Bytes()), mw.FormDataContentType()).Code)
	w := do(http.MethodPost, "/api/v1/prompt-library/covers", "Bearer sk-owner", &form, mw.FormDataContentType())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var uploaded struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &uploaded))
	require.True(t, strings.HasPrefix(uploaded.Data.URL, service.PromptCoverPublicPrefix))

	// Covers are public files (anyone viewing a shared prompt sees them).
	cover := do(http.MethodGet, uploaded.Data.URL, "", nil, "")
	require.Equal(t, http.StatusOK, cover.Code)
	require.Equal(t, img.Bytes(), cover.Body.Bytes())
	require.Equal(t, http.StatusNotFound, do(http.MethodGet, service.PromptCoverPublicPrefix+"..%2f..%2fetc%2fpasswd", "", nil, "").Code)

	create := func(key, cover string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"prompt": "一张国风山水海报", "cover_url": cover, "share": true, "tags": []string{"#国风", "山水"}})
		return do(http.MethodPost, "/api/v1/prompt-library/mine", key, bytes.NewBuffer(body), "application/json")
	}
	// Someone else's cover cannot be attached.
	require.Equal(t, http.StatusBadRequest, create("Bearer sk-other", uploaded.Data.URL).Code)
	w = create("Bearer sk-owner", uploaded.Data.URL)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created struct {
		Data service.PromptItem `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, service.PromptStatusPending, created.Data.Status, "shared prompts wait for review")
	require.Equal(t, []string{"国风", "山水"}, created.Data.Tags)
	require.True(t, created.Data.Mine)
	require.Nil(t, created.Data.OwnerUserID)
	require.NotEmpty(t, created.Data.Title, "a title is derived from the prompt")
	require.Equal(t, []string{"poster", "history"}, created.Data.Scenes, "scenes come from the tags, the title and the prompt")

	// Pending prompts are not public yet, but the owner sees them.
	public := do(http.MethodGet, "/api/v1/prompt-library/items", "", nil, "")
	require.Equal(t, http.StatusOK, public.Code)
	require.NotContains(t, public.Body.String(), "国风山水")
	require.Equal(t, http.StatusUnauthorized, do(http.MethodGet, "/api/v1/prompt-library/mine", "", nil, "").Code)
	mine := do(http.MethodGet, "/api/v1/prompt-library/mine", "Bearer sk-owner", nil, "")
	require.Equal(t, http.StatusOK, mine.Code)
	require.Contains(t, mine.Body.String(), "国风山水")
	require.NotContains(t, do(http.MethodGet, "/api/v1/prompt-library/mine", "Bearer sk-other", nil, "").Body.String(), "国风山水")
}
