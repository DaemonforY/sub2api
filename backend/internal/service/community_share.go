package service

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Share cards: the canvas site's nginx asks for these <head> tags when it serves /w/:id, /u/:handle
// and /c/:id, so link previews (QQ, Feishu, Telegram, search engines…) show the title, a
// description and the cover without running the SPA. Only public content gets a card.

const (
	shareMetaTTL       = time.Minute
	shareMetaSiteName  = "HiveGPT 无限画布"
	shareMetaMaxCached = 5000
)

type shareMetaEntry struct {
	html string
	at   time.Time
}

type shareMetaCache struct {
	mu      sync.Mutex
	entries map[string]shareMetaEntry
}

func (c *shareMetaCache) get(key string, now time.Time) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || now.Sub(e.at) > shareMetaTTL {
		return "", false
	}
	return e.html, true
}

func (c *shareMetaCache) put(key, value string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil || len(c.entries) > shareMetaMaxCached {
		c.entries = map[string]shareMetaEntry{}
	}
	c.entries[key] = shareMetaEntry{html: value, at: now}
}

// ShareCard is the content of one card.
type ShareCard struct {
	Title       string
	Description string
	// Image: a path on the main site (/api/v1/community/media/...) or an absolute https URL.
	Image       string
	ImageWidth  int
	ImageHeight int
	// Type: article (work, collection) or profile.
	Type string
}

func excerpt(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")
	return truncateRunes(text, n)
}

// shareCard builds the card for a canvas path ("/w/12", "/u/xiaolin", "/c/3"); nil when there is none.
func (s *CommunityService) shareCard(ctx context.Context, path string) (*ShareCard, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] == "" {
		return nil, nil
	}
	switch parts[0] {
	case "w":
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, nil
		}
		w, err := s.repo.GetWork(ctx, id)
		if err != nil || w == nil || w.Status != WorkStatusApproved || w.Visibility == WorkVisibilityPrivate || (w.Kind == WorkKindSite && !w.Site.Live()) {
			return nil, err
		}
		if p, err := s.repo.GetProfileByUser(ctx, w.UserID); err != nil || p == nil || p.Status != ProfileStatusActive {
			return nil, err
		}
		author := w.Author.DisplayName
		if author == "" {
			author = "@" + w.Author.Handle
		}
		title := w.Title
		if title == "" {
			title = "AI 作品"
			if w.Kind == WorkKindSite {
				title = "AI 网页"
			}
		}
		desc := w.Description
		if desc == "" && w.ShowPrompt {
			desc = w.Prompt
		}
		if desc == "" {
			desc = fmt.Sprintf("%s 在 %s 发布的 AI 作品", author, shareMetaSiteName)
		}
		card := &ShareCard{Title: fmt.Sprintf("%s · %s", title, author), Description: excerpt(desc, 120), Type: "article"}
		if len(w.Media) > 0 {
			m := w.Media[0]
			card.Image = CommunityMediaURL(m.ThumbFile)
			card.ImageWidth = min(m.Width, communityThumbWidth)
			if m.Width > 0 {
				card.ImageHeight = m.Height * card.ImageWidth / m.Width
			}
		}
		return card, nil
	case "u":
		p, err := s.repo.GetProfileByHandle(ctx, strings.ToLower(parts[1]))
		if err != nil || p == nil || p.Status != ProfileStatusActive {
			return nil, err
		}
		name := p.DisplayName
		if name == "" {
			name = p.Handle
		}
		desc := p.Bio
		if desc == "" {
			desc = fmt.Sprintf("%d 个作品 · %d 位粉丝 · 在 %s 创作", p.WorksCount, p.FollowersCount, shareMetaSiteName)
		}
		card := &ShareCard{Title: fmt.Sprintf("%s (@%s)", name, p.Handle), Description: excerpt(desc, 120), Type: "profile"}
		if p.AvatarFile != "" {
			card.Image, card.ImageWidth, card.ImageHeight = CommunityMediaURL(p.AvatarFile), communityAvatarSize, communityAvatarSize
		} else if works, err := s.repo.ListWorks(ctx, WorkQuery{Feed: "user", UserID: p.UserID, Limit: 1}); err == nil && len(works) > 0 {
			card.Image = CommunityMediaURL(works[0].CoverThumb)
		}
		return card, nil
	case "c":
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, nil
		}
		c, err := s.repo.GetCollection(ctx, id)
		if err != nil || c == nil || c.Visibility != "public" {
			return nil, err
		}
		desc := c.Description
		if desc == "" {
			desc = fmt.Sprintf("%d 个作品的作品集", c.WorksCount)
		}
		card := &ShareCard{Title: c.Title, Description: excerpt(desc, 120), Type: "article"}
		if c.Author != nil && c.Author.Handle != "" {
			name := c.Author.DisplayName
			if name == "" {
				name = "@" + c.Author.Handle
			}
			card.Title = fmt.Sprintf("%s · %s 的作品集", c.Title, name)
		}
		if len(c.CoverFiles) > 0 {
			card.Image = CommunityMediaURL(c.CoverFiles[0])
		}
		return card, nil
	}
	return nil, nil
}

