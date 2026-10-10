package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Uploaded works: a video (MP4 / WebM) or an animated SVG the user made elsewhere. They live in
// the user's 历史创作 next to generated works and go through the same gallery review, but the agent
// never touches them. Files are kept in video.dir/<project id>/ like narration audio.

const (
	VideoModeUpload = "upload"

	VideoUploadKindVideo = "video"
	VideoUploadKindSVG   = "svg"

	VideoUploadMaxVideoBytes  = 100 << 20
	VideoUploadMaxSVGBytes    = 2 << 20
	VideoUploadMaxPosterBytes = 2 << 20
	VideoUploadMaxSeconds     = 180
	VideoUploadPerDay         = 20
	VideoUploadStorageBytes   = 2 << 30

	// Admins upload showcase works from the review page: longer and larger files, no daily count
	// or storage quota. 250 MB stays under the server's global request body limit (256 MB).
	VideoUploadAdminMaxVideoBytes = 250 << 20
	VideoUploadAdminMaxSeconds    = 30 * 60

	videoMediaURLTTL = 12 * time.Hour
)

var (
	ErrVideoUploadType      = infraerrors.BadRequest("VIDEO_UPLOAD_TYPE", "只支持 MP4、WebM 视频或 SVG 动画，请换一个文件（Only MP4, WebM or SVG files）")
	ErrVideoUploadVideoBig  = infraerrors.BadRequest("VIDEO_UPLOAD_TOO_LARGE", "视频不能超过 100 MB，请压缩或剪短后再传（Videos must be 100 MB or smaller）")
	ErrVideoUploadSVGBig    = infraerrors.BadRequest("VIDEO_UPLOAD_SVG_TOO_LARGE", "SVG 不能超过 2 MB，请删掉不需要的图层或压缩路径后再传（SVG files must be 2 MB or smaller）")
	ErrVideoUploadTooLong   = infraerrors.BadRequest("VIDEO_UPLOAD_TOO_LONG", "视频不能超过 3 分钟，请剪短后再传（Videos must be 3 minutes or shorter）")
	ErrVideoUploadCodec     = infraerrors.BadRequest("VIDEO_UPLOAD_CODEC", "这个视频的编码浏览器播放不了，请导出为 H.264 编码的 MP4 后再传（Export as H.264 MP4）")
	ErrVideoUploadMOV       = infraerrors.BadRequest("VIDEO_UPLOAD_MOV", "这是 MOV（QuickTime）文件，请导出或转换为 MP4 后再传（Convert the MOV file to MP4）")
	ErrVideoUploadDuration  = infraerrors.BadRequest("VIDEO_UPLOAD_DURATION", "读不到这个视频的时长，请用剪辑软件重新导出为 MP4 后再传（Could not read the video duration; re-export as MP4）")
	ErrVideoUploadBroken    = infraerrors.BadRequest("VIDEO_UPLOAD_BROKEN", "视频文件不完整或已损坏，请重新导出后再传（The video file is damaged or incomplete）")
	ErrVideoUploadNoVideo   = infraerrors.BadRequest("VIDEO_UPLOAD_NO_VIDEO", "文件里没有视频画面（可能是纯音频），请换一个视频文件（No video track found）")
	ErrVideoUploadSVG       = infraerrors.BadRequest("VIDEO_UPLOAD_SVG", "SVG 文件格式不正确，请用 UTF-8 编码导出标准 SVG 后再传（Invalid SVG file）")
	ErrVideoUploadSVGLarge  = infraerrors.BadRequest("VIDEO_UPLOAD_SVG_COMPLEX", "SVG 元素太多或层级太深，请简化后再传（The SVG is too complex）")
	ErrVideoUploadPoster    = infraerrors.BadRequest("VIDEO_UPLOAD_POSTER", "封面只支持 2 MB 以内的 JPG、WebP 或 PNG 图片（Cover must be a JPG, WebP or PNG up to 2 MB）")
	ErrVideoUploadDaily     = infraerrors.TooManyRequests("VIDEO_UPLOAD_DAILY", "24 小时内最多上传 20 个作品，请明天再来（Upload limit reached: 20 per day）")
	ErrVideoUploadStorage   = infraerrors.BadRequest("VIDEO_UPLOAD_STORAGE", "上传作品的总空间已用满（2 GB），请先删除一些不需要的上传作品（Upload storage is full: 2 GB）")
	ErrVideoUploadAgent     = infraerrors.BadRequest("VIDEO_UPLOAD_AGENT", "上传的作品不能让 AI 修改；要改动请重新上传，或者用一句话让 AI 生成新作品（Uploaded works cannot be edited by the agent）")
	ErrVideoUploadAdminBig  = infraerrors.BadRequest("VIDEO_UPLOAD_TOO_LARGE", "管理员上传的视频不能超过 250 MB，请压缩后再传（Admin uploads must be 250 MB or smaller）")
	ErrVideoUploadAdminLong = infraerrors.BadRequest("VIDEO_UPLOAD_TOO_LONG", "管理员上传的视频不能超过 30 分钟，请剪短后再传（Admin uploads must be 30 minutes or shorter）")
	ErrVideoUploadNotVideo  = infraerrors.BadRequest("VIDEO_UPLOAD_NOT_VIDEO", "只有上传的视频可以更换封面（Only uploaded videos have a cover）")
)

