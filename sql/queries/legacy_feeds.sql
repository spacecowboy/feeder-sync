-- name: SelectAllLegacyFeeds :many
SELECT content_hash, content, etag, user_db_id
FROM legacy_feeds;

-- name: SelectLegacyFeeds :many
SELECT user_id, content_hash, content, etag
FROM legacy_feeds
INNER JOIN users ON legacy_feeds.user_db_id = users.db_id
WHERE user_id = $1;

-- name: UpdateLegacyFeeds :exec
INSERT INTO legacy_feeds (user_db_id, content_hash, content, etag)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_db_id) DO UPDATE
SET content_hash = excluded.content_hash,
    content = excluded.content,
    etag = excluded.etag;
