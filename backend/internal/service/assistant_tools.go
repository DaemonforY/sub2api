package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// The support assistant's tools (账户诊断): read-only lookups of the asking user's own account —
// balance, plans, keys, recent errors and usage, the models a key can call — plus a search of the
// FAQ and tutorials. Every tool is bound to the signed-in user's ID when the run starts; the model
// can't name another user. Key secrets are never returned.

// AssistantSources: where the account tools read from. Any of them may be nil (that tool is left out).
type AssistantSources struct {
	Users interface {
		GetByID(ctx context.Context, id int64) (*User, error)
	}
	Usage interface {
		ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams) ([]UsageLog, *pagination.PaginationResult, error)
	}
	Errors interface {
		ListUserErrorRequests(ctx context.Context, userID int64, filter *OpsErrorLogFilter) (*UserErrorRequestList, error)
	}
	// ErrorView: the same switch that shows users their own failed requests (用量 → 错误请求).
	ErrorView interface {
		IsUserErrorViewAllowed(ctx context.Context) bool
	}
	Subs interface {
		ListActiveUserSubscriptions(ctx context.Context, userID int64) ([]UserSubscription, error)
	}
	Runs AgentRunRepository
}

const assistantToolTime = "2006-01-02 15:04"

func assistantTime(t time.Time) string { return t.In(learnDayZone).Format(assistantToolTime) }

func assistantMoney(v float64) float64 { return math.Round(v*10000) / 10000 }

var assistantKeyStatus = map[string]string{
	StatusAPIKeyActive:         "可用",
	StatusAPIKeyDisabled:       "已停用",
	StatusAPIKeyQuotaExhausted: "额度已用完",
	StatusAPIKeyExpired:        "已过期",
}

func assistantStatusText(m map[string]string, s string) string {
	if v, ok := m[s]; ok {
		return v
	}
	return s
}

var errAssistantToolUnavailable = errors.New("暂时查不到，请稍后再试或联系人工客服")

func assistantObject(props map[string]any) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	return map[string]any{"type": "object", "properties": props}
}

