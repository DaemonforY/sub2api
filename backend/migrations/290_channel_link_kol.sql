-- 博主合作: a channel link can carry a promo code (?promo=) so the blogger's audience gets the
-- exclusive sign-up bonus without typing it.
ALTER TABLE channel_links ADD COLUMN IF NOT EXISTS promo_code VARCHAR(32) NOT NULL DEFAULT '';
