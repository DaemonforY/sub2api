package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // decoders for image.Decode
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Pictures as material: the user adds photos / screenshots / product shots on the create page,
// the script is written looking at them (vision) and the scenes show them with S.image(id).
//
// An image is uploaded on its own first (POST /api/v1/video/images) while the user is still
// writing: it is decoded, scaled to at most 2048 px, re-encoded (dropping EXIF and anything else
// in the file) and parked in video.dir/_stage/<user>/ under a content hash. Creating the project
// moves the chosen ones into the project directory as img1…imgN.

const (
	VideoImagesMax        = 9
	VideoImageMaxBytes    = 20 << 20
	videoImageMaxPixels   = 60_000_000 // decoded size cap (a 20 MB file can hide a huge bitmap)
	videoImageMaxSide     = 2048
	videoImageModelSide   = 1024 // what the model sees
	videoImageStagePerDay = 90
	videoImageStageTTL    = 24 * time.Hour
)

var (
	ErrVideoImageType  = infraerrors.BadRequest("VIDEO_IMAGE_TYPE", "图片只支持 JPG、PNG、WebP 或 GIF，请换一张（Only JPG, PNG, WebP or GIF images）")
	ErrVideoImageBig   = infraerrors.BadRequest("VIDEO_IMAGE_TOO_LARGE", "图片不能超过 20 MB，请压缩后再传（Images must be 20 MB or smaller）")
	ErrVideoImageHuge  = infraerrors.BadRequest("VIDEO_IMAGE_TOO_MANY_PIXELS", "图片分辨率太大，请缩小到 6000 万像素以内再传（The image has too many pixels）")
	ErrVideoImageDaily = infraerrors.TooManyRequests("VIDEO_IMAGE_DAILY", fmt.Sprintf("24 小时内最多上传 %d 张素材图，请明天再来（Image upload limit reached）", videoImageStagePerDay))
	ErrVideoImageGone  = infraerrors.BadRequest("VIDEO_IMAGE_GONE", "有图片已经过期或没有传完，请删掉后重新添加（An image is missing; add it again）")
	ErrVideoImageCount = infraerrors.BadRequest("VIDEO_IMAGE_COUNT", fmt.Sprintf("最多添加 %d 张图片（Up to %d images）", VideoImagesMax, VideoImagesMax))
)

// VideoImage is one picture of a project (in options.images and, once the script exists, spec.images).
type VideoImage struct {
	ID     string `json:"id"` // img1…
	File   string `json:"file"`
	Mime   string `json:"mime"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
	Desc   string `json:"desc,omitempty"` // what is in it, written by the model with the script
	URL    string `json:"url,omitempty"`  // filled in for the API and the player, not trusted on input
}

// VideoStagedImage is the answer to an image upload: the token the create request refers to.
type VideoStagedImage struct {
	Token  string `json:"token"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
}

var videoStageTokenRe = regexp.MustCompile(`^[0-9a-f]{24}\.(jpg|png)$`)

func (s *VideoService) stageDir(userID int64) string {
	return filepath.Join(s.audioDir, "_stage", fmt.Sprint(userID))
}

