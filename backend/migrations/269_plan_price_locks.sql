-- 老用户锁价: a member who paid for a plan keeps that plan's price on renewal while the
-- subscription doesn't lapse (expired no more than plan_price_lock_grace_days ago). Raising a
-- plan's price then only affects new buyers; a lower current price always wins.
CREATE TABLE IF NOT EXISTS plan_price_locks (
    user_id        BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id        BIGINT        NOT NULL,
    locked_price   DECIMAL(20,2) NOT NULL,
    first_order_id BIGINT,
    last_order_id  BIGINT,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plan_id)
);
CREATE INDEX IF NOT EXISTS idx_plan_price_locks_plan ON plan_price_locks (plan_id);

-- Everyone who already bought a plan is locked at its current price (no price has been raised yet).
INSERT INTO plan_price_locks (user_id, plan_id, locked_price, first_order_id, last_order_id)
SELECT o.user_id, o.plan_id, p.price, MIN(o.id), MAX(o.id)
FROM payment_orders o
JOIN subscription_plans p ON p.id = o.plan_id
JOIN users u ON u.id = o.user_id
WHERE o.status = 'COMPLETED' AND o.order_type = 'subscription' AND o.plan_id IS NOT NULL
GROUP BY o.user_id, o.plan_id, p.price
ON CONFLICT (user_id, plan_id) DO NOTHING;
