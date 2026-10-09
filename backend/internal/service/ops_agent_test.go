//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---- alert evaluator: minimum sample for rate rules ----

type opsEvalRepo struct {
	stubOpsRepo
	rules    []*OpsAlertRule
	active   *OpsAlertEvent
	created  []*OpsAlertEvent
	resolved []int64
}

func (r *opsEvalRepo) ListAlertRules(context.Context) ([]*OpsAlertRule, error) { return r.rules, nil }
func (r *opsEvalRepo) GetLatestSystemMetrics(context.Context, int) (*OpsSystemMetricsSnapshot, error) {
	return nil, nil
}
func (r *opsEvalRepo) UpsertJobHeartbeat(context.Context, *OpsUpsertJobHeartbeatInput) error {
	return nil
}
func (r *opsEvalRepo) GetActiveAlertEvent(context.Context, int64) (*OpsAlertEvent, error) {
	return r.active, nil
}
func (r *opsEvalRepo) GetLatestAlertEvent(context.Context, int64) (*OpsAlertEvent, error) {
	return nil, nil
}
func (r *opsEvalRepo) CreateAlertEvent(_ context.Context, e *OpsAlertEvent) (*OpsAlertEvent, error) {
	e.ID = int64(100 + len(r.created))
	r.created = append(r.created, e)
	return e, nil
}
func (r *opsEvalRepo) UpdateAlertEventStatus(_ context.Context, id int64, status string, _ *time.Time) error {
	if status == OpsAlertStatusResolved {
		r.resolved = append(r.resolved, id)
	}
	return nil
}

func TestOpsAlertMinRateSample(t *testing.T) {
	rule := &OpsAlertRule{ID: 1, Name: "错误率过高", Enabled: true, MetricType: "error_rate", Operator: ">", Threshold: 5, WindowMinutes: 5, Severity: "P1"}

	// 14 requests, 5 failed (36%): under the default minimum of 20 — not judged, the firing alert resolves.
	repo := &opsEvalRepo{rules: []*OpsAlertRule{rule}, active: &OpsAlertEvent{ID: 9, RuleID: 1, Status: OpsAlertStatusFiring}}
	repo.overview = &OpsDashboardOverview{RequestCountSLA: 14, ErrorRate: 5.0 / 14}
	var hooked []string
	svc := NewOpsAlertEvaluatorService(nil, repo, nil, nil, nil, nil)
	svc.SetAlertHook(func(_ *OpsAlertRule, e *OpsAlertEvent) { hooked = append(hooked, e.Title) })
	svc.evaluateOnce(time.Minute)
	require.Empty(t, repo.created)
	require.Equal(t, []int64{9}, repo.resolved)
	require.Empty(t, hooked)

	// 40 requests at 36%: judged, fires, and the hook hears about it.
	repo = &opsEvalRepo{rules: []*OpsAlertRule{rule}}
	repo.overview = &OpsDashboardOverview{RequestCountSLA: 40, ErrorRate: 0.36}
	svc = NewOpsAlertEvaluatorService(nil, repo, nil, nil, nil, nil)
	svc.SetAlertHook(func(_ *OpsAlertRule, e *OpsAlertEvent) { hooked = append(hooked, e.Title) })
	svc.evaluateOnce(time.Minute)
	require.Len(t, repo.created, 1)
	require.Equal(t, []string{"P1: 错误率过高"}, hooked)

	// Non-rate rules ignore the minimum; 0 turns it off.
	svc.minRateSample = 0
	repo.overview = &OpsDashboardOverview{RequestCountSLA: 3, ErrorRate: 0.5}
	v, ok := svc.computeRuleMetric(context.Background(), rule, nil, time.Now().Add(-time.Minute), time.Now(), "", nil)
	require.True(t, ok)
	require.InDelta(t, 50, v, 0.01)
}

