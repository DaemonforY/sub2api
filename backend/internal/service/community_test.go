//go:build unit

package service

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateCommunityHandle(t *testing.T) {
	for _, ok := range []string{"abc", "xiao_lin", "a2026", "abcdefghij0123456789"} {
		require.NoError(t, validateCommunityHandle(ok), ok)
	}
	for _, bad := range []string{"ab", "1abc", "_abc", "a__b", "Abc", "abc-d", "abcdefghij01234567890", "小林"} {
		require.ErrorIs(t, validateCommunityHandle(bad), ErrCommunityHandleInvalid, bad)
	}
	for _, reserved := range []string{"admin", "explore", "hivegpt_cn", "official_x", "alipay_kefu"} {
		require.ErrorIs(t, validateCommunityHandle(reserved), ErrCommunityHandleReserved, reserved)
	}
}

func TestCommunityTextFlagsAndNormalization(t *testing.T) {
	require.Empty(t, communityTextFlags("月光下的古风庭院", "水墨风格"))
	require.NotEmpty(t, communityTextFlags("今晚 百家乐 开局"))
	require.NotEmpty(t, communityTextFlags("NSFW portrait"))
	require.Equal(t, []string{"国风", "插画", "海报"}, normalizeTags([]string{" 国风 ", "#插画", "国风", "", "海报"}))
	require.Len(t, normalizeTags([]string{"a", "b", "c", "d", "e", "f", "g"}), communityMaxTags)
	require.Equal(t, json.RawMessage(`{}`), normalizeParams(json.RawMessage(`not json`)))
	require.Equal(t, json.RawMessage(`{"size":"1:1"}`), normalizeParams(json.RawMessage(`{"size":"1:1"}`)))
	require.Equal(t, WorkVisibilityPublic, normalizeVisibility("everyone"))
	require.Equal(t, "canvas", normalizeSource("other"))
}

func encodeTestImage(t *testing.T, w, h int, asJPEG bool) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	var buf bytes.Buffer
	if asJPEG {
		require.NoError(t, jpeg.Encode(&buf, img, nil))
	} else {
		require.NoError(t, png.Encode(&buf, img))
	}
	return buf.Bytes()
}

func TestCommunityMediaStore(t *testing.T) {
	store := NewCommunityMediaStore(t.TempDir())
	_, err := store.SaveImage([]byte("not an image"))
	require.ErrorIs(t, err, ErrCommunityImageInvalid)
	_, err = store.SaveImage(nil)
	require.ErrorIs(t, err, ErrCommunityImageInvalid)

	small, err := store.SaveImage(encodeTestImage(t, 200, 100, true))
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", small.MimeType)
	require.Equal(t, 200, small.Width)
	require.Regexp(t, `^[a-f0-9]{32}\.jpg$`, small.File)
	require.Regexp(t, `^[a-f0-9]{32}_t\.jpg$`, small.ThumbFile)
	_, ok := store.Path(small.ThumbFile)
	require.True(t, ok)
	_, ok = store.Path("../etc/passwd")
	require.False(t, ok)

	avatar, err := store.SaveAvatar(encodeTestImage(t, 600, 300, false))
	require.NoError(t, err)
	path, _ := store.Path(avatar)
	f, err := openForTest(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	require.NoError(t, err)
	require.Equal(t, communityAvatarSize, cfg.Width)
	require.Equal(t, communityAvatarSize, cfg.Height, "avatars are square")
}

func TestWorkVisibility(t *testing.T) {
	w := &Work{UserID: 1, Status: WorkStatusApproved, Visibility: WorkVisibilityUnlisted}
	require.True(t, visibleTo(w, 0), "unlisted works open by link")
	w.Visibility = WorkVisibilityPrivate
	require.False(t, visibleTo(w, 2))
	require.True(t, visibleTo(w, 1), "the author always sees their work")
	w.Visibility, w.Status = WorkVisibilityPublic, WorkStatusPending
	require.False(t, visibleTo(w, 0))
}

func openForTest(path string) (*os.File, error) { return os.Open(path) }

func TestRenderShareMetaEscapesAndResolvesImages(t *testing.T) {
	out := renderShareMeta(&ShareCard{Title: `月光 "庭院" <b>`, Description: "a & b", Image: "/api/v1/community/media/x_t.jpg", ImageWidth: 640, ImageHeight: 360, Type: "article"}, "https://hivegpt.cn/", "https://canvas.hivegpt.cn/w/1")
	require.Contains(t, out, `<meta property="og:title" content="月光 &#34;庭院&#34; &lt;b&gt;" />`)
	require.Contains(t, out, `content="a &amp; b"`)
	require.Contains(t, out, `<meta property="og:image" content="https://hivegpt.cn/api/v1/community/media/x_t.jpg" />`)
	require.Contains(t, out, `<meta property="og:url" content="https://canvas.hivegpt.cn/w/1" />`)
	require.Contains(t, out, `summary_large_image`)
	require.NotContains(t, out, "<b>")

	noImage := renderShareMeta(&ShareCard{Title: "x", Type: "profile"}, "https://hivegpt.cn", "")
	require.NotContains(t, noImage, "og:image")
	require.NotContains(t, noImage, "og:url")
}

func TestTrustedShareURL(t *testing.T) {
	allowed := []string{"https://canvas.hivegpt.cn", " https://canvas2.example.com/ "}
	require.Equal(t, "https://canvas.hivegpt.cn/w/12", TrustedShareURL("https://canvas.hivegpt.cn/w/12?utm=1#x", allowed))
	require.Equal(t, "https://canvas2.example.com/u/a", TrustedShareURL("https://canvas2.example.com/u/a", allowed))
	require.Empty(t, TrustedShareURL("https://evil.example/w/1", allowed))
	require.Empty(t, TrustedShareURL("http://canvas.hivegpt.cn/w/1", allowed))
	require.Empty(t, TrustedShareURL("javascript:alert(1)", allowed))
}

func TestFeedSnapshot(t *testing.T) {
	now := time.Date(2026, 10, 2, 12, 0, 0, 500_000_000, time.UTC)
	require.Equal(t, now, FeedSnapshot(0, now))
	require.Equal(t, now, FeedSnapshot(now.Add(time.Hour).Unix(), now), "future times fall back to now")
	require.Equal(t, now, FeedSnapshot(now.Add(-7*time.Hour).Unix(), now), "stale times fall back to now")
	require.True(t, now.Add(-time.Hour).Truncate(time.Second).Equal(FeedSnapshot(now.Add(-time.Hour).Unix(), now)))
	require.Equal(t, now.Unix()+1, FeedSnapshotUnix(now), "rounded up so the page's works stay in")
	require.Equal(t, now.Truncate(time.Second).Unix(), FeedSnapshotUnix(now.Truncate(time.Second)))
}
