-- name: GetHostByEmail :one
SELECT id, email, password_hash, name FROM host_users WHERE email = $1;

-- name: TouchHostLogin :exec
UPDATE host_users SET last_login_at = NOW() WHERE id = $1;

-- name: UpsertHost :exec
INSERT INTO host_users (email, password_hash, name)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name          = EXCLUDED.name;
