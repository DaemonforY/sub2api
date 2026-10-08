-- AI 动画 jobs started from the canvas (canvas.<domain>). The server calls its own gateway with the
-- user's API key and keeps going when the browser is closed; the canvas polls for the result.
-- `meta` is the canvas' own bookkeeping (which record / version the result belongs to), opaque here.
-- Only the extracted SVG is stored; rows older than 7 days are deleted when new jobs are created.
CREATE TABLE IF NOT EXISTS animation_jobs (
    id                BIGSERIAL    PRIMARY KEY,
    user_id           BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id        BIGINT       NOT NULL,
    status            VARCHAR(16)  NOT NULL DEFAULT 'pending',
    model             VARCHAR(100) NOT NULL,
    meta              JSONB        NOT NULL DEFAULT '{}'::jsonb,
    svg               TEXT         NOT NULL DEFAULT '',
    error             TEXT         NOT NULL DEFAULT '',
    chars             INTEGER      NOT NULL DEFAULT 0,
    prompt_tokens     INTEGER      NOT NULL DEFAULT 0,
    completion_tokens INTEGER      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_animation_jobs_user_created ON animation_jobs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_animation_jobs_active ON animation_jobs (status) WHERE status IN ('pending', 'running');

COMMENT ON TABLE animation_jobs IS '无限画布「AI 动画」的后台生成任务：服务器用用户的 Key 调用本站网关，关掉浏览器也继续';
