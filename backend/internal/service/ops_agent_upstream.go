package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The ops assistant's view of an upstream that is itself a Sub2API site (an API-key account here
// points at it): read through that site's own admin API with an Admin API Key (x-api-key) the
// admin pasted in 运维助手 settings. Only the fields below are passed on — never its keys or
// credentials.

const (
	opsUpstreamTimeout   = 8 * time.Second
	opsUpstreamMaxBody   = 8 << 20
	opsUpstreamErrorPage = 500
	opsUpstreamUsagePage = 1000
)

type opsUpstreamClient struct {
	name    string
	baseURL string
	key     string
	http    *http.Client
	now     func() time.Time
}

var errOpsUpstreamAuth = errors.New("上游站拒绝了管理员 API Key（401/403）：请在上游后台「系统设置」重新生成 Admin API Key 并填到运维助手设置里")

// get calls GET path?query and decodes the {code, message, data} envelope's data into out.
func (c *opsUpstreamClient) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", c.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "hivegpt-ops-agent/1")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("连不上上游站 %s：%v", c.name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, opsUpstreamMaxBody))
	if err != nil {
		return fmt.Errorf("读取上游站响应失败：%v", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return errOpsUpstreamAuth
	}
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("上游站返回的不是 JSON（HTTP %d），请确认地址是 Sub2API 站点根地址", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || env.Code != 0 {
		return fmt.Errorf("上游站返回错误（HTTP %d）：%s", resp.StatusCode, agentCut(env.Message, 200))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(env.Data, out)
}

func opsUpstreamRange(start, end time.Time) url.Values {
	return url.Values{"start_time": {start.UTC().Format(time.RFC3339)}, "end_time": {end.UTC().Format(time.RFC3339)}}
}

// Overview: the upstream site's own totals for the window.
func (c *opsUpstreamClient) Overview(ctx context.Context, start, end time.Time) (map[string]any, error) {
	var o struct {
		SuccessCount         int64   `json:"success_count"`
		ErrorCountTotal      int64   `json:"error_count_total"`
		BusinessLimitedCount int64   `json:"business_limited_count"`
		RequestCountTotal    int64   `json:"request_count_total"`
		SLA                  float64 `json:"sla"`
		ErrorRate            float64 `json:"error_rate"`
		UpstreamErrorRate    float64 `json:"upstream_error_rate"`
		Upstream429Count     int64   `json:"upstream_429_count"`
	}
	if err := c.get(ctx, "/api/v1/admin/ops/dashboard/overview", opsUpstreamRange(start, end), &o); err != nil {
		return nil, err
	}
	return map[string]any{
		"success": o.SuccessCount, "errors": o.ErrorCountTotal, "business_limited": o.BusinessLimitedCount,
		"requests": o.RequestCountTotal, "success_rate_pct": opsPct(o.SLA), "error_rate_pct": opsPct(o.ErrorRate),
		"upstream_error_rate_pct": opsPct(o.UpstreamErrorRate), "upstream_429": o.Upstream429Count,
	}, nil
}

func opsPct(v float64) float64 { return float64(int(v*10000+0.5)) / 100 }

// opsUpstreamKind: one kind of upstream failure (same account, status and message).
type opsUpstreamKind struct {
	Account    string   `json:"account"`
	StatusCode int      `json:"status_code"`
	Message    string   `json:"message"`
	Count      int      `json:"count"`
	Models     []string `json:"models"`
	First      string   `json:"first"`
	Last       string   `json:"last"`
}

// opsUpstreamAccountCount: an upstream account's failures and, when its usage log could be read, successes.
type opsUpstreamAccountCount struct {
	Account string `json:"account"`
	Errors  int    `json:"errors"`
	Success *int   `json:"success,omitempty"`
}

type opsUpstreamError struct {
	CreatedAt   time.Time `json:"created_at"`
	Phase       string    `json:"phase"`
	Type        string    `json:"type"`
	StatusCode  int       `json:"status_code"`
	Model       string    `json:"model"`
	Message     string    `json:"message"`
	UserID      *int64    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	AccountID   *int64    `json:"account_id"`
	AccountName string    `json:"account_name"`
	GroupName   string    `json:"group_name"`
	RequestPath string    `json:"request_path"`
}

func (c *opsUpstreamClient) errors(ctx context.Context, start, end time.Time) ([]opsUpstreamError, int, error) {
	q := opsUpstreamRange(start, end)
	q.Set("page", "1")
	q.Set("page_size", strconv.Itoa(opsUpstreamErrorPage))
	q.Set("view", "all")
	var page struct {
		Items []opsUpstreamError `json:"items"`
		Total int                `json:"total"`
	}
	if err := c.get(ctx, "/api/v1/admin/ops/errors", q, &page); err != nil {
		return nil, 0, err
	}
	return page.Items, page.Total, nil
}

func opsUpstreamAccountKey(id *int64, name string) string {
	if id == nil {
		return "（未分配账号）"
	}
	return strings.TrimSpace(fmt.Sprintf("#%d %s", *id, name))
}

// ErrorSummary groups the upstream's failed requests in the window: by kind (account, status,
// message), per account, per group and per user, plus the per-account successes from its usage log.
func (c *opsUpstreamClient) ErrorSummary(ctx context.Context, start, end time.Time) (map[string]any, error) {
	items, total, err := c.errors(ctx, start, end)
	if err != nil {
		return nil, err
	}
	type kind struct {
		account, msg string
		status       int
	}
	kinds := map[kind]*struct {
		n           int
		models      map[string]bool
		first, last time.Time
	}{}
	errByAccount, byGroup, byUser := map[int64]int{}, map[string]int{}, map[string]int{}
	names := map[int64]string{}
	for _, e := range items {
		acct := opsUpstreamAccountKey(e.AccountID, e.AccountName)
		k := kind{acct, agentCut(strings.TrimSpace(e.Message), 160), e.StatusCode}
		v := kinds[k]
		if v == nil {
			v = &struct {
				n           int
				models      map[string]bool
				first, last time.Time
			}{models: map[string]bool{}, first: e.CreatedAt, last: e.CreatedAt}
			kinds[k] = v
		}
		v.n++
		if e.Model != "" {
			v.models[e.Model] = true
		}
		if e.CreatedAt.Before(v.first) {
			v.first = e.CreatedAt
		}
		if e.CreatedAt.After(v.last) {
			v.last = e.CreatedAt
		}
		id := int64(0)
		if e.AccountID != nil {
			id = *e.AccountID
			if e.AccountName != "" {
				names[id] = e.AccountName
			}
		}
		errByAccount[id]++
		if g := strings.TrimSpace(e.GroupName); g != "" {
			byGroup[g]++
		}
		if e.UserID != nil {
			byUser[strings.TrimSpace(fmt.Sprintf("#%d %s", *e.UserID, e.UserEmail))]++
		}
	}
	groups := []opsUpstreamKind{}
	for k, v := range kinds {
		models := make([]string, 0, len(v.models))
		for m := range v.models {
			models = append(models, m)
		}
		sort.Strings(models)
		if len(models) > 5 {
			models = models[:5]
		}
		groups = append(groups, opsUpstreamKind{Account: k.account, StatusCode: k.status, Message: k.msg, Count: v.n,
			Models: models, First: agentClock(v.first), Last: agentClock(v.last)})
	}
	sort.Slice(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Account != b.Account {
			return a.Account < b.Account
		}
		return a.Message < b.Message
	})
	if len(groups) > 15 {
		groups = groups[:15]
	}

	out := map[string]any{"errors": total, "kinds": groups, "errors_by_group": opsTopCounts(byGroup, 8), "errors_by_user": opsTopCounts(byUser, 8)}
	if total > len(items) {
		out["note"] = fmt.Sprintf("报错共 %d 条，只汇总了最近 %d 条", total, len(items))
	}
	success, complete, err := c.successByAccount(ctx, start, end, names)
	accounts := map[int64]bool{}
	for k := range errByAccount {
		accounts[k] = true
	}
	for k := range success {
		accounts[k] = true
	}
	perAccount := []opsUpstreamAccountCount{}
	for id := range accounts {
		label := "（未分配账号）"
		if id > 0 {
			label = opsUpstreamAccountKey(&id, names[id])
		}
		row := opsUpstreamAccountCount{Account: label, Errors: errByAccount[id]}
		if err == nil {
			n := success[id]
			row.Success = &n
		}
		perAccount = append(perAccount, row)
	}
	sort.Slice(perAccount, func(i, j int) bool {
		a, b := perAccount[i], perAccount[j]
		return a.Errors > b.Errors || (a.Errors == b.Errors && a.Account < b.Account)
	})
	out["by_account"] = perAccount
	switch {
	case err != nil:
		out["success_note"] = "各账号成功数查不到：" + err.Error()
	case !complete:
		out["success_note"] = fmt.Sprintf("成功数只统计了最近 %d 条用量记录，窗口较早的部分没算进去", opsUpstreamUsagePage)
	}
	return out, nil
}

// successByAccount counts the upstream's successful requests per account in the window, from the
// newest usage-log rows (its usage list filters by date only). complete is false when the rows
// didn't reach back to start.
func (c *opsUpstreamClient) successByAccount(ctx context.Context, start, end time.Time, names map[int64]string) (map[int64]int, bool, error) {
	loc := learnDayZone
	q := url.Values{
		"page": {"1"}, "page_size": {strconv.Itoa(opsUpstreamUsagePage)}, "sort_by": {"created_at"}, "sort_order": {"desc"},
		"start_date": {start.In(loc).Format("2006-01-02")}, "end_date": {end.In(loc).Format("2006-01-02")},
		"timezone": {loc.String()}, "exact_total": {"false"},
	}
	var page struct {
		Items []struct {
			CreatedAt time.Time `json:"created_at"`
			AccountID *int64    `json:"account_id"`
			Account   *struct {
				Name string `json:"name"`
			} `json:"account"`
		} `json:"items"`
	}
	if err := c.get(ctx, "/api/v1/admin/usage", q, &page); err != nil {
		return nil, false, err
	}
	out := map[int64]int{}
	complete := len(page.Items) < opsUpstreamUsagePage
	for _, u := range page.Items {
		if u.CreatedAt.Before(start) {
			complete = true
			continue
		}
		if !u.CreatedAt.Before(end) {
			continue
		}
		id := int64(0)
		if u.AccountID != nil {
			id = *u.AccountID
		}
		if u.Account != nil && u.Account.Name != "" && id > 0 {
			names[id] = u.Account.Name
		}
		out[id]++
	}
	return out, complete, nil
}

// ListErrors: the upstream's latest failed requests (optionally one account or model).
func (c *opsUpstreamClient) ListErrors(ctx context.Context, start, end time.Time, accountID int64, model string, limit int) (map[string]any, error) {
	items, total, err := c.errors(ctx, start, end)
	if err != nil {
		return nil, err
	}
	model = strings.ToLower(strings.TrimSpace(model))
	out := []map[string]any{}
	for _, e := range items {
		if accountID > 0 && (e.AccountID == nil || *e.AccountID != accountID) {
			continue
		}
		if model != "" && !strings.Contains(strings.ToLower(e.Model), model) {
			continue
		}
		row := map[string]any{"time": agentClock(e.CreatedAt), "account": opsUpstreamAccountKey(e.AccountID, e.AccountName),
			"model": e.Model, "status_code": e.StatusCode, "phase": e.Phase, "type": e.Type, "message": agentCut(e.Message, 300)}
		if e.GroupName != "" {
			row["group"] = e.GroupName
		}
		if e.UserID != nil {
			row["user"] = strings.TrimSpace(fmt.Sprintf("#%d %s", *e.UserID, e.UserEmail))
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return map[string]any{"errors_in_window": total, "rows": out}, nil
}

// Accounts: the upstream's accounts with scheduling state and, for Codex OAuth accounts, the usage
// windows it tracks. Credentials are never read out of the response.
func (c *opsUpstreamClient) Accounts(ctx context.Context) (map[string]any, error) {
	var page struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
	}
	if err := c.get(ctx, "/api/v1/admin/accounts", url.Values{"page": {"1"}, "page_size": {"100"}}, &page); err != nil {
		return nil, err
	}
	keep := []string{"id", "name", "platform", "type", "status", "schedulable", "concurrency", "priority", "error_message",
		"rate_limited_at", "rate_limit_reset_at", "overload_until", "temp_unschedulable_until", "temp_unschedulable_reason", "last_used_at"}
	extraKeep := []string{"codex_5h_used_percent", "codex_7d_used_percent", "codex_5h_reset_at", "codex_7d_reset_at", "codex_usage_updated_at", "plan_type"}
	out := []map[string]any{}
	for _, a := range page.Items {
		row := map[string]any{}
		for _, k := range keep {
			if v, ok := a[k]; ok && v != nil && v != "" {
				row[k] = v
			}
		}
		if groups, ok := a["groups"].([]any); ok {
			names := []string{}
			for _, g := range groups {
				if m, ok := g.(map[string]any); ok {
					if n, ok := m["name"].(string); ok {
						names = append(names, n)
					}
				}
			}
			row["groups"] = names
		}
		if extra, ok := a["extra"].(map[string]any); ok {
			for _, k := range extraKeep {
				if v, ok := extra[k]; ok && v != nil && v != "" {
					row[k] = v
				}
			}
		}
		out = append(out, row)
	}
	return map[string]any{"total": page.Total, "accounts": out}, nil
}

// agentClock: a time in the site's zone for tool results.
func agentClock(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(learnDayZone).Format("01-02 15:04:05")
}

type opsKeyCount struct {
	Count int    `json:"count"`
	Key   string `json:"key"`
}

func opsTopCounts(m map[string]int, n int) []opsKeyCount {
	out := make([]opsKeyCount, 0, len(m))
	for k, v := range m {
		out = append(out, opsKeyCount{Count: v, Key: k})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || (out[i].Count == out[j].Count && out[i].Key < out[j].Key)
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
