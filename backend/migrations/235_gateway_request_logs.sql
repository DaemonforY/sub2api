-- Full gateway request log: one row per request that reaches the API gateway
-- (/v1, /v1beta, /backend-api, /antigravity and top-level aliases such as
-- /responses, /models, /chat/completions), regardless of outcome.
-- Unlike usage_logs (billed successes only), this also captures failed,
-- rejected (invalid key, disabled group, quota) and unmatched-route requests.
-- Admin-only; api_key holds the full credential presented by the caller.
CREATE TABLE IF NOT EXISTS gateway_request_logs (
    id           BIGSERIAL PRIMARY KEY,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    request_id   VARCHAR(64)  NOT NULL DEFAULT '',
    method       VARCHAR(16)  NOT NULL DEFAULT '',
    url          TEXT         NOT NULL DEFAULT '',
    path         VARCHAR(512) NOT NULL DEFAULT '',
    status_code  INTEGER      NOT NULL DEFAULT 0,
    success      BOOLEAN      NOT NULL DEFAULT FALSE,
    error_code   VARCHAR(64)  NOT NULL DEFAULT '',
    duration_ms  BIGINT       NOT NULL DEFAULT 0,
    api_key      TEXT         NOT NULL DEFAULT '',
    api_key_id   BIGINT       NULL,
    user_id      BIGINT       NULL,
    group_id     BIGINT       NULL,
    account_id   BIGINT       NULL,
    model        VARCHAR(128) NOT NULL DEFAULT '',
    client_ip    VARCHAR(64)  NOT NULL DEFAULT '',
    user_agent   VARCHAR(512) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_gateway_request_logs_created_at
    ON gateway_request_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_gateway_request_logs_user_created
    ON gateway_request_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gateway_request_logs_api_key_created
    ON gateway_request_logs (api_key_id, created_at DESC);

COMMENT ON TABLE gateway_request_logs IS
    'Every API gateway request (success or failure) with full URL and presented API key; admin-only, pruned by request_log.retention_days';
