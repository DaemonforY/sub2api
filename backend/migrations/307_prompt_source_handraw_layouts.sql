-- 手绘图型 (yang0/handraw-style, MIT with attribution): the repository's layouts — comic storyboards,
-- infographics, e-commerce images, social cards and IP character sheets — next to its styles (292).
-- Each layout's prompt is its own Markdown file; the sample image is the cover. Synced daily.
INSERT INTO prompt_sources (id, name, format, url, homepage) VALUES
    ('handraw-layouts', '手绘图型 · yang0/handraw-style', 'handraw-layouts', 'https://raw.githubusercontent.com/yang0/handraw-style/master/skills/handdraw-style-prompter/references/layouts.json', 'https://github.com/yang0/handraw-style')
ON CONFLICT (id) DO NOTHING;
