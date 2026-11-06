package openai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsIncludeGPT6Astra(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-6-astra")
	require.Contains(t, DefaultModelIDs(), "gpt-6")
	var displayName string
	for _, model := range DefaultModels {
		if model.ID == "gpt-6-astra" {
			displayName = model.DisplayName
			break
		}
	}
	require.Equal(t, "GPT-6 Astra", displayName)
}

func TestDefaultModelsPreferConcreteGPT56SolForAccountTests(t *testing.T) {
	require.NotEmpty(t, DefaultModels)
	require.Equal(t, "gpt-5.6-sol", DefaultModels[0].ID)
}

func TestDefaultModelsIncludeGPTImage25(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-flare")
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.5-sunburst")
}

func TestDefaultModelsIncludeGPTImage26(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.6-flare")
	require.Contains(t, DefaultModelIDs(), "gpt-image-2.6-sunburst")
}

func TestDefaultModelsIncludeGPT61TerraAndGPT6Family(t *testing.T) {
	ids := DefaultModelIDs()
	for _, id := range []string{"gpt-6.1-terra", "gpt-6.1-sol", "gpt-6-sol", "gpt-6-luna"} {
		require.Contains(t, ids, id)
	}
	for _, model := range DefaultModels {
		if model.ID == "gpt-6.1-terra" {
			require.Equal(t, "GPT-6.1 Terra", model.DisplayName)
			return
		}
	}
	t.Fatal("gpt-6.1-terra missing from DefaultModels")
}
