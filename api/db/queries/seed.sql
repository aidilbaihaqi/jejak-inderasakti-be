-- name: UpsertQuestion :exec
INSERT INTO questions (id, site, level, type, prompt, options, explanation, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE SET
    site              = EXCLUDED.site,
    level             = EXCLUDED.level,
    type              = EXCLUDED.type,
    prompt            = EXCLUDED.prompt,
    options           = EXCLUDED.options,
    explanation       = EXCLUDED.explanation,
    active            = EXCLUDED.active;

-- name: UpsertSchool :exec
INSERT INTO schools (name, jenjang, city, verified)
VALUES ($1, $2, $3, TRUE)
ON CONFLICT (name) DO UPDATE SET
    jenjang = EXCLUDED.jenjang,
    city    = EXCLUDED.city;
