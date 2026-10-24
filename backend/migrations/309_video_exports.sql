-- HiveGPT 视频 MP4 export and the publishing kit. An export is one render of a project in one layout
-- (landscape 1920×1080 or portrait 1080×1920, 60 fps) by the videorender container; the files live
-- next to the narration audio (video.dir/<project>/exports/). `signature` identifies what was rendered,
-- so a changed film is rendered again. video_publish_copy keeps the AI-written titles, descriptions and
-- topics per platform (抖音 / 视频号 / 小红书 / B站).
CREATE TABLE IF NOT EXISTS video_exports (
    id         TEXT PRIMARY KEY,
    project_id UUID             NOT NULL REFERENCES video_projects(id) ON DELETE CASCADE,
    user_id    BIGINT           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    layout     VARCHAR(16)      NOT NULL,
    status     VARCHAR(16)      NOT NULL DEFAULT 'queued',
    signature  VARCHAR(64)      NOT NULL DEFAULT '',
    file       TEXT             NOT NULL DEFAULT '',
    cover      TEXT             NOT NULL DEFAULT '',
    size       BIGINT           NOT NULL DEFAULT 0,
    duration   DOUBLE PRECISION NOT NULL DEFAULT 0,
    error      TEXT             NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_video_exports_project ON video_exports (project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_exports_user_created ON video_exports (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_exports_unfinished ON video_exports (created_at) WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS video_publish_copy (
    project_id UUID        PRIMARY KEY REFERENCES video_projects(id) ON DELETE CASCADE,
    copy       JSONB       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
