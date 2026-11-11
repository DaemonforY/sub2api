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

func TestGameCraftRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewGameCraftRepository(integrationDB)
	tag := fmt.Sprintf("测%d", time.Now().UnixNano()%100000)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM game_craft_recipes WHERE result LIKE $1`, tag+"%")
		_, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM game_craft_elements WHERE name LIKE $1`, tag+"%")
	})

	// Migration 326 seeds the five starters.
	got, err := repo.Elements(ctx, service.CraftStarterNames())
	require.NoError(t, err)
	require.Len(t, got, 5)
	require.Equal(t, "🔥", got["火"].Emoji)

	none, err := repo.Recipe(ctx, "水", "火")
	require.NoError(t, err)
	require.Nil(t, none)

	steam, created, err := repo.Save(ctx, "水", "火", service.CraftElement{Name: tag + "汽", Emoji: "💨", Description: "d"}, 0)
	require.NoError(t, err)
	require.True(t, created)
	require.Positive(t, steam.ID)

	// A second answer for the same pair loses to the stored one.
	again, created, err := repo.Save(ctx, "水", "火", service.CraftElement{Name: tag + "别的"}, 0)
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, tag+"汽", again.Name)

	// A different pair reaching an existing element is not a discovery.
	_, created, err = repo.Save(ctx, "水", "水", service.CraftElement{Name: tag + "汽"}, 0)
	require.NoError(t, err)
	require.False(t, created)

	r, err := repo.Recipe(ctx, "水", "火")
	require.NoError(t, err)
	require.Equal(t, tag+"汽", r.Name)

	total, recent, err := repo.Stats(ctx, 5)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(6))
	require.Equal(t, tag+"汽", recent[0].Name)
	for _, el := range recent {
		require.NotContains(t, service.CraftStarterNames(), el.Name)
	}
}
