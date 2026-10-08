-- One "get started" reminder email per user (ActivationReminderService); a row means it was sent
-- (or skipped for a placeholder address) and is never sent again.
CREATE TABLE IF NOT EXISTS activation_reminders (
    user_id BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
