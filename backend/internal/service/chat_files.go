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
	// The page shows pictures as JPEG previews: a drawn 1536×1024 PNG is 2–3 MB, which took 11 s
	// to reach a user in China; its preview is about a tenth of that.
	chatPreviewEdge     = 1280
	chatPreviewMinBytes = 400 << 10
	chatPreviewSuffix   = ".preview.jpg"
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

// Preview is the picture for showing on the page: a JPEG of at most 1280 px, made on first use and
// kept beside the original. Small files and GIFs are returned as they are.
func (s *ChatFileStore) Preview(name string) ([]byte, string, error) {
	p, ok := s.path(name)
	if !ok {
		return nil, "", os.ErrNotExist
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, "", err
	}
	mime := http.DetectContentType(data)
	if len(data) <= chatPreviewMinBytes || mime == "image/gif" {
		return data, mime, nil
	}
	if cached, err := os.ReadFile(p + chatPreviewSuffix); err == nil && len(cached) > 0 {
		return cached, "image/jpeg", nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mime, nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if edge := max(w, h); edge > chatPreviewEdge {
		w, h = max(1, w*chatPreviewEdge/edge), max(1, h*chatPreviewEdge/edge)
	}
	var out bytes.Buffer
	if err := encodeJPEGTo(&out, scaleTo(img, w, h), 85); err != nil {
		return data, mime, nil
	}
	tmp := p + ".tmp" + chatPreviewSuffix
	if os.WriteFile(tmp, out.Bytes(), 0o644) == nil && os.Rename(tmp, p+chatPreviewSuffix) != nil {
		_ = os.Remove(tmp)
	}
	return out.Bytes(), "image/jpeg", nil
}

func (s *ChatFileStore) Remove(names ...string) {
	for _, name := range names {
		if p, ok := s.path(name); ok {
			_ = os.Remove(p)
			_ = os.Remove(p + chatPreviewSuffix)
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
