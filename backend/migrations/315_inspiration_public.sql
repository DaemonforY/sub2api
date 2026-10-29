-- 诗词灵感 for users (/poems): what each paid 按画面找 cost, for the balance audit and the admin stats.
CREATE TABLE IF NOT EXISTS lib_find_charges (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT        NOT NULL,
    cost       NUMERIC(20, 8) NOT NULL,
    scene      TEXT          NOT NULL DEFAULT '',
    model      TEXT          NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lib_find_charges_user ON lib_find_charges (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lib_find_charges_created ON lib_find_charges (created_at);
