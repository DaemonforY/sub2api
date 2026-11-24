package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// Shape of the failed event account 10 received on 2026-11-24.
const openAIOverloadFailedEventFixture = `{"type":"response.failed","response":{"status":"failed","error":{"code":"server_error","headers":{"retry-after":"374","retry-after-ms":"374000"},"message":"Our servers are currently overloaded. Please try again later."}}}`

func TestOpenAIOverloadShedTrackerNeedsARepeat(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	var tracker openAIOverloadShedTracker

	tripped, _ := tracker.record(10, base, 0)
	require.False(t, tripped, "one shed is a blip")
	require.False(t, tracker.isPenalized(10, base))

	tripped, _ = tracker.record(10, base.Add(time.Second), 0)
	require.False(t, tripped, "sheds within the collapse interval are one observation")

	tripped, until := tracker.record(10, base.Add(90*time.Second), 0)
	require.True(t, tripped)
	require.Equal(t, base.Add(90*time.Second+openAIOverloadPenaltyDefault), until)
	require.True(t, tracker.isPenalized(10, until.Add(-time.Nanosecond)))
	require.False(t, tracker.isPenalized(10, until))
	require.False(t, tracker.isPenalized(11, base.Add(90*time.Second)), "other accounts are unaffected")
	require.Equal(t, 1, tracker.activePenaltyCount(base.Add(90*time.Second)))
	require.Equal(t, 0, tracker.activePenaltyCount(until))
}

func TestOpenAIOverloadShedTrackerWindowExpires(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	var tracker openAIOverloadShedTracker

	tripped, _ := tracker.record(10, base, 0)
	require.False(t, tripped)
	tripped, _ = tracker.record(10, base.Add(openAIOverloadShedWindow+time.Second), 0)
	require.False(t, tripped, "sheds further apart than the window do not add up")
}

func TestOpenAIOverloadShedTrackerKeepsLongerPenalty(t *testing.T) {
	base := time.Unix(1_800_000_000, 0)
	var tracker openAIOverloadShedTracker

	tracker.record(10, base, 0)
	_, first := tracker.record(10, base.Add(10*time.Second), 10*time.Minute)
	require.Equal(t, base.Add(10*time.Second+openAIOverloadPenaltyMax), first)

	tracker.record(10, base.Add(20*time.Second), 0)
	tripped, second := tracker.record(10, base.Add(30*time.Second), time.Second)
	require.True(t, tripped)
	require.Equal(t, first, second, "a shorter new penalty must not cut an existing one")
}

func TestOpenAIOverloadPenaltyClamp(t *testing.T) {
	require.Equal(t, openAIOverloadPenaltyDefault, openAIOverloadPenaltyFor(0))
	require.Equal(t, openAIOverloadPenaltyMin, openAIOverloadPenaltyFor(6*time.Second))
	require.Equal(t, 3*time.Minute, openAIOverloadPenaltyFor(3*time.Minute))
	require.Equal(t, openAIOverloadPenaltyMax, openAIOverloadPenaltyFor(374*time.Second))
}

func TestOpenAIOverloadRetryAfter(t *testing.T) {
	require.Equal(t, 374*time.Second, openAIOverloadRetryAfter([]byte(openAIOverloadFailedEventFixture)))
	require.Equal(t, 6*time.Second, openAIOverloadRetryAfter([]byte(`{"type":"error","error":{"code":"server_is_overloaded","headers":{"retry-after":"6"}}}`)))
	require.Zero(t, openAIOverloadRetryAfter([]byte(`{"error":{"message":"Our servers are currently overloaded."}}`)))

	headers := http.Header{}
	headers.Set("Retry-After", "120")
	require.Equal(t, 2*time.Minute, openAIOverloadRetryAfterHeader(headers))
	headers.Set("Retry-After-Ms", "1500")
	require.Equal(t, 1500*time.Millisecond, openAIOverloadRetryAfterHeader(headers))
	require.Zero(t, openAIOverloadRetryAfterHeader(nil))
}

