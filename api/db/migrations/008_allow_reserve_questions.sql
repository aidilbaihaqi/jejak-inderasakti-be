-- +goose Up
-- site 0 = cross-site reserve questions (ids X-01..X-06)
ALTER TABLE questions DROP CONSTRAINT questions_site_check;
ALTER TABLE questions ADD CONSTRAINT questions_site_check CHECK (site BETWEEN 0 AND 5);

-- +goose Down
DELETE FROM questions WHERE site = 0;
ALTER TABLE questions DROP CONSTRAINT questions_site_check;
ALTER TABLE questions ADD CONSTRAINT questions_site_check CHECK (site BETWEEN 1 AND 5);
