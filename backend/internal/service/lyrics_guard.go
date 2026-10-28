package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Lyrics in the 灵感库 are copyrighted and kept for internal reference only. LyricsGuard refuses
// texts that are about to become output (课本诗画 work texts, scenes and image prompts, 素材库 prompts)
// when they contain a lyric line: lyricsGramLen Chinese characters in a row from one line of stored
// lyrics. A run that also appears in the public-domain poem corpus is allowed, since songs often quote
// classical poems.

const lyricsGramLen = 7

type lyricsSource interface {
	// AllLyrics returns every song's id, title and lyrics.
	AllLyrics(ctx context.Context) ([]LibSong, error)
	// PoemsContain reports whether a public-domain poem contains the characters in a row (punctuation
	// between them allowed).
	PoemsContain(ctx context.Context, han string) (bool, error)
}

// LyricsHit is a lyric run found in a text.
type LyricsHit struct {
	Run       string `json:"run"`
	SongID    int64  `json:"song_id"`
	SongTitle string `json:"song_title"`
}

type LyricsGuard struct {
	src lyricsSource

	mu     sync.Mutex
	grams  map[string]int64 // run → song id
	titles map[int64]string
	ready  bool
	public map[string]bool // runs found in the poem corpus (or not)
}

func NewLyricsGuard(src lyricsSource) *LyricsGuard {
	return &LyricsGuard{src: src, public: map[string]bool{}}
}

// Invalidate makes the next check reload the lyrics (after a song is saved or deleted).
func (g *LyricsGuard) Invalidate() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.ready = false
	g.mu.Unlock()
}

func (g *LyricsGuard) load(ctx context.Context) error {
	if g.ready {
		return nil
	}
	songs, err := g.src.AllLyrics(ctx)
	if err != nil {
		return err
	}
	g.grams, g.titles = map[string]int64{}, map[int64]string{}
	for _, s := range songs {
		g.titles[s.ID] = s.Title
		for _, line := range strings.Split(s.Lyrics, "\n") {
			for _, run := range lyricsGrams(hanOnly(line)) {
				if _, ok := g.grams[run]; !ok {
					g.grams[run] = s.ID
				}
			}
		}
	}
	g.ready = true
	return nil
}

// Check returns the lyric runs a text contains (at most one per song), leaving out runs that are
// quotes of public-domain poems.
func (g *LyricsGuard) Check(ctx context.Context, texts ...string) ([]LyricsHit, error) {
	if g == nil || g.src == nil {
		return nil, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.load(ctx); err != nil {
		return nil, err
	}
	if len(g.grams) == 0 {
		return nil, nil
	}
	var hits []LyricsHit
	seen := map[int64]bool{}
	for _, t := range texts {
		han := hanOnly(t)
		for _, run := range lyricsGrams(han) {
			id, ok := g.grams[run]
			if !ok || seen[id] {
				continue
			}
			public, known := g.public[run]
			if !known {
				p, err := g.src.PoemsContain(ctx, run)
				if err != nil {
					return nil, err
				}
				public = p
				g.public[run] = p
			}
			if public {
				continue
			}
			seen[id] = true
			hits = append(hits, LyricsHit{Run: run, SongID: id, SongTitle: g.titles[id]})
		}
	}
	return hits, nil
}

// Refuse is Check as an error for the save paths: nil when there is no lyric in the texts.
func (g *LyricsGuard) Refuse(ctx context.Context, texts ...string) error {
	hits, err := g.Check(ctx, texts...)
	if err != nil || len(hits) == 0 {
		return err
	}
	h := hits[0]
	return infraerrors.BadRequest("LYRICS_IN_WORK", fmt.Sprintf(
		"内容里有歌词原句「%s」（来自灵感库里的《%s》）。歌词只能作内部参考，不能用在作品、画面描述或提示词里，请改成自己的话（Lyrics can't be used in works）",
		h.Run, h.SongTitle))
}

func hanOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			_, _ = b.WriteRune(r)
		}
	}
	return b.String()
}

func lyricsGrams(han string) []string {
	rs := []rune(han)
	if len(rs) < lyricsGramLen {
		return nil
	}
	out := make([]string, 0, len(rs)-lyricsGramLen+1)
	for i := 0; i+lyricsGramLen <= len(rs); i++ {
		out = append(out, string(rs[i:i+lyricsGramLen]))
	}
	return out
}
