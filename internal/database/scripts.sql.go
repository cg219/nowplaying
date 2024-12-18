// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: scripts.sql

package database

import (
	"context"
	"database/sql"
)

const addToHistory = `-- name: AddToHistory :exec
INSERT INTO history_spotify(artist_name, album_name, track_name, timestamp)
VALUES(?, ?, ?, ?)
`

type AddToHistoryParams struct {
	ArtistName string
	AlbumName  sql.NullString
	TrackName  string
	Timestamp  int64
}

func (q *Queries) AddToHistory(ctx context.Context, arg AddToHistoryParams) error {
	_, err := q.db.ExecContext(ctx, addToHistory,
		arg.ArtistName,
		arg.AlbumName,
		arg.TrackName,
		arg.Timestamp,
	)
	return err
}

const historyToScrobbles = `-- name: HistoryToScrobbles :exec
INSERT INTO scrobbles(artist_name, track_name, album_name, timestamp, duration, source, uid, track_number, artist_name)
SELECT artist_name, track_name, album_name, timestamp, 30000, "spotify-local", 1, 0, artist_name
FROM history_spotify
`

func (q *Queries) HistoryToScrobbles(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, historyToScrobbles)
	return err
}