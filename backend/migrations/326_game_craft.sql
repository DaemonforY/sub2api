-- 小游戏「AI 万物合成」(/games/craft): players drag two elements together and the text model invents
-- the result. Every element anyone has made is a row of game_craft_elements (id = order of discovery
-- across the whole site); every pair tried is a row of game_craft_recipes, so the model is asked about
-- a pair only once and everyone gets the same answer after that. a <= b (sorted names).
CREATE TABLE IF NOT EXISTS game_craft_elements (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(32)  NOT NULL UNIQUE,
    emoji         VARCHAR(32)  NOT NULL DEFAULT '',
    description   VARCHAR(200) NOT NULL DEFAULT '',
    discovered_by BIGINT       NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS game_craft_recipes (
    a          VARCHAR(32) NOT NULL,
    b          VARCHAR(32) NOT NULL,
    result     VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (a, b)
);

CREATE INDEX IF NOT EXISTS idx_game_craft_elements_created ON game_craft_elements (created_at DESC);

INSERT INTO game_craft_elements (name, emoji, description) VALUES
    ('金', '🪙', '五行之金，坚硬、锋利，主收敛。'),
    ('木', '🌳', '五行之木，生长、舒展，主生发。'),
    ('水', '💧', '五行之水，柔软、流动，主润下。'),
    ('火', '🔥', '五行之火，炎热、向上，主炎上。'),
    ('土', '⛰️', '五行之土，厚重、承载，主化育。')
ON CONFLICT (name) DO NOTHING;

COMMENT ON TABLE game_craft_elements IS '小游戏「AI 万物合成」：全站已被合成出来的元素，id 是发现顺序';
COMMENT ON TABLE game_craft_recipes IS '小游戏「AI 万物合成」：两个元素（按名字排序）合成的结果，每对只问一次模型';
