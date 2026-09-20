-- name: SearchSchools :many
SELECT id, name, jenjang FROM schools
WHERE verified AND name != ALL($2::text[]) AND ($1::text = '' OR name ILIKE '%' || $1::text || '%')
ORDER BY name
LIMIT 20;

-- name: ListSchoolsByName :many
SELECT id, name, jenjang FROM schools WHERE name = ANY($1::text[]) ORDER BY name;

-- name: GetSchoolName :one
SELECT name FROM schools WHERE id = $1;
