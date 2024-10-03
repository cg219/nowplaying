package app

import (
	"context"
	"database/sql"
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

type AuthCfg struct {
    config Config
    username string
    client *http.Client
    ctx context.Context
    LastFMSession *LastFM
    SpotifySession *Spotify
    listenInterval time.Ticker
    database *database.Queries
}

type Session interface {
    AuthWithDB(context.Context) error
    Listen(context.Context, *chan any) error
}

func Run(config Config) error {
    cfg := &AuthCfg{
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
    provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("./migrations"))

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


    // lastfm := NewLastFM(cfg.config.App.Name, LastFMConfig(cfg.config.LastFM), cfg.database)
    spotify := NewSpotify(cfg.config.App.Name, SpotifyConfig(cfg.config.Spotify), cfg.database)
    scrobbler := NewScrobbler(cfg.config.App.Name, cfg.database)
    yts := NewYoutube(cfg.ctx)
    exit := make(chan struct{})
    output := make(chan any)
    sessions := []Session{ spotify, scrobbler }
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
            case *SpotifySong:
                log.Printf("Music URL: %s\n", yts.Search(fmt.Sprintf("%s - %s", v.Artist, v.Name)))
                scrobbler.Scrobble(context.Background(), Scrobble{
                    ArtistName: v.Artist,
                    TrackName: v.Name,
                    AlbumName: v.Album.Name,
                    AlbumArtist: v.Album.Artist,
                    Timestamp: v.Timestamp,
                    Duration: v.Duration,
                    TrackNumber: fmt.Sprintf("%d", v.TrackNumber),
                    Source: "spotify-local",
                    Uid: cfg.config.App.Id,
                }) 
            }
        }
    }()

    <- exit
    return nil
}
