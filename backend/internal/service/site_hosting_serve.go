package service

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Serving the hosted sites at <name>.<domain>. Every response carries headers that keep the pages
// from sniffing types or posting forms elsewhere, and HTML pages get a small "hosted by" badge with
// a report link.

// SiteNameFromHost returns the site name for a request host. ok is true for any host under the
// hosting domain (name is "" for the bare domain); false means the request is for the main site.
func (s *SiteHostingService) SiteNameFromHost(host string) (name string, ok bool) {
	domain := s.defaults.Domain
	if domain == "" {
		return "", false
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if h, _, found := strings.Cut(host, ":"); found {
		host = h
	}
	if host == domain {
		return "", true
	}
	if !strings.HasSuffix(host, "."+domain) {
		return "", false
	}
	return strings.TrimSuffix(host, "."+domain), true
}

// KnownSite answers Caddy's on-demand TLS check: issue certificates only for existing sites (and
// the bare hosting domain, which shows a "no such site" page).
func (s *SiteHostingService) KnownSite(ctx context.Context, host string) bool {
	name, ok := s.SiteNameFromHost(host)
	if ok && name == "" {
		return true
	}
	if !ok || !siteNameRe.MatchString(name) {
		return false
	}
	site, err := s.lookup(ctx, name)
	if err == nil && site == nil {
		// A recently renamed site's old name answers with a redirect, which needs a certificate too.
		site, err = s.renamedTo(ctx, name)
	}
	return err == nil && site != nil
}

func (s *SiteHostingService) lookup(ctx context.Context, name string) (*Site, error) {
	s.mu.Lock()
	if hit, ok := s.lookups[name]; ok && s.now().Sub(hit.at) < siteLookupTTL {
		s.mu.Unlock()
		return hit.site, nil
	}
	s.mu.Unlock()
	site, err := s.repo.GetSiteByName(ctx, name)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if len(s.lookups) > 10000 {
		s.lookups = map[string]siteLookup{}
	}
	s.lookups[name] = siteLookup{site: site, at: s.now()}
	s.mu.Unlock()
	return site, nil
}

func (s *SiteHostingService) forget(name string) {
	s.mu.Lock()
	delete(s.lookups, name)
	s.mu.Unlock()
}

func setSiteSecurityHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	// Pages may load scripts and styles from anywhere, but forms only post back to the page's
	// own site and pages cannot be framed by other sites (lowers their value for phishing).
	h.Set("Content-Security-Policy", "form-action 'self'; frame-ancestors 'self'; base-uri 'self'")
	h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
}

