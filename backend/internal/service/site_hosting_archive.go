package service

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Unpacking an upload (one HTML file or a zip of a static site) into a version directory, with
// the limits that keep hostile archives out: size and file-count caps measured while reading,
// no path traversal, no links, an allowlist of static file types and an index.html at the root.

var siteContentTypes = map[string]string{
	".html": "text/html; charset=utf-8", ".htm": "text/html; charset=utf-8",
	".css": "text/css; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".mjs": "text/javascript; charset=utf-8",
	".json": "application/json", ".map": "application/json", ".webmanifest": "application/manifest+json",
	".txt": "text/plain; charset=utf-8", ".md": "text/plain; charset=utf-8", ".csv": "text/csv; charset=utf-8", ".xml": "application/xml",
	".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif", ".ico": "image/x-icon", ".bmp": "image/bmp",
	".woff": "font/woff", ".woff2": "font/woff2", ".ttf": "font/ttf", ".otf": "font/otf",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg", ".m4a": "audio/mp4",
	".mp4": "video/mp4", ".webm": "video/webm", ".wasm": "application/wasm", ".pdf": "application/pdf",
}

// SiteContentType is the served type for a file name ("" for types that are not allowed).
func SiteContentType(name string) string {
	return siteContentTypes[strings.ToLower(path.Ext(name))]
}

const siteMaxPathLen = 200

var (
	ErrSiteUploadInvalid  = infraerrors.BadRequest("SITE_UPLOAD_INVALID", "请上传一个 .html 文件，或包含 index.html 的 .zip 压缩包（Upload an .html file or a .zip with index.html）")
	ErrSiteUploadTooLarge = infraerrors.New(413, "SITE_TOO_LARGE", "上传的文件太大：请删掉不需要的文件或压缩图片后再上传（Upload is too large）")
	ErrSiteNoIndex        = infraerrors.BadRequest("SITE_NO_INDEX", "压缩包里没有 index.html：网站首页必须叫 index.html，放在压缩包根目录（或唯一的文件夹里）（index.html is missing）")
)

func siteTooLarge(limit int64) error {
	return infraerrors.New(413, "SITE_TOO_LARGE", fmt.Sprintf("网站解压后不能超过 %s，请删掉不需要的文件或压缩图片后再上传（Site is too large）", formatSiteBytes(limit)))
}

func siteTooManyFiles(limit int) error {
	return infraerrors.BadRequest("SITE_TOO_MANY_FILES", fmt.Sprintf("网站文件不能超过 %d 个（Too many files）", limit))
}

func siteBadFile(name, why string) error {
	return infraerrors.BadRequest("SITE_FILE_REJECTED", fmt.Sprintf("文件「%s」%s（File rejected）", truncateRunes(name, 80), why))
}

func formatSiteBytes(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%gMB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%dKB", n>>10)
}

// siteFile is one file of an unpacked upload.
type siteFile struct {
	Path string
	Data []byte
}

// unpackSiteUpload turns an upload into the site's files (paths relative to the site root).
func unpackSiteUpload(fileName string, data []byte, maxBytes int64, maxFiles int) ([]siteFile, error) {
	ext := strings.ToLower(path.Ext(fileName))
	switch {
	case ext == ".html" || ext == ".htm":
		if int64(len(data)) > maxBytes {
			return nil, siteTooLarge(maxBytes)
		}
		if !utf8.Valid(data) && !looksLikeHTML(data) {
			return nil, ErrSiteUploadInvalid
		}
		return []siteFile{{Path: "index.html", Data: data}}, nil
	case ext == ".zip" || bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return unpackSiteZip(data, maxBytes, maxFiles)
	}
	return nil, ErrSiteUploadInvalid
}

func looksLikeHTML(data []byte) bool {
	head := strings.ToLower(string(data[:min(len(data), 512)]))
	return strings.Contains(head, "<html") || strings.Contains(head, "<!doctype")
}

