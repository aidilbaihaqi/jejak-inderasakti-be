-- name: InsertAnswer :execrows
INSERT INTO answers (room_player_id, question_id, served_at, answered_at, option_id, correct, points)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (room_player_id, question_id) DO NOTHING;

-- name: UpdatePlayerProgress :exec
UPDATE room_players
SET score = $2, correct_count = $3, total_ms = $4, current_index = $5, streak = $6,
    finished_at = CASE WHEN $7::bool THEN COALESCE(finished_at, NOW()) ELSE finished_at END
WHERE id = $1;
