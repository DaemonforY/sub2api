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

func TestSiteDraftRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "sd-" + suffix + "@sd.test", Username: "sd" + suffix[len(suffix)-4:]})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM site_drafts WHERE user_id = $1`, user.ID)
	})
	repo := NewSiteDraftRepository(integrationDB)

	d := &service.SiteDraft{UserID: user.ID, KeyID: 3, Status: service.SiteDraftGenerating, Title: "咖啡馆",
		SiteDraftData: service.SiteDraftData{Brief: service.SiteDraftBrief{Description: "咖啡馆官网", Images: 1}, DraftHTML: "<html>半",
			Turns: []service.SiteDraftTurn{}, Images: []service.SiteDraftImage{}, Events: []service.ArticleEvent{}}}
	require.NoError(t, repo.Create(ctx, d))
	n, err := repo.CountActive(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// A draft with no page yet fails on restart; one with a page goes back to ready.
	_, err = repo.FailActive(ctx, "重启了")
	require.NoError(t, err)
	got, err := repo.Get(ctx, user.ID, d.ID)
	require.NoError(t, err)
	require.Equal(t, service.SiteDraftFailed, got.Status)
	require.Empty(t, got.DraftHTML)
	require.Equal(t, "重启了", got.Events[len(got.Events)-1].Text)

	got.HTML, got.Status, got.SiteID, got.SiteURL = "<html>ok</html>", service.SiteDraftDrawing, 9, "https://x.s.test"
	require.NoError(t, repo.Save(ctx, got))
	_, err = repo.FailActive(ctx, "重启了")
	require.NoError(t, err)
	got, _ = repo.Get(ctx, user.ID, d.ID)
	require.Equal(t, service.SiteDraftReady, got.Status)
	require.Equal(t, "<html>ok</html>", got.HTML)

	list, err := repo.List(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "咖啡馆官网", list[0].Brief.Description)
	require.Equal(t, int64(9), list[0].SiteID)
	require.Empty(t, list[0].HTML, "the list leaves out the page")

	other, err := repo.Get(ctx, user.ID+100000, d.ID)
	require.NoError(t, err)
	require.Nil(t, other)

	ids, err := repo.DeleteBefore(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Contains(t, ids, d.ID)
}
