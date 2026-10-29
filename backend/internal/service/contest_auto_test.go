package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func cst(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, contestAutoZone)
}

func TestBuildAutoContest(t *testing.T) {
	tpl := DefaultContestAutoTemplate()
	c, err := BuildAutoContest(tpl, 2026, time.November)
	require.NoError(t, err)
	require.Equal(t, "11 月 AI 画图赛", c.Title)
	require.Equal(t, "2026-11", c.AutoPeriod)
	require.Equal(t, ContestStatusPublished, c.Status)
	require.True(t, c.SubmissionStartAt.Equal(cst(2026, 11, 1, 0, 0)))
	require.True(t, c.VotingStartAt.Equal(cst(2026, 11, 1, 0, 0)))
	require.True(t, c.SubmissionEndAt.Equal(cst(2026, 11, 22, 23, 59).Add(59*time.Second)), "submissions run to the end of the 22nd")
	require.True(t, c.VotingEndAt.Equal(cst(2026, 11, 30, 20, 0)))
	require.Equal(t, 5, c.VotesPerUser)
	require.Equal(t, 5, c.MinVotesForPrize)
	require.Equal(t, 0, c.MinAccountAgeHours)
	require.True(t, c.RequireReview)
	require.Len(t, c.Prizes, 1)
	require.Equal(t, ContestPrizeCustom, c.Prizes[0].Type)

	// {year} / {month} in every text; December rolls into next year's January correctly.
	tpl.TitleTemplate = "{year} 年 {month} 月赛"
	tpl.Rules = "截止到 {month} 月底"
	c, err = BuildAutoContest(tpl, 2026, time.December)
	require.NoError(t, err)
	require.Equal(t, "2026 年 12 月赛", c.Title)
	require.Equal(t, "截止到 12 月底", c.Rules)
	require.True(t, c.VotingEndAt.Equal(cst(2026, 12, 31, 20, 0)))

	// A 28th cut-off in February would close after voting ends: it is clamped to the voting end.
	tpl.SubmissionEndDay = 28
	c, err = BuildAutoContest(tpl, 2027, time.February)
	require.NoError(t, err)
	require.True(t, c.SubmissionEndAt.Equal(cst(2027, 2, 28, 20, 0)))

	for _, bad := range []func(*ContestAutoTemplate){
		func(t *ContestAutoTemplate) { t.VotingEndTime = "8pm" },
		func(t *ContestAutoTemplate) { t.VotingEndTime = "24:00" },
		func(t *ContestAutoTemplate) { t.SubmissionEndDay = 0 },
		func(t *ContestAutoTemplate) { t.SubmissionEndDay = 29 },
		func(t *ContestAutoTemplate) { t.VotesPerUser = 0 },
	} {
		tp := DefaultContestAutoTemplate()
		bad(&tp)
		_, err := BuildAutoContest(tp, 2026, time.November)
		require.Error(t, err)
	}
}

func TestNormalizeContestAutoTemplate(t *testing.T) {
	tpl := DefaultContestAutoTemplate()
	tpl.TitleTemplate = "  {month} 月赛  "
	require.NoError(t, NormalizeContestAutoTemplate(&tpl))
	require.Equal(t, "{month} 月赛", tpl.TitleTemplate)

	tpl.TitleTemplate = " "
	require.Error(t, NormalizeContestAutoTemplate(&tpl))
	tpl = DefaultContestAutoTemplate()
	tpl.LeadDays = 11
	require.Error(t, NormalizeContestAutoTemplate(&tpl))
}

func TestContestAutoMonthsAndDue(t *testing.T) {
	// 2026-12-31 16:30 UTC is already January 1st in China.
	months := contestAutoMonths(time.Date(2026, 12, 31, 16, 30, 0, 0, time.UTC))
	require.Equal(t, [2][2]int{{2027, 1}, {2027, 2}}, months)

	nov, err := BuildAutoContest(DefaultContestAutoTemplate(), 2026, time.November)
	require.NoError(t, err)
	require.False(t, contestAutoDue(nov, 3, "", cst(2026, 10, 28, 23, 59)), "before the 3-day lead")
	require.True(t, contestAutoDue(nov, 3, "", cst(2026, 10, 29, 0, 0)))
	require.True(t, contestAutoDue(nov, 3, "2026-10", cst(2026, 11, 10, 0, 0)), "switched on mid-month: still created")
	require.False(t, contestAutoDue(nov, 3, "2026-11", cst(2026, 11, 2, 0, 0)), "a month is created once")
	require.True(t, contestAutoDue(nov, 3, "", cst(2026, 11, 22, 23, 59)))
	require.False(t, contestAutoDue(nov, 3, "", cst(2026, 11, 23, 0, 0)), "submissions already closed")
}

func TestContestCoverOrDefault(t *testing.T) {
	require.Equal(t, ContestMonthlyCover, ContestCoverOrDefault(&Contest{AutoPeriod: "2026-11"}))
	require.Equal(t, "/x.png", ContestCoverOrDefault(&Contest{AutoPeriod: "2026-11", CoverImage: "/x.png"}))
	require.Equal(t, "", ContestCoverOrDefault(&Contest{}))
}
