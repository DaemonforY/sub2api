-- Sign-in sessions of the canvas (canvas.<domain>) with a HiveGPT account. The browser holds the
-- token in an HttpOnly cookie on the main site's host (path /api/v1/canvas), so scripts on the
-- canvas cannot read it; only its SHA-256 is stored. Sessions slide for 30 days and can be signed
-- out from the main site.
CREATE TABLE IF NOT EXISTS canvas_sessions (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   CHAR(64)     NOT NULL UNIQUE,
    user_agent   VARCHAR(255) NOT NULL DEFAULT '',
    ip           VARCHAR(64)  NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ  NOT NULL,
    revoked_at   TIMESTAMPTZ  NULL
);

CREATE INDEX IF NOT EXISTS idx_canvas_sessions_user ON canvas_sessions (user_id, last_used_at DESC);

COMMENT ON TABLE canvas_sessions IS '无限画布用 HiveGPT 账号登录的会话（Cookie 只存哈希）';
