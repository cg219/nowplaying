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
