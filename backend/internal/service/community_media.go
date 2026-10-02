package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif" // decoders for uploaded works
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Images of community works on the data volume: the original as uploaded (keeping any metadata
// the generator wrote, such as AI-generation marks) and a JPEG thumbnail for the feed.

const (
	CommunityMediaPublicPrefix = "/api/v1/community/media/"
	CommunityImageMaxBytes     = 15 << 20
	communityImageMaxPixels    = 60_000_000
	communityThumbWidth        = 640
	// Avatars are cropped to a square of this size.
	communityAvatarSize = 256
)

var (
	ErrCommunityImageInvalid  = infraerrors.BadRequest("COMMUNITY_IMAGE_INVALID", "只支持 PNG、JPG、WebP、GIF 图片（Unsupported image）")
	ErrCommunityImageTooLarge = infraerrors.BadRequest("COMMUNITY_IMAGE_TOO_LARGE", "单张图片不能超过 15 MB、6000 万像素，可以先在「图片工具」里压缩（Image too large）")
)

var communityMediaFileRe = regexp.MustCompile(`^[a-f0-9]{32}(_t)?\.(png|jpg|webp|gif)$`)

var communityImageExt = map[string]string{"image/png": "png", "image/jpeg": "jpg", "image/webp": "webp", "image/gif": "gif"}

type CommunityMediaStore struct {
	dir string
}

func NewCommunityMediaStore(dir string) *CommunityMediaStore {
	if strings.TrimSpace(dir) == "" {
		dir = "./data/community"
	}
	return &CommunityMediaStore{dir: filepath.Clean(dir)}
}

// StoredImage is a saved upload.
type StoredImage struct {
	File      string
	ThumbFile string
	MimeType  string
	Width     int
	Height    int
	Size      int64
}

func randomMediaName() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func (s *CommunityMediaStore) write(name string, data []byte) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create community dir: %w", err)
	}
	tmp := filepath.Join(s.dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write community image: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(s.dir, name)); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("store community image: %w", err)
	}
	return nil
}

// decodeUpload sniffs and bounds-checks an upload before decoding it.
func decodeUpload(data []byte) (image.Image, string, image.Config, error) {
	if len(data) == 0 {
		return nil, "", image.Config{}, ErrCommunityImageInvalid
	}
	if len(data) > CommunityImageMaxBytes {
		return nil, "", image.Config{}, ErrCommunityImageTooLarge
	}
	mime := http.DetectContentType(data)
	if _, ok := communityImageExt[mime]; !ok {
		return nil, "", image.Config{}, ErrCommunityImageInvalid
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, "", image.Config{}, ErrCommunityImageInvalid
	}
	if cfg.Width*cfg.Height > communityImageMaxPixels {
		return nil, "", image.Config{}, ErrCommunityImageTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", image.Config{}, ErrCommunityImageInvalid
	}
	return img, mime, cfg, nil
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var out bytes.Buffer
	if err := encodeJPEGTo(&out, img, 82); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func encodeJPEGTo(w io.Writer, img image.Image, quality int) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// scaleTo draws src into a w×h RGBA on white (thumbnails are JPEG: no transparency).
func scaleTo(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// SaveImage stores a work image and its thumbnail.
func (s *CommunityMediaStore) SaveImage(data []byte) (*StoredImage, error) {
	img, mime, cfg, err := decodeUpload(data)
	if err != nil {
		return nil, err
	}
	base, err := randomMediaName()
	if err != nil {
		return nil, err
	}
	out := &StoredImage{File: base + "." + communityImageExt[mime], ThumbFile: base + "_t.jpg", MimeType: mime, Width: cfg.Width, Height: cfg.Height, Size: int64(len(data))}
	tw, th := cfg.Width, cfg.Height
	if tw > communityThumbWidth {
		th = max(1, cfg.Height*communityThumbWidth/cfg.Width)
		tw = communityThumbWidth
	}
	thumb, err := encodeJPEG(scaleTo(img, tw, th))
	if err != nil {
		return nil, err
	}
	if err := s.write(out.File, data); err != nil {
		return nil, err
	}
	if err := s.write(out.ThumbFile, thumb); err != nil {
		s.Remove(out.File)
		return nil, err
	}
	return out, nil
}

// SaveAvatar centre-crops to a square and stores a JPEG.
func (s *CommunityMediaStore) SaveAvatar(data []byte) (string, error) {
	img, _, cfg, err := decodeUpload(data)
	if err != nil {
		return "", err
	}
	side := min(cfg.Width, cfg.Height)
	b := img.Bounds()
	x0 := b.Min.X + (cfg.Width-side)/2
	y0 := b.Min.Y + (cfg.Height-side)/2
	square := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(square, square.Bounds(), img, image.Point{X: x0, Y: y0}, draw.Src)
	size := min(side, communityAvatarSize)
	encoded, err := encodeJPEG(scaleTo(square, size, size))
	if err != nil {
		return "", err
	}
	base, err := randomMediaName()
	if err != nil {
		return "", err
	}
	name := base + ".jpg"
	if err := s.write(name, encoded); err != nil {
		return "", err
	}
	return name, nil
}

// Path resolves a stored file name (ok=false for anything else).
func (s *CommunityMediaStore) Path(name string) (string, bool) {
	if !communityMediaFileRe.MatchString(name) {
		return "", false
	}
	return filepath.Join(s.dir, name), true
}

func (s *CommunityMediaStore) Remove(names ...string) {
	for _, name := range names {
		if p, ok := s.Path(name); ok {
			_ = os.Remove(p)
		}
	}
}

// CommunityMediaURL is the public path of a stored file ("" stays "").
func CommunityMediaURL(name string) string {
	if name == "" {
		return ""
	}
	return CommunityMediaPublicPrefix + name
}
