//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestContestAuto_MonthlySchedule(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM contests WHERE auto_period LIKE '2031-%'`)
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM contest_auto_settings`)
	})
	_, err := integrationDB.ExecContext(ctx, `DELETE FROM contest_auto_settings`)
	require.NoError(t, err)

	repo := NewContestRepository(integrationDB)
	svc := service.NewContestService(repo, NewUserRepository(integrationEntClient, integrationDB),
		NewRedeemCodeRepository(integrationEntClient), nil, nil, service.NewContestImageStore(t.TempDir()))
	zone := time.FixedZone("CST", 8*3600)
	now := time.Date(2031, 3, 29, 9, 0, 0, 0, zone) // 3 days before April: April is due, March's submissions closed
	svc.SetClock(func() time.Time { return now })
	countPeriod := func(p string) int {
		var n int
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM contests WHERE auto_period=$1`, p).Scan(&n))
		return n
	}

	// Never saved: off, with the default template.
	st, err := svc.GetAutoSettings(ctx)
	require.NoError(t, err)
	require.False(t, st.Enabled)
	require.Equal(t, "{month} 月 AI 画图赛", st.Template.TitleTemplate)

	// Start: April is created and published at once.
	st, err = svc.SaveAutoSettings(ctx, true, service.DefaultContestAutoTemplate(), 0)
	require.NoError(t, err)
	require.True(t, st.Enabled)
	require.Len(t, st.Months, 2)
	require.Equal(t, "2031-03", st.Months[0].Period)
	require.Equal(t, "missed", st.Months[0].Status)
	require.Equal(t, "2031-04", st.Months[1].Period)
	require.Equal(t, "created", st.Months[1].Status)
	require.Equal(t, "4 月 AI 画图赛", st.Months[1].Title)
	require.Equal(t, service.ContestPhaseUpcoming, st.Months[1].Phase)
	april, err := svc.AdminGetContest(ctx, st.Months[1].ContestID)
	require.NoError(t, err)
	require.Equal(t, "2031-04", april.AutoPeriod)
	require.Equal(t, service.ContestStatusPublished, april.Status)
	require.True(t, april.VotingEndAt.Equal(time.Date(2031, 4, 30, 20, 0, 0, 0, zone)))

	// The loop runs every minute: never a second April.
	svc.RunAutoContests(ctx)
	svc.RunAutoContests(ctx)
	require.Equal(t, 1, countPeriod("2031-04"))

	// Two instances racing on the same month: one row, the loser reports not created.
	dup, err := service.BuildAutoContest(service.DefaultContestAutoTemplate(), 2031, time.April)
	require.NoError(t, err)
	created, err := repo.CreateAutoContest(ctx, dup)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, 1, countPeriod("2031-04"))

	// A cancelled and deleted month is not brought back.
	_, err = svc.CancelContest(ctx, april.ID)
	require.NoError(t, err)
	require.NoError(t, svc.DeleteContest(ctx, april.ID))
	svc.RunAutoContests(ctx)
	require.Equal(t, 0, countPeriod("2031-04"))
	st, err = svc.GetAutoSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "skipped", st.Months[1].Status)

	// Stop: May is not created; the page says it is paused.
	_, err = svc.SaveAutoSettings(ctx, false, service.DefaultContestAutoTemplate(), 0)
	require.NoError(t, err)
	now = time.Date(2031, 4, 29, 9, 0, 0, 0, zone)
	svc.RunAutoContests(ctx)
	require.Equal(t, 0, countPeriod("2031-05"))
	st, err = svc.GetAutoSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "paused", st.Months[1].Status)

	// Start again with an edited template: May is created from it.
	tpl := service.DefaultContestAutoTemplate()
	tpl.TitleTemplate = "{year} 年 {month} 月画图赛"
	tpl.VotesPerUser = 3
	st, err = svc.SaveAutoSettings(ctx, true, tpl, 0)
	require.NoError(t, err)
	require.Equal(t, "created", st.Months[1].Status)
	require.Equal(t, "2031 年 5 月画图赛", st.Months[1].Title)
	may, err := svc.AdminGetContest(ctx, st.Months[1].ContestID)
	require.NoError(t, err)
	require.Equal(t, 3, may.VotesPerUser)

	// A bad template is refused and the saved one stays.
	bad := tpl
	bad.VotingEndTime = "8pm"
	_, err = svc.SaveAutoSettings(ctx, true, bad, 0)
	require.Error(t, err)
	st, err = svc.GetAutoSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, "20:00", st.Template.VotingEndTime)
}
