package app

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/cg219/nowplaying/internal/database"
)

type Scrobbler struct {
    Username string
    Duration time.Duration
    db *database.Queries
}

type Scrobble struct {
    ArtistName string
    TrackName string
    Timestamp int
    AlbumArtist string
    AlbumName string
    TrackNumber string
    Mbid string
    Source string
    Duration int
    Uid int
}

func NewScrobbler(u string, db *database.Queries) *Scrobbler {
    return &Scrobbler{
        Username: u,
        db: db,
        Duration: time.Second * 15,
    }
}

func (s *Scrobbler) Listen(ctx context.Context, out *chan any) error {
    done := make(chan bool)
    timer := time.NewTicker(s.Duration)
    defer close(done)

    for {
        select {
        case <- done:
            return nil
        case <- timer.C:
            err := s.CheckLastTrack(ctx)

            if err != nil {
                log.Printf("Oops: %s\n", err)
                return err
            }
        }
    }
}

func (s *Scrobbler) Auth(ctx context.Context) error {
    return nil
}

func (s *Scrobbler) AuthWithDB(ctx context.Context) error {
    return nil
}

func (s *Scrobbler) Scrobble(ctx context.Context, sc Scrobble) error {
    dbValue, err := s.db.GetLatestTrack(ctx)

    if err == sql.ErrNoRows {
        s.db.SaveScrobble(ctx, scrobbleToParams(sc))
        return nil
    }

    if sc.ArtistName == dbValue.ArtistName && sc.TrackName == dbValue.TrackName && sc.Duration == int(dbValue.Duration) && sc.Timestamp <= int(dbValue.Timestamp + dbValue.Duration) {
        return nil
    }

    s.db.SaveScrobble(ctx, scrobbleToParams(sc))
    return nil
}

func scrobbleToParams(sc Scrobble) database.SaveScrobbleParams {
    return database.SaveScrobbleParams{
        ArtistName: sc.ArtistName,
        TrackName: sc.TrackName,
        Timestamp: int64(sc.Timestamp),
        AlbumName: sql.NullString{ String: sc.AlbumName, Valid: sc.ArtistName != "" },
        AlbumArtist: sql.NullString{ String: sc.AlbumArtist, Valid: sc.AlbumArtist != "" },
        Source: sql.NullString{ String: sc.Source, Valid: sc.Source != "" },
        Mbid: sql.NullString{ String: sc.Mbid, Valid: sc.Mbid != "" },
        TrackNumber: sql.NullString{ String: sc.TrackNumber, Valid: sc.TrackNumber != "" },
        Duration: int64(sc.Duration),
        Uid: int64(sc.Uid),
    }
}

func (s *Scrobbler) CheckLastTrack(ctx context.Context) error {
    // log.Printf("LastFM: %s - %s\n",tracklist.Recent.Tracks[0].Artist.Name, tracklist.Recent.Tracks[0].Name)
    dbValue, err := s.db.GetLatestTrack(ctx)
    if err != nil {
        if err == sql.ErrNoRows {
            log.Println("Scrobbler: No Tracks Yet")
            return nil
        }

        return err
    }

    log.Printf("Scrobbler: %s - %s\n", dbValue.ArtistName, dbValue.TrackName)
    return nil
}
