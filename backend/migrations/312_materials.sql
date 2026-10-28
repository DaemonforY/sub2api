-- 诗画素材库 phase 1 (docs/MATERIAL_LIBRARY_PLAN.md): texts are split into segments, segments are
-- tagged with imagery, and every drawing records which segment, style and imagery it used.
--
-- Copyright is enforced here, not only in code: a text whose license_status is reference_only (popular
-- lyrics, authors dead fewer than 50 years) keeps no original words — mat_texts.body must be empty and
-- a trigger refuses segment content for it. Songs are always reference_only.
--
-- mat_authors          poets / lyricists, death_year drives the suggested license
-- mat_texts            one poem, song or set of original lines
-- mat_text_segments    the part a picture is drawn from (a couplet, a stanza, a chorus)
-- mat_imagery          an image word (孤烟, 柳, 孤灯) and how to paint it
-- mat_segment_imagery  which imagery a segment carries (AI drafts, an admin confirms)
-- mat_styles           painting styles appended to prompts (9 seeded below)
-- mat_artworks         one drawing and everything that went into it
-- mat_artwork_imagery  which imagery a drawing used
-- mat_artwork_renders  finished posters of approved drawings, one per aspect ratio
-- mat_blocklist        terms checked before a prompt is sent (artists, studios, asking for text)
CREATE TABLE IF NOT EXISTS mat_authors (
    id          BIGSERIAL   PRIMARY KEY,
    name        VARCHAR(64) NOT NULL,
    aliases     JSONB       NOT NULL DEFAULT '[]',
    dynasty     VARCHAR(16) NOT NULL DEFAULT '',
    birth_year  INT,
    death_year  INT,
    bio         TEXT        NOT NULL DEFAULT '',
    kind        VARCHAR(16) NOT NULL DEFAULT 'poet' CHECK (kind IN ('poet', 'lyricist', 'writer', 'platform')),
    created_by  BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (name, dynasty)
);

CREATE TABLE IF NOT EXISTS mat_texts (
    id             BIGSERIAL    PRIMARY KEY,
    slug           VARCHAR(64)  NOT NULL UNIQUE,
    kind           VARCHAR(16)  NOT NULL CHECK (kind IN ('shi', 'ci', 'qu', 'fu', 'wen', 'modern_poem', 'song', 'original')),
    title          VARCHAR(128) NOT NULL,
    author_id      BIGINT       REFERENCES mat_authors(id) ON DELETE SET NULL,
    performer      VARCHAR(64)  NOT NULL DEFAULT '',
    body           TEXT         NOT NULL DEFAULT '',
    translation    TEXT         NOT NULL DEFAULT '',
    appreciation   TEXT         NOT NULL DEFAULT '',
    license_status VARCHAR(16)  NOT NULL CHECK (license_status IN ('public_domain', 'original', 'licensed', 'reference_only')),
    license_note   TEXT         NOT NULL DEFAULT '',
    sources        TEXT         NOT NULL DEFAULT '',
    status         VARCHAR(16)  NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'review', 'published', 'hidden')),
    verify1_by     BIGINT,
    verify1_at     TIMESTAMPTZ,
    verify2_by     BIGINT,
    verify2_at     TIMESTAMPTZ,
    external_url   TEXT         NOT NULL DEFAULT '',
    themes         JSONB        NOT NULL DEFAULT '[]',
    seasons        JSONB        NOT NULL DEFAULT '[]',
    textbook       JSONB        NOT NULL DEFAULT '[]',
    shihua_work_id BIGINT       UNIQUE,
    created_by     BIGINT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT mat_texts_reference_only_no_body CHECK (license_status <> 'reference_only' OR (body = '' AND translation = '')),
    CONSTRAINT mat_texts_song_reference_only CHECK (kind <> 'song' OR license_status = 'reference_only')
);

CREATE INDEX IF NOT EXISTS idx_mat_texts_kind ON mat_texts (kind, status, id);
CREATE INDEX IF NOT EXISTS idx_mat_texts_author ON mat_texts (author_id);

CREATE TABLE IF NOT EXISTS mat_text_segments (
    id         BIGSERIAL   PRIMARY KEY,
    text_id    BIGINT      NOT NULL REFERENCES mat_texts(id) ON DELETE CASCADE,
    seq        INT         NOT NULL,
    label      VARCHAR(32) NOT NULL DEFAULT '',
    content    TEXT        NOT NULL DEFAULT '',
    scene_desc TEXT        NOT NULL DEFAULT '',
    is_famous  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (text_id, seq)
);

