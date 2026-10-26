//go:build unit

package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConversationArchive_ListImages(t *testing.T) {
	h := newArchiveHarness(t)
	ctx := context.Background()
	h.repo.wall = []ArchiveImageItem{{ID: 9, UserID: 45, Prompt: "a cat"}, {ID: 4, UserID: 45}}

	// Audited before anything is read; without a user the range defaults to the last 30 days.
	before := len(h.repo.access)
	page, err := h.svc.ListImages(ctx, 9, ArchiveImageFilter{Model: " gpt-image-2 ", Limit: 2}, "ip", "ua")
	require.NoError(t, err)
	require.Len(t, h.repo.access, before+1)
	require.Equal(t, ContentLogActionArchiveView, h.repo.access[before].Action)
	require.Contains(t, h.repo.access[before].Detail, `"op":"images"`)
	require.Equal(t, "gpt-image-2", h.repo.lastIF.Model)
	require.NotNil(t, h.repo.lastIF.From)
	require.WithinDuration(t, time.Now().AddDate(0, 0, -30), *h.repo.lastIF.From, time.Minute)
	require.Len(t, page.Items, 2)
	require.Equal(t, int64(4), *page.NextBeforeID, "a full page continues below its last id")
	require.Equal(t, []string{"gpt-image-2"}, page.Models)

	// Later pages skip the model list; a short page ends; the limit is capped.
	page, err = h.svc.ListImages(ctx, 9, ArchiveImageFilter{BeforeID: 4, Limit: 1000}, "", "")
	require.NoError(t, err)
	require.Nil(t, page.Models)
	require.Nil(t, page.NextBeforeID)
	require.Equal(t, archiveImagesMax, h.repo.lastIF.Limit)

	// q resolves to a user (then no default range); an unknown q returns nothing without a query.
	h.repo.resolve["a@b.c"] = 45
	_, err = h.svc.ListImages(ctx, 9, ArchiveImageFilter{Q: "a@b.c"}, "", "")
	require.NoError(t, err)
	require.Equal(t, int64(45), h.repo.lastIF.UserID)
	require.Nil(t, h.repo.lastIF.From)
	require.Equal(t, archiveImagesDefault, h.repo.lastIF.Limit)
	calls := h.repo.wallCalls
	page, err = h.svc.ListImages(ctx, 9, ArchiveImageFilter{Q: "nobody"}, "", "")
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Equal(t, calls, h.repo.wallCalls)
	require.Equal(t, int64(0), h.repo.access[len(h.repo.access)-1].TargetUserID)

	// No audit row, no data.
	h.repo.accessErr = errors.New("db down")
	_, err = h.svc.ListImages(ctx, 9, ArchiveImageFilter{}, "", "")
	require.Error(t, err)
	require.Equal(t, calls, h.repo.wallCalls)
}

// noisyPNG is a PNG too big to send as its own thumbnail (noise doesn't compress).
func noisyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewSource(1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	require.Greater(t, buf.Len(), archiveThumbMinBytes)
	return buf.Bytes()
}

func TestConversationArchive_OpenImageThumb(t *testing.T) {
	h := newArchiveHarness(t)
	ctx := context.Background()
	h.repo.images = []ArchiveImage{
		{ID: 1, TurnID: 1, UserID: 45, ObjectKey: "archive/img/45/big.png", Mime: "image/png"},
		{ID: 2, TurnID: 1, UserID: 45, ObjectKey: "archive/img/45/small.png", Mime: "image/png"},
		{ID: 3, TurnID: 1, UserID: 45, ObjectKey: "archive/img/45/gone.png", Mime: "image/png"},
	}
	_, _, err := h.svc.OpenImageThumb(ctx, 1)
	require.ErrorIs(t, err, ErrArchiveNotConfigured)

	h.enable(t)
	big := noisyPNG(t, 960, 640)
	require.NoError(t, h.store.PutObject(ctx, "archive/img/45/big.png", big, "image/png", "", true))
	require.NoError(t, h.store.PutObject(ctx, "archive/img/45/small.png", archivePNG(), "image/png", "", true))

	// Made on first view: a JPEG at most 480 px, stored beside the original with the bucket's SSE.
	data, mime, err := h.svc.OpenImageThumb(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", mime)
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, 480, cfg.Width)
	require.Equal(t, 320, cfg.Height)
	stored := h.store.object("archive/img/45/big.png" + archiveThumbSuffix)
	require.Equal(t, data, stored.body)
	require.True(t, stored.sse)

	// Second view reads the stored thumbnail (the original isn't needed any more).
	h.store.mu.Lock()
	delete(h.store.objects, "archive/img/45/big.png")
	h.store.mu.Unlock()
	again, _, err := h.svc.OpenImageThumb(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, data, again)

	// Small images come back as they are, without a stored copy.
	data, mime, err = h.svc.OpenImageThumb(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, archivePNG(), data)
	require.Equal(t, "image/png", mime)
	require.Empty(t, h.store.object("archive/img/45/small.png"+archiveThumbSuffix).body)

	_, _, err = h.svc.OpenImageThumb(ctx, 3)
	require.Error(t, err, "original missing from the bucket")
	_, _, err = h.svc.OpenImageThumb(ctx, 99)
	require.ErrorIs(t, err, ErrArchiveImageNotFound)
	_, _, err = h.svc.OpenImageThumb(ctx, 0)
	require.ErrorIs(t, err, ErrArchiveImageNotFound)
}