func TestOpsAlertRuntimeSettingsMinSample(t *testing.T) {
	ctx := context.Background()
	settings := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: settings}

	// Stored settings from before the field existed: the default applies.
	settings.values[SettingKeyOpsAlertRuntimeSettings] = `{"evaluation_interval_seconds":60}`
	cfg, err := svc.GetOpsAlertRuntimeSettings(ctx)
	require.NoError(t, err)
	require.NotNil(t, cfg.MinRateSampleRequests)
	require.Equal(t, 20, *cfg.MinRateSampleRequests)

	zero := 0
	cfg.MinRateSampleRequests = &zero
	saved, err := svc.UpdateOpsAlertRuntimeSettings(ctx, cfg)
	require.NoError(t, err)
	require.Equal(t, 0, *saved.MinRateSampleRequests, "0 turns the check off and is kept")

	bad := -1
	cfg.MinRateSampleRequests = &bad
	_, err = svc.UpdateOpsAlertRuntimeSettings(ctx, cfg)
	require.Error(t, err)
}

// ---- settings ----

type opsAgentRepoStub struct {
	mu      sync.Mutex
	summary *OpsAgentErrorSummary
	rows    []OpsAgentErrorRow
	filters []OpsAgentErrorFilter
}

func (r *opsAgentRepoStub) ErrorSummary(context.Context, time.Time, time.Time) (*OpsAgentErrorSummary, error) {
	if r.summary == nil {
		return nil, errors.New("db down")
	}
	return r.summary, nil
}
func (r *opsAgentRepoStub) ListErrors(_ context.Context, f OpsAgentErrorFilter) ([]OpsAgentErrorRow, error) {
	r.mu.Lock()
	r.filters = append(r.filters, f)
	r.mu.Unlock()
	return r.rows, nil
}
func (r *opsAgentRepoStub) Timeline(context.Context, time.Time, time.Time, time.Duration) ([]OpsAgentTimelineBucket, error) {
	return []OpsAgentTimelineBucket{{Start: time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC), Success: 10, Errors: 5}}, nil
}
func (r *opsAgentRepoStub) Accounts(context.Context) ([]OpsAgentAccount, error) {
	return []OpsAgentAccount{{ID: 1, Name: "sub2api", Platform: PlatformOpenAI, Type: "apikey", Status: StatusActive, Schedulable: true, BaseHost: "gorustai.com", Groups: []string{"Codex"}}}, nil
}
func (r *opsAgentRepoStub) AlertEvents(context.Context, time.Time, int) ([]OpsAgentAlert, error) {
	return []OpsAgentAlert{}, nil
}

func newOpsAgentForTest(t *testing.T, gw string) (*OpsAgentService, mapSettings, *opsAgentRepoStub, *agentRunsStub) {
	t.Helper()
	gpt := &Group{Name: "GPT", Platform: PlatformOpenAI}
	keys := &learnKeysStub{keys: []APIKey{
		{ID: 7, UserID: 1, Name: "运维", Key: "sk-admin", Status: StatusActive, Group: gpt},
		{ID: 8, UserID: 2, Name: "别人的", Key: "sk-other", Status: StatusActive, Group: gpt},
	}}
	learn, _ := newL2Service(t, gw, LearnSources{Keys: keys})
	settings := mapSettings{SettingKeySiteName: "HiveGPT"}
	repo := &opsAgentRepoStub{summary: &OpsAgentErrorSummary{Success: 9, Errors: 6,
		Groups: []OpsAgentErrorGroup{{StatusCode: 502, Phase: "upstream", Type: "upstream_error", Message: "Upstream service temporarily unavailable", Count: 6, Models: []string{"gpt-5.5"}, Accounts: []string{"#1 sub2api"}}}}}
	runs := &agentRunsStub{}
	svc := NewOpsAgentService(learn, settings, prefixEncryptor{}, repo, runs, nil, nil)
	return svc, settings, repo, runs
}

