//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInspirationRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewInspirationRepository(integrationDB)
	tag := fmt.Sprintf("测%d", time.Now().UnixNano()%1000000)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM lib_poems WHERE author LIKE $1`, tag+"%")
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM lib_songs WHERE title LIKE $1`, tag+"%")
	})
	insert := func(coll, title, author, dynasty, content string, famous bool, pop float64) int64 {
		var id int64
		require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO lib_poems (collection, title, author, dynasty, content, famous, popularity)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`, coll, title, tag+author, dynasty, content, famous, pop).Scan(&id))
		return id
	}
	cunju := insert("lidai", "村居", "高鼎", "清", "草长莺飞二月天，拂堤杨柳醉春烟。\n儿童散学归来早，忙趁东风放纸鸢。", false, 0)
	insert("tang", "春晓", "孟浩然", "唐", "春眠不觉晓，处处闻啼鸟。\n夜来风雨声，花落知多少。", true, 9)
	insert("song_shi", "纸鸢", "无名", "宋", "东风吹纸鸢，儿童笑语喧。", false, 2)

	// Words are all required; famous and popular first; has_more pages.
	poems, more, err := repo.SearchPoems(ctx, service.LibPoemQuery{Q: tag, Limit: 2})
	require.NoError(t, err)
	require.True(t, more)
	require.Equal(t, "春晓", poems[0].Title)
	poems, _, err = repo.SearchPoems(ctx, service.LibPoemQuery{Q: tag + " 纸鸢 儿童", Limit: 10})
	require.NoError(t, err)
	require.Len(t, poems, 2)
	poems, _, err = repo.SearchPoems(ctx, service.LibPoemQuery{Q: tag, Dynasty: "清", Limit: 10})
	require.NoError(t, err)
	require.Len(t, poems, 1)
	require.Equal(t, cunju, poems[0].ID)
	poems, _, err = repo.SearchPoems(ctx, service.LibPoemQuery{Q: "100%_" + tag, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, poems, "% and _ are literal")

	// Stars.
	require.NoError(t, repo.StarPoem(ctx, cunju, true))
	poems, _, err = repo.SearchPoems(ctx, service.LibPoemQuery{Q: tag, Starred: true, Limit: 10})
	require.NoError(t, err)
	require.Len(t, poems, 1)
	require.True(t, poems[0].Starred)

	// Find: the most words first.
	found, err := repo.FindPoems(ctx, []string{"纸鸢", "东风", "儿童", "杨柳", tag}, service.LibPoemQuery{}, 10)
	require.NoError(t, err)
	require.Equal(t, cunju, found[0].ID)
	require.Equal(t, 5, found[0].Score)

	p, err := repo.FindPoemByTitle(ctx, "村居", tag+"高鼎")
	require.NoError(t, err)
	require.Equal(t, cunju, p.ID)
	p, err = repo.FindPoemByTitle(ctx, "没有这首", "")
	require.NoError(t, err)
	require.Nil(t, p)

	// A run inside one line, and one across the comma between two half-lines.
	ok, err := repo.PoemsContain(ctx, "草长莺飞二月天")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.PoemsContain(ctx, "月天拂堤杨柳醉")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = repo.PoemsContain(ctx, "完全不在诗里的句")
	require.NoError(t, err)
	require.False(t, ok)

	counts, err := repo.PoemCounts(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, counts["lidai"], 1)

	// Songs.
	year := 2003
	song := &service.LibSong{Title: tag + "某歌", Artist: "某歌手", Year: &year, Lyrics: "第一行歌词有东风\n第二行没有\n第三行也写东风",
		Imagery: []string{"东风", "纸鸢"}, Moods: []string{"怀旧"}, Tags: []string{"中国风"},
		Allusions: []service.LibAllusion{{Title: "村居", Author: "高鼎", PoemID: &cunju, Note: "放风筝"}}}
	require.NoError(t, repo.CreateSong(ctx, song))
	require.NotZero(t, song.ID)
	got, err := repo.GetSong(ctx, song.ID)
	require.NoError(t, err)
	require.Equal(t, 2003, *got.Year)
	require.Equal(t, cunju, *got.Allusions[0].PoemID)
	require.Equal(t, []string{"东风", "纸鸢"}, got.Imagery)

	list, total, err := repo.ListSongs(ctx, service.LibSongQuery{Q: tag + " 东风", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Contains(t, list[0].Lyrics, "第一行")
	list, total, err = repo.ListSongs(ctx, service.LibSongQuery{Q: tag, Tag: "怀旧", YearFrom: 2000, YearTo: 2005, Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	_, total, err = repo.ListSongs(ctx, service.LibSongQuery{Q: tag, YearFrom: 2010, Limit: 10})
	require.NoError(t, err)
	require.Zero(t, total)

	got.Summary, got.Year = "改过", nil
	require.NoError(t, repo.UpdateSong(ctx, got))
	got, err = repo.GetSong(ctx, song.ID)
	require.NoError(t, err)
	require.Equal(t, "改过", got.Summary)
	require.Nil(t, got.Year)

	facets, err := repo.SongFacets(ctx)
	require.NoError(t, err)
	require.Contains(t, facets.Artists, service.LibFacet{Name: "某歌手", Count: 1})
	all, err := repo.AllLyrics(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, all)

	require.NoError(t, repo.StarSong(ctx, song.ID, true))
	require.NoError(t, repo.DeleteSong(ctx, song.ID))
	got, err = repo.GetSong(ctx, song.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}
