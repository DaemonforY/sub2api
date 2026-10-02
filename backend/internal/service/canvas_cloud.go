package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Canvas cloud sync: the canvas keeps projects, assets and workbench records in the browser; signed
// in, it syncs them through the user's HiveGPT account. The canvas does the merging — the server only
// stores files under a fixed set of paths ("canvas/manifest.json", "assets/files/image_xx.png"…) within
// a per-user quota. Files are served back to their owner only, as downloads, never rendered.

const (
	settingCanvasCloudQuotaMB           = "canvas_cloud_quota_mb"
	settingCanvasCloudSubscriberQuotaMB = "canvas_cloud_subscriber_quota_mb"
	CanvasCloudDefaultQuotaMB           = 200
	CanvasCloudDefaultSubscriberQuotaMB = 2048
	// CanvasCloudMaxFileBytes bounds one upload (a manifest with many projects, or one video).
	CanvasCloudMaxFileBytes = 100 << 20
)

// canvasCloudPathRe: <domain>/manifest.json or <domain>/files/<name>.
var canvasCloudPathRe = regexp.MustCompile(`^(canvas|assets|image-workbench|video-workbench)/(manifest\.json|files/[A-Za-z0-9._-]{1,200})$`)

var canvasCloudMimes = map[string]bool{"application/json": true, "application/octet-stream": true}

var (
	ErrCanvasCloudPath     = infraerrors.BadRequest("CANVAS_CLOUD_PATH", "同步文件路径不正确（Invalid path）")
	ErrCanvasCloudNotFound = infraerrors.NotFound("CANVAS_CLOUD_NOT_FOUND", "云端没有这个文件（Not found）")
	ErrCanvasCloudTooLarge = infraerrors.BadRequest("CANVAS_CLOUD_FILE_TOO_LARGE", "单个文件超过 100MB，无法同步（File too large）")
	ErrCanvasCloudEmpty    = infraerrors.BadRequest("CANVAS_CLOUD_FILE_EMPTY", "不能上传空文件（Empty file）")
)

// ErrCanvasCloudQuota: the upload would exceed the user's quota.
func ErrCanvasCloudQuota(quotaMB int64, subscribed bool) error {
	hint := "可以删掉不用的画布或资产后再同步"
	if !subscribed {
		hint += "，订阅用户空间更大"
	}
	return infraerrors.Forbidden("CANVAS_CLOUD_QUOTA", fmt.Sprintf("云同步空间已满（上限 %dMB），%s（Cloud storage full）", quotaMB, hint))
}

