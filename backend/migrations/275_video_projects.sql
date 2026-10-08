-- HiveGPT 视频 (video.<domain>): AI-made narrated HTML videos and animations.
-- `spec` holds the script, each scene's narration timing and the scene code the player runs; the
-- narration audio itself lives on disk (video.dir/<project id>/<scene>-<hash>.mp3).
CREATE TABLE IF NOT EXISTS video_projects (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id   BIGINT       NOT NULL,
    mode         VARCHAR(16)  NOT NULL DEFAULT 'film',
    title        VARCHAR(200) NOT NULL DEFAULT '',
    prompt       TEXT         NOT NULL,
    options      JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status       VARCHAR(20)  NOT NULL DEFAULT 'running',
    stage        VARCHAR(20)  NOT NULL DEFAULT '',
    error        TEXT         NOT NULL DEFAULT '',
    spec         JSONB,
    duration     REAL         NOT NULL DEFAULT 0,
    width        INTEGER      NOT NULL DEFAULT 1920,
    height       INTEGER      NOT NULL DEFAULT 1080,
    usage        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    visibility   VARCHAR(16)  NOT NULL DEFAULT 'private',
    category     VARCHAR(40)  NOT NULL DEFAULT '',
    featured     BOOLEAN      NOT NULL DEFAULT FALSE,
    views        INTEGER      NOT NULL DEFAULT 0,
    remixes      INTEGER      NOT NULL DEFAULT 0,
    remix_of     VARCHAR(64)  NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_video_projects_user ON video_projects (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_projects_gallery ON video_projects (visibility, category, published_at DESC) WHERE visibility IN ('public', 'pending');
CREATE INDEX IF NOT EXISTS idx_video_projects_running ON video_projects (status) WHERE status = 'running';

-- The agent panel: the user's messages, the agent's steps, questions and replies.
CREATE TABLE IF NOT EXISTS video_project_events (
    id         BIGSERIAL    PRIMARY KEY,
    project_id UUID         NOT NULL REFERENCES video_projects(id) ON DELETE CASCADE,
    kind       VARCHAR(16)  NOT NULL,
    text       TEXT         NOT NULL DEFAULT '',
    data       JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_video_project_events_project ON video_project_events (project_id, id);

-- Snapshots of the spec before each change, so a change can be undone.
CREATE TABLE IF NOT EXISTS video_project_versions (
    id         BIGSERIAL    PRIMARY KEY,
    project_id UUID         NOT NULL REFERENCES video_projects(id) ON DELETE CASCADE,
    note       VARCHAR(200) NOT NULL DEFAULT '',
    spec       JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_video_project_versions_project ON video_project_versions (project_id, id DESC);

COMMENT ON TABLE video_projects IS 'HiveGPT 视频：AI 生成的 HTML 视频与动画（脚本、配音时间轴、分镜代码），可发布到案例库';
