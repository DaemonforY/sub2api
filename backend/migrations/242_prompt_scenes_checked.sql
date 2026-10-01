-- Scenes of a source item that were checked by a model (bundled corrections or the admin-configured
-- model). Syncs keep checked scenes while the source prompt is unchanged; a changed prompt is
-- classified again.
ALTER TABLE prompt_items ADD COLUMN IF NOT EXISTS scenes_checked BOOLEAN NOT NULL DEFAULT FALSE;
