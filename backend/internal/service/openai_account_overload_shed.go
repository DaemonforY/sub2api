package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// Upstream capacity shedding ("Our servers are currently overloaded",
// server_is_overloaded / slow_down) is treated as request-scoped: the failed
// request is retried on the same account and the account stays healthy. That
// is right for an isolated shed, but OpenAI also sheds one account repeatedly
// for minutes at a time and says so with a retry-after of several minutes.
//
// Motivating incident (2026-11-24 10:31–15:18): Codex OAuth account 10 shed
// 150+ streams, usually after output had started, so the gateway could not fail
// over and users saw the error. The other account in the same groups served
// 742 requests without a single error in that time, yet account 10 kept
// getting new requests and its sticky sessions.
//
// The window is longer than the retry-after OpenAI sends (~5–6 minutes): on
// 2026-11-24 account 10 shed once per retry-after period, 5m05s apart, which a
// 5-minute window never caught.
//
// An account that sheds twice within the window is deprioritized for the
// upstream's retry-after (clamped). Deprioritized means it is left out of
// selection while another account can serve; when it is the only usable
// account, the fail-open selection pass in selectAccountWithScheduler admits it
// again, so this never turns partial failures into "no available accounts".
const (
	openAIOverloadShedThreshold = 2
	openAIOverloadShedWindow    = 10 * time.Minute
	// openAIOverloadShedCollapse merges sheds that land together: a bare error
	// event followed by response.failed for the same request, or several
	// streams cut in the same instant, are one observation.
	openAIOverloadShedCollapse   = 2 * time.Second
	openAIOverloadPenaltyDefault = 2 * time.Minute
	openAIOverloadPenaltyMin     = time.Minute
	openAIOverloadPenaltyMax     = 5 * time.Minute
)

type openAIOverloadShedEntry struct {
	sheds         []time.Time
	penaltyUntil  time.Time
	lastTouchedAt time.Time
}

// openAIOverloadShedTracker is per process and in memory, like the proxy stream
// circuit: a restart forgets it and every penalty expires on its own.
// The zero value is ready to use.
type openAIOverloadShedTracker struct {
	mu      sync.Mutex
	entries map[int64]*openAIOverloadShedEntry
}