// ShareMetaHTML renders the <head> tags for a canvas path. mainSite is the main site's origin (for
// image URLs); pageURL, when trusted, becomes og:url. Empty string: no card.
func (s *CommunityService) ShareMetaHTML(ctx context.Context, path, mainSite, pageURL string) (string, error) {
	key := path + "|" + pageURL
	if cached, ok := s.shareCache.get(key, s.now()); ok {
		return cached, nil
	}
	card, err := s.shareCard(ctx, path)
	if err != nil {
		return "", err
	}
	out := ""
	if card != nil {
		out = renderShareMeta(card, mainSite, pageURL)
	}
	s.shareCache.put(key, out, s.now())
	return out, nil
}

func renderShareMeta(card *ShareCard, mainSite, pageURL string) string {
	e := html.EscapeString
	var b strings.Builder
	title := card.Title + " · " + shareMetaSiteName
	fmt.Fprintf(&b, "<title>%s</title>\n", e(title))
	fmt.Fprintf(&b, "<meta name=\"description\" content=\"%s\" />\n", e(card.Description))
	fmt.Fprintf(&b, "<meta property=\"og:site_name\" content=\"%s\" />\n", e(shareMetaSiteName))
	fmt.Fprintf(&b, "<meta property=\"og:type\" content=\"%s\" />\n", e(card.Type))
	fmt.Fprintf(&b, "<meta property=\"og:title\" content=\"%s\" />\n", e(card.Title))
	fmt.Fprintf(&b, "<meta property=\"og:description\" content=\"%s\" />\n", e(card.Description))
	if pageURL != "" {
		fmt.Fprintf(&b, "<meta property=\"og:url\" content=\"%s\" />\n<link rel=\"canonical\" href=\"%s\" />\n", e(pageURL), e(pageURL))
	}
	if card.Image != "" {
		img := card.Image
		if strings.HasPrefix(img, "/") {
			img = strings.TrimRight(mainSite, "/") + img
		}
		fmt.Fprintf(&b, "<meta property=\"og:image\" content=\"%s\" />\n", e(img))
		if card.ImageWidth > 0 && card.ImageHeight > 0 {
			fmt.Fprintf(&b, "<meta property=\"og:image:width\" content=\"%d\" />\n<meta property=\"og:image:height\" content=\"%d\" />\n", card.ImageWidth, card.ImageHeight)
		}
		fmt.Fprintf(&b, "<meta name=\"twitter:card\" content=\"summary_large_image\" />\n<meta name=\"twitter:image\" content=\"%s\" />\n", e(img))
	} else {
		fmt.Fprint(&b, "<meta name=\"twitter:card\" content=\"summary\" />\n")
	}
	fmt.Fprintf(&b, "<meta name=\"twitter:title\" content=\"%s\" />\n", e(card.Title))
	return b.String()
}

// TrustedShareURL returns pageURL when it is an https URL on one of the allowed canvas origins.
func TrustedShareURL(pageURL string, allowedOrigins []string) string {
	u, err := url.Parse(pageURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	origin := u.Scheme + "://" + u.Host
	for _, allowed := range allowedOrigins {
		if strings.TrimRight(strings.TrimSpace(allowed), "/") == origin {
			u.RawQuery, u.Fragment = "", ""
			return u.String()
		}
	}
	return ""
}

// Restricted authors -----------------------------------------------------------------------------

// RestrictedAuthor is a profile banned from publishing (admin list).
type RestrictedAuthor struct {
	UserID      int64     `json:"user_id"`
	Email       string    `json:"email"`
	Handle      string    `json:"handle"`
	DisplayName string    `json:"display_name"`
	WorksCount  int       `json:"works_count"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *CommunityService) AdminRestricted(ctx context.Context) ([]RestrictedAuthor, error) {
	list, err := s.repo.ListRestrictedProfiles(ctx, 200)
	if list == nil {
		list = []RestrictedAuthor{}
	}
	return list, err
}