// assistantTools builds the tool list for one run. userID <= 0 (a visitor) gets only the doc search.
func (s *AssistantService) assistantTools(ctx context.Context, userID int64) []AgentTool {
	tools := []AgentTool{s.searchDocsTool()}
	if userID <= 0 {
		return tools
	}
	src := s.sources
	var keys learnKeySource
	if s.learn != nil {
		keys = s.learn.sources.Keys
	}
	if src.Users != nil {
		tools = append(tools, AgentTool{
			Name:        "get_my_account",
			Description: "查询当前用户自己的账户：余额（美元）、账户状态、并发上限，以及生效中的订阅套餐（分组、到期时间、日/周/月用量和上限）。",
			Label:       "正在查看你的账户和余额",
			Parameters:  assistantObject(nil),
			Run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				u, err := src.Users.GetByID(ctx, userID)
				if err != nil || u == nil {
					return nil, errAssistantToolUnavailable
				}
				out := map[string]any{
					"balance_usd":   assistantMoney(u.Balance),
					"status":        assistantStatusText(map[string]string{StatusActive: "正常", StatusDisabled: "已禁用"}, u.Status),
					"concurrency":   u.Concurrency,
					"registered_at": assistantTime(u.CreatedAt),
				}
				if u.FrozenBalance > 0 {
					out["frozen_balance_usd"] = assistantMoney(u.FrozenBalance)
				}
				if src.Subs != nil {
					subs, err := src.Subs.ListActiveUserSubscriptions(ctx, userID)
					if err == nil {
						var list []map[string]any
						for _, sub := range subs {
							item := map[string]any{"expires_at": assistantTime(sub.ExpiresAt),
								"daily_used_usd": assistantMoney(sub.DailyUsageUSD), "weekly_used_usd": assistantMoney(sub.WeeklyUsageUSD),
								"monthly_used_usd": assistantMoney(sub.MonthlyUsageUSD)}
							if g := sub.Group; g != nil {
								item["group"] = g.Name
								for k, v := range map[string]*float64{"daily_limit_usd": g.DailyLimitUSD, "weekly_limit_usd": g.WeeklyLimitUSD, "monthly_limit_usd": g.MonthlyLimitUSD} {
									if v != nil && *v > 0 {
										item[k] = *v
									}
								}
							}
							list = append(list, item)
						}
						out["subscriptions"] = list
					}
				}
				return out, nil
			},
		})
	}
	if keys != nil {
		tools = append(tools, AgentTool{
			Name:        "list_my_keys",
			Description: "列出当前用户自己的 API Key（不含密钥本身）：ID、名称、状态、所属分组和平台、额度与已用、过期时间、5 小时/1 天/7 天限额、最后使用时间。",
			Label:       "正在查看你的 API Key",
			Parameters:  assistantObject(nil),
			Run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				list, _, err := keys.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 50}, APIKeyListFilters{})
				if err != nil {
					return nil, errAssistantToolUnavailable
				}
				out := []map[string]any{}
				for _, k := range list {
					item := map[string]any{"id": k.ID, "name": k.Name, "status": assistantStatusText(assistantKeyStatus, k.Status),
						"created_at": assistantTime(k.CreatedAt)}
					if k.Group != nil {
						item["group"] = k.Group.Name
						item["platform"] = k.Group.Platform
					} else {
						item["group"] = "（未绑定分组）"
					}
					if k.Quota > 0 {
						item["quota_usd"], item["quota_used_usd"] = assistantMoney(k.Quota), assistantMoney(k.QuotaUsed)
					}
					if k.ExpiresAt != nil {
						item["expires_at"] = assistantTime(*k.ExpiresAt)
					}
					for name, lim := range map[string][2]float64{"limit_5h": {k.RateLimit5h, k.Usage5h}, "limit_1d": {k.RateLimit1d, k.Usage1d}, "limit_7d": {k.RateLimit7d, k.Usage7d}} {
						if lim[0] > 0 {
							item[name] = fmt.Sprintf("已用 $%.4g / 上限 $%.4g", lim[1], lim[0])
						}
					}
					if k.LastUsedAt != nil {
						item["last_used_at"] = assistantTime(*k.LastUsedAt)
					}
					out = append(out, item)
				}
				return map[string]any{"keys": out, "total": len(out)}, nil
			},
		})
		tools = append(tools, AgentTool{
			Name:        "list_key_models",
			Description: "查询当前用户某个 API Key 能调用的模型列表（按它所在分组）。用来判断「分组不支持模型」「模型不存在」一类问题。key_id 用 list_my_keys 查到的 ID。",
			Label:       "正在查这个 Key 能用哪些模型",
			Parameters: map[string]any{"type": "object", "required": []string{"key_id"}, "properties": map[string]any{
				"key_id": map[string]any{"type": "integer", "description": "API Key 的 ID"}}},
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var args struct {
					KeyID int64 `json:"key_id"`
				}
				_ = json.Unmarshal(raw, &args)
				k, err := keys.GetByID(ctx, args.KeyID)
				if err != nil || k == nil || k.UserID != userID {
					return nil, errors.New("没有找到这个 Key，请先用 list_my_keys 查 ID")
				}
				if !k.IsActive() {
					return map[string]any{"key": k.Name, "status": assistantStatusText(assistantKeyStatus, k.Status), "models": []string{},
						"note": "这个 Key 当前不可用，任何模型都调用不了"}, nil
				}
				models, err := s.keyModels(ctx, k.Key)
				if err != nil {
					return nil, err
				}
				out := map[string]any{"key": k.Name, "models": models, "count": len(models)}
				if k.Group != nil {
					out["group"] = k.Group.Name
				}
				return out, nil
			},
		})
	}
	if src.Errors != nil && (src.ErrorView == nil || src.ErrorView.IsUserErrorViewAllowed(ctx)) {
		tools = append(tools, AgentTool{
			Name:        "get_recent_errors",
			Description: "查询当前用户最近失败的 API 请求（最多 10 条，新的在前）：时间、Key 名称、分组、模型、状态码、错误分类和错误信息。排查报错时先调用它。",
			Label:       "正在查你最近的报错",
			Parameters: assistantObject(map[string]any{
				"hours": map[string]any{"type": "integer", "description": "查最近多少小时，1–168，默认 72"},
				"model": map[string]any{"type": "string", "description": "只看某个模型（可选，模糊匹配）"}}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var args struct {
					Hours int    `json:"hours"`
					Model string `json:"model"`
				}
				_ = json.Unmarshal(raw, &args)
				if args.Hours <= 0 || args.Hours > 168 {
					args.Hours = 72
				}
				start := s.now().Add(-time.Duration(args.Hours) * time.Hour)
				list, err := src.Errors.ListUserErrorRequests(ctx, userID, &OpsErrorLogFilter{StartTime: &start, Model: strings.TrimSpace(args.Model), Page: 1, PageSize: 10})
				if err != nil || list == nil {
					return nil, errAssistantToolUnavailable
				}
				out := []map[string]any{}
				for _, e := range list.Items {
					item := map[string]any{"time": assistantTime(e.CreatedAt), "key": e.KeyName, "model": e.Model,
						"status_code": e.StatusCode, "category": e.Category, "message": agentCut(e.Message, 300), "endpoint": e.InboundEndpoint}
					if e.GroupName != "" {
						item["group"] = e.GroupName
					}
					if e.KeyDeleted {
						item["key_deleted"] = true
					}
					out = append(out, item)
				}
				return map[string]any{"hours": args.Hours, "total": list.Total, "errors": out}, nil
			},
		})
	}
	if src.Usage != nil {
		tools = append(tools, AgentTool{
			Name:        "get_recent_usage",
			Description: "查询当前用户最近成功的调用记录（新的在前）：时间、Key 名称、模型、输入/输出/缓存 token、实际扣费（美元）、耗时。用来回答「扣了多少钱」「为什么这么贵」「调用有没有成功」。",
			Label:       "正在查你最近的用量",
			Parameters: assistantObject(map[string]any{
				"limit": map[string]any{"type": "integer", "description": "条数，1–20，默认 10"}}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var args struct {
					Limit int `json:"limit"`
				}
				_ = json.Unmarshal(raw, &args)
				if args.Limit <= 0 || args.Limit > 20 {
					args.Limit = 10
				}
				logs, _, err := src.Usage.ListByUser(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: args.Limit, SortBy: "created_at", SortOrder: "desc"})
				if err != nil {
					return nil, errAssistantToolUnavailable
				}
				names := map[int64]string{}
				if keys != nil {
					if list, _, err := keys.ListByUserID(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, APIKeyListFilters{}); err == nil {
						for _, k := range list {
							names[k.ID] = k.Name
						}
					}
				}
				out := []map[string]any{}
				for _, l := range logs {
					item := map[string]any{"time": assistantTime(l.CreatedAt), "model": l.Model,
						"input_tokens": l.InputTokens, "output_tokens": l.OutputTokens, "cache_read_tokens": l.CacheReadTokens,
						"cost_usd": assistantMoney(l.ActualCost)}
					if n := names[l.APIKeyID]; n != "" {
						item["key"] = n
					}
					if l.DurationMs != nil {
						item["duration_ms"] = *l.DurationMs
					}
					if l.ImageCount > 0 {
						item["images"] = l.ImageCount
					}
					out = append(out, item)
				}
				return map[string]any{"records": out}, nil
			},
		})
	}
	return tools
}

