-- Creative community of HiveGPT 无限画布 (pages on the canvas site, data here): public profiles,
-- works (AI images with their prompts), collections, likes, favorites, follows, notifications and
-- reports. A user has no public profile until they create one (first publish).

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id          BIGINT       PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    handle           VARCHAR(20)  NOT NULL,
    display_name     VARCHAR(40)  NOT NULL DEFAULT '',
    avatar_file      VARCHAR(64)  NOT NULL DEFAULT '',
    bio              VARCHAR(200) NOT NULL DEFAULT '',
    -- active | banned (cannot publish; works hidden)
    status           VARCHAR(16)  NOT NULL DEFAULT 'active',
    works_count      INTEGER      NOT NULL DEFAULT 0,
    followers_count  INTEGER      NOT NULL DEFAULT 0,
    following_count  INTEGER      NOT NULL DEFAULT 0,
    likes_received   INTEGER      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_profiles_handle ON user_profiles (LOWER(handle));

CREATE TABLE IF NOT EXISTS works (
    id              BIGSERIAL    PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(80)  NOT NULL DEFAULT '',
    description     VARCHAR(1000) NOT NULL DEFAULT '',
    prompt          TEXT         NOT NULL DEFAULT '',
    show_prompt     BOOLEAN      NOT NULL DEFAULT TRUE,
    model           VARCHAR(100) NOT NULL DEFAULT '',
    params          JSONB        NOT NULL DEFAULT '{}'::jsonb,
    -- canvas | image_workbench | tools | contest
    source          VARCHAR(32)  NOT NULL DEFAULT 'canvas',
    tags            TEXT[]       NOT NULL DEFAULT '{}',
    -- public | unlisted (link only) | private
    visibility      VARCHAR(16)  NOT NULL DEFAULT 'public',
    -- approved | pending (review) | rejected | hidden (by an admin)
    status          VARCHAR(16)  NOT NULL DEFAULT 'approved',
    review_reason   VARCHAR(200) NOT NULL DEFAULT '',
    review_flags    TEXT[]       NOT NULL DEFAULT '{}',
    featured_at     TIMESTAMPTZ  NULL,
    cover_width     INTEGER      NOT NULL DEFAULT 0,
    cover_height    INTEGER      NOT NULL DEFAULT 0,
    like_count      INTEGER      NOT NULL DEFAULT 0,
    favorite_count  INTEGER      NOT NULL DEFAULT 0,
    remix_count     INTEGER      NOT NULL DEFAULT 0,
    view_count      INTEGER      NOT NULL DEFAULT 0,
    report_count    INTEGER      NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_works_user_created ON works (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_works_feed ON works (created_at DESC) WHERE visibility = 'public' AND status = 'approved';
CREATE INDEX IF NOT EXISTS idx_works_status ON works (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_works_tags ON works USING GIN (tags);

CREATE TABLE IF NOT EXISTS work_media (
    id          BIGSERIAL    PRIMARY KEY,
    work_id     BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    position    INTEGER      NOT NULL,
    file        VARCHAR(64)  NOT NULL,
    thumb_file  VARCHAR(64)  NOT NULL,
    mime_type   VARCHAR(32)  NOT NULL,
    width       INTEGER      NOT NULL,
    height      INTEGER      NOT NULL,
    size_bytes  BIGINT       NOT NULL,
    UNIQUE (work_id, position)
);

CREATE TABLE IF NOT EXISTS collections (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        VARCHAR(60)  NOT NULL,
    description  VARCHAR(300) NOT NULL DEFAULT '',
    -- public | private
    visibility   VARCHAR(16)  NOT NULL DEFAULT 'public',
    works_count  INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_collections_user ON collections (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS collection_items (
    collection_id  BIGINT       NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    work_id        BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    position       INTEGER      NOT NULL DEFAULT 0,
    added_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (collection_id, work_id)
);

CREATE TABLE IF NOT EXISTS follows (
    follower_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id  BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CHECK (follower_id <> followee_id)
);
CREATE INDEX IF NOT EXISTS idx_follows_followee ON follows (followee_id, created_at DESC);

CREATE TABLE IF NOT EXISTS work_likes (
    work_id     BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (work_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_work_likes_user ON work_likes (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS work_favorites (
    work_id     BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (work_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_work_favorites_user ON work_favorites (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS community_notifications (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- follow | like | favorite | remix | featured | rejected | hidden
    kind        VARCHAR(16)  NOT NULL,
    actor_id    BIGINT       NULL REFERENCES users(id) ON DELETE CASCADE,
    work_id     BIGINT       NULL REFERENCES works(id) ON DELETE CASCADE,
    detail      VARCHAR(200) NOT NULL DEFAULT '',
    read_at     TIMESTAMPTZ  NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_community_notifications_user ON community_notifications (user_id, created_at DESC);
-- One unread notice per actor, kind and work: liking, unliking and liking again does not spam.
CREATE UNIQUE INDEX IF NOT EXISTS idx_community_notifications_dedupe ON community_notifications (user_id, kind, COALESCE(actor_id, 0), COALESCE(work_id, 0)) WHERE read_at IS NULL;

CREATE TABLE IF NOT EXISTS work_reports (
    id          BIGSERIAL    PRIMARY KEY,
    work_id     BIGINT       NOT NULL REFERENCES works(id) ON DELETE CASCADE,
    reporter_id BIGINT       NULL REFERENCES users(id) ON DELETE SET NULL,
    reason      VARCHAR(32)  NOT NULL,
    detail      VARCHAR(500) NOT NULL DEFAULT '',
    ip          VARCHAR(64)  NOT NULL DEFAULT '',
    -- open | resolved | dismissed
    status      VARCHAR(16)  NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_work_reports_status ON work_reports (status, created_at DESC);

COMMENT ON TABLE user_profiles IS '社区个人主页（首次发布作品时创建）';
COMMENT ON TABLE works IS '社区作品（AI 生成图片及提示词）';
COMMENT ON TABLE community_notifications IS '社区站内通知：关注、点赞、收藏、做同款、精选、审核';
