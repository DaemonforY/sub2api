//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// libRepoFake keeps poems and songs in memory; matching is plain substring search.
type libRepoFake struct {
	poems  []LibPoem
	songs  map[int64]*LibSong
	nextID int64
	loads  int
}

func newLibRepoFake() *libRepoFake {
	return &libRepoFake{songs: map[int64]*LibSong{}, nextID: 1, poems: []LibPoem{
		{ID: 1, Collection: "lidai", Title: "村居", Author: "高鼎", Dynasty: "清", Content: "草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。", Tags: []string{}},
		{ID: 2, Collection: "song_ci", Title: "水调歌头·明月几时有", Author: "苏轼", Dynasty: "宋", Content: "明月几时有，把酒问青天。\n但愿人长久，千里共婵娟。", Tags: []string{}, Famous: true},
		{ID: 3, Collection: "tang", Title: "春晓", Author: "孟浩然", Dynasty: "唐", Content: "春眠不觉晓，处处闻啼鸟。", Tags: []string{}, Famous: true},
	}}
}

func (r *libRepoFake) SearchPoems(_ context.Context, q LibPoemQuery) ([]LibPoem, bool, error) {
	var out []LibPoem
	for _, p := range r.poems {
		ok := true
		for _, t := range libTerms(q.Q) {
			ok = ok && strings.Contains(p.Title+p.Author+p.Content, t)
		}
		if ok {
			out = append(out, p)
		}
	}
	return out, false, nil
}

