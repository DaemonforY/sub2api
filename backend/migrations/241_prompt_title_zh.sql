-- Chinese titles for prompt-library entries whose source title is English (translated from a
-- bundled dictionary or an admin-configured model). Shown instead of the source title; cleared
-- when the source changes the title. Keyword search covers both.
ALTER TABLE prompt_items ADD COLUMN IF NOT EXISTS title_zh VARCHAR(200) NOT NULL DEFAULT '';

CREATE OR REPLACE FUNCTION prompt_items_search_text() RETURNS trigger AS $$
BEGIN
    NEW.search_text := lower(
        NEW.title || ' ' || NEW.title_zh || ' ' || NEW.description || ' ' || array_to_string(NEW.tags, ' ') || ' ' ||
        COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(NEW.source_tags)), '') || ' ' ||
        left(NEW.prompt, 800));
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prompt_items_search_text ON prompt_items;
CREATE TRIGGER trg_prompt_items_search_text
    BEFORE INSERT OR UPDATE OF title, title_zh, description, tags, source_tags, prompt ON prompt_items
    FOR EACH ROW EXECUTE FUNCTION prompt_items_search_text();
