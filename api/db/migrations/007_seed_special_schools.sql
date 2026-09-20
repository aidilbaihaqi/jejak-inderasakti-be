-- +goose Up
INSERT INTO schools (name, jenjang, verified) VALUES
    ('Sekolah lain', NULL, TRUE),
    ('Umum / General visitor', 'Umum', TRUE)
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM schools WHERE name IN ('Sekolah lain', 'Umum / General visitor');
