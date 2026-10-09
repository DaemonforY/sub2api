package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Gallery ad images uploaded by an admin, stored under video.dir/ads and served publicly.

var videoAdFileName = regexp.MustCompile(`^[0-9a-f]{24}\.(png|jpg|webp|gif)$`)

var ErrVideoAdImage = infraerrors.BadRequest("VIDEO_AD_IMAGE", "广告图片需为 PNG / JPG / WebP / GIF，且不超过 2 MB（Image must be PNG, JPG, WebP or GIF up to 2 MB）")

const videoAdMaxBytes = 2 << 20

// SaveAdImage stores an uploaded ad image and returns its public path.
func (s *VideoService) SaveAdImage(_ context.Context, admin *User, data []byte) (string, error) {
	if admin == nil || admin.Role != RoleAdmin {
		return "", ErrVideoForbidden
	}
	if len(data) == 0 || len(data) > videoAdMaxBytes {
		return "", ErrVideoAdImage
	}
	ext := ""
	switch http.DetectContentType(data) {
	case "image/png":
		ext = "png"
	case "image/jpeg":
		ext = "jpg"
	case "image/webp":
		ext = "webp"
	case "image/gif":
		ext = "gif"
	default:
		return "", ErrVideoAdImage
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:12]) + "." + ext
	dir := filepath.Join(s.audioDir, "ads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		return "", err
	}
	return "/api/v1/video/ads/" + name, nil
}

// AdImagePath returns the file of an ad image, or ErrVideoNotFound.
func (s *VideoService) AdImagePath(file string) (string, error) {
	if !videoAdFileName.MatchString(file) {
		return "", ErrVideoNotFound
	}
	path := filepath.Join(s.audioDir, "ads", file)
	if _, err := os.Stat(path); err != nil {
		return "", ErrVideoNotFound
	}
	return path, nil
}
