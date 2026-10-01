//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type memCanvasSessions struct {
	mu     sync.Mutex
	next   int64
	byHash map[string]*service.CanvasSession
}

func (m *memCanvasSessions) CreateCanvasSession(_ context.Context, s *service.CanvasSession, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	s.ID = m.next
	cp := *s
	m.byHash[hash] = &cp
	return nil
}

func (m *memCanvasSessions) GetCanvasSessionByHash(_ context.Context, hash string) (*service.CanvasSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.byHash[hash]; s != nil {
		cp := *s
		return &cp, nil
	}
	return nil, nil
}

func (m *memCanvasSessions) find(id int64) *service.CanvasSession {
	for _, s := range m.byHash {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (m *memCanvasSessions) TouchCanvasSession(_ context.Context, id int64, at, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.find(id); s != nil {
		s.LastUsedAt, s.ExpiresAt = at, expires
	}
	return nil
}

func (m *memCanvasSessions) RevokeCanvasSession(_ context.Context, userID, id int64, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.find(id); s != nil && s.UserID == userID && s.RevokedAt == nil {
		s.RevokedAt = &at
		return true, nil
	}
	return false, nil
}

func (m *memCanvasSessions) ListCanvasSessions(_ context.Context, userID int64, now time.Time) ([]service.CanvasSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []service.CanvasSession
	for _, s := range m.byHash {
		if s.UserID == userID && s.RevokedAt == nil && s.ExpiresAt.After(now) {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *memCanvasSessions) TrimCanvasSessions(context.Context, int64, int, time.Time) error {
	return nil
}

type canvasUsers struct{ users map[int64]*service.User }

func (u canvasUsers) GetByID(_ context.Context, id int64) (*service.User, error) {
	if user := u.users[id]; user != nil {
		return user, nil
	}
	return nil, service.ErrUserNotFound
}

type canvasSubs struct{}

func (canvasSubs) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	return []service.UserSubscription{{Status: service.SubscriptionStatusActive, ExpiresAt: time.Now().Add(48 * time.Hour), Group: &service.Group{Name: "畅享版"}}}, nil
}

type canvasSettings struct {
	service.SettingRepository
	values map[string]string
}

func (s canvasSettings) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", errors.New("setting not found")
}

type canvasAffiliates struct{ service.AffiliateRepository }

func (canvasAffiliates) EnsureUserAffiliate(_ context.Context, userID int64) (*service.AffiliateSummary, error) {
	return &service.AffiliateSummary{UserID: userID, AffCode: "ALICE2024"}, nil
}

const canvasOrigin = "https://canvas.example.test"

func newCanvasTestRouter(t *testing.T) (*gin.Engine, *memCanvasSessions, *service.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	user := &service.User{ID: 7, Email: "alice@example.com", Username: "alice", Balance: 12.5, Status: service.StatusActive, Role: "user"}
	repo := &memCanvasSessions{byHash: map[string]*service.CanvasSession{}}
	settings := service.NewSettingService(canvasSettings{values: map[string]string{service.SettingKeyAffiliateEnabled: "true"}}, &config.Config{})
	affiliates := service.NewAffiliateService(canvasAffiliates{}, settings, nil, nil)
	h := NewCanvasSessionHandler(service.NewCanvasSessionService(repo, canvasUsers{users: map[int64]*service.User{7: user}}, canvasSubs{}), affiliates)
	r := gin.New()
	r.Use(middleware.CORS(config.CORSConfig{AllowedOrigins: []string{canvasOrigin}}))
	// Stand-in for the panel JWT: the connect popup is signed in as user 7.
	r.POST("/api/v1/user/canvas-sessions", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})
		c.Next()
	}, h.Create)
	canvas := r.Group("/api/v1/canvas", middleware.CanvasOriginGuard([]string{canvasOrigin}))
	canvas.POST("/logout", h.Logout)
	canvas.GET("/me", middleware.CanvasSessionAuth(h.Sessions()), h.Me)
	return r, repo, user
}

