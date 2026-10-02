-- Canvas cloud sync: each signed-in user's synced files (domain manifests and media), stored on
-- disk under canvas_cloud.dir/<user_id>/; one row per logical path.
CREATE TABLE IF NOT EXISTS canvas_cloud_files (
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path        VARCHAR(300) NOT NULL,
    file        VARCHAR(80)  NOT NULL,
    size        BIGINT       NOT NULL,
    mime        VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream',
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, path)
);
