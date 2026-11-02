package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// openAIModelAutoSyncInterval 新模型发布是天级事件，6 小时一轮足够及时。
	openAIModelAutoSyncInterval = 6 * time.Hour
	// openAIModelAutoSyncStartDelay 启动后等价格表加载完再跑第一轮，否则新模型会被误判为「没有价格」。
	openAIModelAutoSyncStartDelay = 3 * time.Minute
	openAIModelAutoSyncTimeout    = 2 * time.Minute
)

// 新模型暂时不能加的原因（写进状态和邮件）。
const (
	openAIModelAutoSyncReasonNoPrice    = "no_price"
	openAIModelAutoSyncReasonNoUpstream = "no_upstream"
)

// OpenAIModelAutoSyncService 周期性把 OpenAI 官方新发布的 Codex 模型自动加进
// OpenAI API Key 账号的模型白名单（credentials.model_mapping），免得每次 OpenAI 出新模型
// 都要手工改账号。
//
// 官方模型列表来自任一可用 OpenAI OAuth 账号拉到的 ChatGPT Codex 模型清单（只取
// visibility=list 的 gpt-* 模型）。只有同时满足下面两条的新模型才会被加进账号：
//   - 本站价格表能确定性地查到它的 token 价格（GetIdentifiedModelPricing，不走模糊匹配）——
//     否则这类请求会按 $0 记账；
//   - 该账号自己的上游模型清单里已经有它——否则请求只会被上游拒绝。
//
// 只加不删，也不改已有映射（比如把下线模型转给新模型仍需人判断）。OAuth 账号与空白名单
// 账号不动：前者的 credentials 会被令牌刷新并发写入，后者本来就放行所有模型。
// 加了模型或出现暂时不能加的新模型时，给运维告警收件人发邮件；同一模型同一原因只提醒一次。
type OpenAIModelAutoSyncService struct {
	accounts openAIModelAutoSyncAccountStore
	persist  func(ctx context.Context, account *Account, credentials map[string]any) error
	catalog  openAIModelAutoSyncCatalog
	pricing  openAIModelAutoSyncPricing
	settings SettingRepository
	notify   func(ctx context.Context, subject, body string)
	now      func() time.Time

	interval   time.Duration
	startDelay time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

type openAIModelAutoSyncAccountStore interface {
	ListByPlatform(ctx context.Context, platform string) ([]Account, error)
}

type openAIModelAutoSyncCatalog interface {
	FetchCodexModelsManifest(ctx context.Context, account *Account, clientVersion, ifNoneMatch string) (*CodexModelsManifest, error)
}

type openAIModelAutoSyncPricing interface {
	GetIdentifiedModelPricing(modelName string) *LiteLLMModelPricing
}

// OpenAIModelAutoSyncState 是一轮同步后保存到设置里的记录。
type OpenAIModelAutoSyncState struct {
	LastRunAt   time.Time                     `json:"last_run_at"`
	LastError   string                        `json:"last_error,omitempty"`
	Official    []string                      `json:"official,omitempty"`
	Added       []OpenAIModelAutoSyncAddition `json:"added,omitempty"`
	Pending     map[string]string             `json:"pending,omitempty"`      // 模型 → 暂时不能加的原因
	NotifiedFor map[string]string             `json:"notified_for,omitempty"` // 已提醒过的 模型 → 原因
}

// OpenAIModelAutoSyncAddition 记录一次自动加进白名单的模型。
type OpenAIModelAutoSyncAddition struct {
	AccountID   int64     `json:"account_id"`
	AccountName string    `json:"account_name"`
	Model       string    `json:"model"`
	At          time.Time `json:"at"`
}

func NewOpenAIModelAutoSyncService(
	accounts openAIModelAutoSyncAccountStore,
	persist func(ctx context.Context, account *Account, credentials map[string]any) error,
	catalog openAIModelAutoSyncCatalog,
	pricing openAIModelAutoSyncPricing,
	settings SettingRepository,
	notify func(ctx context.Context, subject, body string),
	interval time.Duration,
) *OpenAIModelAutoSyncService {
	return &OpenAIModelAutoSyncService{
		accounts:   accounts,
		persist:    persist,
		catalog:    catalog,
		pricing:    pricing,
		settings:   settings,
		notify:     notify,
		now:        time.Now,
		interval:   interval,
		startDelay: openAIModelAutoSyncStartDelay,
		stopCh:     make(chan struct{}),
	}
}

func (s *OpenAIModelAutoSyncService) Start() {
	if s == nil || s.accounts == nil || s.persist == nil || s.catalog == nil || s.pricing == nil || s.settings == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		select {
		case <-time.After(s.startDelay):
		case <-s.stopCh:
			return
		}
		s.runScheduled()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runScheduled()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *OpenAIModelAutoSyncService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.wg.Wait()
}

func (s *OpenAIModelAutoSyncService) runScheduled() {
	ctx, cancel := context.WithTimeout(context.Background(), openAIModelAutoSyncTimeout)
	defer cancel()
	if !s.enabled(ctx) {
		return
	}
	if _, err := s.RunOnce(ctx); err != nil {
		slog.Warn("openai_model_auto_sync_failed", "error", err)
	}
}

func (s *OpenAIModelAutoSyncService) enabled(ctx context.Context) bool {
	value, err := s.settings.GetValue(ctx, SettingKeyOpenAIModelAutoSyncEnabled)
	if err != nil || strings.TrimSpace(value) == "" {
		return true
	}
	return strings.TrimSpace(value) == "true"
}

// RunOnce 执行一轮同步并返回保存的状态。
func (s *OpenAIModelAutoSyncService) RunOnce(ctx context.Context) (*OpenAIModelAutoSyncState, error) {
	state := s.loadState(ctx)
	state.LastRunAt = s.now()
	state.LastError = ""
	state.Added = nil

	accounts, err := s.accounts.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return s.fail(ctx, state, fmt.Errorf("list openai accounts: %w", err))
	}

	official, err := s.officialModels(ctx, accounts)
	if err != nil {
		return s.fail(ctx, state, err)
	}
	state.Official = official

	pending := map[string]string{}
	for i := range accounts {
		account := &accounts[i]
		if !openAIModelAutoSyncTarget(account) {
			continue
		}
		mapping := account.GetModelMapping()
		var missing []string
		for _, model := range official {
			if !mappingSupportsRequestedModel(mapping, model) {
				missing = append(missing, model)
			}
		}
		if len(missing) == 0 {
			continue
		}

		var upstream map[string]struct{}
		upstreamErr := error(nil)
		var toAdd []string
		for _, model := range missing {
			if !s.hasTokenPrice(model) {
				pending[model] = openAIModelAutoSyncReasonNoPrice
				continue
			}
			if upstream == nil && upstreamErr == nil {
				upstream, upstreamErr = s.manifestModels(ctx, account, false)
				if upstreamErr != nil {
					slog.Warn("openai_model_auto_sync_upstream_catalog_failed", "account_id", account.ID, "error", upstreamErr)
				}
			}
			if upstreamErr != nil {
				continue // 上游清单拉不到：这一轮不碰这个账号，下一轮再试
			}
			if _, ok := upstream[model]; !ok {
				pending[model] = openAIModelAutoSyncReasonNoUpstream
				continue
			}
			toAdd = append(toAdd, model)
		}
		if len(toAdd) == 0 {
			continue
		}

		credentials := make(map[string]any, len(account.Credentials)+1)
		for k, v := range account.Credentials {
			credentials[k] = v
		}
		merged := make(map[string]any, len(mapping)+len(toAdd))
		for k, v := range mapping {
			merged[k] = v
		}
		for _, model := range toAdd {
			merged[model] = model
		}
		credentials["model_mapping"] = merged
		if err := s.persist(ctx, account, credentials); err != nil {
			slog.Warn("openai_model_auto_sync_update_failed", "account_id", account.ID, "models", toAdd, "error", err)
			continue
		}
		for _, model := range toAdd {
			state.Added = append(state.Added, OpenAIModelAutoSyncAddition{
				AccountID: account.ID, AccountName: account.Name, Model: model, At: state.LastRunAt,
			})
			// 加上后，这个模型此前的「暂时不能加」提醒作废；以后再出问题会重新提醒。
			delete(state.NotifiedFor, model)
		}
		slog.Info("openai_model_auto_sync_added", "account_id", account.ID, "account_name", account.Name, "models", toAdd)
	}

	// 某个模型已经加进了某个账号，就不再算「暂时不能加」。
	for _, added := range state.Added {
		delete(pending, added.Model)
	}
	state.Pending = pending

	var newlyPending []string
	if state.NotifiedFor == nil {
		state.NotifiedFor = map[string]string{}
	}
	for model, reason := range pending {
		if state.NotifiedFor[model] != reason {
			newlyPending = append(newlyPending, model)
		}
	}
	sort.Strings(newlyPending)

	if len(state.Added) > 0 || len(newlyPending) > 0 {
		// 先记下已提醒，再发邮件：邮件服务出错时不会每轮都重发。
		for _, model := range newlyPending {
			state.NotifiedFor[model] = pending[model]
		}
		s.saveState(ctx, state)
		if s.notify != nil {
			subject, body := openAIModelAutoSyncMail(state.Added, newlyPending, pending)
			s.notify(ctx, subject, body)
		}
		return state, nil
	}
	s.saveState(ctx, state)
	return state, nil
}

