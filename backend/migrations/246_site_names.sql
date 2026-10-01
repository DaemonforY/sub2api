-- Chosen site names: after a rename the old name redirects to the site and stays reserved for it
-- for 30 days (previous_name, renamed_at); renames are limited to one every 7 days.
ALTER TABLE sites ADD COLUMN IF NOT EXISTS previous_name VARCHAR(40) NOT NULL DEFAULT '';
ALTER TABLE sites ADD COLUMN IF NOT EXISTS renamed_at TIMESTAMPTZ NULL;
CREATE INDEX IF NOT EXISTS idx_sites_previous_name ON sites (previous_name) WHERE previous_name <> '';
