package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // decoders for the drawings and uploaded posters
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// shihuaFiles keeps the 课本诗画 pictures under dir/works/<work id>/ and hands out signed links
// (/api/v1/shihua/files/<path>?e=&s=) so the admin page can show them in <img> and <canvas>. The
// signing key is random per process: links stop working after a restart, and the page asks again.
type shihuaFiles struct {
	dir   string
	key   []byte
	now   func() time.Time
	route string         // the file route links point at
	relRe *regexp.Regexp // the paths it serves
}

const (
	shihuaFileRoute     = "/api/v1/shihua/files/"
	shihuaLinkLifetime  = 12 * time.Hour
	shihuaThumbWidth    = 480
	shihuaMaxRenderSize = 30 << 20
	shihuaMaxRenderEdge = 6000
)

var shihuaRelRe = regexp.MustCompile(`^works/[0-9]+/[0-9]+(-thumb|-2x|-[0-9x]+)?\.(png|jpg|webp)$`)

func newShihuaFiles(dir string) *shihuaFiles {
	return newSignedFiles(dir, shihuaFileRoute, shihuaRelRe)
}

// newSignedFiles is a picture store with its own signing key, route and allowed paths (the 素材库
// uses one too, so links of one feature never open the other's files).
func newSignedFiles(dir, route string, relRe *regexp.Regexp) *shihuaFiles {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return &shihuaFiles{dir: dir, key: key, now: time.Now, route: route, relRe: relRe}
}

func (f *shihuaFiles) mac(rel, exp string) string {
	m := hmac.New(sha256.New, f.key)
	_, _ = io.WriteString(m, rel+"|"+exp)
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// sign returns a link to rel valid for at least 12 hours (the expiry is rounded to the hour, so the
// same picture keeps the same link and stays cached).
func (f *shihuaFiles) sign(rel string) string {
	if rel == "" {
		return ""
	}
	exp := strconv.FormatInt(f.now().Add(shihuaLinkLifetime).Truncate(time.Hour).Add(time.Hour).Unix(), 10)
	return f.route + rel + "?e=" + exp + "&s=" + f.mac(rel, exp)
}

// Open checks a signed link and returns the file's path on disk.
func (f *shihuaFiles) Open(rel, exp, sig string) (string, bool) {
	rel = strings.TrimPrefix(rel, "/")
	if !f.relRe.MatchString(rel) || !hmac.Equal([]byte(sig), []byte(f.mac(rel, exp))) {
		return "", false
	}
	if e, err := strconv.ParseInt(exp, 10, 64); err != nil || f.now().Unix() > e {
		return "", false
	}
	return filepath.Join(f.dir, filepath.FromSlash(rel)), true
}

// signImage fills in what the admin page needs: the signed links and whether the model drew a title.
func (f *shihuaFiles) signImage(img *ShihuaImage) {
	img.TitleDrawn = strings.Contains(img.Prompt, shihuaTitleMarker)
	if img.File == "" {
		return
	}
	img.URL = f.sign(img.File)
	img.ThumbURL = f.sign(shihuaThumbRel(img.WorkID, img.ID))
	img.UpscaledURL = f.sign(img.UpscaledFile)
}

func shihuaThumbRel(workID, imageID int64) string {
	return fmt.Sprintf("works/%d/%d-thumb.jpg", workID, imageID)
}

func (f *shihuaFiles) write(rel string, data []byte) error {
	path := filepath.Join(f.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *shihuaFiles) read(rel string) ([]byte, error) {
	if !f.relRe.MatchString(rel) {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(filepath.Join(f.dir, filepath.FromSlash(rel)))
}

func (f *shihuaFiles) removeWork(workID int64) {
	_ = os.RemoveAll(filepath.Join(f.dir, "works", strconv.FormatInt(workID, 10)))
}

func shihuaExt(data []byte) string {
	switch http.DetectContentType(data) {
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	}
	return "png"
}

// saveDrawing stores the model's picture as returned plus a small JPEG for the pick grid.
func (f *shihuaFiles) saveDrawing(workID, imageID int64, data []byte) (string, int, int, error) {
	return f.saveWithThumb(fmt.Sprintf("works/%d/%d.%s", workID, imageID, shihuaExt(data)), shihuaThumbRel(workID, imageID), data)
}

// saveWithThumb stores a picture at rel and a small JPEG of it at thumbRel; it returns rel and the
// picture's size.
func (f *shihuaFiles) saveWithThumb(rel, thumbRel string, data []byte) (string, int, int, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, err
	}
	if err := f.write(rel, data); err != nil {
		return "", 0, 0, err
	}
	b := img.Bounds()
	tw := min(shihuaThumbWidth, b.Dx())
	th := b.Dy() * tw / max(b.Dx(), 1)
	thumb := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, b, xdraw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 82}); err != nil {
		return "", 0, 0, err
	}
	if err := f.write(thumbRel, buf.Bytes()); err != nil {
		return "", 0, 0, err
	}
	return rel, b.Dx(), b.Dy(), nil
}

