-- 注册试用去重: each sign-up's browser (visitor_id from the register body) and IP prefix (/24 for
-- IPv4, /48 for IPv6 — never the full IP), and whether it got the sign-up default subscription.
-- A new sign-up whose visitor_id or prefix got one within growth_signup_trial_dedup_days gets only
-- the default balance (SignupTrialGuard). No foreign key: deleting an account must not reopen the trial.
CREATE TABLE IF NOT EXISTS signup_fingerprints (
    user_id        BIGINT       PRIMARY KEY,
    visitor_id     VARCHAR(64)  NOT NULL DEFAULT '',
    ip_prefix      VARCHAR(48)  NOT NULL DEFAULT '',
    trial_granted  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_signup_fingerprints_visitor ON signup_fingerprints (visitor_id, created_at)
    WHERE visitor_id <> '' AND trial_granted;
CREATE INDEX IF NOT EXISTS idx_signup_fingerprints_ip ON signup_fingerprints (ip_prefix, created_at)
    WHERE ip_prefix <> '' AND trial_granted;

-- Backfill the last 90 days of sign-up trials so the guard knows them from day one: the browser
-- from user_attributions and the prefix of the user's first analytics event (sign-up itself was
-- never logged with an IP). Private / loopback prefixes are left out, as the guard does.
INSERT INTO signup_fingerprints (user_id, visitor_id, ip_prefix, trial_granted, created_at)
SELECT u.id,
       COALESCE(ua.visitor_id, ''),
       COALESCE((SELECT e.ip_prefix FROM analytics_events e
                 WHERE e.user_id = u.id AND e.ip_prefix <> ''
                   AND e.ip_prefix !~ '^(10\.|127\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.|::|f[cd]|fe80)'
                 ORDER BY e.created_at, e.id LIMIT 1), ''),
       TRUE,
       u.created_at
FROM users u
LEFT JOIN user_attributions ua ON ua.user_id = u.id
WHERE u.created_at >= NOW() - INTERVAL '90 days'
  AND EXISTS (SELECT 1 FROM user_subscriptions s
              WHERE s.user_id = u.id AND s.notes LIKE '%auto assigned by signup defaults%')
ON CONFLICT (user_id) DO NOTHING;
