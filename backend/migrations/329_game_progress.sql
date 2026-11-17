-- 小游戏可选登录: a signed-in player's progress (one JSON blob per game, written by the game) and best
-- score per leaderboard. Visitors still play without an account; these rows exist only for players who
-- signed in to keep progress across devices and appear on the site-wide leaderboard.
CREATE TABLE IF NOT EXISTS game_saves (
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game       VARCHAR(24) NOT NULL,
    data       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, game)
);

CREATE TABLE IF NOT EXISTS game_scores (
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game       VARCHAR(24) NOT NULL,
    board      VARCHAR(24) NOT NULL DEFAULT 'main',
    score      BIGINT      NOT NULL,
    detail     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, game, board)
);

CREATE INDEX IF NOT EXISTS idx_game_scores_board ON game_scores (game, board, score DESC, updated_at);

COMMENT ON TABLE game_saves IS '小游戏：登录玩家的云端进度（每个游戏一份 JSON，由游戏自己定义内容）';
COMMENT ON TABLE game_scores IS '小游戏：登录玩家在每个排行榜上的最好成绩，越高越好';
