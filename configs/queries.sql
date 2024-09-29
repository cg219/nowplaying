-- name: GetLastFMSession :one
SELECT lastfm_session_name, lastfm_session_key
FROM users
WHERE username = ?;

-- name: SaveLastFMSession :exec
UPDATE users
SET lastfm_session_name = ?,
    lastfm_session_key = ?
WHERE username = ?;

-- name: SaveUser :exec
INSERT INTO users(username)
VALUES(?);
