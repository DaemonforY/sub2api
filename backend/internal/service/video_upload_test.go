//go:build unit

package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The files in testdata/video_upload were made with ffmpeg (64×36 test pattern): h264.mp4 2.5 s,
// rotated.mp4 the same with a 90° display matrix, frag.mp4 fragmented 2 s, mpeg4.mp4 (MPEG-4 Part 2
// codec), mov.mov (QuickTime), audio.m4a (AAC only), vp9.webm 3 s, piped.webm / live.webm VP8 3 s
// written to a pipe (no duration in the header; live.webm has clusters of unknown size).

func uploadFixture(t *testing.T, name string) VideoUploadFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "video_upload", name))
	require.NoError(t, err)
	return VideoUploadFile{Name: name, Size: int64(len(data)), Reader: bytes.NewReader(data)}
}

func bytesFile(name string, data []byte) VideoUploadFile {
	return VideoUploadFile{Name: name, Size: int64(len(data)), Reader: bytes.NewReader(data)}
}

func TestProbeVideoMedia(t *testing.T) {
	cases := []struct {
		file           string
		mime, codec    string
		duration       float64
		width, height  int
		approxDuration bool
	}{
		{file: "h264.mp4", mime: "video/mp4", codec: "avc1", duration: 2.5, width: 64, height: 36},
		{file: "rotated.mp4", mime: "video/mp4", codec: "avc1", duration: 2.5, width: 36, height: 64},
		{file: "frag.mp4", mime: "video/mp4", codec: "avc1", duration: 2, width: 64, height: 36, approxDuration: true},
		{file: "vp9.webm", mime: "video/webm", codec: "V_VP9", duration: 3, width: 64, height: 36},
		{file: "piped.webm", mime: "video/webm", codec: "V_VP8", duration: 3, width: 64, height: 36, approxDuration: true},
		{file: "live.webm", mime: "video/webm", codec: "V_VP8", duration: 3, width: 64, height: 36, approxDuration: true},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			f := uploadFixture(t, tc.file)
			info, err := probeVideoMedia(f.Reader, f.Size)
			require.NoError(t, err)
			require.Equal(t, tc.mime, info.Mime)
			require.Equal(t, tc.codec, info.Codec)
			require.Equal(t, tc.width, info.Width)
			require.Equal(t, tc.height, info.Height)
			if tc.approxDuration {
				// Without a header duration it is the last frame's time (one frame short).
				require.InDelta(t, tc.duration, info.Duration, 0.15)
			} else {
				require.InDelta(t, tc.duration, info.Duration, 0.01)
			}
		})
	}
}

