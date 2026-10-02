-- Comments on community works: top-level comments and one level of replies (parent_id is the
-- top-level comment; reply_to_id the comment answered, which may itself be a reply).
CREATE TABLE IF NOT EXISTS work_comments (
    id               BIGSERIAL    PRIMARY KEY,
    work_id          BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id        BIGINT       NULL REFERENCES work_comments(id) ON DELETE CASCADE,
    reply_to_id      BIGINT       NULL REFERENCES work_comments(id) ON DELETE SET NULL,
    reply_to_user_id BIGINT       NULL REFERENCES users(id) ON DELETE SET NULL,
    body             VARCHAR(500) NOT NULL,
    -- approved | pending (review) | hidden (by an admin) | deleted (by its author or the work's author)
    status           VARCHAR(16)  NOT NULL DEFAULT 'approved',
    review_flags     TEXT[]       NOT NULL DEFAULT '{}',
    -- Approved replies (top-level comments only).
    reply_count      INTEGER      NOT NULL DEFAULT 0,
    report_count     INTEGER      NOT NULL DEFAULT 0,
    -- Kept for accountability, never shown.
    ip               VARCHAR(64)  NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_work_comments_work ON work_comments (work_id, created_at DESC) WHERE parent_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_work_comments_parent ON work_comments (parent_id, created_at) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_work_comments_user ON work_comments (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_work_comments_review ON work_comments (created_at DESC) WHERE status = 'pending' OR report_count > 0;

-- Approved comments and replies; the author may close a work's comments.
ALTER TABLE works ADD COLUMN IF NOT EXISTS comment_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE works ADD COLUMN IF NOT EXISTS comments_closed BOOLEAN NOT NULL DEFAULT FALSE;

-- Reports may be about a comment of the work.
ALTER TABLE work_reports ADD COLUMN IF NOT EXISTS comment_id BIGINT NULL REFERENCES work_comments(id) ON DELETE CASCADE;
