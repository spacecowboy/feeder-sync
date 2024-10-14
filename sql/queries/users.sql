-- name: GetUserDbId :one
SELECT db_id FROM users WHERE user_id = $1 LIMIT 1;

-- name: InsertUser :one
INSERT INTO users (user_id, legacy_sync_code)
VALUES ($1, $2)
RETURNING *;

-- name: GetAllUsers :many
SELECT
    db_id,
    user_id
FROM users;

-- name: GetUserDbIdBySyncCode :one
SELECT db_id FROM users WHERE legacy_sync_code = $1 LIMIT 1;

-- name: GetUserBySyncCode :one
SELECT * FROM users WHERE legacy_sync_code = $1 LIMIT 1;

-- name: GetUserByUserId :one
SELECT * FROM users WHERE user_id = $1 LIMIT 1;
