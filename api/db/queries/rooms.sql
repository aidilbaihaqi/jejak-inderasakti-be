-- name: LockRoomCreation :exec
SELECT pg_advisory_xact_lock(4201);

-- name: CountActiveRooms :one
SELECT COUNT(*) FROM rooms
WHERE status != 'ended' AND created_at > NOW() - INTERVAL '2 hours';

-- name: InsertRoom :one
INSERT INTO rooms (pin, host_id, jenjang, short_session, accuracy_mode, question_ids)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetActiveRoomByPIN :one
SELECT id, pin, host_id, jenjang, short_session, accuracy_mode, status, question_ids, started_at
FROM rooms
WHERE pin = $1 AND status != 'ended';

-- name: GetRoomByID :one
SELECT id, pin, host_id, jenjang, short_session, accuracy_mode, status, question_ids, started_at
FROM rooms
WHERE id = $1;

-- name: LockRoomForJoin :one
SELECT id FROM rooms WHERE id = $1 AND status != 'ended' FOR UPDATE;

-- name: MarkRoomStarted :execrows
UPDATE rooms SET status = 'running', started_at = NOW() WHERE id = $1 AND status = 'lobby';

-- name: MarkRoomEnded :exec
UPDATE rooms SET status = 'ended', ended_at = NOW() WHERE id = $1 AND status != 'ended';
