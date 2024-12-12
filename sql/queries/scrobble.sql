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

-- name: GetRecentScrobbles :many
SELECT artist_name, track_name, timestamp, duration
FROM scrobbles
WHERE uid = ?
ORDER BY timestamp DESC
LIMIT 5;

-- name: GetTopTracksOfYear :many
SELECT track_name, artist_name, count(id) as plays
FROM scrobbles
WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*365))
GROUP BY track_name
ORDER BY plays DESC
LIMIT ?;

-- name: GetTopTracksOfMonth :many
SELECT track_name, artist_name, count(id) as plays
FROM scrobbles
WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*30))
GROUP BY track_name
ORDER BY plays DESC
LIMIT ?;

-- name: GetTopTracksOfWeek :many
SELECT track_name, artist_name, count(id) as plays
FROM scrobbles
WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*7))
GROUP BY track_name
ORDER BY plays DESC
LIMIT ?;

-- name: GetTopArtistsOfYear :many
With splits as (
  SELECT trim(value) as artist
  FROM scrobbles, json_each('["' || replace(artist_name, ',', '","') || '"]')
  WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*365))
)
SELECT artist, count(*) as plays
FROM splits
group by artist
order by plays DESC
limit ?;

-- name: GetTopArtistsOfMonth :many
With splits as (
  SELECT trim(value) as artist
  FROM scrobbles, json_each('["' || replace(artist_name, ',', '","') || '"]')
  WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*30))
)
SELECT artist, count(*) as plays
FROM splits
group by artist
order by plays DESC
limit ?;

-- name: GetTopArtistsOfWeek :many
With splits as (
  SELECT trim(value) as artist
  FROM scrobbles, json_each('["' || replace(artist_name, ',', '","') || '"]')
  WHERE uid = 1 AND (timestamp / 1000) >= (strftime("%s", "now") - (60*60*24*7))
)
SELECT artist, count(*) as plays
FROM splits
group by artist
order by plays DESC
limit ?;
