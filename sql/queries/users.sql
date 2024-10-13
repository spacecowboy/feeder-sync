-- name: GetUserDbId :one
SELECT db_id FROM users WHERE user_id = $1 LIMIT 1;

-- name: InsertUser :exec
INSERT INTO users (user_id, legacy_sync_code) VALUES ($1, $2) RETURNING db_id;

-- name: SelectAllUsers :many
SELECT db_id, user_id
FROM users;

-- name: SelectUserDbIdBySyncCode :one
SELECT db_id FROM users WHERE legacy_sync_code = $1 LIMIT 1;

-- name: SelectUserDbId :one
SELECT db_id FROM users WHERE user_id = $1 LIMIT 1;

