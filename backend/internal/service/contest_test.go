package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func contestAt(base time.Time) *Contest {
	return &Contest{
		Title: "AI 绘画大赛", Status: ContestStatusPublished,
		SubmissionStartAt: base, SubmissionEndAt: base.Add(48 * time.Hour),
		VotingStartAt: base.Add(24 * time.Hour), VotingEndAt: base.Add(72 * time.Hour),
		MaxEntriesPerUser: 1, VotesPerUser: 3,
	}
}

func TestContestPhaseAt(t *testing.T) {
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	c := contestAt(base)
	cases := []struct {
		at   time.Time
		want string
	}{
		{base.Add(-time.Minute), ContestPhaseUpcoming},
		{base, ContestPhaseSubmitting},
		{base.Add(30 * time.Hour), ContestPhaseSubmitAndVote},
		{base.Add(50 * time.Hour), ContestPhaseVoting},
		{base.Add(72 * time.Hour), ContestPhaseTallying},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, ContestPhaseAt(c, tc.at), tc.at)
	}
	c.Status = ContestStatusSettled
	require.Equal(t, ContestPhaseSettled, ContestPhaseAt(c, base))
	c.Status = ContestStatusDraft
	require.Equal(t, ContestPhaseDraft, ContestPhaseAt(c, base))

	gap := contestAt(base)
	gap.VotingStartAt = base.Add(60 * time.Hour)
	require.Equal(t, ContestPhaseWaitingVote, ContestPhaseAt(gap, base.Add(50*time.Hour)))
	require.False(t, ContestAcceptsVotes(gap, base.Add(50*time.Hour)))
	require.False(t, ContestAcceptsEntries(gap, base.Add(50*time.Hour)))
}

func TestNormalizeAndValidateContest(t *testing.T) {
	base := time.Now()
	ok := contestAt(base)
	ok.Prizes = []ContestPrize{
		{RankFrom: 2, RankTo: 3, Type: ContestPrizeBalance, Amount: 5, Label: "二等奖"},
		{RankFrom: 1, RankTo: 1, Type: ContestPrizeBalance, Amount: 20, Label: "一等奖"},
		{RankFrom: 4, RankTo: 10, Type: ContestPrizeCustom, Amount: 99, Label: "周边礼包"},
	}
	require.NoError(t, NormalizeAndValidateContest(ok))
	require.Equal(t, 1, ok.Prizes[0].RankFrom, "prizes sorted by rank")
	require.Zero(t, ok.Prizes[2].Amount, "custom prizes carry no amount")

	mutate := func(f func(c *Contest)) error {
		c := contestAt(base)
		f(c)
		return NormalizeAndValidateContest(c)
	}
	require.Error(t, mutate(func(c *Contest) { c.Title = " " }))
	require.Error(t, mutate(func(c *Contest) { c.VotingEndAt = c.SubmissionEndAt.Add(-time.Hour) }))
	require.Error(t, mutate(func(c *Contest) { c.VotingStartAt = c.SubmissionStartAt.Add(-time.Hour) }))
	require.Error(t, mutate(func(c *Contest) { c.VotesPerUser = 0 }))
	require.Error(t, mutate(func(c *Contest) { c.CoverImage = "javascript:alert(1)" }))
	require.Error(t, mutate(func(c *Contest) {
		c.Prizes = []ContestPrize{{RankFrom: 1, RankTo: 3, Type: ContestPrizeBalance, Amount: 1}, {RankFrom: 3, RankTo: 4, Type: ContestPrizeBalance, Amount: 1}}
	}), "overlapping ranges")
	require.Error(t, mutate(func(c *Contest) { c.Prizes = []ContestPrize{{RankFrom: 1, RankTo: 1, Type: ContestPrizeBalance}} }), "balance needs amount")
	require.Error(t, mutate(func(c *Contest) { c.Prizes = []ContestPrize{{RankFrom: 1, RankTo: 1, Type: ContestPrizeCustom}} }), "custom needs label")
	require.Error(t, mutate(func(c *Contest) { c.Prizes = []ContestPrize{{RankFrom: 1, RankTo: 1, Type: "cash", Amount: 1}} }))
}