func TestOpsAgentSettings(t *testing.T) {
	ctx := context.Background()
	svc, settings, _, _ := newOpsAgentForTest(t, "http://127.0.0.1:1")

	st := svc.Settings(ctx)
	require.False(t, st.Enabled)
	require.Equal(t, opsAgentDefaultModel, st.Model)

	_, err := svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, Model: "gpt-5.5"})
	require.ErrorIs(t, err, ErrAssistantKey, "enabled needs a key")
	_, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Model: "gpt-5.5", KeyID: 8})
	require.ErrorIs(t, err, ErrAssistantKey, "another user's key")
	for _, bad := range []string{"gorustai.com", "ftp://x.com", "https://u:p@x.com", "https://x.com/?a=1"} {
		_, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Model: "gpt-5.5", UpstreamURL: bad})
		require.ErrorIs(t, err, ErrOpsAgentSettings, bad)
	}

	st, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, AutoAlert: true, Model: "gpt-5.5", KeyID: 7,
		UpstreamName: "gorustai", UpstreamURL: " https://gorustai.com/ ", UpstreamKey: "admin-secret-1"})
	require.NoError(t, err)
	require.True(t, st.Enabled)
	require.True(t, st.AutoAlert)
	require.Equal(t, "运维", st.KeyName)
	require.Equal(t, "https://gorustai.com", st.UpstreamURL)
	require.True(t, st.UpstreamKeySet)
	require.Empty(t, st.UpstreamKey, "the key is never returned")
	require.NotEqual(t, "admin-secret-1", settings[settingOpsAgentUpKey], "stored encrypted")
	require.NotEmpty(t, settings[settingOpsAgentUpKey])

	// An empty key keeps the stored one; clearing removes it.
	stored := settings[settingOpsAgentUpKey]
	st, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, Model: "gpt-5.5", KeyID: 7, UpstreamURL: "https://gorustai.com"})
	require.NoError(t, err)
	require.True(t, st.UpstreamKeySet)
	require.Equal(t, stored, settings[settingOpsAgentUpKey])
	require.NotNil(t, svc.upstream(ctx))

	st, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, Model: "gpt-5.5", KeyID: 7, UpstreamURL: "https://gorustai.com", ClearUpstreamKey: true})
	require.NoError(t, err)
	require.False(t, st.UpstreamKeySet)
	require.Nil(t, svc.upstream(ctx))
	_, err = svc.TestUpstream(ctx)
	require.ErrorIs(t, err, ErrOpsAgentUpstream)
}

// ---- upstream Sub2API site ----

