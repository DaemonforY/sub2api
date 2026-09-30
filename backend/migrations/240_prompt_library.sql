-- Prompt library shared by the canvas (canvas.<domain>): community sources synced by the server,
-- prompts users save for themselves or share, admin curation (scenes / tags / visibility) and
-- per-user usage that drives the "most used" order and "for you" recommendations.

CREATE TABLE IF NOT EXISTS prompt_sources (
    id             VARCHAR(64)  PRIMARY KEY,
    name           VARCHAR(128) NOT NULL,
    -- registry: JSON array in the yukkcat/image-prompts format; youmind: YouMind references manifest.
    format         VARCHAR(16)  NOT NULL DEFAULT 'registry',
    url            TEXT         NOT NULL,
    homepage       TEXT         NOT NULL DEFAULT '',
    enabled        BOOLEAN      NOT NULL DEFAULT TRUE,
    item_count     INT          NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMPTZ,
    last_error     TEXT         NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

INSERT INTO prompt_sources (id, name, format, url, homepage) VALUES
    ('banana-prompt-quicker', 'Banana Prompt Quicker', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/banana-prompt-quicker.json', 'https://glidea.github.io/banana-prompt-quicker/'),
    ('davidwu-gpt-image2-prompts', 'DavidWu GPT Image 2', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/davidwu-gpt-image2-prompts.json', 'https://github.com/davidwuw0811-boop/awesome-gpt-image2-prompts'),
    ('freestylefly-gpt-image-2', 'Freestylefly GPT Image 2', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/freestylefly-gpt-image-2.json', 'https://github.com/freestylefly/awesome-gpt-image-2'),
    ('awesome-gpt-image', 'Awesome GPT Image', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/awesome-gpt-image.json', 'https://github.com/ZeroLu/awesome-gpt-image'),
    ('awesome-gpt4o-image-prompts', 'Awesome GPT-4o', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/awesome-gpt4o-image-prompts.json', 'https://github.com/ImgEdify/Awesome-GPT4o-Image-Prompts'),
    ('youmind-gpt-image-2', 'YouMind GPT Image 2', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/youmind-gpt-image-2.json', 'https://github.com/YouMind-OpenLab/awesome-gpt-image-2'),
    ('youmind-nano-banana-pro', 'YouMind Nano Banana Pro', 'registry', 'https://raw.githubusercontent.com/yukkcat/image-prompts/main/dist/sources/youmind-nano-banana-pro.json', 'https://github.com/YouMind-OpenLab/awesome-nano-banana-pro-prompts'),
    ('youmind-ai-image-prompts', 'YouMind AI Image Prompts', 'youmind', 'https://raw.githubusercontent.com/YouMind-OpenLab/ai-image-prompts-skill/main/references/manifest.json', 'https://github.com/YouMind-OpenLab/ai-image-prompts-skill')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS prompt_items (
    id                   BIGSERIAL    PRIMARY KEY,
    -- Source id, or 'user' for prompts saved by users.
    source_id            VARCHAR(64)  NOT NULL,
    external_id          VARCHAR(160) NOT NULL,
    owner_user_id        BIGINT       REFERENCES users(id) ON DELETE CASCADE,
    kind                 VARCHAR(8)   NOT NULL DEFAULT 'image',
    title                VARCHAR(200) NOT NULL,
    prompt               TEXT         NOT NULL,
    description          TEXT         NOT NULL DEFAULT '',
    cover_url            TEXT         NOT NULL DEFAULT '',
    reference_image_urls JSONB        NOT NULL DEFAULT '[]'::jsonb,
    source_tags          JSONB        NOT NULL DEFAULT '[]'::jsonb,
    scenes               TEXT[]       NOT NULL DEFAULT '{}',
    tags                 TEXT[]       NOT NULL DEFAULT '{}',
    model                VARCHAR(32)  NOT NULL DEFAULT 'unknown',
    lang                 VARCHAR(4)   NOT NULL DEFAULT 'en',
    needs_reference      BOOLEAN      NOT NULL DEFAULT FALSE,
    -- Automatic flags (nsfw / sensitive); flagged items are hidden until an admin decides.
    auto_flags           TEXT[]       NOT NULL DEFAULT '{}',
    author               VARCHAR(200) NOT NULL DEFAULT '',
    source_url           TEXT         NOT NULL DEFAULT '',
    -- public / private (user prompts only; source prompts are public).
    visibility           VARCHAR(16)  NOT NULL DEFAULT 'public',
    -- active / hidden / pending (shared, awaiting review) / rejected / duplicate.
    status               VARCHAR(16)  NOT NULL DEFAULT 'active',
    review_note          VARCHAR(500) NOT NULL DEFAULT '',
    -- Set once an admin edits the item: source syncs then keep its title, scenes, tags and status.
    curated              BOOLEAN      NOT NULL DEFAULT FALSE,
    featured             BOOLEAN      NOT NULL DEFAULT FALSE,
    quality_score        INT          NOT NULL DEFAULT 0,
    use_count            BIGINT       NOT NULL DEFAULT 0,
    favorite_count       BIGINT       NOT NULL DEFAULT 0,
    dedupe_key           VARCHAR(300) NOT NULL DEFAULT '',
    -- Hash of the synced fields: unchanged source items are not rewritten on every sync.
    sync_hash            VARCHAR(64)  NOT NULL DEFAULT '',
    -- Lower-cased title, description, tags and the start of the prompt (kept by a trigger) for keyword search.
    search_text          TEXT         NOT NULL DEFAULT '',
    published_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_prompt_items_public_popular ON prompt_items (use_count DESC, favorite_count DESC, quality_score DESC, id DESC) WHERE status = 'active' AND visibility = 'public';
CREATE INDEX IF NOT EXISTS idx_prompt_items_scenes ON prompt_items USING GIN (scenes);
CREATE INDEX IF NOT EXISTS idx_prompt_items_owner ON prompt_items (owner_user_id, updated_at DESC) WHERE owner_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_prompt_items_dedupe ON prompt_items (dedupe_key);
CREATE INDEX IF NOT EXISTS idx_prompt_items_status_source ON prompt_items (status, source_id);

CREATE OR REPLACE FUNCTION prompt_items_search_text() RETURNS trigger AS $$
BEGIN
    NEW.search_text := lower(
        NEW.title || ' ' || NEW.description || ' ' || array_to_string(NEW.tags, ' ') || ' ' ||
        COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(NEW.source_tags)), '') || ' ' ||
        left(NEW.prompt, 800));
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prompt_items_search_text ON prompt_items;
CREATE TRIGGER trg_prompt_items_search_text
    BEFORE INSERT OR UPDATE OF title, description, tags, source_tags, prompt ON prompt_items
    FOR EACH ROW EXECUTE FUNCTION prompt_items_search_text();

-- Trigram index for keyword search when pg_trgm is available (best effort, like 065).
DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'pg_trgm extension not created: %', SQLERRM;
    END;
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_prompt_items_search_trgm ON prompt_items USING gin (search_text gin_trgm_ops)';
    END IF;
END
$$;

-- One row per (user, prompt): how often the user drew with / copied it and whether it is a favorite.
CREATE TABLE IF NOT EXISTS prompt_item_uses (
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id      BIGINT      NOT NULL REFERENCES prompt_items(id) ON DELETE CASCADE,
    uses         INT         NOT NULL DEFAULT 0,
    favorited    BOOLEAN     NOT NULL DEFAULT FALSE,
    last_used_at TIMESTAMPTZ,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_prompt_item_uses_user_updated ON prompt_item_uses (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_prompt_item_uses_item ON prompt_item_uses (item_id);

-- Cover images users upload for their own prompts (public files on the data volume).
CREATE TABLE IF NOT EXISTS prompt_covers (
    file       VARCHAR(64) PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    size_bytes BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_covers_user_created ON prompt_covers (user_id, created_at DESC);

COMMENT ON TABLE prompt_sources IS '提示词库的社区来源（服务器定时同步）';
COMMENT ON TABLE prompt_items IS '提示词库条目：社区来源 + 用户保存/分享的提示词，含人工标注与使用次数';
COMMENT ON TABLE prompt_item_uses IS '用户对提示词的使用与收藏记录（用于排序和猜你喜欢）';
COMMENT ON TABLE prompt_covers IS '用户为自己的提示词上传的封面图';
