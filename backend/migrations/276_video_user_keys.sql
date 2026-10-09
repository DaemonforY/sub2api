-- HiveGPT 视频: each user's 「HiveGPT 视频」 API key per video group (the group of a model offered on
-- the video site), created when the user first runs that model.
CREATE TABLE IF NOT EXISTS video_user_keys (
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id   BIGINT      NOT NULL,
    api_key_id BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);
