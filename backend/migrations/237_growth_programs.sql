-- Growth programs: education (school email) verification and the invitee first-order bonus.

-- A user proves ownership of a school email (e.g. *.edu.cn) with an emailed code.
-- Independent of the login email, so students who registered with QQ mail can verify too.
-- One school email can verify only one account.
CREATE TABLE IF NOT EXISTS user_edu_verifications (
    user_id     BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email       VARCHAR(255) NOT NULL,
    verified_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_edu_verifications_email
    ON user_edu_verifications (LOWER(email));

COMMENT ON TABLE user_edu_verifications IS '教育邮箱认证（学生/教师）';

-- The invitee first-order bonus is recorded in the affiliate ledger as action 'invitee_bonus'
-- (user_id = invitee who received it, source_user_id = inviter, source_order_id = the order).
-- At most one bonus per invited user, enforced here so concurrent fulfillments cannot double-grant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_invitee_bonus_once
    ON user_affiliate_ledger (user_id)
    WHERE action = 'invitee_bonus';
