-- name: GetAllLegacyFeeds :many
SELECT content_hash, content, etag, user_db_id
FROM legacy_feeds;

-- name: GetLegacyFeeds :one
SELECT user_id, content_hash, content, etag
FROM legacy_feeds
INNER JOIN users ON legacy_feeds.user_db_id = users.db_id
WHERE user_id = $1
LIMIT 1;

-- name: GetLegacyFeedsEtag :one
select
    etag
from legacy_feeds
inner join users on legacy_feeds.user_db_id = users.db_id
where user_id = $1
limit 1;

-- name: UpdateLegacyFeeds :one
INSERT INTO legacy_feeds (user_db_id, content_hash, content, etag)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_db_id) DO UPDATE
SET content_hash = excluded.content_hash,
    content = excluded.content,
    etag = excluded.etag
RETURNING db_id;
