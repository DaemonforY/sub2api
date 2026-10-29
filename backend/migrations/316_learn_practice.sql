-- 学画画 (docs/DRAWING_COURSE_PLAN.md): one row per critiqued exercise attempt in a lesson (a drawing
-- made in the page, an uploaded picture or a written appreciation). Pictures are not kept, only the
-- prompt or text and the scores. Free drawings and critiques are counted in learn_runs (kind
-- draw / critique).
CREATE TABLE IF NOT EXISTS learn_practice (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id   VARCHAR(32) NOT NULL,
    exercise_id VARCHAR(64) NOT NULL,
    kind        VARCHAR(16) NOT NULL,
    prompt      TEXT        NOT NULL DEFAULT '',
    score       INTEGER     NOT NULL,
    items       JSONB       NOT NULL DEFAULT '[]',
    key_id      BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_learn_practice_user ON learn_practice (user_id, exercise_id, score DESC);
