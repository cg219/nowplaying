// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package database

import (
	"context"
	"database/sql"
)

const getLastFMSession = `-- name: GetLastFMSession :one
SELECT lastfm_session_name, lastfm_session_key
FROM users
WHERE username = ?
`

type GetLastFMSessionRow struct {
	LastfmSessionName sql.NullString
	LastfmSessionKey  sql.NullString
}

func (q *Queries) GetLastFMSession(ctx context.Context, username string) (GetLastFMSessionRow, error) {
	row := q.db.QueryRowContext(ctx, getLastFMSession, username)
	var i GetLastFMSessionRow
	err := row.Scan(&i.LastfmSessionName, &i.LastfmSessionKey)
	return i, err
}

const getLatestTrack = `-- name: GetLatestTrack :one
SELECT artist_name, track_name, timestamp, duration
FROM scrobbles
ORDER BY timestamp DESC
LIMIT 1
`

type GetLatestTrackRow struct {
	ArtistName string
	TrackName  string
	Timestamp  int64
	Duration   int64
}

func (q *Queries) GetLatestTrack(ctx context.Context) (GetLatestTrackRow, error) {
	row := q.db.QueryRowContext(ctx, getLatestTrack)
	var i GetLatestTrackRow
	err := row.Scan(
		&i.ArtistName,
		&i.TrackName,
		&i.Timestamp,
		&i.Duration,
	)
	return i, err
}

const getSpotifySession = `-- name: GetSpotifySession :one
SELECT spotify_access_token, spotify_refresh_token
FROM users
WHERE username = ?
`

type GetSpotifySessionRow struct {
	SpotifyAccessToken  sql.NullString
	SpotifyRefreshToken sql.NullString
}

func (q *Queries) GetSpotifySession(ctx context.Context, username string) (GetSpotifySessionRow, error) {
	row := q.db.QueryRowContext(ctx, getSpotifySession, username)
	var i GetSpotifySessionRow
	err := row.Scan(&i.SpotifyAccessToken, &i.SpotifyRefreshToken)
	return i, err
}

const getUser = `-- name: GetUser :one
SELECT id, username
FROM users
WHERE username = ?
`

type GetUserRow struct {
	ID       int64
	Username string
}

func (q *Queries) GetUser(ctx context.Context, username string) (GetUserRow, error) {
	row := q.db.QueryRowContext(ctx, getUser, username)
	var i GetUserRow
	err := row.Scan(&i.ID, &i.Username)
	return i, err
}

const getUserSession = `-- name: GetUserSession :one
SELECT accessToken, refreshToken, valid
FROM sessions
WHERE accessToken = ? AND refreshToken = ?
LIMIT 1
`

type GetUserSessionParams struct {
	Accesstoken  string
	Refreshtoken string
}

func (q *Queries) GetUserSession(ctx context.Context, arg GetUserSessionParams) (Session, error) {
	row := q.db.QueryRowContext(ctx, getUserSession, arg.Accesstoken, arg.Refreshtoken)
	var i Session
	err := row.Scan(&i.Accesstoken, &i.Refreshtoken, &i.Valid)
	return i, err
}

const getUserWithPassword = `-- name: GetUserWithPassword :one
SELECT username, password
FROM users
WHERE username = ?
`

type GetUserWithPasswordRow struct {
	Username string
	Password interface{}
}

func (q *Queries) GetUserWithPassword(ctx context.Context, username string) (GetUserWithPasswordRow, error) {
	row := q.db.QueryRowContext(ctx, getUserWithPassword, username)
	var i GetUserWithPasswordRow
	err := row.Scan(&i.Username, &i.Password)
	return i, err
}

const invalidateUserSession = `-- name: InvalidateUserSession :exec
UPDATE sessions
SET valid = 0
WHERE accessToken = ? AND refreshToken = ?
`

type InvalidateUserSessionParams struct {
	Accesstoken  string
	Refreshtoken string
}

