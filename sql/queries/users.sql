-- name: SaveUser :exec
INSERT INTO users(username, password)
VALUES(?, ?);

-- name: GetUser :one
SELECT id, username
FROM users
WHERE username = ?;

-- name: GetUserWithPassword :one
SELECT username, password
FROM users
WHERE username = ?;

-- name: SetPasswordReset :exec
UPDATE users
SET reset = ?,
    reset_time = ?
WHERE username = ?;

-- name: ResetPassword :exec
UPDATE users
SET reset = NULL,
    reset_time = NULL,
    password = ?
WHERE reset = ? AND reset_time > ?;

-- name: CanResetPassword :one
SELECT reset_time > ? AS valid, username
FROM users
WHERE reset = ?;

-- name: SaveApiKey :exec
INSERT INTO apikeys(name, key, uid)
VALUES(?, ?, ?);

-- name: GetApiKeysForUid :many
SELECT key, name
FROM apikeys
WHERE uid = ?;

-- name: CheckValidApiKey :one
SELECT EXISTS (
    SELECT 1
    FROM apikeys
    WHERE key = ?
) as valid;

-- name: GetUserFromApiKey :one
SELECT username
FROM users
JOIN apikeys
ON users.id = apikeys.uid
WHERE apikeys.key = ?;
