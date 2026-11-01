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

func TestTietuProjectRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "tietu-" + suffix + "@tietu.test", Username: "tt" + suffix[len(suffix)-4:]})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM tietu_projects WHERE user_id = $1`, user.ID)
	})
	repo := NewTietuProjectRepository(integrationDB)

	p := &service.TietuProject{UserID: user.ID, KeyID: 42, Status: service.TietuPlanning, Title: "桃花源记",
		TietuData: service.TietuData{Brief: service.TietuBrief{Input: "桃花源记", Mode: "auto", Panels: 6},
			Suggestions: []service.TietuSuggestion{}, Covers: []service.TietuImage{}, Panels: []service.TietuImage{},
			Events: []service.ArticleEvent{{At: time.Now(), Kind: "info", Text: "开始"}}}}
	require.NoError(t, repo.Create(ctx, p))
	require.NotZero(t, p.ID)
	n, err := repo.CountActive(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	p.Status = service.TietuDone
	p.Poem = &service.TietuPoem{Title: "桃花源记", Author: "陶渊明", Text: "晋太元中", Sources: []service.TietuSource{}, Variants: []service.TietuVariant{}}
	p.Panels = []service.TietuImage{{Name: "p1", Kind: "panel", N: 1, Status: "ok", Check: &service.TietuCheck{OK: true, Issues: []string{}, Minor: []string{}}}}
	p.Post = &service.TietuPost{Title: "桃花源记｜连环画", Body: "正文"}
	pushed := time.Now().UTC().Truncate(time.Millisecond)
	p.PushedAt = &pushed
	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.Get(ctx, user.ID, p.ID)
	require.NoError(t, err)
	require.Equal(t, "陶渊明", got.Poem.Author)
	require.True(t, got.Panels[0].Check.OK)
	require.Equal(t, "正文", got.Post.Body)
	other, err := repo.Get(ctx, user.ID+100000, p.ID)
	require.NoError(t, err)
	require.Nil(t, other, "another user's project is invisible")

	list, err := repo.List(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 6, list[0].Brief.Panels)
	require.Nil(t, list[0].Post, "lists leave out the heavy parts")
	require.True(t, list[0].PushedAt.Equal(pushed))

	running := &service.TietuProject{UserID: user.ID, KeyID: 42, Status: service.TietuChecking, TietuData: service.TietuData{Events: []service.ArticleEvent{}}}
	require.NoError(t, repo.Create(ctx, running))
	_, err = repo.FailActive(ctx, "服务器重启")
	require.NoError(t, err)
	got, err = repo.Get(ctx, user.ID, running.ID)
	require.NoError(t, err)
	require.Equal(t, service.TietuFailed, got.Status)
	require.Equal(t, "服务器重启", got.Events[len(got.Events)-1].Text)

	_, err = integrationDB.ExecContext(ctx, `UPDATE tietu_projects SET created_at = NOW() - INTERVAL '40 days' WHERE id = $1`, p.ID)
	require.NoError(t, err)
	ids, err := repo.DeleteBefore(ctx, time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.Contains(t, ids, p.ID)
	require.NotContains(t, ids, running.ID)
	require.NoError(t, repo.Delete(ctx, user.ID, running.ID))
}
