//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanvasCloudPaths(t *testing.T) {
	for _, path := range []string{"canvas/manifest.json", "assets/files/image_abc.png", "image-workbench/manifest.json", "video-workbench/files/v.mp4", "comic/manifest.json", "comic/files/image_x1Y-2.png"} {
		require.True(t, canvasCloudPathRe.MatchString(path), path)
	}
	for _, path := range []string{"comics/manifest.json", "comic/other.json", "comic/files/../x", "comic/files/", "../comic/manifest.json", "comic/files/a/b.png"} {
		require.False(t, canvasCloudPathRe.MatchString(path), path)
	}
}
