-- Latency alert rules start working (the evaluator now computes p95/p99_latency_ms and ttft_*_ms).
-- The seeded P95 2000 ms / P99 3000 ms thresholds suit plain API calls but every streamed model
-- answer takes longer than that, so untouched defaults move to whole-request limits that mean
-- trouble; rules an admin already edited are left alone.
UPDATE ops_alert_rules
SET threshold = 180000, description = '当整次请求 P95 耗时超过 180 秒且持续 10 分钟时触发告警（流式回答本身较慢，阈值按整次请求计）', updated_at = NOW()
WHERE name = 'P95延迟过高' AND metric_type = 'p95_latency_ms' AND threshold = 2000;

UPDATE ops_alert_rules
SET threshold = 300000, description = '当整次请求 P99 耗时超过 300 秒且持续 10 分钟时触发告警（流式回答本身较慢，阈值按整次请求计）', updated_at = NOW()
WHERE name = 'P99延迟过高' AND metric_type = 'p99_latency_ms' AND threshold = 3000;

-- Time to first token is what users feel as "卡": normally 2–5 s; 20 s for 10 minutes means the
-- upstream is struggling.
INSERT INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    created_at, updated_at
) VALUES (
    '首字延迟过高',
    '当首 Token 延迟中位数超过 20 秒且持续 10 分钟时触发告警（用户会感觉「卡」，多半是上游慢）',
    true, 'ttft_p50_ms', '>', 20000.0, 10, 10, 'P1', true, 60, NOW(), NOW()
) ON CONFLICT (name) DO NOTHING;