func (f *shihuaFiles) saveUpscaled(workID, imageID int64, data []byte) (string, error) {
	rel := fmt.Sprintf("works/%d/%d-2x.%s", workID, imageID, shihuaExt(data))
	return rel, f.write(rel, data)
}

// saveRender stores a poster uploaded by the admin page (JPEG or PNG).
func (f *shihuaFiles) saveRender(workID, imageID int64, aspect string, data []byte) (string, int, int, error) {
	return f.savePoster(func(ext string) string { return fmt.Sprintf("works/%d/%d-%s.%s", workID, imageID, aspect, ext) }, data, ErrShihuaInvalid)
}

// savePoster checks an uploaded poster (JPEG or PNG, size and edge limits) and stores it at the path
// relFor builds from its extension; bad uploads return invalid.
func (f *shihuaFiles) savePoster(relFor func(ext string) string, data []byte, invalid error) (string, int, int, error) {
	if len(data) == 0 || len(data) > shihuaMaxRenderSize {
		return "", 0, 0, invalid
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width > shihuaMaxRenderEdge || cfg.Height > shihuaMaxRenderEdge {
		return "", 0, 0, invalid
	}
	rel := relFor(shihuaExt(data))
	return rel, cfg.Width, cfg.Height, f.write(rel, data)
}

// OpenFile checks a signed link (for the file route).
func (s *ShihuaService) OpenFile(rel, exp, sig string) (string, bool) {
	return s.files.Open(rel, exp, sig)
}

var shihuaAspectNames = map[string]string{"2x3": "2比3", "3x4": "小红书3比4", "9x16": "竖版9比16", "1x1": "方图", "16x9": "横版16比9", "235x100": "公众号头图"}

// ExportZip writes the posters of the given works (all aspects when aspects is empty) and a text file
// per work with the poster text, the full text and where it comes from.
func (s *ShihuaService) ExportZip(ctx context.Context, out io.Writer, workIDs []int64, aspects []string) error {
	works, err := s.repo.ListWorks(ctx, ShihuaWorkQuery{IDs: workIDs})
	if err != nil {
		return err
	}
	renders, err := s.repo.ListRenders(ctx, workIDs)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, a := range aspects {
		want[a] = true
	}
	zw := zip.NewWriter(out)
	for i := range works {
		w := &works[i]
		prefix := fmt.Sprintf("%03d-%s", i+1, shihuaFileName(w.Title))
		for _, r := range renders {
			if r.WorkID != w.ID || (len(want) > 0 && !want[r.Aspect]) || w.PickedImageID == nil || r.ImageID != *w.PickedImageID {
				continue
			}
			data, err := s.files.read(r.File)
			if err != nil {
				continue
			}
			name := fmt.Sprintf("%s/%s-%s%s", r.Aspect, prefix, shihuaAspectNames[r.Aspect], filepath.Ext(r.File))
			fw, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store, Modified: r.CreatedAt})
			if err != nil {
				return err
			}
			if _, err := fw.Write(data); err != nil {
				return err
			}
		}
		fw, err := zw.Create("文字/" + prefix + ".txt")
		if err != nil {
			return err
		}
		if _, err := io.WriteString(fw, shihuaTextFile(w)); err != nil {
			return err
		}
	}
	return zw.Close()
}

var shihuaUnsafeName = regexp.MustCompile(`[\\/:*?"<>|\s]+`)

func shihuaFileName(s string) string {
	return shihuaUnsafeName.ReplaceAllString(s, "_")
}

func shihuaTextFile(w *ShihuaWork) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", w.Title)
	if w.Author != "" || w.Dynasty != "" {
		fmt.Fprintf(&b, "%s %s\n", strings.TrimSpace(w.Dynasty), w.Author)
	}
	if w.Grade != "" {
		fmt.Fprintf(&b, "出处：统编版 %s\n", w.Grade)
	}
	fmt.Fprintf(&b, "\n【海报上的正文】\n%s\n", w.PosterText())
	if w.DisplayText != "" {
		fmt.Fprintf(&b, "\n【全文】\n%s\n", w.FullText)
		if w.ExcerptNote != "" {
			fmt.Fprintf(&b, "\n节选说明：%s\n", w.ExcerptNote)
		}
	}
	if w.FamousLines != "" {
		fmt.Fprintf(&b, "\n【名句】\n%s\n", w.FamousLines)
	}
	if w.VariantNote != "" {
		fmt.Fprintf(&b, "\n版本说明：%s\n", w.VariantNote)
	}
	_, _ = b.WriteString("\n图片由 AI 生成（gpt-image-2），正文为程序排版并经人工校对。\n")
	return b.String()
}
