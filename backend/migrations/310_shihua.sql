-- 课本诗画: an admin pipeline that turns textbook poems and short classical prose into illustrated
-- posters. The image model draws the picture with the title and the author's name; the body text,
-- the seal and the AI label are set by the admin page in the browser, so every character is exact.
--
-- shihua_works   the poems / passages (text checked twice before it may go on a poster)
-- shihua_jobs    a batch of drawings started together
-- shihua_images  one drawing (queued → running → succeeded / failed / canceled), retried on 429 / 5xx
-- shihua_renders the finished poster of a picked drawing in one aspect ratio
CREATE TABLE IF NOT EXISTS shihua_works (
    id              BIGSERIAL    PRIMARY KEY,
    slug            VARCHAR(64)  NOT NULL UNIQUE,
    title           VARCHAR(64)  NOT NULL,
    author          VARCHAR(64)  NOT NULL DEFAULT '',
    dynasty         VARCHAR(16)  NOT NULL DEFAULT '',
    genre           VARCHAR(16)  NOT NULL DEFAULT 'shi',
    stage           VARCHAR(16)  NOT NULL DEFAULT '',
    grade           VARCHAR(16)  NOT NULL DEFAULT '',
    textbooks       JSONB        NOT NULL DEFAULT '[]',
    full_text       TEXT         NOT NULL,
    display_text    TEXT         NOT NULL DEFAULT '',
    excerpt_note    TEXT         NOT NULL DEFAULT '',
    famous_lines    TEXT         NOT NULL DEFAULT '',
    sources         TEXT         NOT NULL DEFAULT '',
    variant_note    TEXT         NOT NULL DEFAULT '',
    text_status     VARCHAR(16)  NOT NULL DEFAULT 'draft',
    verify1_by      BIGINT,
    verify1_at      TIMESTAMPTZ,
    verify2_by      BIGINT,
    verify2_at      TIMESTAMPTZ,
    theme           JSONB        NOT NULL DEFAULT '{}',
    picked_image_id BIGINT,
    sort            INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shihua_works_grade ON shihua_works (stage, grade, sort, id);

CREATE TABLE IF NOT EXISTS shihua_jobs (
    id         BIGSERIAL   PRIMARY KEY,
    name       VARCHAR(64) NOT NULL DEFAULT '',
    candidates INT         NOT NULL DEFAULT 1,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS shihua_images (
    id             BIGSERIAL     PRIMARY KEY,
    work_id        BIGINT        NOT NULL REFERENCES shihua_works(id) ON DELETE CASCADE,
    job_id         BIGINT        REFERENCES shihua_jobs(id) ON DELETE SET NULL,
    status         VARCHAR(16)   NOT NULL DEFAULT 'queued',
    attempts       INT           NOT NULL DEFAULT 0,
    not_before     TIMESTAMPTZ,
    prompt         TEXT          NOT NULL,
    size           VARCHAR(16)   NOT NULL DEFAULT '1024x1536',
    quality        VARCHAR(16)   NOT NULL DEFAULT 'high',
    error          TEXT          NOT NULL DEFAULT '',
    duration_ms    INT           NOT NULL DEFAULT 0,
    cost_usd       NUMERIC(10,4) NOT NULL DEFAULT 0,
    file           TEXT          NOT NULL DEFAULT '',
    width          INT           NOT NULL DEFAULT 0,
    height         INT           NOT NULL DEFAULT 0,
    upscale_status VARCHAR(16)   NOT NULL DEFAULT '',
    upscale_error  TEXT          NOT NULL DEFAULT '',
    upscaled_file  TEXT          NOT NULL DEFAULT '',
    title_check    VARCHAR(8)    NOT NULL DEFAULT '',
    rejected       BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_shihua_images_work ON shihua_images (work_id, id);
CREATE INDEX IF NOT EXISTS idx_shihua_images_job ON shihua_images (job_id);
CREATE INDEX IF NOT EXISTS idx_shihua_images_queue ON shihua_images (id) WHERE status = 'queued';
CREATE INDEX IF NOT EXISTS idx_shihua_images_upscale ON shihua_images (id) WHERE upscale_status = 'queued';
CREATE INDEX IF NOT EXISTS idx_shihua_images_started ON shihua_images (started_at);

CREATE TABLE IF NOT EXISTS shihua_renders (
    id         BIGSERIAL   PRIMARY KEY,
    image_id   BIGINT      NOT NULL REFERENCES shihua_images(id) ON DELETE CASCADE,
    work_id    BIGINT      NOT NULL REFERENCES shihua_works(id) ON DELETE CASCADE,
    aspect     VARCHAR(8)  NOT NULL,
    file       TEXT        NOT NULL,
    width      INT         NOT NULL DEFAULT 0,
    height     INT         NOT NULL DEFAULT 0,
    layout     JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (image_id, aspect)
);

CREATE INDEX IF NOT EXISTS idx_shihua_renders_work ON shihua_renders (work_id);
