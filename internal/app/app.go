package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"

	"github.com/cg219/nowplaying/internal/database"
	"github.com/dghubble/oauth1"
	"github.com/dghubble/oauth1/twitter"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
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
    Data struct {
        Path string `yaml:"path"`
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
    Twitter struct {
        Id string `yaml:"id"`
        Secret string `yaml:"secret"`
        Redirect string `yaml:"redirect"`
    }
    Discogs struct {
        Key string `yaml:"key"`
        Secret string `yaml:"secret"`
    } `json:"discogs"`
}

type AppCfg struct {
    config Config
    username string
    LastFMSession *LastFM
    TwitterOAuth oauth1.Config
    listenInterval time.Ticker
    database *database.Queries
    haveNewSessions bool
}

type Session interface {
    AuthWithDB(context.Context) error
    Listen(context.Context, *chan any, chan bool)
    Encode() []byte
    Decode([]byte) error
}

func NewConfig() *Config {
    cfg := &Config{}

    cfg.LastFM.Key = os.Getenv("LASTFM_KEY")
    cfg.LastFM.Secret = os.Getenv("LASTFM_SECRET")
    cfg.Turso.Name = os.Getenv("TURSO_NAME")
    cfg.Turso.Url = os.Getenv("TURSO_URL")
    cfg.Turso.Token = os.Getenv("TURSO_TOKEN")
    cfg.Spotify.Id = os.Getenv("SPOTIFY_ID")
    cfg.Spotify.Secret = os.Getenv("SPOTIFY_SECRET")
    cfg.Spotify.Redirect = os.Getenv("SPOTIFY_REDIRECT")
    cfg.App.Name = os.Getenv("APP_USERNAME")
    cfg.Twitter.Id = os.Getenv("TWITTER_ID")
    cfg.Twitter.Secret = os.Getenv("TWITTER_SECRET")
    cfg.Twitter.Redirect = os.Getenv("TWITTER_REDIRECT")
    cfg.Discogs.Key = os.Getenv("DISCOGS_KEY")
    cfg.Discogs.Secret = os.Getenv("DISCOGS_SECRET")
    cfg.Data.Path = os.Getenv("APP_DATA")
    cfg.App.Id, _ = strconv.Atoi(os.Getenv("APP_UID"))

    return cfg
}

func AppLoop(cfg *AppCfg) bool {
    cfg.haveNewSessions = false
    encodedSessions, _ := cfg.database.GetActiveMusicSessions(context.Background())
    sessions := []Session{}
    restartLoop := false

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

    exit := make(chan struct{})
    output := make(chan any)
    listening := make(chan bool)
    defer func() {
        _, ok := <- exit
        if ok {
            close(exit)
        }

        _, ok = <- output
        if ok {
            close(output)
        }

        _, ok = <- listening
        if ok {
            close(listening)
        }
    }()

    for si, s := range sessions {
        go func() {
            s.AuthWithDB(context.Background())
            s.Listen(context.Background(), &output, listening);

            <- listening

            log.Println("Something Goin on")
            sessions = slices.Delete(sessions, si, si + 1)
        }()
    }

    go func() {
        for {
            if cfg.haveNewSessions {
                log.Println("new session added")
                exit <- struct{}{}
                return
            }
        }
    }()

    go func() {
        for v := range output {
            switch v := v.(type) {
            case SpotifyListenValue:
                user, _ := cfg.database.GetUser(context.Background(), v.Username)
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
                    log.Printf("SCROBBLED: %s - %s\n", v.Song.Artist, v.Song.Name)
                }

            }
        }
    }()

    <- exit
    log.Println("Exiting AppLoop()")

    if cfg.haveNewSessions {
        restartLoop = true
    }
    return restartLoop
}

func Run(config Config) error {
    cfg := &AppCfg{
        config: config,
        username: config.App.Name,
        listenInterval: *time.NewTicker(5 * time.Second),
        TwitterOAuth: oauth1.Config {
            ConsumerKey: config.Twitter.Id,
            ConsumerSecret: config.Twitter.Secret,
            CallbackURL: config.Twitter.Redirect,
            Endpoint: twitter.AuthorizeEndpoint,
        },
    }

    cwd, _ := os.Getwd();
    db, err := sql.Open("sqlite", filepath.Join(cwd, config.Data.Path))
    if err != nil {
        return err
    }

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
        log.Printf("goose: %s, %s\n", r.Source.Path, r.Duration)
    }

    cfg.database = database.New(db)

    go func() {
        StartServer(cfg)
    }()

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    done := make(chan struct{})
    run := make(chan struct{})

    loop := func() {
        log.Println("running AppLoop()")
        restart := AppLoop(cfg)

        if !restart {
            close(done)
        } else {
            run <- struct{}{}
        }
    }

    go loop()

    for {
        select {
        case <- ctx.Done():
            log.Println("terminating AppLoop()")
            return nil
        case <-done:
            log.Println("AppLoop() complete")
            return nil
        case <-run:
            log.Println("restart AppLoop()")
            go loop()
        }
    }
}
