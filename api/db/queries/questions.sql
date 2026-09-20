-- name: ListSelectableQuestions :many
SELECT id, site, level FROM questions WHERE active = TRUE AND site BETWEEN 1 AND 5 ORDER BY id;

-- name: GetQuestionsByIDs :many
SELECT id, site, level, type, prompt, options, explanation FROM questions WHERE id = ANY($1::text[]);
