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
	"golang.org/x/crypto/bcrypt"
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

func (r *serveRepo) GetSiteByPreviousName(context.Context, string, time.Time) (*Site, error) {
	return nil, nil
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
		svc.ServeSite(w, httptest.NewRequest(m, path, nil), name, "203.0.113.1")
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

func TestCheckSiteContentFlagsRiskyPages(t *testing.T) {
	page := func(body string) []siteFile {
		return []siteFile{{Path: "index.html", Data: []byte("<html><head><title>T</title><style>.x{}</style></head><body>" + body + "<script>var a='博彩'</script></body></html>")}}
	}
	flags, excerpt, risky := checkSiteContent(page("<h1>我的作品集</h1><p>欢迎来到我的主页</p>"))
	require.False(t, risky, "%v", flags)
	require.Contains(t, excerpt, "我的作品集")
	require.NotContains(t, excerpt, "博彩", "scripts are not page text")

	cases := map[string]string{
		"password_field":   `<form><input name="u"><input type="password" name="p"></form>`,
		"card_or_id_field": `<input name="idcard" placeholder="身份证号">`,
		"gambling:百家乐":     `<p>真钱百家乐，充值即送</p>`,
		"brand_login:工商银行": `<h1>中国工商银行 账户安全中心</h1><p>请输入验证码完成身份验证</p>`,
		"fraud:刷单":         `<p>在家刷单，日结</p>`,
	}
	for want, body := range cases {
		flags, _, risky := checkSiteContent(page(body))
		require.True(t, risky, want)
		require.Contains(t, flags, want)
	}
	require.Equal(t, "疑似赌博、密码输入框", siteFlagSummary([]string{"gambling:x", "password_field", "gambling:y"}))
}

func TestSiteReviewModelDecides(t *testing.T) {
	var verdict string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"verdict\":\"` + verdict + `\",\"reason\":\"仿冒银行登录\"}"}}]}`))
	}))
	defer srv.Close()
	settings := &siteSettingsStub{values: map[string]string{settingSitesReviewBaseURL: srv.URL + "/v1", settingSitesReviewModel: "m", settingSitesReviewAPIKey: "k"}}
	svc := NewSiteHostingService(nil, nil, nil, settings, DefaultSiteHostingConfig("s.example.test"), t.TempDir(), "")
	ctx := context.Background()
	login := []siteFile{{Path: "index.html", Data: []byte(`<p>登录演示</p><input type="password">`)}}

	verdict = "allow"
	review := svc.reviewSite(ctx, svc.Config(ctx), login)
	require.Equal(t, SiteReviewApproved, review.Status, "the model can clear a keyword false positive")
	require.Equal(t, "model", review.By)
	verdict = "review"
	review = svc.reviewSite(ctx, svc.Config(ctx), []siteFile{{Path: "index.html", Data: []byte(`<p>hello</p>`)}})
	require.Equal(t, SiteReviewPending, review.Status)
	require.Contains(t, review.Reason, "仿冒银行登录")

	settings.values[settingSitesReviewAll] = "true"
	verdict = "allow"
	svc.cfg = nil
	review = svc.reviewSite(ctx, svc.Config(ctx), []siteFile{{Path: "index.html", Data: []byte(`<p>hello</p>`)}})
	require.Equal(t, SiteReviewPending, review.Status, "review-all queues everything")
}

type siteSettingsStub struct{ values map[string]string }

func (s *siteSettingsStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, k := range keys {
		out[k] = s.values[k]
	}
	return out, nil
}
func (s *siteSettingsStub) SetMultiple(_ context.Context, values map[string]string) error {
	for k, v := range values {
		s.values[k] = v
	}
	return nil
}

func TestSitePasswordGateAndPreview(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeSiteFiles(filepath.Join(dir, "5", "v1"), []siteFile{{Path: "index.html", Data: []byte("<body>online</body>")}}))
	require.NoError(t, writeSiteFiles(filepath.Join(dir, "5", "v2"), []siteFile{{Path: "index.html", Data: []byte("<body>pending</body>")}}))
	hash, _ := bcrypt.GenerateFromPassword([]byte("open-sesame"), bcrypt.MinCost)
	site := &Site{ID: 5, Name: "abc1234", Status: SiteStatusActive, Version: 1, PendingVersion: 2, PasswordHash: string(hash), UpdatedAt: time.Now()}
	svc := NewSiteHostingService(&serveRepo{sites: map[string]*Site{"abc1234": site}}, nil, nil, nil, DefaultSiteHostingConfig("s.example.test"), dir, "https://hivegpt.cn").WithSecret("k")
	serve := func(req *http.Request) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		svc.ServeSite(w, req, "abc1234", "198.51.100.7")
		return w
	}

	w := serve(httptest.NewRequest(http.MethodGet, "/docs/", nil))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "需要访问密码")
	require.Contains(t, w.Body.String(), `value="/docs/"`)

	form := func(pw string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, siteUnlockPath, strings.NewReader("password="+pw+"&next=/"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}
	require.Contains(t, serve(form("wrong")).Body.String(), "密码不正确")
	w = serve(form("open-sesame"))
	require.Equal(t, http.StatusSeeOther, w.Code)
	cookie := w.Result().Cookies()[0]
	require.Equal(t, siteAuthCookie, cookie.Name)
	require.True(t, cookie.HttpOnly)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	w = serve(req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "online")

	// Changing the password logs visitors out.
	hash2, _ := bcrypt.GenerateFromPassword([]byte("other-pass"), bcrypt.MinCost)
	site.PasswordHash = string(hash2)
	require.Equal(t, http.StatusUnauthorized, serve(req).Code)

	// Too many wrong passwords from one visitor are refused.
	for i := 0; i < siteUnlockAttempts; i++ {
		serve(form("nope"))
	}
	require.Contains(t, serve(form("other-pass")).Body.String(), "尝试次数太多")

	// A signed preview link shows the pending version (no password, no caching) and remembers it.
	preview := strings.TrimPrefix(svc.PreviewURL("abc1234", 5, 2), "https://abc1234.s.example.test")
	w = serve(httptest.NewRequest(http.MethodGet, preview, nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "pending")
	require.Contains(t, w.Body.String(), "预览：第 2 版")
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(w.Result().Cookies()[0])
	require.Contains(t, serve(req).Body.String(), "pending")
	require.Equal(t, http.StatusUnauthorized, serve(httptest.NewRequest(http.MethodGet, "/?"+sitePreviewParam+"=2.9999999999.forged", nil)).Code)
}

func TestSiteStatsCollector(t *testing.T) {
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, time.Local)
	c := newSiteStatsCollector(func() time.Time { return now })
	c.record(1, "a", true, 100)
	c.record(1, "a", true, 100)
	c.record(1, "b", true, 100)
	c.record(1, "b", false, 50)
	got := c.drain()
	require.Len(t, got, 1)
	require.Equal(t, int64(3), got[0].Views)
	require.Equal(t, int64(2), got[0].Visitors)
	require.Equal(t, int64(350), got[0].Bytes)
	c.record(1, "a", true, 10)
	require.Equal(t, int64(0), c.drain()[0].Visitors, "the same visitor is counted once a day")
	now = now.Add(24 * time.Hour)
	c.record(1, "a", true, 10)
	require.Equal(t, int64(1), c.drain()[0].Visitors, "a new day counts visitors again")
	require.Empty(t, c.drain())
}

func TestValidateSiteName(t *testing.T) {
	for _, ok := range []string{"abc", "my-page", "shop2026", "a1-b2-c3", "abcdefghijklmnopqrstuvwxyz1234"} {
		require.NoError(t, validateSiteName(ok), ok)
	}
	for _, bad := range []string{"ab", "1abc", "-abc", "abc-", "a--b", "my_page", "我的", "abcdefghijklmnopqrstuvwxyz12345", "My-Page"} {
		require.ErrorIs(t, validateSiteName(bad), ErrSiteNameInvalid, bad)
	}
	for _, reserved := range []string{"www", "admin", "hivegpt", "my-hivegpt-shop", "alipay-login", "bank-of-x", "police-notice"} {
		require.ErrorIs(t, validateSiteName(reserved), ErrSiteNameReserved, reserved)
	}
	require.Equal(t, "my-page", normalizeSiteName("  My-Page "))
}