// record notes a shed on accountID. When the account reaches the threshold it
// is deprioritized until now+penalty (an existing longer penalty is kept) and
// record returns true with that time.
func (t *openAIOverloadShedTracker) record(accountID int64, now time.Time, retryAfter time.Duration) (bool, time.Time) {
	if t == nil || accountID <= 0 {
		return false, time.Time{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries == nil {
		t.entries = make(map[int64]*openAIOverloadShedEntry)
	}
	t.pruneLocked(now)
	entry := t.entries[accountID]
	if entry == nil {
		entry = &openAIOverloadShedEntry{}
		t.entries[accountID] = entry
	}
	entry.lastTouchedAt = now
	if n := len(entry.sheds); n > 0 && now.Sub(entry.sheds[n-1]) < openAIOverloadShedCollapse {
		return false, entry.penaltyUntil
	}
	entry.sheds = append(entry.sheds, now)
	if len(entry.sheds) < openAIOverloadShedThreshold {
		return false, entry.penaltyUntil
	}
	entry.sheds = entry.sheds[:0]
	until := now.Add(openAIOverloadPenaltyFor(retryAfter))
	if until.After(entry.penaltyUntil) {
		entry.penaltyUntil = until
	}
	return true, entry.penaltyUntil
}

func (t *openAIOverloadShedTracker) pruneLocked(now time.Time) {
	cutoff := now.Add(-openAIOverloadShedWindow)
	for id, entry := range t.entries {
		kept := entry.sheds[:0]
		for _, at := range entry.sheds {
			if at.After(cutoff) {
				kept = append(kept, at)
			}
		}
		entry.sheds = kept
		if len(kept) == 0 && !now.Before(entry.penaltyUntil) {
			delete(t.entries, id)
		}
	}
}

func (t *openAIOverloadShedTracker) isPenalized(accountID int64, now time.Time) bool {
	if t == nil || accountID <= 0 {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.entries[accountID]
	return entry != nil && now.Before(entry.penaltyUntil)
}

func (t *openAIOverloadShedTracker) activePenaltyCount(now time.Time) int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	for _, entry := range t.entries {
		if now.Before(entry.penaltyUntil) {
			count++
		}
	}
	return count
}

func openAIOverloadPenaltyFor(retryAfter time.Duration) time.Duration {
	if retryAfter <= 0 {
		return openAIOverloadPenaltyDefault
	}
	if retryAfter < openAIOverloadPenaltyMin {
		return openAIOverloadPenaltyMin
	}
	if retryAfter > openAIOverloadPenaltyMax {
		return openAIOverloadPenaltyMax
	}
	return retryAfter
}

// openAIOverloadRetryAfter reads the retry-after OpenAI puts inside the failed
// event (response.error.headers), e.g. {"retry-after":"374","retry-after-ms":"374000"}.
func openAIOverloadRetryAfter(payload []byte) time.Duration {
	if len(payload) == 0 {
		return 0
	}
	for _, prefix := range []string{"response.error.headers.", "error.headers."} {
		if ms := strings.TrimSpace(gjson.GetBytes(payload, prefix+"retry-after-ms").String()); ms != "" {
			if v, err := strconv.ParseFloat(ms, 64); err == nil && v > 0 {
				return time.Duration(v * float64(time.Millisecond))
			}
		}
		if sec := strings.TrimSpace(gjson.GetBytes(payload, prefix+"retry-after").String()); sec != "" {
			if v, err := strconv.ParseFloat(sec, 64); err == nil && v > 0 {
				return time.Duration(v * float64(time.Second))
			}
		}
	}
	return 0
}

// openAIOverloadRetryAfterHeader reads Retry-After-Ms / Retry-After (seconds).
func openAIOverloadRetryAfterHeader(headers http.Header) time.Duration {
	if headers == nil {
		return 0
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(headers.Get("Retry-After-Ms")), 64); err == nil && v > 0 {
		return time.Duration(v * float64(time.Millisecond))
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(headers.Get("Retry-After")), 64); err == nil && v > 0 {
		return time.Duration(v * float64(time.Second))
	}
	return 0
}

// noteOpenAIAccountCapacityShed records a capacity shed seen on account and
// deprioritizes the account once sheds repeat.
// headers are the upstream HTTP response headers of a non-streamed error, or
// nil for an event inside a 200 stream (whose outer headers describe success).
func (s *OpenAIGatewayService) noteOpenAIAccountCapacityShed(account *Account, payload []byte, headers http.Header) {
	if s == nil || account == nil || account.ID <= 0 || NormalizeOpenAICompatiblePlatform(account.Platform) != PlatformOpenAI {
		return
	}
	retryAfter := openAIOverloadRetryAfter(payload)
	if retryAfter <= 0 {
		retryAfter = openAIOverloadRetryAfterHeader(headers)
	}
	tripped, until := s.overloadSheds.record(account.ID, time.Now(), retryAfter)
	log := logger.L().With(zap.String("component", "service.openai_gateway"))
	if tripped {
		log.Warn("openai.account_overload_deprioritized",
			zap.Int64("account_id", account.ID),
			zap.String("account_name", account.Name),
			zap.Duration("upstream_retry_after", retryAfter),
			zap.Time("until", until),
		)
		return
	}
	log.Info("openai.account_overload_strike",
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.Duration("upstream_retry_after", retryAfter),
	)
}

// isOpenAIAccountOverloadDeprioritized reports whether selection should skip
// account because it keeps shedding load. The fail-open pass ignores it.
func (s *OpenAIGatewayService) isOpenAIAccountOverloadDeprioritized(ctx context.Context, account *Account) bool {
	if s == nil || account == nil {
		return false
	}
	if openAIProxyStreamQuarantineBypassed(ctx) {
		return false
	}
	return s.overloadSheds.isPenalized(account.ID, time.Now())
}

// isOpenAIAccountSoftExcluded groups the selection exclusions that must give
// way when nothing else can serve: proxy stream quarantine and overload
// deprioritization. Both are lifted by the fail-open selection pass.
func (s *OpenAIGatewayService) isOpenAIAccountSoftExcluded(ctx context.Context, account *Account) bool {
	return s.isOpenAIProxyStreamQuarantined(ctx, account) || s.isOpenAIAccountOverloadDeprioritized(ctx, account)
}
