package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/pressly/goose/v3"
	"github.com/tursodatabase/go-libsql"
)

type Config struct {
    LastFM struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
    } `yaml:"lastfm"`
    Turso struct {
        Name string `yaml:"name"`
        Url string `yaml:"url"`
        Token string `yaml:"token"`
    }
    Spotify struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    } `yaml:"spotify"`
    App struct {
        Name string `yaml:"name"`
        Id int `yaml:"id"`
    } `yaml:"app"`
}

type AppCfg struct {
    config Config
    username string
    client *http.Client
    ctx context.Context
    LastFMSession *LastFM
    SpotifySession *Spotify
    listenInterval time.Ticker
    database *database.Queries
    haveNewSessions bool
}

type Session interface {
    AuthWithDB(context.Context) error
    Listen(context.Context, *chan any) error
    Encode() []byte
    Decode([]byte) error
}

func Run(config Config) error {
    cfg := &AppCfg{
        config: config,
        username: config.App.Name,
        client: &http.Client{
            Timeout: time.Second * 60,
        },
        listenInterval: *time.NewTicker(5 * time.Second),
        ctx: context.Background(),
    }

    dbName := config.Turso.Name
    dbUrl := config.Turso.Url
    dbAuthToken := config.Turso.Token
    tmp, err := os.MkdirTemp("", "libdata-*")

    if  err != nil {
        return err
    }

    defer os.RemoveAll(tmp)
    dbPath := filepath.Join(tmp, dbName)
    conn, err := libsql.NewEmbeddedReplicaConnector(dbPath, dbUrl, libsql.WithAuthToken(dbAuthToken))

    if err != nil {
        return err
    }

    defer conn.Close()
    db := sql.OpenDB(conn)
    defer db.Close()
    provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("./sql/migrations"))

    if err != nil {
        return err
    }

    results, err := provider.Up(context.Background())

    if err != nil {
        return err
    }


    for _, r := range results {
        log.Println("goose: %s, %s", r.Source.Path, r.Duration)
    }

    cfg.database = database.New(db)

    go func() {
        StartServer(cfg)
    }()

    encodedSessions, err := cfg.database.GetActiveMusicSessions(cfg.ctx)
    if err != nil {
        return err
    }

    sessions := []Session{}

    for _, es := range encodedSessions {
        d, _ := base64.StdEncoding.DecodeString(es.Data)

        switch es.Type {
        case "spotify":
            s := NewSpotifyFromEncoded(d, SpotifyConfig(cfg.config.Spotify), cfg.database)
            s.Id = int(es.ID)
            sessions = append(sessions, s)
        case `applemusic`:
            fmt.Println("Apple Music: ")
        }
    }

    yts := NewYoutube(cfg.ctx)
    exit := make(chan struct{})
    output := make(chan any)
    defer close(output)
    defer close(exit)

    for _, s := range sessions {
        go func() {
            s.AuthWithDB(cfg.ctx)

            if err := s.Listen(cfg.ctx, &output); err != nil {
                log.Printf("Oops: %s", err)
            }
            exit <- struct{}{}
        }()
    }

    go func() {
        for v := range output {
            switch v := v.(type) {
            case SpotifyListenValue:
                user, _ := cfg.database.GetUser(cfg.ctx, v.Username)
                scrobbler := NewScrobbler(v.Username, cfg.database)
                if ok := scrobbler.Scrobble(context.Background(), Scrobble{
                    ArtistName: v.Song.Artist,
                    TrackName: v.Song.Name,
                    AlbumName: v.Song.Album.Name,
                    AlbumArtist: v.Song.Album.Artist,
                    Timestamp: v.Song.Timestamp,
                    Duration: v.Song.Duration,
                    TrackNumber: fmt.Sprintf("%d", v.Song.TrackNumber),
                    Source: "spotify-local",
                    Uid: int(user.ID),
                    Progress: v.Song.Progress,
                }); ok {
                    log.Printf("Music URL: %s\n", yts.Search(fmt.Sprintf("%s - %s", v.Song.Artist, v.Song.Name)))
                }

            }
        }
    }()

    <- exit
    return nil
}
