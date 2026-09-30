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
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// memBlobRepo is an in-memory UserAppBlobRepository for route tests.
type memBlobRepo struct {
	blobs map[string]service.UserAppBlob
}

func (r *memBlobRepo) FindBlobBySHA(_ context.Context, userID int64, sha string) (*service.UserAppBlob, error) {
	for _, b := range r.blobs {
		if b.UserID == userID && b.SHA256 == sha {
			return &b, nil
		}
	}
	return nil, nil
}
func (r *memBlobRepo) GetBlob(_ context.Context, userID int64, id string) (*service.UserAppBlob, error) {
	if b, ok := r.blobs[id]; ok && b.UserID == userID {
		return &b, nil
	}
	return nil, nil
}
func (r *memBlobRepo) InsertBlob(_ context.Context, b *service.UserAppBlob) error {
	r.blobs[b.ID] = *b
	return nil
}
func (r *memBlobRepo) TouchBlob(context.Context, int64, string) error { return nil }
func (r *memBlobRepo) BlobsOverQuota(context.Context, int64, int, int64) ([]service.UserAppBlob, error) {
	return nil, nil
}
func (r *memBlobRepo) DeleteBlobs(_ context.Context, _ int64, ids []string) error {
	for _, id := range ids {
		delete(r.blobs, id)
	}
	return nil
}

func TestAppStateBlobRoutes_UploadThenOwnerOnlyDownload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	blobs := service.NewUserAppBlobService(&memBlobRepo{blobs: map[string]service.UserAppBlob{}}, t.TempDir())
	h := &handler.Handlers{AppState: handler.NewUserAppStateHandler(service.NewUserAppStateService(nil), blobs)}
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
	// Registering static /blobs next to /:namespace must not panic.
	RegisterAppStateRoutes(r.Group("/api/v1"), h, apiKeyAuth, nil, middleware.NewPanelRateLimiter(nil, nil))

	var img bytes.Buffer
	require.NoError(t, png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	upload := func() string {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, _ := mw.CreateFormFile("file", "ref.png")
		_, _ = part.Write(img.Bytes())
		require.NoError(t, mw.Close())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/app-state/blobs", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer sk-owner")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var resp struct {
			Data service.UserAppBlob `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "image/png", resp.Data.MimeType)
		return resp.Data.ID
	}
	id := upload()
	require.Equal(t, id, upload(), "same bytes are deduplicated")

	get := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/app-state/blobs/"+id, nil)
		req.Header.Set("Authorization", key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	owner := get("Bearer sk-owner")
	require.Equal(t, http.StatusOK, owner.Code)
	require.Equal(t, img.Bytes(), owner.Body.Bytes())
	require.Equal(t, http.StatusNotFound, get("Bearer sk-other").Code)
}
