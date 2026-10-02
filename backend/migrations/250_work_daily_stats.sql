-- Daily views and remixes per work, for the creator stats page (likes, favorites and follows
-- carry their own timestamps). Days are China Standard Time dates.
CREATE TABLE IF NOT EXISTS work_daily_stats (
    work_id  BIGINT  NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    day      DATE    NOT NULL,
    views    INTEGER NOT NULL DEFAULT 0,
    remixes  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (work_id, day)
);

CREATE INDEX IF NOT EXISTS idx_work_daily_stats_day ON work_daily_stats (day);
CREATE INDEX IF NOT EXISTS idx_work_likes_work_created ON work_likes (work_id, created_at);
CREATE INDEX IF NOT EXISTS idx_work_favorites_work_created ON work_favorites (work_id, created_at);
CREATE INDEX IF NOT EXISTS idx_follows_followee_created ON follows (followee_id, created_at);
