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

func TestLearnRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	u1 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln-1-" + suffix + "@test.local", Username: "ln1" + suffix[len(suffix)-6:]})
	u2 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln-2-" + suffix + "@test.local", Username: "ln2" + suffix[len(suffix)-6:]})
	repo := NewLearnRepository(integrationDB)
	since := time.Now().Add(-time.Minute)

	require.NoError(t, repo.MarkDone(ctx, u1.ID, []string{"a1", "a2"}))
	require.NoError(t, repo.MarkDone(ctx, u1.ID, []string{"a2", "b1"})) // repeats are ignored
	done, err := repo.Progress(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, done, 3)

	before, err := repo.CountRuns(ctx, 0, "", since)
	require.NoError(t, err)
	ok, err := repo.StartRun(ctx, u1.ID, "a3", "run", 0)
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, ok, "ok", 1200, 30, 90))
	failed, err := repo.StartRun(ctx, u1.ID, "a3", "run", 0)
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, failed, "failed", 300, 0, 0))
	_, err = repo.StartRun(ctx, u2.ID, "a4", "run", 0) // still pending: counts
	require.NoError(t, err)

	n, err := repo.CountRuns(ctx, u1.ID, "run", since)
	require.NoError(t, err)
	require.Equal(t, 1, n) // the failed run does not count
	all, err := repo.CountRuns(ctx, 0, "", since)
	require.NoError(t, err)
	require.Equal(t, before+2, all)

	// Calls on the learner's own key and other kinds are not free runs.
	own, err := repo.StartRun(ctx, u1.ID, "a3", "run", 42)
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, own, "ok", 100, 1, 1))
	tutor, err := repo.StartRun(ctx, u1.ID, "a3", "tutor", 0)
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, tutor, "ok", 100, 1, 1))
	n, err = repo.CountRuns(ctx, u1.ID, "run", since)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = repo.CountRuns(ctx, u1.ID, "tutor", since)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	stats, err := repo.Stats(ctx, since)
	require.NoError(t, err)
	require.GreaterOrEqual(t, stats.Learners, 1)
	require.GreaterOrEqual(t, stats.RunsToday, 2)
	require.GreaterOrEqual(t, stats.FailedRuns7d, 1)
	require.GreaterOrEqual(t, stats.Tokens7d, 120)
	var a3 *service.LearnLessonStat
	for i := range stats.Lessons {
		if stats.Lessons[i].LessonID == "a3" {
			a3 = &stats.Lessons[i]
		}
	}
	require.NotNil(t, a3)
	require.GreaterOrEqual(t, a3.Runs, 1)
}

