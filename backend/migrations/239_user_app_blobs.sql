-- Images that companion apps sync with a user's account (e.g. the canvas' draft reference images).
-- Files live on the data volume (<data>/app-state-blobs/<id>.<ext>); this table owns them per user,
-- dedupes by content hash, and drives quota eviction by last use.
CREATE TABLE IF NOT EXISTS user_app_blobs (
    id           VARCHAR(32)  PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sha256       CHAR(64)     NOT NULL,
    mime_type    VARCHAR(32)  NOT NULL,
    size_bytes   BIGINT       NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_app_blobs_user_sha ON user_app_blobs (user_id, sha256);
CREATE INDEX IF NOT EXISTS idx_user_app_blobs_user_last_used ON user_app_blobs (user_id, last_used_at DESC);

COMMENT ON TABLE user_app_blobs IS '用户在配套应用中同步的图片（如无限画布草稿的参考图）';
