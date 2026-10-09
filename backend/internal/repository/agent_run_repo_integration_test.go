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

func TestAgentRunRepository(t *testing.T) {
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	agent := "t" + suffix[len(suffix)-12:]
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(context.Background(), `DELETE FROM agent_runs WHERE agent = $1`, agent) })
	user := mustCreateUser(t, integrationEntClient, &service.User{Email: "agent-" + suffix + "@agent.test", Username: "agent" + suffix[len(suffix)-4:]})
	repo := NewAgentRunRepository(integrationDB)

	uid := user.ID
	first, err := repo.Create(ctx, &service.AgentRunRecord{Agent: agent, UserID: &uid, Question: "为什么报错", Answer: "因为…",
		Steps: []service.AgentStep{{Tool: "get_recent_errors", Args: "{}", Result: `{"errors":[]}`, Ms: 12}},
		Model: "gpt-5.6-terra", ModelCalls: 2, PromptTokens: 300, CompletionTokens: 40, Status: "ok", DurationMs: 1500})
	require.NoError(t, err)
	_, err = repo.Create(ctx, &service.AgentRunRecord{Agent: agent, Question: "访客的问题", Steps: []service.AgentStep{}, Status: "error", Error: "上游超时"})
	require.NoError(t, err)

	list, total, err := repo.List(ctx, agent, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, list, 2)
	require.Equal(t, "访客的问题", list[0].Question, "newest first")
	require.Nil(t, list[0].UserID)
	require.Equal(t, "上游超时", list[0].Error)
	got := list[1]
	require.Equal(t, first, got.ID)
	require.Equal(t, user.Email, got.UserEmail)
	require.Equal(t, []service.AgentStep{{Tool: "get_recent_errors", Args: "{}", Result: `{"errors":[]}`, Ms: 12}}, got.Steps)
	require.Equal(t, 300, got.PromptTokens)
	require.EqualValues(t, 1500, got.DurationMs)

	page2, _, err := repo.List(ctx, agent, 2, 1)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	require.Equal(t, first, page2[0].ID)

	_, err = integrationDB.ExecContext(ctx, `UPDATE agent_runs SET created_at = NOW() - INTERVAL '40 days' WHERE id = $1`, first)
	require.NoError(t, err)
	n, err := repo.DeleteBefore(ctx, time.Now().Add(-30*24*time.Hour))
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, int64(1))
	_, total, err = repo.List(ctx, agent, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)
}
