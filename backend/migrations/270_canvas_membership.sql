-- 创作会员 (canvas creator membership): images saved from the canvas image workbench come without
-- the 「AI 生成 · <site>」 watermark while member_until is in the future. Bought as a payment order
-- (order_type = 'membership', subscription_days = the days bought) or granted by an admin.
CREATE TABLE IF NOT EXISTS canvas_memberships (
    user_id            BIGINT      PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    member_until       TIMESTAMPTZ NOT NULL,
    -- When the member accepted that unwatermarked images must be labelled 「AI 生成」 when published
    -- (《人工智能生成合成内容标识办法》第九条).
    terms_accepted_at  TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_canvas_memberships_until ON canvas_memberships (member_until);

COMMENT ON TABLE canvas_memberships IS '创作会员：有效期内画布生图工作台保存的图片不加「AI 生成」水印';

-- Who received images without the explicit AI label, kept at least six months (第九条 requires
-- the provider to log the recipients of unlabelled content). Rows older than 200 days are pruned.
CREATE TABLE IF NOT EXISTS canvas_unmarked_saves (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    width       INTEGER      NOT NULL DEFAULT 0,
    height      INTEGER      NOT NULL DEFAULT 0,
    client_ip   VARCHAR(64)  NOT NULL DEFAULT '',
    user_agent  VARCHAR(255) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_canvas_unmarked_saves_user ON canvas_unmarked_saves (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_canvas_unmarked_saves_created ON canvas_unmarked_saves (created_at);

COMMENT ON TABLE canvas_unmarked_saves IS '创作会员保存无水印图片的记录（按标识办法第九条留存，至少六个月）';
