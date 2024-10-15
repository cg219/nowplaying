// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: users.sql

package database

import (
	"context"
)

const getUser = `-- name: GetUser :one
SELECT id, username
FROM users
WHERE username = ?
`

type GetUserRow struct {
	ID       int64
	Username string
}

func (q *Queries) GetUser(ctx context.Context, username string) (GetUserRow, error) {
	row := q.db.QueryRowContext(ctx, getUser, username)
	var i GetUserRow
	err := row.Scan(&i.ID, &i.Username)
	return i, err
}

const getUserWithPassword = `-- name: GetUserWithPassword :one
SELECT username, password
FROM users
WHERE username = ?
`

type GetUserWithPasswordRow struct {
	Username string
	Password interface{}
}

func (q *Queries) GetUserWithPassword(ctx context.Context, username string) (GetUserWithPasswordRow, error) {
	row := q.db.QueryRowContext(ctx, getUserWithPassword, username)
	var i GetUserWithPasswordRow
	err := row.Scan(&i.Username, &i.Password)
	return i, err
}

const saveUser = `-- name: SaveUser :exec
INSERT INTO users(username, password)
VALUES(?, ?)
`

type SaveUserParams struct {
	Username string
	Password interface{}
}

func (q *Queries) SaveUser(ctx context.Context, arg SaveUserParams) error {
	_, err := q.db.ExecContext(ctx, saveUser, arg.Username, arg.Password)
	return err
}