// VideoUpload is the file of an uploaded work, kept in spec.upload.
type VideoUpload struct {
	Kind       string  `json:"kind"` // video / svg
	File       string  `json:"file"`
	Mime       string  `json:"mime"`
	Size       int64   `json:"size"`
	Duration   float64 `json:"duration,omitempty"`
	Codec      string  `json:"codec,omitempty"`
	Poster     string  `json:"poster,omitempty"`
	PosterMime string  `json:"poster_mime,omitempty"`
	PosterSize int64   `json:"poster_size,omitempty"`
	Name       string  `json:"name,omitempty"` // the original file name
}

// VideoMediaLinks are where the player loads an uploaded work: plain URLs for public works, signed
// ones (valid for a few hours) for the owner's and the reviewer's private copies.
type VideoMediaLinks struct {
	Kind   string `json:"kind"`
	URL    string `json:"url"`
	Poster string `json:"poster,omitempty"`
}

// VideoUploadFile is one uploaded file part.
type VideoUploadFile struct {
	Name   string
	Size   int64
	Reader io.ReaderAt
}

// videoUploadLimits are the size and length caps of one upload; quota adds the per-user daily count
// and storage checks.
type videoUploadLimits struct {
	maxBytes   int64
	maxSeconds float64
	quota      bool
	errBig     error
	errLong    error
}

var (
	videoUserUploadLimits  = videoUploadLimits{VideoUploadMaxVideoBytes, VideoUploadMaxSeconds, true, ErrVideoUploadVideoBig, ErrVideoUploadTooLong}
	videoAdminUploadLimits = videoUploadLimits{VideoUploadAdminMaxVideoBytes, VideoUploadAdminMaxSeconds, false, ErrVideoUploadAdminBig, ErrVideoUploadAdminLong}
)

// VideoUploadMaxBytesFor is the largest video the key's user may upload.
func VideoUploadMaxBytesFor(key *APIKey) int64 {
	return videoUploadLimitsFor(key).maxBytes
}

func videoUploadLimitsFor(key *APIKey) videoUploadLimits {
	if key != nil && key.User != nil && key.User.Role == RoleAdmin {
		return videoAdminUploadLimits
	}
	return videoUserUploadLimits
}

type VideoUploadInput struct {
	Title       string
	Description string
	Category    string
	File        VideoUploadFile
	Poster      *VideoUploadFile
}

