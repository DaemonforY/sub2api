package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Posters: the admin page lays out an approved drawing in the browser (shihuaLayout, the same code as
// 课本诗画) and uploads one poster per aspect ratio with the words it set. The words are checked here
// before the poster is kept: only texts that may carry their words get any, and those words must be
// the text's own, character for character. The pixels come from the browser, so this guards against
// mistakes, not against an admin who wants around it.

var (
	ErrMaterialNotApproved  = infraerrors.BadRequest("MATERIAL_NOT_APPROVED", "只有审核通过的作品才能排版（Only approved artworks can be laid out）")
	ErrMaterialWordsRefOnly = infraerrors.BadRequest("MATERIAL_WORDS_REF_ONLY",
		"这篇是「仅作参考」，画上不能放原文，只能放落款（Reference-only text: no original words on the picture）")
	ErrMaterialWordsMismatch = infraerrors.BadRequest("MATERIAL_WORDS_MISMATCH",
		"排上的文字和原文对不上，请检查有没有错字、漏字（The words do not match the text）")
	ErrMaterialFilesOff = infraerrors.BadRequest("MATERIAL_FILES_OFF", "图片存储未启用（Picture storage is off）")
)

// MaterialRender is a finished poster (mat_artwork_renders).
type MaterialRender struct {
	ID        int64           `json:"id"`
	ArtworkID int64           `json:"artwork_id"`
	Aspect    string          `json:"aspect"`
	File      string          `json:"file"`
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Layout    json.RawMessage `json:"layout"`
	CreatedAt time.Time       `json:"created_at"`
	URL       string          `json:"url,omitempty"`
}

// materialRenderRepo is the poster part of MaterialRepository.
type materialRenderRepo interface {
	// SaveArtworkRender inserts or replaces the poster of an artwork in an aspect.
	SaveArtworkRender(ctx context.Context, r *MaterialRender) error
	ListArtworkRenders(ctx context.Context, artworkIDs []int64) ([]MaterialRender, error)
}

func materialRenderRel(artworkID int64, aspect, ext string) string {
	return fmt.Sprintf("artworks/%d/%d-%s.%s", artworkID/1000, artworkID, aspect, ext)
}

// MaterialByline is the only credit line a poster may carry: 作者《标题》.
func MaterialByline(author, title string) string {
	author, title = strings.TrimSpace(author), strings.TrimSpace(title)
	if title == "" {
		return author
	}
	return author + "《" + title + "》"
}

// CheckMaterialLayoutWords checks the words a poster sets. seg is nil when the drawing's segment is
// gone; then no words are allowed at all.
func CheckMaterialLayoutWords(text, byline string, seg *MaterialSegmentContext) error {
	text, byline = strings.TrimSpace(text), strings.TrimSpace(byline)
	if seg == nil {
		if text != "" || byline != "" {
			return ErrMaterialWordsMismatch
		}
		return nil
	}
	if byline != "" && byline != MaterialByline(seg.AuthorName, seg.Title) {
		return infraerrors.BadRequest("MATERIAL_BYLINE", "落款只能写「"+MaterialByline(seg.AuthorName, seg.Title)+"」（Byline must be author and title）")
	}
	if text == "" {
		return nil
	}
	if !MaterialTextOverlayAllowed(seg.LicenseStatus) {
		return ErrMaterialWordsRefOnly
	}
	source := seg.Body
	if strings.TrimSpace(source) == "" {
		source = seg.Content
	}
	if core := materialCore(text); core == "" || !strings.Contains(materialCore(source), core) {
		return ErrMaterialWordsMismatch
	}
	return nil
}

func (s *MaterialService) filesOrErr() (*shihuaFiles, error) {
	if s.queue == nil || s.queue.files == nil {
		return nil, ErrMaterialFilesOff
	}
	return s.queue.files, nil
}

// SaveRender keeps the poster of an approved drawing in one aspect, after checking its words.
func (s *MaterialService) SaveRender(ctx context.Context, artworkID int64, aspect string, layout json.RawMessage, data []byte) (*MaterialRender, error) {
	files, err := s.filesOrErr()
	if err != nil {
		return nil, err
	}
	if !shihuaAspects[aspect] {
		return nil, ErrMaterialInvalid
	}
	a, err := s.repo.GetArtwork(ctx, artworkID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrMaterialArtworkNotFound
	}
	if a.Status != MaterialArtworkSucceeded || a.ReviewStatus != MaterialReviewApproved {
		return nil, ErrMaterialNotApproved
	}
	if len(layout) == 0 || len(layout) > 64<<10 || !json.Valid(layout) {
		return nil, ErrMaterialInvalid
	}
	var words struct {
		Text   string `json:"text"`
		Byline string `json:"byline"`
	}
	if err := json.Unmarshal(layout, &words); err != nil {
		return nil, ErrMaterialInvalid
	}
	var seg *MaterialSegmentContext
	if a.SegmentID != nil {
		if seg, err = s.repo.GetSegment(ctx, *a.SegmentID); err != nil {
			return nil, err
		}
	}
	if err := CheckMaterialLayoutWords(words.Text, words.Byline, seg); err != nil {
		return nil, err
	}
	file, w, h, err := files.savePoster(func(ext string) string { return materialRenderRel(a.ID, aspect, ext) }, data, ErrMaterialInvalid)
	if err != nil {
		return nil, err
	}
	r := &MaterialRender{ArtworkID: a.ID, Aspect: aspect, File: file, Width: w, Height: h, Layout: layout}
	if err := s.repo.SaveArtworkRender(ctx, r); err != nil {
		return nil, err
	}
	r.URL = files.sign(r.File)
	return r, nil
}

// ListRenders returns the posters of the given drawings with signed links.
func (s *MaterialService) ListRenders(ctx context.Context, artworkIDs []int64) ([]MaterialRender, error) {
	if len(artworkIDs) == 0 || len(artworkIDs) > materialMaxArtworksPage {
		return nil, ErrMaterialInvalid
	}
	renders, err := s.repo.ListArtworkRenders(ctx, artworkIDs)
	if err != nil {
		return nil, err
	}
	if files, err := s.filesOrErr(); err == nil {
		for i := range renders {
			renders[i].URL = files.sign(renders[i].File)
		}
	}
	return renders, nil
}

// ExportZip writes the posters of the approved drawings among artworkIDs (all aspects when aspects is
// empty), one folder per aspect, plus a text file per drawing saying where it comes from.
func (s *MaterialService) ExportZip(ctx context.Context, out io.Writer, artworkIDs []int64, aspects []string) error {
	files, err := s.filesOrErr()
	if err != nil {
		return err
	}
	if len(artworkIDs) == 0 || len(artworkIDs) > materialMaxArtworksPage {
		return ErrMaterialInvalid
	}
	renders, err := s.repo.ListArtworkRenders(ctx, artworkIDs)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, a := range aspects {
		want[a] = true
	}
	zw := zip.NewWriter(out)
	n := 0
	for _, id := range artworkIDs {
		a, err := s.repo.GetArtwork(ctx, id)
		if err != nil {
			return err
		}
		if a == nil || a.ReviewStatus != MaterialReviewApproved {
			continue
		}
		var seg *MaterialSegmentContext
		if a.SegmentID != nil {
			if seg, err = s.repo.GetSegment(ctx, *a.SegmentID); err != nil {
				return err
			}
		}
		n++
		title := a.TextTitle
		if seg != nil {
			title = firstNonEmpty(seg.Title, title)
		}
		prefix := fmt.Sprintf("%03d-%s-%d", n, shihuaFileName(firstNonEmpty(title, "作品")), a.ID)
		wrote := false
		for _, r := range renders {
			if r.ArtworkID != a.ID || (len(want) > 0 && !want[r.Aspect]) {
				continue
			}
			data, err := files.read(r.File)
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
			wrote = true
		}
		if !wrote {
			n--
			continue
		}
		fw, err := zw.Create("说明/" + prefix + ".txt")
		if err != nil {
			return err
		}
		if _, err := io.WriteString(fw, materialExportNote(a, seg)); err != nil {
			return err
		}
	}
	return zw.Close()
}

// materialExportNote says what a poster shows, where the words come from and that the picture is AI.
func materialExportNote(a *MaterialArtwork, seg *MaterialSegmentContext) string {
	var b strings.Builder
	if seg != nil {
		fmt.Fprintf(&b, "篇名：%s\n作者：%s %s\n片段：%s\n", seg.Title, seg.Dynasty, seg.AuthorName, seg.Label)
		license := map[string]string{MaterialLicensePublicDomain: "公有领域", MaterialLicenseOriginal: "原创", MaterialLicenseLicensed: "已授权",
			MaterialLicenseReference: "仅作参考（画上不放原文）"}[seg.LicenseStatus]
		fmt.Fprintf(&b, "版权状态：%s\n", license)
		if MaterialTextOverlayAllowed(seg.LicenseStatus) && seg.Content != "" {
			fmt.Fprintf(&b, "原文：%s\n", seg.Content)
		}
		if seg.SceneDesc != "" {
			fmt.Fprintf(&b, "画面：%s\n", seg.SceneDesc)
		}
	}
	fmt.Fprintf(&b, "画风：%s\n模型：%s（%s）\n评分：%d\n\n提示词：\n%s\n\n配图由 AI 生成。\n", a.StyleName, a.Model, a.Size, a.Rating, a.PromptFinal)
	return b.String()
}
