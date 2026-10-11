package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 容量告警: every opsCapacityInterval the configured upstream site's Codex OAuth accounts are read
// (运维助手 → 上游站, the same admin API the assistant uses) and the alert recipients get one mail
// per account per 7-day window when the window is running out early — so a blogger's traffic spike
// shows up before users start seeing "no available account". On unless 容量告警 is switched off.

const (
	settingOpsAgentCapacityAlert = "ops_agent_capacity_alert" // "false" = off
	settingOpsCapacityAlerted    = "ops_capacity_alerted"     // JSON {account id: window reset} already mailed
	opsCapacityInterval          = 30 * time.Minute
	opsCapacityWindow            = 7 * 24 * time.Hour
	opsCapacityUsedPct           = 80.0 // used this much …
	opsCapacityMinLeft           = 48 * time.Hour
	opsCapacityMinElapsed        = 48 * time.Hour // … or projected to run out, once the pace means something
)

// OpsCapacityAccount is one upstream Codex account's 7-day window.
type OpsCapacityAccount struct {
	ID           int64
	Name         string
	UsedPct      float64
	ResetAt      time.Time
	ProjectedPct float64 // at reset, at the pace so far; 0 before opsCapacityMinElapsed
	Reason       string  // why it is alerting; empty when fine
}

// evaluateCapacity fills ProjectedPct and Reason.
func evaluateCapacity(a *OpsCapacityAccount, now time.Time) {
	left := a.ResetAt.Sub(now)
	if a.ResetAt.IsZero() || left <= 0 || left > opsCapacityWindow {
		return
	}
	elapsed := opsCapacityWindow - left
	if elapsed >= opsCapacityMinElapsed {
		a.ProjectedPct = math.Round(a.UsedPct / (float64(elapsed) / float64(opsCapacityWindow)))
	}
	switch {
	case a.UsedPct >= opsCapacityUsedPct && left >= opsCapacityMinLeft:
		a.Reason = fmt.Sprintf("已用 %.0f%%，离重置还有 %.1f 天", a.UsedPct, left.Hours()/24)
	case a.ProjectedPct >= 100:
		a.Reason = fmt.Sprintf("按目前速度，重置前会用到约 %.0f%%（现在 %.0f%%，离重置 %.1f 天）", a.ProjectedPct, a.UsedPct, left.Hours()/24)
	}
}

