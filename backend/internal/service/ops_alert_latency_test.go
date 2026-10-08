//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeRuleMetricLatencyAndTTFT(t *testing.T) {
	p := func(v int) *int { return &v }
	overview := &OpsDashboardOverview{
		RequestCountSLA: 40,
		Duration:        OpsPercentiles{P95: p(95_000), P99: p(150_000)},
		TTFT:            OpsPercentiles{P50: p(27_000), P95: p(70_000)},
	}
	svc := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{overview: overview}}
	now := time.Now()
	for metric, want := range map[string]float64{
		"p95_latency_ms": 95_000, "p99_latency_ms": 150_000, "ttft_p50_ms": 27_000, "ttft_p95_ms": 70_000,
	} {
		v, ok := svc.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: metric}, nil, now.Add(-10*time.Minute), now, "", nil)
		require.True(t, ok, metric)
		require.InDelta(t, want, v, 0.1, metric)
	}

	// No traffic or no samples: nothing to judge.
	quiet := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{overview: &OpsDashboardOverview{TTFT: OpsPercentiles{P50: p(1)}}}}
	_, ok := quiet.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "ttft_p50_ms"}, nil, now.Add(-time.Minute), now, "", nil)
	require.False(t, ok)
	empty := &OpsAlertEvaluatorService{opsRepo: &stubOpsRepo{overview: &OpsDashboardOverview{RequestCountSLA: 5}}}
	_, ok = empty.computeRuleMetric(context.Background(), &OpsAlertRule{MetricType: "p95_latency_ms"}, nil, now.Add(-time.Minute), now, "", nil)
	require.False(t, ok)
}
