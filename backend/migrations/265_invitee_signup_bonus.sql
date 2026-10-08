-- Invitee sign-up trial credit: someone who signs up with an invite gets a little balance to try the
-- API with (growth_invitee_signup_bonus, off at 0). Recorded in the affiliate ledger as action
-- 'invitee_signup_bonus', at most once per user.
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_invitee_signup_bonus_once
    ON user_affiliate_ledger (user_id)
    WHERE action = 'invitee_signup_bonus';
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_signup_bonus_day
    ON user_affiliate_ledger (created_at)
    WHERE action = 'invitee_signup_bonus';
