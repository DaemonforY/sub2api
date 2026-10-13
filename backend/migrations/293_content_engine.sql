-- 内容引擎 (docs/CONTENT_ENGINE_PLAN.md): topics become a master article and one version per platform
-- (公众号 / 知乎 / CSDN / 头条 / 小红书), written in the background with the owner's own API key and
-- stopping at 待审核; the owner approves each version, which gets its own channel link (and promo code
-- where a platform allows no links), publishes it and records the post URL. Every table carries the
-- owner so the same tables serve users later; today the owner is an admin.

-- Per-owner settings (key, model, daily schedule, platforms, style, WeChat push). The push key is
-- AES-encrypted (SecretEncryptor) and never returned.
CREATE TABLE IF NOT EXISTS content_owner_settings (
    owner_user_id   BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    data            JSONB        NOT NULL DEFAULT '{}'::jsonb,
    notify_key_enc  TEXT         NOT NULL DEFAULT '',
    enabled         BOOLEAN      NOT NULL DEFAULT FALSE,
    last_run_day    VARCHAR(10)  NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 选题池: suggested by the model (from support questions, GEO questions, tutorials, the web) or added
-- by hand; the daily run takes the best-scored pending ones.
CREATE TABLE IF NOT EXISTS content_topics (
    id             BIGSERIAL    PRIMARY KEY,
    owner_user_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title          VARCHAR(200) NOT NULL,
    angle          VARCHAR(500) NOT NULL DEFAULT '',
    notes          TEXT         NOT NULL DEFAULT '',
    source         VARCHAR(32)  NOT NULL DEFAULT 'manual',
    reason         VARCHAR(500) NOT NULL DEFAULT '',
    score          INT          NOT NULL DEFAULT 50,
    status         VARCHAR(16)  NOT NULL DEFAULT 'pending',
    item_id        BIGINT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_topics_owner_status ON content_topics (owner_user_id, status, score DESC);

-- 母稿: one per topic run; `data` holds the brief, outline, draft, review notes, final Markdown,
-- sources, progress events and token usage.
CREATE TABLE IF NOT EXISTS content_items (
    id             BIGSERIAL    PRIMARY KEY,
    owner_user_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id       BIGINT,
    api_key_id     BIGINT       NOT NULL,
    status         VARCHAR(16)  NOT NULL,
    title          VARCHAR(200) NOT NULL DEFAULT '',
    data           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    error          TEXT         NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_items_owner_created ON content_items (owner_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_content_items_active ON content_items (status) WHERE status IN ('outlining', 'drafting', 'reviewing', 'humanizing', 'adapting');

-- 平台版本: the text to publish on one platform. The body keeps {{link}} / {{promo}} placeholders
-- until approval creates the channel link (link_code) and promo code.
CREATE TABLE IF NOT EXISTS content_variants (
    id             BIGSERIAL    PRIMARY KEY,
    item_id        BIGINT       NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    owner_user_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform       VARCHAR(16)  NOT NULL,
    kind           VARCHAR(16)  NOT NULL DEFAULT 'article',
    status         VARCHAR(16)  NOT NULL,
    title          VARCHAR(200) NOT NULL DEFAULT '',
    body           TEXT         NOT NULL DEFAULT '',
    data           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    link_code      VARCHAR(32)  NOT NULL DEFAULT '',
    promo_code     VARCHAR(32)  NOT NULL DEFAULT '',
    scheduled_at   TIMESTAMPTZ,
    post_url       VARCHAR(500) NOT NULL DEFAULT '',
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_content_variants_owner_published ON content_variants (owner_user_id, published_at DESC) WHERE published_at IS NOT NULL;

-- 写作规则 (改进记忆): what the owner wants avoided or done, from rejection reasons or typed in; the
-- enabled ones go into every rewrite.
CREATE TABLE IF NOT EXISTS content_rules (
    id             BIGSERIAL    PRIMARY KEY,
    owner_user_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule           VARCHAR(300) NOT NULL,
    origin         VARCHAR(16)  NOT NULL DEFAULT 'manual',
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_rules_owner ON content_rules (owner_user_id, id);

COMMENT ON TABLE content_items IS '内容引擎：选题 → 大纲 → 初稿 → 审校 → 去 AI 味 → 各平台版本，停在待审核，用 owner 自己的 Key';
COMMENT ON TABLE content_variants IS '内容引擎：某个平台的版本，审核通过时生成渠道链接 / 优惠码，发布后登记帖子链接';
