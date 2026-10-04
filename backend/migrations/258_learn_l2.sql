-- AI 学习 L2: quizzes, checkpoints, certificates, the AI tutor and mock interviews.

-- Runs are also tutor questions and interview gradings (kind); key_id is the learner's own key when
-- they go on past the free runs (billed as usual on that key, not counted as free).
ALTER TABLE learn_runs ADD COLUMN IF NOT EXISTS kind VARCHAR(16) NOT NULL DEFAULT 'run';
ALTER TABLE learn_runs ADD COLUMN IF NOT EXISTS key_id BIGINT;

-- Best quiz result per lesson (answers are graded on the server).
CREATE TABLE IF NOT EXISTS learn_quiz_results (
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lesson_id  VARCHAR(32) NOT NULL,
    correct    INTEGER     NOT NULL DEFAULT 0,
    total      INTEGER     NOT NULL DEFAULT 0,
    attempts   INTEGER     NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, lesson_id)
);

-- Checkpoints the server verified (has a key, published a work, ...).
CREATE TABLE IF NOT EXISTS learn_checkpoints (
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkpoint_id VARCHAR(32) NOT NULL,
    passed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, checkpoint_id)
);

-- One live certificate per learner and track (a revoked one may be claimed again); public at /learn/cert.html?c=<code>.
CREATE TABLE IF NOT EXISTS learn_certificates (
    code         VARCHAR(16)  PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    track        VARCHAR(8)   NOT NULL,
    display_name VARCHAR(40)  NOT NULL,
    project_url  VARCHAR(500) NOT NULL DEFAULT '',
    quiz_score   INTEGER      NOT NULL DEFAULT 0,
    issued_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at   TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_learn_certificates_user_track ON learn_certificates (user_id, track) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_learn_certificates_issued ON learn_certificates (issued_at DESC);

-- Mock interviews: questions picked from a topic's bank, one graded answer each.
CREATE TABLE IF NOT EXISTS learn_interviews (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic       VARCHAR(16) NOT NULL,
    questions   JSONB       NOT NULL DEFAULT '[]',
    answers     JSONB       NOT NULL DEFAULT '[]',
    status      VARCHAR(16) NOT NULL DEFAULT 'active',
    score       INTEGER     NOT NULL DEFAULT 0,
    key_id      BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_learn_interviews_user ON learn_interviews (user_id, created_at DESC);
