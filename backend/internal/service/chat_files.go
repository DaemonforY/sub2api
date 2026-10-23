package service

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ChatFileStore keeps the pictures of AI 对话 on disk, flat in one directory under random names.
// They are private: the page fetches them through the API, which checks the owner.

const (
	ChatFileMaxBytes  = 20 << 20
	chatFileMaxPixels = 40_000_000
)

var (
	chatFileExt  = map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp", "image/gif": "gif"}
	chatFileName = regexp.MustCompile(`^[0-9a-f]{32}\.(png|jpg|webp|gif)$`)

	ErrChatFileInvalid  = infraerrors.BadRequest("CHAT_FILE_TYPE", "只支持 PNG、JPG、WebP、GIF 图片（Unsupported image）")
	ErrChatFileTooLarge = infraerrors.BadRequest("CHAT_FILE_TOO_LARGE", "图片太大了，请换一张 20MB 以内的（Image too large）")
)

type ChatFileStore struct {
	dir string
}

func NewChatFileStore(dir string) *ChatFileStore {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/chat"
	}
	return &ChatFileStore{dir: filepath.Clean(dir)}
}

type chatStored struct {
	Name   string
	Mime   string
	Width  int
	Height int
}

// Save checks that data is a picture and writes it.
func (s *ChatFileStore) Save(data []byte) (*chatStored, error) {
	if len(data) == 0 {
		return nil, ErrChatFileInvalid
	}
	if len(data) > ChatFileMaxBytes {
		return nil, ErrChatFileTooLarge
	}
	mime := http.DetectContentType(data)
	ext, ok := chatFileExt[mime]
	if !ok {
		return nil, ErrChatFileInvalid
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrChatFileInvalid
	}
	if cfg.Width*cfg.Height > chatFileMaxPixels {
		return nil, ErrChatFileTooLarge
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	name := hex.EncodeToString(b[:]) + "." + ext
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fmt.Errorf("create chat dir: %w", err)
	}
	tmp := filepath.Join(s.dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return nil, fmt.Errorf("write chat file: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("write chat file: %w", err)
	}
	return &chatStored{Name: name, Mime: mime, Width: cfg.Width, Height: cfg.Height}, nil
}

func (s *ChatFileStore) path(name string) (string, bool) {
	if !chatFileName.MatchString(name) {
		return "", false
	}
	return filepath.Join(s.dir, name), true
}

func (s *ChatFileStore) Read(name string) ([]byte, error) {
	p, ok := s.path(name)
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(p)
}

// DataURL is the picture as a data: URL for a model call ("" if it is gone).
func (s *ChatFileStore) DataURL(name string) string {
	data, err := s.Read(name)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func (s *ChatFileStore) Remove(names ...string) {
	for _, name := range names {
		if p, ok := s.path(name); ok {
			_ = os.Remove(p)
		}
	}
}

// OlderThan lists the stored files last written before t.
func (s *ChatFileStore) OlderThan(t time.Time) []string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !chatFileName.MatchString(e.Name()) {
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().Before(t) {
			out = append(out, e.Name())
		}
	}
	return out
}
