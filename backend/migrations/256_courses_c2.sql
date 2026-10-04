-- Courses C2: limited-time price, student discount switch, trial video, daily page views.
ALTER TABLE courses ADD COLUMN IF NOT EXISTS sale_price NUMERIC(10,2) NOT NULL DEFAULT 0;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS sale_ends_at TIMESTAMPTZ NULL;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS edu_discount BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS trial_video_url VARCHAR(500) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS course_daily_stats (
    course_id BIGINT  NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    day       DATE    NOT NULL,
    views     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (course_id, day)
);
