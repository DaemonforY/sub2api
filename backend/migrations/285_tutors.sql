-- AI 助教: assistants teachers build for their class. A teacher picks a template, fills in the
-- subject, grade and style, adds their own materials and shares a link / QR code with a class
-- password. Students join with a name (no account) and ask; answers run on one of the teacher's own
-- API keys through this site's gateway, so the teacher pays and sees usage as usual.
CREATE TABLE IF NOT EXISTS tutors (
    id               BIGSERIAL    PRIMARY KEY,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id       BIGINT       NOT NULL,
    name             VARCHAR(60)  NOT NULL,
    template         VARCHAR(20)  NOT NULL,
    subject          VARCHAR(40)  NOT NULL DEFAULT '',
    grade            VARCHAR(40)  NOT NULL DEFAULT '',
    style            VARCHAR(100) NOT NULL DEFAULT '',
    answer_mode      VARCHAR(10)  NOT NULL DEFAULT 'guide',
    rules            TEXT         NOT NULL DEFAULT '',
    greeting         TEXT         NOT NULL DEFAULT '',
    share_code       VARCHAR(16)  NOT NULL UNIQUE,
    pass_code        VARCHAR(32)  NOT NULL DEFAULT '',
    per_student_day  INTEGER      NOT NULL DEFAULT 20,
    daily_cap        INTEGER      NOT NULL DEFAULT 300,
    enabled          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tutors_user ON tutors (user_id, created_at DESC);

-- The teacher's materials, as extracted text (Word / PPT / PDF / text, or pasted).
CREATE TABLE IF NOT EXISTS tutor_materials (
    id          BIGSERIAL    PRIMARY KEY,
    tutor_id    BIGINT       NOT NULL REFERENCES tutors(id) ON DELETE CASCADE,
    name        VARCHAR(200) NOT NULL,
    chars       INTEGER      NOT NULL DEFAULT 0,
    content     TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tutor_materials_tutor ON tutor_materials (tutor_id);

-- Students who joined with the class password: a name and a device token (only its hash is kept).
CREATE TABLE IF NOT EXISTS tutor_students (
    id            BIGSERIAL    PRIMARY KEY,
    tutor_id      BIGINT       NOT NULL REFERENCES tutors(id) ON DELETE CASCADE,
    name          VARCHAR(30)  NOT NULL,
    token_hash    CHAR(64)     NOT NULL UNIQUE,
    questions     INTEGER      NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tutor_id, name)
);

-- What students asked and what the tutor answered, for the teacher's 学情 page (kept 180 days).
CREATE TABLE IF NOT EXISTS tutor_messages (
    id          BIGSERIAL    PRIMARY KEY,
    tutor_id    BIGINT       NOT NULL REFERENCES tutors(id) ON DELETE CASCADE,
    student_id  BIGINT       NOT NULL REFERENCES tutor_students(id) ON DELETE CASCADE,
    question    TEXT         NOT NULL,
    answer      TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_tutor_messages_tutor_created ON tutor_messages (tutor_id, created_at DESC);

COMMENT ON TABLE tutors IS 'AI 助教：老师用模板建的班级助教，学生凭链接和口令免注册提问，费用记在老师自己的 Key 上';
