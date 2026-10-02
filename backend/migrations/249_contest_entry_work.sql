-- Contest entries made from a canvas community work link back to it (the work page shows its
-- contests, the entry links to the work). One live entry per work and contest.
ALTER TABLE contest_entries ADD COLUMN IF NOT EXISTS work_id BIGINT NULL REFERENCES works(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_contest_entries_contest_work
    ON contest_entries (contest_id, work_id)
    WHERE work_id IS NOT NULL AND status NOT IN ('withdrawn', 'rejected');

CREATE INDEX IF NOT EXISTS idx_contest_entries_work ON contest_entries (work_id) WHERE work_id IS NOT NULL;
