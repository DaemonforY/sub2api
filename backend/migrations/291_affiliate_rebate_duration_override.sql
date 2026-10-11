-- 博主合作: per-inviter rebate validity (days after the invitee signs up; 0 = forever, NULL = the
-- global affiliate_rebate_duration_days), so a partner can be offered longer terms than everyone.
ALTER TABLE user_affiliates ADD COLUMN IF NOT EXISTS aff_rebate_duration_days INTEGER;
COMMENT ON COLUMN user_affiliates.aff_rebate_duration_days IS '专属返利有效期（天，0 表示永久，NULL 表示沿用全局）';
