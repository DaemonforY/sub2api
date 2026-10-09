package service

import (
	"context"
	"strings"

	"github.com/tidwall/gjson"
)

// A Responses request carrying OpenAI's hosted web_search tool (web_search, web_search_preview, …)
// can fail upstream with a 5xx when the upstream account cannot run the tool, while the same model
// works for every other request. Such failures describe the request, not the account, so they must
// not put the account+model into a transient cooldown: that blocked the model for the whole group.

type openAIHostedWebSearchCtxKey struct{}

// OpenAIBodyHasHostedWebSearchTool reports whether a Responses body lists a hosted web_search tool.
func OpenAIBodyHasHostedWebSearchTool(body []byte) bool {
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		if strings.HasPrefix(tool.Get("type").String(), toolTypeWebSearchPrefix) {
			return true
		}
	}
	return false
}

// WithOpenAIHostedWebSearchRequest marks ctx as serving a request that uses the hosted web_search tool.
func WithOpenAIHostedWebSearchRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, openAIHostedWebSearchCtxKey{}, true)
}

func isOpenAIHostedWebSearchRequest(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(openAIHostedWebSearchCtxKey{}).(bool)
	return v
}
