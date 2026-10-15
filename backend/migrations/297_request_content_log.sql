-- 请求内容日志: for accounts an admin explicitly turns on (content_log_targets), the gateway keeps a
-- small, redacted record of what each API request asked and what the model did — the latest user
-- message, the tool calls the model emitted and the head of its text reply. Secrets are redacted
-- before storage, rows expire after 7 days, and every admin enable / disable / view is written to
-- content_log_access_logs (kept 180 days). Never shown to the user: only admin endpoints read these.
CREATE TABLE IF NOT EXISTS content_log_targets (
    user_id     BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    note        TEXT        NOT NULL DEFAULT '',
    created_by  BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS request_content_logs (
    id                BIGSERIAL   PRIMARY KEY,
    user_id           BIGINT      NOT NULL,
    api_key_id        BIGINT,
    group_id          BIGINT,
    request_id        TEXT        NOT NULL DEFAULT '',
    session_id        TEXT        NOT NULL DEFAULT '',
    model             TEXT        NOT NULL DEFAULT '',
    inbound_endpoint  TEXT        NOT NULL DEFAULT '',
    user_agent        TEXT        NOT NULL DEFAULT '',
    ip_address        TEXT        NOT NULL DEFAULT '',
    user_text         TEXT        NOT NULL DEFAULT '',
    tool_calls        JSONB       NOT NULL DEFAULT '[]',
    reply_head        TEXT        NOT NULL DEFAULT '',
    input_tokens      INT         NOT NULL DEFAULT 0,
    output_tokens     INT         NOT NULL DEFAULT 0,
    cache_read_tokens INT         NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_request_content_logs_user_id ON request_content_logs (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_request_content_logs_created_at ON request_content_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_request_content_logs_user_session ON request_content_logs (user_id, session_id, id DESC);

CREATE TABLE IF NOT EXISTS content_log_access_logs (
    id              BIGSERIAL   PRIMARY KEY,
    admin_id        BIGINT      NOT NULL,
    action          TEXT        NOT NULL CHECK (action IN ('enable', 'disable', 'view')),
    target_user_id  BIGINT      NOT NULL,
    detail          TEXT        NOT NULL DEFAULT '',
    ip_address      TEXT        NOT NULL DEFAULT '',
    user_agent      TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_log_access_logs_target ON content_log_access_logs (target_user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_content_log_access_logs_created_at ON content_log_access_logs (created_at);