// CanvasCloudFile is one stored file.
type CanvasCloudFile struct {
	Path      string    `json:"path"`
	File      string    `json:"-"`
	Size      int64     `json:"size"`
	Mime      string    `json:"mime"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CanvasCloudUsage is shown next to the sync button.
type CanvasCloudUsage struct {
	UsedBytes  int64 `json:"used_bytes"`
	QuotaBytes int64 `json:"quota_bytes"`
	Files      int   `json:"files"`
	Subscribed bool  `json:"subscribed"`
}

type CanvasCloudRepository interface {
	GetCanvasCloudFile(ctx context.Context, userID int64, path string) (*CanvasCloudFile, error)
	ListCanvasCloudFiles(ctx context.Context, userID int64) ([]CanvasCloudFile, error)
	CanvasCloudUsage(ctx context.Context, userID int64) (bytes int64, files int, err error)
	// PutCanvasCloudFile upserts the row and returns the replaced file's disk name ("" if new).
	PutCanvasCloudFile(ctx context.Context, userID int64, f *CanvasCloudFile) (replaced string, err error)
	DeleteCanvasCloudFile(ctx context.Context, userID int64, path string) (file string, err error)
}

type CanvasCloudService struct {
	repo     CanvasCloudRepository
	dir      string
	settings imageToolSettings
	subs     canvasSubscriptionReader
}

func NewCanvasCloudService(repo CanvasCloudRepository, dir string, settings imageToolSettings, subs canvasSubscriptionReader) *CanvasCloudService {
	return &CanvasCloudService{repo: repo, dir: dir, settings: settings, subs: subs}
}

// CanvasCloudQuotaSettings are the admin's quotas in MB.
type CanvasCloudQuotaSettings struct {
	QuotaMB           int64 `json:"cloud_quota_mb"`
	SubscriberQuotaMB int64 `json:"cloud_subscriber_quota_mb"`
}

func readCanvasCloudQuotas(ctx context.Context, settings imageToolSettings) CanvasCloudQuotaSettings {
	q := CanvasCloudQuotaSettings{QuotaMB: CanvasCloudDefaultQuotaMB, SubscriberQuotaMB: CanvasCloudDefaultSubscriberQuotaMB}
	if settings == nil {
		return q
	}
	values, err := settings.GetMultiple(ctx, []string{settingCanvasCloudQuotaMB, settingCanvasCloudSubscriberQuotaMB})
	if err != nil {
		return q
	}
	if v, err := strconv.ParseInt(values[settingCanvasCloudQuotaMB], 10, 64); err == nil && v >= 0 {
		q.QuotaMB = v
	}
	if v, err := strconv.ParseInt(values[settingCanvasCloudSubscriberQuotaMB], 10, 64); err == nil && v >= 0 {
		q.SubscriberQuotaMB = v
	}
	return q
}

func saveCanvasCloudQuotas(ctx context.Context, settings imageToolSettings, q CanvasCloudQuotaSettings) error {
	if settings == nil {
		return nil
	}
	if q.QuotaMB < 0 || q.SubscriberQuotaMB < 0 || q.QuotaMB > 1<<20 || q.SubscriberQuotaMB > 1<<20 {
		return infraerrors.BadRequest("CANVAS_CLOUD_QUOTA_INVALID", "云同步空间请填 0–1048576 之间的 MB 数")
	}
	return settings.SetMultiple(ctx, map[string]string{
		settingCanvasCloudQuotaMB:           strconv.FormatInt(q.QuotaMB, 10),
		settingCanvasCloudSubscriberQuotaMB: strconv.FormatInt(q.SubscriberQuotaMB, 10),
	})
}

func (s *CanvasCloudService) subscribed(ctx context.Context, userID int64) bool {
	if s.subs == nil {
		return false
	}
	subs, err := s.subs.ListActiveByUserID(ctx, userID)
	if err != nil {
		return false
	}
	for _, sub := range subs {
		if sub.IsActive() {
			return true
		}
	}
	return false
}

func (s *CanvasCloudService) quota(ctx context.Context, userID int64) (int64, bool) {
	q := readCanvasCloudQuotas(ctx, s.settings)
	if s.subscribed(ctx, userID) {
		return q.SubscriberQuotaMB << 20, true
	}
	return q.QuotaMB << 20, false
}

func (s *CanvasCloudService) Usage(ctx context.Context, userID int64) (*CanvasCloudUsage, error) {
	used, files, err := s.repo.CanvasCloudUsage(ctx, userID)
	if err != nil {
		return nil, err
	}
	quota, subscribed := s.quota(ctx, userID)
	return &CanvasCloudUsage{UsedBytes: used, QuotaBytes: quota, Files: files, Subscribed: subscribed}, nil
}

func (s *CanvasCloudService) List(ctx context.Context, userID int64) ([]CanvasCloudFile, error) {
	return s.repo.ListCanvasCloudFiles(ctx, userID)
}

func (s *CanvasCloudService) userDir(userID int64) string {
	return filepath.Join(s.dir, strconv.FormatInt(userID, 10))
}

// Open returns a stored file for its owner (the caller closes it).
func (s *CanvasCloudService) Open(ctx context.Context, userID int64, path string) (*CanvasCloudFile, *os.File, error) {
	if !canvasCloudPathRe.MatchString(path) {
		return nil, nil, ErrCanvasCloudPath
	}
	f, err := s.repo.GetCanvasCloudFile(ctx, userID, path)
	if err != nil {
		return nil, nil, err
	}
	if f == nil {
		return nil, nil, ErrCanvasCloudNotFound
	}
	file, err := os.Open(filepath.Join(s.userDir(userID), f.File))
	if err != nil {
		return nil, nil, ErrCanvasCloudNotFound
	}
	return f, file, nil
}

// CanvasCloudMime keeps media types (image/*, video/*, audio/*), JSON and octet-stream; anything
// else is stored as octet-stream.
func CanvasCloudMime(contentType string) string {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "application/octet-stream"
	}
	mt = strings.ToLower(mt)
	if canvasCloudMimes[mt] {
		return mt
	}
	for _, prefix := range []string{"image/", "video/", "audio/"} {
		if strings.HasPrefix(mt, prefix) && mt != "image/svg+xml" && len(mt) <= 100 {
			return mt
		}
	}
	return "application/octet-stream"
}

// Put stores body (at most CanvasCloudMaxFileBytes) at path, within the user's quota.
func (s *CanvasCloudService) Put(ctx context.Context, userID int64, path, contentType string, body io.Reader) (*CanvasCloudFile, error) {
	if !canvasCloudPathRe.MatchString(path) {
		return nil, ErrCanvasCloudPath
	}
	dir := s.userDir(userID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create canvas cloud dir: %w", err)
	}
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, err
	}
	name := hex.EncodeToString(buf[:]) + ".bin"
	tmp := filepath.Join(dir, "."+name+".tmp")
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return nil, fmt.Errorf("create canvas cloud file: %w", err)
	}
	size, err := io.Copy(out, io.LimitReader(body, CanvasCloudMaxFileBytes+1))
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("write canvas cloud file: %w", err)
	}
	if size > CanvasCloudMaxFileBytes {
		_ = os.Remove(tmp)
		return nil, ErrCanvasCloudTooLarge
	}
	if size == 0 {
		_ = os.Remove(tmp)
		return nil, ErrCanvasCloudEmpty
	}

	// Quota: what is stored, minus the file being replaced, plus this one.
	used, _, err := s.repo.CanvasCloudUsage(ctx, userID)
	if err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if old, err := s.repo.GetCanvasCloudFile(ctx, userID, path); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	} else if old != nil {
		used -= old.Size
	}
	if quota, subscribed := s.quota(ctx, userID); used+size > quota {
		_ = os.Remove(tmp)
		return nil, ErrCanvasCloudQuota(quota>>20, subscribed)
	}

	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("store canvas cloud file: %w", err)
	}
	f := &CanvasCloudFile{Path: path, File: name, Size: size, Mime: CanvasCloudMime(contentType), UpdatedAt: time.Now()}
	replaced, err := s.repo.PutCanvasCloudFile(ctx, userID, f)
	if err != nil {
		_ = os.Remove(filepath.Join(dir, name))
		return nil, err
	}
	if replaced != "" && replaced != name {
		_ = os.Remove(filepath.Join(dir, replaced))
	}
	return f, nil
}

func (s *CanvasCloudService) Delete(ctx context.Context, userID int64, path string) error {
	if !canvasCloudPathRe.MatchString(path) {
		return ErrCanvasCloudPath
	}
	file, err := s.repo.DeleteCanvasCloudFile(ctx, userID, path)
	if err != nil {
		return err
	}
	if file != "" {
		if err := os.Remove(filepath.Join(s.userDir(userID), file)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove canvas cloud file: %w", err)
		}
	}
	return nil
}
