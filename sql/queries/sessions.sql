-- name: GetLastFMSession :one
SELECT lastfm_session_name, lastfm_session_key
FROM users
WHERE username = ?;

-- name: GetSpotifySession :one
SELECT spotify_access_token, spotify_refresh_token
FROM users
WHERE username = ?;

-- name: GetTwitterSession :one
SELECT twitter_request_token, twitter_request_secret, twitter_oauth_token, twitter_oauth_secret
FROM users
WHERE username = ?;

-- name: GetTwitterSessionByRequestToken :one
SELECT twitter_request_token, twitter_request_secret, twitter_oauth_token, twitter_oauth_secret
FROM users
WHERE twitter_request_token = ?;

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

-- name: SaveTwitterSession :exec
UPDATE users
SET twitter_request_token = ?,
    twitter_request_secret = ?,
    twitter_oauth_token = ?,
    twitter_oauth_secret = ?
WHERE username = ?;

-- name: UpdateSpotifyAccessToken :exec
UPDATE users
SET spotify_access_token = ?
WHERE username = ?;

-- name: GetUserSession :one
SELECT accessToken, refreshToken, valid
FROM sessions
WHERE accessToken = ? AND refreshToken = ?
LIMIT 1;

-- name: SaveUserSession :exec
INSERT INTO sessions(accessToken, refreshToken)
VALUES(?, ?);

-- name: InvalidateUserSession :exec
UPDATE sessions
SET valid = 0
WHERE accessToken = ? AND refreshToken = ?;

-- name: SaveMusicSession :exec
INSERT INTO music_sessions(data, type, active, uid)
VALUES(?, ?, ?, ?);

-- name: GetActiveMusicSessions :many
SELECT id, data, type, active
FROM music_sessions
WHERE active = 1;

-- name: GetUserMusicSessions :many
SELECT id, data, type, active
FROM music_sessions
WHERE uid = ?;

-- name: RemoveInactiveMusicSessions :exec
DELETE FROM music_sessions
WHERE active = 0;

-- name: DeactivateMusicSession :exec
UPDATE music_sessions
SET active = 0
WHERE id = ?;

-- name: ActivateMusicSession :exec
UPDATE music_sessions
SET active = 1
WHERE id = ?;
