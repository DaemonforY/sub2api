package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// PromptCoverMaxBytes caps a user's cover image (the canvas compresses before uploading).
	PromptCoverMaxBytes = 3 << 20
	// PromptCoverPublicPrefix is where covers are served (public, like contest images).
	PromptCoverPublicPrefix = "/api/v1/prompt-library/covers/"
	promptCoversPerDay      = 60
)

var (
	ErrPromptCoverInvalid  = infraerrors.BadRequest("PROMPT_COVER_INVALID", "封面图格式不支持，请使用 PNG、JPEG、WebP 或 GIF")
	ErrPromptCoverTooLarge = infraerrors.BadRequest("PROMPT_COVER_TOO_LARGE", "封面图太大（上限 3MB），请压缩后再试")
	ErrPromptCoverTooMany  = infraerrors.TooManyRequests("PROMPT_COVER_TOO_MANY", "今天上传的封面图太多了，请明天再试")
)

var promptCoverFileRe = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg|webp|gif)$`)

// PromptCoverStore keeps cover images on the data volume.
type PromptCoverStore struct {
	dir string
}

func NewPromptCoverStore(dir string) *PromptCoverStore {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/prompt-covers"
	}
	return &PromptCoverStore{dir: filepath.Clean(dir)}
}

// Path resolves a stored file name; ok=false for anything that is not one of ours.
func (s *PromptCoverStore) Path(file string) (string, bool) {
	if !promptCoverFileRe.MatchString(file) {
		return "", false
	}
	return filepath.Join(s.dir, file), true
}

func (s *PromptCoverStore) save(data []byte) (string, error) {
	ext, ok := contestImageExt[http.DetectContentType(data)]
	if !ok {
		return "", ErrPromptCoverInvalid
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate cover name: %w", err)
	}
	name := hex.EncodeToString(buf[:]) + "." + ext
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", fmt.Errorf("create cover dir: %w", err)
	}
	tmp := filepath.Join(s.dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", fmt.Errorf("write cover: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("store cover: %w", err)
	}
	return name, nil
}

// PromptCoverURL is the public path of a stored cover.
func PromptCoverURL(file string) string { return PromptCoverPublicPrefix + file }

// PromptCoverFileFromURL accepts the path returned by an upload (optionally with the site origin).
func PromptCoverFileFromURL(url string) (string, bool) {
	idx := strings.Index(url, PromptCoverPublicPrefix)
	if idx < 0 {
		return "", false
	}
	if idx > 0 && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return "", false
	}
	file := url[idx+len(PromptCoverPublicPrefix):]
	return file, promptCoverFileRe.MatchString(file)
}

// UploadCover stores a cover image for one of the user's prompts and returns its public path.
func (s *PromptLibraryService) UploadCover(ctx context.Context, userID int64, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrPromptCoverInvalid
	}
	if len(data) > PromptCoverMaxBytes {
		return "", ErrPromptCoverTooLarge
	}
	recent, err := s.repo.CountCoversSince(ctx, userID, s.now().Add(-24*time.Hour))
	if err != nil {
		return "", err
	}
	if recent >= promptCoversPerDay {
		return "", ErrPromptCoverTooMany
	}
	file, err := s.covers.save(data)
	if err != nil {
		return "", err
	}
	if err := s.repo.InsertCover(ctx, file, userID, int64(len(data))); err != nil {
		if p, ok := s.covers.Path(file); ok {
			_ = os.Remove(p)
		}
		return "", err
	}
	return PromptCoverURL(file), nil
}
