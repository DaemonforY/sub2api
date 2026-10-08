-- 渠道链接: one short link per promotion channel. /go/<code> redirects to target_path with
-- utm_source / utm_medium / utm_campaign=<code> (and aff=<aff_code> for partner links); the
-- admin page counts each link's visitors, sign-ups, activation and payments by that campaign.
CREATE TABLE IF NOT EXISTS channel_links (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(32)  NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    source      VARCHAR(32)  NOT NULL,
    medium      VARCHAR(32)  NOT NULL DEFAULT '',
    target_path VARCHAR(200) NOT NULL DEFAULT '/',
    aff_code    VARCHAR(32)  NOT NULL DEFAULT '',
    note        VARCHAR(300) NOT NULL DEFAULT '',
    clicks      BIGINT       NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_campaign ON analytics_events (campaign) WHERE campaign <> '';
CREATE INDEX IF NOT EXISTS idx_user_attributions_campaign ON user_attributions (campaign) WHERE campaign <> '';