func TestAssignContestRanks_TieBreakers(t *testing.T) {
	t0 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	at := func(h int) *time.Time { v := t0.Add(time.Duration(h) * time.Hour); return &v }
	ranked := []ContestRankedEntry{
		{EntryID: 1, Votes: 5, LastVoteAt: at(10), CreatedAt: t0},
		{EntryID: 2, Votes: 7, LastVoteAt: at(20), CreatedAt: t0},
		{EntryID: 3, Votes: 5, LastVoteAt: at(5), CreatedAt: t0.Add(time.Hour)}, // reached 5 votes earlier than #1
		{EntryID: 4, Votes: 0, CreatedAt: t0.Add(2 * time.Hour)},
		{EntryID: 5, Votes: 0, CreatedAt: t0.Add(time.Hour)}, // no votes: earlier submission wins
	}
	AssignContestRanks(ranked)
	order := []int64{}
	for i, r := range ranked {
		require.Equal(t, i+1, r.Rank)
		order = append(order, r.EntryID)
	}
	require.Equal(t, []int64{2, 3, 1, 5, 4}, order)
}

func TestBuildContestAwards(t *testing.T) {
	prizes := []ContestPrize{
		{RankFrom: 1, RankTo: 1, Type: ContestPrizeBalance, Amount: 20, Label: "一等奖"},
		{RankFrom: 2, RankTo: 3, Type: ContestPrizeBalance, Amount: 5, Label: "二等奖"},
		{RankFrom: 5, RankTo: 5, Type: ContestPrizeCustom, Label: "幸运奖"},
	}
	ranked := []ContestRankedEntry{
		{EntryID: 10, UserID: 1, Votes: 9, Rank: 1},
		{EntryID: 11, UserID: 1, Votes: 8, Rank: 2}, // same user: skipped with one-prize-per-user
		{EntryID: 12, UserID: 2, Votes: 6, Rank: 3},
		{EntryID: 13, UserID: 3, Votes: 4, Rank: 4},
		{EntryID: 14, UserID: 4, Votes: 3, Rank: 5}, // place 4: tier gap, no prize
		{EntryID: 15, UserID: 5, Votes: 1, Rank: 6}, // place 5: custom prize
		{EntryID: 16, UserID: 6, Votes: 0, Rank: 7},
	}
	awards := BuildContestAwards(7, ranked, prizes, true, 1)
	got := map[int64]int{}
	for _, a := range awards {
		got[a.EntryID] = a.Place
		require.Equal(t, int64(7), a.ContestID)
		require.Equal(t, ContestAwardPending, a.Status)
	}
	require.Equal(t, map[int64]int{10: 1, 12: 2, 13: 3, 15: 5}, got)
	require.Equal(t, 20.0, awards[0].Amount)
	require.Equal(t, ContestPrizeCustom, awards[3].PrizeType)

	multi := BuildContestAwards(7, ranked, prizes, false, 1)
	require.Equal(t, int64(11), multi[1].EntryID, "same user may win twice when allowed")

	strict := BuildContestAwards(7, ranked, prizes, true, 5)
	require.Len(t, strict, 2, "entries below min votes never win")

	require.Empty(t, BuildContestAwards(7, ranked, nil, true, 1))
}

func TestContestAuthorName(t *testing.T) {
	require.Equal(t, "小明", ContestAuthorName("小明", "a@b.com"))
	require.Equal(t, "zh***@example.com", ContestAuthorName("", "zhangsan@example.com"))
	require.Equal(t, "a***@x.cn", ContestAuthorName("", "ab@x.cn"))
	require.Equal(t, "***", ContestAuthorName("", "broken"))
}

func TestContestImageStore(t *testing.T) {
	dir := t.TempDir()
	store := NewContestImageStore(dir)

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	name, err := store.Save(buf.Bytes())
	require.NoError(t, err)
	require.Regexp(t, `^[a-f0-9]{32}\.png$`, name)
	p, ok := store.Path(name)
	require.True(t, ok)
	require.FileExists(t, p)
	require.Equal(t, dir, filepath.Dir(p))

	_, err = store.Save([]byte("<svg onload=alert(1)>"))
	require.ErrorIs(t, err, ErrContestImageInvalid, "non-raster content is rejected by sniffing")
	_, ok = store.Path("../../etc/passwd")
	require.False(t, ok)
	_, ok = store.Path("deadbeef.png")
	require.False(t, ok)

	store.Remove(name)
	_, statErr := os.Stat(p)
	require.True(t, os.IsNotExist(statErr))
}
