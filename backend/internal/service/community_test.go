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