func (s *OpenAIModelAutoSyncService) fail(ctx context.Context, state *OpenAIModelAutoSyncState, err error) (*OpenAIModelAutoSyncState, error) {
	state.LastError = err.Error()
	s.saveState(ctx, state)
	return state, err
}

// officialModels 依次尝试可用的 OpenAI OAuth 账号，拿到第一份成功的 ChatGPT Codex 模型清单。
func (s *OpenAIModelAutoSyncService) officialModels(ctx context.Context, accounts []Account) ([]string, error) {
	var lastErr error
	tried := 0
	for i := range accounts {
		account := &accounts[i]
		if !account.IsOpenAIOAuth() || !account.IsActive() {
			continue
		}
		tried++
		models, err := s.manifestModels(ctx, account, true)
		if err != nil {
			lastErr = err
			continue
		}
		if len(models) == 0 {
			lastErr = fmt.Errorf("account %d returned an empty Codex model list", account.ID)
			continue
		}
		out := make([]string, 0, len(models))
		for model := range models {
			out = append(out, model)
		}
		sort.Strings(out)
		return out, nil
	}
	if tried == 0 {
		return nil, fmt.Errorf("no active OpenAI OAuth account to read the official Codex model list from")
	}
	return nil, fmt.Errorf("fetch official Codex model list: %w", lastErr)
}

