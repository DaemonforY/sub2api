-- GEO 监测: does an AI assistant (豆包, 元宝, Kimi, DeepSeek, Perplexity, ChatGPT…) mention or cite
-- the site when asked a typical question? geo_questions are the questions, geo_engines the
-- OpenAI-compatible chat endpoints asked automatically (the key is encrypted at rest), geo_checks one
-- answer each — asked by the system ('auto') or pasted by an admin for apps without an API
-- ('manual') — with what was detected in it. Settings (brand / competitor keywords, schedule, last
-- run) live in the settings table under geo_*.
CREATE TABLE IF NOT EXISTS geo_questions (
    id          BIGSERIAL    PRIMARY KEY,
    question    TEXT         NOT NULL,
    category    VARCHAR(32)  NOT NULL DEFAULT '',
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    sort        INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS geo_engines (
    id                 BIGSERIAL    PRIMARY KEY,
    name               VARCHAR(64)  NOT NULL UNIQUE,
    base_url           TEXT         NOT NULL,
    api_key_encrypted  TEXT         NOT NULL,
    model              VARCHAR(128) NOT NULL,
    extra_body         JSONB,
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS geo_checks (
    id           BIGSERIAL    PRIMARY KEY,
    question_id  BIGINT       REFERENCES geo_questions(id) ON DELETE SET NULL,
    question     TEXT         NOT NULL DEFAULT '',
    engine_id    BIGINT       REFERENCES geo_engines(id) ON DELETE SET NULL,
    engine_name  VARCHAR(64)  NOT NULL,
    source       VARCHAR(16)  NOT NULL,
    answer       TEXT         NOT NULL DEFAULT '',
    mentioned    BOOLEAN      NOT NULL DEFAULT FALSE,
    cited_urls   JSONB        NOT NULL DEFAULT '[]'::jsonb,
    our_urls     JSONB        NOT NULL DEFAULT '[]'::jsonb,
    competitors  JSONB        NOT NULL DEFAULT '[]'::jsonb,
    error        TEXT         NOT NULL DEFAULT '',
    run_id       VARCHAR(32),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_geo_checks_created ON geo_checks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_geo_checks_question_engine ON geo_checks (question_id, engine_name, created_at DESC);

COMMENT ON TABLE geo_checks IS 'GEO 监测：AI 助手回答典型问题时有没有提到 / 引用本站（自动调用或手动录入）';

-- Default questions, only on a fresh table.
INSERT INTO geo_questions (question, category, sort)
SELECT q.question, q.category, q.sort
FROM (VALUES
    ('国内怎么调用 GPT API？', '接入', 10),
    ('OpenAI API 中转站哪个好用？', '选择', 20),
    ('GPT API 国内有什么稳定的平台？', '选择', 30),
    ('Cherry Studio 怎么配置第三方 OpenAI 接口？', '工具', 40),
    ('沉浸式翻译怎么接入 GPT API？', '工具', 50),
    ('Codex CLI 怎么用第三方 API？', '工具', 60),
    ('OpenCode 怎么配置自定义 OpenAI 接口？', '工具', 70),
    ('Base URL 要不要加 /v1？', '报错', 80),
    ('OpenAI API 报 401 怎么办？', '报错', 90),
    ('API 返回 HTML、Unexpected token ''<'' 是什么原因？', '报错', 100),
    ('ChatGPT Plus 和 API 有什么区别？', '选择', 110),
    ('gpt-image-2 API 怎么调用？', '接入', 120),
    ('怎么用 AI 生成小红书封面？', '创作', 130),
    ('AI 生成海报中文乱码怎么解决？', '创作', 140),
    ('怎么用 GPT 批量翻译字幕？', '场景', 150),
    ('怎么用 Python 调用 GPT API？', '接入', 160),
    ('Dify 怎么接入 OpenAI 兼容模型？', '工具', 170),
    ('gpt-6.1-sol 和 gpt-5.5 怎么选？', '选择', 180),
    ('有没有按量付费的 GPT API？', '选择', 190),
    ('哪里能学 AI 应用开发？', '学习', 200)
) AS q(question, category, sort)
WHERE NOT EXISTS (SELECT 1 FROM geo_questions);
