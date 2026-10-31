//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type memDesktopStore struct{ m map[string]string }

func (s *memDesktopStore) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := s.m[key]
	return v, ok, nil
}
func (s *memDesktopStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	s.m[key] = value
	return nil
}
func (s *memDesktopStore) SetNX(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	if _, ok := s.m[key]; ok {
		return false, nil
	}
	s.m[key] = value
	return true, nil
}
func (s *memDesktopStore) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(s.m, k)
	}
	return nil
}

type fakeDesktopKeys struct {
	keys    []APIKey
	created int
	updated []UpdateAPIKeyRequest
}

func (f *fakeDesktopKeys) List(_ context.Context, userID int64, _ pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	var out []APIKey
	for _, k := range f.keys {
		if k.UserID == userID && strings.Contains(k.Name, filters.Search) {
			out = append(out, k)
		}
	}
	return out, nil, nil
}
func (f *fakeDesktopKeys) Create(_ context.Context, userID int64, req CreateAPIKeyRequest) (*APIKey, error) {
	f.created++
	k := APIKey{ID: int64(100 + len(f.keys)), UserID: userID, Key: "sk-new", Name: req.Name, GroupID: req.GroupID, Status: StatusAPIKeyActive}
	f.keys = append(f.keys, k)
	return &k, nil
}
func (f *fakeDesktopKeys) Update(_ context.Context, id, userID int64, req UpdateAPIKeyRequest) (*APIKey, error) {
	f.updated = append(f.updated, req)
	for i := range f.keys {
		if f.keys[i].ID == id && f.keys[i].UserID == userID {
			if req.GroupID != nil {
				f.keys[i].GroupID = req.GroupID
			}
			if req.Status != nil {
				f.keys[i].Status = *req.Status
			}
			if req.Name != nil {
				f.keys[i].Name = *req.Name
			}
			k := f.keys[i]
			return &k, nil
		}
	}
	return nil, ErrDesktopLoginNotFound
}

type fakeDesktopSubs struct{ subs []UserSubscription }

func (f fakeDesktopSubs) ListActiveUserSubscriptions(context.Context, int64) ([]UserSubscription, error) {
	return f.subs, nil
}

type fakeDesktopUsers struct{}

func (fakeDesktopUsers) GetByID(_ context.Context, id int64) (*User, error) {
	return &User{ID: id, Username: "微信用户_ab12"}, nil
}

type fakeDesktopRouting struct{ group *Group }

func (f fakeDesktopRouting) DefaultPayAsYouGoGroup(context.Context, *User, string) *Group {
	return f.group
}

func newDesktopTestService(subs []UserSubscription, keys *fakeDesktopKeys) (*DesktopLoginService, *memDesktopStore) {
	store := &memDesktopStore{m: map[string]string{}}
	return &DesktopLoginService{
		store:   store,
		keys:    keys,
		subs:    fakeDesktopSubs{subs: subs},
		users:   fakeDesktopUsers{},
		routing: fakeDesktopRouting{group: &Group{ID: 5, Name: "GPT-按量", Platform: PlatformOpenAI}},
		now:     time.Now,
	}, store
}

const testDeviceID = "a1b2c3d4-e5f6-7890-abcd-ef0123456789"

func TestDesktopLogin_ApproveIssuesKeyInLongestOpenAISubscription(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	subs := []UserSubscription{
		{GroupID: 8, ExpiresAt: now.Add(24 * time.Hour), Group: &Group{ID: 8, Name: "Codex-轻量版", Platform: PlatformOpenAI}},
		{GroupID: 2, ExpiresAt: now.Add(30 * 24 * time.Hour), Group: &Group{ID: 2, Name: "Codex-Pro-白领办公版", Platform: PlatformOpenAI}},
		{GroupID: 9, ExpiresAt: now.Add(90 * 24 * time.Hour), Group: &Group{ID: 9, Name: "Claude", Platform: PlatformAnthropic}},
	}
	keys := &fakeDesktopKeys{}
	svc, store := newDesktopTestService(subs, keys)

	started, err := svc.Start(ctx, testDeviceID, "  张三的\x00笔记本#1  ", testComputerIP)
	require.NoError(t, err)
	require.Equal(t, "/desktop-login?t="+started.TicketID, started.LoginPath)
	require.Len(t, started.DeviceCode, 64)

	res, err := svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusPending, res.Status)

	_, err = svc.Poll(ctx, started.TicketID, strings.Repeat("0", 64))
	require.ErrorIs(t, err, ErrDesktopLoginBadCode)

	info, err := svc.Info(ctx, started.TicketID, testPhoneIP)
	require.NoError(t, err)
	require.Equal(t, "张三的笔记本1", info.DeviceName)

	status, err := svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, false)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusApproved, status)
	_, err = svc.Decide(ctx, started.TicketID, 43, true, testPhoneIP, false)
	require.ErrorIs(t, err, ErrDesktopLoginNotPending)

	res, err = svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusDone, res.Status)
	require.Equal(t, "sk-new", res.APIKey)
	require.Equal(t, "Codex-Pro-白领办公版", res.GroupName)
	require.Equal(t, "桌面助手 · 张三的笔记本1 #a1b2c3d4", res.KeyName)
	require.Equal(t, "微信用户_ab12", res.UserName)
	require.Equal(t, int64(2), *keys.keys[0].GroupID)
	require.Equal(t, int64(42), keys.keys[0].UserID)

	// The ticket is gone after the key is handed out; the claim marker is left to expire.
	res, err = svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusExpired, res.Status)
	_, ok := store.m[desktopLoginKeyPrefix+started.TicketID]
	require.False(t, ok)
}