-- No original words for reference_only texts, whichever table they would land in.
CREATE OR REPLACE FUNCTION mat_segment_license_guard() RETURNS trigger AS $$
BEGIN
    IF NEW.content <> '' AND EXISTS (
        SELECT 1 FROM mat_texts WHERE id = NEW.text_id AND license_status = 'reference_only'
    ) THEN
        RAISE EXCEPTION 'mat_text_segments: text % is reference_only, segment content must be empty', NEW.text_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mat_segment_license_guard ON mat_text_segments;
CREATE TRIGGER trg_mat_segment_license_guard BEFORE INSERT OR UPDATE ON mat_text_segments
    FOR EACH ROW EXECUTE FUNCTION mat_segment_license_guard();

CREATE OR REPLACE FUNCTION mat_text_license_guard() RETURNS trigger AS $$
BEGIN
    IF NEW.license_status = 'reference_only' AND EXISTS (
        SELECT 1 FROM mat_text_segments WHERE text_id = NEW.id AND content <> ''
    ) THEN
        RAISE EXCEPTION 'mat_texts: text % still has segment content, clear it before marking reference_only', NEW.id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mat_text_license_guard ON mat_texts;
CREATE TRIGGER trg_mat_text_license_guard BEFORE UPDATE OF license_status ON mat_texts
    FOR EACH ROW EXECUTE FUNCTION mat_text_license_guard();

CREATE TABLE IF NOT EXISTS mat_imagery (
    id                 BIGSERIAL   PRIMARY KEY,
    name               VARCHAR(32) NOT NULL UNIQUE,
    aliases            JSONB       NOT NULL DEFAULT '[]',
    category           VARCHAR(16) NOT NULL DEFAULT '',
    meaning            TEXT        NOT NULL DEFAULT '',
    visual_zh          TEXT        NOT NULL DEFAULT '',
    visual_en          TEXT        NOT NULL DEFAULT '',
    emotions           JSONB       NOT NULL DEFAULT '[]',
    seasons            JSONB       NOT NULL DEFAULT '[]',
    times              JSONB       NOT NULL DEFAULT '[]',
    weathers           JSONB       NOT NULL DEFAULT '[]',
    eras               JSONB       NOT NULL DEFAULT '[]',
    palette_ids        JSONB       NOT NULL DEFAULT '[]',
    style_ids          JSONB       NOT NULL DEFAULT '[]',
    example_artwork_id BIGINT,
    status             VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'review', 'published', 'hidden')),
    created_by         BIGINT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mat_imagery_category ON mat_imagery (category, status, id);

CREATE TABLE IF NOT EXISTS mat_segment_imagery (
    segment_id BIGINT      NOT NULL REFERENCES mat_text_segments(id) ON DELETE CASCADE,
    imagery_id BIGINT      NOT NULL REFERENCES mat_imagery(id) ON DELETE CASCADE,
    weight     SMALLINT    NOT NULL DEFAULT 2 CHECK (weight BETWEEN 1 AND 3),
    source     VARCHAR(8)  NOT NULL DEFAULT 'manual' CHECK (source IN ('ai', 'manual')),
    confirmed  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (segment_id, imagery_id)
);

CREATE INDEX IF NOT EXISTS idx_mat_segment_imagery_imagery ON mat_segment_imagery (imagery_id);

