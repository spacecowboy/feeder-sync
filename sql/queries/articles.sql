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
