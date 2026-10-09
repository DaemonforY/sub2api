package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHostedWebSearchFailureDoesNotCoolDownModel(t *testing.T) {
	svc := &OpenAIGatewayService{}
	svc.rateLimitService = NewRateLimitService(transientCooldownAccountRepo{}, nil, &config.Config{}, nil, nil)
	account := &Account{ID: 5301, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"message":"Upstream request failed","type":"upstream_error"}}`)

	ctx := WithOpenAIHostedWebSearchRequest(context.Background())
	for i := 0; i < 3; i++ {
		require.False(t, svc.handleOpenAIAccountUpstreamError(ctx, account, http.StatusBadGateway, http.Header{}, body, "gpt-6-luna"))
	}
	require.False(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-6-luna"),
		"a failing web_search request must not block the model for everyone else")

	// Without the tool the same failures still cool the model down.
	for i := 0; i < 3; i++ {
		svc.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusBadGateway, http.Header{}, body, "gpt-6-luna")
	}
	require.True(t, svc.isOpenAIAccountModelRuntimeBlocked(account, "gpt-6-luna"))
}

func TestOpenAIBodyHasHostedWebSearchTool(t *testing.T) {
	require.True(t, OpenAIBodyHasHostedWebSearchTool([]byte(`{"tools":[{"type":"function","name":"f"},{"type":"web_search"}]}`)))
	require.True(t, OpenAIBodyHasHostedWebSearchTool([]byte(`{"tools":[{"type":"web_search_preview"}]}`)))
	require.False(t, OpenAIBodyHasHostedWebSearchTool([]byte(`{"tools":[{"type":"function","name":"web_search"}]}`)), "a user function named web_search is not the hosted tool")
	require.False(t, OpenAIBodyHasHostedWebSearchTool([]byte(`{"input":"hi"}`)))
}
