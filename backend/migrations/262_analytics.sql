-- First-party usage analytics (埋点). Browser events from the main site, /learn and /editor, and
-- each new user's first-touch source. Raw events are kept 90 days (AnalyticsService deletes older).
-- No input text, prompts, keys or full IPs are stored: ip_prefix is the /24 (IPv4) or /48 (IPv6).
CREATE TABLE IF NOT EXISTS analytics_events (
    id            BIGSERIAL    PRIMARY KEY,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    event         VARCHAR(32)  NOT NULL,
    app           VARCHAR(16)  NOT NULL DEFAULT 'main',
    path          VARCHAR(200) NOT NULL DEFAULT '',
    visitor_id    VARCHAR(64)  NOT NULL,
    session_id    VARCHAR(64)  NOT NULL DEFAULT '',
    user_id       BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    source        VARCHAR(64)  NOT NULL DEFAULT 'direct',
    medium        VARCHAR(64)  NOT NULL DEFAULT '',
    campaign      VARCHAR(64)  NOT NULL DEFAULT '',
    referrer_host VARCHAR(128) NOT NULL DEFAULT '',
    device        VARCHAR(16)  NOT NULL DEFAULT '',
    ip_prefix     VARCHAR(48)  NOT NULL DEFAULT '',
    props         JSONB
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_created ON analytics_events (created_at);
CREATE INDEX IF NOT EXISTS idx_analytics_events_event_created ON analytics_events (event, created_at);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_created ON analytics_events (user_id, created_at) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_analytics_events_visitor ON analytics_events (visitor_id);

-- Where a user came from, captured in the browser on the first visit and saved at sign-up.
CREATE TABLE IF NOT EXISTS user_attributions (
    user_id       BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    visitor_id    VARCHAR(64)  NOT NULL DEFAULT '',
    source        VARCHAR(64)  NOT NULL DEFAULT 'direct',
    medium        VARCHAR(64)  NOT NULL DEFAULT '',
    campaign      VARCHAR(64)  NOT NULL DEFAULT '',
    aff_code      VARCHAR(64)  NOT NULL DEFAULT '',
    referrer_host VARCHAR(128) NOT NULL DEFAULT '',
    landing_path  VARCHAR(200) NOT NULL DEFAULT '',
    first_seen_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_attributions_source ON user_attributions (source);
