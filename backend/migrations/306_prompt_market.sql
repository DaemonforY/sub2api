-- 提示词交易: users sell a prompt or a pack (合集) of their own prompts for 余额; buyers pay from
-- their balance and the seller is credited the price minus the platform fee (no cash-out).
-- A listing snapshots the prompts when submitted; an admin reviews every new listing and every
-- change before it goes live (a change to a live listing waits in draft while the old one sells).

CREATE TABLE IF NOT EXISTS prompt_listings (
    id             BIGSERIAL     PRIMARY KEY,
    seller_user_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title          VARCHAR(60)   NOT NULL,
    description    TEXT          NOT NULL DEFAULT '',
    cover_url      TEXT          NOT NULL DEFAULT '',
    sample_urls    JSONB         NOT NULL DEFAULT '[]'::jsonb,
    -- In 余额 (USD), two decimals.
    price          NUMERIC(12,2) NOT NULL,
    item_count     INT           NOT NULL,
    -- pending / active / rejected / hidden (taken down by an admin) / withdrawn (by the seller)
    status         VARCHAR(16)   NOT NULL DEFAULT 'pending',
    review_note    VARCHAR(500)  NOT NULL DEFAULT '',
    -- Automatic flags of the content (nsfw / sensitive) for the reviewer.
    flags          TEXT[]        NOT NULL DEFAULT '{}',
    -- A change to an approved listing: {title, description, cover_url, sample_urls, price, items}.
    draft          JSONB,
    sales_count    INT           NOT NULL DEFAULT 0,
    published_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_listings_active ON prompt_listings (sales_count DESC, published_at DESC) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_prompt_listings_seller ON prompt_listings (seller_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_listings_review ON prompt_listings (updated_at) WHERE status = 'pending' OR draft IS NOT NULL;

-- The approved content of a listing (what buyers get).
CREATE TABLE IF NOT EXISTS prompt_listing_items (
    listing_id       BIGINT      NOT NULL REFERENCES prompt_listings(id) ON DELETE CASCADE,
    position         INT         NOT NULL,
    title            VARCHAR(60) NOT NULL,
    prompt           TEXT        NOT NULL,
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    kind             VARCHAR(8)  NOT NULL DEFAULT 'image',
    dedupe_key       VARCHAR(300) NOT NULL DEFAULT '',
    -- The seller's prompt it was copied from (informational).
    source_prompt_id BIGINT      REFERENCES prompt_items(id) ON DELETE SET NULL,
    PRIMARY KEY (listing_id, position)
);

CREATE INDEX IF NOT EXISTS idx_prompt_listing_items_dedupe ON prompt_listing_items (dedupe_key);

CREATE TABLE IF NOT EXISTS prompt_purchases (
    id             BIGSERIAL     PRIMARY KEY,
    listing_id     BIGINT        NOT NULL REFERENCES prompt_listings(id) ON DELETE CASCADE,
    buyer_user_id  BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_user_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price          NUMERIC(12,2) NOT NULL,
    fee_rate       NUMERIC(5,4)  NOT NULL,
    seller_amount  NUMERIC(12,4) NOT NULL,
    -- paid / refunded
    status         VARCHAR(16)   NOT NULL DEFAULT 'paid',
    refunded_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- One paid purchase per buyer and listing; buying again after a refund is allowed.
CREATE UNIQUE INDEX IF NOT EXISTS idx_prompt_purchases_once ON prompt_purchases (listing_id, buyer_user_id) WHERE status = 'paid';
CREATE INDEX IF NOT EXISTS idx_prompt_purchases_buyer ON prompt_purchases (buyer_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_purchases_seller ON prompt_purchases (seller_user_id, created_at DESC);

COMMENT ON TABLE prompt_listings IS '提示词交易：用户上架出售的提示词（单条或合集），用余额购买';
COMMENT ON TABLE prompt_listing_items IS '提示词交易：上架内容快照（审核通过的版本，买家拿到的内容）';
COMMENT ON TABLE prompt_purchases IS '提示词交易：购买记录（余额支付，卖家按扣除平台分成后的金额入余额）';
