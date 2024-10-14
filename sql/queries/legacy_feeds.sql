-- name: GetAllLegacyFeeds :many
SELECT
    *
FROM legacy_feeds;

-- name: GetLegacyFeeds :one
SELECT
    *
FROM legacy_feeds
WHERE user_db_id = $1
LIMIT 1;

-- name: GetLegacyFeedsEtag :one
SELECT etag
FROM legacy_feeds
WHERE user_db_id = $1
LIMIT 1;

-- name: UpdateLegacyFeeds :one
INSERT INTO legacy_feeds (user_db_id, content_hash, content, etag)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_db_id) DO UPDATE
SET content_hash = excluded.content_hash,
content = excluded.content,
etag = excluded.etag
RETURNING db_id;
