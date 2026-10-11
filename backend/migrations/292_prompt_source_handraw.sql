-- 手绘风格图鉴 (yang0/handraw-style, MIT with attribution): 327 numbered hand-drawn styles, each a
-- prompt with a subject placeholder and its sample image as the cover. Items credit the author and
-- link the repository, as the license requires.
INSERT INTO prompt_sources (id, name, format, url, homepage) VALUES
    ('handraw-style', '手绘风格图鉴 · yang0/handraw-style', 'handraw', 'https://raw.githubusercontent.com/yang0/handraw-style/master/skills/handdraw-style-prompter/references/styles.json', 'https://github.com/yang0/handraw-style')
ON CONFLICT (id) DO NOTHING;
