//go:build integration

package repository

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func contestTestPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	return buf.Bytes()
}

func TestContestFlow_SubmitVoteSettleGrant(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().Format("150405.000000")
	mk := func(name string) *service.User {
		return mustCreateUser(t, integrationEntClient, &service.User{Email: name + "-" + suffix + "@contest.test"})
	}
	alice, bob, carol, dave := mk("alice"), mk("bob"), mk("carol"), mk("dave")
	// Prize grants write committed redeem codes; remove them so suites that count every
	// redeem code in the shared database (TestRedeemCodeRepoSuite) are unaffected.
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(),
			`DELETE FROM redeem_codes WHERE used_by = ANY($1)`, pq.Array([]int64{alice.ID, bob.ID, carol.ID, dave.ID}))
	})

	repo := NewContestRepository(integrationDB)
	svc := service.NewContestService(repo, NewUserRepository(integrationEntClient, integrationDB),
		NewRedeemCodeRepository(integrationEntClient), nil, nil, service.NewContestImageStore(t.TempDir()))

	now := time.Now().UTC()
	contest, err := svc.CreateContest(ctx, &service.Contest{
		Title: "国庆 AI 绘画赛", Status: service.ContestStatusPublished,
		SubmissionStartAt: now.Add(-2 * time.Hour), SubmissionEndAt: now.Add(time.Hour),
		VotingStartAt: now.Add(-time.Hour), VotingEndAt: now.Add(time.Hour),
		MaxEntriesPerUser: 1, VotesPerUser: 2, OnePrizePerUser: true, MinVotesForPrize: 1,
		Prizes: []service.ContestPrize{
			{RankFrom: 1, RankTo: 1, Type: service.ContestPrizeBalance, Amount: 20, Label: "一等奖"},
			{RankFrom: 2, RankTo: 2, Type: service.ContestPrizeCustom, Label: "定制周边"},
		},
	}, 0)
	require.NoError(t, err)
	require.Equal(t, service.ContestPhaseSubmitAndVote, contest.Phase)

	img := contestTestPNG(t)
	aEntry, err := svc.SubmitEntry(ctx, contest.ID, alice.ID, service.ContestEntryInput{Title: "月球上的猫", Image: img})
	require.NoError(t, err)
	require.Equal(t, service.ContestEntryApproved, aEntry.Status)
	_, err = svc.SubmitEntry(ctx, contest.ID, alice.ID, service.ContestEntryInput{Title: "第二幅", Image: img})
	require.ErrorIs(t, err, service.ErrContestEntryLimit)
	bEntry, err := svc.SubmitEntry(ctx, contest.ID, bob.ID, service.ContestEntryInput{Title: "赛博长城", Image: img})
	require.NoError(t, err)

	// Voting rules.
	require.ErrorIs(t, svc.Vote(ctx, contest.ID, aEntry.ID, alice.ID, "1.1.1.1"), service.ErrContestSelfVote)
	require.NoError(t, svc.Vote(ctx, contest.ID, aEntry.ID, carol.ID, "2.2.2.2"))
	require.ErrorIs(t, svc.Vote(ctx, contest.ID, aEntry.ID, carol.ID, "2.2.2.2"), service.ErrContestAlreadyVoted)
	require.NoError(t, svc.Vote(ctx, contest.ID, bEntry.ID, carol.ID, "2.2.2.2"))
	require.NoError(t, svc.Vote(ctx, contest.ID, aEntry.ID, dave.ID, "3.3.3.3"))
	require.ErrorIs(t, svc.Vote(ctx, contest.ID, bEntry.ID, bob.ID, "4.4.4.4"), service.ErrContestSelfVote)

	detail, err := svc.GetPublicContest(ctx, contest.ID, carol.ID)
	require.NoError(t, err)
	require.Equal(t, 2, detail.Viewer.VotesUsed)
	require.Equal(t, 0, detail.Viewer.VotesLeft)
	require.False(t, detail.Viewer.CanVote)

	// Unvote frees a vote; the limit is enforced per contest.
	require.NoError(t, svc.Unvote(ctx, contest.ID, bEntry.ID, carol.ID))
	require.ErrorIs(t, svc.Unvote(ctx, contest.ID, bEntry.ID, carol.ID), service.ErrContestNotVoted)
	require.NoError(t, svc.Vote(ctx, contest.ID, bEntry.ID, carol.ID, "2.2.2.2"))

	board, err := svc.Leaderboard(ctx, contest.ID, 10)
	require.NoError(t, err)
	require.Len(t, board, 2)
	require.Equal(t, aEntry.ID, board[0].EntryID)
	require.Equal(t, 2, board[0].Votes)
	require.False(t, board[0].Final)

	// Cannot settle before the deadline.
	_, err = svc.SettleContest(ctx, contest.ID)
	require.ErrorIs(t, err, service.ErrContestNotSettleable)

	// Move the deadline into the past, then plant a vote stamped after it: it must not count.
	// Genuine votes happened "10 minutes ago"; the deadline was 1 minute ago.
	_, err = integrationDB.ExecContext(ctx, `UPDATE contest_votes SET created_at = created_at - interval '10 minutes' WHERE contest_id=$1`, contest.ID)
	require.NoError(t, err)
	cutoff := time.Now().UTC().Add(-time.Minute)
	_, err = integrationDB.ExecContext(ctx, `UPDATE contests SET voting_end_at=$2, submission_end_at=$2 WHERE id=$1`, contest.ID, cutoff)
	require.NoError(t, err)
	for _, voter := range []int64{alice.ID, dave.ID} {
		_, err = integrationDB.ExecContext(ctx, `INSERT INTO contest_votes (contest_id, entry_id, user_id, created_at) VALUES ($1,$2,$3,$4)`,
			contest.ID, bEntry.ID, voter, cutoff.Add(30*time.Second))
		require.NoError(t, err)
	}
	require.ErrorIs(t, svc.Vote(ctx, contest.ID, bEntry.ID, mk("late").ID, "5.5.5.5"), service.ErrContestNotVoting)

	settled, err := svc.SettleContest(ctx, contest.ID)
	require.NoError(t, err)
	require.Equal(t, service.ContestStatusSettled, settled.Status)
	again, err := svc.SettleContest(ctx, contest.ID)
	require.NoError(t, err, "settling twice is a no-op")
	require.Equal(t, service.ContestStatusSettled, again.Status)

	final, err := svc.Leaderboard(ctx, contest.ID, 10)
	require.NoError(t, err)
	require.True(t, final[0].Final)
	require.Equal(t, aEntry.ID, final[0].EntryID, "late votes for bob are ignored")
	require.Equal(t, 2, final[0].Votes)
	require.Equal(t, 1, final[1].Votes)

	awards, err := svc.ListAwards(ctx, contest.ID)
	require.NoError(t, err)
	require.Len(t, awards, 2)
	require.Equal(t, alice.ID, awards[0].UserID)
	require.Equal(t, service.ContestPrizeBalance, awards[0].PrizeType)
	require.Equal(t, bob.ID, awards[1].UserID)

	before := userBalance(t, alice.ID)
	granted, failed, err := svc.GrantAllBalanceAwards(ctx, contest.ID)
	require.NoError(t, err)
	require.Equal(t, 1, granted)
	require.Zero(t, failed)
	require.InDelta(t, before+20, userBalance(t, alice.ID), 1e-9)

	_, err = svc.GrantAward(ctx, contest.ID, awards[0].ID, "")
	require.ErrorIs(t, err, service.ErrContestAwardNotGrant, "a granted prize cannot be paid twice")
	require.InDelta(t, before+20, userBalance(t, alice.ID), 1e-9)

	custom, err := svc.GrantAward(ctx, contest.ID, awards[1].ID, "顺丰 SF123")
	require.NoError(t, err)
	require.Equal(t, service.ContestAwardGranted, custom.Status)

	_, err = svc.UpdateContest(ctx, contest.ID, &service.Contest{Title: "x"})
	require.ErrorIs(t, err, service.ErrContestNotEditable)
}

