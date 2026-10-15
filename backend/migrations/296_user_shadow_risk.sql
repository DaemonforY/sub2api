-- 静默限制: a restricted account looks normal to its owner, but free credit (sign-up trial
-- subscription, gifted balance, promo codes) only works up to a small daily allowance; past it the
-- user gets the same 今日额度已用完 / 余额不足 errors as anyone else. Money the user paid works as usual.
-- Never shown to the user: only admin endpoints read this table.
CREATE TABLE IF NOT EXISTS user_shadows (
    user_id         BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    reason          VARCHAR(500) NOT NULL DEFAULT '',
    source          VARCHAR(16)  NOT NULL DEFAULT 'admin', -- auto (sign-up rules) | admin
    related_user_id BIGINT,
    created_by      BIGINT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 风控事件: what the risk monitor found, one row per user and rule (later hits don't repeat).
CREATE TABLE IF NOT EXISTS risk_events (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    rule        VARCHAR(32)  NOT NULL,
    detail      VARCHAR(500) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, rule)
);
CREATE INDEX IF NOT EXISTS idx_risk_events_created ON risk_events (created_at DESC);
