-- 卡密商品 (card-key goods, e.g. Muse): a product page sells one card key per order. The admin
-- imports the keys into stock; a paid order (order_type = 'goods', goods_id) takes the oldest
-- available key and shows it to the buyer. Keys are stored encrypted (AES-GCM, SecretEncryptor).
CREATE TABLE IF NOT EXISTS card_goods (
    id             BIGSERIAL     PRIMARY KEY,
    slug           VARCHAR(40)   NOT NULL UNIQUE,
    name           VARCHAR(60)   NOT NULL,
    subtitle       VARCHAR(120)  NOT NULL DEFAULT '',
    description    TEXT          NOT NULL DEFAULT '',
    -- Shown to the buyer next to the delivered key (how to activate / redeem).
    usage_note     TEXT          NOT NULL DEFAULT '',
    price          NUMERIC(12,2) NOT NULL,
    original_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    -- 0 = no limit on how many keys one user can buy.
    per_user_limit INTEGER       NOT NULL DEFAULT 0,
    enabled        BOOLEAN       NOT NULL DEFAULT FALSE,
    sort_order     INTEGER       NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE card_goods IS '卡密商品：付款后自动从库存发一条卡密';

-- status: available (in stock) | sold (delivered to order_id / user_id) | void (removed by the admin).
CREATE TABLE IF NOT EXISTS card_goods_items (
    id            BIGSERIAL    PRIMARY KEY,
    goods_id      BIGINT       NOT NULL REFERENCES card_goods(id) ON DELETE CASCADE,
    content_enc   TEXT         NOT NULL,
    -- sha256 of the plain key, so the same key is never imported twice.
    content_hash  CHAR(64)     NOT NULL,
    -- First and last characters, for the admin list (the key itself is only decrypted for its buyer).
    preview       VARCHAR(40)  NOT NULL DEFAULT '',
    status        VARCHAR(16)  NOT NULL DEFAULT 'available',
    order_id      BIGINT       UNIQUE,
    user_id       BIGINT       REFERENCES users(id) ON DELETE SET NULL,
    batch_note    VARCHAR(60)  NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    delivered_at  TIMESTAMPTZ,
    UNIQUE (goods_id, content_hash)
);

CREATE INDEX IF NOT EXISTS idx_card_goods_items_stock ON card_goods_items (goods_id, id) WHERE status = 'available';
CREATE INDEX IF NOT EXISTS idx_card_goods_items_user ON card_goods_items (user_id, delivered_at DESC) WHERE user_id IS NOT NULL;

COMMENT ON TABLE card_goods_items IS '卡密库存：available 在库 / sold 已发货 / void 已作废';

ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS goods_id BIGINT NULL;
CREATE INDEX IF NOT EXISTS idx_payment_orders_goods ON payment_orders (goods_id, status) WHERE goods_id IS NOT NULL;
