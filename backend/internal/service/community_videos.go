package service

import (
	"bytes"
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Video works (kind = video): one MP4 / WebM clip plus a cover image (drawn from a frame by the
// canvas) that serves as the work's first image, so feeds, share cards and the admin queue need
// nothing new. Frames are not checked automatically, so video works wait for review unless the
// admin turns that off.

const (
	WorkKindVideo = "video"
	// CommunityVideoMaxBytes bounds one clip (generated clips are a few MB to ~30 MB).
	CommunityVideoMaxBytes = 60 << 20
	communityVideoMaxMs    = 10 * 60 * 1000

	// settingCommunityVideoReview: "false" publishes video works without review (default: review).
	settingCommunityVideoReview = "community_video_review"
)

var (
	ErrCommunityVideoInvalid  = infraerrors.BadRequest("COMMUNITY_VIDEO_INVALID", "只支持 MP4、WebM 视频，并需要一张封面图（Unsupported video）")
	ErrCommunityVideoTooLarge = infraerrors.BadRequest("COMMUNITY_VIDEO_TOO_LARGE", "视频不能超过 60 MB（Video too large）")
)

// WorkVideo is the clip of a video work.
type WorkVideo struct {
	File       string `json:"-"`
	URL        string `json:"url"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
	DurationMs int    `json:"duration_ms"`
}

// sniffVideo recognises MP4 (ISO BMFF "ftyp" box; QuickTime brands included) and WebM (EBML).
func sniffVideo(data []byte) (mime, ext string, ok bool) {
	if len(data) >= 12 && bytes.Equal(data[4:8], []byte("ftyp")) {
		return "video/mp4", "mp4", true
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return "video/webm", "webm", true
	}
	return "", "", false
}

// SaveVideo stores a work's clip as uploaded.
func (s *CommunityMediaStore) SaveVideo(data []byte) (*WorkVideo, error) {
	if len(data) > CommunityVideoMaxBytes {
		return nil, ErrCommunityVideoTooLarge
	}
	mime, ext, ok := sniffVideo(data)
	if !ok {
		return nil, ErrCommunityVideoInvalid
	}
	base, err := randomMediaName()
	if err != nil {
		return nil, err
	}
	name := base + "." + ext
	if err := s.write(name, data); err != nil {
		return nil, err
	}
	return &WorkVideo{File: name, MimeType: mime, SizeBytes: int64(len(data))}, nil
}

// videoReview: video works wait for review unless the admin switched it off.
func (s *CommunityService) videoReview(ctx context.Context) bool {
	if s.settings == nil {
		return true
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingCommunityVideoReview})
	return err != nil || values[settingCommunityVideoReview] != "false"
}

func clampVideoDuration(ms int) int {
	return min(max(ms, 0), communityVideoMaxMs)
}
