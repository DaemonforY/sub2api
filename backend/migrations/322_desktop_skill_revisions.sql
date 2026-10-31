-- 桌面助手技能库的修订历史. A published skill is instructions every user's Codex follows, so every
-- change is kept: before a skill is saved, published / hidden or deleted, its current row is copied
-- here in the same transaction (snapshot = the whole row as JSON). Rows stay after the skill is
-- deleted. The admin editor lists them and can load an old version back into the form.
CREATE TABLE IF NOT EXISTS desktop_skill_revisions (
    id          BIGSERIAL    PRIMARY KEY,
    skill_id    BIGINT       NOT NULL,
    version     INTEGER      NOT NULL,
    action      VARCHAR(16)  NOT NULL, -- update / status / delete: what replaced this version
    actor_id    BIGINT,
    snapshot    JSONB        NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_desktop_skill_revisions_skill ON desktop_skill_revisions (skill_id, id DESC);
