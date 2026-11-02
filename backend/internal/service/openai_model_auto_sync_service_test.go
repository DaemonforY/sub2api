package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type modelAutoSyncAccounts struct{ accounts []Account }

func (f *modelAutoSyncAccounts) ListByPlatform(_ context.Context, platform string) ([]Account, error) {
	out := make([]Account, 0, len(f.accounts))
	for _, a := range f.accounts {
		if a.Platform == platform {
			out = append(out, a)
		}
	}
	return out, nil
}

type modelAutoSyncCatalog struct {
	byAccount map[int64][]map[string]string
	errFor    map[int64]error
	calls     map[int64]int
}

func (f *modelAutoSyncCatalog) FetchCodexModelsManifest(_ context.Context, account *Account, _, _ string) (*CodexModelsManifest, error) {
	if f.calls == nil {
		f.calls = map[int64]int{}
	}
	f.calls[account.ID]++
	if err := f.errFor[account.ID]; err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{"models": f.byAccount[account.ID]})
	return &CodexModelsManifest{Body: body}, nil
}

type modelAutoSyncPricing struct{ priced map[string]bool }

func (f *modelAutoSyncPricing) GetIdentifiedModelPricing(model string) *LiteLLMModelPricing {
	if f.priced[model] {
		return &LiteLLMModelPricing{InputCostPerToken: 1e-6, OutputCostPerToken: 5e-6}
	}
	return nil
}

type modelAutoSyncSettings struct{ values map[string]string }

func (f *modelAutoSyncSettings) Get(_ context.Context, key string) (*Setting, error) {
	return &Setting{Key: key, Value: f.values[key]}, nil
}
func (f *modelAutoSyncSettings) GetValue(_ context.Context, key string) (string, error) {
	return f.values[key], nil
}
func (f *modelAutoSyncSettings) Set(_ context.Context, key, value string) error {
	f.values[key] = value
	return nil
}
func (f *modelAutoSyncSettings) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (f *modelAutoSyncSettings) SetMultiple(context.Context, map[string]string) error { return nil }
func (f *modelAutoSyncSettings) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (f *modelAutoSyncSettings) Delete(context.Context, string) error                 { return nil }

type modelAutoSyncMail struct{ subject, body string }

type modelAutoSyncFixture struct {
	svc      *OpenAIModelAutoSyncService
	accounts *modelAutoSyncAccounts
	catalog  *modelAutoSyncCatalog
	pricing  *modelAutoSyncPricing
	settings *modelAutoSyncSettings
	persists map[int64][]map[string]any
	mails    []modelAutoSyncMail
}

func listed(slug string) map[string]string {
	return map[string]string{"slug": slug, "visibility": "list"}
}

func newModelAutoSyncFixture() *modelAutoSyncFixture {
	f := &modelAutoSyncFixture{persists: map[int64][]map[string]any{}}
	f.accounts = &modelAutoSyncAccounts{accounts: []Account{
		{ID: 10, Name: "codex 1x", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive,
			Credentials: map[string]any{"access_token": "tok"}},
		{ID: 1, Name: "sub2api", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"api_key": "sk-up", "base_url": "https://up.example",
				"model_mapping": map[string]any{"gpt-6.1-sol": "gpt-6.1-sol", "gpt-5.4-mini": "gpt-5.6-luna"}}},
		// 空白名单（放行所有）、透传、停用、其他平台的账号都不该被改。
		{ID: 3, Name: "open", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"api_key": "sk-3"}},
		{ID: 4, Name: "passthrough", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"api_key": "sk-4", "model_mapping": map[string]any{"gpt-6.1-sol": "gpt-6.1-sol"}},
			Extra:       map[string]any{"openai_passthrough": true}},
		{ID: 5, Name: "disabled", Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusDisabled,
			Credentials: map[string]any{"api_key": "sk-5", "model_mapping": map[string]any{"gpt-6.1-sol": "gpt-6.1-sol"}}},
	}}
	f.catalog = &modelAutoSyncCatalog{byAccount: map[int64][]map[string]string{
		10: {
			listed("gpt-6.1-sol"), listed("gpt-6.1-terra"), listed("gpt-7-nova"), listed("gpt-7-mini"),
			{"slug": "codex-auto-review", "visibility": "list"},
			{"slug": "gpt-6-hidden", "visibility": "hide"},
			listed("gpt-image-2"),
		},
		1: {listed("gpt-6.1-sol"), listed("gpt-6.1-terra"), listed("gpt-7-nova")},
	}}
	f.pricing = &modelAutoSyncPricing{priced: map[string]bool{"gpt-6.1-sol": true, "gpt-6.1-terra": true, "gpt-7-mini": true}}
	f.settings = &modelAutoSyncSettings{values: map[string]string{}}
	persist := func(_ context.Context, account *Account, credentials map[string]any) error {
		f.persists[account.ID] = append(f.persists[account.ID], credentials)
		// 像真实仓库一样回写，下一轮能看到新白名单。
		for i := range f.accounts.accounts {
			if f.accounts.accounts[i].ID == account.ID {
				f.accounts.accounts[i].Credentials = credentials
			}
		}
		return nil
	}
	notify := func(_ context.Context, subject, body string) {
		f.mails = append(f.mails, modelAutoSyncMail{subject, body})
	}
	f.svc = NewOpenAIModelAutoSyncService(f.accounts, persist, f.catalog, f.pricing, f.settings, notify, time.Hour)
	f.svc.now = func() time.Time { return time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC) }
	return f
}

