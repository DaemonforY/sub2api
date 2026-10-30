package service

import (
	"fmt"
	mathrand "math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func weightedTestAccount(id int64, priority, weight int) *Account {
	acc := &Account{ID: id, Priority: priority}
	if weight != 0 {
		acc.Extra = map[string]any{AccountSchedulingWeightExtraKey: float64(weight)}
	}
	return acc
}

func weightedTestIDs(accounts []*Account) []int64 {
	ids := make([]int64, len(accounts))
	for i, acc := range accounts {
		ids[i] = acc.ID
	}
	return ids
}

func TestAccountSchedulingWeight_ParsesAndClamps(t *testing.T) {
	require.Equal(t, 0, (*Account)(nil).SchedulingWeight())
	require.Equal(t, 0, (&Account{}).SchedulingWeight())
	require.Equal(t, 20, weightedTestAccount(1, 1, 20).SchedulingWeight())
	require.Equal(t, 0, (&Account{Extra: map[string]any{AccountSchedulingWeightExtraKey: float64(-5)}}).SchedulingWeight())
	require.Equal(t, maxAccountSchedulingWeight, (&Account{Extra: map[string]any{AccountSchedulingWeightExtraKey: float64(99999)}}).SchedulingWeight())
	require.Equal(t, 7, (&Account{Extra: map[string]any{AccountSchedulingWeightExtraKey: "7"}}).SchedulingWeight())
}

func TestApplySchedulingWeights_NoWeightsKeepsOrder(t *testing.T) {
	accounts := []*Account{
		weightedTestAccount(1, 1, 0),
		weightedTestAccount(2, 1, 0),
		weightedTestAccount(3, 2, 0),
	}
	applySchedulingWeightsWithinPriority(accounts, func(a *Account) *Account { return a }, nil)
	require.Equal(t, []int64{1, 2, 3}, weightedTestIDs(accounts))
}

func TestApplySchedulingWeights_StaysWithinPriorityAndPutsUnweightedLast(t *testing.T) {
	accounts := []*Account{
		weightedTestAccount(1, 1, 0),
		weightedTestAccount(2, 1, 5),
		weightedTestAccount(3, 1, 0),
		weightedTestAccount(4, 1, 5),
		weightedTestAccount(5, 2, 100),
	}
	applySchedulingWeightsWithinPriority(accounts, func(a *Account) *Account { return a }, nil)
	ids := weightedTestIDs(accounts)
	require.ElementsMatch(t, []int64{2, 4}, ids[:2])
	require.Equal(t, []int64{1, 3, 5}, ids[2:], "unweighted keep their order; a heavier lower-priority account never jumps ahead")
}

func TestApplySchedulingWeights_FollowsWeightRatio(t *testing.T) {
	rng := mathrand.New(mathrand.NewPCG(1, 2))
	firsts := map[int64]int{}
	const rounds = 20000
	for i := 0; i < rounds; i++ {
		accounts := []*Account{
			weightedTestAccount(1, 1, 20),
			weightedTestAccount(2, 1, 60),
			weightedTestAccount(3, 1, 20),
		}
		applySchedulingWeightsWithinPriority(accounts, func(a *Account) *Account { return a }, rng.Float64)
		firsts[accounts[0].ID]++
	}
	share := func(id int64) float64 { return float64(firsts[id]) / rounds }
	require.InDelta(t, 0.2, share(1), 0.02, fmt.Sprint(firsts))
	require.InDelta(t, 0.6, share(2), 0.02, fmt.Sprint(firsts))
	require.InDelta(t, 0.2, share(3), 0.02, fmt.Sprint(firsts))
}

func TestApplySchedulingWeights_WorksOnAccountWithLoad(t *testing.T) {
	items := []accountWithLoad{
		{account: weightedTestAccount(1, 1, 0), loadInfo: &AccountLoadInfo{}},
		{account: weightedTestAccount(2, 1, 3), loadInfo: &AccountLoadInfo{}},
	}
	applySchedulingWeightsWithinPriority(items, func(item accountWithLoad) *Account { return item.account }, nil)
	require.Equal(t, int64(2), items[0].account.ID)
	require.Equal(t, int64(1), items[1].account.ID)
}

func TestBuildOpenAIWeightedSelectionOrder_SchedulingWeightTiers(t *testing.T) {
	candidates := []openAIAccountCandidateScore{
		{account: weightedTestAccount(1, 1, 0), score: 9},
		{account: weightedTestAccount(2, 2, 50), score: 9},
		{account: weightedTestAccount(3, 1, 10), score: 1},
		{account: weightedTestAccount(4, 1, 0), score: 5},
	}
	order := buildOpenAIWeightedSelectionOrder(candidates, OpenAIAccountScheduleRequest{SessionHash: "s"})
	ids := make([]int64, len(order))
	for i, c := range order {
		ids[i] = c.account.ID
	}
	// priority 1: weighted (3) first, then unweighted by score (1 before 4); then priority 2.
	require.Equal(t, []int64{3, 1, 4, 2}, ids)
}

func TestBuildOpenAIWeightedSelectionOrder_SchedulingWeightRatio(t *testing.T) {
	firsts := map[int64]int{}
	const rounds = 20000
	for i := 0; i < rounds; i++ {
		candidates := []openAIAccountCandidateScore{
			{account: weightedTestAccount(1, 1, 30), score: 2},
			{account: weightedTestAccount(2, 1, 10), score: 2},
		}
		order := buildOpenAIWeightedSelectionOrder(candidates, OpenAIAccountScheduleRequest{SessionHash: fmt.Sprintf("s-%d", i)})
		firsts[order[0].account.ID]++
	}
	require.InDelta(t, 0.75, float64(firsts[1])/rounds, 0.03, fmt.Sprint(firsts))
}
