-- 内容运营 V2 写作台 (docs/CONTENT_ENGINE_V2.md §2). Desk articles live in content_items (status
-- 'desk', state in data->'desk'); this adds the owner's 作者档案 and the text snapshots.

-- 作者档案: who the owner is, what they really did, their stances and how they write, plus up to
-- five of their own articles as samples. Every 写作台 prompt writes as this person.
CREATE TABLE IF NOT EXISTS content_personas (
    owner_user_id  BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    data           JSONB        NOT NULL DEFAULT '{}'::jsonb,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Snapshots of a desk article's text: each AI hand-over (chosen draft, merge), manual saves,
-- restores and the final text.
CREATE TABLE IF NOT EXISTS content_item_versions (
    id             BIGSERIAL    PRIMARY KEY,
    item_id        BIGINT       NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    owner_user_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind           VARCHAR(16)  NOT NULL,
    label          VARCHAR(100) NOT NULL DEFAULT '',
    markdown       TEXT         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_content_item_versions_item ON content_item_versions (item_id, id DESC);

COMMENT ON TABLE content_personas IS '内容运营写作台：作者档案（身份、真实经历、立场、表达习惯、范文）';
COMMENT ON TABLE content_item_versions IS '内容运营写作台：稿子正文的版本快照';