// checkVideoUpload identifies an uploaded file. Videos are checked in place; an SVG comes back
// sanitized (the bytes to store).
func checkVideoUpload(f VideoUploadFile, lim videoUploadLimits) (*VideoUpload, *videoMediaInfo, []byte, error) {
	if f.Reader == nil || f.Size <= 0 {
		return nil, nil, nil, ErrVideoUploadType
	}
	head := make([]byte, min(f.Size, 512))
	if _, err := f.Reader.ReadAt(head, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, nil, ErrVideoUploadBroken
	}
	if looksLikeSVG(head) {
		if f.Size > VideoUploadMaxSVGBytes {
			return nil, nil, nil, ErrVideoUploadSVGBig
		}
		raw := make([]byte, f.Size)
		if _, err := f.Reader.ReadAt(raw, 0); err != nil && !errors.Is(err, io.EOF) {
			return nil, nil, nil, ErrVideoUploadBroken
		}
		clean, info, err := sanitizeSVG(raw)
		switch {
		case errors.Is(err, errSVGLarge):
			return nil, nil, nil, ErrVideoUploadSVGLarge
		case err != nil:
			return nil, nil, nil, ErrVideoUploadSVG
		}
		up := &VideoUpload{Kind: VideoUploadKindSVG, Mime: "image/svg+xml", Size: int64(len(clean))}
		return up, &videoMediaInfo{Mime: up.Mime, Ext: ".svg", Width: info.Width, Height: info.Height}, clean, nil
	}
	if f.Size > lim.maxBytes {
		return nil, nil, nil, lim.errBig
	}
	info, err := probeVideoMedia(f.Reader, f.Size)
	switch {
	case errors.Is(err, errVideoMediaUnknown):
		return nil, nil, nil, ErrVideoUploadType
	case errors.Is(err, errVideoMediaQuickTime):
		return nil, nil, nil, ErrVideoUploadMOV
	case errors.Is(err, errVideoMediaNoTrack):
		return nil, nil, nil, ErrVideoUploadNoVideo
	case errors.Is(err, errVideoMediaCodec):
		return nil, nil, nil, ErrVideoUploadCodec
	case errors.Is(err, errVideoMediaDuration):
		return nil, nil, nil, ErrVideoUploadDuration
	case err != nil:
		return nil, nil, nil, ErrVideoUploadBroken
	}
	if info.Duration > lim.maxSeconds+0.5 {
		return nil, nil, nil, lim.errLong
	}
	up := &VideoUpload{Kind: VideoUploadKindVideo, Mime: info.Mime, Size: f.Size, Duration: info.Duration, Codec: info.Codec}
	return up, info, nil, nil
}

// looksLikeSVG checks the start of a text file for an <svg> root (after a BOM, an XML declaration,
// comments or a DOCTYPE).
func looksLikeSVG(head []byte) bool {
	s := strings.TrimPrefix(string(head), "\uFEFF")
	for range 8 {
		s = strings.TrimLeft(s, " \t\r\n")
		switch {
		case strings.HasPrefix(s, "<?"):
			_, rest, ok := strings.Cut(s, "?>")
			if !ok {
				return false
			}
			s = rest
		case strings.HasPrefix(s, "<!--"):
			_, rest, ok := strings.Cut(s, "-->")
			if !ok {
				return false
			}
			s = rest
		case strings.HasPrefix(s, "<!"):
			end := ">"
			if i, j := strings.Index(s, "["), strings.Index(s, ">"); i >= 0 && i < j {
				end = "]>" // a DOCTYPE with an internal subset
			}
			_, rest, ok := strings.Cut(s, end)
			if !ok {
				return false
			}
			s = rest
		default:
			return strings.HasPrefix(s, "<svg") || strings.HasPrefix(s, "<svg:svg")
		}
	}
	return false
}

func readPoster(f *VideoUploadFile) ([]byte, string, string, error) {
	if f == nil || f.Reader == nil || f.Size <= 0 || f.Size > VideoUploadMaxPosterBytes {
		return nil, "", "", ErrVideoUploadPoster
	}
	data := make([]byte, f.Size)
	if _, err := f.Reader.ReadAt(data, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, "", "", ErrVideoUploadPoster
	}
	mime, ext, ok := sniffVideoPoster(data)
	if !ok {
		return nil, "", "", ErrVideoUploadPoster
	}
	return data, mime, ext, nil
}

func videoUploadFileName(prefix, ext string, sum []byte) string {
	return prefix + "-" + hex.EncodeToString(sum)[:12] + ext
}