func TestLearnRepositoryL2(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	u1 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln2-1-" + suffix + "@test.local", Username: "lm1" + suffix[len(suffix)-6:]})
	repo := NewLearnRepository(integrationDB)

	// Quizzes keep the best score; a changed quiz (new total) takes the latest.
	q, err := repo.SaveQuizResult(ctx, u1.ID, "a3", 2, 4)
	require.NoError(t, err)
	require.Equal(t, 1, q.Attempts)
	q, err = repo.SaveQuizResult(ctx, u1.ID, "a3", 4, 4)
	require.NoError(t, err)
	q, err = repo.SaveQuizResult(ctx, u1.ID, "a3", 1, 4)
	require.NoError(t, err)
	require.Equal(t, 4, q.Correct)
	require.Equal(t, 3, q.Attempts)
	q, err = repo.SaveQuizResult(ctx, u1.ID, "a3", 1, 5)
	require.NoError(t, err)
	require.Equal(t, 1, q.Correct)
	require.Equal(t, 5, q.Total)
	results, err := repo.QuizResults(ctx, u1.ID)
	require.NoError(t, err)
	require.Equal(t, 5, results["a3"].Total)

	require.NoError(t, repo.PassCheckpoint(ctx, u1.ID, "key"))
	require.NoError(t, repo.PassCheckpoint(ctx, u1.ID, "key"))
	cps, err := repo.Checkpoints(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, cps, 1)

	// One live certificate per track; revoked ones can be claimed again but not restored over a new one.
	code1 := "T" + suffix[len(suffix)-9:]
	require.NoError(t, repo.CreateCertificate(ctx, &service.LearnCertificate{Code: code1, UserID: u1.ID, Track: "a", DisplayName: "小林", IssuedAt: time.Now()}))
	require.NoError(t, repo.CreateCertificate(ctx, &service.LearnCertificate{Code: "U" + suffix[len(suffix)-9:], UserID: u1.ID, Track: "a", DisplayName: "x", IssuedAt: time.Now()}))
	certs, err := repo.CertificatesByUser(ctx, u1.ID)
	require.NoError(t, err)
	require.Len(t, certs, 1)
	require.Equal(t, code1, certs[0].Code)
	c, err := repo.CertificateByCode(ctx, code1)
	require.NoError(t, err)
	require.Equal(t, "小林", c.DisplayName)
	require.NoError(t, repo.SetCertificateRevoked(ctx, code1, true))
	code2 := "V" + suffix[len(suffix)-9:]
	require.NoError(t, repo.CreateCertificate(ctx, &service.LearnCertificate{Code: code2, UserID: u1.ID, Track: "a", DisplayName: "小林", IssuedAt: time.Now()}))
	require.ErrorIs(t, repo.SetCertificateRevoked(ctx, code1, false), service.ErrLearnCertRestore)
	missing, err := repo.CertificateByCode(ctx, "NOPE")
	require.NoError(t, err)
	require.Nil(t, missing)
	list, total, err := repo.ListCertificates(ctx, 10, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 2)
	require.NotEmpty(t, list[0].UserEmail)

	// Interviews: answers saved only on top of the expected ones.
	iv := &service.LearnInterview{UserID: u1.ID, Topic: "rag", Status: "active", Questions: []service.LearnInterviewQ{{ID: "rag1", Q: "q1"}, {ID: "rag2", Q: "q2"}}}
	require.NoError(t, repo.CreateInterview(ctx, iv))
	n, err := repo.CountInterviews(ctx, u1.ID, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	iv.Answers = []service.LearnInterviewAnswer{{Answer: "a", Score: 6}}
	require.NoError(t, repo.SaveInterview(ctx, iv))
	require.ErrorIs(t, repo.SaveInterview(ctx, iv), service.ErrLearnInterviewDone) // stale
	now := time.Now()
	iv.Answers = append(iv.Answers, service.LearnInterviewAnswer{Answer: "b", Score: 8})
	iv.Status, iv.Score, iv.FinishedAt = "finished", 70, &now
	require.NoError(t, repo.SaveInterview(ctx, iv))
	got, err := repo.GetInterview(ctx, iv.ID)
	require.NoError(t, err)
	require.Len(t, got.Answers, 2)
	require.Equal(t, "finished", got.Status)
	best, err := repo.BestInterviewScores(ctx, u1.ID)
	require.NoError(t, err)
	require.Equal(t, 70, best["rag"])
	ivs, err := repo.ListInterviews(ctx, u1.ID, 5)
	require.NoError(t, err)
	require.Len(t, ivs, 1)
}

func TestLearnRepositoryL3(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	u1 := mustCreateUser(t, integrationEntClient, &service.User{Email: "ln3-1-" + suffix + "@test.local", Username: "lw1" + suffix[len(suffix)-6:]})
	repo := NewLearnRepository(integrationDB)
	since := time.Now().Add(-24 * time.Hour)

	code := "W" + suffix[len(suffix)-9:]
	require.NoError(t, repo.CreateCertificate(ctx, &service.LearnCertificate{Code: code, UserID: u1.ID, Track: "c", DisplayName: "小周", IssuedAt: time.Now(), Showcase: true}))
	wall, err := repo.Showcase(ctx, "c", 50)
	require.NoError(t, err)
	require.Contains(t, codesOf(wall), code)
	require.NoError(t, repo.SetShowcaseHidden(ctx, code, true))
	wall, err = repo.Showcase(ctx, "", 50)
	require.NoError(t, err)
	require.NotContains(t, codesOf(wall), code)
	require.NoError(t, repo.SetShowcaseHidden(ctx, code, false))
	require.NoError(t, repo.SetShowcase(ctx, u1.ID, "c", false))
	wall, err = repo.Showcase(ctx, "c", 50)
	require.NoError(t, err)
	require.NotContains(t, codesOf(wall), code)
	require.ErrorIs(t, repo.SetShowcase(ctx, u1.ID, "a", true), service.ErrLearnCertNotFound)
	require.ErrorIs(t, repo.SetShowcaseHidden(ctx, "NOPE", true), service.ErrLearnCertNotFound)

	require.NoError(t, repo.MarkDone(ctx, u1.ID, []string{"c1", "c2"}))
	run, err := repo.StartRun(ctx, u1.ID, "c2", "tutor", 0)
	require.NoError(t, err)
	require.NoError(t, repo.FinishRun(ctx, run, "ok", 10, 1, 1))
	_, err = repo.SaveQuizResult(ctx, u1.ID, "c1", 4, 4)
	require.NoError(t, err)

	progress, err := repo.TrackProgress(ctx)
	require.NoError(t, err)
	found := false
	for _, p := range progress {
		if p.UserID == u1.ID && p.Track == "c" {
			found = true
			require.Equal(t, 2, p.Lessons)
		}
	}
	require.True(t, found)
	counts, err := repo.CertificateCounts(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, counts["c"], 1)
	days, err := repo.Daily(ctx, since)
	require.NoError(t, err)
	require.NotEmpty(t, days)
	last := days[len(days)-1]
	require.GreaterOrEqual(t, last.Tutor+last.Completions+last.Certificates, 1)
	require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, last.Date)
	quiz, err := repo.QuizStats(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, quiz["c1"].Passed, 1)
}

func codesOf(certs []service.LearnCertificate) []string {
	out := make([]string, 0, len(certs))
	for _, c := range certs {
		out = append(out, c.Code)
	}
	return out
}
