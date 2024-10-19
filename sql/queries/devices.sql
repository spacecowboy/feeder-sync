-- name: InsertDevice :one
INSERT INTO devices (
    device_id, device_name, last_seen, legacy_device_id, user_db_id
)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteDevice :many
DELETE FROM devices
WHERE user_db_id = $1 AND device_id = $2
RETURNING device_id;

-- name: DeleteDeviceWithLegacyId :many
DELETE FROM devices
WHERE user_db_id = $1 AND legacy_device_id = $2
RETURNING legacy_device_id;

-- name: GetAllDevices :many
SELECT * FROM devices;

-- name: GetDevices :many
SELECT
    *
FROM devices
WHERE user_db_id = $1;

-- name: GetLegacyDevicesEtag :one
SELECT
    sha256(convert_to(string_agg(device_name, '' ORDER BY device_name), 'UTF8'))
FROM devices
WHERE user_db_id = $1
GROUP BY user_db_id;

-- name: GetLegacyDevice :one
SELECT
    *
FROM devices
WHERE user_db_id = $1 AND legacy_device_id = $2
LIMIT 1;

-- name: UpdateLastSeenForDevice :exec
UPDATE devices
SET last_seen = $1
WHERE db_id = $2;