func TestDesktopLogin_ReusesDeviceKeyAndMovesIt(t *testing.T) {
	ctx := context.Background()
	old := int64(5)
	keys := &fakeDesktopKeys{keys: []APIKey{
		{ID: 7, UserID: 42, Key: "sk-old", Name: "桌面助手 · 旧名字 #a1b2c3d4", GroupID: &old, Status: "inactive"},
		{ID: 8, UserID: 42, Key: "sk-other", Name: "我的第一个 Key", Status: StatusAPIKeyActive},
	}}
	subs := []UserSubscription{{GroupID: 8, ExpiresAt: time.Now().Add(time.Hour), Group: &Group{ID: 8, Name: "Codex-轻量版", Platform: PlatformOpenAI}}}
	svc, _ := newDesktopTestService(subs, keys)

	started, err := svc.Start(ctx, testDeviceID, "MacBook", testComputerIP)
	require.NoError(t, err)
	_, err = svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, false)
	require.NoError(t, err)
	res, err := svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, "sk-old", res.APIKey)
	require.Equal(t, 0, keys.created)
	require.Len(t, keys.updated, 1)
	require.Equal(t, int64(8), *keys.keys[0].GroupID)
	require.Equal(t, StatusAPIKeyActive, keys.keys[0].Status)
	require.Equal(t, "桌面助手 · MacBook #a1b2c3d4", keys.keys[0].Name)
}

func TestDesktopLogin_NoSubscriptionUsesPayAsYouGo(t *testing.T) {
	ctx := context.Background()
	keys := &fakeDesktopKeys{}
	svc, _ := newDesktopTestService(nil, keys)
	started, err := svc.Start(ctx, testDeviceID, "", testComputerIP)
	require.NoError(t, err)
	_, err = svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, false)
	require.NoError(t, err)
	res, err := svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, "GPT-按量", res.GroupName)
	require.Equal(t, "桌面助手 · 我的电脑 #a1b2c3d4", res.KeyName)
}

func TestDesktopLogin_RejectAndBadInput(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDesktopTestService(nil, &fakeDesktopKeys{})

	_, err := svc.Start(ctx, "bad id!", "x", testComputerIP)
	require.ErrorIs(t, err, ErrDesktopLoginBadDevice)
	_, err = svc.Info(ctx, "not-a-ticket", testPhoneIP)
	require.ErrorIs(t, err, ErrDesktopLoginNotFound)

	started, err := svc.Start(ctx, testDeviceID, "PC", testComputerIP)
	require.NoError(t, err)
	status, err := svc.Decide(ctx, started.TicketID, 42, false, testPhoneIP, false)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusRejected, status)
	res, err := svc.Poll(ctx, started.TicketID, started.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusRejected, res.Status)
}

func TestDesktopLogin_ExpiredTicketCannotBeSaved(t *testing.T) {
	ctx := context.Background()
	svc, _ := newDesktopTestService(nil, &fakeDesktopKeys{})
	started, err := svc.Start(ctx, testDeviceID, "PC", testComputerIP)
	require.NoError(t, err)
	svc.now = func() time.Time { return time.Now().Add(desktopLoginTicketTTL + time.Second) }
	_, err = svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, false)
	require.ErrorIs(t, err, ErrDesktopLoginNotFound)
}

// The computer and the phone on the same home network (one /24), as in a normal sign-in.
const (
	testComputerIP = "198.51.100.7"
	testPhoneIP    = "198.51.100.23"
)

func TestDesktopLogin_OtherNetworkNeedsConfirmNearby(t *testing.T) {
	svc, _ := newDesktopTestService(nil, &fakeDesktopKeys{})
	ctx := context.Background()
	// A phishing QR: the computer asks from somewhere else entirely.
	started, err := svc.Start(ctx, testDeviceID, "HiveGPT 官方验证", "203.0.113.50")
	require.NoError(t, err)

	info, err := svc.Info(ctx, started.TicketID, testPhoneIP)
	require.NoError(t, err)
	require.False(t, info.SameNetwork)
	require.Equal(t, "203.0.113.*", info.ComputerNetwork)
	require.Equal(t, "未命名电脑", info.DeviceName, "names posing as HiveGPT / an official check are replaced")

	_, err = svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, false)
	require.ErrorIs(t, err, ErrDesktopLoginNotNearby)
	// Rejecting never needs the confirmation.
	status, err := svc.Decide(ctx, started.TicketID, 42, false, testPhoneIP, false)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusRejected, status)

	// Phone on mobile data, computer on Wi-Fi: allowed once the user confirms it is in front of them.
	started, err = svc.Start(ctx, testDeviceID, "MacBook", "203.0.113.50")
	require.NoError(t, err)
	status, err = svc.Decide(ctx, started.TicketID, 42, true, testPhoneIP, true)
	require.NoError(t, err)
	require.Equal(t, DesktopLoginStatusApproved, status)
}

func TestDesktopNetworkHelpers(t *testing.T) {
	require.True(t, desktopSameNetwork("198.51.100.7", "198.51.100.200"))
	require.False(t, desktopSameNetwork("198.51.100.7", "198.51.101.7"))
	require.False(t, desktopSameNetwork("", ""), "unknown addresses never count as the same network")
	require.True(t, desktopSameNetwork("2001:db8:1:2::1", "2001:db8:1:ffff::9"))
	require.Equal(t, "2001:db8:1::*", desktopMaskedNetwork("2001:db8:1:2::1"))
	require.Equal(t, "", desktopMaskedNetwork("not an ip"))
	require.Equal(t, "MacBook Pro", cleanDeviceName("MacBook Pro"))
	require.Equal(t, "未命名电脑", cleanDeviceName("hivegpt客服"))
}
