package web

import (
	"bytes"
	htmlpkg "html"
	"io/fs"
	"regexp"
	"strings"
	"sync"
)

// Lesson pages as plain text for the AI tutor: the built /learn page's title and article text.

var (
	learnTitleRe   = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	learnDropRe    = regexp.MustCompile(`(?is)<(script|style|svg|button)\b.*?</(script|style|svg|button)>`)
	learnBlockRe   = regexp.MustCompile(`(?i)</?(p|div|h[1-6]|li|tr|pre|br|table|blockquote|summary|details)\b[^>]*>`)
	learnTagRe     = regexp.MustCompile(`(?s)<[^>]+>`)
	learnSpaceRe   = regexp.MustCompile(`[ \t\r\f\v]+`)
	learnNewlineRe = regexp.MustCompile(`\n\s*\n+`)
	learnLessonRe  = regexp.MustCompile(`^[a-z][0-9]{1,2}$`)
	learnArticleRe = regexp.MustCompile(`^bigdata/(spark|flink|paimon)/[a-z0-9-]{1,40}$`)
	learnTextCache sync.Map
)

type learnText struct{ title, text string }

// lessonTextFrom reads learn/<track>/<id>.html (a lesson) or learn/<path>.html (a 大数据 article).
func lessonTextFrom(fsys fs.FS, lessonID string) (string, string) {
	if fsys == nil {
		return "", ""
	}
	file := ""
	switch {
	case learnLessonRe.MatchString(lessonID):
		file = "learn/" + lessonID[:1] + "/" + lessonID + ".html"
	case learnArticleRe.MatchString(lessonID):
		file = "learn/" + lessonID + ".html"
	default:
		return "", ""
	}
	if v, ok := learnTextCache.Load(lessonID); ok {
		if t, ok := v.(learnText); ok {
			return t.title, t.text
		}
	}
	raw, err := fs.ReadFile(fsys, file)
	if err != nil {
		return "", ""
	}
	title, text := pageText(raw)
	learnTextCache.Store(lessonID, learnText{title, text})
	return title, text
}

// pageText is a built VitePress page's title (without the " · site" suffix) and its article text.
func pageText(raw []byte) (string, string) {
	page := string(raw)
	title := ""
	if m := learnTitleRe.FindStringSubmatch(page); m != nil {
		title = htmlpkg.UnescapeString(m[1])
		if i := strings.Index(title, " · "); i > 0 {
			title = title[:i]
		}
		if i := strings.Index(title, " | "); i > 0 {
			title = title[:i]
		}
	}
	body := page
	if i := strings.Index(body, `class="vp-doc`); i >= 0 {
		body = body[i:]
	}
	if i := strings.Index(body, "</main>"); i >= 0 {
		body = body[:i]
	}
	body = learnDropRe.ReplaceAllString(body, " ")
	body = learnBlockRe.ReplaceAllString(body, "\n")
	body = learnTagRe.ReplaceAllString(body, "")
	body = htmlpkg.UnescapeString(body)
	body = learnSpaceRe.ReplaceAllString(body, " ")
	body = learnNewlineRe.ReplaceAllString(body, "\n")
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(body), `class="vp-doc`))
	if i := strings.Index(text, ">"); i >= 0 && i < 200 && !strings.Contains(text[:i], "\n") {
		text = strings.TrimSpace(text[i+1:])
	}
	return title, text
}

// LearnPage is one built page of the learning site as plain text (the support assistant's knowledge).
type LearnPage struct {
	URL   string // e.g. /learn/codex/hivegpt
	Title string
	Text  string
}

// learnPagesFrom lists every page under learn/ (except the 404 page), skipping pages without text.
func learnPagesFrom(fsys fs.FS) []LearnPage {
	if fsys == nil {
		return nil
	}
	var out []LearnPage
	_ = fs.WalkDir(fsys, "learn", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(name, ".html") || name == "learn/404.html" ||
			strings.HasPrefix(name, "learn/assets/") {
			return nil
		}
		raw, err := fs.ReadFile(fsys, name)
		if err != nil || !bytes.Contains(raw, []byte(`class="vp-doc`)) {
			return nil
		}
		title, text := pageText(raw)
		if strings.TrimSpace(text) == "" {
			return nil
		}
		url := "/" + strings.TrimSuffix(name, ".html")
		if strings.HasSuffix(url, "/index") {
			url = strings.TrimSuffix(url, "index")
		}
		out = append(out, LearnPage{URL: url, Title: title, Text: text})
		return nil
	})
	return out
}
