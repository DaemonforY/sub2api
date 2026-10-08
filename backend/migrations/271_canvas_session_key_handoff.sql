-- Signing the canvas in by full-page redirect (the WeChat in-app browser cannot use the popup):
-- the main site's connect page stores the API key the user picked on the new canvas session, and
-- the canvas takes it once through GET /api/v1/canvas/connect-key (the key never goes in a URL).
ALTER TABLE canvas_sessions ADD COLUMN IF NOT EXISTS handoff_api_key_id BIGINT;
