//go:build unit

package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeVideoSettingRepo struct {
	mu sync.Mutex
	m  map[string]string
}

func (f *fakeVideoSettingRepo) Get(context.Context, string) (*Setting, error) { return nil, ErrSettingNotFound }
func (f *fakeVideoSettingRepo) GetValue(_ context.Context, k string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.m[k]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}
func (f *fakeVideoSettingRepo) Set(_ context.Context, k, v string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[k] = v
	return nil
}
func (f *fakeVideoSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (f *fakeVideoSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (f *fakeVideoSettingRepo) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (f *fakeVideoSettingRepo) Delete(context.Context, string) error                 { return nil }

type fakeVideoKeys struct {
	mu      sync.Mutex
	store   map[[2]int64]int64
	keys    map[int64]*APIKey
	granted [][2]int64
	n       int
}

func (f *fakeVideoKeys) GetUserKey(_ context.Context, u, g int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.store[[2]int64{u, g}], nil
}
func (f *fakeVideoKeys) SetUserKey(_ context.Context, u, g, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[[2]int64{u, g}] = id
	return nil
}
func (f *fakeVideoKeys) GetByID(_ context.Context, id int64) (*APIKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.keys[id], nil
}
func (f *fakeVideoKeys) Create(_ context.Context, k *APIKey) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	k.ID = int64(100 + f.n)
	f.keys[k.ID] = k
	return nil
}
func (f *fakeVideoKeys) AddGroupToAllowedGroups(_ context.Context, u, g int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.granted = append(f.granted, [2]int64{u, g})
	return nil
}

func TestVideoConfiguredModelRunsThroughItsGroupKey(t *testing.T) {
	var mu sync.Mutex
	var seen [][2]string
	model := &fakeModel{broken: map[string]int{}}
	stream := func(ctx context.Context, key, m string, msgs []LearnMessage, n int, onDelta func(string) error) (*LearnTutorResult, error) {
		mu.Lock()
		seen = append(seen, [2]string{key, m})
		mu.Unlock()
		return model.stream(ctx, key, m, msgs, n, onDelta)
	}
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, stream, t.TempDir())
	settings := &fakeVideoSettingRepo{m: map[string]string{}}
	keys := &fakeVideoKeys{store: map[[2]int64]int64{}, keys: map[int64]*APIKey{}}
	n := 0
	svc.SetModelBilling(settings, nil, keys, keys, keys, func() (string, error) { n++; return "sk-video-" + string(rune('a'+n)), nil })
	ctx := context.Background()
	admin := &User{Role: RoleAdmin}

	_, err := svc.SaveSettings(ctx, &User{Role: "user"}, VideoSettings{})
	require.ErrorIs(t, err, ErrVideoForbidden)
	_, err = svc.SaveSettings(ctx, admin, VideoSettings{Models: []VideoModelOption{{ID: "gpt-5.5", Name: "GPT"}}})
	require.ErrorIs(t, err, ErrVideoSettings, "a model needs a group")
	_, err = svc.SaveSettings(ctx, admin, VideoSettings{Ads: []VideoAd{{Title: "x", Image: "javascript:alert(1)", Link: "https://a.cn"}}})
	require.ErrorIs(t, err, ErrVideoSettings, "ad images must be uploaded files or https")
	st, err := svc.SaveSettings(ctx, admin, VideoSettings{
		Models: []VideoModelOption{{ID: "gpt-5.5", Name: "GPT-5.5", GroupID: 11}, {ID: "claude-opus-5-5", Name: "Claude Opus 5.5", GroupID: 12, Default: true}},
		Ads:    []VideoAd{{Title: "主站", Image: "https://hivegpt.cn/a.png", Link: "https://hivegpt.cn", Enabled: true}, {Title: "off", Image: "https://hivegpt.cn/b.png", Link: "https://hivegpt.cn"}},
	})
	require.NoError(t, err)
	require.Equal(t, 3, st.AdEvery, "clamped")
	require.Len(t, svc.Models(ctx), 2)
	ads, _ := svc.Ads(ctx)
	require.Len(t, ads, 1, "only enabled ads")

	// An unknown model falls back to the default one; the run uses the user's video key.
	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "loader", Options: VideoOptions{Model: "gpt-4o"}})
	require.NoError(t, err)
	require.Equal(t, "claude-opus-5-5", p.Options.Model)
	svc.Wait()
	p2, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "loader 2", Options: VideoOptions{Model: "claude-opus-5-5"}})
	require.NoError(t, err)
	_ = p2
	svc.Wait()

	require.Equal(t, 1, n, "one key per user and group, reused")
	require.Equal(t, [][2]int64{{7, 12}}, keys.granted, "the group is allowed for the user before the key is made")
	k := keys.keys[101]
	require.Equal(t, int64(7), k.UserID)
	require.Equal(t, int64(12), *k.GroupID)
	require.Equal(t, videoSystemKeyName, k.Name)
	for _, c := range seen {
		require.Equal(t, [2]string{"sk-video-b", "claude-opus-5-5"}, c, "never the signed-in key")
	}
	require.NotEmpty(t, seen)
}

func TestVideoWithoutConfiguredModelsUsesOwnKey(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	var keysSeen []string
	stream := func(ctx context.Context, key, m string, msgs []LearnMessage, n int, onDelta func(string) error) (*LearnTutorResult, error) {
		keysSeen = append(keysSeen, key)
		return model.stream(ctx, key, m, msgs, n, onDelta)
	}
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, stream, t.TempDir())
	_, err := svc.Create(context.Background(), videoKeyA, VideoCreateInput{Mode: "motion", Prompt: "loader"})
	require.NoError(t, err)
	svc.Wait()
	require.Equal(t, []string{"sk-a"}, keysSeen)
}

func TestVideoAdImage(t *testing.T) {
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, nil, t.TempDir())
	png := []byte("\x89PNG\r\n\x1a\n0000000000000000")
	_, err := svc.SaveAdImage(context.Background(), &User{Role: "user"}, png)
	require.ErrorIs(t, err, ErrVideoForbidden)
	_, err = svc.SaveAdImage(context.Background(), &User{Role: RoleAdmin}, []byte("<svg onload=alert(1)>"))
	require.ErrorIs(t, err, ErrVideoAdImage)
	path, err := svc.SaveAdImage(context.Background(), &User{Role: RoleAdmin}, png)
	require.NoError(t, err)
	require.Regexp(t, `^/api/v1/video/ads/[0-9a-f]{24}\.png$`, path)
	_, err = svc.AdImagePath(path[len("/api/v1/video/ads/"):])
	require.NoError(t, err)
	_, err = svc.AdImagePath("../../etc/passwd")
	require.ErrorIs(t, err, ErrVideoNotFound)
}