// StageImage checks, normalises and parks one picture for a project the user is about to create.
func (s *VideoService) StageImage(ctx context.Context, key *APIKey, f VideoUploadFile) (*VideoStagedImage, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	if f.Reader == nil || f.Size <= 0 {
		return nil, ErrVideoImageType
	}
	if f.Size > VideoImageMaxBytes {
		return nil, ErrVideoImageBig
	}
	since := s.now().Add(-24 * time.Hour)
	if s.recentImages(key.UserID, since) >= videoImageStagePerDay {
		return nil, ErrVideoImageDaily
	}
	data, ext, w, h, err := normalizeVideoImage(io.NewSectionReader(f.Reader, 0, f.Size), videoImageMaxSide)
	if err != nil {
		return nil, err
	}
	dir := s.stageDir(key.UserID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s.cleanStage(dir)
	sum := sha256.Sum256(data)
	token := hex.EncodeToString(sum[:12]) + ext
	if err := os.WriteFile(filepath.Join(dir, token), data, 0o644); err != nil {
		return nil, err
	}
	s.noteImage(key.UserID, since)
	return &VideoStagedImage{Token: token, Width: w, Height: h, Size: int64(len(data))}, nil
}

// cleanStage drops parked pictures nobody used within a day.
func (s *VideoService) cleanStage(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := s.now().Add(-videoImageStageTTL)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// attachImages copies the staged pictures named by tokens into the project as img1…imgN.
func (s *VideoService) attachImages(p *VideoProject, tokens []string) ([]VideoImage, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	if len(tokens) > VideoImagesMax {
		return nil, ErrVideoImageCount
	}
	dir := s.projectDir(p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := make([]VideoImage, 0, len(tokens))
	for i, token := range tokens {
		img, err := s.stagedImage(p, token, fmt.Sprintf("img%d", i+1))
		if err != nil {
			return nil, err
		}
		out = append(out, img)
	}
	return out, nil
}

// stagedImage copies one staged picture into the project directory under the given id.
func (s *VideoService) stagedImage(p *VideoProject, token, id string) (VideoImage, error) {
	if !videoStageTokenRe.MatchString(token) {
		return VideoImage{}, ErrVideoImageGone
	}
	data, err := os.ReadFile(filepath.Join(s.stageDir(p.UserID), token))
	if err != nil {
		return VideoImage{}, ErrVideoImageGone
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return VideoImage{}, ErrVideoImageGone
	}
	mime := "image/jpeg"
	if format == "png" {
		mime = "image/png"
	}
	img := VideoImage{ID: id, File: "img-" + token, Mime: mime, Width: cfg.Width, Height: cfg.Height, Size: int64(len(data))}
	if err := os.WriteFile(filepath.Join(s.projectDir(p.ID), filepath.Base(img.File)), data, 0o644); err != nil { //nolint:gosec // G703: token matched videoStageTokenRe (hex + .jpg/.png), Base keeps it in dir
		return VideoImage{}, err
	}
	return img, nil
}

// normalizeVideoImage decodes a picture (JPEG / PNG / WebP / GIF first frame), scales it to at
// most maxSide on the long edge and re-encodes it: PNG when it has transparency, JPEG otherwise.
func normalizeVideoImage(r io.ReadSeeker, maxSide int) ([]byte, string, int, int, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return nil, "", 0, 0, ErrVideoImageType
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > videoImageMaxPixels {
		return nil, "", 0, 0, ErrVideoImageHuge
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, "", 0, 0, err
	}
	src, _, err := image.Decode(r)
	if err != nil {
		return nil, "", 0, 0, ErrVideoImageType
	}
	img := scaleVideoImage(src, maxSide)
	b := img.Bounds()
	var buf bytes.Buffer
	if videoImageOpaque(img) {
		if err := jpeg.Encode(&buf, flattenVideoImage(img), &jpeg.Options{Quality: 88}); err != nil {
			return nil, "", 0, 0, err
		}
		return buf.Bytes(), ".jpg", b.Dx(), b.Dy(), nil
	}
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, img); err != nil {
		return nil, "", 0, 0, err
	}
	return buf.Bytes(), ".png", b.Dx(), b.Dy(), nil
}

func scaleVideoImage(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	if w >= h {
		h, w = max(1, h*maxSide/w), maxSide
	} else {
		w, h = max(1, w*maxSide/h), maxSide
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

// videoImageOpaque reports whether every pixel is opaque (sampled on large images).
func videoImageOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	b := img.Bounds()
	step := max(1, max(b.Dx(), b.Dy())/512)
	for y := b.Min.Y; y < b.Max.Y; y += step {
		for x := b.Min.X; x < b.Max.X; x += step {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return false
			}
		}
	}
	return true
}

// flattenVideoImage draws the picture on white (JPEG has no alpha; paletted GIFs need converting).
func flattenVideoImage(img image.Image) image.Image {
	if _, ok := img.(*image.YCbCr); ok {
		return img
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)
	return dst
}

// imagesForModel returns the project's pictures as small JPEG / PNG data URLs for a vision call,
// by picture id (pictures that cannot be read are left out).
func (s *VideoService) imagesForModel(p *VideoProject) map[string]string {
	out := map[string]string{}
	for _, img := range p.Options.Images {
		f, err := os.Open(filepath.Join(s.projectDir(p.ID), filepath.Base(img.File)))
		if err != nil {
			continue
		}
		data, ext, _, _, err := normalizeVideoImage(f, videoImageModelSide)
		_ = f.Close()
		if err != nil {
			continue
		}
		mime := "image/jpeg"
		if ext == ".png" {
			mime = "image/png"
		}
		out[img.ID] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	}
	return out
}

// orderedImages lists the data URLs in the project's picture order.
func orderedImages(p *VideoProject, data map[string]string) []string {
	var out []string
	for _, img := range p.Options.Images {
		if d := data[img.ID]; d != "" {
			out = append(out, d)
		}
	}
	return out
}

// videoImageFile finds a project picture by file name.
func videoImageFile(p *VideoProject, file string) *VideoImage {
	for i := range p.Options.Images {
		if p.Options.Images[i].File == file {
			return &p.Options.Images[i]
		}
	}
	return nil
}

// fillImageURLs sets the picture links the player loads: plain for public works, signed otherwise.
// The descriptions live in spec.images (written with the script); options.images is the source list.
func (s *VideoService) fillImageURLs(p *VideoProject) {
	if p == nil || len(p.Options.Images) == 0 {
		return
	}
	signed := p.Visibility != VideoVisibilityPublic
	for i := range p.Options.Images {
		p.Options.Images[i].URL = s.mediaURL(p.ID, p.Options.Images[i].File, signed)
	}
	if p.Spec != nil {
		for i := range p.Spec.Images {
			p.Spec.Images[i].URL = s.mediaURL(p.ID, p.Spec.Images[i].File, signed)
		}
	}
}

// VideoPlayerImage is a picture as the player runtime receives it.
type VideoPlayerImage struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// PlayerImages lists a project's pictures for the player (after fillImageURLs).
func PlayerImages(spec *VideoSpec) []VideoPlayerImage {
	out := []VideoPlayerImage{}
	if spec == nil {
		return out
	}
	for _, img := range spec.Images {
		out = append(out, VideoPlayerImage{ID: img.ID, URL: img.URL, Width: img.Width, Height: img.Height})
	}
	return out
}

// specImages copies the project's pictures into a new spec, keeping descriptions the model gave.
func specImages(p *VideoProject, descs map[string]string) []VideoImage {
	if len(p.Options.Images) == 0 {
		return nil
	}
	out := make([]VideoImage, len(p.Options.Images))
	copy(out, p.Options.Images)
	for i := range out {
		out[i].URL = ""
		if d := strings.TrimSpace(descs[out[i].ID]); d != "" {
			out[i].Desc = clipRunes(d, 300)
		}
	}
	return out
}

func (s *VideoService) recentImages(userID int64, since time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, t := range s.images[userID] {
		if t.After(since) {
			n++
		}
	}
	return n
}

func (s *VideoService) noteImage(userID int64, since time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.images == nil {
		s.images = map[int64][]time.Time{}
	}
	kept := []time.Time{s.now()}
	for _, t := range s.images[userID] {
		if t.After(since) {
			kept = append(kept, t)
		}
	}
	s.images[userID] = kept
}
