-- name: GetRoomLeaderboard :many
SELECT rp.nickname, COALESCE(s.name, '')::text AS school_name, rp.jenjang,
       rp.score, rp.correct_count, rp.total_ms,
       (RANK() OVER (ORDER BY rp.score DESC, rp.correct_count DESC, rp.total_ms ASC))::int AS rank
FROM room_players rp
LEFT JOIN schools s ON s.id = rp.school_id
WHERE rp.room_id = $1
ORDER BY rank, rp.nickname;

-- name: GetSchoolLeaderboard :many
SELECT s.name AS school_name, AVG(top5.score)::float8 AS avg_score, COUNT(top5.id)::int AS player_count
FROM (
    SELECT rp.id, rp.school_id, rp.score,
           ROW_NUMBER() OVER (PARTITION BY rp.school_id ORDER BY rp.score DESC) AS rnk
    FROM room_players rp
    WHERE rp.finished_at IS NOT NULL
) top5
JOIN schools s ON s.id = top5.school_id
WHERE top5.rnk <= 5 AND s.name != ALL($1::text[])
GROUP BY s.id, s.name
HAVING COUNT(top5.id) >= 3
ORDER BY avg_score DESC, s.name;