func TestCheckVideoUploadRejects(t *testing.T) {
	cases := map[string]struct {
		file VideoUploadFile
		want error
	}{
		"quicktime":    {uploadFixture(t, "mov.mov"), ErrVideoUploadMOV},
		"audio only":   {uploadFixture(t, "audio.m4a"), ErrVideoUploadNoVideo},
		"mpeg4 codec":  {uploadFixture(t, "mpeg4.mp4"), ErrVideoUploadCodec},
		"png renamed":  {bytesFile("x.mp4", []byte("\x89PNG\r\n\x1a\n0000000000000000")), ErrVideoUploadType},
		"html as svg":  {bytesFile("x.svg", []byte("<html><script>alert(1)</script></html>")), ErrVideoUploadType},
		"empty":        {bytesFile("x.mp4", nil), ErrVideoUploadType},
		"broken svg":   {bytesFile("x.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect></svg>`)), ErrVideoUploadSVG},
		"svg entities": {bytesFile("x.svg", []byte(`<!DOCTYPE svg [<!ENTITY x "y">]><svg xmlns="http://www.w3.org/2000/svg">&x;</svg>`)), ErrVideoUploadSVG},
		"svg too big":  {bytesFile("x.svg", append([]byte(`<svg>`), make([]byte, VideoUploadMaxSVGBytes)...)), ErrVideoUploadSVGBig},
		"truncated mp4": {func() VideoUploadFile {
			f := uploadFixture(t, "h264.mp4")
			data := make([]byte, 600)
			_, _ = f.Reader.ReadAt(data, 0)
			return bytesFile("x.mp4", data)
		}(), ErrVideoUploadBroken},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, _, err := checkVideoUpload(tc.file)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestCheckVideoUploadTooLong(t *testing.T) {
	// h264.mp4 with mvhd's duration patched to 200 s (timescale 1000).
	f := uploadFixture(t, "h264.mp4")
	data := make([]byte, f.Size)
	_, _ = f.Reader.ReadAt(data, 0)
	i := bytes.Index(data, []byte("mvhd"))
	require.Positive(t, i)
	body := data[i+4:]
	require.Equal(t, byte(0), body[0])
	ts := uint32(body[12])<<24 | uint32(body[13])<<16 | uint32(body[14])<<8 | uint32(body[15])
	dur := ts * 200
	body[16], body[17], body[18], body[19] = byte(dur>>24), byte(dur>>16), byte(dur>>8), byte(dur)
	_, _, _, err := checkVideoUpload(bytesFile("long.mp4", data))
	require.ErrorIs(t, err, ErrVideoUploadTooLong)
}

func TestSanitizeSVG(t *testing.T) {
	in := `<?xml version="1.0"?>
<!-- made by hand -->
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape"
     viewBox="0 0 400 300" onload="alert(1)" inkscape:version="1.0">
  <style>@import url(https://evil.example/x.css); .a { fill: url(#g); background: url("https://evil.example/t.png"); animation: spin 2s infinite }
  @keyframes spin { to { transform: rotate(360deg) } } .b { b\61 ckground: u\72l(https://evil.example/) }</style>
  <script>alert(1)</script>
  <defs><linearGradient id="g"><stop offset="0" stop-color="red"/></linearGradient></defs>
  <rect class="a" width="10" height="10" onclick="alert(1)" fill="url(https://evil.example/#p)" style="stroke:url('http://evil.example/s')">
    <animate attributeName="x" from="0" to="100" dur="2s" repeatCount="indefinite"/>
  </rect>
  <set attributeName="href" to="javascript:alert(1)"/>
  <animate attributeName="onclick" to="alert(1)"/>
  <a xlink:href="javascript:alert(1)"><text>link</text></a>
  <a href="  java&#9;script:alert(1)"><text>tab</text></a>
  <use href="#g"/>
  <use xlink:href="https://evil.example/sprite.svg#icon"/>
  <image href="https://evil.example/x.png" width="10" height="10"/>
  <image href="data:image/png;base64,iVBORw0KGgo=" width="10" height="10"/>
  <image href="data:image/svg+xml;base64,PHN2Zz4=" width="10" height="10"/>
  <foreignObject><div xmlns="http://www.w3.org/1999/xhtml"><iframe src="https://evil.example"></iframe></div></foreignObject>
  <iframe src="https://evil.example"/>
  <inkscape:thing/>
  <text x="5" y="5">a &lt; b &amp; c</text>
</svg>`
	out, info, err := sanitizeSVG([]byte(in))
	require.NoError(t, err)
	s := string(out)
	for _, bad := range []string{"script", "onload", "onclick", "evil.example", "javascript", "foreignObject", "iframe", "inkscape", "@import", "svg+xml;base64", "DOCTYPE", `\`} {
		require.NotContains(t, strings.ToLower(s), strings.ToLower(bad), bad)
	}
	for _, good := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 400 300">`,
		`fill: url(#g)`, `@keyframes spin`, `<animate attributeName="x"`, `<use href="#g"></use>`,
		`href="data:image/png;base64,iVBORw0KGgo="`, `a &lt; b &amp; c`, `<stop offset="0" stop-color="red"></stop>`,
		`<text>link</text>`,
	} {
		require.Contains(t, s, good)
	}
	require.Equal(t, svgInfo{Width: 1920, Height: 1440}, info)

	// The cleaned document parses again to the same thing.
	again, _, err := sanitizeSVG(out)
	require.NoError(t, err)
	require.Equal(t, s, string(again))
}

func TestSanitizeSVGRejects(t *testing.T) {
	for name, in := range map[string]string{
		"html root":      `<html xmlns="http://www.w3.org/1999/xhtml"><body/></html>`,
		"foreign root":   `<svg xmlns="http://example.com/not-svg"></svg>`,
		"two roots":      `<svg xmlns="http://www.w3.org/2000/svg"></svg><svg xmlns="http://www.w3.org/2000/svg"></svg>`,
		"unclosed":       `<svg xmlns="http://www.w3.org/2000/svg"><g>`,
		"unknown entity": `<svg xmlns="http://www.w3.org/2000/svg"><text>&lol;</text></svg>`,
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := sanitizeSVG([]byte(in))
			require.Error(t, err)
		})
	}
	deep := strings.Repeat("<g>", svgMaxDepth+2) + strings.Repeat("</g>", svgMaxDepth+2)
	_, _, err := sanitizeSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg">` + deep + `</svg>`))
	require.ErrorIs(t, err, errSVGLarge)
}

func TestSVGSize(t *testing.T) {
	size := func(attrs string) svgInfo {
		_, info, err := sanitizeSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" ` + attrs + `></svg>`))
		require.NoError(t, err)
		return info
	}
	require.Equal(t, svgInfo{1920, 1080}, size(`viewBox="0 0 1600 900"`))
	require.Equal(t, svgInfo{1080, 1920}, size(`viewBox="0,0,90,160"`))
	require.Equal(t, svgInfo{1920, 1920}, size(`width="24px" height="24"`))
	require.Equal(t, svgInfo{1920, 1080}, size(`width="100%" height="100%"`))
	require.Equal(t, svgInfo{1920, 480}, size(`viewBox="0 0 1000 10"`)) // clamped to 4:1
}

func TestLooksLikeSVG(t *testing.T) {
	require.True(t, looksLikeSVG([]byte("\uFEFF<?xml version=\"1.0\"?>\n<!-- c -->\n<!DOCTYPE svg>\n<svg>")))
	require.True(t, looksLikeSVG([]byte("  <svg:svg xmlns:svg=\"http://www.w3.org/2000/svg\">")))
	require.False(t, looksLikeSVG([]byte("<html><svg>")))
	require.False(t, looksLikeSVG([]byte("<!-- never closed")))
}

func TestSniffVideoPoster(t *testing.T) {
	for in, want := range map[string]string{
		"\xFF\xD8\xFF\xE0rest":         "image/jpeg",
		"RIFF\x00\x00\x00\x00WEBPVP8 ": "image/webp",
		"\x89PNG\r\n\x1a\nrest":        "image/png",
	} {
		mime, _, ok := sniffVideoPoster([]byte(in))
		require.True(t, ok)
		require.Equal(t, want, mime)
	}
	_, _, ok := sniffVideoPoster([]byte("<svg></svg>"))
	require.False(t, ok)
}

func TestVideoUploadFlow(t *testing.T) {
	repo := newFakeVideoRepo()
	dir := t.TempDir()
	s := newVideoService(repo, &fakeTTS{}, nil, dir)
	key := &APIKey{ID: 7, UserID: 42, Key: "sk-test"}
	ctx := context.Background()

	poster := []byte("\xFF\xD8\xFF\xE0jpeg")
	p, err := s.Upload(ctx, key, VideoUploadInput{Title: " 我的片子 ", Description: "介绍", Category: "science", File: uploadFixture(t, "rotated.mp4"), Poster: &VideoUploadFile{Name: "p.jpg", Size: int64(len(poster)), Reader: bytes.NewReader(poster)}})
	require.NoError(t, err)
	require.Equal(t, VideoModeUpload, p.Mode)
	require.Equal(t, VideoStatusReady, p.Status)
	require.Equal(t, "我的片子", p.Title)
	require.Equal(t, "science", p.Category)
	require.Equal(t, 36, p.Width)
	require.Equal(t, 64, p.Height)
	require.InDelta(t, 2.5, p.Duration, 0.01)
	up := p.Spec.Upload
	require.Equal(t, VideoUploadKindVideo, up.Kind)
	require.FileExists(t, filepath.Join(dir, p.ID, up.File))
	require.FileExists(t, filepath.Join(dir, p.ID, up.Poster))
	require.True(t, strings.HasPrefix(up.File, "media-") && strings.HasSuffix(up.File, ".mp4"))

	// Private: signed links only.
	require.NotNil(t, p.Media)
	require.Contains(t, p.Media.URL, "?exp=")
	_, err = s.MediaFile(ctx, p.ID, up.File, "", "")
	require.ErrorIs(t, err, ErrVideoNotFound)
	exp, sig := linkQuery(t, p.Media.URL)
	m, err := s.MediaFile(ctx, p.ID, up.File, exp, sig)
	require.NoError(t, err)
	require.Equal(t, "video/mp4", m.Mime)
	require.False(t, m.Public)
	_, err = s.MediaFile(ctx, p.ID, up.Poster, exp, sig) // signature is per file
	require.ErrorIs(t, err, ErrVideoNotFound)
	_, err = s.MediaFile(ctx, p.ID, "../../etc/passwd", exp, sig)
	require.ErrorIs(t, err, ErrVideoNotFound)
	s.now = func() time.Time { return time.Now().Add(13 * time.Hour) }
	_, err = s.MediaFile(ctx, p.ID, up.File, exp, sig)
	require.ErrorIs(t, err, ErrVideoNotFound, "expired link")
	s.now = time.Now

	// The agent does not work on uploads.
	_, err = s.Message(ctx, key, p.ID, "改一下", "")
	require.ErrorIs(t, err, ErrVideoUploadAgent)

	// Publish → review → public: plain links.
	_, err = s.Publish(ctx, key.UserID, p.ID, "science", "")
	require.NoError(t, err)
	require.NoError(t, s.Review(ctx, &User{Role: RoleAdmin}, p.ID, "approve", "", nil))
	w, err := s.Work(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/video/works/"+p.ID+"/media/"+up.File, w.Media.URL)
	m, err = s.MediaFile(ctx, p.ID, up.Poster, "", "")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", m.Mime)
	require.True(t, m.Public)

	// A new cover replaces the old file.
	png := []byte("\x89PNG\r\n\x1a\nimage")
	p2, err := s.SetPoster(ctx, key.UserID, p.ID, &VideoUploadFile{Size: int64(len(png)), Reader: bytes.NewReader(png)})
	require.NoError(t, err)
	require.NotEqual(t, up.Poster, p2.Spec.Upload.Poster)
	require.NoFileExists(t, filepath.Join(dir, p.ID, up.Poster))

	// SVG: sanitized on disk.
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><script>alert(1)</script><circle r="5"><animate attributeName="r" values="5;20;5" dur="1s" repeatCount="indefinite"/></circle></svg>`
	ps, err := s.Upload(ctx, key, VideoUploadInput{File: bytesFile("pulse.svg", []byte(svg))})
	require.NoError(t, err)
	require.Equal(t, "pulse", ps.Title)
	require.Equal(t, "other", ps.Category)
	require.Equal(t, 1920, ps.Width)
	require.Equal(t, 1920, ps.Height)
	stored, err := os.ReadFile(filepath.Join(dir, ps.ID, ps.Spec.Upload.File))
	require.NoError(t, err)
	require.NotContains(t, string(stored), "script")
	require.Contains(t, string(stored), "<animate")
	_, err = s.SetPoster(ctx, key.UserID, ps.ID, &VideoUploadFile{Size: int64(len(png)), Reader: bytes.NewReader(png)})
	require.ErrorIs(t, err, ErrVideoUploadNotVideo)

	// Deleting removes the files.
	require.NoError(t, s.Delete(ctx, key.UserID, p.ID))
	require.NoDirExists(t, filepath.Join(dir, p.ID))
}

func TestVideoUploadQuota(t *testing.T) {
	repo := newFakeVideoRepo()
	s := newVideoService(repo, &fakeTTS{}, nil, t.TempDir())
	key := &APIKey{ID: 7, UserID: 42, Key: "sk-test"}
	ctx := context.Background()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	var last *VideoProject
	for i := 0; i < VideoUploadPerDay; i++ {
		p, err := s.Upload(ctx, key, VideoUploadInput{File: bytesFile("a.svg", svg)})
		require.NoError(t, err)
		last = p
	}
	_, err := s.Upload(ctx, key, VideoUploadInput{File: bytesFile("a.svg", svg)})
	require.ErrorIs(t, err, ErrVideoUploadDaily)
	// Deleting does not give uploads back.
	require.NoError(t, s.Delete(ctx, key.UserID, last.ID))
	_, err = s.Upload(ctx, key, VideoUploadInput{File: bytesFile("a.svg", svg)})
	require.ErrorIs(t, err, ErrVideoUploadDaily)
	// Another user is not affected; storage is capped.
	other := &APIKey{ID: 8, UserID: 43, Key: "sk-other"}
	repo.projects["00000000-0000-0000-0000-999999999999"] = &VideoProject{ID: "00000000-0000-0000-0000-999999999999", UserID: 43, Mode: VideoModeUpload,
		CreatedAt: time.Now().Add(-48 * time.Hour), Spec: &VideoSpec{Upload: &VideoUpload{Size: VideoUploadStorageBytes - 10}}}
	_, err = s.Upload(ctx, other, VideoUploadInput{File: bytesFile("a.svg", svg)})
	require.ErrorIs(t, err, ErrVideoUploadStorage)
}

func linkQuery(t *testing.T, u string) (string, string) {
	t.Helper()
	_, q, ok := strings.Cut(u, "?")
	require.True(t, ok)
	var exp, sig string
	for _, kv := range strings.Split(q, "&") {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "exp":
			exp = v
		case "sig":
			sig = v
		}
	}
	return exp, sig
}