func TestContestReview_RejectRefundsVotes(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().Format("150405.000000")
	author := mustCreateUser(t, integrationEntClient, &service.User{Email: "author-" + suffix + "@contest.test"})
	voter := mustCreateUser(t, integrationEntClient, &service.User{Email: "voter-" + suffix + "@contest.test"})
	svc := service.NewContestService(NewContestRepository(integrationDB), NewUserRepository(integrationEntClient, integrationDB),
		NewRedeemCodeRepository(integrationEntClient), nil, nil, service.NewContestImageStore(t.TempDir()))

	now := time.Now().UTC()
	c, err := svc.CreateContest(ctx, &service.Contest{
		Title: "审核赛", Status: service.ContestStatusPublished, RequireReview: true,
		SubmissionStartAt: now.Add(-time.Hour), SubmissionEndAt: now.Add(time.Hour),
		VotingStartAt: now.Add(-time.Hour), VotingEndAt: now.Add(2 * time.Hour),
		MaxEntriesPerUser: 1, VotesPerUser: 1,
	}, 0)
	require.NoError(t, err)

	e, err := svc.SubmitEntry(ctx, c.ID, author.ID, service.ContestEntryInput{Title: "待审", Image: contestTestPNG(t)})
	require.NoError(t, err)
	require.Equal(t, service.ContestEntryPending, e.Status)
	require.ErrorIs(t, svc.Vote(ctx, c.ID, e.ID, voter.ID, ""), service.ErrContestEntryNotOpen)

	require.NoError(t, svc.ReviewEntry(ctx, c.ID, e.ID, service.ContestEntryApproved, ""))
	require.NoError(t, svc.Vote(ctx, c.ID, e.ID, voter.ID, ""))

	votes, err := svc.AdminListEntryVotes(ctx, c.ID, e.ID)
	require.NoError(t, err)
	require.Len(t, votes, 1)
	require.NotEmpty(t, votes[0].UserEmail)

	require.NoError(t, svc.ReviewEntry(ctx, c.ID, e.ID, service.ContestEntryDisqualified, "刷票"))
	detail, err := svc.GetPublicContest(ctx, c.ID, voter.ID)
	require.NoError(t, err)
	require.Equal(t, 1, detail.Viewer.VotesLeft, "disqualifying refunds the voter")
}

func userBalance(t *testing.T, userID int64) float64 {
	t.Helper()
	var b float64
	require.NoError(t, integrationDB.QueryRow(`SELECT balance FROM users WHERE id=$1`, userID).Scan(&b))
	return b
}