// capacityAccounts turns the upstream's account list into the Codex accounts with a 7-day window.
func capacityAccounts(rows []map[string]any) []OpsCapacityAccount {
	out := []OpsCapacityAccount{}
	for _, r := range rows {
		if s, _ := r["status"].(string); s != "" && s != "active" {
			continue
		}
		used, ok := opsNumber(r["codex_7d_used_percent"])
		if !ok {
			continue
		}
		reset, _ := r["codex_7d_reset_at"].(string)
		at, err := time.Parse(time.RFC3339, reset)
		if err != nil {
			continue
		}
		id, _ := opsNumber(r["id"])
		name, _ := r["name"].(string)
		out = append(out, OpsCapacityAccount{ID: int64(id), Name: name, UsedPct: used, ResetAt: at})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func opsNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

var opsCapacityOnce sync.Once

// startCapacityAlerts runs the check in the background for the life of the process.
func (s *OpsAgentService) startCapacityAlerts() {
	opsCapacityOnce.Do(func() {
		go func() {
			t := time.NewTicker(opsCapacityInterval)
			defer t.Stop()
			for range t.C {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				if err := s.CheckCapacity(ctx); err != nil {
					logger.LegacyPrintf("service.ops_agent", "[OpsCapacity] check failed: %v", err)
				}
				cancel()
			}
		}()
	})
}

// CheckCapacity reads the upstream accounts once and mails the accounts newly running out. Quiet
// (nil) when the upstream isn't configured or 容量告警 is off.
func (s *OpsAgentService) CheckCapacity(ctx context.Context) error {
	if s.settings == nil {
		return nil
	}
	v, err := s.settings.GetMultiple(ctx, []string{settingOpsAgentCapacityAlert, settingOpsCapacityAlerted})
	if err != nil || v[settingOpsAgentCapacityAlert] == "false" {
		return err
	}
	up := s.upstream(ctx)
	if up == nil {
		return nil
	}
	res, err := up.Accounts(ctx)
	if err != nil {
		return err
	}
	rows, _ := res["accounts"].([]map[string]any)
	accounts := capacityAccounts(rows)
	now := s.now()
	alerted := map[string]string{}
	_ = json.Unmarshal([]byte(v[settingOpsCapacityAlerted]), &alerted)
	var fresh []OpsCapacityAccount
	for i := range accounts {
		a := &accounts[i]
		evaluateCapacity(a, now)
		window := a.ResetAt.UTC().Format(time.RFC3339)
		if a.Reason != "" && alerted[strconv.FormatInt(a.ID, 10)] != window {
			fresh = append(fresh, *a)
			alerted[strconv.FormatInt(a.ID, 10)] = window
		}
	}
	if len(fresh) == 0 {
		return nil
	}
	if !s.mailCapacity(ctx, up.name, fresh, accounts) {
		return nil // no recipients: try again next time
	}
	b, _ := json.Marshal(alerted)
	return s.settings.SetMultiple(ctx, map[string]string{settingOpsCapacityAlerted: string(b)})
}

// mailCapacity reports whether a mail went out.
func (s *OpsAgentService) mailCapacity(ctx context.Context, upstream string, fresh, all []OpsCapacityAccount) bool {
	if s.capacityMail != nil {
		s.capacityMail(capacityMailSubject(upstream, fresh), capacityMailBody(upstream, fresh, all))
		return true
	}
	if s.email == nil || s.ops == nil {
		return false
	}
	cfg, err := s.ops.GetEmailNotificationConfig(ctx)
	if err != nil || cfg == nil || !cfg.Alert.Enabled || len(cfg.Alert.Recipients) == 0 {
		return false
	}
	subject, body := capacityMailSubject(upstream, fresh), capacityMailBody(upstream, fresh, all)
	sent := false
	for _, to := range cfg.Alert.Recipients {
		if addr := strings.TrimSpace(to); addr != "" {
			if err := s.email.SendEmail(ctx, addr, subject, body); err != nil {
				logger.LegacyPrintf("service.ops_agent", "[OpsCapacity] mail to %s failed: %v", addr, err)
				continue
			}
			sent = true
		}
	}
	return sent
}

func capacityMailSubject(upstream string, fresh []OpsCapacityAccount) string {
	names := make([]string, 0, len(fresh))
	for _, a := range fresh {
		names = append(names, a.Name)
	}
	return fmt.Sprintf("[容量告警] 上游 %s 的账号 7 天额度快用完：%s", upstream, strings.Join(names, "、"))
}

func capacityMailBody(upstream string, fresh, all []OpsCapacityAccount) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<h3>上游 %s 的 Codex 账号 7 天额度快用完</h3><ul>`, html.EscapeString(upstream))
	for _, a := range fresh {
		fmt.Fprintf(&b, "<li><b>%s</b>：%s</li>", html.EscapeString(a.Name), html.EscapeString(a.Reason))
	}
	_, _ = b.WriteString(`</ul><table border="1" cellpadding="6" style="border-collapse:collapse"><tr><th>账号</th><th>已用</th><th>重置时间</th><th>按速度预计</th></tr>`)
	for _, a := range all {
		proj := "—"
		if a.ProjectedPct > 0 {
			proj = fmt.Sprintf("%.0f%%", a.ProjectedPct)
		}
		fmt.Fprintf(&b, "<tr><td>%s</td><td>%.0f%%</td><td>%s</td><td>%s</td></tr>", html.EscapeString(a.Name), a.UsedPct,
			a.ResetAt.In(learnDayZone).Format("01-02 15:04"), proj)
	}
	_, _ = b.WriteString(`</table><p>可以做的：加一个上游账号；暂停正在推广的博主优惠码（「优惠码」里停用）；错开下一位博主的发布时间。</p>` +
		`<p style="color:#888;font-size:12px">每个账号每个 7 天窗口最多提醒一次。在后台「运维助手 → 设置」里可以关闭容量告警。</p>`)
	return b.String()
}
