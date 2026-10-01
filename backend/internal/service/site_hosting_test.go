//go:build unit

package service

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type zipEntry struct {
	name string
	body string
	mode os.FileMode
}

func makeZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			h.SetMode(e.mode)
		}
		w, err := zw.CreateHeader(h)
		require.NoError(t, err)
		_, _ = w.Write([]byte(e.body))
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestUnpackSiteUploadAcceptsHTMLAndFolderZips(t *testing.T) {
	files, err := unpackSiteUpload("page.HTML", []byte("<!doctype html><p>hi"), 1<<20, 10)
	require.NoError(t, err)
	require.Equal(t, []siteFile{{Path: "index.html", Data: []byte("<!doctype html><p>hi")}}, files)

	data := makeZip(t, zipEntry{name: "my-site/index.html", body: "<p>home"}, zipEntry{name: "my-site/css/a.css", body: "p{}"},
		zipEntry{name: "__MACOSX/my-site/._index.html", body: "junk"}, zipEntry{name: "my-site/.DS_Store", body: "junk"}, zipEntry{name: "my-site/img/"})
	files, err = unpackSiteUpload("site.zip", data, 1<<20, 10)
	require.NoError(t, err)
	require.Equal(t, []string{"css/a.css", "index.html"}, []string{files[0].Path, files[1].Path}, "the single folder becomes the root, junk is skipped")
}

func TestUnpackSiteUploadRejectsHostileArchives(t *testing.T) {
	code := func(err error) string { return infraerrors.Reason(err) }
	cases := map[string]struct {
		data   []byte
		reason string
	}{
		"traversal":     {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: "../../etc/cron.d/x.txt", body: "x"}), "SITE_FILE_REJECTED"},
		"absolute":      {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: "/etc/passwd.txt", body: "x"}), "SITE_FILE_REJECTED"},
		"server code":   {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: "shell.php", body: "<?php"}), "SITE_FILE_REJECTED"},
		"hidden":        {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: ".git/config.txt", body: "x"}), "SITE_FILE_REJECTED"},
		"symlink":       {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: "link.html", body: "/etc/passwd", mode: os.ModeSymlink | 0o777}), "SITE_FILE_REJECTED"},
		"no index":      {makeZip(t, zipEntry{name: "home.html", body: "x"}), "SITE_NO_INDEX"},
		"bomb":          {makeZip(t, zipEntry{name: "index.html", body: strings.Repeat("0", 3<<20)}), "SITE_TOO_LARGE"},
		"too many":      {makeZip(t, zipEntry{name: "index.html"}, zipEntry{name: "a.css"}, zipEntry{name: "b.css"}, zipEntry{name: "c.css"}), "SITE_TOO_MANY_FILES"},
		"not a zip":     {[]byte("hello"), "SITE_UPLOAD_INVALID"},
		"backslash dot": {makeZip(t, zipEntry{name: "index.html", body: "x"}, zipEntry{name: "..\\x.css", body: "x"}), "SITE_FILE_REJECTED"},
	}
	for name, tc := range cases {
		_, err := unpackSiteUpload("site.zip", tc.data, 1<<20, 3)
		require.Error(t, err, name)
		require.Equal(t, tc.reason, code(err), name)
	}
}

func TestSiteNameFromHost(t *testing.T) {
	svc := NewSiteHostingService(nil, nil, nil, nil, DefaultSiteHostingConfig("s.Example.top."), t.TempDir(), "https://hivegpt.cn")
	for host, want := range map[string][2]any{
		"abc1234.s.example.top":        {"abc1234", true},
		"ABC1234.S.EXAMPLE.TOP:443":    {"abc1234", true},
		"s.example.top":                {"", true},
		"hivegpt.cn":                   {"", false},
		"evil-s.example.top.attack.cn": {"", false},
		"xs.example.top":               {"", false},
	} {
		name, ok := svc.SiteNameFromHost(host)
		require.Equal(t, want[0], name, host)
		require.Equal(t, want[1], ok, host)
	}
	off := NewSiteHostingService(nil, nil, nil, nil, DefaultSiteHostingConfig(""), t.TempDir(), "")
	_, ok := off.SiteNameFromHost("abc.s.example.top")
	require.False(t, ok, "hosting off: every host is the main site")
}