// ServeSite answers a request for a hosted site (name "" = the bare hosting domain).
func (s *SiteHostingService) ServeSite(w http.ResponseWriter, r *http.Request, name, clientIP string) {
	setSiteSecurityHeaders(w.Header())
	unlock := r.URL.Path == siteUnlockPath
	if r.Method != http.MethodGet && r.Method != http.MethodHead && (!unlock || r.Method != http.MethodPost) {
		w.Header().Set("Allow", "GET, HEAD")
		s.statusPage(w, http.StatusMethodNotAllowed, "不支持的请求", "托管的网站是静态网页，只支持浏览。")
		return
	}
	if name == "" || !siteNameRe.MatchString(name) {
		s.statusPage(w, http.StatusNotFound, "网站不存在", "这个地址没有对应的网站。")
		return
	}
	site, err := s.lookup(r.Context(), name)
	if err != nil {
		s.statusPage(w, http.StatusServiceUnavailable, "暂时无法访问", "请稍后刷新重试。")
		return
	}
	if site == nil {
		if moved, err := s.renamedTo(r.Context(), name); err == nil && moved != nil {
			target := s.siteURL(moved.Name) + r.URL.RequestURI()
			// 302, not 301: a cached permanent redirect would loop if the owner later renames back.
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		s.statusPage(w, http.StatusNotFound, "网站不存在", "这个网站不存在或已被删除。")
		return
	}
	// A signed preview link (review queue, the owner's pending upload) opens a version that is not
	// online yet; it is remembered in a cookie so the page's own links keep working.
	version := site.Version
	preview := 0
	if token := r.URL.Query().Get(sitePreviewParam); token != "" {
		if preview = s.previewVersion(site, token); preview > 0 {
			http.SetCookie(w, &http.Cookie{Name: sitePreviewCookie, Value: token, Path: "/", MaxAge: 3600, HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode})
		}
	} else if c, err := r.Cookie(sitePreviewCookie); err == nil {
		preview = s.previewVersion(site, c.Value)
	}
	if preview > 0 {
		version = preview
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex")
	} else {
		switch site.Status {
		case SiteStatusActive:
		case SiteStatusPending:
			s.statusPage(w, http.StatusServiceUnavailable, "网站审核中", "这个网站刚刚发布，正在审核，通过后即可访问。")
			return
		case SiteStatusDisabled:
			s.statusPage(w, http.StatusGone, "网站已下线", "该网站因违反使用规范已被下线。")
			return
		default:
			s.statusPage(w, http.StatusServiceUnavailable, "网站已暂停", "站长的订阅已到期或未续费，网站暂时无法访问。")
			return
		}
		if site.PasswordHash != "" && !s.unlocked(r, site) {
			s.passwordGate(w, r, site, clientIP, unlock)
			return
		}
	}
	if version == 0 {
		s.statusPage(w, http.StatusNotFound, "网站不存在", "这个网站还没有可以显示的内容。")
		return
	}
	if unlock {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	root := s.versionDir(site.ID, version)
	reqPath := path.Clean("/" + r.URL.Path)
	file, redirect := resolveSiteFile(root, reqPath, strings.HasSuffix(r.URL.Path, "/"))
	if redirect {
		target := reqPath + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	}
	status := http.StatusOK
	if file == "" {
		file = filepath.Join(root, "404.html")
		if _, err := os.Stat(file); err != nil {
			s.statusPage(w, http.StatusNotFound, "页面不存在", "这个网站里没有这个页面。")
			return
		}
		status = http.StatusNotFound
	}
	s.serveFile(w, r, site, version, preview > 0, file, status, clientIP)
}

func (s *SiteHostingService) unlocked(r *http.Request, site *Site) bool {
	c, err := r.Cookie(siteAuthCookie)
	return err == nil && s.validAuthToken(site, c.Value)
}

const (
	siteUnlockWindow   = 15 * time.Minute
	siteUnlockAttempts = 10
)

// passwordGate shows the password form, or checks a submitted password (rate limited per visitor).
func (s *SiteHostingService) passwordGate(w http.ResponseWriter, r *http.Request, site *Site, clientIP string, submit bool) {
	message := ""
	if submit && r.Method == http.MethodPost {
		key := site.Name + "|" + clientIP
		if !s.allowUnlockAttempt(key) {
			message = "尝试次数太多，请 15 分钟后再试。"
		} else if bcrypt.CompareHashAndPassword([]byte(site.PasswordHash), []byte(r.PostFormValue("password"))) == nil {
			exp := s.now().Add(siteAuthTTL).Unix()
			http.SetCookie(w, &http.Cookie{Name: siteAuthCookie, Value: s.authToken(site, exp), Path: "/", MaxAge: int(siteAuthTTL.Seconds()), HttpOnly: true,
				Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode})
			next := r.PostFormValue("next")
			if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
				next = "/"
			}
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		} else {
			message = "密码不正确，请重试。"
		}
	}
	next := r.URL.Path
	if submit {
		next = r.PostFormValue("next")
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusUnauthorized)
	errHTML := ""
	if message != "" {
		errHTML = `<p style="color:#dc2626;margin:0 0 12px">` + html.EscapeString(message) + `</p>`
	}
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>需要访问密码</title></head>`+
		`<body style="margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:-apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;background:#f5f5f4;color:#1c1917">`+
		`<form method="post" action="%s" style="background:#fff;padding:28px;border-radius:12px;box-shadow:0 1px 3px rgba(0,0,0,.1);width:min(340px,90vw)">`+
		`<h1 style="font-size:20px;margin:0 0 8px">需要访问密码</h1><p style="margin:0 0 16px;color:#57534e">站长为这个网站设置了密码，请输入后访问。</p>%s`+
		`<input type="hidden" name="next" value="%s"><input type="password" name="password" autofocus required placeholder="访问密码" style="box-sizing:border-box;width:100%%;padding:10px;border:1px solid #d6d3d1;border-radius:8px;font-size:15px">`+
		`<button type="submit" style="margin-top:12px;width:100%%;padding:10px;border:0;border-radius:8px;background:#0d9488;color:#fff;font-size:15px;cursor:pointer">进入网站</button>`+
		`<p style="margin:16px 0 0;font-size:12px;color:#78716c">由 HiveGPT 提供网站托管 · <a href="%s" style="color:#78716c">举报</a></p></form></body></html>`,
		siteUnlockPath, errHTML, html.EscapeString(next), html.EscapeString(s.reportURL(site.Name)))
}

func (s *SiteHostingService) allowUnlockAttempt(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unlockTries == nil || len(s.unlockTries) > 50000 {
		s.unlockTries = map[string][]time.Time{}
	}
	now := s.now()
	recent := s.unlockTries[key][:0]
	for _, t := range s.unlockTries[key] {
		if now.Sub(t) < siteUnlockWindow {
			recent = append(recent, t)
		}
	}
	if len(recent) >= siteUnlockAttempts {
		s.unlockTries[key] = recent
		return false
	}
	s.unlockTries[key] = append(recent, now)
	return true
}

// resolveSiteFile maps a URL path to a file: /a/ → a/index.html, /a → a, a.html or (redirect) a/.
func resolveSiteFile(root, reqPath string, trailingSlash bool) (file string, redirect bool) {
	rel := filepath.FromSlash(strings.TrimPrefix(reqPath, "/"))
	isFile := func(p string) bool {
		info, err := os.Lstat(p)
		return err == nil && info.Mode().IsRegular()
	}
	if trailingSlash || reqPath == "/" {
		if p := filepath.Join(root, rel, "index.html"); isFile(p) {
			return p, false
		}
		return "", false
	}
	if p := filepath.Join(root, rel); isFile(p) {
		return p, false
	}
	if path.Ext(reqPath) == "" {
		if p := filepath.Join(root, rel+".html"); isFile(p) {
			return p, false
		}
		if isFile(filepath.Join(root, rel, "index.html")) {
			return "", true
		}
	}
	return "", false
}

func (s *SiteHostingService) serveFile(w http.ResponseWriter, r *http.Request, site *Site, version int, preview bool, file string, status int, clientIP string) {
	ctype := SiteContentType(file)
	if ctype == "" {
		s.statusPage(w, http.StatusNotFound, "页面不存在", "这个网站里没有这个页面。")
		return
	}
	data, err := os.ReadFile(file) // #nosec G304 -- path resolved inside the site's version directory
	if err != nil {
		s.statusPage(w, http.StatusNotFound, "页面不存在", "这个网站里没有这个页面。")
		return
	}
	h := w.Header()
	h.Set("Content-Type", ctype)
	page := strings.HasPrefix(ctype, "text/html")
	switch {
	case preview:
		if page {
			data = injectSiteBadge(data, s.previewBadge(version))
		}
	case page:
		data = injectSiteBadge(data, s.badge(site.Name))
		h.Set("Cache-Control", "public, max-age=60")
	case site.PasswordHash != "":
		h.Set("Cache-Control", "private, max-age=300")
	default:
		h.Set("Cache-Control", "public, max-age=300")
	}
	if page && site.PasswordHash != "" && !preview {
		h.Set("Cache-Control", "private, no-cache")
	}
	if !preview {
		s.stats.record(site.ID, clientIP, page && status == http.StatusOK, len(data))
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			_, _ = w.Write(data)
		}
		return
	}
	h.Set("ETag", fmt.Sprintf(`"%d-%d-%x"`, site.ID, version, len(data)))
	http.ServeContent(w, r, "", site.UpdatedAt.Truncate(time.Second), bytes.NewReader(data))
}

func (s *SiteHostingService) reportURL(name string) string {
	return s.mainSiteURL + "/site-report?site=" + url.QueryEscape(name)
}

func (s *SiteHostingService) badge(name string) []byte {
	return []byte(`<a href="` + html.EscapeString(s.reportURL(name)) + `" target="_blank" rel="noopener" data-hivegpt-badge ` +
		`style="all:initial;position:fixed;right:8px;bottom:8px;z-index:2147483647;font:12px/20px -apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;` +
		`background:rgba(0,0,0,.55);color:#fff;padding:0 8px;border-radius:10px;text-decoration:none;cursor:pointer">由 HiveGPT 托管 · 举报</a>`)
}

