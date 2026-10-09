package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// The ops assistant's tools: read-only lookups of this site's failed and successful requests,
// accounts and alerts, plus the configured upstream site's (ops_agent_upstream.go). Times go to the
// model as Beijing time.

const (
	opsAgentDefaultMinutes  = 60
	opsAgentMaxMinutes      = 7 * 24 * 60
	opsAgentMaxUpMinutes    = 12 * 60
	opsAgentTimelineBuckets = 24
)

var errOpsAgentData = errors.New("查询失败，请稍后再试")

func opsMinutes(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	if v < 5 {
		return 5
	}
	return v
}

func opsIntProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
func opsStrProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func opsRate(errs, success int) float64 {
	if errs+success == 0 {
		return 0
	}
	return float64(int(float64(errs)*10000/float64(errs+success)+0.5)) / 100
}

func opsTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return agentClock(*t)
}

func (s *OpsAgentService) window(minutes int) (time.Time, time.Time) {
	end := s.now()
	return end.Add(-time.Duration(minutes) * time.Minute), end
}

func (s *OpsAgentService) tools(up *opsUpstreamClient) []AgentTool {
	var tools []AgentTool
	if s.repo != nil {
		tools = append(tools, s.localTools()...)
	}
	if up != nil {
		tools = append(tools, upstreamTools(up, s.window)...)
	}
	return tools
}