type serveRepo struct {
	SiteHostingRepository
	sites map[string]*Site
}

func (r *serveRepo) GetSiteByName(_ context.Context, name string) (*Site, error) {
	return r.sites[name], nil
}

func TestServeSite(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "5", "v2")
	require.NoError(t, writeSiteFiles(root, []siteFile{
		{Path: "index.html", Data: []byte("<html><body><h1>Hi</h1></BODY></html>")},
		{Path: "about.html", Data: []byte("<p>about")},
		{Path: "docs/index.html", Data: []byte("<p>docs")},
		{Path: "app.js", Data: []byte("alert(1)")},
		{Path: "404.html", Data: []byte("<p>custom 404")},
	}))
	require.NoError(t, os.Symlink("/etc/passwd", filepath.Join(root, "passwd.txt")))
	repo := &serveRepo{sites: map[string]*Site{
		"abc1234":  {ID: 5, Name: "abc1234", Status: SiteStatusActive, Version: 2, UpdatedAt: time.Now()},
		"offline1": {ID: 6, Name: "offline1", Status: SiteStatusDisabled, Version: 1},
		"unpaid12": {ID: 7, Name: "unpaid12", Status: SiteStatusUnpaid, Version: 1},
	}}
	svc := NewSiteHostingService(repo, nil, nil, nil, DefaultSiteHostingConfig("s.example.top"), dir, "https://hivegpt.cn")
	get := func(name, path string, method ...string) *httptest.ResponseRecorder {
		m := http.MethodGet
		if len(method) > 0 {
			m = method[0]
		}
		w := httptest.NewRecorder()
		svc.ServeSite(w, httptest.NewRequest(m, path, nil), name)
		return w
	}

	w := get("abc1234", "/")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	require.Contains(t, w.Header().Get("Content-Security-Policy"), "form-action 'self'")
	body := w.Body.String()
	require.Contains(t, body, "由 HiveGPT 托管 · 举报")
	require.Contains(t, body, "https://hivegpt.cn/site-report?site=abc1234")
	require.Less(t, strings.Index(body, "data-hivegpt-badge"), strings.Index(body, "</BODY>"), "badge goes before </body>")

	require.Equal(t, "<p>about", strings.Split(get("abc1234", "/about").Body.String(), "<a ")[0], "pretty URLs map to .html")
	w = get("abc1234", "/docs")
	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/docs/", w.Header().Get("Location"))
	require.Contains(t, get("abc1234", "/docs/").Body.String(), "<p>docs")
	w = get("abc1234", "/app.js")
	require.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
	require.Equal(t, "alert(1)", w.Body.String(), "scripts get no badge")

	w = get("abc1234", "/../../../etc/passwd")
	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "custom 404", "the site's own 404 page")
	require.Equal(t, http.StatusNotFound, get("abc1234", "/passwd.txt").Code, "links are never followed")
	require.Equal(t, http.StatusMethodNotAllowed, get("abc1234", "/", http.MethodPost).Code)
	require.Equal(t, http.StatusGone, get("offline1", "/").Code)
	require.Equal(t, http.StatusServiceUnavailable, get("unpaid12", "/").Code)
	require.Equal(t, http.StatusNotFound, get("nosuch1", "/").Code)
	require.Equal(t, http.StatusNotFound, get("", "/").Code)
}

func TestInjectSiteBadgeWithoutBody(t *testing.T) {
	require.Equal(t, "<p>x</p>[b]", string(injectSiteBadge([]byte("<p>x</p>"), []byte("[b]"))))
}

func TestRandomSiteNamesAreValid(t *testing.T) {
	for i := 0; i < 200; i++ {
		name := randomSiteName()
		require.Regexp(t, `^[a-z][a-z0-9]{6}$`, name)
		require.True(t, siteNameRe.MatchString(name))
	}
}