func (s *SiteHostingService) previewBadge(version int) []byte {
	return []byte(fmt.Sprintf(`<div data-hivegpt-badge style="all:initial;position:fixed;left:8px;top:8px;z-index:2147483647;font:13px/22px -apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;`+
		`background:#d97706;color:#fff;padding:0 10px;border-radius:11px">预览：第 %d 版（未上线）</div>`, version))
}

// injectSiteBadge puts the badge right before the last </body> (or at the end).
func injectSiteBadge(page, badge []byte) []byte {
	idx := bytes.LastIndex(bytes.ToLower(page), []byte("</body"))
	out := make([]byte, 0, len(page)+len(badge))
	if idx < 0 {
		return append(append(out, page...), badge...)
	}
	out = append(out, page[:idx]...)
	out = append(out, badge...)
	return append(out, page[idx:]...)
}

func (s *SiteHostingService) statusPage(w http.ResponseWriter, status int, title, message string) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title></head>`+
		`<body style="margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:-apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;background:#f5f5f4;color:#1c1917">`+
		`<div style="text-align:center;padding:24px"><h1 style="font-size:22px;margin:0 0 8px">%s</h1><p style="margin:0 0 16px;color:#57534e">%s</p>`+
		`<a href="%s" style="color:#0d9488">由 HiveGPT 提供网站托管</a></div></body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(message), html.EscapeString(s.mainSiteURL))
}