func (s *AssistantService) searchDocsTool() AgentTool {
	return AgentTool{
		Name:        "search_docs",
		Description: "在本站说明和 AI 学习站教程里检索。系统提示里的参考资料不够回答时，换个说法再查。",
		Label:       "正在查教程",
		Parameters: map[string]any{"type": "object", "required": []string{"query"}, "properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "检索关键词或问题"}}},
		Run: func(_ context.Context, raw json.RawMessage) (any, error) {
			var args struct {
				Query string `json:"query"`
			}
			_ = json.Unmarshal(raw, &args)
			if strings.TrimSpace(args.Query) == "" {
				return nil, errors.New("query 不能为空")
			}
			out := []map[string]string{}
			for _, h := range s.knowledge().search(args.Query, 4) {
				out = append(out, map[string]string{"title": h.Title, "url": h.URL, "text": h.Text})
			}
			return map[string]any{"results": out}, nil
		},
	}
}

// keyModels asks this site's gateway which models a key can call (GET /v1/models with that key).
func (s *AssistantService) keyModels(ctx context.Context, key string) ([]string, error) {
	gateway := ""
	if s.learn != nil {
		gateway = s.learn.gatewayURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gateway+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", "hivegpt-assistant/1")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errAssistantToolUnavailable
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询模型列表失败：%s", upstreamErrorText(raw, resp.StatusCode))
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return nil, errAssistantToolUnavailable
	}
	models := []string{}
	for _, m := range body.Data {
		if m.ID != "" && len(models) < 100 {
			models = append(models, m.ID)
		}
	}
	return models, nil
}
