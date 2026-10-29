-- Monthly contests created automatically from a template (admin 比赛管理 → 每月自动比赛).
-- auto_period ("2026-11") marks a contest the job created; the unique index keeps two app
-- instances from creating the same month twice.
ALTER TABLE contests ADD COLUMN IF NOT EXISTS auto_period VARCHAR(16) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS uq_contests_auto_period ON contests (auto_period) WHERE auto_period <> '';

-- One row: the switch, the template and the last month created. A month at or before
-- last_period is never created again, so a cancelled or deleted contest stays gone.
CREATE TABLE IF NOT EXISTS contest_auto_settings (
    id          SMALLINT     PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled     BOOLEAN      NOT NULL DEFAULT FALSE,
    template    JSONB        NOT NULL DEFAULT '{}'::jsonb,
    last_period VARCHAR(16)  NOT NULL DEFAULT '',
    updated_by  BIGINT       NULL,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
