package claude

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsContainsClaudeFable51(t *testing.T) {
	t.Parallel()

	for _, model := range DefaultModels {
		if model.ID == "claude-fable-5-1" {
			require.Equal(t, "Claude Fable 5.1", model.DisplayName)
			require.Equal(t, "2026-09-01T00:00:00Z", model.CreatedAt)
			return
		}
	}
	t.Fatal("claude-fable-5-1 missing from DefaultModels")
}

func TestDefaultModelsListsClaudeFable52First(t *testing.T) {
	t.Parallel()

	require.Equal(t, "claude-fable-5-2", DefaultModels[0].ID)
	require.Equal(t, "Claude Fable 5.2", DefaultModels[0].DisplayName)
	require.Equal(t, "2026-11-09T00:00:00Z", DefaultModels[0].CreatedAt)
}

func TestDefaultModelsListsClaude55Family(t *testing.T) {
	t.Parallel()

	ids := DefaultModelIDs()
	require.Contains(t, ids, "claude-opus-5-5")
	require.Contains(t, ids, "claude-sonnet-5-5")
}
