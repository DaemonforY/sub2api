-- Server-side image tools (background removal, super-resolution) for the canvas. Every successful
-- run is recorded; subscribers get a few free runs a day, the rest is charged to the balance.
CREATE TABLE IF NOT EXISTS image_tool_uses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT,
    tool VARCHAR(32) NOT NULL,
    free BOOLEAN NOT NULL DEFAULT FALSE,
    cost DECIMAL(20, 8) NOT NULL DEFAULT 0,
    input_bytes INTEGER NOT NULL DEFAULT 0,
    output_bytes INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_image_tool_uses_user_created ON image_tool_uses (user_id, created_at DESC);