func TestOpenAIModelAutoSync_AddsPricedUpstreamModelsAndRemembersPending(t *testing.T) {
	f := newModelAutoSyncFixture()

	state, err := f.svc.RunOnce(context.Background())
	require.NoError(t, err)

	// 官方清单只取 visibility=list 的 gpt-* 文本模型。
	require.Equal(t, []string{"gpt-6.1-sol", "gpt-6.1-terra", "gpt-7-mini", "gpt-7-nova"}, state.Official)

	// 只改账号 1：有价格且上游已开通的 gpt-6.1-terra 加进去，原有映射（含改名映射）保留。
	require.Len(t, f.persists, 1)
	require.Len(t, f.persists[1], 1)
	mapping, ok := f.persists[1][0]["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-6.1-terra", mapping["gpt-6.1-terra"])
	require.Equal(t, "gpt-5.6-luna", mapping["gpt-5.4-mini"])
	require.Equal(t, "gpt-6.1-sol", mapping["gpt-6.1-sol"])
	require.Len(t, mapping, 3)
	require.Equal(t, "sk-up", f.persists[1][0]["api_key"], "other credentials survive")

	require.Equal(t, map[string]string{
		"gpt-7-nova": openAIModelAutoSyncReasonNoPrice,    // 上游有，但本站没价格：不能按 $0 放行
		"gpt-7-mini": openAIModelAutoSyncReasonNoUpstream, // 有价格，但上游没开通
	}, state.Pending)
	require.Len(t, state.Added, 1)
	require.Equal(t, "gpt-6.1-terra", state.Added[0].Model)

	require.Len(t, f.mails, 1)
	require.Contains(t, f.mails[0].subject, "gpt-6.1-terra")
	require.Contains(t, f.mails[0].body, "gpt-7-nova")
	require.Contains(t, f.mails[0].body, "价格")
	require.Contains(t, f.mails[0].body, "gpt-7-mini")

	// 第二轮：没有新变化，不写账号、不发邮件（同一原因只提醒一次）。
	_, err = f.svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Len(t, f.persists[1], 1)
	require.Len(t, f.mails, 1)

	// 价格同步到了、上游也开通了 gpt-7-mini：第三轮自动加上并通知。
	f.pricing.priced["gpt-7-nova"] = true
	f.catalog.byAccount[1] = append(f.catalog.byAccount[1], listed("gpt-7-mini"))
	state, err = f.svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Len(t, f.persists[1], 2)
	mapping, ok = f.persists[1][1]["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-7-nova", mapping["gpt-7-nova"])
	require.Equal(t, "gpt-7-mini", mapping["gpt-7-mini"])
	require.Empty(t, state.Pending)
	require.Empty(t, state.NotifiedFor)
	require.Len(t, f.mails, 2)
	require.Contains(t, f.mails[1].subject, "gpt-7-mini")
	require.Contains(t, f.mails[1].subject, "gpt-7-nova")

	// 状态保存在设置里。
	var saved OpenAIModelAutoSyncState
	require.NoError(t, json.Unmarshal([]byte(f.settings.values[SettingKeyOpenAIModelAutoSyncState]), &saved))
	require.Len(t, saved.Added, 2)
}

func TestOpenAIModelAutoSync_SkipsAccountWhenUpstreamCatalogFails(t *testing.T) {
	f := newModelAutoSyncFixture()
	f.catalog.errFor = map[int64]error{1: errors.New("upstream down")}

	state, err := f.svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Empty(t, f.persists)
	require.Empty(t, state.Added)
	// 拉不到上游清单时不把有价格的模型误报成「上游没开通」，只提醒没价格的。
	require.Equal(t, map[string]string{"gpt-7-nova": openAIModelAutoSyncReasonNoPrice}, state.Pending)
}

