-- Web-page works: a work can present one of the author's hosted sites (kind = 'site'); the cover
-- images are screenshots. Shown publicly only while the site is live; one work per site.
ALTER TABLE works ADD COLUMN IF NOT EXISTS kind VARCHAR(16) NOT NULL DEFAULT 'image';
ALTER TABLE works ADD COLUMN IF NOT EXISTS site_id BIGINT NULL REFERENCES sites(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_works_site ON works (site_id) WHERE site_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_works_kind_created ON works (kind, created_at DESC);
