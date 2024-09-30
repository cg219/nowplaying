package app

import (
	"context"
	"database/sql"
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
    Listen(context.Context) error
}

// TODO:
// - Lookup track on youtube for a link
// - Setup ai to verify the youtube link
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

    //TODO: Properly wait for server to start befor running auth. Temp fix to start server first when testing locally
    // time.Sleep(time.Second * 2)

    sessions := []Session{
        NewLastFM(cfg.config.App.Name, LastFMConfig(cfg.config.LastFM), cfg.database),
        NewSpotify(cfg.config.App.Name, SpotifyConfig(cfg.config.Spotify), cfg.database),
    }
    exit := make(chan struct{})
    defer close(exit)

    for _, s := range sessions {
        go func() {
            s.AuthWithDB(cfg.ctx)

            if err := s.Listen(cfg.ctx); err != nil {
                log.Printf("Oops: %s", err)
            }
            exit <- struct{}{}
        }()
    }

    <- exit
    return nil
}
