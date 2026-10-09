-- Agent runs: one row per question answered by a tool-using assistant (first the homepage 智能客服),
-- with the tools it called, so admins can see what it looked up and why an answer went wrong.
-- `steps` is a JSON array of {tool, args, result, error, ms}; results are cut to a length.
-- Rows older than 30 days are deleted as new runs come in.
CREATE TABLE IF NOT EXISTS agent_runs (
    id                BIGSERIAL    PRIMARY KEY,
    agent             VARCHAR(32)  NOT NULL,
    user_id           BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    question          TEXT         NOT NULL DEFAULT '',
    answer            TEXT         NOT NULL DEFAULT '',
    steps             JSONB        NOT NULL DEFAULT '[]'::jsonb,
    model             VARCHAR(100) NOT NULL DEFAULT '',
    model_calls       INTEGER      NOT NULL DEFAULT 0,
    prompt_tokens     INTEGER      NOT NULL DEFAULT 0,
    completion_tokens INTEGER      NOT NULL DEFAULT 0,
    status            VARCHAR(16)  NOT NULL DEFAULT 'ok',
    error             TEXT         NOT NULL DEFAULT '',
    duration_ms       INTEGER      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_agent_created ON agent_runs (agent, created_at DESC);

COMMENT ON TABLE agent_runs IS '工具型助手（首页智能客服等）的运行记录：问题、调用的工具和结果、用量，保留 30 天';
