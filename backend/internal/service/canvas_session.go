package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Signing in to the canvas (canvas.<domain>) with a HiveGPT account.
//
// The main site's connect popup (same origin as the API) creates a session and sets its token as
// an HttpOnly cookie scoped to /api/v1/canvas; the canvas calls those endpoints with credentials.
// The cookie is SameSite=Lax: the canvas is same-site with the main site and gets it, the hosted
// sites (another registrable domain) never do. Sessions can only reach the canvas endpoints.

const (
	CanvasSessionCookie = "hg_canvas_session"
	CanvasSessionPath   = "/api/v1/canvas"
	CanvasSessionTTL    = 30 * 24 * time.Hour
	// last_used_at / expires_at are refreshed at most this often.
	canvasSessionTouchEvery = 10 * time.Minute
	canvasSessionsPerUser   = 20
)

var ErrCanvasSessionInvalid = infraerrors.Unauthorized("CANVAS_SESSION_INVALID", "登录已失效，请重新用 HiveGPT 账号登录（Please sign in again）")

type CanvasSession struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"-"`
	UserAgent  string     `json:"user_agent"`
	IP         string     `json:"ip"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt time.Time  `json:"last_used_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"-"`
	// Current: the session making this request (listing from the canvas).
	Current bool `json:"current,omitempty"`
}

type CanvasSessionRepository interface {
	CreateCanvasSession(ctx context.Context, s *CanvasSession, tokenHash string) error
	GetCanvasSessionByHash(ctx context.Context, tokenHash string) (*CanvasSession, error)
	TouchCanvasSession(ctx context.Context, id int64, at, expires time.Time) error
	RevokeCanvasSession(ctx context.Context, userID, id int64, at time.Time) (bool, error)
	ListCanvasSessions(ctx context.Context, userID int64, now time.Time) ([]CanvasSession, error)
	// TrimCanvasSessions revokes all but the newest `keep` live sessions of a user.
	TrimCanvasSessions(ctx context.Context, userID int64, keep int, at time.Time) error
}

type canvasUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type canvasSubscriptionReader interface {
	ListActiveByUserID(ctx context.Context, userID int64) ([]UserSubscription, error)
}

type CanvasSessionService struct {
	repo  CanvasSessionRepository
	users canvasUserReader
	subs  canvasSubscriptionReader
	now   func() time.Time
}

func NewCanvasSessionService(repo CanvasSessionRepository, users canvasUserReader, subs canvasSubscriptionReader) *CanvasSessionService {
	return &CanvasSessionService{repo: repo, users: users, subs: subs, now: time.Now}
}

func hashCanvasToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Create starts a session for the user and returns its token (to be set as the cookie).
func (s *CanvasSessionService) Create(ctx context.Context, userID int64, userAgent, ip string) (string, *CanvasSession, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	now := s.now()
	session := &CanvasSession{UserID: userID, UserAgent: truncateRunes(strings.TrimSpace(userAgent), 255), IP: truncateRunes(ip, 64), CreatedAt: now, LastUsedAt: now, ExpiresAt: now.Add(CanvasSessionTTL)}
	if err := s.repo.CreateCanvasSession(ctx, session, hashCanvasToken(token)); err != nil {
		return "", nil, err
	}
	if err := s.repo.TrimCanvasSessions(ctx, userID, canvasSessionsPerUser, now); err != nil {
		return "", nil, err
	}
	return token, session, nil
}

// Authenticate resolves a cookie token to a live session of an active user, sliding its expiry.
func (s *CanvasSessionService) Authenticate(ctx context.Context, token string) (*CanvasSession, *User, error) {
	if token == "" || len(token) > 128 {
		return nil, nil, ErrCanvasSessionInvalid
	}
	session, err := s.repo.GetCanvasSessionByHash(ctx, hashCanvasToken(token))
	if err != nil {
		return nil, nil, err
	}
	now := s.now()
	if session == nil || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return nil, nil, ErrCanvasSessionInvalid
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil || user == nil || !user.IsActive() {
		return nil, nil, ErrCanvasSessionInvalid
	}
	if now.Sub(session.LastUsedAt) >= canvasSessionTouchEvery {
		session.LastUsedAt, session.ExpiresAt = now, now.Add(CanvasSessionTTL)
		if err := s.repo.TouchCanvasSession(ctx, session.ID, session.LastUsedAt, session.ExpiresAt); err != nil {
			return nil, nil, err
		}
	}
	return session, user, nil
}

// Revoke signs a session out (true when it was live).
func (s *CanvasSessionService) Revoke(ctx context.Context, userID, sessionID int64) (bool, error) {
	return s.repo.RevokeCanvasSession(ctx, userID, sessionID, s.now())
}

func (s *CanvasSessionService) List(ctx context.Context, userID int64) ([]CanvasSession, error) {
	sessions, err := s.repo.ListCanvasSessions(ctx, userID, s.now())
	if sessions == nil {
		sessions = []CanvasSession{}
	}
	return sessions, err
}

// CanvasMe is what the canvas shows in its account menu.
type CanvasMe struct {
	UserID        int64                `json:"user_id"`
	Username      string               `json:"username"`
	Email         string               `json:"email"`
	AvatarURL     string               `json:"avatar_url"`
	Balance       float64              `json:"balance"`
	Subscriptions []CanvasSubscription `json:"subscriptions"`
	// AffCode: the user's invite code, when invites are enabled (set by the handler).
	AffCode string `json:"aff_code,omitempty"`
}

type CanvasSubscription struct {
	GroupName string    `json:"group_name"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *CanvasSessionService) Me(ctx context.Context, user *User) (*CanvasMe, error) {
	me := &CanvasMe{UserID: user.ID, Username: user.Username, Email: maskCanvasEmail(user.Email), AvatarURL: user.AvatarURL, Balance: user.Balance, Subscriptions: []CanvasSubscription{}}
	subs, err := s.subs.ListActiveByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	for _, sub := range subs {
		if !sub.IsActive() {
			continue
		}
		name := ""
		if sub.Group != nil {
			name = sub.Group.Name
		}
		me.Subscriptions = append(me.Subscriptions, CanvasSubscription{GroupName: name, ExpiresAt: sub.ExpiresAt})
	}
	return me, nil
}

// maskCanvasEmail keeps enough of the address to recognise it: "ab***@qq.com".
func maskCanvasEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok {
		return ""
	}
	runes := []rune(local)
	if len(runes) > 2 {
		runes = runes[:2]
	}
	return string(runes) + "***@" + domain
}
