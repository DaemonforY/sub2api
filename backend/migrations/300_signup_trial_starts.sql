-- 注册试用从第一次调用开始计时: the sign-up default subscription is first granted for the "unused"
-- window (e.g. 7 days) and recorded here; once the user's first request is billed to it,
-- SignupTrialStartService moves its term to [first call, first call + trial_days).
-- closed_reason: started (term moved) | changed (the subscription's expiry was changed by a purchase,
-- redeem code or admin before the first call, so it is left alone) | unused (window passed unused).
CREATE TABLE IF NOT EXISTS signup_trial_starts (
    subscription_id    BIGINT      PRIMARY KEY REFERENCES user_subscriptions(id) ON DELETE CASCADE,
    user_id            BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trial_days         INT         NOT NULL,
    pending_expires_at TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at         TIMESTAMPTZ,
    closed_reason      VARCHAR(16)
);
CREATE INDEX IF NOT EXISTS idx_signup_trial_starts_open ON signup_trial_starts (created_at) WHERE closed_reason IS NULL;
CREATE INDEX IF NOT EXISTS idx_signup_trial_starts_user ON signup_trial_starts (user_id);
