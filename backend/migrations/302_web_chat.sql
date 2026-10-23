-- AI 对话 (/chat): a web chat page on the user's own API keys. Conversations and messages are kept
-- on the server so they follow the user across devices; pictures (uploaded or drawn) are files
-- under the chat media dir, listed in chat_files.
CREATE TABLE IF NOT EXISTS chat_conversations (
    id            BIGSERIAL     PRIMARY KEY,
    user_id       BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         VARCHAR(100)  NOT NULL DEFAULT '',
    system_prompt TEXT          NOT NULL DEFAULT '',
    -- The key and model last used, so reopening the conversation picks them again.
    api_key_id    BIGINT,
    model         VARCHAR(100)  NOT NULL DEFAULT '',
    pinned        BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_conversations_user ON chat_conversations (user_id, pinned DESC, updated_at DESC);

COMMENT ON TABLE chat_conversations IS 'AI 对话：用户在网页上用自己的 Key 聊天的会话';

-- role: user | assistant. kind: text | image (an image-mode turn: the prompt and the drawn picture).
-- status: done | stopped (the user pressed 停止, content is what arrived) | error.
CREATE TABLE IF NOT EXISTS chat_messages (
    id                BIGSERIAL     PRIMARY KEY,
    conversation_id   BIGINT        NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    role              VARCHAR(16)   NOT NULL,
    kind              VARCHAR(16)   NOT NULL DEFAULT 'text',
    content           TEXT          NOT NULL DEFAULT '',
    reasoning         TEXT          NOT NULL DEFAULT '',
    -- Picture file names (chat_files.name), in order.
    files             JSONB         NOT NULL DEFAULT '[]',
    model             VARCHAR(100)  NOT NULL DEFAULT '',
    status            VARCHAR(16)   NOT NULL DEFAULT 'done',
    error             TEXT          NOT NULL DEFAULT '',
    prompt_tokens     INTEGER       NOT NULL DEFAULT 0,
    completion_tokens INTEGER       NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation ON chat_messages (conversation_id, id);

-- message_id is NULL from upload until the picture is sent; unsent uploads are swept after a day.
CREATE TABLE IF NOT EXISTS chat_files (
    name        VARCHAR(64)   PRIMARY KEY,
    user_id     BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_id  BIGINT        REFERENCES chat_messages(id) ON DELETE CASCADE,
    mime        VARCHAR(32)   NOT NULL,
    size        INTEGER       NOT NULL,
    width       INTEGER       NOT NULL DEFAULT 0,
    height      INTEGER       NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_files_user ON chat_files (user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_chat_files_unsent ON chat_files (created_at) WHERE message_id IS NULL;
