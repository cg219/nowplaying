-- name: GetLatestTrack :one
SELECT artist_name, track_name, timestamp, duration
FROM scrobbles
WHERE uid = ?
ORDER BY timestamp DESC
LIMIT 1;

-- name: SaveScrobble :exec
INSERT INTO scrobbles(artist_name, track_name, album_name, album_artist, mbid, track_number, duration, timestamp, source, uid)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: RemoveScrobble :exec
DELETE FROM scrobbles
WHERE id = ?;
