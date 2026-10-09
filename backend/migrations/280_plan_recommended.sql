-- 推荐套餐: one or more plans can be highlighted on the purchase and pricing pages.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS recommended BOOLEAN NOT NULL DEFAULT FALSE;
