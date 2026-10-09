-- AI 助教: teachers' own assistants. `task` is what a 自定义助教 does, in the teacher's words;
-- `suggestions` are the first questions shown to students (any template; empty = the template's).
ALTER TABLE tutors ADD COLUMN IF NOT EXISTS task TEXT NOT NULL DEFAULT '';
ALTER TABLE tutors ADD COLUMN IF NOT EXISTS suggestions JSONB NOT NULL DEFAULT '[]'::jsonb;
