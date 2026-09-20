-- name: InsertPlayer :one
INSERT INTO room_players (room_id, nickname, school_id, jenjang, avatar, lang)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: CountRoomPlayers :one
SELECT COUNT(*) FROM room_players WHERE room_id = $1;

-- name: ListRoomPlayers :many
SELECT rp.id, rp.nickname, rp.avatar, rp.lang, COALESCE(s.name, '')::text AS school_name,
       rp.score, rp.correct_count, rp.total_ms, rp.current_index, rp.streak, (rp.finished_at IS NOT NULL)::bool AS finished
FROM room_players rp
LEFT JOIN schools s ON s.id = rp.school_id
WHERE rp.room_id = $1
ORDER BY rp.created_at;

-- name: DeletePlayer :exec
DELETE FROM room_players WHERE id = $1;
