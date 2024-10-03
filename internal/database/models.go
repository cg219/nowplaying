// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package database

import (
	"database/sql"
)

type Scrobble struct {
	ID          int64
	ArtistName  string
	TrackName   string
	AlbumName   sql.NullString
	AlbumArtist sql.NullString
	TrackNumber sql.NullString
	Duration    int64
	Timestamp   int64
	Source      sql.NullString
	Mbid        sql.NullString
}

type Session struct {
	Accesstoken  string
	Refreshtoken string
	Valid        sql.NullInt64
}

type User struct {
	ID                  int64
	Username            string
	SpotifyAccessToken  sql.NullString
	SpotifyRefreshToken sql.NullString
	LastfmSessionName   sql.NullString
	LastfmSessionKey    sql.NullString
	Password            interface{}
}
