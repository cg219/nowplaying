// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: users.sql

package database

import (
	"context"
	"database/sql"
)

const canResetPassword = `-- name: CanResetPassword :one
SELECT reset_time > ? AS valid, username
FROM users
WHERE reset = ?
`

type CanResetPasswordParams struct {
	ResetTime sql.NullInt64
	Reset     sql.NullString
}

type CanResetPasswordRow struct {
	Valid    bool
	Username string
}

func (q *Queries) CanResetPassword(ctx context.Context, arg CanResetPasswordParams) (CanResetPasswordRow, error) {
	row := q.db.QueryRowContext(ctx, canResetPassword, arg.ResetTime, arg.Reset)
	var i CanResetPasswordRow
	err := row.Scan(&i.Valid, &i.Username)
	return i, err
}

const checkValidApiKey = `-- name: CheckValidApiKey :one
SELECT EXISTS (
    SELECT 1
    FROM apikeys
    WHERE key = ?
) as valid
`

func (q *Queries) CheckValidApiKey(ctx context.Context, key string) (int64, error) {
	row := q.db.QueryRowContext(ctx, checkValidApiKey, key)
	var valid int64
	err := row.Scan(&valid)
	return valid, err
}

const getApiKeysForUid = `-- name: GetApiKeysForUid :many
SELECT key, name
FROM apikeys
WHERE uid = ?
`

type GetApiKeysForUidRow struct {
	Key  string
	Name string
}

func (q *Queries) GetApiKeysForUid(ctx context.Context, uid sql.NullInt64) ([]GetApiKeysForUidRow, error) {
	rows, err := q.db.QueryContext(ctx, getApiKeysForUid, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetApiKeysForUidRow
	for rows.Next() {
		var i GetApiKeysForUidRow
		if err := rows.Scan(&i.Key, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

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

const getUserFromApiKey = `-- name: GetUserFromApiKey :one
SELECT username
FROM users
JOIN apikeys
ON users.id = apikeys.uid
WHERE apikeys.key = ?
`

func (q *Queries) GetUserFromApiKey(ctx context.Context, key string) (string, error) {
	row := q.db.QueryRowContext(ctx, getUserFromApiKey, key)
	var username string
	err := row.Scan(&username)
	return username, err
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

const resetPassword = `-- name: ResetPassword :exec
UPDATE users
SET reset = NULL,
    reset_time = NULL,
    password = ?
WHERE reset = ? AND reset_time > ?
`

type ResetPasswordParams struct {
	Password  interface{}
	Reset     sql.NullString
	ResetTime sql.NullInt64
}

func (q *Queries) ResetPassword(ctx context.Context, arg ResetPasswordParams) error {
	_, err := q.db.ExecContext(ctx, resetPassword, arg.Password, arg.Reset, arg.ResetTime)
	return err
}

const saveApiKey = `-- name: SaveApiKey :exec
INSERT INTO apikeys(name, key, uid)
VALUES(?, ?, ?)
`

type SaveApiKeyParams struct {
	Name string
	Key  string
	Uid  sql.NullInt64
}

func (q *Queries) SaveApiKey(ctx context.Context, arg SaveApiKeyParams) error {
	_, err := q.db.ExecContext(ctx, saveApiKey, arg.Name, arg.Key, arg.Uid)
	return err
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

const setPasswordReset = `-- name: SetPasswordReset :exec
UPDATE users
SET reset = ?,
    reset_time = ?
WHERE username = ?
`

type SetPasswordResetParams struct {
	Reset     sql.NullString
	ResetTime sql.NullInt64
	Username  string
}

func (q *Queries) SetPasswordReset(ctx context.Context, arg SetPasswordResetParams) error {
	_, err := q.db.ExecContext(ctx, setPasswordReset, arg.Reset, arg.ResetTime, arg.Username)
	return err
}