func unpackSiteZip(data []byte, maxBytes int64, maxFiles int) ([]siteFile, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrSiteUploadInvalid
	}
	var files []siteFile
	seen := map[string]bool{}
	var total int64
	for _, entry := range zr.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		if entry.FileInfo().IsDir() || strings.HasSuffix(name, "/") || skipSiteEntry(name) {
			continue
		}
		if entry.Mode()&os.ModeType != 0 {
			return nil, siteBadFile(name, "是链接或特殊文件，不能上传")
		}
		clean, err := cleanSitePath(name)
		if err != nil {
			return nil, err
		}
		if SiteContentType(clean) == "" {
			return nil, siteBadFile(clean, "的类型不支持：只能上传网页、样式、脚本、图片、字体、音视频等静态文件")
		}
		if seen[strings.ToLower(clean)] {
			continue
		}
		if len(files) >= maxFiles {
			return nil, siteTooManyFiles(maxFiles)
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, ErrSiteUploadInvalid
		}
		// Sizes in the zip header can lie: count what is actually decompressed.
		content, err := io.ReadAll(io.LimitReader(rc, maxBytes-total+1))
		_ = rc.Close()
		if err != nil {
			return nil, ErrSiteUploadInvalid
		}
		total += int64(len(content))
		if total > maxBytes {
			return nil, siteTooLarge(maxBytes)
		}
		seen[strings.ToLower(clean)] = true
		files = append(files, siteFile{Path: clean, Data: content})
	}
	files = stripSingleFolder(files)
	for _, f := range files {
		if f.Path == "index.html" {
			sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
			return files, nil
		}
	}
	return nil, ErrSiteNoIndex
}

// skipSiteEntry drops the junk archivers add (macOS resource forks, Finder / Explorer metadata).
func skipSiteEntry(name string) bool {
	base := path.Base(name)
	return strings.HasPrefix(name, "__MACOSX/") || strings.Contains(name, "/__MACOSX/") || base == ".DS_Store" || base == "Thumbs.db" || base == "desktop.ini"
}

func cleanSitePath(name string) (string, error) {
	if !utf8.ValidString(name) || len(name) > siteMaxPathLen {
		return "", siteBadFile(name, "的文件名太长或含有无法识别的字符")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", siteBadFile(name, "的文件名含有控制字符")
		}
	}
	if strings.HasPrefix(name, "/") || filepath.VolumeName(name) != "" {
		return "", siteBadFile(name, "使用了绝对路径")
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", siteBadFile(name, "的路径指向了网站目录之外")
	}
	for _, part := range strings.Split(clean, "/") {
		if strings.HasPrefix(part, ".") && part != ".well-known" {
			return "", siteBadFile(name, "是隐藏文件或在隐藏文件夹里")
		}
	}
	return clean, nil
}

// stripSingleFolder handles zips of a folder ("my-site/index.html"): the folder becomes the root.
func stripSingleFolder(files []siteFile) []siteFile {
	if len(files) == 0 {
		return files
	}
	for _, f := range files {
		if f.Path == "index.html" {
			return files
		}
	}
	first, _, ok := strings.Cut(files[0].Path, "/")
	if !ok {
		return files
	}
	prefix := first + "/"
	for _, f := range files {
		if !strings.HasPrefix(f.Path, prefix) {
			return files
		}
	}
	out := make([]siteFile, len(files))
	for i, f := range files {
		out[i] = siteFile{Path: strings.TrimPrefix(f.Path, prefix), Data: f.Data}
	}
	return out
}

// writeSiteFiles writes the files under dir (created fresh).
func writeSiteFiles(dir string, files []siteFile) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	for _, f := range files {
		target := filepath.Join(dir, filepath.FromSlash(f.Path))
		rel, err := filepath.Rel(dir, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return siteBadFile(f.Path, "的路径指向了网站目录之外")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(target, f.Data, 0o640); err != nil {
			return err
		}
	}
	return nil
}
