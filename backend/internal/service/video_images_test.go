//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func testPicture(t *testing.T, w, h int, alpha bool, format string) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if alpha && x < w/2 {
				a = 0
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 200, A: a})
		}
	}
	var buf bytes.Buffer
	if format == "jpeg" {
		require.NoError(t, jpeg.Encode(&buf, img, nil))
	} else {
		require.NoError(t, png.Encode(&buf, img))
	}
	return buf.Bytes()
}

func TestNormalizeVideoImage(t *testing.T) {
	data, ext, w, h, err := normalizeVideoImage(bytes.NewReader(testPicture(t, 3000, 1500, false, "png")), 2048)
	require.NoError(t, err)
	require.Equal(t, ".jpg", ext, "opaque pictures become JPEG")
	require.Equal(t, 2048, w)
	require.Equal(t, 1024, h)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.Equal(t, 2048, cfg.Width)

	_, ext, w, _, err = normalizeVideoImage(bytes.NewReader(testPicture(t, 100, 50, true, "png")), 2048)
	require.NoError(t, err)
	require.Equal(t, ".png", ext, "transparency is kept")
	require.Equal(t, 100, w)

	_, _, _, _, err = normalizeVideoImage(bytes.NewReader([]byte("<svg/>")), 2048)
	require.ErrorIs(t, err, ErrVideoImageType)
}

func TestLearnMessageWithImages(t *testing.T) {
	b, err := json.Marshal(LearnMessage{Role: "user", Content: "看图", Images: []string{"data:image/png;base64,AAA"}})
	require.NoError(t, err)
	require.JSONEq(t, `{"role":"user","content":[{"type":"text","text":"看图"},{"type":"image_url","image_url":{"url":"data:image/png;base64,AAA"}}]}`, string(b))
	b, err = json.Marshal(LearnMessage{Role: "user", Content: "hi"})
	require.NoError(t, err)
	require.JSONEq(t, `{"role":"user","content":"hi"}`, string(b))
	var in LearnMessage
	require.NoError(t, json.Unmarshal([]byte(`{"role":"user","content":"x","images":["data:x"]}`), &in))
	require.Empty(t, in.Images, "clients cannot set images")
}

func TestVideoWithPictures(t *testing.T) {
	model := &fakeModel{broken: map[string]int{}}
	var mu sync.Mutex
	var scriptImages, sceneImages int
	var sceneSystem string
	stream := func(ctx context.Context, key, m string, msgs []LearnMessage, n int, onDelta func(string) error) (*LearnTutorResult, error) {
		last := msgs[len(msgs)-1]
		mu.Lock()
		switch {
		case strings.Contains(msgs[0].Content, "scriptwriter"):
			scriptImages = len(last.Images)
			mu.Unlock()
			ans := "```json\n" + `{"title":"杯子","scenes":[{"title":"一","narration":"第一段旁白。","images":["img1","img9"],"visual":"img1 放大"},{"title":"二","narration":"第二段旁白。","images":[],"visual":"v2"}],"image_notes":[{"id":"img1","desc":"白色杯子居中"},{"id":"img2","desc":"红色盒子"}]}` + "\n```"
			_ = onDelta(ans)
			return &LearnTutorResult{PromptTokens: 10, CompletionTokens: 5}, nil
		case strings.Contains(msgs[0].Content, "motion designer") && strings.Contains(msgs[1].Content, "Scene s1"):
			sceneImages, sceneSystem = len(msgs[1].Images), msgs[0].Content
		}
		mu.Unlock()
		return model.stream(ctx, key, m, msgs, n, onDelta)
	}
	dir := t.TempDir()
	svc := newVideoService(newFakeVideoRepo(), &fakeTTS{}, stream, dir)
	ctx := context.Background()

	var tokens []string
	for _, pic := range [][]byte{testPicture(t, 400, 300, false, "jpeg"), testPicture(t, 200, 200, true, "png")} {
		st, err := svc.StageImage(ctx, videoKeyA, VideoUploadFile{Name: "a", Size: int64(len(pic)), Reader: bytes.NewReader(pic)})
		require.NoError(t, err)
		tokens = append(tokens, st.Token)
	}
	_, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "介绍杯子", Images: []string{"../../etc/passwd"}})
	require.ErrorIs(t, err, ErrVideoImageGone)
	_, err = svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "x", Images: make([]string, VideoImagesMax+1)})
	require.ErrorIs(t, err, ErrVideoImageCount)
	// Another user cannot use someone else's staged picture.
	_, err = svc.Create(ctx, &APIKey{ID: 2, UserID: 8, Key: "sk-b"}, VideoCreateInput{Mode: "film", Prompt: "x", Images: tokens[:1]})
	require.ErrorIs(t, err, ErrVideoImageGone)

	p, err := svc.Create(ctx, videoKeyA, VideoCreateInput{Mode: "film", Prompt: "介绍杯子", Images: tokens,
		Options: VideoOptions{Images: []VideoImage{{ID: "img1", File: "../../secret"}}}})
	require.NoError(t, err)
	svc.Wait()
	got, err := svc.Get(ctx, 7, p.ID)
	require.NoError(t, err)
	require.Equal(t, VideoStatusReady, got.Status, got.Error)
	require.Len(t, got.Options.Images, 2)
	require.Equal(t, "img1", got.Options.Images[0].ID)
	require.Equal(t, "image/png", got.Options.Images[1].Mime)
	require.FileExists(t, filepath.Join(dir, p.ID, got.Options.Images[0].File))
	require.Equal(t, 2, scriptImages, "the script is written looking at the pictures")
	require.Equal(t, 1, sceneImages, "scene 1 sees the picture it shows")
	require.Contains(t, sceneSystem, "S.image(")
	require.Len(t, got.Spec.Images, 2)
	require.Equal(t, "白色杯子居中", got.Spec.Images[0].Desc)
	require.Equal(t, []string{"img1"}, got.Spec.Scenes[0].Images, "unknown ids are dropped")
	require.Empty(t, got.Spec.Scenes[1].Images)

	// The player gets signed links for a private work; the file is served with them only.
	img := got.Spec.Images[0]
	require.Contains(t, img.URL, "?exp=")
	require.Len(t, PlayerImages(got.Spec), 2)
	exp, sig := linkQuery(t, img.URL)
	m, err := svc.MediaFile(ctx, p.ID, img.File, exp, sig)
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", m.Mime)
	_, err = svc.MediaFile(ctx, p.ID, img.File, "", "")
	require.ErrorIs(t, err, ErrVideoNotFound)
	_, err = svc.MediaFile(ctx, p.ID, "img-nothere.jpg", exp, sig)
	require.ErrorIs(t, err, ErrVideoNotFound)

	// Deleting the project removes its pictures.
	require.NoError(t, svc.Delete(ctx, 7, p.ID))
	_, err = os.Stat(filepath.Join(dir, p.ID))
	require.True(t, os.IsNotExist(err))
}
