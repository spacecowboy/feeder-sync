-- name: InsertDevice :one
INSERT INTO devices (device_id, device_name, last_seen, user_db_id)
VALUES ($1, $2, $3, $4) RETURNING db_id;

-- name: DeleteDevice :exec
DELETE FROM devices
WHERE user_db_id = $1 AND device_id = $2;

-- name: SelectAllDevices :many
SELECT user_db_id, device_id, device_name, last_seen FROM devices;

-- name: SelectDevices :many
SELECT users.db_id, user_id, device_id, device_name, last_seen
FROM devices
INNER JOIN users ON devices.user_db_id = users.db_id
WHERE user_id = $1;

-- name: SelectLegacyDevicesEtag :one
SELECT sha256(convert_to(string_agg(device_name, '' ORDER BY device_name), 'UTF8'))
FROM devices
INNER JOIN users ON devices.user_db_id = users.db_id
WHERE legacy_sync_code = $1
GROUP BY user_db_id;

-- name: UpdateLastSeenForDevice :exec
UPDATE devices
SET last_seen = $1
WHERE user_db_id = $2 AND device_id = $3;