func TestOpenAIStreamTerminalSideEffectsRecordCapacityShed(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 10, Name: "codex 1x", Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	payload := []byte(openAIOverloadFailedEventFixture)
	message := extractOpenAISSEErrorMessage(payload)

	svc.handleOpenAIStreamTerminalAccountSideEffects(nil, account, payload, message, nil)
	require.False(t, svc.overloadSheds.isPenalized(10, time.Now()))

	// A later shed of a different request (outside the collapse interval).
	svc.overloadSheds.mu.Lock()
	for _, entry := range svc.overloadSheds.entries {
		for i := range entry.sheds {
			entry.sheds[i] = entry.sheds[i].Add(-time.Minute)
		}
	}
	svc.overloadSheds.mu.Unlock()
	svc.handleOpenAIStreamTerminalAccountSideEffects(nil, account, payload, message, nil)
	require.True(t, svc.overloadSheds.isPenalized(10, time.Now()))
	require.True(t, svc.overloadSheds.isPenalized(10, time.Now().Add(openAIOverloadPenaltyMax-time.Second)),
		"the event's retry-after (374s) is clamped to the maximum penalty")
}

func TestOpenAIStreamTerminalSideEffectsIgnoreOtherFailures(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 10, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	payload := []byte(`{"type":"response.failed","response":{"error":{"code":"server_error","message":"An error occurred while processing your request."}}}`)
	for i := 0; i < 3; i++ {
		svc.handleOpenAIStreamTerminalAccountSideEffects(nil, account, payload, extractOpenAISSEErrorMessage(payload), nil)
	}
	require.Equal(t, 0, len(svc.overloadSheds.entries))
}

func newOverloadSelectionTestService(accounts []Account) *OpenAIGatewayService {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	return &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
}

func deprioritizeForTest(svc *OpenAIGatewayService, accountID int64) {
	now := time.Now()
	svc.overloadSheds.record(accountID, now.Add(-time.Minute), 0)
	tripped, _ := svc.overloadSheds.record(accountID, now, 0)
	if !tripped {
		panic("expected the overload tracker to trip")
	}
}

func TestSelectAccountWithScheduler_SkipsOverloadDeprioritizedAccount(t *testing.T) {
	accounts := []Account{
		{ID: 2411001, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 5, Priority: 0},
		{ID: 2411002, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 5, Priority: 5},
	}
	svc := newOverloadSelectionTestService(accounts)

	selection, _, err := svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-6.1-sol", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, int64(2411001), selection.Account.ID, "the better-priority account wins while healthy")

	deprioritizeForTest(svc, 2411001)
	selection, _, err = svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-6.1-sol", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, int64(2411002), selection.Account.ID)
}

func TestSelectAccountWithScheduler_OverloadDeprioritizedSoleAccountStillServes(t *testing.T) {
	accounts := []Account{
		{ID: 2411003, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 5, Priority: 0},
	}
	svc := newOverloadSelectionTestService(accounts)
	deprioritizeForTest(svc, 2411003)

	selection, _, err := svc.SelectAccountWithScheduler(context.Background(), nil, "", "", "gpt-6.1-sol", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err, "deprioritization must fail open instead of returning no available accounts")
	require.NotNil(t, selection)
	require.Equal(t, int64(2411003), selection.Account.ID)
	require.True(t, svc.overloadSheds.isPenalized(2411003, time.Now()), "fail-open does not clear the penalty")
}

func TestSelectAccountWithScheduler_StickySessionLeavesOverloadDeprioritizedAccount(t *testing.T) {
	accounts := []Account{
		{ID: 2411004, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 5, Priority: 0},
		{ID: 2411005, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 5, Priority: 5},
	}
	svc := newOverloadSelectionTestService(accounts)
	svc.cache = &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:overload_session": 2411004}}

	selection, _, err := svc.SelectAccountWithScheduler(context.Background(), nil, "", "overload_session", "gpt-6.1-sol", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, int64(2411004), selection.Account.ID, "sticky binding holds while the account is healthy")

	deprioritizeForTest(svc, 2411004)
	selection, _, err = svc.SelectAccountWithScheduler(context.Background(), nil, "", "overload_session", "gpt-6.1-sol", nil, OpenAIUpstreamTransportAny, false)
	require.NoError(t, err)
	require.Equal(t, int64(2411005), selection.Account.ID)
}