func (s *OpsAgentService) localTools() []AgentTool {
	repo := s.repo
	return []AgentTool{
		{
			Name: "get_error_overview",
			Description: "本站某个时间窗口内的总览：成功数、失败数、错误率、用户侧限制（余额不足等）数，失败按类型分组（状态码、上游状态码、环节、报错信息、涉及模型和账号、首次和最近时间），" +
				"以及按账号、模型、用户的成功/失败数。排查任何问题先调用它。",
			Label:      "正在汇总报错",
			Parameters: assistantObject(map[string]any{"minutes": opsIntProp("最近多少分钟，5–10080，默认 60")}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var a struct {
					Minutes int `json:"minutes"`
				}
				_ = json.Unmarshal(raw, &a)
				m := opsMinutes(a.Minutes, opsAgentDefaultMinutes, opsAgentMaxMinutes)
				start, end := s.window(m)
				sum, err := repo.ErrorSummary(ctx, start, end)
				if err != nil {
					return nil, errOpsAgentData
				}
				groups := []map[string]any{}
				for _, g := range sum.Groups {
					row := map[string]any{"status_code": g.StatusCode, "phase": g.Phase, "type": g.Type, "message": g.Message, "count": g.Count,
						"users": g.Users, "models": g.Models, "accounts": g.Accounts, "first": agentClock(g.First), "last": agentClock(g.Last)}
					if g.UpstreamStatus != nil {
						row["upstream_status"] = *g.UpstreamStatus
					}
					if g.BusinessLimited {
						row["business_limited"] = true
					}
					groups = append(groups, row)
				}
				return map[string]any{
					"window":  fmt.Sprintf("%s – %s（%d 分钟）", agentClock(start), agentClock(end), m),
					"success": sum.Success, "errors": sum.Errors, "business_limited": sum.BusinessLimited,
					"error_rate_pct":               opsRate(sum.Errors, sum.Success),
					"error_rate_excl_business_pct": opsRate(sum.Errors-sum.BusinessLimited, sum.Success),
					"failure_kinds":                groups, "by_account": sum.ByAccount, "by_model": sum.ByModel, "by_user": sum.ByUser,
				}, nil
			},
		},
		{
			Name: "list_errors",
			Description: "本站最近失败的请求明细（新的在前）：时间、用户、分组、账号、模型、路径、是否流式、状态码、上游状态码、环节、报错信息、上游原始返回、依次试过的账号、耗时。" +
				"可按模型、账号 ID、用户 ID、状态码过滤。",
			Label: "正在查看报错明细",
			Parameters: assistantObject(map[string]any{
				"minutes":     opsIntProp("最近多少分钟，5–10080，默认 60"),
				"model":       opsStrProp("模型名（模糊匹配，可选）"),
				"account_id":  opsIntProp("本站上游账号 ID（可选）"),
				"user_id":     opsIntProp("用户 ID（可选）"),
				"status_code": opsIntProp("状态码，匹配返回给用户的或上游返回的（可选）"),
				"limit":       opsIntProp("条数，1–50，默认 20"),
			}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var a struct {
					Minutes    int    `json:"minutes"`
					Model      string `json:"model"`
					AccountID  int64  `json:"account_id"`
					UserID     int64  `json:"user_id"`
					StatusCode int    `json:"status_code"`
					Limit      int    `json:"limit"`
				}
				_ = json.Unmarshal(raw, &a)
				start, end := s.window(opsMinutes(a.Minutes, opsAgentDefaultMinutes, opsAgentMaxMinutes))
				rows, err := repo.ListErrors(ctx, OpsAgentErrorFilter{Start: start, End: end, Model: a.Model, AccountID: a.AccountID,
					UserID: a.UserID, StatusCode: a.StatusCode, Limit: a.Limit})
				if err != nil {
					return nil, errOpsAgentData
				}
				out := []map[string]any{}
				for _, e := range rows {
					row := map[string]any{"time": agentClock(e.Time), "model": e.Model, "path": e.Path, "stream": e.Stream,
						"status_code": e.StatusCode, "phase": e.Phase, "type": e.Type, "message": e.Message}
					for k, v := range map[string]string{"user": e.UserEmail, "group": e.Group, "account": e.Account, "upstream_detail": e.UpstreamDetail} {
						if v != "" {
							row[k] = v
						}
					}
					if e.UserID != nil {
						row["user_id"] = *e.UserID
					}
					if e.UpstreamStatus != nil {
						row["upstream_status"] = *e.UpstreamStatus
					}
					if len(e.Tried) > 0 {
						row["tried"] = e.Tried
					}
					if e.DurationMs != nil {
						row["duration_ms"] = *e.DurationMs
					}
					out = append(out, row)
				}
				return map[string]any{"rows": out}, nil
			},
		},
		{
			Name:        "get_timeline",
			Description: "本站成功数和失败数随时间的变化（按时间段分桶），用来判断问题从什么时候开始、是否还在持续。",
			Label:       "正在看报错时间线",
			Parameters: assistantObject(map[string]any{
				"minutes":        opsIntProp("最近多少分钟，10–10080，默认 180"),
				"bucket_minutes": opsIntProp("每段多少分钟（可选，默认自动分成约 24 段）"),
			}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var a struct {
					Minutes       int `json:"minutes"`
					BucketMinutes int `json:"bucket_minutes"`
				}
				_ = json.Unmarshal(raw, &a)
				m := opsMinutes(a.Minutes, 180, opsAgentMaxMinutes)
				bucket := a.BucketMinutes
				if bucket <= 0 {
					bucket = max(1, m/opsAgentTimelineBuckets)
				}
				bucket = max(bucket, m/60) // at most ~60 rows
				start, end := s.window(m)
				list, err := repo.Timeline(ctx, start, end, time.Duration(bucket)*time.Minute)
				if err != nil {
					return nil, errOpsAgentData
				}
				out := []map[string]any{}
				for _, b := range list {
					out = append(out, map[string]any{"from": agentClock(b.Start), "success": b.Success, "errors": b.Errors,
						"error_rate_pct": opsRate(b.Errors, b.Success)})
				}
				return map[string]any{"bucket_minutes": bucket, "buckets": out, "note": "没有请求的时间段不列出"}, nil
			},
		},
		{
			Name: "list_accounts",
			Description: "本站所有上游账号的状态：平台、类型、是否可调度、并发、优先级、所属分组、API 地址的域名、错误信息、限流/过载/临时停用到什么时候、最后使用时间。" +
				"用来判断某个分组是不是没有可用账号、某个账号是否被限流。",
			Label:      "正在查看上游账号状态",
			Parameters: assistantObject(nil),
			Run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				list, err := repo.Accounts(ctx)
				if err != nil {
					return nil, errOpsAgentData
				}
				out := []map[string]any{}
				for _, a := range list {
					row := map[string]any{"id": a.ID, "name": a.Name, "platform": a.Platform, "type": a.Type, "status": a.Status,
						"schedulable": a.Schedulable, "concurrency": a.Concurrency, "priority": a.Priority, "groups": a.Groups}
					for k, v := range map[string]string{"base_host": a.BaseHost, "error_message": agentCut(a.ErrorMessage, 200),
						"rate_limited_at": opsTimePtr(a.RateLimitedAt), "rate_limit_reset_at": opsTimePtr(a.RateLimitResetAt),
						"overload_until": opsTimePtr(a.OverloadUntil), "temp_unschedulable_until": opsTimePtr(a.TempUnschedulableUntil),
						"temp_unschedulable_reason": agentCut(a.TempUnschedulableWhy, 200), "last_used_at": opsTimePtr(a.LastUsedAt)} {
						if v != "" {
							row[k] = v
						}
					}
					out = append(out, row)
				}
				return map[string]any{"accounts": out, "now": agentClock(s.now())}, nil
			},
		},
		{
			Name:        "list_alerts",
			Description: "本站最近的告警事件：标题、描述（指标、阈值、当前值、窗口）、级别、状态（firing 未恢复 / resolved 已恢复）、触发和恢复时间。",
			Label:       "正在查看告警记录",
			Parameters:  assistantObject(map[string]any{"hours": opsIntProp("最近多少小时，1–168，默认 24")}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var a struct {
					Hours int `json:"hours"`
				}
				_ = json.Unmarshal(raw, &a)
				if a.Hours <= 0 || a.Hours > 168 {
					a.Hours = 24
				}
				list, err := repo.AlertEvents(ctx, s.now().Add(-time.Duration(a.Hours)*time.Hour), 30)
				if err != nil {
					return nil, errOpsAgentData
				}
				out := []map[string]any{}
				for _, e := range list {
					row := map[string]any{"title": e.Title, "description": e.Description, "severity": e.Severity, "status": e.Status, "fired_at": agentClock(e.FiredAt)}
					if e.ResolvedAt != nil {
						row["resolved_at"] = agentClock(*e.ResolvedAt)
					}
					out = append(out, row)
				}
				return map[string]any{"alerts": out}, nil
			},
		},
	}
}

