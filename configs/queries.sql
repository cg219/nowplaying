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

-- name: UpdateSpotifyAccessToken :exec
UPDATE users
SET spotify_access_token = ?
WHERE username = ?;

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

-- name: GetLatestTrack :one
SELECT artist_name, track_name, timestamp, duration
FROM scrobbles
ORDER BY timestamp DESC
LIMIT 1;

-- name: SaveScrobble :exec
INSERT INTO scrobbles(artist_name, track_name, album_name, album_artist, mbid, track_number, duration, timestamp, source)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: RemoveScrobble :exec
DELETE FROM scrobbles
WHERE id = ?;

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
