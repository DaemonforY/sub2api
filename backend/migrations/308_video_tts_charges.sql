-- HiveGPT 视频: paid narration (豆包语音 2.0) is charged per character from the balance, one row per
-- recorded scene, so every deduction can be traced.
CREATE TABLE IF NOT EXISTS video_tts_charges (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id TEXT           NOT NULL,
    scene_id   TEXT           NOT NULL,
    voice      TEXT           NOT NULL,
    chars      INT            NOT NULL,
    cost       NUMERIC(20, 8) NOT NULL,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_video_tts_charges_user_created ON video_tts_charges (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_video_tts_charges_project ON video_tts_charges (project_id);
