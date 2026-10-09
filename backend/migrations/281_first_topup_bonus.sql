-- 首充奖励: the bonus for a user's first gateway-paid balance top-up is recorded in the affiliate
-- ledger as action 'first_topup_bonus' (source_order_id = the order). At most one per user,
-- enforced here so concurrent or retried fulfillments cannot double-grant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_first_topup_once
    ON user_affiliate_ledger (user_id)
    WHERE action = 'first_topup_bonus';
