-- 我的提示词: users keep their own prompts in folders and pin the ones they use most.
-- Deleting a folder moves its prompts back to 未分类 unless the user asked to delete them too.

CREATE TABLE IF NOT EXISTS prompt_folders (
    id         BIGSERIAL   PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(40) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_prompt_folders_user_name ON prompt_folders (user_id, lower(name));

ALTER TABLE prompt_items ADD COLUMN IF NOT EXISTS folder_id BIGINT REFERENCES prompt_folders(id) ON DELETE SET NULL;
ALTER TABLE prompt_items ADD COLUMN IF NOT EXISTS pinned BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_prompt_items_owner_folder ON prompt_items (owner_user_id, folder_id) WHERE owner_user_id IS NOT NULL;

COMMENT ON TABLE prompt_folders IS '用户给自己的提示词建的文件夹';
COMMENT ON COLUMN prompt_items.folder_id IS '用户提示词所在文件夹（空 = 未分类）';
COMMENT ON COLUMN prompt_items.pinned IS '用户置顶的提示词（「我的」里排在最前）';
