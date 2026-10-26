-- 图片墙 (archive image wall): images newest first, often narrowed to a date range across all
-- users, which the (turn_id) and (user_id, id) indexes don't serve.
CREATE INDEX IF NOT EXISTS idx_conversation_archive_images_created_at ON conversation_archive_images (created_at);
