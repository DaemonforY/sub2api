-- Static-site hosting for subscribers: each site is served at <name>.<sites domain>. Files live on
-- disk (<sites dir>/<id>/v<version>/); the rows track ownership, status and billing.
CREATE TABLE IF NOT EXISTS sites (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(40) NOT NULL UNIQUE,
    title VARCHAR(100) NOT NULL DEFAULT '',
    -- active | disabled (by an admin) | unpaid (renewal failed) | lapsed (subscription ended)
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    status_reason VARCHAR(300) NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 0,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    -- Sites beyond the subscription's free allowance are paid for every 30 days from the balance.
    paid BOOLEAN NOT NULL DEFAULT FALSE,
    paid_until TIMESTAMPTZ,
    -- When the owner's subscription was first seen missing (grace period, then offline).
    lapsed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sites_user ON sites (user_id);
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites (status);

CREATE TABLE IF NOT EXISTS site_versions (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, version)
);

-- Charges for paid sites (shown to the user next to the site).
CREATE TABLE IF NOT EXISTS site_charges (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT REFERENCES sites(id) ON DELETE SET NULL,
    site_name VARCHAR(40) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20, 8) NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_site_charges_user ON site_charges (user_id, created_at DESC);

-- Abuse reports from visitors (the badge on every hosted page links to the report form).
CREATE TABLE IF NOT EXISTS site_reports (
    id BIGSERIAL PRIMARY KEY,
    site_id BIGINT REFERENCES sites(id) ON DELETE SET NULL,
    site_name VARCHAR(40) NOT NULL,
    reason VARCHAR(40) NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    contact VARCHAR(200) NOT NULL DEFAULT '',
    reporter_ip VARCHAR(64) NOT NULL DEFAULT '',
    -- open | resolved | dismissed
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    handled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_site_reports_status ON site_reports (status, created_at DESC);
