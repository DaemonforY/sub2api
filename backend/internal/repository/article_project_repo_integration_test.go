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

func TestArticleProjectRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "article-" + suffix + "@article.test", Username: "art" + suffix[len(suffix)-4:]})
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM article_projects WHERE user_id = $1`, user.ID)
	})
	repo := NewArticleProjectRepository(integrationDB)

	p := &service.ArticleProject{UserID: user.ID, KeyID: 42, Status: service.ArticleOutlining, Title: "国庆出游",
		ArticleData: service.ArticleData{Brief: service.ArticleBrief{Topic: "国庆出游", Length: "standard", Images: 2, Search: true},
			Sources: []service.ArticleSource{}, Images: []service.ArticleImage{}, Events: []service.ArticleEvent{{At: time.Now(), Kind: "info", Text: "开始"}}}}
	require.NoError(t, repo.Create(ctx, p))
	require.NotZero(t, p.ID)

	n, err := repo.CountActive(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	p.Status = service.ArticleDone
	p.Title = "国庆出游，3 个省钱办法"
	p.Markdown = "正文"
	p.Outline = &service.ArticleOutline{Title: p.Title, Sections: []service.ArticleSection{{Heading: "一", Points: []string{"a"}}}}
	pushed := time.Now().UTC().Truncate(time.Millisecond)
	p.PushedAt = &pushed
	require.NoError(t, repo.Save(ctx, p))

	got, err := repo.Get(ctx, user.ID, p.ID)
	require.NoError(t, err)
	require.Equal(t, "国庆出游，3 个省钱办法", got.Title)
	require.Equal(t, int64(42), got.KeyID)
	require.Equal(t, "正文", got.Markdown)
	require.Equal(t, "一", got.Outline.Sections[0].Heading)
	require.True(t, got.Brief.Search)
	other, err := repo.Get(ctx, user.ID+100000, p.ID)
	require.NoError(t, err)
	require.Nil(t, other, "another user's article is invisible")

	list, err := repo.List(ctx, user.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 2, list[0].Brief.Images)
	require.Empty(t, list[0].Markdown, "lists leave out the article")
	require.NotNil(t, list[0].PushedAt)
	require.True(t, list[0].PushedAt.Equal(pushed))

	running := &service.ArticleProject{UserID: user.ID, KeyID: 42, Status: service.ArticleWriting, ArticleData: service.ArticleData{Events: []service.ArticleEvent{}}}
	require.NoError(t, repo.Create(ctx, running))
	_, err = repo.FailActive(ctx, "服务器重启")
	require.NoError(t, err)
	got, err = repo.Get(ctx, user.ID, running.ID)
	require.NoError(t, err)
	require.Equal(t, service.ArticleFailed, got.Status)
	require.Equal(t, "服务器重启", got.Error)
	require.Equal(t, "服务器重启", got.Events[len(got.Events)-1].Text)

	_, err = integrationDB.ExecContext(ctx, `UPDATE article_projects SET created_at = NOW() - INTERVAL '40 days' WHERE id = $1`, p.ID)
	require.NoError(t, err)
	ids, err := repo.DeleteBefore(ctx, time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.Contains(t, ids, p.ID)
	require.NotContains(t, ids, running.ID)

	require.NoError(t, repo.Delete(ctx, user.ID, running.ID))
	got, err = repo.Get(ctx, user.ID, running.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}
