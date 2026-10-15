-- 注册试用去重 / 自动静默限制: each sign-up's browser (visitor_id from the register body) and IP prefix
-- (/24 for IPv4, /64 for IPv6 — never the full IP), and whether it got the sign-up default
-- subscription. SignupTrialGuard matches new sign-ups against it. No foreign key: deleting an account
-- must not make its fingerprint disappear.
CREATE TABLE IF NOT EXISTS signup_fingerprints (
    user_id        BIGINT       PRIMARY KEY,
    visitor_id     VARCHAR(64)  NOT NULL DEFAULT '',
    ip_prefix      VARCHAR(48)  NOT NULL DEFAULT '',
    trial_granted  BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
-- Both lookups (recent trial by browser; banned / admin-restricted account by browser or prefix)
-- filter on one column plus created_at.
CREATE INDEX IF NOT EXISTS idx_signup_fingerprints_visitor ON signup_fingerprints (visitor_id, created_at)
    WHERE visitor_id <> '';
CREATE INDEX IF NOT EXISTS idx_signup_fingerprints_ip ON signup_fingerprints (ip_prefix, created_at)
    WHERE ip_prefix <> '';

-- Backfill the last 90 days of sign-up trials so the guard knows them from day one: the browser
-- from user_attributions and the prefix of the user's first analytics event (sign-up itself was
-- never logged with an IP). Private, CGNAT and loopback prefixes are left out, as the guard does, and
-- so is IPv6: analytics keeps a /48, while sign-ups are matched by /64.
INSERT INTO signup_fingerprints (user_id, visitor_id, ip_prefix, trial_granted, created_at)
SELECT u.id,
       COALESCE(ua.visitor_id, ''),
       COALESCE((SELECT e.ip_prefix FROM analytics_events e
                 WHERE e.user_id = u.id AND e.ip_prefix <> ''
                   AND e.ip_prefix !~ '^(10\.|127\.|192\.168\.|172\.(1[6-9]|2[0-9]|3[01])\.|100\.(6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])\.)'
                   AND e.ip_prefix NOT LIKE '%:%'
                 ORDER BY e.created_at, e.id LIMIT 1), ''),
       TRUE,
       u.created_at
FROM users u
LEFT JOIN user_attributions ua ON ua.user_id = u.id
WHERE u.created_at >= NOW() - INTERVAL '90 days'
  AND EXISTS (SELECT 1 FROM user_subscriptions s
              WHERE s.user_id = u.id AND s.notes LIKE '%auto assigned by signup defaults%')
ON CONFLICT (user_id) DO NOTHING;
