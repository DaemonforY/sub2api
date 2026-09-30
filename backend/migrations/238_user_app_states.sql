-- Per-user JSON documents for companion apps (e.g. the canvas at canvas.<domain>): prompt favorites
-- and prompt drafts that follow the account across devices. One row per (user, namespace);
-- `version` implements compare-and-set so two devices cannot silently overwrite each other.
CREATE TABLE IF NOT EXISTS user_app_states (
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    namespace  VARCHAR(64)  NOT NULL,
    value      JSONB        NOT NULL,
    version    BIGINT       NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, namespace)
);

COMMENT ON TABLE user_app_states IS '用户在配套应用（无限画布等）中的同步数据：提示词收藏、草稿';
