-- AI 学习: tutor questions on 大数据 articles are recorded under the page path
-- (e.g. bigdata/flink/08-watermark-window), longer than a lesson id.
ALTER TABLE learn_runs ALTER COLUMN lesson_id TYPE VARCHAR(64);
