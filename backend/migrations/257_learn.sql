-- AI 学习 (/learn): completed lessons per user, and the runs of lesson examples (each a model call
-- through the gateway with the admin's learning key; also the daily free-run counter).
CREATE TABLE IF NOT EXISTS learn_progress (
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id    VARCHAR(32) NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, lesson_id)
);

CREATE TABLE IF NOT EXISTS learn_runs (
    id                BIGSERIAL   PRIMARY KEY,
    user_id           BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id         VARCHAR(32) NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'pending',
    latency_ms        INTEGER     NOT NULL DEFAULT 0,
    prompt_tokens     INTEGER     NOT NULL DEFAULT 0,
    completion_tokens INTEGER     NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_learn_runs_user_created ON learn_runs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_learn_runs_created ON learn_runs (created_at DESC);
