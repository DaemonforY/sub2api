package routes

import (
	"bytes"
	"context"
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

// notFoundContestRepo embeds the interface (nil) and only implements the call the
// submit path makes first, so reaching it proves authentication succeeded.
type notFoundContestRepo struct {
	service.ContestRepository
	gotID int64
}

func (r *notFoundContestRepo) GetContest(_ context.Context, id int64) (*service.Contest, error) {
	r.gotID = id
	return nil, service.ErrContestNotFound
}

func TestContestKeyEntriesRoute_UsesAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &notFoundContestRepo{}
	svc := service.NewContestService(repo, nil, nil, nil, nil, service.NewContestImageStore(t.TempDir()))
	h := &handler.Handlers{Contest: handler.NewContestHandler(svc)}

	var keyAuthCalls, jwtCalls int
	apiKeyAuth := middleware.APIKeyAuthMiddleware(func(c *gin.Context) {
		keyAuthCalls++
		if c.GetHeader("Authorization") != "Bearer sk-valid" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Next()
	})
	jwtAuth := middleware.JWTAuthMiddleware(func(c *gin.Context) { jwtCalls++; c.AbortWithStatus(http.StatusUnauthorized) })
	optionalJWT := middleware.OptionalJWTAuthMiddleware(func(c *gin.Context) { c.Next() })

	r := gin.New()
	RegisterContestRoutes(r.Group("/api/v1"), h, jwtAuth, optionalJWT, apiKeyAuth, nil, middleware.NewPanelRateLimiter(nil, nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/contests/7/key-entries", nil))
	require.Equal(t, http.StatusUnauthorized, w.Code, "no key -> rejected by key auth")

	var img bytes.Buffer
	require.NoError(t, png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	require.NoError(t, form.WriteField("title", "来自画布"))
	part, err := form.CreateFormFile("image", "a.png")
	require.NoError(t, err)
	_, _ = part.Write(img.Bytes())
	require.NoError(t, form.Close())

	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contests/7/key-entries", &body)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer sk-valid")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, "valid key -> passes auth and reaches the contest lookup")
	require.Equal(t, int64(7), repo.gotID, "handler reached the service")
	require.Equal(t, 2, keyAuthCalls)
	require.Zero(t, jwtCalls, "the key route never consults the web session")
}