// Upload stores a user's own video or SVG animation as a finished project.
func (s *VideoService) Upload(ctx context.Context, key *APIKey, in VideoUploadInput) (*VideoProject, error) {
	if key == nil || key.Key == "" {
		return nil, ErrVideoKey
	}
	lim := videoUploadLimitsFor(key)
	up, info, svg, err := checkVideoUpload(in.File, lim)
	if err != nil {
		return nil, err
	}
	var poster []byte
	var posterExt string
	if in.Poster != nil && up.Kind == VideoUploadKindVideo {
		if poster, up.PosterMime, posterExt, err = readPoster(in.Poster); err != nil {
			return nil, err
		}
		up.PosterSize = int64(len(poster))
	}
	since := s.now().Add(-24 * time.Hour)
	count, used, err := s.repo.UploadStats(ctx, key.UserID, since)
	if err != nil {
		return nil, err
	}
	if lim.quota && max(count, s.recentUploads(key.UserID, since)) >= VideoUploadPerDay {
		return nil, ErrVideoUploadDaily
	}
	if lim.quota && used+up.Size+up.PosterSize > VideoUploadStorageBytes {
		return nil, ErrVideoUploadStorage
	}

	name := strings.TrimSpace(filepath.Base(strings.ReplaceAll(in.File.Name, `\`, "/")))
	up.Name = clipRunes(name, 120)
	title := clipRunes(in.Title, videoTitleMaxChars)
	if title == "" {
		title = clipRunes(strings.TrimSuffix(name, filepath.Ext(name)), videoTitleMaxChars)
	}
	if title == "" {
		title = "未命名作品"
	}
	description := clipRunes(in.Description, videoPromptMaxChars)
	category := in.Category
	if _, ok := findVideoCategory(category); !ok {
		category = "other"
		if up.Kind == VideoUploadKindVideo {
			category = "film"
		}
	}
	p := &VideoProject{
		UserID: key.UserID, APIKeyID: key.ID, Mode: VideoModeUpload, Title: title, Prompt: description,
		Options: VideoOptions{Category: category}, Status: VideoStatusRunning, Width: info.Width, Height: info.Height,
		Visibility: VideoVisibilityPrivate, Category: category,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	s.noteUpload(key.UserID, since)
	fail := func(err error) (*VideoProject, error) {
		_, _ = s.repo.Delete(ctx, key.UserID, p.ID)
		_ = os.RemoveAll(s.projectDir(p.ID))
		return nil, err
	}
	dir := s.projectDir(p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fail(err)
	}
	if svg != nil {
		sum := sha256.Sum256(svg)
		up.File = videoUploadFileName("media", ".svg", sum[:])
		if err := os.WriteFile(filepath.Join(dir, up.File), svg, 0o644); err != nil {
			return fail(err)
		}
	} else {
		file, err := copyVideoUpload(dir, info.Ext, in.File)
		if err != nil {
			return fail(err)
		}
		up.File = file
	}
	if poster != nil {
		sum := sha256.Sum256(poster)
		up.Poster = videoUploadFileName("poster", posterExt, sum[:])
		if err := os.WriteFile(filepath.Join(dir, up.Poster), poster, 0o644); err != nil {
			return fail(err)
		}
	}
	p.Spec = &VideoSpec{Title: title, Summary: clipRunes(description, 300), Width: info.Width, Height: info.Height, Loop: up.Kind == VideoUploadKindSVG, Upload: up}
	p.Duration = up.Duration
	p.Status = VideoStatusReady
	if err := s.repo.Save(ctx, p); err != nil {
		return fail(err)
	}
	kind := "视频"
	if up.Kind == VideoUploadKindSVG {
		kind = "SVG 动画"
	}
	_ = s.repo.AddEvent(ctx, p.ID, &VideoEvent{Kind: "step", Text: fmt.Sprintf("已上传%s「%s」", kind, up.Name), Data: videoStepData("done")})
	return s.own(ctx, key.UserID, p.ID)
}

// copyVideoUpload copies a checked video into the project directory under a content-hash name.
func copyVideoUpload(dir, ext string, f VideoUploadFile) (string, error) {
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), io.NewSectionReader(f.Reader, 0, f.Size)); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	name := videoUploadFileName("media", ext, h.Sum(nil))
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return "", err
	}
	return name, os.Rename(tmp.Name(), filepath.Join(dir, name))
}

// SetPoster replaces the cover of an uploaded video.
func (s *VideoService) SetPoster(ctx context.Context, userID int64, id string, f *VideoUploadFile) (*VideoProject, error) {
	p, err := s.own(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.setPoster(ctx, p, f); err != nil {
		return nil, err
	}
	return s.own(ctx, userID, id)
}

// AdminSetPoster replaces the cover of any uploaded video (作品管理 on the review page).
func (s *VideoService) AdminSetPoster(ctx context.Context, admin *User, id string, f *VideoUploadFile) (*VideoProject, error) {
	p, err := s.AdminWork(ctx, admin, id)
	if err != nil {
		return nil, err
	}
	if err := s.setPoster(ctx, p, f); err != nil {
		return nil, err
	}
	return s.AdminWork(ctx, admin, id)
}

func (s *VideoService) setPoster(ctx context.Context, p *VideoProject, f *VideoUploadFile) error {
	if p.Mode != VideoModeUpload || p.Spec == nil || p.Spec.Upload == nil || p.Spec.Upload.Kind != VideoUploadKindVideo {
		return ErrVideoUploadNotVideo
	}
	data, mime, ext, err := readPoster(f)
	if err != nil {
		return err
	}
	up := p.Spec.Upload
	sum := sha256.Sum256(data)
	name := videoUploadFileName("poster", ext, sum[:])
	dir := s.projectDir(p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		return err
	}
	old := up.Poster
	up.Poster, up.PosterMime, up.PosterSize = name, mime, int64(len(data))
	if err := s.repo.Save(ctx, p); err != nil {
		return err
	}
	if old != "" && old != name {
		_ = os.Remove(filepath.Join(dir, filepath.Base(old)))
	}
	return nil
}

func (s *VideoService) recentUploads(userID int64, since time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, t := range s.uploads[userID] {
		if t.After(since) {
			n++
		}
	}
	return n
}

func (s *VideoService) noteUpload(userID int64, since time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.uploads == nil {
		s.uploads = map[int64][]time.Time{}
	}
	kept := []time.Time{s.now()}
	for _, t := range s.uploads[userID] {
		if t.After(since) {
			kept = append(kept, t)
		}
	}
	s.uploads[userID] = kept
}

func (s *VideoService) projectDir(id string) string {
	return filepath.Join(s.audioDir, filepath.Base(id))
}

// --- serving ------------------------------------------------------------------------------------

func (s *VideoService) mediaSig(id, file string, exp int64) string {
	m := hmac.New(sha256.New, s.mediaKey)
	_, _ = fmt.Fprintf(m, "%s\n%s\n%d", id, file, exp)
	return hex.EncodeToString(m.Sum(nil))[:32]
}

// mediaURL links a file of a work; private works get a signature that expires (rounded to the hour
// so the URL, and the browser's cached copy, stay the same for a while).
func (s *VideoService) mediaURL(id, file string, signed bool) string {
	if file == "" {
		return ""
	}
	u := "/api/v1/video/works/" + id + "/media/" + url.PathEscape(file)
	if !signed {
		return u
	}
	exp := s.now().Truncate(time.Hour).Add(videoMediaURLTTL).Unix()
	return u + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + s.mediaSig(id, file, exp)
}

func (s *VideoService) mediaLinks(id, visibility string, up *VideoUpload) *VideoMediaLinks {
	if up == nil {
		return nil
	}
	signed := visibility != VideoVisibilityPublic
	return &VideoMediaLinks{Kind: up.Kind, URL: s.mediaURL(id, up.File, signed), Poster: s.mediaURL(id, up.Poster, signed)}
}

// withMedia fills in the media links of an uploaded work.
func (s *VideoService) withMedia(p *VideoProject) *VideoProject {
	if p != nil && p.Mode == VideoModeUpload && p.Spec != nil {
		p.Media = s.mediaLinks(p.ID, p.Visibility, p.Spec.Upload)
	}
	return p
}

func (s *VideoService) cardsWithMedia(cards []VideoCard) []VideoCard {
	for i := range cards {
		if cards[i].Mode == VideoModeUpload {
			cards[i].Media = s.mediaLinks(cards[i].ID, cards[i].Visibility, cards[i].Upload)
		}
	}
	return cards
}

// VideoMediaFile is a file of an uploaded work ready to serve.
type VideoMediaFile struct {
	Path   string
	Mime   string
	SVG    bool
	Public bool
}

// MediaFile returns an uploaded work's file (or cover) when the work is public or the link is signed
// and not expired.
func (s *VideoService) MediaFile(ctx context.Context, id, file, exp, sig string) (*VideoMediaFile, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Mode != VideoModeUpload || p.Spec == nil || p.Spec.Upload == nil || file == "" {
		return nil, ErrVideoNotFound
	}
	up := p.Spec.Upload
	public := p.Visibility == VideoVisibilityPublic
	if !public {
		e, err := strconv.ParseInt(exp, 10, 64)
		if err != nil || e < s.now().Unix() || !hmac.Equal([]byte(sig), []byte(s.mediaSig(p.ID, file, e))) {
			return nil, ErrVideoNotFound
		}
	}
	out := &VideoMediaFile{Public: public}
	switch file {
	case up.File:
		out.Mime, out.SVG = up.Mime, up.Kind == VideoUploadKindSVG
	case up.Poster:
		out.Mime = up.PosterMime
	default:
		return nil, ErrVideoNotFound
	}
	out.Path = filepath.Join(s.projectDir(p.ID), filepath.Base(file))
	if _, err := os.Stat(out.Path); err != nil {
		return nil, ErrVideoNotFound
	}
	return out, nil
}

func newVideoMediaKey() []byte {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	return k
}
