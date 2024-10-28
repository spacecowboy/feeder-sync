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

-- name: DeleteUser :many
DELETE FROM users WHERE user_id = $1 RETURNING user_id;

-- name: GetUsersWithoutDevices :many
SELECT *
FROM users
WHERE db_id NOT IN (
    SELECT user_db_id
    FROM devices
)
LIMIT 10000;

-- name: GetUsersWithDevices :many
SELECT
    sqlc.embed(users),
    max(devices.last_seen) AS last_seen
FROM users
INNER JOIN devices ON users.db_id = devices.user_db_id
GROUP BY users.db_id
ORDER BY last_seen DESC
LIMIT 10000;
