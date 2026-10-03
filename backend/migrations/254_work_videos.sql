-- Video works (kind = 'video'): the clip lives beside the work; its cover is work_media position 0.
ALTER TABLE works ADD COLUMN IF NOT EXISTS video_file VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE works ADD COLUMN IF NOT EXISTS video_mime VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE works ADD COLUMN IF NOT EXISTS video_size BIGINT NOT NULL DEFAULT 0;
ALTER TABLE works ADD COLUMN IF NOT EXISTS video_duration_ms INTEGER NOT NULL DEFAULT 0;
