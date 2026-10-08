-- 课程创作者：用户申请成为创作者（管理员审核），发布自己的课程（首次上架和每次修改都要管理员审核，
-- 网盘发货信息不用审核），课程订单按创作者的手续费比例（为空用全站默认）分成，结算期过后可以申请提现，
-- 管理员线下打款后标记已打款。
CREATE TABLE IF NOT EXISTS course_creators (
    user_id            BIGINT        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status             VARCHAR(16)   NOT NULL DEFAULT 'pending',
    display_name       VARCHAR(40)   NOT NULL,
    bio                TEXT          NOT NULL DEFAULT '',
    contact            VARCHAR(100)  NOT NULL DEFAULT '',
    plan               TEXT          NOT NULL DEFAULT '',
    commission_percent NUMERIC(5,2)  NULL,
    admin_note         VARCHAR(500)  NOT NULL DEFAULT '',
    reviewed_at        TIMESTAMPTZ   NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT course_creators_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'suspended'))
);
CREATE INDEX IF NOT EXISTS idx_course_creators_status ON course_creators (status, created_at DESC);

-- owner_id 为空是平台自己的课程。创作者的修改存在 draft 里，审核通过后才覆盖线上字段。
ALTER TABLE courses ADD COLUMN IF NOT EXISTS owner_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS draft JSONB NULL;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS review_status VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE courses ADD COLUMN IF NOT EXISTS review_note VARCHAR(500) NOT NULL DEFAULT '';
ALTER TABLE courses ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMPTZ NULL;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ NULL;
CREATE INDEX IF NOT EXISTS idx_courses_owner ON courses (owner_id) WHERE owner_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_courses_review ON courses (review_status, submitted_at) WHERE review_status = 'pending';

-- 每笔创作者课程订单一行：实付（不含支付手续费）和成交时的手续费比例；退款按订单状态实时扣减。
CREATE TABLE IF NOT EXISTS course_creator_earnings (
    order_id           BIGINT        PRIMARY KEY REFERENCES payment_orders(id) ON DELETE CASCADE,
    creator_id         BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id          BIGINT        NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    gross_cny          NUMERIC(12,2) NOT NULL,
    commission_percent NUMERIC(5,2)  NOT NULL,
    available_at       TIMESTAMPTZ   NOT NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_course_creator_earnings_creator ON course_creator_earnings (creator_id, created_at DESC);

CREATE TABLE IF NOT EXISTS course_creator_withdrawals (
    id            BIGSERIAL     PRIMARY KEY,
    user_id       BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cny_amount    NUMERIC(12,2) NOT NULL,
    method        VARCHAR(16)   NOT NULL,
    -- 收款账号与实名用 AES-GCM 加密保存
    account_enc   TEXT          NOT NULL,
    real_name_enc TEXT          NOT NULL,
    user_note     VARCHAR(200)  NOT NULL DEFAULT '',
    status        VARCHAR(16)   NOT NULL DEFAULT 'pending',
    admin_note    VARCHAR(500)  NOT NULL DEFAULT '',
    reviewed_by   BIGINT        NULL REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at   TIMESTAMPTZ   NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT course_creator_withdrawals_status_check CHECK (status IN ('pending', 'paid', 'rejected', 'cancelled')),
    CONSTRAINT course_creator_withdrawals_method_check CHECK (method IN ('alipay', 'wechat')),
    CONSTRAINT course_creator_withdrawals_amount_check CHECK (cny_amount > 0)
);
CREATE INDEX IF NOT EXISTS idx_course_creator_withdrawals_user ON course_creator_withdrawals (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_course_creator_withdrawals_status ON course_creator_withdrawals (status, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_course_creator_withdrawals_one_pending
    ON course_creator_withdrawals (user_id) WHERE status = 'pending';
