//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type channelLinkRepoStub struct {
	links  map[string]*ChannelLink
	clicks map[int64]int
	nextID int64
}

func newChannelLinkRepoStub() *channelLinkRepoStub {
	return &channelLinkRepoStub{links: map[string]*ChannelLink{}, clicks: map[int64]int{}}
}

func (r *channelLinkRepoStub) List(context.Context) ([]ChannelLink, error) { return nil, nil }
func (r *channelLinkRepoStub) GetByCode(_ context.Context, code string) (*ChannelLink, error) {
	return r.links[code], nil
}
func (r *channelLinkRepoStub) Create(_ context.Context, l *ChannelLink) error {
	if r.links[l.Code] != nil {
		return ErrChannelLinkCodeTaken
	}
	r.nextID++
	l.ID = r.nextID
	r.links[l.Code] = l
	return nil
}
func (r *channelLinkRepoStub) Update(context.Context, *ChannelLink) error { return nil }
func (r *channelLinkRepoStub) Delete(context.Context, int64) error         { return nil }
func (r *channelLinkRepoStub) AddClick(_ context.Context, id int64) error {
	r.clicks[id]++
	return nil
}

func TestChannelLinkCreateAndResolve(t *testing.T) {
	repo := newChannelLinkRepoStub()
	s := NewChannelLinkService(repo, nil)
	ctx := context.Background()

	l, err := s.Create(ctx, ChannelLinkInput{Name: "小红书 Codex 教程", Source: "XiaoHongShu", Medium: "post", TargetPath: "/learn/connect/"})
	require.NoError(t, err)
	require.Regexp(t, `^xiaohongshu-[a-z2-9]{4}$`, l.Code)

	const browser = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"
	require.Equal(t, "/learn/connect/?utm_campaign="+l.Code+"&utm_medium=post&utm_source=xiaohongshu", s.Resolve(ctx, l.Code, browser))
	require.Equal(t, 1, repo.clicks[l.ID])
	// Bots don't count, unknown codes go home.
	s.Resolve(ctx, l.Code, "Googlebot/2.1")
	require.Equal(t, 1, repo.clicks[l.ID])
	require.Equal(t, "/", s.Resolve(ctx, "nope-0000", browser))
	require.Equal(t, "/", s.Resolve(ctx, "../admin", browser))

	// A custom code, an existing query string kept.
	l2, err := s.Create(ctx, ChannelLinkInput{Code: "bili-oct", Name: "B 站视频", Source: "bilibili", TargetPath: "/pricing?tab=plans"})
	require.NoError(t, err)
	require.Equal(t, "/pricing?tab=plans&utm_campaign=bili-oct&utm_source=bilibili", ChannelLinkTarget(l2))
	_, err = s.Create(ctx, ChannelLinkInput{Code: "bili-oct", Name: "again", Source: "bilibili"})
	require.ErrorIs(t, err, ErrChannelLinkCodeTaken)
}

func TestChannelLinkValidation(t *testing.T) {
	s := NewChannelLinkService(newChannelLinkRepoStub(), nil)
	ctx := context.Background()
	for _, in := range []ChannelLinkInput{
		{Name: "", Source: "v2ex"},
		{Name: "x", Source: ""},
		{Name: "x", Source: "v2 ex"},
		{Name: "x", Source: "v2ex", Code: "A"},
		{Name: "x", Source: "v2ex", TargetPath: "https://evil.example/"},
		{Name: "x", Source: "v2ex", TargetPath: "//evil.example/"},
		{Name: "x", Source: "v2ex", TargetPath: "/\\evil.example"},
		{Name: "x", Source: "v2ex", TargetPath: "pricing"},
		{Name: "x", Source: "v2ex", AffCode: "ABC123"}, // no affiliate service to check it
	} {
		_, err := s.Create(ctx, in)
		require.Error(t, err, "%+v", in)
	}
	l, err := s.Create(ctx, ChannelLinkInput{Name: "x", Source: "v2ex"})
	require.NoError(t, err)
	require.Equal(t, "/", l.TargetPath)
}
