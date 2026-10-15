//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClassifyClientUA(t *testing.T) {
	cases := map[string]string{
		"axonhub/1.0":             ClientRelay,
		"CodexPlusPlus/RelayTest": ClientRelay,
		"Go-http-client/2.0":      ClientRelay,
		"fasthttp":                ClientRelay,
		"codex_cli_rs/0.46.0 (Mac OS 15.6; arm64)": ClientCodexCLI,
		"Codex Desktop/0.1.2 (Windows)":            ClientCodexDesktop,
		"claude-cli/2.0.14 (external, cli)":        ClientClaudeCode,
		"CherryStudio/1.6.3":                       ClientCherryStudio,
		"Mozilla/5.0 (Macintosh) Chrome/141.0":     ClientBrowser,
		"OpenAI/Python 1.109.1":                    ClientSDK,
		"":                                         ClientOther,
		"something-else":                           ClientOther,
	}
	for ua, want := range cases {
		require.Equal(t, want, ClassifyClientUA(ua), ua)
	}
}

func TestSummarizeUserAgents(t *testing.T) {
	clients, relay, ids := summarizeUserAgents([]InsightsUARow{
		{UserID: 1, UserAgent: "codex_cli_rs/0.46.0", Requests: 10, CostUSD: 1},
		{UserID: 2, UserAgent: "axonhub/1.0", Requests: 5, CostUSD: 2},
		{UserID: 2, UserAgent: "Go-http-client/2.0", Requests: 7, CostUSD: 1},
		{UserID: 3, UserAgent: "codex_cli_rs/0.47.0", Requests: 1, CostUSD: 0.5},
	})
	require.Equal(t, InsightsClient{Class: ClientRelay, Requests: 12, Users: 1, CostUSD: 3}, clients[0])
	require.Equal(t, InsightsClient{Class: ClientCodexCLI, Requests: 11, Users: 2, CostUSD: 1.5}, clients[1])
	require.Len(t, relay, 1)
	require.Equal(t, int64(2), relay[0].User.ID)
	require.Equal(t, "Go-http-client/2.0", relay[0].UserAgent, "their busiest relay UA")
	require.Equal(t, int64(12), relay[0].Requests)
	require.Equal(t, []int64{2}, ids)
}

func TestScoreUserRisk(t *testing.T) {
	p := &UserInsightsProfile{
		User:    InsightsUserRef{ID: 61, Status: StatusActive},
		Clients: []InsightsClientUA{{UserAgent: "CodexPlusPlus/RelayTest", Class: ClientRelay}},
		Linked: []InsightsLinked{
			{User: InsightsUserRef{ID: 48, Status: StatusDisabled}, Via: []string{"visitor"}},
			{User: InsightsUserRef{ID: 50, Status: StatusDisabled}, Via: []string{"visitor", "ip_prefix"}},
			{User: InsightsUserRef{ID: 77, Status: StatusActive}, Via: []string{"inviter"}},
		},
		Spend: InsightsSpend{TotalUSD: 30},
	}
	score, reasons := scoreUserRisk(p)
	require.Equal(t, 85, score, "two banned twins (40) + 2 linked (10) + relay UA (25) + free spend (10)")
	require.Contains(t, reasons[0], "已封禁")

	clean, reasons := scoreUserRisk(&UserInsightsProfile{User: InsightsUserRef{ID: 1, Status: StatusActive}, Spend: InsightsSpend{HasRealPaying: true}})
	require.Zero(t, clean)
	require.Equal(t, []string{"-20 付过款"}, reasons)
}

type userRiskRepoFake struct {
	UserRiskRepository
	hits       map[string][]RiskHit
	stored     map[string]bool
	calls      []string
	burstArg   int
	relaySince time.Time
}

func (r *userRiskRepoFake) rule(name string) ([]RiskHit, error) {
	r.calls = append(r.calls, name)
	return r.hits[name], nil
}
func (r *userRiskRepoFake) LinkedFlaggedSignups(context.Context, time.Time) ([]RiskHit, error) {
	return r.rule(RiskRuleLinkedSignup)
}
func (r *userRiskRepoFake) TrialDay1Exhausted(context.Context, time.Time) ([]RiskHit, error) {
	return r.rule(RiskRuleTrialDay1)
}
func (r *userRiskRepoFake) TrialSameIPSignups(_ context.Context, _ time.Time, window time.Duration) ([]RiskHit, error) {
	if window != 30*24*time.Hour {
		panic("unexpected window")
	}
	return r.rule(RiskRuleTrialSameIP)
}
func (r *userRiskRepoFake) RelayFreeSpend(_ context.Context, since time.Time, pattern string) ([]RiskHit, error) {
	r.relaySince = since
	if pattern != relayUAPattern {
		panic("unexpected pattern")
	}
	return r.rule(RiskRuleRelayFree)
}
func (r *userRiskRepoFake) BurstSignups(_ context.Context, _ time.Time, threshold int) ([]RiskHit, error) {
	r.burstArg = threshold
	return r.rule(RiskRuleBurstSignups)
}
func (r *userRiskRepoFake) InviteeSharesInviterFingerprint(context.Context, time.Time) ([]RiskHit, error) {
	return r.rule(RiskRuleInviteeFingerprint)
}
func (r *userRiskRepoFake) InsertRiskEvents(_ context.Context, rule string, hits []RiskHit) ([]RiskEvent, error) {
	out := []RiskEvent{}
	for _, h := range hits {
		key := rule + "/" + strconv.FormatInt(h.UserID, 10)
		if r.stored[key] {
			continue
		}
		r.stored[key] = true
		out = append(out, RiskEvent{UserID: h.UserID, Rule: rule, Detail: h.Detail})
	}
	return out, nil
}

func TestRiskMonitor_RunsEnabledRulesAndMailsNewEventsOnce(t *testing.T) {
	repo := &userRiskRepoFake{stored: map[string]bool{}, hits: map[string][]RiskHit{
		RiskRuleLinkedSignup: {{UserID: 61, Detail: "same browser as #48"}},
		RiskRuleRelayFree:    {{UserID: 60, Detail: "axonhub"}},
	}}
	growth := growthForTest(func(s *GrowthSettings) { s.RiskTrialDay1 = false; s.RiskBurstThreshold = 4 })
	svc := NewUserRiskService(repo, nil, growth, nil, nil)
	var mails []string
	svc.mail = func(subject, body string) bool {
		mails = append(mails, subject+"\n"+body)
		return true
	}

	fresh, err := svc.RunMonitor(context.Background())
	require.NoError(t, err)
	require.Len(t, fresh, 2)
	require.NotContains(t, repo.calls, RiskRuleTrialDay1, "turned-off rules don't run")
	require.Equal(t, 4, repo.burstArg)
	require.WithinDuration(t, time.Now().Add(-time.Hour), repo.relaySince, time.Minute, "relay UAs: the last hour only")
	require.Contains(t, repo.calls, RiskRuleTrialSameIP, "shared-IP trials are reported (never restricted)")
	require.Len(t, mails, 1)
	require.Contains(t, mails[0], "[风控] 2 条新事件")
	require.Contains(t, mails[0], "新注册关联到已封禁/静默账号")

	// Same hits next time: already reported, no mail.
	fresh, err = svc.RunMonitor(context.Background())
	require.NoError(t, err)
	require.Empty(t, fresh)
	require.Len(t, mails, 1)
}