func upstreamTools(up *opsUpstreamClient, window func(int) (time.Time, time.Time)) []AgentTool {
	minutesArg := func(raw json.RawMessage, def int) (int, time.Time, time.Time) {
		var a struct {
			Minutes int `json:"minutes"`
		}
		_ = json.Unmarshal(raw, &a)
		m := opsMinutes(a.Minutes, def, opsAgentMaxUpMinutes)
		start, end := window(m)
		return m, start, end
	}
	name := up.name
	return []AgentTool{
		{
			Name:        "upstream_overview",
			Description: fmt.Sprintf("上游站 %s 自己在某个时间窗口内的总览：请求数、成功数、失败数、成功率、错误率、上游错误率、429 次数。", name),
			Label:       "正在查看上游站总览",
			Parameters:  assistantObject(map[string]any{"minutes": opsIntProp("最近多少分钟，5–720，默认 60")}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				_, start, end := minutesArg(raw, opsAgentDefaultMinutes)
				return up.Overview(ctx, start, end)
			},
		},
		{
			Name: "upstream_error_summary",
			Description: fmt.Sprintf("上游站 %s 在某个时间窗口内的失败汇总：按（账号、状态码、报错信息）分类计数，每个账号的成功数和失败数，按分组、用户的失败数。"+
				"本站看到上游 502/503/超时时，用它判断是上游哪个账号出的问题。", name),
			Label:      "正在汇总上游站的报错",
			Parameters: assistantObject(map[string]any{"minutes": opsIntProp("最近多少分钟，5–720，默认 60")}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				m, start, end := minutesArg(raw, opsAgentDefaultMinutes)
				out, err := up.ErrorSummary(ctx, start, end)
				if err != nil {
					return nil, err
				}
				out["window"] = fmt.Sprintf("%s – %s（%d 分钟）", agentClock(start), agentClock(end), m)
				return out, nil
			},
		},
		{
			Name:        "upstream_list_errors",
			Description: fmt.Sprintf("上游站 %s 最近失败的请求明细（新的在前）：时间、账号、模型、状态码、环节、报错信息、分组、用户。可按上游账号 ID、模型过滤。", name),
			Label:       "正在查看上游站的报错明细",
			Parameters: assistantObject(map[string]any{
				"minutes":    opsIntProp("最近多少分钟，5–720，默认 60"),
				"account_id": opsIntProp("上游站的账号 ID（可选）"),
				"model":      opsStrProp("模型名（模糊匹配，可选）"),
				"limit":      opsIntProp("条数，1–30，默认 15"),
			}),
			Run: func(ctx context.Context, raw json.RawMessage) (any, error) {
				var a struct {
					AccountID int64  `json:"account_id"`
					Model     string `json:"model"`
					Limit     int    `json:"limit"`
				}
				_ = json.Unmarshal(raw, &a)
				if a.Limit <= 0 || a.Limit > 30 {
					a.Limit = 15
				}
				_, start, end := minutesArg(raw, opsAgentDefaultMinutes)
				return up.ListErrors(ctx, start, end, a.AccountID, a.Model, a.Limit)
			},
		},
		{
			Name: "upstream_accounts",
			Description: fmt.Sprintf("上游站 %s 的账号状态：是否可调度、并发、优先级、所属分组、错误信息、限流/过载/临时停用到什么时候，"+
				"Codex 账号的套餐和 5 小时 / 7 天用量百分比。", name),
			Label:      "正在查看上游站的账号",
			Parameters: assistantObject(nil),
			Run: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return up.Accounts(ctx)
			},
		},
	}
}
