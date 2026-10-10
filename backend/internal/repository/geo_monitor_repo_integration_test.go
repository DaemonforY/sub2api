//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGeoMonitorRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewGeoMonitorRepository(integrationDB)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// The migration seeds the default questions.
	seeded, err := repo.ListQuestions(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(seeded), 20)

	q := &service.GeoQuestion{Question: "测试问题 " + suffix, Category: "测试", Enabled: true, Sort: 9999}
	require.NoError(t, repo.CreateQuestion(ctx, q))
	e := &service.GeoEngine{Name: "geo-test-" + suffix, BaseURL: "https://api.example.com/v1", APIKeyEncrypted: "enc", Model: "m",
		ExtraBody: json.RawMessage(`{"enable_search": true}`), Enabled: true}
	require.NoError(t, repo.CreateEngine(ctx, e))
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM geo_checks WHERE question_id = $1 OR engine_name LIKE $2`, q.ID, "%"+suffix)
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM geo_questions WHERE id = $1`, q.ID)
		_, _ = integrationDB.ExecContext(bg, `DELETE FROM geo_engines WHERE id = $1`, e.ID)
	})
	require.ErrorIs(t, repo.CreateEngine(ctx, &service.GeoEngine{Name: e.Name, BaseURL: "https://x", APIKeyEncrypted: "e", Model: "m"}), service.ErrGeoEngineNameTaken)
	got, err := repo.GetEngine(ctx, e.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"enable_search": true}`, string(got.ExtraBody))
	got.ExtraBody = nil
	require.NoError(t, repo.UpdateEngine(ctx, got))
	got, _ = repo.GetEngine(ctx, e.ID)
	require.Nil(t, got.ExtraBody)

	qid, eid := q.ID, e.ID
	old := &service.GeoCheck{QuestionID: &qid, Question: q.Question, EngineID: &eid, EngineName: e.Name, Source: service.GeoSourceAuto, Answer: "旧", RunID: "r1"}
	require.NoError(t, repo.InsertCheck(ctx, old))
	newer := &service.GeoCheck{QuestionID: &qid, Question: q.Question, EngineID: &eid, EngineName: e.Name, Source: service.GeoSourceAuto,
		Answer: "hivegpt", Mentioned: true, CitedURLs: []string{"https://hivegpt.cn/"}, OurURLs: []string{"https://hivegpt.cn/"}, RunID: "r2"}
	require.NoError(t, repo.InsertCheck(ctx, newer))
	failed := &service.GeoCheck{QuestionID: &qid, Question: q.Question, EngineID: &eid, EngineName: e.Name, Source: service.GeoSourceAuto, Error: "HTTP 500"}
	require.NoError(t, repo.InsertCheck(ctx, failed))
	manual := &service.GeoCheck{QuestionID: &qid, Question: q.Question, EngineName: "豆包" + suffix, Source: service.GeoSourceManual, Answer: "不知道"}
	require.NoError(t, repo.InsertCheck(ctx, manual))

	latest, err := repo.LatestChecks(ctx)
	require.NoError(t, err)
	var mine []service.GeoCheck
	for _, c := range latest {
		if c.QuestionID != nil && *c.QuestionID == q.ID {
			mine = append(mine, c)
		}
	}
	require.Len(t, mine, 2, "one per engine, errors skipped")
	for _, c := range mine {
		if c.EngineName == e.Name {
			require.Equal(t, newer.ID, c.ID)
			require.Equal(t, []string{"https://hivegpt.cn/"}, c.OurURLs)
			require.Equal(t, "r2", c.RunID)
		} else {
			require.Nil(t, c.EngineID)
			require.Equal(t, "", c.RunID)
		}
	}

	yes := true
	page, total, err := repo.ListChecks(ctx, service.GeoCheckFilter{QuestionID: q.ID, Limit: 2})
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Len(t, page, 2)
	require.Equal(t, manual.ID, page[0].ID)
	_, total, err = repo.ListChecks(ctx, service.GeoCheckFilter{QuestionID: q.ID, Engine: e.Name, Mentioned: &yes, Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	_, total, err = repo.ListChecks(ctx, service.GeoCheckFilter{QuestionID: q.ID, Source: service.GeoSourceManual, Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)

	weekly, err := repo.WeeklyRates(ctx, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	var found bool
	for _, w := range weekly {
		if w.EngineName == e.Name {
			found = true
			require.Equal(t, 1, w.Total)
			require.Equal(t, 1, w.Mentioned)
			require.Equal(t, time.Monday, w.WeekStart.Weekday())
		}
	}
	require.True(t, found)

	require.NoError(t, repo.DeleteCheck(ctx, manual.ID))
	require.ErrorIs(t, repo.DeleteCheck(ctx, manual.ID), service.ErrGeoCheckNotFound)
	// Deleting the engine keeps its answers.
	require.NoError(t, repo.DeleteEngine(ctx, e.ID))
	_, total, err = repo.ListChecks(ctx, service.GeoCheckFilter{QuestionID: q.ID, Limit: 10})
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	_, err = repo.GetEngine(ctx, e.ID)
	require.ErrorIs(t, err, service.ErrGeoEngineNotFound)
}
