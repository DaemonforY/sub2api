-- AI 写文章 (公众号排版 → AI 写文章): one row per article the agent writes for a user. The server
-- researches, outlines (the user confirms the outline), writes and draws the pictures with the
-- user's own API key, and keeps going when the tab is closed; the editor then pushes the result to
-- the user's 公众号 draft box from the browser (the AppSecret never reaches the server).
-- `data` holds the brief, sources, outline, Markdown, pictures and progress events; pictures are
-- files under <data dir>/articles/<id>/. Rows older than 30 days are deleted with their files.
CREATE TABLE IF NOT EXISTS article_projects (
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

CREATE INDEX IF NOT EXISTS idx_article_projects_user_created ON article_projects (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_article_projects_active ON article_projects (status) WHERE status IN ('outlining', 'writing', 'drawing');

COMMENT ON TABLE article_projects IS '公众号「AI 写文章」：搜资料、出大纲、写全文、配图的后台任务，用用户自己的 Key，关掉页面也继续';
