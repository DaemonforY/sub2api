-- HiveGPT 视频 我的风格: a user's saved series look, so every film of an account looks the same.
-- A card is saved from a finished film: its shared kit (spec.common: palette, fonts, radius, chrome,
-- the recurring character), theme, ratio and base style, plus house rules the user keeps from
-- 复盘 (retrospectives). New films that pick the card adapt that kit instead of designing a new one.
CREATE TABLE IF NOT EXISTS video_styles (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           VARCHAR(40)  NOT NULL,
    ratio          VARCHAR(16)  NOT NULL DEFAULT '',
    base_style     VARCHAR(40)  NOT NULL DEFAULT '',
    theme          JSONB        NOT NULL DEFAULT '{}'::jsonb,
    common         TEXT         NOT NULL DEFAULT '',
    character_desc TEXT         NOT NULL DEFAULT '',
    rules          JSONB        NOT NULL DEFAULT '[]'::jsonb,
    source_project UUID         NULL REFERENCES video_projects(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_video_styles_user ON video_styles (user_id, updated_at DESC);
