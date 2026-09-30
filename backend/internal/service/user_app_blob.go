package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	// UserAppBlobMaxBytes caps one synced image (same limit as contest entries).
	UserAppBlobMaxBytes = 10 << 20
	// Per-user quota; the least recently used images are evicted beyond it.
	userAppBlobMaxCount = 40
	userAppBlobMaxTotal = 100 << 20
)

var (
	ErrUserAppBlobInvalid  = infraerrors.BadRequest("APP_BLOB_INVALID", "图片格式不支持，请使用 PNG、JPEG、WebP 或 GIF")
	ErrUserAppBlobTooLarge = infraerrors.BadRequest("APP_BLOB_TOO_LARGE", "图片太大（上限 10MB），请压缩后再试")
	ErrUserAppBlobNotFound = infraerrors.NotFound("APP_BLOB_NOT_FOUND", "图片不存在或已被清理")
)

var userAppBlobIDRe = regexp.MustCompile(`^[a-f0-9]{32}$`)

// UserAppBlob is the metadata of one synced image.
type UserAppBlob struct {
	ID         string    `json:"id"`
	UserID     int64     `json:"-"`
	SHA256     string    `json:"sha256"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"-"`
}

type UserAppBlobRepository interface {
	FindBlobBySHA(ctx context.Context, userID int64, sha string) (*UserAppBlob, error)
	GetBlob(ctx context.Context, userID int64, id string) (*UserAppBlob, error)
	InsertBlob(ctx context.Context, blob *UserAppBlob) error
	TouchBlob(ctx context.Context, userID int64, id string) error
	// BlobsOverQuota returns the user's blobs outside the newest maxCount / maxTotal bytes (by last use).
	BlobsOverQuota(ctx context.Context, userID int64, maxCount int, maxTotal int64) ([]UserAppBlob, error)
	DeleteBlobs(ctx context.Context, userID int64, ids []string) error
}

// UserAppBlobService stores synced images on the data volume, owned per user.
type UserAppBlobService struct {
	repo UserAppBlobRepository
	dir  string
}

func NewUserAppBlobService(repo UserAppBlobRepository, dir string) *UserAppBlobService {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/app-state-blobs"
	}
	return &UserAppBlobService{repo: repo, dir: filepath.Clean(dir)}
}

// Upload stores an image for the user; the same bytes uploaded twice return the same blob.
func (s *UserAppBlobService) Upload(ctx context.Context, userID int64, data []byte) (*UserAppBlob, error) {
	if len(data) == 0 {
		return nil, ErrUserAppBlobInvalid
	}
	if len(data) > UserAppBlobMaxBytes {
		return nil, ErrUserAppBlobTooLarge
	}
	mime := http.DetectContentType(data)
	ext, ok := contestImageExt[mime]
	if !ok {
		return nil, ErrUserAppBlobInvalid
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	if existing, err := s.repo.FindBlobBySHA(ctx, userID, sha); err != nil {
		return nil, err
	} else if existing != nil {
		if _, statErr := os.Stat(s.path(existing.ID, existing.MimeType)); statErr == nil {
			_ = s.repo.TouchBlob(ctx, userID, existing.ID)
			return existing, nil
		}
		// File vanished (volume restore etc.): drop the row and store again.
		_ = s.repo.DeleteBlobs(ctx, userID, []string{existing.ID})
	}

	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, fmt.Errorf("generate blob id: %w", err)
	}
	blob := &UserAppBlob{ID: hex.EncodeToString(buf[:]), UserID: userID, SHA256: sha, MimeType: mime, SizeBytes: int64(len(data))}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("create app blob dir: %w", err)
	}
	final := s.path(blob.ID, mime)
	tmp := filepath.Join(s.dir, "."+blob.ID+"."+ext+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return nil, fmt.Errorf("write app blob: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("store app blob: %w", err)
	}
	if err := s.repo.InsertBlob(ctx, blob); err != nil {
		_ = os.Remove(final)
		// A concurrent upload of the same bytes won the unique index: return that one.
		if existing, findErr := s.repo.FindBlobBySHA(ctx, userID, sha); findErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	s.evictOverQuota(ctx, userID)
	return blob, nil
}

// Open returns the file of a blob owned by the user.
func (s *UserAppBlobService) Open(ctx context.Context, userID int64, id string) (path string, blob *UserAppBlob, err error) {
	if !userAppBlobIDRe.MatchString(id) {
		return "", nil, ErrUserAppBlobNotFound
	}
	blob, err = s.repo.GetBlob(ctx, userID, id)
	if err != nil {
		return "", nil, err
	}
	if blob == nil {
		return "", nil, ErrUserAppBlobNotFound
	}
	path = s.path(blob.ID, blob.MimeType)
	if _, statErr := os.Stat(path); statErr != nil {
		return "", nil, ErrUserAppBlobNotFound
	}
	_ = s.repo.TouchBlob(ctx, userID, id)
	return path, blob, nil
}

func (s *UserAppBlobService) evictOverQuota(ctx context.Context, userID int64) {
	stale, err := s.repo.BlobsOverQuota(ctx, userID, userAppBlobMaxCount, userAppBlobMaxTotal)
	if err != nil || len(stale) == 0 {
		return
	}
	ids := make([]string, 0, len(stale))
	for _, blob := range stale {
		ids = append(ids, blob.ID)
	}
	if err := s.repo.DeleteBlobs(ctx, userID, ids); err != nil {
		return
	}
	for _, blob := range stale {
		_ = os.Remove(s.path(blob.ID, blob.MimeType))
	}
}

func (s *UserAppBlobService) path(id, mime string) string {
	return filepath.Join(s.dir, id+"."+contestImageExt[mime])
}
