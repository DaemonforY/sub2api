-- 对话归档 (conversation archive): when the admin enables it, every API conversation is stored as
-- sent (gzip JSON, one object per request, deduped by session) plus the images the models
-- generate, in the operator's private S3-compatible bucket for retention_days (default 180).
-- These tables are only the index; the content lives in the bucket. Only admin endpoints read
-- them, and every browse is written to content_log_access_logs ('archive_view'); settings
-- changes are written as 'archive_config'.
CREATE TABLE IF NOT EXISTS conversation_archive_turns (
    id                BIGSERIAL   PRIMARY KEY,
    user_id           BIGINT      NOT NULL,
    api_key_id        BIGINT,
    group_id          BIGINT,
    session_key       TEXT        NOT NULL,
    request_id        TEXT        NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL,
    model             TEXT        NOT NULL DEFAULT '',
    inbound_endpoint  TEXT        NOT NULL DEFAULT '',
    user_agent        TEXT        NOT NULL DEFAULT '',
    ip_address        TEXT        NOT NULL DEFAULT '',
    input_tokens      INT         NOT NULL DEFAULT 0,
    output_tokens     INT         NOT NULL DEFAULT 0,
    cache_read_tokens INT         NOT NULL DEFAULT 0,
    object_key        TEXT        NOT NULL,
    bytes             INT         NOT NULL DEFAULT 0,
    new_item_count    INT         NOT NULL DEFAULT 0,
    reset             BOOLEAN     NOT NULL DEFAULT FALSE,
    image_count       INT         NOT NULL DEFAULT 0,
    preview           TEXT        NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_turns_user ON conversation_archive_turns (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_turns_session ON conversation_archive_turns (user_id, session_key, created_at, id);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_turns_created_at ON conversation_archive_turns (created_at);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_turns_request_id ON conversation_archive_turns (request_id);

CREATE TABLE IF NOT EXISTS conversation_archive_images (
    id          BIGSERIAL   PRIMARY KEY,
    turn_id     BIGINT      NOT NULL REFERENCES conversation_archive_turns(id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL,
    object_key  TEXT        NOT NULL,
    mime        TEXT        NOT NULL,
    bytes       INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_images_turn ON conversation_archive_images (turn_id);
CREATE INDEX IF NOT EXISTS idx_conversation_archive_images_user ON conversation_archive_images (user_id, id DESC);

-- Self-service account deletion records the prefixes to sweep here before the account goes; the
-- archive's purge job deletes {prefix}conv/{user_id}/ and {prefix}img/{user_id}/ for each and
-- removes the row when done (retried while the storage fails or isn't configured).
CREATE TABLE IF NOT EXISTS conversation_archive_purges (
    user_id     BIGINT      PRIMARY KEY,
    prefixes    TEXT[]      NOT NULL DEFAULT '{}',
    attempts    INT         NOT NULL DEFAULT 0,
    last_error  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The shared access trail gains the archive actions.
ALTER TABLE content_log_access_logs DROP CONSTRAINT IF EXISTS content_log_access_logs_action_check;
ALTER TABLE content_log_access_logs ADD CONSTRAINT content_log_access_logs_action_check
    CHECK (action IN ('enable', 'disable', 'view', 'config', 'archive_view', 'archive_config'));
