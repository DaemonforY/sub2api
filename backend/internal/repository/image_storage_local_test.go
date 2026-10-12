//go:build unit

package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLocalImageStorage(t *testing.T) {
	dir := t.TempDir()
	_, err := NewLocalImageStorage(dir, "/api/v1/image-files")
	require.Error(t, err, "the link must be absolute: the canvas loads it from another origin")

	s, err := NewLocalImageStorage(dir, "https://hivegpt.cn/api/v1/image-files/")
	require.NoError(t, err)
	url, err := s.Save(context.Background(), "images/imgtask_abc-0.png", "image/png", []byte("png-bytes"))
	require.NoError(t, err)
	require.Equal(t, "https://hivegpt.cn/api/v1/image-files/images/imgtask_abc-0.png", url)
	data, err := os.ReadFile(filepath.Join(dir, "images", "imgtask_abc-0.png"))
	require.NoError(t, err)
	require.Equal(t, "png-bytes", string(data))

	_, err = s.Save(context.Background(), "../escape.png", "image/png", []byte("x"))
	require.ErrorIs(t, err, errLocalImageKey)
	_, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escape.png"))
	require.True(t, os.IsNotExist(statErr))

	old := filepath.Join(dir, "images", "imgtask_old-0.png")
	require.NoError(t, os.WriteFile(old, []byte("old"), 0o644))
	past := time.Now().Add(-LocalImageFileTTL - time.Hour)
	require.NoError(t, os.Chtimes(old, past, past))
	require.Equal(t, 1, s.removeExpired(time.Now().Add(-LocalImageFileTTL)))
	_, err = os.Stat(old)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(dir, "images", "imgtask_abc-0.png"))
	require.NoError(t, err, "fresh files stay")
}
