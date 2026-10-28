-- HiveGPT 视频: users clone their own voice (豆包声音复刻 2.0). The operator buys voice slots
-- (speaker IDs, S_xxx) in the 火山引擎 console and lists them in the video settings; a user's clone
-- takes a free slot. The recording itself is not kept: it goes to 豆包 for training and is dropped.
-- consent_* record the user's statement that the voice is their own (or that they have the speaker's
-- separate consent), as 深度合成 rules require for editing someone's voice.
CREATE TABLE IF NOT EXISTS video_voice_clones (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    speaker_id    TEXT           NOT NULL,
    name          TEXT           NOT NULL,
    status        TEXT           NOT NULL DEFAULT 'training', -- training | ready | failed | disabled
    error         TEXT           NOT NULL DEFAULT '',
    trainings     INT            NOT NULL DEFAULT 0,
    price         NUMERIC(20, 8) NOT NULL DEFAULT 0,          -- slot fee taken from the balance
    charged       BOOLEAN        NOT NULL DEFAULT FALSE,
    consent_text  TEXT           NOT NULL,
    consent_ip    TEXT           NOT NULL DEFAULT '',
    consent_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    checked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

-- One live clone per slot; a deleted clone frees its slot for the next user (training overwrites it).
CREATE UNIQUE INDEX IF NOT EXISTS uq_video_voice_clones_speaker ON video_voice_clones (speaker_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_video_voice_clones_user ON video_voice_clones (user_id) WHERE deleted_at IS NULL;
