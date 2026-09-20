-- name: ListSelectableQuestions :many
SELECT id, site, level FROM questions WHERE active = TRUE AND site BETWEEN 1 AND 5 ORDER BY id;