// manifestModels 拉一个账号的 Codex 模型清单。official=true 时只取 Codex 选单里列出的 gpt-* 文本模型。
func (s *OpenAIModelAutoSyncService) manifestModels(ctx context.Context, account *Account, official bool) (map[string]struct{}, error) {
	manifest, err := s.catalog.FetchCodexModelsManifest(ctx, account, CodexCanonicalClientVersion(), "")
	if err != nil {
		return nil, err
	}
	if manifest == nil || len(manifest.Body) == 0 {
		return nil, fmt.Errorf("account %d returned no Codex model list", account.ID)
	}
	var envelope struct {
		Models []struct {
			Slug       string `json:"slug"`
			Visibility string `json:"visibility"`
		} `json:"models"`
	}
	if err := json.Unmarshal(manifest.Body, &envelope); err != nil {
		return nil, fmt.Errorf("decode Codex model list of account %d: %w", account.ID, err)
	}
	models := make(map[string]struct{}, len(envelope.Models))
	for _, m := range envelope.Models {
		slug := strings.ToLower(strings.TrimSpace(m.Slug))
		if slug == "" {
			continue
		}
		if official {
			if !strings.EqualFold(strings.TrimSpace(m.Visibility), "list") ||
				!strings.HasPrefix(slug, "gpt-") || isCodexDedicatedMediaModel(slug) {
				continue
			}
		}
		models[slug] = struct{}{}
	}
	return models, nil
}

