//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGameProgressRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewGameProgressRepository(integrationDB)
	tag := fmt.Sprintf("gp%d", time.Now().UnixNano())
	var ids []int64
	for i, name := range []string{"阿浪", "小椰", ""} {
		var id int64
		require.NoError(t, integrationDB.QueryRowContext(ctx, `
			INSERT INTO users (email, username, password_hash, role, status, balance, concurrency)
			VALUES ($1, $2, 'hash', 'user', 'active', 0, 1) RETURNING id`,
			fmt.Sprintf("%s-%d@example.com", tag, i), name).Scan(&id))
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		}
	})
	game := "test-" + tag[len(tag)-8:]

	// Saves: none, then upsert twice.
	none, err := repo.GetSave(ctx, ids[0], game)
	require.NoError(t, err)
	require.Nil(t, none)
	_, err = repo.PutSave(ctx, ids[0], game, json.RawMessage(`{"level":2}`))
	require.NoError(t, err)
	_, err = repo.PutSave(ctx, ids[0], game, json.RawMessage(`{"level":5}`))
	require.NoError(t, err)
	got, err := repo.GetSave(ctx, ids[0], game)
	require.NoError(t, err)
	require.JSONEq(t, `{"level":5}`, string(got.Data))

	// Scores keep the best; a lower one is not an improvement.
	best, improved, err := repo.SubmitScore(ctx, ids[0], game, "main", 300, json.RawMessage(`{"floor":3}`))
	require.NoError(t, err)
	require.True(t, improved)
	require.EqualValues(t, 300, best)
	best, improved, err = repo.SubmitScore(ctx, ids[0], game, "main", 100, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.False(t, improved)
	require.EqualValues(t, 300, best)
	best, improved, err = repo.SubmitScore(ctx, ids[0], game, "main", 500, json.RawMessage(`{"floor":5}`))
	require.NoError(t, err)
	require.True(t, improved)
	require.EqualValues(t, 500, best)

	// Ties: whoever reached the score first ranks higher.
	_, _, err = repo.SubmitScore(ctx, ids[1], game, "main", 500, json.RawMessage(`{}`))
	require.NoError(t, err)
	_, _, err = repo.SubmitScore(ctx, ids[2], game, "main", 900, json.RawMessage(`{}`))
	require.NoError(t, err)

	top, err := repo.Leaderboard(ctx, game, "main", 10)
	require.NoError(t, err)
	require.Len(t, top, 3)
	require.Equal(t, ids[2], top[0].UserID)
	require.Equal(t, ids[0], top[1].UserID)
	require.Equal(t, 2, top[1].Rank)
	require.JSONEq(t, `{"floor":5}`, string(top[1].Detail))
	require.Equal(t, ids[1], top[2].UserID)

	n, err := repo.Players(ctx, game, "main")
	require.NoError(t, err)
	require.Equal(t, 3, n)

	me, err := repo.EntryFor(ctx, ids[1], game, "main")
	require.NoError(t, err)
	require.Equal(t, 3, me.Rank)
	missing, err := repo.EntryFor(ctx, ids[1], game, "other")
	require.NoError(t, err)
	require.Nil(t, missing)

	boards, err := repo.BoardsOf(ctx, ids[0], game)
	require.NoError(t, err)
	require.Equal(t, []string{"main"}, boards)
}
