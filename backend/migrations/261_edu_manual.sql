-- Education verification granted by an admin, for teachers and students whose school has no
-- school email. method: 'email' (code sent to a school email) or 'manual' (admin, with a note).
ALTER TABLE user_edu_verifications
    ADD COLUMN IF NOT EXISTS method     VARCHAR(16)  NOT NULL DEFAULT 'email',
    ADD COLUMN IF NOT EXISTS note       VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS granted_by BIGINT       NULL;

COMMENT ON COLUMN user_edu_verifications.method IS 'email=学校邮箱验证码；manual=管理员手动认证';
COMMENT ON COLUMN user_edu_verifications.note IS '手动认证说明（学校、身份、核验方式）';
COMMENT ON COLUMN user_edu_verifications.granted_by IS '手动认证的管理员用户 ID';
