package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// HiveGPT 视频 settings (admin, on the video site's /review page): the models offered on the create
// page — each runs through its own group, so the video site has its own pricing — and the ad cards
// shown in the gallery. Stored as one JSON value in the settings table.

const videoSettingsKey = "video_settings"

type VideoModelOption struct {
	ID      string `json:"id"`             // model name sent to the gateway
	Name    string `json:"name"`           // shown on the create page
	Note    string `json:"note,omitempty"` // short hint, e.g. 「画面最精致」
	GroupID int64  `json:"group_id"`
	Default bool   `json:"default,omitempty"`
}

type VideoAd struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Text    string `json:"text,omitempty"`
	Image   string `json:"image"` // /api/v1/video/ads/<file> or an https URL
	Link    string `json:"link"`
	Enabled bool   `json:"enabled"`
}

type VideoSettings struct {
	Models  []VideoModelOption `json:"models"`
	Ads     []VideoAd          `json:"ads"`
	AdEvery int                `json:"ad_every"` // one ad after every N works in the gallery
}

type VideoPublicModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Note    string `json:"note,omitempty"`
	Default bool   `json:"default,omitempty"`
}

var (
	ErrVideoSettings = infraerrors.BadRequest("VIDEO_SETTINGS", "设置不完整：每个模型都要填模型名、显示名并选择分组，广告要有图片和链接（Invalid settings）")
	ErrVideoNoModel  = infraerrors.BadRequest("VIDEO_NO_MODEL", "还没有可用的模型，请联系管理员（No model is configured）")
)

type videoSettingsCache struct {
	mu  sync.Mutex
	val *VideoSettings
	at  time.Time
}

func (s *VideoService) Settings(ctx context.Context) VideoSettings {
	s.settingsCache.mu.Lock()
	defer s.settingsCache.mu.Unlock()
	if s.settingsCache.val != nil && time.Since(s.settingsCache.at) < 30*time.Second {
		return *s.settingsCache.val
	}
	out := VideoSettings{AdEvery: 8}
	if s.settings != nil {
		raw, err := s.settings.GetValue(ctx, videoSettingsKey)
		switch {
		case err == nil && raw != "":
			_ = json.Unmarshal([]byte(raw), &out)
		case err != nil && !errors.Is(err, ErrSettingNotFound):
			return out // keep trying on the next call
		}
	}
	s.settingsCache.val, s.settingsCache.at = &out, time.Now()
	return out
}

func (s *VideoService) AdminSettings(ctx context.Context, admin *User) (VideoSettings, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return VideoSettings{}, ErrVideoForbidden
	}
	return s.Settings(ctx), nil
}

func (s *VideoService) SaveSettings(ctx context.Context, admin *User, in VideoSettings) (VideoSettings, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return VideoSettings{}, ErrVideoForbidden
	}
	if s.settings == nil || len(in.Models) > 20 || len(in.Ads) > 20 {
		return VideoSettings{}, ErrVideoSettings
	}
	seen := map[string]bool{}
	hasDefault := false
	for i := range in.Models {
		m := &in.Models[i]
		m.ID, m.Name, m.Note = strings.TrimSpace(m.ID), clipRunes(strings.TrimSpace(m.Name), 30), clipRunes(strings.TrimSpace(m.Note), 30)
		if m.ID == "" || len(m.ID) > 100 || m.Name == "" || m.GroupID <= 0 || seen[m.ID] {
			return VideoSettings{}, ErrVideoSettings
		}
		seen[m.ID] = true
		if s.groups != nil {
			g, err := s.groups.GetByIDLite(ctx, m.GroupID)
			if err != nil || g == nil || !g.IsActive() || g.IsSubscriptionType() {
				return VideoSettings{}, infraerrors.BadRequest("VIDEO_SETTINGS_GROUP", fmt.Sprintf("模型 %s 选的分组不存在、已停用或是订阅分组（Group unavailable）", m.ID))
			}
		}
		if m.Default && hasDefault {
			m.Default = false
		}
		hasDefault = hasDefault || m.Default
	}
	if !hasDefault && len(in.Models) > 0 {
		in.Models[0].Default = true
	}
	for i := range in.Ads {
		a := &in.Ads[i]
		a.Title, a.Text = clipRunes(strings.TrimSpace(a.Title), 40), clipRunes(strings.TrimSpace(a.Text), 80)
		a.Image, a.Link = strings.TrimSpace(a.Image), strings.TrimSpace(a.Link)
		if a.ID == "" {
			a.ID = randomVideoID(6)
		}
		if !videoAdImageOK(a.Image) || !videoHTTPURL(a.Link) {
			return VideoSettings{}, ErrVideoSettings
		}
	}
	in.AdEvery = max(3, min(in.AdEvery, 30))
	raw, err := json.Marshal(in)
	if err != nil {
		return VideoSettings{}, err
	}
	if err := s.settings.Set(ctx, videoSettingsKey, string(raw)); err != nil {
		return VideoSettings{}, err
	}
	s.settingsCache.mu.Lock()
	s.settingsCache.val = nil
	s.settingsCache.mu.Unlock()
	return in, nil
}

