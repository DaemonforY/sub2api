//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSniffVideo(t *testing.T) {
	mime, ext, ok := sniffVideo(append([]byte{0, 0, 0, 0x20}, []byte("ftypqt  \x00\x00\x00\x00")...))
	require.True(t, ok)
	require.Equal(t, "video/mp4", mime)
	require.Equal(t, "mp4", ext)
	mime, ext, ok = sniffVideo([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x9F})
	require.True(t, ok)
	require.Equal(t, "video/webm", mime)
	require.Equal(t, "webm", ext)
	for _, data := range [][]byte{nil, []byte("ftyp"), []byte("\x89PNG\r\n\x1a\n0000"), []byte("<html><video>")} {
		_, _, ok = sniffVideo(data)
		require.False(t, ok)
	}
	require.Equal(t, 0, clampVideoDuration(-5))
	require.Equal(t, communityVideoMaxMs, clampVideoDuration(1<<30))
}

func TestCommunityMediaPathAcceptsVideos(t *testing.T) {
	store := NewCommunityMediaStore(t.TempDir())
	_, ok := store.Path("0123456789abcdef0123456789abcdef.mp4")
	require.True(t, ok)
	_, ok = store.Path("0123456789abcdef0123456789abcdef.webm")
	require.True(t, ok)
	_, ok = store.Path("0123456789abcdef0123456789abcdef_t.mov")
	require.False(t, ok)
}
