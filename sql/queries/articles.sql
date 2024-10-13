-- name: InsertArticle :one
INSERT INTO articles (user_db_id, identifier, read_time, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING db_id;

-- name: GetAllArticles :many
SELECT read_time, identifier, updated_at, user_db_id
FROM articles;

-- name: GetArticles :many
SELECT user_id, read_time, identifier, updated_at
FROM articles
INNER JOIN users ON articles.user_db_id = users.db_id
WHERE users.user_id = $1 AND updated_at > $2
ORDER BY read_time DESC
LIMIT 1000;