func (s *OpenAIModelAutoSyncService) hasTokenPrice(model string) bool {
	pricing := s.pricing.GetIdentifiedModelPricing(model)
	return pricing != nil && !pricing.TokenPricingAbsent
}

// openAIModelAutoSyncTarget 只处理有白名单、会被白名单拦住的 OpenAI API Key 账号。
func openAIModelAutoSyncTarget(account *Account) bool {
	if account == nil || !account.IsOpenAIApiKey() || !account.IsActive() || account.IsCredentialShadow() {
		return false
	}
	if account.IsOpenAIPassthroughEnabled() {
		return false
	}
	return len(account.GetModelMapping()) > 0
}

func (s *OpenAIModelAutoSyncService) loadState(ctx context.Context) *OpenAIModelAutoSyncState {
	state := &OpenAIModelAutoSyncState{}
	if raw, err := s.settings.GetValue(ctx, SettingKeyOpenAIModelAutoSyncState); err == nil && strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), state); err != nil {
			state = &OpenAIModelAutoSyncState{}
		}
	}
	if state.NotifiedFor == nil {
		state.NotifiedFor = map[string]string{}
	}
	return state
}

func (s *OpenAIModelAutoSyncService) saveState(ctx context.Context, state *OpenAIModelAutoSyncState) {
	raw, err := json.Marshal(state)
	if err != nil {
		return
	}
	if err := s.settings.Set(ctx, SettingKeyOpenAIModelAutoSyncState, string(raw)); err != nil {
		slog.Warn("openai_model_auto_sync_state_save_failed", "error", err)
	}
}

func openAIModelAutoSyncReasonText(reason string) string {
	switch reason {
	case openAIModelAutoSyncReasonNoPrice:
		return "本站价格表还没有它的价格（加进去会按 $0 记账），价格同步到之后会自动加上"
	case openAIModelAutoSyncReasonNoUpstream:
		return "账号的上游（如 gorustai）还没开通这个模型，上游开通后会自动加上"
	default:
		return reason
	}
}

func openAIModelAutoSyncMail(added []OpenAIModelAutoSyncAddition, newlyPending []string, pending map[string]string) (string, string) {
	var addedModels []string
	seen := map[string]struct{}{}
	for _, a := range added {
		if _, ok := seen[a.Model]; !ok {
			seen[a.Model] = struct{}{}
			addedModels = append(addedModels, a.Model)
		}
	}
	sort.Strings(addedModels)

	var subject string
	switch {
	case len(addedModels) > 0:
		subject = "[模型同步] 已自动开通 OpenAI 新模型 " + strings.Join(addedModels, "、")
	default:
		subject = "[模型同步] OpenAI 新模型暂时没能开通 " + strings.Join(newlyPending, "、")
	}

	var b strings.Builder
	if len(added) > 0 {
		_, _ = b.WriteString("<p>已自动加进账号的模型白名单，用户现在就能调用：</p><ul>")
		for _, a := range added {
			fmt.Fprintf(&b, "<li>%s → 账号 #%d %s</li>", html.EscapeString(a.Model), a.AccountID, html.EscapeString(a.AccountName))
		}
		_, _ = b.WriteString("</ul>")
	}
	if len(newlyPending) > 0 {
		_, _ = b.WriteString("<p>OpenAI 官方已经发布、但暂时没加的模型：</p><ul>")
		for _, model := range newlyPending {
			fmt.Fprintf(&b, "<li>%s：%s</li>", html.EscapeString(model), html.EscapeString(openAIModelAutoSyncReasonText(pending[model])))
		}
		_, _ = b.WriteString("</ul>")
	}
	_, _ = b.WriteString("<p>官方模型列表来自 ChatGPT Codex（OpenAI OAuth 账号拉取），每 6 小时检查一次；只加不删，下线或改名的模型仍需手工处理。")
	_, _ = b.WriteString("关闭自动同步：设置项 openai_model_auto_sync_enabled 设为 false。</p>")
	return subject, b.String()
}