func canvasRequest(method, path, cookie string, mutate func(*http.Request)) *http.Request {
	req := httptest.NewRequest(method, "https://hivegpt.example.test"+path, nil)
	req.Header.Set("Origin", canvasOrigin)
	req.Header.Set(middleware.CanvasRequestHeader, "1")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: service.CanvasSessionCookie, Value: cookie})
	}
	if mutate != nil {
		mutate(req)
	}
	return req
}

func TestCanvasSessionSignInMeAndLogout(t *testing.T) {
	r, repo, _ := newCanvasTestRouter(t)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "https://hivegpt.example.test/api/v1/user/canvas-sessions", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	setCookie := rec.Header().Get("Set-Cookie")
	require.Contains(t, setCookie, service.CanvasSessionCookie+"=")
	require.Contains(t, setCookie, "Path=/api/v1/canvas")
	require.Contains(t, setCookie, "HttpOnly")
	require.Contains(t, setCookie, "SameSite=Lax")
	token := strings.TrimPrefix(strings.Split(setCookie, ";")[0], service.CanvasSessionCookie+"=")
	require.NotEmpty(t, token)
	for hash := range repo.byHash {
		require.NotEqual(t, token, hash, "only the hash is stored")
	}

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/me", token, nil))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"), "credentials are allowed for the canvas paths")
	require.Equal(t, canvasOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
	var body struct {
		Data service.CanvasMe `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, int64(7), body.Data.UserID)
	require.Equal(t, "al***@example.com", body.Data.Email)
	require.InDelta(t, 12.5, body.Data.Balance, 1e-9)
	require.Len(t, body.Data.Subscriptions, 1)
	require.Equal(t, "畅享版", body.Data.Subscriptions[0].GroupName)
	require.Equal(t, "ALICE2024", body.Data.AffCode, "the canvas adds the invite code to shared links")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodPost, "/api/v1/canvas/logout", token, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Set-Cookie"), "Max-Age=0")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/me", token, nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code, "a signed-out session is rejected")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, canvasRequest(http.MethodPost, "/api/v1/canvas/logout", token, nil))
	require.Equal(t, http.StatusOK, rec.Code, "logging out twice still clears the cookie")
}

func TestCanvasSessionRejectsOtherOriginsAndMissingHeader(t *testing.T) {
	r, _, _ := newCanvasTestRouter(t)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "https://hivegpt.example.test/api/v1/user/canvas-sessions", nil))
	token := strings.TrimPrefix(strings.Split(rec.Header().Get("Set-Cookie"), ";")[0], service.CanvasSessionCookie+"=")

	for name, mutate := range map[string]func(*http.Request){
		"hosted site origin": func(req *http.Request) { req.Header.Set("Origin", "https://evil.s.xinduanju.top") },
		"no origin":          func(req *http.Request) { req.Header.Del("Origin") },
		"no canvas header":   func(req *http.Request) { req.Header.Del(middleware.CanvasRequestHeader) },
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, canvasRequest(http.MethodGet, "/api/v1/canvas/me", token, mutate))
		require.Equal(t, http.StatusForbidden, rec.Code, name)
		if name == "hosted site origin" {
			require.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"), "other origins never get credentials")
		}
	}

	// The preflight for the canvas origin allows the custom header and credentials.
	pre := httptest.NewRequest(http.MethodOptions, "https://hivegpt.example.test/api/v1/canvas/me", nil)
	pre.Header.Set("Origin", canvasOrigin)
	pre.Header.Set("Access-Control-Request-Headers", middleware.CanvasRequestHeader)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, pre)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), middleware.CanvasRequestHeader)

	// Elsewhere in the API credentials stay off.
	other := httptest.NewRequest(http.MethodOptions, "https://hivegpt.example.test/api/v1/user/profile", nil)
	other.Header.Set("Origin", canvasOrigin)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, other)
	require.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
}
