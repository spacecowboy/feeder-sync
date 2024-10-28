-- name: InsertArticle :one
INSERT INTO articles (user_db_id, identifier, read_time, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetAllArticles :many
SELECT
    *
FROM articles;

-- name: GetArticlesUpdatedSince :many
SELECT
    *
FROM articles
WHERE user_db_id = $1 AND updated_at > $2
ORDER BY read_time DESC
LIMIT 1000;

-- name: DeleteArticlesWithUserDbId :many
DELETE FROM articles
WHERE user_db_id = $1
RETURNING identifier;

-- name: DeleteArticlesOlderThanDeviceLastSeen :exec
DELETE FROM articles
WHERE articles.user_db_id = $1 AND articles.updated_at < (
    SELECT devices.last_seen
    FROM devices
    WHERE devices.user_db_id = $1
    ORDER BY devices.last_seen DESC
    LIMIT 1
);
