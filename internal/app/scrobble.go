package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/cg219/nowplaying/internal/database"
)

type Scrobbler struct {
    Username string
    Duration time.Duration
    db *database.Queries
    Id int
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
    Progress int
}

type ScrobbleEncoded struct {
    Username string `json:"u"`
    Duration int `json:"d"`
}

func NewScrobbler(u string, db *database.Queries) *Scrobbler {
    return &Scrobbler{
        Username: u,
        db: db,
        Duration: time.Second * 15,
    }
}

func NewScrobblerFromEncoded(encoded []byte, db *database.Queries) *Scrobbler {
    s := &Scrobbler{ db: db }
    s.Decode(encoded)
    return s
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

func (s *Scrobbler) Encode() []byte {
    data := &ScrobbleEncoded{ Username: s.Username, Duration: int(s.Duration.Milliseconds()) }
    encoded, _ := json.Marshal(data)
    return encoded
}

func (s *Scrobbler) Decode(encoded []byte) error {
    var data ScrobbleEncoded
    err := json.Unmarshal(encoded, &data)
    if err != nil {
        return fmt.Errorf("unmarshal fail: %s", err.Error())
    }

    s.Username = data.Username
    s.Duration = time.Duration(data.Duration)

    return nil
}

func (s *Scrobbler) Auth(ctx context.Context) error {
    return nil
}

func (s *Scrobbler) AuthWithDB(ctx context.Context) error {
    return nil
}

func (s *Scrobbler) Scrobble(ctx context.Context, sc Scrobble) bool {
    dbValue, err := s.db.GetLatestTrack(ctx, int64(sc.Uid))

    if err == sql.ErrNoRows {
        if sc.Duration > int(time.Duration(time.Second * 30).Milliseconds()) {
            if sc.Progress < int(time.Duration(time.Second * 30).Milliseconds()) {
                return false
            }
        } else {
            if float64(sc.Progress) < math.Round(float64(sc.Duration) * .5) {
                return false
            }
        }

        s.db.SaveScrobble(ctx, scrobbleToParams(sc))
        return false
    }

    if sc.ArtistName == dbValue.ArtistName &&
        sc.TrackName == dbValue.TrackName &&
        sc.Duration == int(dbValue.Duration) &&
        sc.Timestamp <= int(dbValue.Timestamp + dbValue.Duration) {
        return false
    }

    if sc.Duration > int(time.Duration(time.Second * 30).Milliseconds()) {
        if sc.Progress < int(time.Duration(time.Second * 30).Milliseconds()) {
            return false
        }
    } else {
        if float64(sc.Progress) < math.Round(float64(sc.Duration) * .5) {
            return false
        }
    }

    s.db.SaveScrobble(ctx, scrobbleToParams(sc))
    return true
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
    user, _ := s.db.GetUser(ctx, s.Username)
    dbValue, err := s.db.GetLatestTrack(ctx, user.ID)
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
