-- 邀请返利提现：用户申请 → 立即从可用返利中扣出（流水 action = 'withdraw'）→ 管理员线下打款后标记已打款，
-- 或驳回 / 用户撤销时退回返利（流水 action = 'withdraw_return'）。
-- 只有被邀请人真实付费订单（非余额支付、已完成）产生、且已过冻结期的返利可以提现，按订单实付（不含手续费）折算人民币。
CREATE TABLE IF NOT EXISTS affiliate_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    quota_amount DECIMAL(20,8) NOT NULL,
    cny_amount DECIMAL(20,2) NOT NULL,
    method VARCHAR(16) NOT NULL,
    -- 收款账号与实名用 AES-GCM 加密保存（与 TOTP 同一密钥）
    account_enc TEXT NOT NULL,
    real_name_enc TEXT NOT NULL,
    user_note VARCHAR(200) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    admin_note VARCHAR(500) NOT NULL DEFAULT '',
    reviewed_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT affiliate_withdrawals_status_check CHECK (status IN ('pending', 'paid', 'rejected', 'cancelled')),
    CONSTRAINT affiliate_withdrawals_method_check CHECK (method IN ('alipay', 'wechat')),
    CONSTRAINT affiliate_withdrawals_amount_check CHECK (quota_amount > 0 AND cny_amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_affiliate_withdrawals_user ON affiliate_withdrawals (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_affiliate_withdrawals_status ON affiliate_withdrawals (status, created_at DESC);
-- 每人同时只能有一笔待处理的提现
CREATE UNIQUE INDEX IF NOT EXISTS idx_affiliate_withdrawals_one_pending
    ON affiliate_withdrawals (user_id) WHERE status = 'pending';

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS withdrawal_id BIGINT NULL REFERENCES affiliate_withdrawals(id) ON DELETE SET NULL;

COMMENT ON TABLE affiliate_withdrawals IS '邀请返利提现申请（人工打款）';
COMMENT ON COLUMN affiliate_withdrawals.quota_amount IS '扣除的返利额度（与 aff_quota 同单位）';
COMMENT ON COLUMN affiliate_withdrawals.cny_amount IS '应打款金额（人民币）';
COMMENT ON COLUMN user_affiliate_ledger.withdrawal_id IS 'withdraw / withdraw_return 流水对应的提现申请';