func TestOpenAIModelAutoSync_NeedsAnOAuthAccountForTheOfficialList(t *testing.T) {
	f := newModelAutoSyncFixture()
	f.accounts.accounts = f.accounts.accounts[1:] // 去掉 OAuth 账号

	state, err := f.svc.RunOnce(context.Background())
	require.Error(t, err)
	require.Contains(t, state.LastError, "OAuth")
	require.Empty(t, f.persists)
	require.Empty(t, f.mails)
}

func TestOpenAIModelAutoSync_DisabledBySetting(t *testing.T) {
	f := newModelAutoSyncFixture()
	f.settings.values[SettingKeyOpenAIModelAutoSyncEnabled] = "false"
	f.svc.runScheduled()
	require.Empty(t, f.catalog.calls)
	require.Empty(t, f.persists)

	f.settings.values[SettingKeyOpenAIModelAutoSyncEnabled] = ""
	require.True(t, f.svc.enabled(context.Background()), "on by default")
}

// 走真实的 FetchCodexModelsManifest：OAuth 账号从（模拟的）ChatGPT 拉官方清单，
// API Key 账号从（模拟的）上游 sub2api 拉它自己的清单。
func TestOpenAIModelAutoSync_WithRealCatalogFetch(t *testing.T) {
	official := `{"models":[
		{"slug":"gpt-6.1-sol","display_name":"GPT-6.1-Sol","visibility":"list","priority":1,"supported_in_api":true},
		{"slug":"gpt-6.1-terra","display_name":"GPT-6.1-Terra","visibility":"list","priority":3,"supported_in_api":true},
		{"slug":"gpt-6-astra","display_name":"GPT-6-Astra","visibility":"list","priority":2,"supported_in_api":true},
		{"slug":"codex-auto-review","display_name":"Codex Auto Review","visibility":"hide","priority":50}
	]}`
	chatgpt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-access-token", r.Header.Get("Authorization"))
		require.NotEmpty(t, r.URL.Query().Get("client_version"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(official))
	}))
	defer chatgpt.Close()
	original := chatgptCodexModelsURL
	chatgptCodexModelsURL = chatgpt.URL
	defer func() { chatgptCodexModelsURL = original }()

	var upstreamURL string
	upstream := &codexModelsHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		upstreamURL = req.URL.String()
		body := `{"models":[{"slug":"gpt-6.1-sol","visibility":"list","priority":50},{"slug":"gpt-6.1-terra","visibility":"list","priority":50}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	gateway := newCodexModelsAPIKeyTestService(upstream)

	oauth := newCodexModelsTestAccount()
	oauth.ID, oauth.Status = 10, StatusActive
	apiKey := newCodexModelsAPIKeyTestAccount("https://gorustai.example")
	apiKey.ID, apiKey.Name, apiKey.Status = 1, "sub2api", StatusActive
	apiKey.Credentials["model_mapping"] = map[string]any{"gpt-6.1-sol": "gpt-6.1-sol"}
	accounts := &modelAutoSyncAccounts{accounts: []Account{*oauth, *apiKey}}

	var persisted map[string]any
	persist := func(_ context.Context, _ *Account, credentials map[string]any) error {
		persisted = credentials
		return nil
	}
	pricing := &modelAutoSyncPricing{priced: map[string]bool{"gpt-6.1-sol": true, "gpt-6.1-terra": true, "gpt-6-astra": true}}
	settings := &modelAutoSyncSettings{values: map[string]string{}}
	var mails []modelAutoSyncMail
	svc := NewOpenAIModelAutoSyncService(accounts, persist, gateway, pricing, settings,
		func(_ context.Context, subject, body string) { mails = append(mails, modelAutoSyncMail{subject, body}) }, time.Hour)

	state, err := svc.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-6-astra", "gpt-6.1-sol", "gpt-6.1-terra"}, state.Official)
	require.Contains(t, upstreamURL, "https://gorustai.example/v1/models?client_version=")
	require.NotNil(t, persisted)
	mapping, ok := persisted["model_mapping"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "gpt-6.1-terra", mapping["gpt-6.1-terra"])
	require.NotContains(t, mapping, "gpt-6-astra", "upstream does not list it yet")
	require.Equal(t, map[string]string{"gpt-6-astra": openAIModelAutoSyncReasonNoUpstream}, state.Pending)
	require.Len(t, mails, 1)
	require.Contains(t, mails[0].subject, "gpt-6.1-terra")
	require.Contains(t, mails[0].body, "gpt-6-astra")
}
