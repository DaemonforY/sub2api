-- AI 助教: the teacher picks 标准 (the learning site's model) or 经济 (a cheaper model) per assistant.
ALTER TABLE tutors ADD COLUMN IF NOT EXISTS model_tier VARCHAR(10) NOT NULL DEFAULT 'standard';
