-- +goose Up
CREATE TABLE schools (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    jenjang     VARCHAR(20),
    city        VARCHAR(100),
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_schools_name UNIQUE (name)
);
CREATE INDEX idx_schools_name ON schools USING GIN (to_tsvector('simple', name));

-- +goose Down
DROP TABLE IF EXISTS schools;
