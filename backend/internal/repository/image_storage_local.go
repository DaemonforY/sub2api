package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"go.uber.org/zap"
)

// LocalImageStorage keeps async image results on this server's disk (<data dir>/image-files) and
// returns <public_base_url>/<key>, served by the public /api/v1/image-files route. Keys hold the
// random task ID, so a link can't be guessed. Files older than LocalImageFileTTL are removed by a
// janitor, matching the image task's own lifetime — no object storage account needed.
type LocalImageStorage struct {
	dir     string
	baseURL string
}

// LocalImageFileTTL: how long a stored result stays downloadable (the task record lives 24h).
const LocalImageFileTTL = 26 * time.Hour

var (
	errLocalImageKey    = errors.New("invalid image key")
	localJanitorOnce    sync.Map // dir -> *sync.Once
	localJanitorEvery   = time.Hour
	localJanitorStartup = 2 * time.Minute
)

func NewLocalImageStorage(dir, publicBaseURL string) (*LocalImageStorage, error) {
	dir = strings.TrimSpace(dir)
	base := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	if dir == "" {
		return nil, errors.New("local image storage: data directory is not set")
	}
	if !strings.HasPrefix(base, "https://") && !strings.HasPrefix(base, "http://") {
		return nil, errors.New("local image storage: public_base_url must be an absolute http(s) URL, e.g. https://example.com/api/v1/image-files")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("local image storage: create %s: %w", dir, err)
	}
	s := &LocalImageStorage{dir: dir, baseURL: base}
	if once, _ := localJanitorOnce.LoadOrStore(dir, &sync.Once{}); once != nil {
		if o, ok := once.(*sync.Once); ok {
			o.Do(func() { go s.janitor() })
		}
	}
	return s, nil
}

func (s *LocalImageStorage) Save(_ context.Context, key, _ string, data []byte) (string, error) {
	path := service.LocalImagePath(s.dir, key)
	if path == "" {
		return "", errLocalImageKey
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return s.baseURL + "/" + strings.TrimLeft(filepath.ToSlash(key), "/"), nil
}

func (s *LocalImageStorage) janitor() {
	time.Sleep(localJanitorStartup)
	for {
		s.removeExpired(time.Now().Add(-LocalImageFileTTL))
		time.Sleep(localJanitorEvery)
	}
}

func (s *LocalImageStorage) removeExpired(before time.Time) int {
	removed := 0
	_ = filepath.WalkDir(s.dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil && info.ModTime().Before(before) {
			if os.Remove(path) == nil {
				removed++
			}
		}
		return nil
	})
	if removed > 0 {
		logger.L().Info("image_storage.local_cleanup", zap.Int("removed", removed), zap.String("dir", s.dir))
	}
	return removed
}
