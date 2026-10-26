package service

import (
	"bytes"
	"context"
	"image"
	"io"
	"net/http"
	"strings"
	"time"
)

// 图片墙: every archived image newest first, filterable by user, date and model, shown as small
// JPEG thumbnails. A thumbnail is made on first view and kept beside the original in the bucket
// (<key>.thumb.jpg), so the account purge (prefix delete) removes it and retention deletes it
// with its image.

const (
	archiveImagesDefault     = 60
	archiveImagesMax         = 120
	archiveThumbEdge         = 480
	archiveThumbQuality      = 80
	archiveThumbSuffix       = ".thumb.jpg"
	archiveThumbMinBytes     = 120 << 10
	archiveImageMaxReadBytes = 48 << 20
	archiveImageModelsDays   = 90
)

// archiveThumbSlots bounds how many thumbnails are decoded at once (a 1536×1024 PNG decodes to
// ~6 MB and a page asks for 60 of them together).
var archiveThumbSlots = make(chan struct{}, 3)

// ArchiveImageFilter selects images, newest first (BeforeID is the cursor: ids strictly below).
type ArchiveImageFilter struct {
	UserID   int64      `json:"user_id,omitempty"`
	Q        string     `json:"q,omitempty"`
	Model    string     `json:"model,omitempty"`
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
	BeforeID int64      `json:"before_id,omitempty"`
	Limit    int        `json:"limit"`
}

// ArchiveImageItem is one tile of the image wall with the turn it came from.
type ArchiveImageItem struct {
	ID              int64     `json:"id"`
	TurnID          int64     `json:"turn_id"`
	UserID          int64     `json:"user_id"`
	Email           string    `json:"email"`
	Mime            string    `json:"mime"`
	Bytes           int       `json:"bytes"`
	CreatedAt       time.Time `json:"created_at"`
	Model           string    `json:"model"`
	InboundEndpoint string    `json:"inbound_endpoint"`
	SessionKey      string    `json:"session_key"`
	RequestID       string    `json:"request_id"`
	// Prompt is the turn's preview: the head of the latest user prompt (200 characters).
	Prompt string `json:"prompt"`
}

// ArchiveImagePage is a page of the image wall. Models (the models that drew images lately) is
// only filled on the first page, for the filter.
type ArchiveImagePage struct {
	Items        []ArchiveImageItem `json:"items"`
	NextBeforeID *int64             `json:"next_before_id"`
	Models       []string           `json:"models,omitempty"`
}

// ListImages returns the image wall (audited as archive_view, op "images").
func (s *ConversationArchiveService) ListImages(ctx context.Context, adminID int64, f ArchiveImageFilter, ip, ua string) (*ArchiveImagePage, error) {
	f.Q = strings.TrimSpace(f.Q)
	f.Model = strings.TrimSpace(f.Model)
	if f.Limit <= 0 {
		f.Limit = archiveImagesDefault
	}
	if f.Limit > archiveImagesMax {
		f.Limit = archiveImagesMax
	}
	if f.UserID < 0 {
		f.UserID = 0
	}
	if f.BeforeID < 0 {
		f.BeforeID = 0
	}
	unresolved := false
	if f.UserID == 0 && f.Q != "" {
		id, err := s.repo.ResolveUserID(ctx, f.Q)
		if err != nil {
			return nil, err
		}
		f.UserID = id
		unresolved = id == 0
	}
	if f.UserID == 0 && f.From == nil {
		from := s.now().AddDate(0, 0, -archiveDefaultBrowseDays)
		f.From = &from
	}
	if err := s.audit(ctx, adminID, f.UserID, map[string]any{"op": "images", "filter": f}, ip, ua); err != nil {
		return nil, err
	}
	page := &ArchiveImagePage{Items: []ArchiveImageItem{}}
	if f.BeforeID == 0 {
		models, err := s.repo.ListImageModels(ctx, s.now().AddDate(0, 0, -archiveImageModelsDays))
		if err != nil {
			return nil, err
		}
		page.Models = models
		if page.Models == nil {
			page.Models = []string{}
		}
	}
	if unresolved {
		return page, nil
	}
	items, err := s.repo.ListImages(ctx, f)
	if err != nil {
		return nil, err
	}
	if items != nil {
		page.Items = items
	}
	if len(page.Items) == f.Limit {
		last := page.Items[len(page.Items)-1].ID
		page.NextBeforeID = &last
	}
	return page, nil
}

// OpenImageThumb returns a small JPEG of an archived image for the wall (not audited, like
// OpenImage: the list that shows it is). Small images and GIFs come back as they are; an image
// that cannot be decoded comes back as the original.
func (s *ConversationArchiveService) OpenImageThumb(ctx context.Context, id int64) ([]byte, string, error) {
	if id <= 0 {
		return nil, "", ErrArchiveImageNotFound
	}
	img, err := s.repo.GetImage(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if img == nil {
		return nil, "", ErrArchiveImageNotFound
	}
	rt := s.currentRuntime()
	if rt == nil || rt.store == nil {
		return nil, "", ErrArchiveNotConfigured
	}
	thumbKey := img.ObjectKey + archiveThumbSuffix
	if data, err := readArchiveObject(ctx, rt.store, thumbKey, archiveImageMaxReadBytes); err == nil && len(data) > 0 {
		return data, "image/jpeg", nil
	}

	select {
	case archiveThumbSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, "", ctx.Err()
	}
	defer func() { <-archiveThumbSlots }()

	data, err := readArchiveObject(ctx, rt.store, img.ObjectKey, archiveImageMaxReadBytes)
	if err != nil {
		return nil, "", archiveStorageError("CONVERSATION_ARCHIVE_IMAGE_FETCH_FAILED", "读取图片失败", "Failed to fetch the image", err)
	}
	mime := strings.TrimSpace(img.Mime)
	if !strings.HasPrefix(mime, "image/") {
		mime = http.DetectContentType(data)
	}
	if len(data) <= archiveThumbMinBytes || mime == "image/gif" {
		return data, mime, nil
	}
	thumb, ok := makeArchiveThumb(data)
	if !ok {
		return data, mime, nil
	}
	// Best effort: a failed write only means the next view makes it again.
	_ = rt.store.PutObject(ctx, thumbKey, thumb, "image/jpeg", "", rt.settings.SSE)
	return thumb, "image/jpeg", nil
}

func readArchiveObject(ctx context.Context, store ArchiveObjectStore, key string, limit int64) ([]byte, error) {
	rc, _, err := store.GetObject(ctx, key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(io.LimitReader(rc, limit))
}

func makeArchiveThumb(data []byte) ([]byte, bool) {
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, false
	}
	if edge := max(w, h); edge > archiveThumbEdge {
		w, h = max(1, w*archiveThumbEdge/edge), max(1, h*archiveThumbEdge/edge)
	}
	var out bytes.Buffer
	if err := encodeJPEGTo(&out, scaleTo(src, w, h), archiveThumbQuality); err != nil {
		return nil, false
	}
	return out.Bytes(), true
}

// archiveImageThumbKeys lists the thumbnails that may exist beside the given image keys.
func archiveImageThumbKeys(imageKeys []string) []string {
	out := make([]string, 0, len(imageKeys))
	for _, k := range imageKeys {
		out = append(out, k+archiveThumbSuffix)
	}
	return out
}
