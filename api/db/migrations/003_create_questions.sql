-- +goose Up
CREATE TABLE questions (
    id          VARCHAR(10) PRIMARY KEY,
    site        SMALLINT NOT NULL CHECK (site BETWEEN 1 AND 5),
    level       SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 3),
    type        VARCHAR(10) NOT NULL CHECK (type IN ('mc', 'tf', 'fill')),
    prompt      JSONB NOT NULL,
    options     JSONB NOT NULL,
    explanation JSONB NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    review_flag BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_questions_site_level_active ON questions (site, level) WHERE active = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_questions_site_level_active;
DROP TABLE IF EXISTS questions;
