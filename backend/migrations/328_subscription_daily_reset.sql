-- 订阅「重置今日额度」: a user pays (daily limit × a set rate) to clear today's usage. From the reset
-- until the next daily rollover (00:00 in the configured timezone) usage counts toward the daily
-- window only — not the weekly or monthly windows — and only the daily limit is checked.
-- daily_reset_until: end of the current reset day (NULL or past = no reset active).
-- daily_reset_count: resets bought within that reset day (stale once daily_reset_until has passed).
ALTER TABLE user_subscriptions
    ADD COLUMN IF NOT EXISTS daily_reset_until TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS daily_reset_count INTEGER NOT NULL DEFAULT 0;
