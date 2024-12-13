-- name: AddToHistory :exec
INSERT INTO history_spotify(artist_name, album_name, track_name, timestamp)
VALUES(?, ?, ?, ?);

-- name: HistoryToScrobbles :exec
INSERT INTO scrobbles(artist_name, track_name, album_name, timestamp, duration, source, uid, track_number, artist_name)
SELECT artist_name, track_name, album_name, timestamp, 30000, "spotify-local", 1, 0, artist_name
FROM history_spotify;
