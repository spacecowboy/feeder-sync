-- name: InsertDevice :one
INSERT INTO devices (device_id, device_name, last_seen, legacy_device_id, user_db_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteDevice :exec
DELETE FROM devices
WHERE user_db_id = $1 AND device_id = $2;

-- name: DeleteDeviceWithLegacyId :exec
delete from devices
where user_db_id = $1 and legacy_device_id = $2;

-- name: GetAllDevices :many
SELECT * FROM devices;

-- name: GetDevices :many
SELECT sqlc.embed(devices), sqlc.embed(users)
FROM devices
INNER JOIN users ON devices.user_db_id = users.db_id
WHERE user_id = $1;

-- name: GetLegacyDevicesEtag :one
SELECT sha256(convert_to(string_agg(device_name, '' ORDER BY device_name), 'UTF8'))
FROM devices
INNER JOIN users ON devices.user_db_id = users.db_id
WHERE legacy_sync_code = $1
GROUP BY user_db_id;

-- name: GetLegacyDevice :one
select
    sqlc.embed(devices), sqlc.embed(users)
from devices
inner join users on devices.user_db_id = users.db_id
where legacy_sync_code = $1 and legacy_device_id = $2
limit 1;

-- name: UpdateLastSeenForDevice :exec
UPDATE devices
SET last_seen = $1
WHERE db_id = $2;