func fakeUpstream(t *testing.T, key string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"code":401,"message":"Invalid admin API key"}`)
			return
		}
		data := ""
		switch r.URL.Path {
		case "/api/v1/admin/ops/dashboard/overview":
			require.NotEmpty(t, r.URL.Query().Get("start_time"))
			data = `{"success_count":585,"error_count_total":190,"business_limited_count":26,"request_count_total":775,"sla":0.7548,"error_rate":0.2452,"upstream_error_rate":0.21,"upstream_429_count":0}`
		case "/api/v1/admin/ops/errors":
			now := time.Now().UTC()
			data = `{"total":4,"items":[
				{"created_at":"` + now.Add(-2*time.Minute).Format(time.RFC3339) + `","phase":"upstream","status_code":503,"model":"gpt-5.5","message":"Our servers are currently overloaded.","user_id":1,"user_email":"admin@x","account_id":9,"account_name":"outlook ph","group_name":"20x 小队"},
				{"created_at":"` + now.Add(-3*time.Minute).Format(time.RFC3339) + `","phase":"upstream","status_code":503,"model":"gpt-6-sol","message":"Our servers are currently overloaded.","user_id":1,"user_email":"admin@x","account_id":9,"account_name":"outlook ph","group_name":"20x 小队"},
				{"created_at":"` + now.Add(-4*time.Minute).Format(time.RFC3339) + `","phase":"upstream","status_code":502,"model":"gpt-5.5","message":"server_error","account_id":8,"account_name":"201ai"},
				{"created_at":"` + now.Add(-5*time.Minute).Format(time.RFC3339) + `","phase":"request","status_code":403,"message":"Insufficient account balance","user_id":203,"user_email":"u@x"}]}`
		case "/api/v1/admin/usage":
			now := time.Now().UTC()
			data = `{"items":[
				{"created_at":"` + now.Add(-time.Minute).Format(time.RFC3339) + `","account_id":8,"account":{"name":"201ai"}},
				{"created_at":"` + now.Add(-time.Minute).Format(time.RFC3339) + `","account_id":8},
				{"created_at":"` + now.Add(-2*time.Minute).Format(time.RFC3339) + `","account_id":9},
				{"created_at":"` + now.Add(-48*time.Hour).Format(time.RFC3339) + `","account_id":9}]}`
		case "/api/v1/admin/accounts":
			data = `{"total":1,"items":[{"id":9,"name":"outlook ph","platform":"openai","type":"oauth","status":"active","schedulable":false,
				"credentials":{"access_token":"at-SECRET","refresh_token":"rt-SECRET"},"proxy":{"password":"px-SECRET"},
				"groups":[{"id":16,"name":"20x 小队"}],"extra":{"codex_7d_used_percent":30,"session_cookie":"ck-SECRET"}}]}`
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"code":404,"message":"not found"}`)
			return
		}
		_, _ = io.WriteString(w, `{"code":0,"message":"success","data":`+data+`}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestOpsUpstreamClient(t *testing.T) {
	ctx := context.Background()
	srv := fakeUpstream(t, "admin-key")
	up := &opsUpstreamClient{name: "gorustai", baseURL: srv.URL, key: "admin-key", http: srv.Client(), now: time.Now}
	end := time.Now()
	start := end.Add(-time.Hour)

	o, err := up.Overview(ctx, start, end)
	require.NoError(t, err)
	require.EqualValues(t, 585, o["success"])
	require.InDelta(t, 24.52, o["error_rate_pct"], 0.001)

	sum, err := up.ErrorSummary(ctx, start, end)
	require.NoError(t, err)
	b, _ := json.Marshal(sum)
	out := string(b)
	require.Contains(t, out, `"account":"#9 outlook ph","errors":2,"success":1`, "per account: errors and successes merged by ID")
	require.Contains(t, out, `"account":"#8 201ai","errors":1,"success":2`, "a usage row without the name still counts for #8")
	require.Contains(t, out, `"count":2,"key":"20x 小队"`)
	require.NotContains(t, out, "success_note", "the usage rows reached back past the window")

	acc, err := up.Accounts(ctx)
	require.NoError(t, err)
	b, _ = json.Marshal(acc)
	require.Contains(t, string(b), `"groups":["20x 小队"]`)
	require.Contains(t, string(b), `"codex_7d_used_percent":30`)
	require.NotContains(t, string(b), "SECRET", "credentials, proxies and other extras are never passed on")

	list, err := up.ListErrors(ctx, start, end, 9, "", 10)
	require.NoError(t, err)
	require.Len(t, list["rows"], 2)

	bad := &opsUpstreamClient{name: "gorustai", baseURL: srv.URL, key: "wrong", http: srv.Client(), now: time.Now}
	_, err = bad.Overview(ctx, start, end)
	require.ErrorIs(t, err, errOpsUpstreamAuth)
}

// ---- chat and alert analysis ----

func TestOpsAgentChat(t *testing.T) {
	ctx := context.Background()
	up := fakeUpstream(t, "admin-key")
	gw := &agentGateway{replies: []string{
		sseToolCall("c1", "get_error_overview", `{"minutes":30}`),
		sseToolCall("c2", "upstream_error_summary", `{"minutes":30}`),
		sseText("上游 #9 outlook ph 过载。"),
	}}
	srv := gw.serve(t)
	svc, _, _, runs := newOpsAgentForTest(t, srv.URL)

	_, err := svc.Chat(ctx, 1, OpsAgentChatInput{Messages: []LearnMessage{{Role: "user", Content: "为什么报错"}}}, nil, nil)
	require.ErrorIs(t, err, ErrOpsAgentOff)

	_, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, Model: "gpt-5.5", KeyID: 7, UpstreamName: "gorustai", UpstreamURL: up.URL, UpstreamKey: "admin-key"})
	require.NoError(t, err)
	_, err = svc.Chat(ctx, 1, OpsAgentChatInput{Messages: []LearnMessage{{Role: "user", Content: "  "}}}, nil, nil)
	require.ErrorIs(t, err, ErrOpsAgentQuestion)

	var labels []string
	var text strings.Builder
	res, err := svc.Chat(ctx, 1, OpsAgentChatInput{Messages: []LearnMessage{{Role: "user", Content: "最近半小时为什么报错"}}},
		func(d string) error { text.WriteString(d); return nil }, func(l string) error { labels = append(labels, l); return nil })
	require.NoError(t, err)
	require.Equal(t, "gpt-test", res.Model)
	require.Equal(t, "上游 #9 outlook ph 过载。", text.String())
	require.Equal(t, []string{"正在汇总报错", "正在汇总上游站的报错"}, labels)

	first := gw.requests[0]
	require.Equal(t, "gpt-5.5", first["model"])
	sys := first["messages"].([]any)[0].(map[string]any)["content"].(string)
	require.Contains(t, sys, "已配置的上游站：gorustai")
	var names []string
	for _, tool := range first["tools"].([]any) {
		names = append(names, tool.(map[string]any)["function"].(map[string]any)["name"].(string))
	}
	require.Contains(t, names, "list_accounts")
	require.Contains(t, names, "upstream_accounts")
	toolResult := gw.requests[1]["messages"].([]any)[3].(map[string]any)["content"].(string)
	require.Contains(t, toolResult, `"error_rate_pct":40`)
	require.Contains(t, gw.requests[2]["messages"].([]any)[5].(map[string]any)["content"].(string), "#9 outlook ph")

	require.Len(t, runs.runs, 1)
	run := runs.runs[0]
	require.Equal(t, AgentOps, run.Agent)
	require.Equal(t, int64(1), *run.UserID)
	require.Len(t, run.Steps, 2)
	for _, s := range run.Steps {
		require.NotContains(t, s.Result, "admin-key")
	}
}

func TestOpsAgentToolsWithoutUpstream(t *testing.T) {
	svc, _, repo, _ := newOpsAgentForTest(t, "http://127.0.0.1:1")
	tools := svc.tools(nil)
	for _, tool := range tools {
		require.False(t, strings.HasPrefix(tool.Name, "upstream_"), tool.Name)
	}
	out, err := runAssistantTool(t, tools, "list_errors", `{"minutes":99999,"model":"gpt","status_code":502,"limit":5}`)
	require.NoError(t, err)
	require.Contains(t, out, `"rows":[]`)
	f := repo.filters[0]
	require.Equal(t, "gpt", f.Model)
	require.Equal(t, 502, f.StatusCode)
	require.InDelta(t, opsAgentMaxMinutes, f.End.Sub(f.Start).Minutes(), 0.1, "minutes are capped")

	out, err = runAssistantTool(t, tools, "list_accounts", `{}`)
	require.NoError(t, err)
	require.Contains(t, out, `"base_host":"gorustai.com"`)

	repo.summary = nil
	_, err = runAssistantTool(t, tools, "get_error_overview", `{}`)
	require.ErrorIs(t, err, errOpsAgentData)
}

func TestOpsAgentAlertAnalysis(t *testing.T) {
	ctx := context.Background()
	gw := &agentGateway{replies: []string{sseText("成功率低是因为上游过载。"), sseText("第二次")}}
	srv := gw.serve(t)
	svc, _, _, runs := newOpsAgentForTest(t, srv.URL)
	fired := time.Date(2026, 10, 9, 10, 2, 51, 0, time.UTC)

	// Off unless 告警自动分析 is on.
	_, err := svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, Model: "gpt-5.5", KeyID: 7})
	require.NoError(t, err)
	svc.OnAlertFired(nil, &OpsAlertEvent{Title: "P0: 成功率过低", FiredAt: fired})
	svc.autoTimer.Stop()
	svc.flushAlerts()
	require.Empty(t, runs.runs)

	_, err = svc.SaveSettings(ctx, 1, OpsAgentSettings{Enabled: true, AutoAlert: true, Model: "gpt-5.5", KeyID: 7})
	require.NoError(t, err)
	svc.autoLast = time.Time{}
	svc.OnAlertFired(nil, &OpsAlertEvent{Title: "P0: 成功率过低", Description: "success_rate < 95.00 (current 64.29)", FiredAt: fired})
	svc.OnAlertFired(&OpsAlertRule{Description: "整体成功率"}, &OpsAlertEvent{Title: "P1: 错误率过高", FiredAt: fired})
	require.NotNil(t, svc.autoTimer, "alerts close together share one timer")
	svc.autoTimer.Stop()
	svc.flushAlerts()
	require.Len(t, runs.runs, 1, "one analysis for both")
	run := runs.runs[0]
	require.Nil(t, run.UserID)
	require.Contains(t, run.Question, "18:02:51 P0: 成功率过低：success_rate < 95.00 (current 64.29)")
	require.Contains(t, run.Question, "P1: 错误率过高：（规则说明：整体成功率）")
	require.Equal(t, "成功率低是因为上游过载。", run.Answer)

	// Within the cooldown: skipped.
	svc.OnAlertFired(nil, &OpsAlertEvent{Title: "P0: 错误率极高", FiredAt: fired})
	svc.autoTimer.Stop()
	svc.flushAlerts()
	require.Len(t, runs.runs, 1)
	require.Nil(t, svc.autoPending)
}
