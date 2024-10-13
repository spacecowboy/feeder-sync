// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: articles.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const insertArticle = `-- name: InsertArticle :exec
INSERT INTO articles (user_db_id, identifier, read_time, updated_at)
VALUES ($1, $2, $3, $4)
`

type InsertArticleParams struct {
	UserDbID   int64
	Identifier string
	ReadTime   pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
}

func (q *Queries) InsertArticle(ctx context.Context, arg InsertArticleParams) error {
	_, err := q.db.Exec(ctx, insertArticle,
		arg.UserDbID,
		arg.Identifier,
		arg.ReadTime,
		arg.UpdatedAt,
	)
	return err
}

const selectAllArticles = `-- name: SelectAllArticles :many
SELECT read_time, identifier, updated_at, user_db_id
FROM articles
`

type SelectAllArticlesRow struct {
	ReadTime   pgtype.Timestamptz
	Identifier string
	UpdatedAt  pgtype.Timestamptz
	UserDbID   int64
}

func (q *Queries) SelectAllArticles(ctx context.Context) ([]SelectAllArticlesRow, error) {
	rows, err := q.db.Query(ctx, selectAllArticles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SelectAllArticlesRow
	for rows.Next() {
		var i SelectAllArticlesRow
		if err := rows.Scan(
			&i.ReadTime,
			&i.Identifier,
			&i.UpdatedAt,
			&i.UserDbID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const selectArticles = `-- name: SelectArticles :many
SELECT user_id, read_time, identifier, updated_at
FROM articles
INNER JOIN users ON articles.user_db_id = users.db_id
WHERE users.user_id = $1 AND updated_at > $2
ORDER BY read_time DESC
LIMIT 1000
`

type SelectArticlesParams struct {
	UserID    string
	UpdatedAt pgtype.Timestamptz
}

type SelectArticlesRow struct {
	UserID     string
	ReadTime   pgtype.Timestamptz
	Identifier string
	UpdatedAt  pgtype.Timestamptz
}

func (q *Queries) SelectArticles(ctx context.Context, arg SelectArticlesParams) ([]SelectArticlesRow, error) {
	rows, err := q.db.Query(ctx, selectArticles, arg.UserID, arg.UpdatedAt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SelectArticlesRow
	for rows.Next() {
		var i SelectArticlesRow
		if err := rows.Scan(
			&i.UserID,
			&i.ReadTime,
			&i.Identifier,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