CREATE TABLE IF NOT EXISTS mat_styles (
    id                 BIGSERIAL   PRIMARY KEY,
    name               VARCHAR(32) NOT NULL UNIQUE,
    category           VARCHAR(16) NOT NULL DEFAULT '',
    desc_zh            TEXT        NOT NULL DEFAULT '',
    prompt_en          TEXT        NOT NULL,
    negative_en        TEXT        NOT NULL DEFAULT '',
    suits_emotions     JSONB       NOT NULL DEFAULT '[]',
    suits_categories   JSONB       NOT NULL DEFAULT '[]',
    options            JSONB       NOT NULL DEFAULT '{}',
    default_size       VARCHAR(16) NOT NULL DEFAULT '1024x1536',
    default_quality    VARCHAR(16) NOT NULL DEFAULT 'high',
    sample_artwork_ids JSONB       NOT NULL DEFAULT '[]',
    enabled            BOOLEAN     NOT NULL DEFAULT TRUE,
    sort               INT         NOT NULL DEFAULT 0,
    created_by         BIGINT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mat_artworks (
    id                   BIGSERIAL     PRIMARY KEY,
    segment_id           BIGINT        REFERENCES mat_text_segments(id) ON DELETE SET NULL,
    style_id             BIGINT        REFERENCES mat_styles(id) ON DELETE SET NULL,
    options              JSONB         NOT NULL DEFAULT '{}',
    prompt_final         TEXT          NOT NULL,
    prompt_parts         JSONB         NOT NULL DEFAULT '{}',
    model                VARCHAR(32)   NOT NULL DEFAULT 'gpt-image-2',
    size                 VARCHAR(16)   NOT NULL DEFAULT '1024x1536',
    quality              VARCHAR(16)   NOT NULL DEFAULT 'high',
    status               VARCHAR(16)   NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled')),
    attempts             INT           NOT NULL DEFAULT 0,
    not_before           TIMESTAMPTZ,
    error                TEXT          NOT NULL DEFAULT '',
    file                 TEXT          NOT NULL DEFAULT '',
    width                INT           NOT NULL DEFAULT 0,
    height               INT           NOT NULL DEFAULT 0,
    cost_usd             NUMERIC(10,4) NOT NULL DEFAULT 0,
    duration_ms          INT           NOT NULL DEFAULT 0,
    parent_id            BIGINT        REFERENCES mat_artworks(id) ON DELETE SET NULL,
    source               VARCHAR(16)   NOT NULL DEFAULT 'library',
    rating               SMALLINT      NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5),
    review_status        VARCHAR(16)   NOT NULL DEFAULT 'pending' CHECK (review_status IN ('pending', 'approved', 'rejected')),
    reject_reason        TEXT          NOT NULL DEFAULT '',
    text_overlay_allowed BOOLEAN       NOT NULL DEFAULT FALSE,
    published_to         JSONB         NOT NULL DEFAULT '[]',
    created_by           BIGINT,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ,
    finished_at          TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_mat_artworks_segment ON mat_artworks (segment_id, id);
CREATE INDEX IF NOT EXISTS idx_mat_artworks_queue ON mat_artworks (id) WHERE status = 'queued';
CREATE INDEX IF NOT EXISTS idx_mat_artworks_review ON mat_artworks (review_status, id);

CREATE TABLE IF NOT EXISTS mat_artwork_imagery (
    artwork_id BIGINT NOT NULL REFERENCES mat_artworks(id) ON DELETE CASCADE,
    imagery_id BIGINT NOT NULL REFERENCES mat_imagery(id) ON DELETE CASCADE,
    PRIMARY KEY (artwork_id, imagery_id)
);

-- A finished poster of an approved drawing in one aspect ratio, laid out by the admin page. layout
-- records the words set on it; the server checks them against the text before saving.
CREATE TABLE IF NOT EXISTS mat_artwork_renders (
    id         BIGSERIAL   PRIMARY KEY,
    artwork_id BIGINT      NOT NULL REFERENCES mat_artworks(id) ON DELETE CASCADE,
    aspect     VARCHAR(8)  NOT NULL,
    file       TEXT        NOT NULL,
    width      INT         NOT NULL DEFAULT 0,
    height     INT         NOT NULL DEFAULT 0,
    layout     JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (artwork_id, aspect)
);

CREATE TABLE IF NOT EXISTS mat_blocklist (
    id         BIGSERIAL    PRIMARY KEY,
    term       VARCHAR(128) NOT NULL,
    kind       VARCHAR(16)  NOT NULL CHECK (kind IN ('artist', 'studio', 'character', 'person', 'symbol', 'text')),
    match      VARCHAR(8)   NOT NULL DEFAULT 'contains' CHECK (match IN ('contains', 'regex')),
    action     VARCHAR(8)   NOT NULL DEFAULT 'block' CHECK (action IN ('block', 'warn', 'strip')),
    applies_to VARCHAR(8)   NOT NULL DEFAULT 'prompt' CHECK (applies_to IN ('prompt', 'text', 'both')),
    reason     TEXT         NOT NULL DEFAULT '',
    enabled    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_by BIGINT,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (term, kind)
);

-- Seed styles: the 9 from promo/lyrics-art/lyric2art.py. No artist, studio or work names.
INSERT INTO mat_styles (name, category, desc_zh, prompt_en, suits_emotions, sort) VALUES
('工笔青花', '国画', '青花瓷的青白两色，工笔细线，宣纸质感',
 'Traditional Chinese gongbi painting in a blue-and-white porcelain palette (cobalt blue, white, soft grey), fine precise brush lines, xuan paper texture, generous empty space.',
 '["思念", "等待", "清雅"]', 10),
('水墨', '国画', '黑白水墨，大留白，可点一点暖色',
 'Chinese ink-wash painting, loose expressive brushwork, black and grey ink with at most one small warm accent, wide negative space, rice paper texture.',
 '["孤独", "离别", "壮阔", "怀古"]', 20),
('淡彩水墨', '国画', '水墨打底，淡淡的绿和粉，江南烟雨感',
 'Light-color Chinese ink and watercolor painting, soft greens, pale pinks and misty whites, airy composition with drifting mist, delicate reflections.',
 '["闲适", "思念", "惆怅"]', 30),
('童年水彩', '插画', '温暖的手绘动画电影风，夏天、乡村、童年',
 'Warm hand-painted animation-film background style in watercolor and gouache, lush summer greens, big cumulus clouds, golden-hour light, nostalgic and gentle.',
 '["童趣", "田园", "怀旧"]', 40),
('日系动画', '动画', '清新日系动画，湿润的光，校园和城市',
 'Japanese anime film still, clean line art, soft cel shading, luminous wet light and reflections, detailed background, emotional and quiet.',
 '["青春", "怀旧", "思念"]', 50),
('油画', '西画', '厚涂油画，光影浓郁',
 'Oil painting with visible impasto brush strokes, rich color, dramatic natural light, painterly texture, museum-quality fine art.',
 '["壮阔", "热烈"]', 60),
('胶片', '摄影', '电影胶片感，低饱和，颗粒',
 'Cinematic 35mm film photograph look, muted low-saturation colors, soft film grain, natural light, shallow depth of field, quiet storytelling frame.',
 '["怀旧", "孤独"]', 70),
('星空', '插画', '宇宙、星光、梦幻科幻',
 'Dreamy cosmic illustration, deep space with nebulae and countless stars, glowing light, vast scale, luminous blues and violets with warm highlights.',
 '["浪漫", "壮阔", "思念"]', 80),
('版画', '版画', '木刻版画，粗线条，两三种颜色',
 'Woodblock print style, bold carved lines, limited palette of two or three flat colors, paper grain, strong graphic composition.',
 '["质朴", "壮阔"]', 90)
ON CONFLICT (name) DO NOTHING;

-- Seed blocklist: asking the model to write words (warn: the image model gets characters wrong), and
-- studios / artists / characters a prompt must not name.
INSERT INTO mat_blocklist (term, kind, match, action, applies_to, reason) VALUES
('(?i)\b(caption|lettering|typography|inscription|headline|subtitle)s?\b', 'text', 'regex', 'warn', 'prompt', '图像模型写字常出错，文字由排版层加'),
('(?i)\bwrit(e|ten|ing) (the )?(poem|words|title|text)\b', 'text', 'regex', 'warn', 'prompt', '图像模型写字常出错，文字由排版层加'),
('题字', 'text', 'contains', 'warn', 'both', '图像模型写字常出错，文字由排版层加'),
('写上', 'text', 'contains', 'warn', 'both', '图像模型写字常出错，文字由排版层加'),
('(?i)ghibli', 'studio', 'regex', 'block', 'prompt', '不模仿具体工作室'),
('吉卜力', 'studio', 'contains', 'block', 'both', '不模仿具体工作室'),
('(?i)\bpixar\b', 'studio', 'regex', 'block', 'prompt', '不模仿具体工作室'),
('(?i)\bdisney\b', 'studio', 'regex', 'block', 'prompt', '不模仿具体工作室'),
('(?i)kyoto animation', 'studio', 'regex', 'block', 'prompt', '不模仿具体工作室'),
('(?i)miyazaki', 'artist', 'regex', 'block', 'prompt', '不模仿具体画家'),
('宫崎骏', 'artist', 'contains', 'block', 'both', '不模仿具体画家'),
('(?i)shinkai', 'artist', 'regex', 'block', 'prompt', '不模仿具体画家'),
('新海诚', 'artist', 'contains', 'block', 'both', '不模仿具体画家'),
('(?i)rutkowski', 'artist', 'regex', 'block', 'prompt', '不模仿具体画家'),
('(?i)in the style of', 'artist', 'regex', 'warn', 'prompt', '「某某风格」常带出画家名，确认没有点名'),
('(?i)\b(pikachu|doraemon|mickey mouse|hello kitty)\b', 'character', 'regex', 'block', 'prompt', '不画版权角色'),
('皮卡丘', 'character', 'contains', 'block', 'both', '不画版权角色'),
('哆啦A梦', 'character', 'contains', 'block', 'both', '不画版权角色')
ON CONFLICT (term, kind) DO NOTHING;
