-- Paid courses: sold through payment orders (order_type = 'course'); buyers get the delivery
-- (a netdisk link and codes, stored encrypted, versioned so the link can be replaced).
CREATE TABLE IF NOT EXISTS courses (
    id             BIGSERIAL     PRIMARY KEY,
    slug           VARCHAR(64)   NOT NULL UNIQUE,
    title          VARCHAR(120)  NOT NULL,
    subtitle       VARCHAR(200)  NOT NULL DEFAULT '',
    category       VARCHAR(32)   NOT NULL DEFAULT '',
    cover_file     VARCHAR(64)   NOT NULL DEFAULT '',
    price          NUMERIC(10,2) NOT NULL,
    original_price NUMERIC(10,2) NOT NULL DEFAULT 0,
    intro_md       TEXT          NOT NULL DEFAULT '',
    outline        JSONB         NOT NULL DEFAULT '[]',
    trial_md       TEXT          NOT NULL DEFAULT '',
    faq_md         TEXT          NOT NULL DEFAULT '',
    status         VARCHAR(16)   NOT NULL DEFAULT 'draft',
    sort_order     INTEGER       NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_courses_status_sort ON courses (status, sort_order DESC, id DESC);

CREATE TABLE IF NOT EXISTS course_deliveries (
    id           BIGSERIAL   PRIMARY KEY,
    course_id    BIGINT      NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    version      INTEGER     NOT NULL,
    link_enc     TEXT        NOT NULL,
    code_enc     TEXT        NOT NULL DEFAULT '',
    password_enc TEXT        NOT NULL DEFAULT '',
    note         TEXT        NOT NULL DEFAULT '',
    created_by   BIGINT      NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (course_id, version)
);

CREATE TABLE IF NOT EXISTS course_enrollments (
    id              BIGSERIAL   PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id       BIGINT      NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    order_id        BIGINT      NULL,
    source          VARCHAR(16) NOT NULL DEFAULT 'purchase',
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    note            TEXT        NOT NULL DEFAULT '',
    first_viewed_at TIMESTAMPTZ NULL,
    last_viewed_at  TIMESTAMPTZ NULL,
    view_count      INTEGER     NOT NULL DEFAULT 0,
    seen_version    INTEGER     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, course_id)
);
CREATE INDEX IF NOT EXISTS idx_course_enrollments_course ON course_enrollments (course_id, created_at DESC);

CREATE TABLE IF NOT EXISTS course_access_logs (
    id         BIGSERIAL   PRIMARY KEY,
    course_id  BIGINT      NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL,
    ip         VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_course_access_logs_user ON course_access_logs (course_id, user_id, created_at DESC);

ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS course_id BIGINT NULL;