func (q *Queries) InvalidateUserSession(ctx context.Context, arg InvalidateUserSessionParams) error {
	_, err := q.db.ExecContext(ctx, invalidateUserSession, arg.Accesstoken, arg.Refreshtoken)
	return err
}

const removeScrobble = `-- name: RemoveScrobble :exec
DELETE FROM scrobbles
WHERE id = ?
`

func (q *Queries) RemoveScrobble(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, removeScrobble, id)
	return err
}

const saveLastFMSession = `-- name: SaveLastFMSession :exec
UPDATE users
SET lastfm_session_name = ?,
    lastfm_session_key = ?
WHERE username = ?
`

type SaveLastFMSessionParams struct {
	LastfmSessionName sql.NullString
	LastfmSessionKey  sql.NullString
	Username          string
}

func (q *Queries) SaveLastFMSession(ctx context.Context, arg SaveLastFMSessionParams) error {
	_, err := q.db.ExecContext(ctx, saveLastFMSession, arg.LastfmSessionName, arg.LastfmSessionKey, arg.Username)
	return err
}

const saveScrobble = `-- name: SaveScrobble :exec
INSERT INTO scrobbles(artist_name, track_name, album_name, album_artist, mbid, track_number, duration, timestamp, source)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
`

type SaveScrobbleParams struct {
	ArtistName  string
	TrackName   string
	AlbumName   sql.NullString
	AlbumArtist sql.NullString
	Mbid        sql.NullString
	TrackNumber sql.NullString
	Duration    int64
	Timestamp   int64
	Source      sql.NullString
}

func (q *Queries) SaveScrobble(ctx context.Context, arg SaveScrobbleParams) error {
	_, err := q.db.ExecContext(ctx, saveScrobble,
		arg.ArtistName,
		arg.TrackName,
		arg.AlbumName,
		arg.AlbumArtist,
		arg.Mbid,
		arg.TrackNumber,
		arg.Duration,
		arg.Timestamp,
		arg.Source,
	)
	return err
}

const saveSpotifySession = `-- name: SaveSpotifySession :exec
UPDATE users
SET spotify_access_token = ?,
    spotify_refresh_token = ?
WHERE username = ?
`

type SaveSpotifySessionParams struct {
	SpotifyAccessToken  sql.NullString
	SpotifyRefreshToken sql.NullString
	Username            string
}

func (q *Queries) SaveSpotifySession(ctx context.Context, arg SaveSpotifySessionParams) error {
	_, err := q.db.ExecContext(ctx, saveSpotifySession, arg.SpotifyAccessToken, arg.SpotifyRefreshToken, arg.Username)
	return err
}

const saveUser = `-- name: SaveUser :exec
INSERT INTO users(username, password)
VALUES(?, ?)
`

type SaveUserParams struct {
	Username string
	Password interface{}
}

func (q *Queries) SaveUser(ctx context.Context, arg SaveUserParams) error {
	_, err := q.db.ExecContext(ctx, saveUser, arg.Username, arg.Password)
	return err
}

const saveUserSession = `-- name: SaveUserSession :exec
INSERT INTO sessions(accessToken, refreshToken)
VALUES(?, ?)
`

type SaveUserSessionParams struct {
	Accesstoken  string
	Refreshtoken string
}

func (q *Queries) SaveUserSession(ctx context.Context, arg SaveUserSessionParams) error {
	_, err := q.db.ExecContext(ctx, saveUserSession, arg.Accesstoken, arg.Refreshtoken)
	return err
}

const updateSpotifyAccessToken = `-- name: UpdateSpotifyAccessToken :exec
UPDATE users
SET spotify_access_token = ?
WHERE username = ?
`

type UpdateSpotifyAccessTokenParams struct {
	SpotifyAccessToken sql.NullString
	Username           string
}

func (q *Queries) UpdateSpotifyAccessToken(ctx context.Context, arg UpdateSpotifyAccessTokenParams) error {
	_, err := q.db.ExecContext(ctx, updateSpotifyAccessToken, arg.SpotifyAccessToken, arg.Username)
	return err
}
