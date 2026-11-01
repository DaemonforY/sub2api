-- AI 做贴图 (公众号排版 → AI 做贴图): one row per 公众号贴图 (newspic) the agent makes for a user from a
-- poem. The server picks or takes the poem, cross-checks its text, plans the pictures (a story series
-- or close-ups of one scene) with period details, draws cover candidates and the panels (the chosen
-- cover is the reference, so characters and style stay the same), checks every picture with a vision
-- model and writes the title and body. The user confirms the plan and picks the cover; the editor
-- frames the pictures to 3:4 and pushes the draft from the browser (the AppSecret never reaches the
-- server). `data` holds all of it; pictures are files under <data dir>/tietu/<id>/. Rows older than
-- 30 days are deleted with their files.
CREATE TABLE IF NOT EXISTS tietu_projects (
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

CREATE INDEX IF NOT EXISTS idx_tietu_projects_user_created ON tietu_projects (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tietu_projects_active ON tietu_projects (status) WHERE status IN ('suggesting', 'planning', 'covering', 'drawing', 'checking');

COMMENT ON TABLE tietu_projects IS '公众号「AI 做贴图」：选诗、核对原文、构思细节和分镜、出封面和分镜图、看图自检、写文案的后台任务，用用户自己的 Key';