func (r *libRepoFake) GetPoem(_ context.Context, id int64) (*LibPoem, error) {
	for _, p := range r.poems {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, nil
}
func (r *libRepoFake) StarPoem(context.Context, int64, bool) error         { return nil }
func (r *libRepoFake) PoemCounts(context.Context) (map[string]int, error) { return map[string]int{}, nil }

func (r *libRepoFake) FindPoems(_ context.Context, words []string, _ LibPoemQuery, limit int) ([]LibPoem, error) {
	var out []LibPoem
	for _, p := range r.poems {
		n := 0
		for _, w := range words {
			if strings.Contains(p.Title+p.Content, w) {
				n++
			}
		}
		if n > 0 {
			p.Score = n
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *libRepoFake) FindPoemByTitle(_ context.Context, title, author string) (*LibPoem, error) {
	for _, p := range r.poems {
		if strings.HasPrefix(p.Title, title) && (author == "" || p.Author == author) {
			return &p, nil
		}
	}
	return nil, nil
}

func (r *libRepoFake) PoemsContain(_ context.Context, han string) (bool, error) {
	for _, p := range r.poems {
		if strings.Contains(hanOnly(p.Content), han) {
			return true, nil
		}
	}
	return false, nil
}

func (r *libRepoFake) ListSongs(_ context.Context, q LibSongQuery) ([]LibSong, int, error) {
	out := []LibSong{}
	for id := int64(1); id < r.nextID; id++ {
		if s, ok := r.songs[id]; ok && (q.Q == "" || strings.Contains(s.Title+s.Lyrics, q.Q)) {
			out = append(out, *s)
		}
	}
	return out, len(out), nil
}

func (r *libRepoFake) GetSong(_ context.Context, id int64) (*LibSong, error) {
	if s, ok := r.songs[id]; ok {
		c := *s
		return &c, nil
	}
	return nil, nil
}

func (r *libRepoFake) CreateSong(_ context.Context, s *LibSong) error {
	s.ID = r.nextID
	r.nextID++
	c := *s
	r.songs[s.ID] = &c
	return nil
}

func (r *libRepoFake) UpdateSong(_ context.Context, s *LibSong) error {
	c := *s
	r.songs[s.ID] = &c
	return nil
}
func (r *libRepoFake) DeleteSong(_ context.Context, id int64) error { delete(r.songs, id); return nil }
func (r *libRepoFake) StarSong(context.Context, int64, bool) error  { return nil }
func (r *libRepoFake) SongFacets(context.Context) (*LibSongFacets, error) {
	return &LibSongFacets{}, nil
}

func (r *libRepoFake) AllLyrics(context.Context) ([]LibSong, error) {
	r.loads++
	var out []LibSong
	for _, s := range r.songs {
		out = append(out, LibSong{ID: s.ID, Title: s.Title, Lyrics: s.Lyrics})
	}
	return out, nil
}

// Made-up lyrics: one original line, one that quotes 苏轼.
const libTestLyrics = "窗外的雨停在青石板旁边\n我在等一封迟到的回信\n但愿人长久千里共婵娟"

func TestLyricsGuardRefusesLyricLinesButNotPoemQuotes(t *testing.T) {
	repo := newLibRepoFake()
	s := NewInspirationService(repo, nil)
	ctx := context.Background()
	song, err := s.CreateSong(ctx, 1, LibSong{Title: "某歌", Lyrics: libTestLyrics})
	require.NoError(t, err)

	hits, err := s.Guard().Check(ctx, "画面：窗外的雨停在，青石板旁边的小巷")
	require.NoError(t, err)
	require.Len(t, hits, 1, "punctuation in between doesn't hide a lyric line")
	require.Equal(t, song.ID, hits[0].SongID)
	require.Equal(t, "某歌", hits[0].SongTitle)

	// The line the song borrows from 苏轼 is public domain.
	hits, err = s.Guard().Check(ctx, "但愿人长久，千里共婵娟。")
	require.NoError(t, err)
	require.Empty(t, hits)
	// Short overlaps are fine.
	hits, err = s.Guard().Check(ctx, "一封回信，雨停了")
	require.NoError(t, err)
	require.Empty(t, hits)

	err = s.Guard().Refuse(ctx, "我在等一封迟到的回信")
	require.Error(t, err)
	require.Contains(t, err.Error(), "歌词")

	// Saving a song reloads the lyrics; a deleted song no longer counts.
	require.NoError(t, s.DeleteSong(ctx, song.ID))
	hits, err = s.Guard().Check(ctx, "我在等一封迟到的回信")
	require.NoError(t, err)
	require.Empty(t, hits)
	require.Equal(t, 2, repo.loads)
}

func TestShihuaRefusesWorksAndScenesWithLyrics(t *testing.T) {
	shihua, _ := newShihuaTestService(t)
	insp := ProvideInspirationService(newLibRepoFake(), shihua, nil)
	ctx := context.Background()
	_, err := insp.CreateSong(ctx, 1, LibSong{Title: "某歌", Lyrics: libTestLyrics})
	require.NoError(t, err)

	in := shihuaTestWork()
	in.Theme = &ShihuaTheme{PromptTheme: "一个人在窗前，窗外的雨停在青石板旁边"}
	_, err = shihua.CreateWork(ctx, in)
	require.Error(t, err)

	in.Theme = &ShihuaTheme{PromptTheme: "初秋傍晚，雨后空山"}
	w, err := shihua.CreateWork(ctx, in)
	require.NoError(t, err)
	_, err = shihua.Redraw(ctx, w.ID, ShihuaRedrawInput{Count: 1, PromptTheme: "我在等一封迟到的回信"})
	require.Error(t, err)

	res, err := shihua.Import(ctx, []ShihuaWorkInput{{Slug: "x-song", Title: "某题", Stage: "primary", FullText: "我在等一封迟到的回信"}}, true)
	require.NoError(t, err)
	require.Len(t, res.Errors, 1)
	require.Zero(t, res.Created)
}

func TestInspirationSongsHideLyricsInLists(t *testing.T) {
	s := NewInspirationService(newLibRepoFake(), nil)
	ctx := context.Background()
	_, err := s.CreateSong(ctx, 1, LibSong{Title: "某歌", Lyrics: libTestLyrics, Imagery: []string{" 青石板 ", "青石板", ""}})
	require.NoError(t, err)
	list, total, err := s.ListSongs(ctx, LibSongQuery{Q: "回信"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Empty(t, list[0].Lyrics)
	require.True(t, list[0].HasLyrics)
	require.Equal(t, []string{"我在等一封迟到的回信"}, list[0].Matches)
	require.Equal(t, []string{"青石板"}, list[0].Imagery)

	full, err := s.GetSong(ctx, list[0].ID)
	require.NoError(t, err)
	require.Equal(t, libTestLyrics, full.Lyrics)

	_, err = s.CreateSong(ctx, 1, LibSong{Title: "坏链接", LinkURL: "javascript:alert(1)"})
	require.Error(t, err)
}

func TestInspirationFindPicksOnlyRealPoems(t *testing.T) {
	repo := newLibRepoFake()
	s := NewInspirationService(repo, nil)
	ctx := context.Background()
	_, err := s.CreateSong(ctx, 1, LibSong{Title: "风筝歌", Lyrics: "东风吹着我的风筝飞得很高很高"})
	require.NoError(t, err)
	var calls []string
	s.chat = func(_ context.Context, system, user string) (string, string, error) {
		calls = append(calls, user)
		if strings.Contains(system, "检索词") {
			return `{"keywords":["纸鸢","东风","儿童","a long modern phrase","纸鸢"]}`, "m", nil
		}
		// 999 is not a candidate: dropped.
		return `好的 {"picks":[{"id":999,"reason":"编的"},{"id":1,"reason":"儿童放纸鸢，正合画面"}]}`, "m", nil
	}
	res, err := s.FindByScene(ctx, LibFindRequest{Scene: "春天孩子放风筝"})
	require.NoError(t, err)
	require.Equal(t, []string{"纸鸢", "东风", "儿童"}, res.Keywords)
	require.Len(t, res.Picks, 1)
	require.Equal(t, int64(1), res.Picks[0].Poem.ID)
	require.Equal(t, "儿童放纸鸢，正合画面", res.Picks[0].Reason)
	require.Contains(t, calls[1], "id=1 《村居》")
	require.Len(t, res.Songs, 1)
	require.Empty(t, res.Songs[0].Lyrics)
	require.NotEmpty(t, res.Songs[0].Matches)

	s.chat = nil
	_, err = s.FindByScene(ctx, LibFindRequest{Scene: "x"})
	require.ErrorIs(t, err, ErrLibNoModel)
}

func TestInspirationAnalyzeDropsWhatRepeatsTheLyrics(t *testing.T) {
	s := NewInspirationService(newLibRepoFake(), nil)
	ctx := context.Background()
	song, err := s.CreateSong(ctx, 1, LibSong{Title: "某歌", Lyrics: libTestLyrics})
	require.NoError(t, err)
	s.chat = func(context.Context, string, string) (string, string, error) {
		return `{"summary":"歌里唱我在等一封迟到的回信，写等待","imagery":["青石板","雨巷"],"moods":["等待"],"tags":["民谣"],
"allusions":[{"title":"水调歌头","author":"苏轼","note":"借中秋怀人的句子收尾"},{"title":"","author":"x","note":"空标题丢掉"}]}`, "m", nil
	}
	a, err := s.AnalyzeSong(ctx, song.ID)
	require.NoError(t, err)
	require.Empty(t, a.Summary, "the summary quoted a lyric line")
	require.Equal(t, []string{"summary"}, a.Dropped)
	require.Equal(t, []string{"青石板", "雨巷"}, a.Imagery)
	require.Len(t, a.Allusions, 1)
	require.Equal(t, int64(2), *a.Allusions[0].PoemID)

	empty, err := s.CreateSong(ctx, 1, LibSong{Title: "无词"})
	require.NoError(t, err)
	_, err = s.AnalyzeSong(ctx, empty.ID)
	require.Error(t, err)
}

func TestInspirationToShihuaMakesADraftOnce(t *testing.T) {
	shihua, _ := newShihuaTestService(t)
	s := NewInspirationService(newLibRepoFake(), shihua)
	ctx := context.Background()
	w, created, err := s.ToShihua(ctx, 1, "primary")
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "poem-1", w.Slug)
	require.Equal(t, ShihuaTextDraft, w.TextStatus)
	require.Contains(t, w.Sources, "请对照课本校对")

	again, created, err := s.ToShihua(ctx, 1, "primary")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, w.ID, again.ID)

	_, _, err = s.ToShihua(ctx, 404, "")
	require.True(t, errors.Is(err, ErrLibPoemNotFound))
}
