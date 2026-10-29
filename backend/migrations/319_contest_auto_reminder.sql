-- 每月自动比赛: the last month whose "about to be published" reminder email went out.
ALTER TABLE contest_auto_settings ADD COLUMN IF NOT EXISTS reminded_period VARCHAR(16) NOT NULL DEFAULT '';
