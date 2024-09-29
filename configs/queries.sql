-- name: GetLastFMSession :one
SELECT lastfm_session_name, lastfm_session_key
FROM users
WHERE username = ?;

-- name: GetSpotifySession :one
SELECT spotify_access_token, spotify_refresh_token
FROM users
WHERE username = ?;

-- name: SaveLastFMSession :exec
UPDATE users
SET lastfm_session_name = ?,
    lastfm_session_key = ?
WHERE username = ?;

-- name: SaveSpotifySession :exec
UPDATE users
SET spotify_access_token = ?,
    spotify_refresh_token = ?
WHERE username = ?;

-- name: SaveUser :exec
INSERT INTO users(username)
VALUES(?);
