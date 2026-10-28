-- 灵感库 (admin): a read-only corpus of public-domain poems to browse and search, and song cards with
-- lyrics the admin pastes in for internal reference only.

-- Loaded with tools/poetry_corpus (chinese-poetry + Werneror/Poetry, both MIT); never edited here
-- apart from starred.
CREATE TABLE IF NOT EXISTS lib_poems (
    id          BIGSERIAL PRIMARY KEY,
    collection  VARCHAR(16) NOT NULL,           -- tang / song_shi / song_ci / yuanqu / shijing / chuci / lidai …
    title       TEXT        NOT NULL,
    author      TEXT        NOT NULL,
    dynasty     VARCHAR(16) NOT NULL DEFAULT '',
    content     TEXT        NOT NULL,           -- one line per row
    notes       TEXT        NOT NULL DEFAULT '',
    tags        TEXT[]      NOT NULL DEFAULT '{}',
    famous      BOOLEAN     NOT NULL DEFAULT FALSE, -- in 唐诗三百首 / 宋词三百首 / 千家诗 / 古文观止
    popularity  REAL        NOT NULL DEFAULT 0,     -- log10 of web hits, +3 when famous
    starred     BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_lib_poems_order ON lib_poems (famous DESC, popularity DESC, id);
CREATE INDEX IF NOT EXISTS idx_lib_poems_author ON lib_poems (author);
CREATE INDEX IF NOT EXISTS idx_lib_poems_starred ON lib_poems (id) WHERE starred;

-- Lyrics are copyrighted: they stay in this table for the admin to read and search. They are never
-- served to non-admins, exported, or put into prompts, and LyricsGuard refuses work texts, scenes and
-- prompts that contain a lyric line.
CREATE TABLE IF NOT EXISTS lib_songs (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT        NOT NULL,
    artist      TEXT        NOT NULL DEFAULT '',
    lyricist    TEXT        NOT NULL DEFAULT '',
    composer    TEXT        NOT NULL DEFAULT '',
    year        INT,
    album       TEXT        NOT NULL DEFAULT '',
    lyrics      TEXT        NOT NULL DEFAULT '',
    summary     TEXT        NOT NULL DEFAULT '', -- 意境概括, in our own words
    imagery     TEXT[]      NOT NULL DEFAULT '{}',
    moods       TEXT[]      NOT NULL DEFAULT '{}',
    tags        TEXT[]      NOT NULL DEFAULT '{}',
    allusions   JSONB       NOT NULL DEFAULT '[]', -- [{title, author, poem_id?, note}] 化用的古诗
    notes       TEXT        NOT NULL DEFAULT '',
    link_url    TEXT        NOT NULL DEFAULT '',
    starred     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_by  BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lib_songs_artist ON lib_songs (artist);

-- Trigram indexes for keyword search when pg_trgm is available (best effort, like 065).
DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'pg_trgm extension not created: %', SQLERRM;
    END;
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_lib_poems_text_trgm ON lib_poems USING gin ((title || '' '' || author || '' '' || content) gin_trgm_ops)';
    END IF;
END
$$;
