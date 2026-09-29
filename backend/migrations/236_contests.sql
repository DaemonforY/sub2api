-- Community contests (e.g. AI drawing contests): users submit entries, logged-in
-- users vote, the ranking is frozen at voting_end_at and prizes are awarded.
CREATE TABLE IF NOT EXISTS contests (
    id                     BIGSERIAL PRIMARY KEY,
    title                  VARCHAR(200) NOT NULL,
    description            TEXT         NOT NULL DEFAULT '',
    rules                  TEXT         NOT NULL DEFAULT '',
    cover_image            VARCHAR(512) NOT NULL DEFAULT '',
    -- draft | published | settled | cancelled
    status                 VARCHAR(20)  NOT NULL DEFAULT 'draft',
    submission_start_at    TIMESTAMPTZ  NOT NULL,
    submission_end_at      TIMESTAMPTZ  NOT NULL,
    voting_start_at        TIMESTAMPTZ  NOT NULL,
    voting_end_at          TIMESTAMPTZ  NOT NULL,
    max_entries_per_user   INTEGER      NOT NULL DEFAULT 1,
    votes_per_user         INTEGER      NOT NULL DEFAULT 3,
    allow_self_vote        BOOLEAN      NOT NULL DEFAULT FALSE,
    require_review         BOOLEAN      NOT NULL DEFAULT FALSE,
    min_account_age_hours  INTEGER      NOT NULL DEFAULT 0,
    one_prize_per_user     BOOLEAN      NOT NULL DEFAULT TRUE,
    min_votes_for_prize    INTEGER      NOT NULL DEFAULT 1,
    -- [{"rank_from":1,"rank_to":1,"type":"balance|custom","amount":10,"label":"一等奖"}]
    prizes                 JSONB        NOT NULL DEFAULT '[]'::jsonb,
    settled_at             TIMESTAMPTZ  NULL,
    created_by             BIGINT       NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contests_status_voting_end ON contests (status, voting_end_at);

CREATE TABLE IF NOT EXISTS contest_entries (
    id             BIGSERIAL PRIMARY KEY,
    contest_id     BIGINT       NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id        BIGINT       NOT NULL,
    title          VARCHAR(120) NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    prompt         TEXT         NOT NULL DEFAULT '',
    image_file     VARCHAR(128) NOT NULL,
    -- pending | approved | rejected | withdrawn | disqualified
    status         VARCHAR(20)  NOT NULL DEFAULT 'approved',
    review_note    VARCHAR(500) NOT NULL DEFAULT '',
    vote_count     INTEGER      NOT NULL DEFAULT 0,
    final_rank     INTEGER      NULL,
    final_votes    INTEGER      NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contest_entries_contest_status_votes ON contest_entries (contest_id, status, vote_count DESC);
CREATE INDEX IF NOT EXISTS idx_contest_entries_user ON contest_entries (contest_id, user_id);

CREATE TABLE IF NOT EXISTS contest_votes (
    id          BIGSERIAL PRIMARY KEY,
    contest_id  BIGINT      NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    entry_id    BIGINT      NOT NULL REFERENCES contest_entries(id) ON DELETE CASCADE,
    user_id     BIGINT      NOT NULL,
    client_ip   VARCHAR(64) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_contest_votes_entry_user UNIQUE (entry_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_contest_votes_contest_user ON contest_votes (contest_id, user_id);
CREATE INDEX IF NOT EXISTS idx_contest_votes_entry_created ON contest_votes (entry_id, created_at);

CREATE TABLE IF NOT EXISTS contest_awards (
    id          BIGSERIAL PRIMARY KEY,
    contest_id  BIGINT         NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    entry_id    BIGINT         NOT NULL REFERENCES contest_entries(id) ON DELETE CASCADE,
    user_id     BIGINT         NOT NULL,
    place       INTEGER        NOT NULL,
    prize_type  VARCHAR(20)    NOT NULL,
    amount      NUMERIC(20, 8) NOT NULL DEFAULT 0,
    label       VARCHAR(200)   NOT NULL DEFAULT '',
    -- pending | granting | granted | failed
    status      VARCHAR(20)    NOT NULL DEFAULT 'pending',
    note        VARCHAR(500)   NOT NULL DEFAULT '',
    granted_at  TIMESTAMPTZ    NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_contest_awards_entry UNIQUE (contest_id, entry_id)
);
