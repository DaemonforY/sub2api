-- AI 建站 (我的网站 → AI 生成网站): one row per page the agent builds for a user. The model writes a
-- single-file HTML page and gpt-image-2 draws its pictures with the user's own API key; the user
-- keeps changing it by chat and publishes it through site hosting (a new site or a new version).
-- `data` holds the brief, the current HTML, a few earlier versions for undo, the change requests,
-- the pictures and progress events; pictures are files under <data dir>/site-drafts/<id>/.
-- Drafts untouched for 90 days are deleted with their files (published sites are not affected).
CREATE TABLE IF NOT EXISTS site_drafts (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id  BIGINT       NOT NULL,
    status      VARCHAR(16)  NOT NULL,
    title       VARCHAR(200) NOT NULL DEFAULT '',
    data        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    error       TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_site_drafts_user_updated ON site_drafts (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_site_drafts_active ON site_drafts (status) WHERE status IN ('generating', 'drawing');

COMMENT ON TABLE site_drafts IS 'AI 建站：用用户自己的 Key 生成单页网站和配图，对话修改后发布到网站托管；关掉页面也继续';