// AdminGroups lists the active balance groups an admin can give a video model.
func (s *VideoService) AdminGroups(ctx context.Context, admin *User) ([]map[string]any, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return nil, ErrVideoForbidden
	}
	if s.groups == nil {
		return []map[string]any{}, nil
	}
	groups, err := s.groups.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		if g.IsSubscriptionType() {
			continue
		}
		out = append(out, map[string]any{"id": g.ID, "name": g.Name, "platform": g.Platform, "rate_multiplier": g.RateMultiplier, "exclusive": g.IsExclusive})
	}
	return out, nil
}

// Models is the create page's model list (without groups).
func (s *VideoService) Models(ctx context.Context) []VideoPublicModel {
	st := s.Settings(ctx)
	out := make([]VideoPublicModel, 0, len(st.Models))
	for _, m := range st.Models {
		out = append(out, VideoPublicModel{ID: m.ID, Name: m.Name, Note: m.Note, Default: m.Default})
	}
	return out
}

// Ads returns the enabled gallery ads and how many works go between two ads.
func (s *VideoService) Ads(ctx context.Context) ([]VideoAd, int) {
	st := s.Settings(ctx)
	out := []VideoAd{}
	for _, a := range st.Ads {
		if a.Enabled {
			out = append(out, a)
		}
	}
	return out, max(3, st.AdEvery)
}

// videoModel returns the configured model for id, falling back to the default one; ok is false
// when no models are configured (runs then use the user's own key and model, as before).
func (s *VideoService) videoModel(ctx context.Context, id string) (VideoModelOption, bool) {
	st := s.Settings(ctx)
	if len(st.Models) == 0 {
		return VideoModelOption{}, false
	}
	fallback := st.Models[0]
	for _, m := range st.Models {
		if m.ID == id {
			return m, true
		}
		if m.Default {
			fallback = m
		}
	}
	return fallback, true
}

func videoHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && len(raw) <= 500
}

func videoAdImageOK(raw string) bool {
	if name, ok := strings.CutPrefix(raw, "/api/v1/video/ads/"); ok {
		return videoAdFileName.MatchString(name)
	}
	return strings.HasPrefix(raw, "https://") && videoHTTPURL(raw)
}

func randomVideoID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// --- the key a run uses -------------------------------------------------------------------------

// A configured model runs through its own group with a per-user key 「HiveGPT 视频」, created on
// first use; the group is added to the user's allowed groups so the key passes the normal checks
// and the calls are billed to the user's balance at that group's rate.

// VideoKeyStore remembers each user's video key per group.
type VideoKeyStore interface {
	GetUserKey(ctx context.Context, userID, groupID int64) (int64, error)
	SetUserKey(ctx context.Context, userID, groupID, apiKeyID int64) error
}

// VideoKeyMaker reads and creates API keys (an APIKeyRepository subset).
type VideoKeyMaker interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
	Create(ctx context.Context, key *APIKey) error
}

// VideoGroupGrant adds a group to a user's allowed groups (idempotent).
type VideoGroupGrant interface {
	AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error
}

const videoSystemKeyName = "HiveGPT 视频"

// groupKey returns the user's key in groupID, creating it on first use.
func (s *VideoService) groupKey(ctx context.Context, userID, groupID int64) (string, error) {
	if s.keyStore == nil || s.keyMaker == nil || s.grant == nil || s.newKey == nil {
		return "", ErrVideoNoModel
	}
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	if id, err := s.keyStore.GetUserKey(ctx, userID, groupID); err == nil && id > 0 {
		k, err := s.keyMaker.GetByID(ctx, id)
		if err == nil && k != nil && k.UserID == userID && k.GroupID != nil && *k.GroupID == groupID && k.IsActive() {
			return k.Key, nil
		}
	}
	if err := s.grant.AddGroupToAllowedGroups(ctx, userID, groupID); err != nil {
		return "", err
	}
	secret, err := s.newKey()
	if err != nil {
		return "", err
	}
	gid := groupID
	k := &APIKey{UserID: userID, Key: secret, Name: videoSystemKeyName, GroupID: &gid, Status: StatusActive}
	if err := s.keyMaker.Create(ctx, k); err != nil {
		return "", err
	}
	if err := s.keyStore.SetUserKey(ctx, userID, groupID, k.ID); err != nil {
		return "", err
	}
	return k.Key, nil
}

// runKey picks the key and model a run uses: the video group key of the chosen model when models
// are configured, otherwise the user's own key.
func (s *VideoService) runKey(ctx context.Context, p *VideoProject, own string) (string, string, error) {
	m, ok := s.videoModel(ctx, p.Options.Model)
	if !ok {
		return own, p.Options.Model, nil
	}
	key, err := s.groupKey(ctx, p.UserID, m.GroupID)
	if err != nil {
		return "", "", err
	}
	return key, m.ID, nil
}
