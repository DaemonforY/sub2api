-- Site hosting, phase 2: review of every published version, access passwords and daily visit stats.

-- sites.version is the version being served (approved); pending_version waits for an admin.
ALTER TABLE sites ADD COLUMN IF NOT EXISTS pending_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS password_hash VARCHAR(100) NOT NULL DEFAULT '';

-- pending | approved | rejected; flags are what the automatic check found.
ALTER TABLE site_versions ADD COLUMN IF NOT EXISTS review_status VARCHAR(20) NOT NULL DEFAULT 'approved';
ALTER TABLE site_versions ADD COLUMN IF NOT EXISTS review_reason VARCHAR(300) NOT NULL DEFAULT '';
ALTER TABLE site_versions ADD COLUMN IF NOT EXISTS review_flags JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE site_versions ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE site_versions ADD COLUMN IF NOT EXISTS excerpt TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_site_versions_pending ON site_versions (created_at) WHERE review_status = 'pending';

CREATE TABLE IF NOT EXISTS site_daily_stats (
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    day DATE NOT NULL,
    views BIGINT NOT NULL DEFAULT 0,
    visitors BIGINT NOT NULL DEFAULT 0,
    bytes BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (site_id, day)
);
