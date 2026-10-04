-- AI 学习 L3: the learner showcase wall. A certificate holder chooses to show their project on the
-- wall (showcase); admins can take an entry off the wall (showcase_hidden).
ALTER TABLE learn_certificates ADD COLUMN IF NOT EXISTS showcase BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE learn_certificates ADD COLUMN IF NOT EXISTS showcase_hidden BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_learn_certificates_showcase ON learn_certificates (issued_at DESC)
    WHERE showcase AND NOT showcase_hidden AND revoked_at IS NULL;
